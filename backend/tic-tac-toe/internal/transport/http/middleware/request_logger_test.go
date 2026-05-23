package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestLoggerWritesStructuredLog(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
	output := buf.String()
	if !strings.Contains(output, "http request") || !strings.Contains(output, "method=GET") || !strings.Contains(output, "path=/healthz") {
		t.Fatalf("unexpected log output: %s", output)
	}
}

func TestRequestLoggerDoesNotLeakSecrets(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/sessions", strings.NewReader(`{"login":"player","password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "tic-tac-toe.session", Value: "session-1"})

	handler.ServeHTTP(rec, req)

	output := buf.String()
	for _, secret := range []string{"secret", "session-1", "tic-tac-toe.session"} {
		if strings.Contains(output, secret) {
			t.Fatalf("request log leaked %q: %s", secret, output)
		}
	}
	if !strings.Contains(output, "http request") || !strings.Contains(output, "method=POST") || !strings.Contains(output, "path=/auth/sessions") {
		t.Fatalf("unexpected log output: %s", output)
	}
}
