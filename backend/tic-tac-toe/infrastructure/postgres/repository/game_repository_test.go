package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tic-tac-toe/app/domain"
)

func TestGameRepositorySaveAndGet(t *testing.T) {
	game := sampleGame()
	db := &databaseStub{}
	repo := NewGameRepository(db)

	if err := repo.SaveGame(context.Background(), game); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	if db.savedUUID != "game-1" || db.savedField[1][1] != 2 {
		t.Fatalf("unexpected saved game: %q %#v", db.savedUUID, db.savedField)
	}

	got, err := repo.GetGame(context.Background(), "game-1")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}

	assertGame(t, got, game)

	game.Field[0][0] = 9
	assertStoredGameCloned(t, got)
}

func TestGameRepositoryGetMissing(t *testing.T) {
	repo := NewGameRepository(&databaseStub{queryErr: pgx.ErrNoRows})
	_, err := repo.GetGame(context.Background(), "missing")
	if !errors.Is(err, ErrGameNotFound) {
		t.Fatalf("expected ErrGameNotFound, got %v", err)
	}
}

func TestGameRepositorySaveGameIfUnchangedConflict(t *testing.T) {
	db := &databaseStub{rowsAffected: "UPDATE 0"}
	repo := NewGameRepository(db)

	err := repo.SaveGameIfUnchanged(context.Background(), sampleGame(), sampleGame())
	if !errors.Is(err, ErrGameConflict) {
		t.Fatalf("expected ErrGameConflict, got %v", err)
	}
	if db.commits != 0 || db.rollbacks != 1 {
		t.Fatalf("expected rollback without commit, commits=%d rollbacks=%d", db.commits, db.rollbacks)
	}
}

func TestGameRepositoryAppliesCompletedGameStats(t *testing.T) {
	game := sampleGame()
	game.State = domain.GameStatePlayerWins
	game.WinnerUUID = "user-x"
	game.PlayerXUUID = "user-x"
	game.PlayerOUUID = "user-o"

	db := &databaseStub{}
	repo := NewGameRepository(db)

	if err := repo.SaveGame(context.Background(), game); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	if db.statsApplications != 1 {
		t.Fatalf("expected one stats application, got %d", db.statsApplications)
	}
	if db.commits != 1 || db.rollbacks != 0 {
		t.Fatalf("expected commit without rollback, commits=%d rollbacks=%d", db.commits, db.rollbacks)
	}
}

func TestGameRepositorySkipsStatsForActiveGame(t *testing.T) {
	game := sampleGame()
	game.State = domain.GameStatePlayerToMove

	db := &databaseStub{}
	repo := NewGameRepository(db)

	if err := repo.SaveGame(context.Background(), game); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	if db.statsApplications != 0 {
		t.Fatalf("expected no stats application, got %d", db.statsApplications)
	}
	if db.commits != 1 || db.rollbacks != 0 {
		t.Fatalf("expected commit without rollback, commits=%d rollbacks=%d", db.commits, db.rollbacks)
	}
}

func TestGameRepositoryCachesTopPlayers(t *testing.T) {
	db := &databaseStub{topPlayers: []domain.WonGameInfo{sampleTopPlayer()}}
	repo := NewGameRepository(db)

	first, err := repo.ListTopPlayers(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected first leaderboard error: %v", err)
	}
	first[0].Login = "mutated"

	second, err := repo.ListTopPlayers(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected second leaderboard error: %v", err)
	}

	if db.topPlayersQueries != 1 {
		t.Fatalf("expected one leaderboard query, got %d", db.topPlayersQueries)
	}
	if second[0].Login != "player" {
		t.Fatalf("expected cached players to be cloned, got %#v", second)
	}
}

func TestGameRepositoryInvalidatesTopPlayersCacheAfterCompletedGame(t *testing.T) {
	game := sampleGame()
	game.State = domain.GameStateDraw
	game.PlayerXUUID = "user-x"
	game.PlayerOUUID = "user-o"

	db := &databaseStub{topPlayers: []domain.WonGameInfo{sampleTopPlayer()}}
	repo := NewGameRepository(db)

	if _, err := repo.ListTopPlayers(context.Background(), 10); err != nil {
		t.Fatalf("unexpected first leaderboard error: %v", err)
	}
	if err := repo.SaveGame(context.Background(), game); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}
	if _, err := repo.ListTopPlayers(context.Background(), 10); err != nil {
		t.Fatalf("unexpected second leaderboard error: %v", err)
	}

	if db.topPlayersQueries != 2 {
		t.Fatalf("expected cache invalidation and second query, got %d queries", db.topPlayersQueries)
	}
}

