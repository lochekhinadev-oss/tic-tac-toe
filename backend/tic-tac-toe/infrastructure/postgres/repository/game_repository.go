package repository

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/infrastructure/postgres/mapper"
	"tic-tac-toe/internal/logging"
)

var (
	ErrGameNotFound    = domain.ErrGameNotFound
	ErrGameConflict    = domain.ErrGameConflict
	ErrGameNotJoinable = domain.ErrGameNotJoinable
)

const (
	gameRepositoryLogPrefix = "[infrastructure/postgres/repository]"
	leaderboardCacheTTL     = 10 * time.Second
)

type GameRepository struct {
	db datasource.Database

	now                func() time.Time
	leaderboardCacheMu sync.Mutex
	leaderboardCache   map[int]cachedLeaderboard
}

type cachedLeaderboard struct {
	players   []domain.WonGameInfo
	expiresAt time.Time
}

type sqlExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type transactionStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func NewGameRepository(db datasource.Database) *GameRepository {
	return &GameRepository{
		db:               db,
		now:              time.Now,
		leaderboardCache: make(map[int]cachedLeaderboard),
	}
}

func (r *GameRepository) SaveGame(ctx context.Context, game domain.Game) error {
	logGameRepository("save game", "uuid=%q state=%s mode=%s", game.UUID, game.State, game.Mode)
	datasourceGame := mapper.ToDatasourceGame(game)
	if datasourceGame.CreatedAt.IsZero() {
		datasourceGame.CreatedAt = time.Now().UTC()
	}
	field, err := json.Marshal(datasourceGame.Field)
	if err != nil {
		logGameRepository("save game marshal failed", "uuid=%q: %v", game.UUID, err)
		return err
	}

	err = r.withTransaction(ctx, func(tx sqlExecutor) error {
		_, err = tx.Exec(ctx, saveGameQuery, datasourceGame.UUID, field, datasourceGame.Mode, datasourceGame.State, datasourceGame.CreatedAt, datasourceGame.NextPlayerUUID, datasourceGame.WinnerUUID, datasourceGame.PlayerXUUID, datasourceGame.PlayerOUUID)
		if err != nil {
			logGameRepository("save game failed", "uuid=%q: %v", game.UUID, err)
			return err
		}
		return r.applyCompletedGameStats(ctx, tx, game)
	})
	if err != nil {
		return err
	}
	r.invalidateLeaderboardCacheIfCompleted(game)

	logGameRepository("save game ok", "uuid=%q", game.UUID)
	return nil
}

func (r *GameRepository) SaveGameIfUnchanged(ctx context.Context, previous domain.Game, next domain.Game) error {
	logGameRepository("save game if unchanged", "uuid=%q state=%s mode=%s", next.UUID, next.State, next.Mode)

	nextGame := mapper.ToDatasourceGame(next)
	nextField, err := json.Marshal(nextGame.Field)
	if err != nil {
		logGameRepository("save game if unchanged marshal next failed", "uuid=%q: %v", next.UUID, err)
		return err
	}

	previousGame := mapper.ToDatasourceGame(previous)
	previousField, err := json.Marshal(previousGame.Field)
	if err != nil {
		logGameRepository("save game if unchanged marshal previous failed", "uuid=%q: %v", previous.UUID, err)
		return err
	}

	err = r.withTransaction(ctx, func(tx sqlExecutor) error {
		result, err := tx.Exec(
			ctx,
			saveGameIfUnchangedQuery,
			nextGame.UUID,
			nextField,
			nextGame.Mode,
			nextGame.State,
			nextGame.NextPlayerUUID,
			nextGame.WinnerUUID,
			nextGame.PlayerXUUID,
			nextGame.PlayerOUUID,
			previousField,
			previousGame.Mode,
			previousGame.State,
			previousGame.NextPlayerUUID,
			previousGame.WinnerUUID,
			previousGame.PlayerXUUID,
			previousGame.PlayerOUUID,
		)
		if err != nil {
			logGameRepository("save game if unchanged failed", "uuid=%q: %v", next.UUID, err)
			return err
		}
		if result.RowsAffected() == 0 {
			logGameRepository("save game if unchanged conflict", "uuid=%q", next.UUID)
			return ErrGameConflict
		}
		return r.applyCompletedGameStats(ctx, tx, next)
	})
	if err != nil {
		return err
	}
	r.invalidateLeaderboardCacheIfCompleted(next)

	logGameRepository("save game if unchanged ok", "uuid=%q", next.UUID)
	return nil
}

