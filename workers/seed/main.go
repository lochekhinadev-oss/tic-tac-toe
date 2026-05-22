package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/datasource"
)

const (
	seedEnabledEnv = "SEED_ENABLED"
	seedUsersEnv   = "SEED_USERS"
	seedGamesEnv   = "SEED_GAMES"
	passwordCost   = bcrypt.MinCost

	seedRandomSource = 1421424
)

var completedStates = []domain.GameState{domain.GameStatePlayerWins, domain.GameStateDraw}

type seedConfig struct {
	databaseURL string
	enabled     bool
	users       int
	games       int
}

func main() {
	config := parseConfig()
	if !config.enabled {
		log.Printf("seed worker disabled: %s is not enabled", seedEnabledEnv)
		return
	}

	ctx := context.Background()
	if err := waitForDatabase(ctx, config.databaseURL); err != nil {
		log.Fatalf("database not ready: %v", err)
	}

	pool, err := pgxpool.New(ctx, config.databaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	if err := seed(ctx, pool, config); err != nil {
		log.Fatalf("seed database: %v", err)
	}

	log.Printf("seed completed: users=%d games=%d", config.users, config.games)
}

func parseConfig() seedConfig {
	config := seedConfig{}
	flag.StringVar(&config.databaseURL, "database-url", datasource.NewDatabaseConfig().DatabaseURL, "Postgres connection URL")
	config.enabled = envBool(seedEnabledEnv)
	flag.IntVar(&config.users, "users", 0, "Number of users to seed")
	flag.IntVar(&config.games, "games", 0, "Number of games to seed")
	flag.Parse()

	if !config.enabled {
		return config
	}

	if config.users <= 0 {
		config.users = envIntRequired(seedUsersEnv)
	}
	if config.games <= 0 {
		config.games = envIntRequired(seedGamesEnv)
	}

	if config.users <= 0 {
		log.Fatal("users must be positive")
	}
	if config.games <= 0 {
		log.Fatal("games must be positive")
	}
	if config.users < 2 {
		log.Fatal("users must be at least 2 to seed games")
	}
	return config
}

func waitForDatabase(ctx context.Context, databaseURL string) error {
	deadline := time.Now().Add(30 * time.Second)
	for {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			pingErr := pool.Ping(pingCtx)
			cancel()
			pool.Close()
			if pingErr == nil {
				return nil
			}
			err = pingErr
		}

		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(1 * time.Second)
	}
}

func seed(ctx context.Context, pool *pgxpool.Pool, config seedConfig) error {
	if err := ensureSchemaReady(ctx, pool); err != nil {
		return err
	}
	if err := seedUsers(ctx, pool, config.users); err != nil {
		return fmt.Errorf("seed users: %w", err)
	}
	if err := seedGamesAndStats(ctx, pool, config); err != nil {
		return fmt.Errorf("seed games: %w", err)
	}
	return nil
}

func ensureSchemaReady(ctx context.Context, pool *pgxpool.Pool) error {
	requiredTables := []string{"users", "games", "player_stats", "game_stats_applied"}
	for _, table := range requiredTables {
		var exists bool
		err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+table).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("required table %q does not exist; run app migrations before seeding", table)
		}
	}
	return nil
}

func seedUsers(ctx context.Context, pool *pgxpool.Pool, total int) error {
	log.Printf("seeding users: %d", total)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE seed_users (uuid TEXT PRIMARY KEY, login TEXT NOT NULL, password TEXT NOT NULL) ON COMMIT DROP`); err != nil {
		return err
	}

	if err := copySeedUsers(ctx, tx, total); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO users (uuid, login, password)
		SELECT uuid, login, password FROM seed_users
		ON CONFLICT (uuid) DO UPDATE SET
			login = EXCLUDED.login,
			password = EXCLUDED.password
	`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func copySeedUsers(ctx context.Context, tx pgx.Tx, total int) error {
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"seed_users"}, []string{"uuid", "login", "password"}, &userCopySource{total: total})
	return err
}