type databaseStub struct {
	savedUUID         string
	savedField        [][]int
	savedCreatedAt    time.Time
	savedArgs         []any
	lastExecQuery     string
	lastQueryRowQuery string
	lastQueryRowArgs  []any
	lastQueryQuery    string
	lastQueryArgs     []any
	queryErr          error
	queryRows         pgx.Rows
	queryError        error
	queryScanErr      error
	rowsAffected      string
	topPlayers        []domain.WonGameInfo
	topPlayersQueries int
	statsApplications int
	commits           int
	rollbacks         int
}

func (d *databaseStub) Begin(context.Context) (pgx.Tx, error) {
	return &txStub{db: d}, nil
}

func (d *databaseStub) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	d.lastExecQuery = sql
	d.savedArgs = arguments

	if sql == applyCompletedGameStatsQuery {
		d.statsApplications++
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}

	if strings.Contains(sql, "games") {
		d.savedUUID, _ = arguments[0].(string)
		field, _ := arguments[1].([]byte)
		if err := json.Unmarshal(field, &d.savedField); err != nil {
			return pgconn.CommandTag{}, err
		}
		if strings.Contains(sql, "ON CONFLICT") {
			d.savedCreatedAt, _ = arguments[4].(time.Time)
		}
	}

	if d.rowsAffected != "" {
		return pgconn.NewCommandTag(d.rowsAffected), nil
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (d *databaseStub) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	d.lastQueryRowQuery = sql
	d.lastQueryRowArgs = args
	uuid, _ := args[0].(string)
	if d.queryErr != nil {
		return rowStub{err: d.queryErr}
	}

	field, err := json.Marshal(d.savedField)
	createdAt := d.savedCreatedAt
	if createdAt.IsZero() {
		createdAt = sampleGame().CreatedAt
	}
	return rowStub{
		uuid:      uuid,
		field:     field,
		mode:      string(domain.GameModeComputer),
		state:     string(domain.GameStatePlayerToMove),
		createdAt: createdAt,
		err:       err,
	}
}

func (d *databaseStub) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	d.lastQueryQuery = sql
	d.lastQueryArgs = args
	if sql == listTopPlayersQuery {
		d.topPlayersQueries++
		return &rowsStub{topPlayers: d.topPlayers, scanErr: d.queryScanErr}, d.queryError
	}
	return d.queryRows, d.queryError
}

func (d *databaseStub) Ping(context.Context) error {
	return nil
}

func TestGameRepositoryListAndJoin(t *testing.T) {
	game := sampleGame()
	game.Mode = domain.GameModePlayer
	game.State = domain.GameStateWaitingPlayers
	game.PlayerXUUID = "user-x"
	game.NextPlayerUUID = "user-x"

	db := &databaseStub{queryRows: &rowsStub{games: []domain.Game{game}}}
	repo := NewGameRepository(db)

	games, err := repo.ListActiveGames(context.Background())
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	assertGames(t, games, game)

	historyDB := &databaseStub{queryRows: &rowsStub{games: []domain.Game{game}}}
	historyRepo := NewGameRepository(historyDB)
	completed, err := historyRepo.ListCompletedGamesByUserUUID(context.Background(), "user-x")
	if err != nil {
		t.Fatalf("unexpected completed list error: %v", err)
	}
	assertGames(t, completed, game)

	topDB := &databaseStub{topPlayers: []domain.WonGameInfo{sampleTopPlayer()}}
	topRepo := NewGameRepository(topDB)
	players, err := topRepo.ListTopPlayers(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected top players error: %v", err)
	}
	assertPlayers(t, players, sampleTopPlayer())

	db.savedField = game.Field
	joined, err := repo.JoinGame(context.Background(), "game-1", "user-o")
	if err != nil {
		t.Fatalf("unexpected join error: %v", err)
	}
	if joined.UUID != "game-1" {
		t.Fatalf("unexpected joined game: %#v", joined)
	}
}

