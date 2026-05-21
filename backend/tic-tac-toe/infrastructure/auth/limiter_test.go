package auth

import (
	"testing"
	"time"
)

func TestAuthActionLimiterCleansExpiredKeys(t *testing.T) {
	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	limiter := newAuthActionLimiter(10, time.Minute)
	limiter.now = func() time.Time { return now }

	if !limiter.Allow("login:alice") {
		t.Fatal("expected first key to pass")
	}
	if !limiter.Allow("login:bob") {
		t.Fatal("expected second key to pass")
	}

	now = now.Add(time.Minute + time.Second)
	if !limiter.Allow("login:cara") {
		t.Fatal("expected new key to pass")
	}

	if _, ok := limiter.events["login:alice"]; ok {
		t.Fatal("expected expired first key to be removed")
	}
	if _, ok := limiter.events["login:bob"]; ok {
		t.Fatal("expected expired second key to be removed")
	}
}
