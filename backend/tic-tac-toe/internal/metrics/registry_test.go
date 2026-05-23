package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPrometheusMetricsExposeCounters(t *testing.T) {
	ResetForTests()

	ObserveHTTPRequest("/games/{uuid}", "GET", http.StatusOK, 150*time.Millisecond)
	ObserveHTTPRequest("/games/{uuid}", "GET", http.StatusOK, 50*time.Millisecond)
	ObserveHTTPRequest("/auth/sessions", "POST", http.StatusUnauthorized, 10*time.Millisecond)
	ObserveAuthEvent("auth_ok")
	ObserveAuthEvent("authz_rejected")
	ObserveGameEvent("game_created")
	ObserveGameEvent("game_finished")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"# HELP tic_tac_toe_http_requests_total",
		"tic_tac_toe_http_requests_total{method=\"GET\",route=\"/games/{uuid}\",status=\"200\"} 2",
		"tic_tac_toe_http_requests_total{method=\"POST\",route=\"/auth/sessions\",status=\"401\"} 1",
		"# HELP tic_tac_toe_auth_events_total",
		"tic_tac_toe_auth_events_total{event=\"auth_ok\"} 1",
		"tic_tac_toe_auth_events_total{event=\"authz_rejected\"} 1",
		"# HELP tic_tac_toe_game_events_total",
		"tic_tac_toe_game_events_total{event=\"game_created\"} 1",
		"tic_tac_toe_game_events_total{event=\"game_finished\"} 1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected metrics body to contain %q, got %q", want, body)
		}
	}
	if strings.Contains(body, `"httpRequests"`) {
		t.Fatalf("expected prometheus exposition, got json-looking body: %q", body)
	}
}
