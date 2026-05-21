package di

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authservice "tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/internal/transport/http/handler"
	"tic-tac-toe/internal/transport/http/middleware"
)

func TestRouterSystemEndpoints(t *testing.T) {
	router := NewRouter(
		handler.NewGameHandler(gameLogicStub{}, gameStorageStub{}),
		handler.NewAuthHandler(authStub{}),
		handler.NewUserHandler(userServiceStub{}),
		middleware.NewUserAuthenticator(authStub{}),
		&databaseStub{},
	)

	tests := []struct {
		name        string
		method      string
		path        string
		status      int
		message     string
		wantJSONKey string
		contains    string
	}{
		{name: "health", method: http.MethodGet, path: "/healthz", status: http.StatusOK, wantJSONKey: "status"},
		{name: "ready", method: http.MethodGet, path: "/readyz", status: http.StatusOK, wantJSONKey: "status"},
		{name: "swagger ui", method: http.MethodGet, path: "/swagger", status: http.StatusOK, contains: "SwaggerUIBundle"},
		{name: "openapi yaml", method: http.MethodGet, path: "/openapi.yaml", status: http.StatusOK, contains: "swagger: \"2.0\""},
		{name: "openapi json", method: http.MethodGet, path: "/swagger/doc.json", status: http.StatusOK, contains: `"swagger": "2.0"`},
		{name: "not found", method: http.MethodGet, path: "/missing", status: http.StatusNotFound, message: "not found"},
		{name: "method not allowed", method: http.MethodGet, path: "/auth", status: http.StatusMethodNotAllowed, message: "method not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)

			router.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("expected status %d, got %d", tt.status, rec.Code)
			}
			assertSecurityHeaders(t, rec)
			if tt.message != "" {
				assertResponseMessage(t, rec, tt.message)
			}
			if tt.wantJSONKey != "" {
				assertResponseHasKey(t, rec, tt.wantJSONKey)
			}
			if tt.contains != "" && !strings.Contains(rec.Body.String(), tt.contains) {
				t.Fatalf("expected response to contain %q, got %q", tt.contains, rec.Body.String())
			}
		})
	}
}

func TestRouterProtectedRouteRequiresAuth(t *testing.T) {
	router := NewRouter(
		handler.NewGameHandler(gameLogicStub{}, gameStorageStub{}),
		handler.NewAuthHandler(authStub{}),
		handler.NewUserHandler(userServiceStub{}),
		middleware.NewUserAuthenticator(deniedAuthStub{}),
		&databaseStub{},
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/games", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	assertResponseMessage(t, rec, "unauthorized")
}

func TestRouterRejectsInvalidPathUUID(t *testing.T) {
	router := NewRouter(
		handler.NewGameHandler(gameLogicStub{}, gameStorageStub{}),
		handler.NewAuthHandler(authStub{}),
		handler.NewUserHandler(userServiceStub{}),
		middleware.NewUserAuthenticator(authStub{}),
		&databaseStub{},
	)

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/users/not-a-uuid"},
		{method: http.MethodGet, path: "/games/not-a-uuid"},
		{method: http.MethodPost, path: "/games/not-a-uuid/join"},
		{method: http.MethodPost, path: "/games/not-a-uuid/move", body: `{"field":[[1,0,0],[0,0,0],[0,0,0]]}`},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
			assertResponseMessage(t, rec, "invalid uuid")
		})
	}
}

func TestReadyzReportsDatabaseFailure(t *testing.T) {
	router := NewRouter(
		handler.NewGameHandler(gameLogicStub{}, gameStorageStub{}),
		handler.NewAuthHandler(authStub{}),
		handler.NewUserHandler(userServiceStub{}),
		middleware.NewUserAuthenticator(authStub{}),
		&databaseStub{pingErr: http.ErrServerClosed},
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	assertResponseMessage(t, rec, "database not ready")
}

func assertResponseMessage(t *testing.T, rec *httptest.ResponseRecorder, message string) {
	t.Helper()

	var payload struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Message != message {
		t.Fatalf("expected message %q, got %q", message, payload.Message)
	}
}

func assertResponseHasKey(t *testing.T, rec *httptest.ResponseRecorder, key string) {
	t.Helper()

	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := payload[key]; !ok {
		t.Fatalf("expected response key %q, got %#v", key, payload)
	}
}

func assertSecurityHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	headers := rec.Header()
	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options header, got %q", headers.Get("X-Content-Type-Options"))
	}
	if headers.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("expected X-Frame-Options header, got %q", headers.Get("X-Frame-Options"))
	}
	if headers.Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("expected Referrer-Policy header, got %q", headers.Get("Referrer-Policy"))
	}
}

type deniedAuthStub struct{}

func (deniedAuthStub) SignUp(context.Context, authservice.SignUpRequest) (bool, error) {
	return true, nil
}

func (deniedAuthStub) Authenticate(context.Context, authservice.JwtRequest) (authservice.JwtResponse, error) {
	return authservice.JwtResponse{}, nil
}

func (deniedAuthStub) RefreshAccessToken(context.Context, string) (authservice.JwtResponse, error) {
	return authservice.JwtResponse{}, nil
}

func (deniedAuthStub) RefreshRefreshToken(context.Context, string) (authservice.JwtResponse, error) {
	return authservice.JwtResponse{}, nil
}

func (deniedAuthStub) Logout(context.Context, string) error {
	return authservice.ErrInvalidToken
}

func (deniedAuthStub) LogoutAll(context.Context, string) error {
	return authservice.ErrInvalidToken
}

func (deniedAuthStub) AuthenticateToken(context.Context, string) (string, error) {
	return "", authservice.ErrInvalidToken
}
