package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metrics.handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	for _, want := range []string{
		"tic_tac_toe_cleanup_runs_total{result=\"success\"} 1",
		"tic_tac_toe_cleanup_deleted_rows_total{entity=\"users\"} 3",
		"tic_tac_toe_cleanup_deleted_rows_total{entity=\"games\"} 4",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected metrics body to contain %q, got %q", want, body)
		}
	}
	if strings.Contains(body, "\"runCount\"") {
		t.Fatalf("expected prometheus metrics, got json-like body: %q", body)
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

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metrics.handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "tic_tac_toe_cleanup_runs_total{result=\"failure\"} 1") {
		t.Fatalf("expected failure counter, got %q", body)
	}
	if strings.Contains(body, "\"lastError\"") {
		t.Fatalf("expected prometheus metrics, got json-like body: %q", body)
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
