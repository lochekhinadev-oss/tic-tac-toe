package datasource

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pressly/goose/v3"
)

const testDatabaseURL = "postgres://postgres:postgres@localhost:5432/tic_tac_toe?sslmode=disable"

type databaseFake struct {
	pingErr error
	closed  bool
}

func (d *databaseFake) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (d *databaseFake) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }

func (d *databaseFake) QueryRow(context.Context, string, ...any) pgx.Row { return nil }

func (d *databaseFake) Ping(context.Context) error { return d.pingErr }

func (d *databaseFake) Close() { d.closed = true }

func TestNewDatabaseSuccessAndMigrationError(t *testing.T) {
	fake := &databaseFake{}
	manager := newTestDatabaseManager(fake, nil)

	db, err := manager.NewDatabase(NewDatabaseConfig())
	if err != nil {
		t.Fatalf("unexpected new database error: %v", err)
	}
	assertDatabaseResult(t, db, fake, err, nil)

	migrationErr := errors.New("migration failed")
	manager = newTestDatabaseManager(fake, migrationErr)
	db, err = manager.NewDatabase(NewDatabaseConfig())
	assertDatabaseResult(t, db, nil, err, migrationErr)
	if !fake.closed {
		t.Fatal("expected pool to be closed after migration error")
	}
}

func TestNewDatabasePoolAndPingErrors(t *testing.T) {
	poolErr := errors.New("pool failed")
	manager := databaseManager{openPool: func(context.Context, string) (Database, error) { return nil, poolErr }}
	if db, err := manager.NewDatabase(NewDatabaseConfig()); !errors.Is(err, poolErr) || db != nil {
		t.Fatalf("expected pool error, db=%#v err=%v", db, err)
	}

	fake := &databaseFake{pingErr: errors.New("ping failed")}
	manager = newTestDatabaseManager(fake, nil)
	manager.ping = func(context.Context, *sql.DB) error { return fake.Ping(context.Background()) }
	if db, err := manager.NewDatabase(NewDatabaseConfig()); err == nil || db != nil || !fake.closed {
		t.Fatalf("expected ping error and closed pool, db=%#v err=%v closed=%v", db, err, fake.closed)
	}
}

func TestNewDatabaseRejectsInvalidDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "://bad-url")

	db, err := NewDatabase(NewDatabaseConfig())
	if err == nil {
		if db != nil {
			db.Close()
		}
		t.Fatal("expected invalid database url error")
	}
}

func TestRunMigrationsRejectsInvalidDatabaseURL(t *testing.T) {
	if err := newDatabaseManager().runMigrations("://bad-url"); err == nil {
		t.Fatal("expected invalid database url error")
	}
}

func TestRunMigrationsSuccessAndPingError(t *testing.T) {
	manager := databaseManager{
		ping:       func(context.Context, *sql.DB) error { return nil },
		setDialect: func(string) error { return nil },
		gooseUp:    func(*sql.DB, string, ...goose.OptionsFunc) error { return nil },
	}
	if err := manager.runMigrations(testDatabaseURL); err != nil {
		t.Fatalf("unexpected migration success error: %v", err)
	}

	pingErr := errors.New("ping failed")
	manager = databaseManager{ping: func(context.Context, *sql.DB) error { return pingErr }}
	if err := manager.runMigrations(testDatabaseURL); !errors.Is(err, pingErr) {
		t.Fatalf("expected ping error, got %v", err)
	}
}

func TestRunGooseReturnsMigrationError(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://postgres:postgres@127.0.0.1:1/missing?sslmode=disable")
	if err != nil {
		t.Fatalf("open pgx db: %v", err)
	}
	defer db.Close()

	if err := newDatabaseManager().runGoose(db); err == nil {
		t.Fatal("expected migration error")
	}
}

func TestRunGooseSuccessAndDialectError(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://postgres:postgres@127.0.0.1:1/missing?sslmode=disable")
	if err != nil {
		t.Fatalf("open pgx db: %v", err)
	}
	defer db.Close()

	manager := databaseManager{setDialect: func(string) error { return nil }, gooseUp: func(*sql.DB, string, ...goose.OptionsFunc) error { return nil }}
	if err := manager.runGoose(db); err != nil {
		t.Fatalf("unexpected goose success error: %v", err)
	}

	dialectErr := errors.New("dialect failed")
	manager = databaseManager{setDialect: func(string) error { return dialectErr }}
	if err := manager.runGoose(db); !errors.Is(err, dialectErr) {
		t.Fatalf("expected dialect error, got %v", err)
	}
}

func newTestDatabaseManager(fake Database, gooseErr error) databaseManager {
	return databaseManager{
		openPool: func(context.Context, string) (Database, error) { return fake, nil },
		ping:     func(context.Context, *sql.DB) error { return nil },
		setDialect: func(string) error {
			return nil
		},
		gooseUp: func(*sql.DB, string, ...goose.OptionsFunc) error {
			return gooseErr
		},
	}
}

func assertDatabaseResult(t *testing.T, got Database, want Database, err error, wantErr error) {
	t.Helper()

	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected error %v, got %v", wantErr, err)
		}
	}
	if got != want {
		t.Fatalf("unexpected database result: %#v", got)
	}
}
