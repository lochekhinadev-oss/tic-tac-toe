package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	enabledEnv         = "CLEANUP_ENABLED"
	intervalEnv        = "CLEANUP_INTERVAL"
	retentionDaysEnv   = "DATA_RETENTION_DAYS"
	batchSizeEnv       = "CLEANUP_BATCH_SIZE"
	metricsAddrEnv     = "CLEANUP_METRICS_ADDR"
	defaultInterval    = 24 * time.Hour
	defaultRetention   = 180 * 24 * time.Hour
	defaultBatchSize   = 1000
	defaultMetricsAddr = ":9091"
	runTimeout         = 10 * time.Minute
)

type config struct {
	databaseURL string
	enabled     bool
	interval    time.Duration
	retention   time.Duration
	batchSize   int
}

type database interface {
	Ping(ctx context.Context) error
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type cleanupResult struct {
	UsersDeleted int64 `json:"usersDeleted"`
	GamesDeleted int64 `json:"gamesDeleted"`
}

func main() {
	cfg := parseConfig()
	if !cfg.enabled {
		log.Printf("cleanup worker disabled: %s is not enabled", enabledEnv)
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.databaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	if err := waitForDatabase(ctx, pool); err != nil {
		log.Fatalf("database not ready: %v", err)
	}

	metrics := newCleanupMetrics()
	if err := runCleanupOnce(ctx, pool, cfg, metrics); err != nil {
		log.Fatalf("cleanup failed: %v", err)
	}
	startMetricsServer(ctx, envString(metricsAddrEnv, defaultMetricsAddr), metrics)

	ticker := time.NewTicker(cfg.interval)
	defer ticker.Stop()

	log.Printf("cleanup worker started: interval=%s retention=%s batch=%d", cfg.interval, cfg.retention, cfg.batchSize)
	for {
		select {
		case <-ctx.Done():
			log.Printf("cleanup worker stopped: %v", ctx.Err())
			return
		case <-ticker.C:
			if err := runCleanupOnce(ctx, pool, cfg, metrics); err != nil {
				log.Printf("cleanup failed: %v", err)
			}
		}
	}
}

func parseConfig() config {
	cfg := config{
		databaseURL: envString("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/tic_tac_toe?sslmode=disable"),
		enabled:     envBool(enabledEnv, true),
		interval:    envDuration(intervalEnv, defaultInterval),
		retention:   envDuration(retentionDaysEnv, defaultRetention),
		batchSize:   envInt(batchSizeEnv, defaultBatchSize),
	}

	flag.StringVar(&cfg.databaseURL, "database-url", cfg.databaseURL, "Postgres connection URL")
	flag.BoolVar(&cfg.enabled, "enabled", cfg.enabled, "Enable cleanup worker")
	flag.DurationVar(&cfg.interval, "interval", cfg.interval, "Cleanup interval")
	flag.DurationVar(&cfg.retention, "retention", cfg.retention, "Data retention period")
	flag.IntVar(&cfg.batchSize, "batch-size", cfg.batchSize, "Rows deleted per batch")
	flag.Parse()

	if cfg.batchSize <= 0 {
		log.Fatal("batch size must be positive")
	}
	if cfg.retention <= 0 {
		log.Fatal("retention must be positive")
	}
	if cfg.interval <= 0 {
		log.Fatal("interval must be positive")
	}
	return cfg
}

func waitForDatabase(ctx context.Context, db database) error {
	deadline := time.Now().Add(30 * time.Second)
	for {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := db.Ping(pingCtx)
		cancel()
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func runCleanupOnce(ctx context.Context, db database, cfg config, metrics *cleanupMetrics) error {
	startedAt := time.Now().UTC()
	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	result, err := runCleanup(runCtx, db, cfg)
	if metrics != nil {
		metrics.record(startedAt, result, err)
	}
	return err
}

func runCleanup(ctx context.Context, db database, cfg config) (cleanupResult, error) {
	cutoff := time.Now().UTC().Add(-cfg.retention)
	log.Printf("cleanup run started cutoff=%s", cutoff.Format(time.RFC3339))

	users, err := deleteInBatches(ctx, db, deleteOldUsersQuery, cutoff, cfg.batchSize)
	if err != nil {
		return cleanupResult{}, fmt.Errorf("delete old users: %w", err)
	}
	games, err := deleteInBatches(ctx, db, deleteOldGamesQuery, cutoff, cfg.batchSize)
	if err != nil {
		return cleanupResult{}, fmt.Errorf("delete old games: %w", err)
	}

	log.Printf("cleanup run finished users=%d games=%d", users, games)
	return cleanupResult{UsersDeleted: users, GamesDeleted: games}, nil
}

func deleteInBatches(ctx context.Context, db database, query string, cutoff time.Time, batchSize int) (int64, error) {
	var total int64
	for {
		tag, err := db.Exec(ctx, query, cutoff, batchSize)
		if err != nil {
			return total, err
		}
		rows := tag.RowsAffected()
		total += rows
		if rows < int64(batchSize) {
			return total, nil
		}
	}
}

const deleteOldUsersQuery = `
WITH doomed AS (
	SELECT uuid
	FROM users
	WHERE deleted_at IS NOT NULL
	  AND deleted_at < $1
	ORDER BY deleted_at ASC
	LIMIT $2
)
DELETE FROM users
WHERE uuid IN (SELECT uuid FROM doomed)`

const deleteOldGamesQuery = `
WITH doomed AS (
	SELECT uuid
	FROM games
	WHERE state IN ('player_wins', 'draw')
	  AND created_at < $1
	ORDER BY created_at ASC
	LIMIT $2
)
DELETE FROM games
WHERE uuid IN (SELECT uuid FROM doomed)`

func envString(name string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envBool(name string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	case "":
		return fallback
	default:
		return fallback
	}
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	if days, err := strconv.Atoi(value); err == nil && name == retentionDaysEnv {
		return time.Duration(days) * 24 * time.Hour
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func startMetricsServer(ctx context.Context, addr string, metrics *cleanupMetrics) {
	if strings.TrimSpace(addr) == "" || metrics == nil {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.Handle("/metrics", metrics.handler())

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("cleanup metrics server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("cleanup metrics server stopped: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
}
