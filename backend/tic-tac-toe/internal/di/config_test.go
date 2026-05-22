package di

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
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

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes})

	return auth.AuthConfig{
		JWTPrivateKeyPEM: string(privatePEM),
		JWTPublicKeyPEM:  string(publicPEM),
		JWTKeyID:         "kid-1",
		JWTIssuer:        "tic-tac-toe",
		JWTAudience:      "tic-tac-toe-api",
		AccessTokenTTL:   time.Minute,
		RefreshTokenTTL:  time.Hour,
	}
}

func TestValidateConfigsRejectsInvalidValues(t *testing.T) {
	err := ValidateConfigs(
		datasource.DatabaseConfig{DatabaseURL: "://bad-url"},
		auth.AuthConfig{JWTPrivateKeyPEM: "", JWTPublicKeyPEM: "", JWTKeyID: "", JWTIssuer: "", JWTAudience: "", AccessTokenTTL: 0, RefreshTokenTTL: 0},
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
