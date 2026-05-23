package metrics

import (
	"testing"
	"time"
)

func TestObserveHTTPRequestSnapshot(t *testing.T) {
	ResetAuthEventStats()
	ResetHTTPRequestStats()

	ObserveHTTPRequest("/games/{uuid}", "GET", 200, 150*time.Millisecond)
	ObserveHTTPRequest("/games/{uuid}", "GET", 200, 50*time.Millisecond)
	ObserveHTTPRequest("/auth/sessions", "POST", 401, 10*time.Millisecond)

	stats := SnapshotHTTPRequestStats()
	if len(stats) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(stats))
	}

	if stats[0].Route != "/auth/sessions" || stats[0].Method != "POST" || stats[0].Status != 401 || stats[0].Count != 1 {
		t.Fatalf("unexpected first stat: %#v", stats[0])
	}

	if stats[1].Route != "/games/{uuid}" || stats[1].Method != "GET" || stats[1].Status != 200 || stats[1].Count != 2 {
		t.Fatalf("unexpected second stat: %#v", stats[1])
	}

	if stats[1].DurationMS != 200 {
		t.Fatalf("expected duration sum 200ms, got %d", stats[1].DurationMS)
	}
	if stats[1].AverageDurationMS != 100 {
		t.Fatalf("expected average duration 100ms, got %d", stats[1].AverageDurationMS)
	}
}

func TestObserveAuthEventSnapshot(t *testing.T) {
	ResetAuthEventStats()

	ObserveAuthEvent("auth_ok")
	ObserveAuthEvent("auth_ok")
	ObserveAuthEvent("authz_rejected")

	stats := SnapshotAuthEventStats()
	if len(stats) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(stats))
	}

	if stats[0].Event != "auth_ok" || stats[0].Count != 2 {
		t.Fatalf("unexpected first stat: %#v", stats[0])
	}
	if stats[1].Event != "authz_rejected" || stats[1].Count != 1 {
		t.Fatalf("unexpected second stat: %#v", stats[1])
	}
}

func TestObserveGameEventSnapshot(t *testing.T) {
	ResetGameEventStats()

	ObserveGameEvent("game_created")
	ObserveGameEvent("game_created")
	ObserveGameEvent("game_finished")

	stats := SnapshotGameEventStats()
	if len(stats) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(stats))
	}

	if stats[0].Event != "game_created" || stats[0].Count != 2 {
		t.Fatalf("unexpected first stat: %#v", stats[0])
	}
	if stats[1].Event != "game_finished" || stats[1].Count != 1 {
		t.Fatalf("unexpected second stat: %#v", stats[1])
	}
}
