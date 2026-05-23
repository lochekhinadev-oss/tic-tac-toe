package repository

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	googleuuid "github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/infrastructure/postgres/mapper"
	"tic-tac-toe/infrastructure/rediscache"
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

	leaderboardCache rediscache.LeaderboardCache
}

type sqlExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type transactionStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

func NewGameRepository(db datasource.Database, leaderboardCache rediscache.LeaderboardCache) *GameRepository {
	return &GameRepository{
		db:               db,
		leaderboardCache: leaderboardCache,
	}
}

func (r *GameRepository) SaveGame(ctx context.Context, game domain.Game) error {
	logGameRepository("save game", "uuid", game.UUID, "state", game.State, "mode", game.Mode)
	datasourceGame := mapper.ToDatasourceGame(game)
	if datasourceGame.CreatedAt.IsZero() {
		datasourceGame.CreatedAt = time.Now().UTC()
	}
	field, err := json.Marshal(datasourceGame.Field)
	if err != nil {
		logGameRepository("save game marshal failed", "uuid", game.UUID, "error", err)
		return err
	}

	err = r.withTransaction(ctx, func(tx sqlExecutor) error {
		_, err = tx.Exec(ctx, saveGameQuery, datasourceGame.UUID, field, datasourceGame.Mode, datasourceGame.State, datasourceGame.CreatedAt, datasourceGame.NextPlayerUUID, datasourceGame.WinnerUUID, datasourceGame.PlayerXUUID, datasourceGame.PlayerOUUID)
		if err != nil {
			logGameRepository("save game failed", "uuid", game.UUID, "error", err)
			return err
		}
		return r.applyCompletedGameStats(ctx, tx, game)
	})
	if err != nil {
		return err
	}
	r.invalidateLeaderboardCacheIfCompleted(ctx, game)

	logGameRepository("save game ok", "uuid", game.UUID)
	return nil
}

func (r *GameRepository) SaveGameIfUnchanged(ctx context.Context, previous domain.Game, next domain.Game) error {
	logGameRepository("save game if unchanged", "uuid", next.UUID, "state", next.State, "mode", next.Mode)

	nextGame := mapper.ToDatasourceGame(next)
	nextField, err := json.Marshal(nextGame.Field)
	if err != nil {
		logGameRepository("save game if unchanged marshal next failed", "uuid", next.UUID, "error", err)
		return err
	}

	previousGame := mapper.ToDatasourceGame(previous)
	previousField, err := json.Marshal(previousGame.Field)
	if err != nil {
		logGameRepository("save game if unchanged marshal previous failed", "uuid", previous.UUID, "error", err)
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
			logGameRepository("save game if unchanged failed", "uuid", next.UUID, "error", err)
			return err
		}
		if result.RowsAffected() == 0 {
			logGameRepository("save game if unchanged conflict", "uuid", next.UUID)
			return ErrGameConflict
		}
		return r.applyCompletedGameStats(ctx, tx, next)
	})
	if err != nil {
		return err
	}
	r.invalidateLeaderboardCacheIfCompleted(ctx, next)

	logGameRepository("save game if unchanged ok", "uuid", next.UUID)
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

	_, err := executor.Exec(ctx, applyCompletedGameStatsQuery, game.UUID, game.PlayerX.String(), game.PlayerO.String(), string(game.State), game.Winner.String())
	if err != nil {
		logGameRepository("apply completed game stats failed", "uuid", game.UUID, "error", err)
		return err
	}

	logGameRepository("apply completed game stats ok", "uuid", game.UUID)
	return nil
}

func (r *GameRepository) GetGame(ctx context.Context, uuid googleuuid.UUID) (domain.Game, error) {
	logGameRepository("get game", "uuid", uuid)
	game, err := r.queryGame(ctx, getGameQuery, uuid.String())
	if errors.Is(err, pgx.ErrNoRows) {
		logGameRepository("get game not found", "uuid", uuid)
		return domain.Game{}, ErrGameNotFound
	}
	if err != nil {
		logGameRepository("get game failed", "uuid", uuid, "error", err)
		return domain.Game{}, err
	}

	logGameRepository("get game ok", "uuid", uuid, "state", game.State)
	return game, nil
}

func (r *GameRepository) ListActiveGames(ctx context.Context) ([]domain.Game, error) {
	games, err := r.queryGames(ctx, "list active games", listActiveGamesQuery, string(domain.GameModePlayer), string(domain.GameStateWaitingPlayers))
	if err != nil {
		return nil, err
	}

	logGameRepository("list active games ok", "count", len(games))
	return games, nil
}

