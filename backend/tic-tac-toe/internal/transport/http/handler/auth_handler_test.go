package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authservice "tic-tac-toe/infrastructure/auth"
)

type authHandlerServiceStub struct {
	signUpOK        bool
	signUpErr       error
	authResponse    authservice.JwtResponse
	authErr         error
	refreshResponse authservice.JwtResponse
	refreshErr      error
	logoutErr       error
	logoutAllErr    error
}

func (s authHandlerServiceStub) SignUp(context.Context, authservice.SignUpRequest) (bool, error) {
	return s.signUpOK, s.signUpErr
}

func (s authHandlerServiceStub) Authenticate(context.Context, authservice.JwtRequest) (authservice.JwtResponse, error) {
	return s.authResponse, s.authErr
}

func (s authHandlerServiceStub) RefreshAccessToken(context.Context, string) (authservice.JwtResponse, error) {
	return s.refreshResponse, s.refreshErr
}

func (s authHandlerServiceStub) RefreshRefreshToken(context.Context, string) (authservice.JwtResponse, error) {
	return s.refreshResponse, s.refreshErr
}

func (s authHandlerServiceStub) Logout(context.Context, string) error {
	return s.logoutErr
}

func (s authHandlerServiceStub) LogoutAll(context.Context, string) error {
	return s.logoutAllErr
}

func (s authHandlerServiceStub) AuthenticateToken(context.Context, string) (string, error) {
	return testUUID, nil
}

func TestAuthHandlerSignUp(t *testing.T) {
	tests := []struct {
		name    string
		auth    authHandlerServiceStub
		body    string
		status  int
		message string
	}{
		{name: "created", auth: authHandlerServiceStub{signUpOK: true}, body: `{"login":"player","password":"secret"}`, status: http.StatusCreated},
		{name: "invalid body", body: `{`, status: http.StatusBadRequest, message: "invalid request body"},
		{name: "invalid signup", auth: authHandlerServiceStub{signUpErr: authservice.ErrInvalidSignUp}, body: `{"login":"","password":""}`, status: http.StatusBadRequest, message: authservice.ErrInvalidSignUp.Error()},
		{name: "duplicate", auth: authHandlerServiceStub{}, body: `{"login":"player","password":"secret"}`, status: http.StatusConflict},
		{name: "internal", auth: authHandlerServiceStub{signUpErr: errors.New("db failed")}, body: `{"login":"player","password":"secret"}`, status: http.StatusInternalServerError, message: "failed to register user"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(tt.auth)
			rec, req := newAuthRequest(http.MethodPost, "/signup", tt.body)
			handler.SignUp(rec, req)
			assertResponseStatus(t, rec, tt.status)
			assertResponseMessageIfSet(t, rec, tt.status, tt.message)
		})
	}
}

func TestAuthHandlerAuthenticate(t *testing.T) {
	tests := []struct {
		name    string
		auth    authHandlerServiceStub
		body    string
		status  int
		message string
	}{
		{name: "ok", auth: authHandlerServiceStub{authResponse: authservice.JwtResponse{Type: "Bearer", AccessToken: "access", RefreshToken: "refresh"}}, body: `{"login":"player","password":"secret"}`, status: http.StatusOK},
		{name: "invalid body", body: `{`, status: http.StatusBadRequest, message: "invalid request body"},
		{name: "unauthorized", auth: authHandlerServiceStub{authErr: authservice.ErrInvalidCredentials}, status: http.StatusUnauthorized, message: "unauthorized"},
		{name: "internal", auth: authHandlerServiceStub{authErr: errors.New("db failed")}, status: http.StatusInternalServerError, message: "failed to authenticate user"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(tt.auth)
			if tt.body == "" {
				tt.body = `{"login":"player","password":"secret"}`
			}
			rec, req := newAuthRequest(http.MethodPost, "/auth", tt.body)
			handler.Authenticate(rec, req)
			assertResponseStatus(t, rec, tt.status)
			if tt.message != "" {
				assertStatusAndMessage(t, rec, tt.status, tt.message)
				return
			}
			if tt.status == http.StatusOK {
				assertJwtResponse(t, rec, "Bearer", "access", "refresh")
			}
		})
	}
}

