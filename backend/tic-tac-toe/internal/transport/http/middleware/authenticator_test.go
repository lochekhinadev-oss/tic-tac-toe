package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	googleuuid "github.com/google/uuid"

	authservice "tic-tac-toe/infrastructure/auth"
)

const testUUID = "123e4567-e89b-42d3-a456-426614174000"

type authServiceStub struct {
	uuid string
	err  error
}

func (s authServiceStub) AuthenticateSession(context.Context, string) (string, error) {
	return s.uuid, s.err
}

type allowAllAuthorizerStub struct{}

func (allowAllAuthorizerStub) AuthorizeRequest(context.Context, googleuuid.UUID, string, string) (bool, error) {
	return true, nil
}

type denyAuthorizerStub struct{}

func (denyAuthorizerStub) AuthorizeRequest(context.Context, googleuuid.UUID, string, string) (bool, error) {
	return false, nil
}

func TestUserAuthenticatorProtectAllowsAuthorizedRequest(t *testing.T) {
	authenticator := NewUserAuthenticator(authServiceStub{uuid: testUUID}, allowAllAuthorizerStub{})
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		if got := UserUUIDFromContext(r.Context()); got != testUUID {
			t.Fatalf("expected user uuid in context, got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/games", nil)
	req.AddCookie(&http.Cookie{Name: authservice.SessionCookieName, Value: "session-1"})
	authenticator.Protect(next).ServeHTTP(rec, req)

	if !nextCalled || rec.Code != http.StatusNoContent {
		t.Fatalf("expected next handler to run, called=%v status=%d", nextCalled, rec.Code)
	}
}

func TestUserAuthenticatorProtectRejectsUnauthorizedRequest(t *testing.T) {
	authenticator := NewUserAuthenticator(authServiceStub{err: authservice.ErrInvalidToken}, allowAllAuthorizerStub{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/games", nil)

	authenticator.Protect(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler must not run")
	})).ServeHTTP(rec, req)

	assertMiddlewareError(t, rec, http.StatusUnauthorized, "unauthorized")
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("expected WWW-Authenticate header")
	}
}

func TestUserAuthenticatorProtectHandlesAuthServiceError(t *testing.T) {
	authenticator := NewUserAuthenticator(authServiceStub{err: errors.New("db failed")}, allowAllAuthorizerStub{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/games", nil)
	req.AddCookie(&http.Cookie{Name: authservice.SessionCookieName, Value: "session-1"})

	authenticator.Protect(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler must not run")
	})).ServeHTTP(rec, req)

	assertMiddlewareError(t, rec, http.StatusInternalServerError, "failed to authenticate user")
}

func TestUserAuthenticatorProtectRejectsForbiddenRequest(t *testing.T) {
	authenticator := NewUserAuthenticator(authServiceStub{uuid: testUUID}, denyAuthorizerStub{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/games", nil)
	req.AddCookie(&http.Cookie{Name: authservice.SessionCookieName, Value: "session-1"})

	authenticator.Protect(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler must not run")
	})).ServeHTTP(rec, req)

	assertMiddlewareError(t, rec, http.StatusForbidden, "forbidden")
}

func assertMiddlewareError(t *testing.T, rec *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("expected status %d, got %d", status, rec.Code)
	}
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