func TestGameRepositoryUsesParameterizedQueries(t *testing.T) {
	t.Run("get game", func(t *testing.T) {
		db := &databaseStub{savedField: sampleGame().Field}
		repo := NewGameRepository(db)

		_, err := repo.GetGame(context.Background(), sqlInjectionPayload)
		if err != nil {
			t.Fatalf("unexpected get error: %v", err)
		}

		assertQueryDoesNotContainPayload(t, db.lastQueryRowQuery)
		assertArgsContainPayload(t, db.lastQueryRowArgs)
	})

	t.Run("completed games", func(t *testing.T) {
		db := &databaseStub{queryRows: &rowsStub{}}
		repo := NewGameRepository(db)

		_, err := repo.ListCompletedGamesByUserUUID(context.Background(), sqlInjectionPayload)
		if err != nil {
			t.Fatalf("unexpected list error: %v", err)
		}

		assertQueryDoesNotContainPayload(t, db.lastQueryQuery)
		assertArgsContainPayload(t, db.lastQueryArgs)
	})

	t.Run("join game", func(t *testing.T) {
		db := &databaseStub{savedField: sampleGame().Field}
		repo := NewGameRepository(db)

		_, err := repo.JoinGame(context.Background(), sqlInjectionPayload, "user-o")
		if err != nil {
			t.Fatalf("unexpected join error: %v", err)
		}

		assertQueryDoesNotContainPayload(t, db.lastQueryRowQuery)
		assertArgsContainPayload(t, db.lastQueryRowArgs)
	})
}

func TestGameRepositoryErrors(t *testing.T) {
	repo := NewGameRepository(&databaseStub{queryError: errors.New("query failed")})
	if _, err := repo.ListActiveGames(context.Background()); err == nil {
		t.Fatal("expected list query error")
	}

	repo = NewGameRepository(&databaseStub{queryRows: &rowsStub{scanErr: errors.New("scan failed"), games: []domain.Game{{UUID: "game-1"}}}})
	if _, err := repo.ListActiveGames(context.Background()); err == nil {
		t.Fatal("expected list scan error")
	}

	repo = NewGameRepository(&databaseStub{queryError: errors.New("query failed")})
	if _, err := repo.ListCompletedGamesByUserUUID(context.Background(), "user-1"); err == nil {
		t.Fatal("expected completed list query error")
	}

	repo = NewGameRepository(&databaseStub{queryError: errors.New("query failed")})
	if _, err := repo.ListTopPlayers(context.Background(), 10); err == nil {
		t.Fatal("expected top players query error")
	}

	repo = NewGameRepository(&databaseStub{queryScanErr: errors.New("scan failed"), topPlayers: []domain.WonGameInfo{sampleTopPlayer()}})
	if _, err := repo.ListTopPlayers(context.Background(), 10); err == nil {
		t.Fatal("expected top players scan error")
	}

	repo = NewGameRepository(&databaseStub{queryErr: pgx.ErrNoRows})
	if _, err := repo.JoinGame(context.Background(), "game-1", "user-o"); !errors.Is(err, ErrGameNotJoinable) {
		t.Fatalf("expected ErrGameNotJoinable, got %v", err)
	}
}

func sampleGame() domain.Game {
	return domain.Game{
		UUID:      "game-1",
		Mode:      domain.GameModeComputer,
		State:     domain.GameStatePlayerToMove,
		CreatedAt: time.Date(2026, 5, 15, 20, 0, 0, 0, time.UTC),
		Field: domain.Field{
			{1, 0, 0},
			{0, 2, 0},
			{0, 0, 0},
		},
	}
}

func sampleTopPlayer() domain.WonGameInfo {
	return domain.WonGameInfo{
		UserUUID: "user-x",
		Login:    "player",
		WinRatio: 1,
	}
}

func assertGame(t *testing.T, got domain.Game, want domain.Game) {
	t.Helper()

	if got.UUID != want.UUID || got.Field[1][1] != want.Field[1][1] {
		t.Fatalf("unexpected game: %#v", got)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("expected created_at %s, got %s", want.CreatedAt, got.CreatedAt)
	}
}

func assertGames(t *testing.T, got []domain.Game, want domain.Game) {
	t.Helper()

	if len(got) != 1 {
		t.Fatalf("unexpected games: %#v", got)
	}
	assertGame(t, got[0], want)
}