func (r *GameRepository) ListCompletedGamesByUserUUID(ctx context.Context, userUUID googleuuid.UUID) ([]domain.Game, error) {
	logGameRepository("list completed games", "user_uuid", userUUID)
	games, err := r.queryGames(ctx, "list completed games", listCompletedGamesByUserUUIDQuery, userUUID.String(), string(domain.GameStatePlayerWins), string(domain.GameStateDraw))
	if err != nil {
		return nil, err
	}

	logGameRepository("list completed games ok", "user_uuid", userUUID, "count", len(games))
	return games, nil
}

func (r *GameRepository) ListTopPlayers(ctx context.Context, limit int) ([]domain.WonGameInfo, error) {
	logGameRepository("list top players", "limit", limit)
	if players, ok, err := r.cachedTopPlayers(ctx, limit); err != nil {
		logGameRepository("list top players cache read failed", "limit", limit, "error", err)
	} else if ok {
		logGameRepository("list top players cache hit", "limit", limit, "count", len(players))
		return players, nil
	}

	rows, err := r.db.Query(ctx, listTopPlayersQuery, limit)
	if err != nil {
		logGameRepository("list top players query failed", "error", err)
		return nil, err
	}
	defer rows.Close()

	players, err := scanTopPlayers(rows, "list top players")
	if err != nil {
		return nil, err
	}
	if err := r.cacheTopPlayers(ctx, limit, players); err != nil {
		logGameRepository("list top players cache write failed", "limit", limit, "error", err)
	}
	logGameRepository("list top players ok", "count", len(players))
	return players, nil
}

func (r *GameRepository) cachedTopPlayers(ctx context.Context, limit int) ([]domain.WonGameInfo, bool, error) {
	if r.leaderboardCache == nil {
		return nil, false, nil
	}

	players, ok, err := r.leaderboardCache.GetLeaderboard(ctx, limit)
	if err != nil || !ok {
		return nil, ok, err
	}

	return cloneTopPlayers(players), true, nil
}

func (r *GameRepository) cacheTopPlayers(ctx context.Context, limit int, players []domain.WonGameInfo) error {
	if r.leaderboardCache == nil {
		return nil
	}

	return r.leaderboardCache.SetLeaderboard(ctx, limit, cloneTopPlayers(players), leaderboardCacheTTL)
}

func (r *GameRepository) invalidateLeaderboardCacheIfCompleted(ctx context.Context, game domain.Game) {
	if game.State != domain.GameStatePlayerWins && game.State != domain.GameStateDraw {
		return
	}

	if r.leaderboardCache == nil {
		return
	}

	if err := r.leaderboardCache.InvalidateLeaderboard(ctx); err != nil {
		logGameRepository("invalidate leaderboard cache failed", "uuid", game.UUID, "error", err)
	}
}

func cloneTopPlayers(players []domain.WonGameInfo) []domain.WonGameInfo {
	if players == nil {
		return nil
	}
	result := make([]domain.WonGameInfo, len(players))
	copy(result, players)
	return result
}

func (r *GameRepository) JoinGame(ctx context.Context, uuid googleuuid.UUID, userUUID googleuuid.UUID) (domain.Game, error) {
	logGameRepository("join game", "uuid", uuid, "user_uuid", userUUID)
	game, err := r.queryGame(ctx, joinGameQuery, uuid.String(), userUUID.String(), string(domain.GameStatePlayerToMove), string(domain.GameModePlayer), string(domain.GameStateWaitingPlayers))
	if errors.Is(err, pgx.ErrNoRows) {
		logGameRepository("join game not joinable", "uuid", uuid, "user_uuid", userUUID)
		return domain.Game{}, ErrGameNotJoinable
	}
	if err != nil {
		logGameRepository("join game failed", "uuid", uuid, "user_uuid", userUUID, "error", err)
		return domain.Game{}, err
	}

	logGameRepository("join game ok", "uuid", uuid, "user_uuid", userUUID)
	return game, nil
}

func (r *GameRepository) queryGames(ctx context.Context, operation string, query string, args ...any) ([]domain.Game, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		logGameRepository(operation+" query failed", "error", err)
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

func logGameRepository(action string, args ...any) {
	slog.Info(gameRepositoryLogPrefix+" "+action, args...)
}
