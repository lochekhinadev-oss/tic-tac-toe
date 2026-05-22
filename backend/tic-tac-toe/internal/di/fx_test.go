package di

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"tic-tac-toe/internal/transport/http/handler"
	"tic-tac-toe/internal/transport/http/middleware"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("expected non-nil app")
	}
}

func TestAppModule(t *testing.T) {
	configureTestEnv(t)
	if err := fx.ValidateApp(AppModule); err != nil {
		t.Fatalf("invalid fx app graph: %v", err)
	}
}

func configureTestEnv(t *testing.T) {
	t.Helper()

	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/tic_tac_toe?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
	t.Setenv("HTTP_PORT", "8080")
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_KEY_ID", "tic-tac-toe-main")
	t.Setenv("JWT_ISSUER", "tic-tac-toe")
	t.Setenv("JWT_AUDIENCE", "tic-tac-toe-api")
	t.Setenv("JWT_ACCESS_TTL", "15m")
	t.Setenv("JWT_REFRESH_TTL", "168h")

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

	t.Setenv("JWT_PRIVATE_KEY_PEM", string(privatePEM))
	t.Setenv("JWT_PUBLIC_KEY_PEM", string(publicPEM))

	dir := t.TempDir()
	privatePath := filepath.Join(dir, "private.pem")
	publicPath := filepath.Join(dir, "public.pem")
	if err := os.WriteFile(privatePath, privatePEM, 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.WriteFile(publicPath, publicPEM, 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}

	t.Setenv("JWT_PRIVATE_KEY_PATH", privatePath)
	t.Setenv("JWT_PUBLIC_KEY_PATH", publicPath)
}

func TestNewRouter(t *testing.T) {
	gameHandler := handler.NewGameHandler(gameLogicStub{}, gameStorageStub{}, gameStorageStub{})
	authHandler := handler.NewAuthHandler(authStub{})
	userHandler := handler.NewUserHandler(userServiceStub{})
	db := &databaseStub{}
	router := NewRouter(gameHandler, authHandler, userHandler, middleware.NewUserAuthenticator(authStub{}), db)

	tests := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{method: http.MethodGet, path: "/healthz", status: http.StatusOK},
		{method: http.MethodGet, path: "/readyz", status: http.StatusOK},
		{method: http.MethodGet, path: "/metrics", status: http.StatusOK},
		{method: http.MethodGet, path: "/swagger", status: http.StatusOK},
		{method: http.MethodGet, path: "/openapi.yaml", status: http.StatusOK},
		{method: http.MethodGet, path: "/swagger/doc.json", status: http.StatusOK},
		{method: http.MethodPost, path: "/users", body: `{"login":"player","password":"secret"}`, status: http.StatusCreated},
		{method: http.MethodPost, path: "/auth/sessions", body: `{"login":"player","password":"secret"}`, status: http.StatusOK},
		{method: http.MethodPost, path: "/auth/tokens/access", body: `{"refreshToken":"refresh"}`, status: http.StatusOK},
		{method: http.MethodPost, path: "/auth/tokens/refresh", body: `{"refreshToken":"refresh"}`, status: http.StatusOK},
		{method: http.MethodDelete, path: "/auth/sessions/current", body: `{"refreshToken":"refresh"}`, status: http.StatusNoContent},
		{method: http.MethodDelete, path: "/auth/sessions", body: `{"refreshToken":"refresh"}`, status: http.StatusNoContent},
		{method: http.MethodPost, path: "/signup", body: `{"login":"player","password":"secret"}`, status: http.StatusCreated},
		{method: http.MethodPost, path: "/auth", body: `{"login":"player","password":"secret"}`, status: http.StatusOK},
		{method: http.MethodPost, path: "/auth/refresh/access", body: `{"refreshToken":"refresh"}`, status: http.StatusOK},
		{method: http.MethodPost, path: "/auth/refresh", body: `{"refreshToken":"refresh"}`, status: http.StatusOK},
		{method: http.MethodPost, path: "/auth/logout", body: `{"refreshToken":"refresh"}`, status: http.StatusNoContent},
		{method: http.MethodPost, path: "/auth/logout/all", body: `{"refreshToken":"refresh"}`, status: http.StatusNoContent},
		{method: http.MethodGet, path: "/auth", status: http.StatusMethodNotAllowed},
		{method: http.MethodGet, path: "/missing", status: http.StatusNotFound},
		{method: http.MethodPost, path: "/games", status: http.StatusCreated},
		{method: http.MethodGet, path: "/games", status: http.StatusOK},
		{method: http.MethodGet, path: "/games/history", status: http.StatusOK},
		{method: http.MethodGet, path: "/games/leaderboard?n=10", status: http.StatusOK},
		{method: http.MethodGet, path: "/games/123e4567-e89b-42d3-a456-426614174000", status: http.StatusOK},
		{method: http.MethodPost, path: "/games/123e4567-e89b-42d3-a456-426614174000/join", status: http.StatusOK},
		{method: http.MethodPost, path: "/games/123e4567-e89b-42d3-a456-426614174000/move", body: `{"field":[[1,0,0],[0,0,0],[0,0,0]]}`, status: http.StatusOK},
		{method: http.MethodGet, path: "/users/123e4567-e89b-42d3-a456-426614174000", status: http.StatusOK},
		{method: http.MethodDelete, path: "/users/me", status: http.StatusNoContent},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
		if tt.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		router.ServeHTTP(rec, req)
		if rec.Code != tt.status {
			t.Fatalf("%s %s expected status %d, got %d body=%s", tt.method, tt.path, tt.status, rec.Code, rec.Body.String())
		}
	}
}

func TestRegisterDatabaseLifecycle(t *testing.T) {
	lifecycle := fxtest.NewLifecycle(t)
	db := &databaseStub{}

	RegisterDatabaseLifecycle(lifecycle, db)

	if err := lifecycle.Start(context.Background()); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}
	if err := lifecycle.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
	if !db.closed {
		t.Fatal("expected database to be closed")
	}
}

func TestNewHTTPServer(t *testing.T) {
	router := chi.NewRouter()
	server := NewHTTPServer(router, HTTPConfig{Port: "8080"})

	if server == nil {
		t.Fatal("expected non-nil server")
	}

	if server.Addr != ":8080" {
		t.Fatalf("expected :8080, got %q", server.Addr)
	}

	if server.Handler != router {
		t.Fatal("expected server to keep provided router")
	}

	if server.ReadHeaderTimeout <= 0 {
		t.Fatal("expected positive read header timeout")
	}
}

func TestRegisterHTTPServer(t *testing.T) {
	lifecycle := fxtest.NewLifecycle(t)
	server := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.NewServeMux(),
	}

	RegisterHTTPServer(lifecycle, server)

	if err := lifecycle.Start(context.Background()); err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("network listen is not permitted in this environment: %v", err)
		}
		t.Fatalf("unexpected start error: %v", err)
	}

	if err := lifecycle.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}

func TestRegisterHTTPServerReturnsListenError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("network listen is not permitted in this environment: %v", err)
		}
		t.Fatalf("listen test port: %v", err)
	}
	defer listener.Close()

	lifecycle := fxtest.NewLifecycle(t)
	server := &http.Server{
		Addr:    listener.Addr().String(),
		Handler: http.NewServeMux(),
	}

	RegisterHTTPServer(lifecycle, server)

	if err := lifecycle.Start(context.Background()); err == nil {
		t.Fatal("expected listen error")
	}
}