func TestAuthHandlerRefreshAccessToken(t *testing.T) {
	handler := NewAuthHandler(authHandlerServiceStub{
		refreshResponse: authservice.JwtResponse{Type: "Bearer", AccessToken: "access-2", RefreshToken: "refresh"},
	})
	rec, req := newAuthRequest(http.MethodPost, "/auth/refresh/access", `{"refreshToken":"refresh"}`)

	handler.RefreshAccessToken(rec, req)

	assertResponseStatus(t, rec, http.StatusOK)
	assertJwtResponse(t, rec, "Bearer", "access-2", "refresh")
}

func TestAuthHandlerRefreshRefreshToken(t *testing.T) {
	handler := NewAuthHandler(authHandlerServiceStub{
		refreshResponse: authservice.JwtResponse{Type: "Bearer", AccessToken: "access-3", RefreshToken: "refresh-2"},
	})
	rec, req := newAuthRequest(http.MethodPost, "/auth/refresh", `{"refreshToken":"refresh"}`)

	handler.RefreshRefreshToken(rec, req)

	assertResponseStatus(t, rec, http.StatusOK)
	assertJwtResponse(t, rec, "Bearer", "access-3", "refresh-2")
}

func TestAuthHandlerRefreshRefreshTokenUnauthorized(t *testing.T) {
	handler := NewAuthHandler(authHandlerServiceStub{refreshErr: authservice.ErrInvalidToken})
	rec, req := newAuthRequest(http.MethodPost, "/auth/refresh", `{"refreshToken":"refresh"}`)

	handler.RefreshRefreshToken(rec, req)

	assertStatusAndMessage(t, rec, http.StatusUnauthorized, "unauthorized")
}

func TestAuthHandlerLogout(t *testing.T) {
	handler := NewAuthHandler(authHandlerServiceStub{})
	rec, req := newAuthRequest(http.MethodPost, "/auth/logout", `{"refreshToken":"refresh"}`)

	handler.Logout(rec, req)

	assertResponseStatus(t, rec, http.StatusNoContent)
	assertEmptyBody(t, rec)
}

func TestAuthHandlerLogoutAll(t *testing.T) {
	handler := NewAuthHandler(authHandlerServiceStub{})
	rec, req := newAuthRequest(http.MethodPost, "/auth/logout/all", `{"refreshToken":"refresh"}`)

	handler.LogoutAll(rec, req)

	assertResponseStatus(t, rec, http.StatusNoContent)
	assertEmptyBody(t, rec)
}

func TestAuthHandlerRejectsUnsupportedContentType(t *testing.T) {
	handler := NewAuthHandler(authHandlerServiceStub{signUpOK: true})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"login":"player","password":"secret"}`))
	req.Header.Set("Content-Type", "text/plain")

	handler.SignUp(rec, req)

	assertStatusAndMessage(t, rec, http.StatusUnsupportedMediaType, "content type must be application/json")
}

func newAuthRequest(method string, path string, body string) (*httptest.ResponseRecorder, *http.Request) {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return httptest.NewRecorder(), req
}

func assertResponseStatus(t *testing.T, rec *httptest.ResponseRecorder, status int) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("expected status %d, got %d", status, rec.Code)
	}
}

func assertResponseMessageIfSet(t *testing.T, rec *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	if message != "" {
		assertStatusAndMessage(t, rec, status, message)
	}
}

func assertEmptyBody(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rec.Body.String())
	}
}

func assertJwtResponse(t *testing.T, rec *httptest.ResponseRecorder, wantType, wantAccess, wantRefresh string) {
	t.Helper()

	var payload struct {
		Type         string `json:"type"`
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Type != wantType || payload.AccessToken != wantAccess || payload.RefreshToken != wantRefresh {
		t.Fatalf("unexpected jwt payload: %#v", payload)
	}
}
