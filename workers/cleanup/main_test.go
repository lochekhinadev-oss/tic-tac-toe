package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestEnvDurationParsesRetentionDays(t *testing.T) {
	t.Setenv(retentionDaysEnv, "180")
	if got := envDuration(retentionDaysEnv, time.Hour); got != 180*24*time.Hour {
		t.Fatalf("expected 180 days, got %s", got)
	}
}

func TestRunCleanupDeletesOldDataInBatches(t *testing.T) {
	db := &cleanupDatabaseStub{
		rows: map[string][]int64{
			deleteOldUsersQuery: {2, 1},
			deleteOldGamesQuery: {2, 2, 0},
		},
	}
	cfg := config{
		retention:   180 * 24 * time.Hour,
		batchSize:   2,
		interval:    24 * time.Hour,
		enabled:     true,
		databaseURL: "postgres://example",
	}
	metrics := newCleanupMetrics()

	if err := runCleanupOnce(context.Background(), db, cfg, metrics); err != nil {
		t.Fatalf("unexpected cleanup error: %v", err)
	}

	if db.calls[deleteOldUsersQuery] != 2 {
		t.Fatalf("expected 2 user cleanup calls, got %d", db.calls[deleteOldUsersQuery])
	}
	if db.calls[deleteOldGamesQuery] != 3 {
		t.Fatalf("expected 3 game cleanup calls, got %d", db.calls[deleteOldGamesQuery])
	}
	if db.lastBatchSize != 2 {
		t.Fatalf("expected batch size 2, got %d", db.lastBatchSize)
	}
	if db.lastCutoff.IsZero() {
		t.Fatal("expected cutoff to be passed")
	}

	snapshot := metrics.snapshot()
	if snapshot["runCount"].(int64) != 1 {
		t.Fatalf("expected one run, got %#v", snapshot["runCount"])
	}
	if snapshot["successCount"].(int64) != 1 {
		t.Fatalf("expected one success, got %#v", snapshot["successCount"])
	}
	if snapshot["usersDeleted"].(int64) != 3 {
		t.Fatalf("expected 3 deleted users, got %#v", snapshot["usersDeleted"])
	}
	if snapshot["gamesDeleted"].(int64) != 4 {
		t.Fatalf("expected 4 deleted games, got %#v", snapshot["gamesDeleted"])
	}
	if snapshot["lastError"].(string) != "" {
		t.Fatalf("expected empty last error, got %#v", snapshot["lastError"])
	}
}

func TestRunCleanupRecordsError(t *testing.T) {
	db := &cleanupDatabaseStub{
		rows: map[string][]int64{
			deleteOldUsersQuery: {0},
		},
	}
	cfg := config{
		retention:   180 * 24 * time.Hour,
		batchSize:   1,
		interval:    24 * time.Hour,
		enabled:     true,
		databaseURL: "postgres://example",
	}
	metrics := newCleanupMetrics()

	db.execErr = fmt.Errorf("boom")
	if err := runCleanupOnce(context.Background(), db, cfg, metrics); err == nil {
		t.Fatal("expected cleanup error")
	}

	snapshot := metrics.snapshot()
	if snapshot["runCount"].(int64) != 1 {
		t.Fatalf("expected one run, got %#v", snapshot["runCount"])
	}
	if snapshot["successCount"].(int64) != 0 {
		t.Fatalf("expected zero successes, got %#v", snapshot["successCount"])
	}
	if snapshot["lastError"].(string) == "" {
		t.Fatal("expected last error to be recorded")
	}
}

type cleanupDatabaseStub struct {
	rows          map[string][]int64
	calls         map[string]int
	lastBatchSize int
	lastCutoff    time.Time
	execErr       error
}

func (d *cleanupDatabaseStub) Ping(context.Context) error { return nil }

func (d *cleanupDatabaseStub) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if d.execErr != nil {
		return pgconn.CommandTag{}, d.execErr
	}
	if d.calls == nil {
		d.calls = make(map[string]int)
	}
	d.calls[sql]++
	if len(arguments) == 2 {
		if cutoff, ok := arguments[0].(time.Time); ok {
			d.lastCutoff = cutoff
		}
		if batchSize, ok := arguments[1].(int); ok {
			d.lastBatchSize = batchSize
		}
	}
	seq := d.rows[sql]
	if len(seq) == 0 {
		return pgconn.NewCommandTag("DELETE 0"), nil
	}
	rows := seq[0]
	d.rows[sql] = seq[1:]
	return pgconn.NewCommandTag(fmt.Sprintf("DELETE %d", rows)), nil
}