func seedGamesAndStats(ctx context.Context, pool *pgxpool.Pool, config seedConfig) error {
	log.Printf("seeding games: %d", config.games)
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE seed_games (LIKE games INCLUDING DEFAULTS) ON COMMIT DROP`); err != nil {
		return err
	}

	if err := copySeedGames(ctx, tx, config); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO games (
			uuid, field, mode, state, created_at, next_player_uuid, winner_uuid, player_x_uuid, player_o_uuid
		)
		SELECT uuid, field, mode, state, created_at, next_player_uuid, winner_uuid, player_x_uuid, player_o_uuid
		FROM seed_games
		ON CONFLICT (uuid) DO UPDATE SET
			field = EXCLUDED.field,
			mode = EXCLUDED.mode,
			state = EXCLUDED.state,
			created_at = EXCLUDED.created_at,
			next_player_uuid = EXCLUDED.next_player_uuid,
			winner_uuid = EXCLUDED.winner_uuid,
			player_x_uuid = EXCLUDED.player_x_uuid,
			player_o_uuid = EXCLUDED.player_o_uuid
	`); err != nil {
		return err
	}
	if err := rebuildSeedStats(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func copySeedGames(ctx context.Context, tx pgx.Tx, config seedConfig) error {
	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"seed_games"},
		[]string{"uuid", "field", "mode", "state", "created_at", "next_player_uuid", "winner_uuid", "player_x_uuid", "player_o_uuid"},
		newGameCopySource(config.games, config.users),
	)
	return err
}

type userCopySource struct {
	total int
	index int
	row   []any
	err   error
}

func (s *userCopySource) Next() bool {
	if s.err != nil || s.index >= s.total {
		return false
	}

	login := userLogin(s.index)
	hash, err := bcrypt.GenerateFromPassword([]byte(login), passwordCost)
	if err != nil {
		s.err = err
		return false
	}

	s.row = []any{userUUID(s.index), login, string(hash)}
	s.index++
	return true
}

func (s *userCopySource) Values() ([]any, error) { return s.row, nil }
func (s *userCopySource) Err() error             { return s.err }

type gameCopySource struct {
	total  int
	users  int
	index  int
	random *rand.Rand
	row    []any
	err    error
}

func newGameCopySource(total int, users int) *gameCopySource {
	return &gameCopySource{
		total:  total,
		users:  users,
		random: rand.New(rand.NewSource(seedRandomSource)),
	}
}

func (s *gameCopySource) Next() bool {
	if s.err != nil || s.index >= s.total {
		return false
	}

	row, err := gameRow(s.index, s.users, s.random)
	if err != nil {
		s.err = err
		return false
	}

	s.row = row
	s.index++
	return true
}

func (s *gameCopySource) Values() ([]any, error) { return s.row, nil }
func (s *gameCopySource) Err() error             { return s.err }

func gameRow(index int, users int, random *rand.Rand) ([]any, error) {
	playerX := index % users
	playerO := (index*37 + 1) % users
	if playerO == playerX {
		playerO = (playerO + 1) % users
	}

	state := completedStates[index%len(completedStates)]
	winnerUUID := ""
	if state == domain.GameStatePlayerWins {
		if random.Intn(2) == 0 {
			winnerUUID = userUUID(playerX)
		} else {
			winnerUUID = userUUID(playerO)
		}
	}

	field, err := json.Marshal(fieldForState(index, state))
	if err != nil {
		return nil, err
	}

	return []any{
		gameUUID(index),
		string(field),
		string(domain.GameModePlayer),
		string(state),
		time.Now().UTC().Add(-time.Duration(index) * time.Minute),
		"",
		winnerUUID,
		userUUID(playerX),
		userUUID(playerO),
	}, nil
}