func assertPlayers(t *testing.T, got []domain.WonGameInfo, want domain.WonGameInfo) {
	t.Helper()

	if len(got) != 1 {
		t.Fatalf("unexpected top players: %#v", got)
	}
	if got[0] != want {
		t.Fatalf("unexpected top players: %#v", got)
	}
}

func assertStoredGameCloned(t *testing.T, got domain.Game) {
	t.Helper()

	if got.Field[0][0] != 1 {
		t.Fatal("expected stored game to be cloned through mapper")
	}
}

type rowsStub struct {
	games      []domain.Game
	topPlayers []domain.WonGameInfo
	index      int
	scanErr    error
	err        error
}

func (r *rowsStub) Close() {}

func (r *rowsStub) Err() error { return r.err }

func (r *rowsStub) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *rowsStub) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *rowsStub) Next() bool {
	if r.topPlayers != nil {
		if r.index >= len(r.topPlayers) {
			return false
		}
		r.index++
		return true
	}
	if r.index >= len(r.games) {
		return false
	}
	r.index++
	return true
}

func (r *rowsStub) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	if len(dest) == 3 && r.topPlayers != nil {
		player := r.topPlayers[r.index-1]
		setString(dest[0], player.UserUUID)
		setString(dest[1], player.Login)
		setFloat64(dest[2], player.WinRatio)
		return nil
	}
	game := r.games[r.index-1]
	field, err := json.Marshal(game.Field)
	if err != nil {
		return err
	}
	setString(dest[0], game.UUID)
	*(dest[1].(*[]byte)) = field
	setString(dest[2], string(game.Mode))
	setString(dest[3], string(game.State))
	setTime(dest[4], game.CreatedAt)
	setString(dest[5], game.NextPlayerUUID)
	setString(dest[6], game.WinnerUUID)
	setString(dest[7], game.PlayerXUUID)
	setString(dest[8], game.PlayerOUUID)
	return nil
}

func (r *rowsStub) Values() ([]any, error) { return nil, nil }

func (r *rowsStub) RawValues() [][]byte { return nil }

func (r *rowsStub) Conn() *pgx.Conn { return nil }

func (d *databaseStub) Close() {}

type txStub struct {
	db *databaseStub
}

func (t *txStub) Begin(context.Context) (pgx.Tx, error) {
	return t, nil
}

func (t *txStub) Commit(context.Context) error {
	t.db.commits++
	return nil
}

func (t *txStub) Rollback(context.Context) error {
	t.db.rollbacks++
	return nil
}

func (t *txStub) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (t *txStub) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return nil
}

func (t *txStub) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (t *txStub) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (t *txStub) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return t.db.Exec(ctx, sql, arguments...)
}

func (t *txStub) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return t.db.Query(ctx, sql, args...)
}

func (t *txStub) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.db.QueryRow(ctx, sql, args...)
}

func (t *txStub) Conn() *pgx.Conn {
	return nil
}

type rowStub struct {
	uuid      string
	field     []byte
	mode      string
	state     string
	createdAt time.Time
	err       error
}

func (r rowStub) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}

	if len(dest) == 3 {
		setString(dest[0], "user-1")
		setString(dest[1], "player")
		setString(dest[2], "hash")
		return nil
	}

	setString(dest[0], r.uuid)
	*(dest[1].(*[]byte)) = r.field
	setString(dest[2], r.mode)
	setString(dest[3], r.state)
	setTime(dest[4], r.createdAt)
	setString(dest[5], "")
	setString(dest[6], "")
	setString(dest[7], "")
	setString(dest[8], "")
	return nil
}

func setString(dest any, value string) {
	switch target := dest.(type) {
	case *string:
		*target = value
	case *sql.NullString:
		target.String = value
		target.Valid = true
	}
}

func setTime(dest any, value time.Time) {
	switch target := dest.(type) {
	case *time.Time:
		*target = value
	case *sql.NullTime:
		target.Time = value
		target.Valid = true
	}
}

func setFloat64(dest any, value float64) {
	switch target := dest.(type) {
	case *float64:
		*target = value
	case *sql.NullFloat64:
		target.Float64 = value
		target.Valid = true
	}
}