func (r *GameRepository) withTransaction(ctx context.Context, run func(sqlExecutor) error) error {
	starter, ok := r.db.(transactionStarter)
	if !ok {
		return errors.New("database does not support transactions")
	}

	tx, err := starter.Begin(ctx)
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := run(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r *GameRepository) applyCompletedGameStats(ctx context.Context, executor sqlExecutor, game domain.Game) error {
	if game.State != domain.GameStatePlayerWins && game.State != domain.GameStateDraw {
		return nil
	}

	_, err := executor.Exec(ctx, applyCompletedGameStatsQuery, game.UUID, game.PlayerXUUID, game.PlayerOUUID, string(game.State), game.WinnerUUID)
	if err != nil {
		logGameRepository("apply completed game stats failed", "uuid=%q: %v", game.UUID, err)
		return err
	}

	logGameRepository("apply completed game stats ok", "uuid=%q", game.UUID)
	return nil
}

func (r *GameRepository) GetGame(ctx context.Context, uuid string) (domain.Game, error) {
	logGameRepository("get game", "uuid=%q", uuid)
	game, err := r.queryGame(ctx, getGameQuery, uuid)
	if errors.Is(err, pgx.ErrNoRows) {
		logGameRepository("get game not found", "uuid=%q", uuid)
		return domain.Game{}, ErrGameNotFound
	}
	if err != nil {
		logGameRepository("get game failed", "uuid=%q: %v", uuid, err)
		return domain.Game{}, err
	}

	logGameRepository("get game ok", "uuid=%q state=%s", uuid, game.State)
	return game, nil
}

func (r *GameRepository) ListActiveGames(ctx context.Context) ([]domain.Game, error) {
	games, err := r.queryGames(ctx, "list active games", listActiveGamesQuery, string(domain.GameModePlayer), string(domain.GameStateWaitingPlayers))
	if err != nil {
		return nil, err
	}

	logGameRepository("list active games ok", "count=%d", len(games))
	return games, nil
}

func (r *GameRepository) ListCompletedGamesByUserUUID(ctx context.Context, userUUID string) ([]domain.Game, error) {
	logGameRepository("list completed games", "user=%q", userUUID)
	games, err := r.queryGames(ctx, "list completed games", listCompletedGamesByUserUUIDQuery, userUUID, string(domain.GameStatePlayerWins), string(domain.GameStateDraw))
	if err != nil {
		return nil, err
	}

	logGameRepository("list completed games ok", "user=%q count=%d", userUUID, len(games))
	return games, nil
}

func (r *GameRepository) ListTopPlayers(ctx context.Context, limit int) ([]domain.WonGameInfo, error) {
	logGameRepository("list top players", "limit=%d", limit)
	if players, ok := r.cachedTopPlayers(limit); ok {
		logGameRepository("list top players cache hit", "limit=%d count=%d", limit, len(players))
		return players, nil
	}

	rows, err := r.db.Query(ctx, listTopPlayersQuery, limit)
	if err != nil {
		logGameRepository("list top players query failed", "%v", err)
		return nil, err
	}
	defer rows.Close()

	players, err := scanTopPlayers(rows, "list top players")
	if err != nil {
		return nil, err
	}
	r.cacheTopPlayers(limit, players)
	logGameRepository("list top players ok", "count=%d", len(players))
	return players, nil
}

func (r *GameRepository) cachedTopPlayers(limit int) ([]domain.WonGameInfo, bool) {
	r.leaderboardCacheMu.Lock()
	defer r.leaderboardCacheMu.Unlock()

	entry, ok := r.leaderboardCache[limit]
	if !ok {
		return nil, false
	}
	if !r.now().Before(entry.expiresAt) {
		delete(r.leaderboardCache, limit)
		return nil, false
	}
	return cloneTopPlayers(entry.players), true
}

func (r *GameRepository) cacheTopPlayers(limit int, players []domain.WonGameInfo) {
	r.leaderboardCacheMu.Lock()
	defer r.leaderboardCacheMu.Unlock()

	r.leaderboardCache[limit] = cachedLeaderboard{
		players:   cloneTopPlayers(players),
		expiresAt: r.now().Add(leaderboardCacheTTL),
	}
}

func (r *GameRepository) invalidateLeaderboardCacheIfCompleted(game domain.Game) {
	if game.State != domain.GameStatePlayerWins && game.State != domain.GameStateDraw {
		return
	}

	r.leaderboardCacheMu.Lock()
	defer r.leaderboardCacheMu.Unlock()

	r.leaderboardCache = make(map[int]cachedLeaderboard)
}

func cloneTopPlayers(players []domain.WonGameInfo) []domain.WonGameInfo {
	if players == nil {
		return nil
	}
	result := make([]domain.WonGameInfo, len(players))
	copy(result, players)
	return result
}

func (r *GameRepository) JoinGame(ctx context.Context, uuid string, userUUID string) (domain.Game, error) {
	logGameRepository("join game", "uuid=%q user=%q", uuid, userUUID)
	game, err := r.queryGame(ctx, joinGameQuery, uuid, userUUID, string(domain.GameStatePlayerToMove), string(domain.GameModePlayer), string(domain.GameStateWaitingPlayers))
	if errors.Is(err, pgx.ErrNoRows) {
		logGameRepository("join game not joinable", "uuid=%q user=%q", uuid, userUUID)
		return domain.Game{}, ErrGameNotJoinable
	}
	if err != nil {
		logGameRepository("join game failed", "uuid=%q user=%q: %v", uuid, userUUID, err)
		return domain.Game{}, err
	}

	logGameRepository("join game ok", "uuid=%q user=%q", uuid, userUUID)
	return game, nil
}

func (r *GameRepository) queryGames(ctx context.Context, operation string, query string, args ...any) ([]domain.Game, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		logGameRepository(operation+" query failed", "%v", err)
		return nil, err
	}
	defer rows.Close()

	games, err := scanGames(rows, operation)
	if err != nil {
		return nil, err
	}
	return games, nil
}

func (r *GameRepository) queryGame(ctx context.Context, query string, args ...any) (domain.Game, error) {
	return scanGameRow(r.db.QueryRow(ctx, query, args...))
}

func logGameRepository(action string, format string, args ...any) {
	log.Printf(gameRepositoryLogPrefix+" "+action+" "+format, args...)
}
