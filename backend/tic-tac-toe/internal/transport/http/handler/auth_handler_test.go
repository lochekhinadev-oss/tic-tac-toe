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
	signUpOK     bool
	signUpErr    error
	signInOK     authservice.SessionResponse
	signInErr    error
	refreshOK    authservice.SessionResponse
	refreshErr   error
	logoutErr    error
	logoutAllErr error
}

func (s authHandlerServiceStub) SignUp(context.Context, authservice.SignUpRequest) (bool, error) {
	return s.signUpOK, s.signUpErr
}

func (s authHandlerServiceStub) SignIn(context.Context, authservice.SessionRequest) (authservice.SessionResponse, error) {
	return s.signInOK, s.signInErr
}

func (s authHandlerServiceStub) RefreshSession(context.Context, string) (authservice.SessionResponse, error) {
	return s.refreshOK, s.refreshErr
}

func (s authHandlerServiceStub) Logout(context.Context, string) error {
	return s.logoutErr
}

func (s authHandlerServiceStub) LogoutAll(context.Context, string) error {
	return s.logoutAllErr
}

func (s authHandlerServiceStub) AuthenticateSession(context.Context, string) (string, error) {
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
		{name: "ok", auth: authHandlerServiceStub{signInOK: authservice.SessionResponse{UserUUID: testUUID, SessionID: "session-1"}}, body: `{"login":"player","password":"secret"}`, status: http.StatusOK},
		{name: "invalid body", body: `{`, status: http.StatusBadRequest, message: "invalid request body"},
		{name: "unauthorized", auth: authHandlerServiceStub{signInErr: authservice.ErrInvalidCredentials}, status: http.StatusUnauthorized, message: "unauthorized"},
		{name: "internal", auth: authHandlerServiceStub{signInErr: errors.New("db failed")}, status: http.StatusInternalServerError, message: "failed to authenticate user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAuthHandler(tt.auth)
			if tt.body == "" {
				tt.body = `{"login":"player","password":"secret"}`
			}
			rec, req := newAuthRequest(http.MethodPost, "/auth/sessions", tt.body)
			handler.Authenticate(rec, req)
			assertResponseStatus(t, rec, tt.status)
			if tt.message != "" {
				assertAuthStatusAndMessage(t, rec, tt.status, tt.message)
				return
			}
			assertAuthResponse(t, rec, testUUID)
			assertSessionCookie(t, rec)
		})
	}
}

func TestAuthHandlerRefreshSession(t *testing.T) {
	handler := NewAuthHandler(authHandlerServiceStub{
		refreshOK: authservice.SessionResponse{UserUUID: testUUID, SessionID: "session-2"},
	})
	rec, req := newAuthRequest(http.MethodPost, "/auth/refresh/access", "")
	req.AddCookie(&http.Cookie{Name: authservice.SessionCookieName, Value: "session-1"})

	handler.RefreshAccessToken(rec, req)

	assertResponseStatus(t, rec, http.StatusOK)
	assertAuthResponse(t, rec, testUUID)
	assertSessionCookie(t, rec)
}

func TestAuthHandlerRefreshSessionUnauthorized(t *testing.T) {
	handler := NewAuthHandler(authHandlerServiceStub{refreshErr: authservice.ErrInvalidToken})
	rec, req := newAuthRequest(http.MethodPost, "/auth/refresh", "")
	req.AddCookie(&http.Cookie{Name: authservice.SessionCookieName, Value: "session-1"})

	handler.RefreshRefreshToken(rec, req)

	assertAuthStatusAndMessage(t, rec, http.StatusUnauthorized, "unauthorized")
}

func TestAuthHandlerLogout(t *testing.T) {
	handler := NewAuthHandler(authHandlerServiceStub{})
	rec, req := newAuthRequest(http.MethodDelete, "/auth/sessions/current", "")
	req.AddCookie(&http.Cookie{Name: authservice.SessionCookieName, Value: "session-1"})

	handler.Logout(rec, req)

	assertResponseStatus(t, rec, http.StatusNoContent)
	assertEmptyBody(t, rec)
}

func TestAuthHandlerLogoutAll(t *testing.T) {
	handler := NewAuthHandler(authHandlerServiceStub{})
	rec, req := newAuthRequest(http.MethodDelete, "/auth/sessions", "")
	req.AddCookie(&http.Cookie{Name: authservice.SessionCookieName, Value: "session-1"})

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

	assertAuthStatusAndMessage(t, rec, http.StatusUnsupportedMediaType, "content type must be application/json")
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
		assertAuthStatusAndMessage(t, rec, status, message)
	}
}

func assertEmptyBody(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", rec.Body.String())
	}
}

func assertAuthStatusAndMessage(t *testing.T, rec *httptest.ResponseRecorder, status int, message string) {
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

func assertAuthResponse(t *testing.T, rec *httptest.ResponseRecorder, wantUUID string) {
	t.Helper()
	var payload struct {
		UUID string `json:"uuid"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode auth response: %v", err)
	}
	if payload.UUID != wantUUID {
		t.Fatalf("unexpected auth response: %+v", payload)
	}
}

func assertSessionCookie(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if len(rec.Result().Cookies()) == 0 {
		t.Fatal("expected session cookie")
	}
}
