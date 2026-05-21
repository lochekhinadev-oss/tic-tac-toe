package config

import (
	"testing"
	"time"
)

func TestStringAndDuration(t *testing.T) {
	t.Setenv("TEST_STRING", " value ")
	if got := String("TEST_STRING", "fallback"); got != "value" {
		t.Fatalf("expected trimmed string, got %q", got)
	}

	t.Setenv("TEST_EMPTY", "   ")
	if got := String("TEST_EMPTY", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback string, got %q", got)
	}

	t.Setenv("TEST_DURATION", "15s")
	if got := Duration("TEST_DURATION", time.Minute); got != 15*time.Second {
		t.Fatalf("expected parsed duration, got %s", got)
	}

	t.Setenv("TEST_DURATION_BAD", "bad")
	if got := Duration("TEST_DURATION_BAD", time.Minute); got != time.Minute {
		t.Fatalf("expected fallback duration, got %s", got)
	}
}
