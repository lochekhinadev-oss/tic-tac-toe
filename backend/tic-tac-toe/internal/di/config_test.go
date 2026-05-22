package di

import (
	"strings"
	"testing"
	"time"

	"tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/infrastructure/rediscache"
)

func TestValidateConfigs(t *testing.T) {
	authConfig := testAuthConfig(t)
	if err := ValidateConfigs(
		datasource.DatabaseConfig{DatabaseURL: "postgres://postgres:postgres@localhost:5432/tic_tac_toe?sslmode=disable"},
		authConfig,
		rediscache.Config{URL: "redis://localhost:6379/0"},
		HTTPConfig{Port: "8080"},
	); err != nil {
		t.Fatalf("expected valid configs, got %v", err)
	}
}

func TestNormalizeHTTPPort(t *testing.T) {
	cases := map[string]string{
		"":       ":8080",
		"8080":   ":8080",
		":8080":  ":8080",
		" 8080 ": ":8080",
	}

	for input, expected := range cases {
		if got := normalizeHTTPPort(input); got != expected {
			t.Fatalf("normalizeHTTPPort(%q) = %q, want %q", input, got, expected)
		}
	}
}

func testAuthConfig(t *testing.T) auth.AuthConfig {
	t.Helper()
	return auth.AuthConfig{
		SessionCookieName: "tic-tac-toe.session",
		SessionTTL:        time.Hour,
	}
}

func TestValidateConfigsRejectsInvalidValues(t *testing.T) {
	err := ValidateConfigs(
		datasource.DatabaseConfig{DatabaseURL: "://bad-url"},
		auth.AuthConfig{SessionCookieName: "", SessionTTL: 0},
		rediscache.Config{URL: "redis://localhost:6379/0"},
		HTTPConfig{Port: ""},
	)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if got := err.Error(); !strings.Contains(got, "database config") || !strings.Contains(got, "auth config") || !strings.Contains(got, "http config") {
		t.Fatalf("expected combined validation error, got %v", err)
	}
}
