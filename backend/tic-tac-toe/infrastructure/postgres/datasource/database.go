package datasource

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"tic-tac-toe/internal/config"
)

const (
	defaultDatabaseURL = "postgres://postgres:postgres@localhost:5432/tic_tac_toe?sslmode=disable"
	migrationTimeout   = 5 * time.Second
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Database interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Ping(ctx context.Context) error
	Close()
}

type DatabaseConfig struct {
	DatabaseURL string
}

func (c DatabaseConfig) Validate() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf(errDatabaseURLMustNotBeEmpty)
	}
	if _, err := pgxpool.ParseConfig(c.DatabaseURL); err != nil {
		return fmt.Errorf("%s: %w", errInvalidDatabaseURL, err)
	}
	return nil
}

type databaseManager struct {
	openPool   func(context.Context, string) (Database, error)
	ping       func(context.Context, *sql.DB) error
	setDialect func(string) error
	gooseUp    func(*sql.DB, string, ...goose.OptionsFunc) error
}

func NewDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{DatabaseURL: config.String("DATABASE_URL", defaultDatabaseURL)}
}

func NewDatabase(config DatabaseConfig) (Database, error) {
	return newDatabaseManager().NewDatabase(config)
}

func newDatabaseManager() databaseManager {
	return databaseManager{
		openPool:   defaultOpenPool,
		ping:       defaultPing,
		setDialect: defaultSetDialect,
		gooseUp:    defaultGooseUp,
	}
}

func (m databaseManager) withDefaults() databaseManager {
	if m.openPool == nil {
		m.openPool = defaultOpenPool
	}
	if m.ping == nil {
		m.ping = defaultPing
	}
	if m.setDialect == nil {
		m.setDialect = defaultSetDialect
	}
	if m.gooseUp == nil {
		m.gooseUp = defaultGooseUp
	}
	return m
}

func (m databaseManager) NewDatabase(config DatabaseConfig) (Database, error) {
	m = m.withDefaults()
	logDatabase(msgOpeningDatabaseConnection)

	pool, err := m.openPool(context.Background(), config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errOpenDatabase, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%s: %w", errPingDatabase, err)
	}

	if err := m.runMigrations(config.DatabaseURL); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%s: %w", errRunMigrations, err)
	}

	logDatabase(msgDatabaseReady)
	return pool, nil
}

func (m databaseManager) runMigrations(databaseURL string) error {
	m = m.withDefaults()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("%s: %w", errParseDatabaseURLForMigration, err)
	}

	db := stdlib.OpenDB(*config.ConnConfig)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	if err := m.ping(ctx, db); err != nil {
		return fmt.Errorf("%s: %w", errPingDatabaseForMigration, err)
	}

	if err := m.runGoose(db); err != nil {
		return err
	}

	return nil
}

func (m databaseManager) runGoose(db *sql.DB) error {
	m = m.withDefaults()
	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())

	if err := m.setDialect("postgres"); err != nil {
		return fmt.Errorf("%s: %w", errSetGooseDialect, err)
	}
	if err := m.gooseUp(db, "migrations"); err != nil {
		return fmt.Errorf("%s: %w", errRunMigrations, err)
	}

	return nil
}

func defaultOpenPool(ctx context.Context, databaseURL string) (Database, error) {
	return pgxpool.New(ctx, databaseURL)
}

func defaultPing(ctx context.Context, db *sql.DB) error {
	return db.PingContext(ctx)
}

func defaultSetDialect(dialect string) error {
	return goose.SetDialect(dialect)
}

func defaultGooseUp(db *sql.DB, migrationsDir string, options ...goose.OptionsFunc) error {
	return goose.Up(db, migrationsDir, options...)
}
