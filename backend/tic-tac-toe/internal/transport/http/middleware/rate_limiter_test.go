package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIPRateLimiterRejectsRequestsAfterLimit(t *testing.T) {
	limiter := NewIPRateLimiter(IPRateLimiterConfig{Limit: 1, Window: time.Minute})
	limiter.now = func() time.Time {
		return time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	}

	handler := limiter.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	first := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodGet, "/games", nil)
	firstReq.RemoteAddr = "192.0.2.1:1234"
	handler.ServeHTTP(first, firstReq)
	if first.Code != http.StatusNoContent {
		t.Fatalf("expected first request status %d, got %d", http.StatusNoContent, first.Code)
	}

	second := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodGet, "/games", nil)
	secondReq.RemoteAddr = "192.0.2.1:4321"
	handler.ServeHTTP(second, secondReq)

	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request status %d, got %d", http.StatusTooManyRequests, second.Code)
	}
	assertRateLimitMessage(t, second)
}

func TestIPRateLimiterAllowsRequestsAfterWindow(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	limiter := NewIPRateLimiter(IPRateLimiterConfig{Limit: 1, Window: time.Minute})
	limiter.now = func() time.Time { return now }

	if !limiter.allow("192.0.2.1") {
		t.Fatal("expected first request to pass")
	}

	now = now.Add(time.Minute + time.Second)
	if !limiter.allow("192.0.2.1") {
		t.Fatal("expected request after window to pass")
	}
}

func TestIPRateLimiterCleansExpiredKeys(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	limiter := NewIPRateLimiter(IPRateLimiterConfig{Limit: 10, Window: time.Minute})
	limiter.now = func() time.Time { return now }

	if !limiter.allow("192.0.2.1") {
		t.Fatal("expected first key to pass")
	}
	if !limiter.allow("192.0.2.2") {
		t.Fatal("expected second key to pass")
	}

	now = now.Add(time.Minute + time.Second)
	if !limiter.allow("192.0.2.3") {
		t.Fatal("expected new key to pass")
	}

	if _, ok := limiter.requests["192.0.2.1"]; ok {
		t.Fatal("expected expired first key to be removed")
	}
	if _, ok := limiter.requests["192.0.2.2"]; ok {
		t.Fatal("expected expired second key to be removed")
	}
}

func assertRateLimitMessage(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	var payload struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Message != "too many requests" {
		t.Fatalf("unexpected message %q", payload.Message)
	}
}