func fieldForState(index int, state domain.GameState) [][]int {
	if state == domain.GameStateDraw {
		return [][]int{
			{domain.CellX, domain.CellO, domain.CellX},
			{domain.CellX, domain.CellO, domain.CellO},
			{domain.CellO, domain.CellX, domain.CellX},
		}
	}
	if index%2 == 0 {
		return [][]int{
			{domain.CellX, domain.CellX, domain.CellX},
			{domain.CellO, domain.CellO, domain.CellEmpty},
			{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
		}
	}
	return [][]int{
		{domain.CellO, domain.CellO, domain.CellO},
		{domain.CellX, domain.CellX, domain.CellEmpty},
		{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
	}
}

func rebuildSeedStats(ctx context.Context, tx pgx.Tx) error {
	playerWins := string(domain.GameStatePlayerWins)
	draw := string(domain.GameStateDraw)

	if _, err := tx.Exec(ctx, `
		DELETE FROM game_stats_applied
		WHERE game_uuid IN (
			SELECT games.uuid
			FROM games
			WHERE games.player_x_uuid IN (
				SELECT player_x_uuid FROM seed_games
				UNION
				SELECT player_o_uuid FROM seed_games
			)
			   OR games.player_o_uuid IN (
				SELECT player_x_uuid FROM seed_games
				UNION
				SELECT player_o_uuid FROM seed_games
			)
		)
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM player_stats
		WHERE user_uuid IN (
			SELECT player_x_uuid FROM seed_games
			UNION
			SELECT player_o_uuid FROM seed_games
		)
	`); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		WITH player_results AS (
			SELECT
				uuid AS game_uuid,
				player_x_uuid AS user_uuid,
				CASE WHEN winner_uuid = player_x_uuid THEN 1 ELSE 0 END AS wins,
				CASE WHEN state = $1 AND winner_uuid <> player_x_uuid THEN 1 ELSE 0 END AS losses,
				CASE WHEN state = $2 THEN 1 ELSE 0 END AS draws
			FROM games
			WHERE state IN ($1, $2)
			  AND player_x_uuid IN (
			      SELECT player_x_uuid FROM seed_games
			      UNION
			      SELECT player_o_uuid FROM seed_games
			  )
			UNION ALL
			SELECT
				uuid AS game_uuid,
				player_o_uuid AS user_uuid,
				CASE WHEN winner_uuid = player_o_uuid THEN 1 ELSE 0 END AS wins,
				CASE WHEN state = $1 AND winner_uuid <> player_o_uuid THEN 1 ELSE 0 END AS losses,
				CASE WHEN state = $2 THEN 1 ELSE 0 END AS draws
			FROM games
			WHERE state IN ($1, $2)
			  AND player_o_uuid IN (
			      SELECT player_x_uuid FROM seed_games
			      UNION
			      SELECT player_o_uuid FROM seed_games
			  )
			  AND player_o_uuid <> $3
		),
		inserted_games AS (
			INSERT INTO game_stats_applied (game_uuid)
			SELECT DISTINCT game_uuid FROM player_results
			ON CONFLICT (game_uuid) DO NOTHING
			RETURNING game_uuid
		),
		stats AS (
			SELECT
				player_results.user_uuid,
				SUM(player_results.wins) AS wins,
				SUM(player_results.losses) AS losses,
				SUM(player_results.draws) AS draws,
				COUNT(*) AS completed_games
			FROM player_results
			JOIN inserted_games ON inserted_games.game_uuid = player_results.game_uuid
			GROUP BY player_results.user_uuid
		)
		INSERT INTO player_stats (user_uuid, wins, losses, draws, completed_games, updated_at)
		SELECT user_uuid, wins, losses, draws, completed_games, now()
		FROM stats
		ON CONFLICT (user_uuid) DO UPDATE SET
			wins = EXCLUDED.wins,
			losses = EXCLUDED.losses,
			draws = EXCLUDED.draws,
			completed_games = EXCLUDED.completed_games,
			updated_at = now()
	`, playerWins, draw, domain.ComputerPlayerUUID)
	return err
}

func userLogin(index int) string { return fmt.Sprintf("%04d", index) }
func userUUID(index int) string  { return fmt.Sprintf("00000000-0000-4000-8000-%012d", index) }
func gameUUID(index int) string  { return fmt.Sprintf("10000000-0000-4000-8000-%012d", index) }

func envIntRequired(key string) int {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s must be set to run seed worker", key)
	}

	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil || parsed <= 0 {
		log.Fatalf("%s must be a positive integer", key)
	}
	return parsed
}

func envBool(key string) bool {
	value := strings.TrimSpace(os.Getenv(key))
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
