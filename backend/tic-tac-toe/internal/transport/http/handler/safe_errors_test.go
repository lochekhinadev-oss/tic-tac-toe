package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authservice "tic-tac-toe/infrastructure/auth"
)

const internalErrorText = "postgres password leaked in stack trace"

func TestHandlersDoNotExposeInternalErrors(t *testing.T) {
	tests := []struct {
		name        string
		handlerCall func(*httptest.ResponseRecorder)
		status      int
		message     string
	}{
		{
			name: "signup",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := NewAuthHandler(authHandlerServiceStub{signUpErr: errors.New(internalErrorText)})
				_, req := newAuthRequest(http.MethodPost, "/users", `{"login":"player","password":"secret"}`)
				handler.SignUp(rec, req)
			},
			status:  http.StatusInternalServerError,
			message: "failed to register user",
		},
		{
			name: "authenticate",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := NewAuthHandler(authHandlerServiceStub{authErr: errors.New(internalErrorText)})
				_, req := newAuthRequest(http.MethodPost, "/auth/sessions", `{"login":"player","password":"secret"}`)
				handler.Authenticate(rec, req)
			},
			status:  http.StatusInternalServerError,
			message: "failed to authenticate user",
		},
		{
			name: "refresh access",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := NewAuthHandler(authHandlerServiceStub{refreshErr: errors.New(internalErrorText)})
				_, req := newAuthRequest(http.MethodPost, "/auth/tokens/access", `{"refreshToken":"refresh"}`)
				handler.RefreshAccessToken(rec, req)
			},
			status:  http.StatusInternalServerError,
			message: "failed to refresh access token",
		},
		{
			name: "refresh refresh",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := NewAuthHandler(authHandlerServiceStub{refreshErr: errors.New(internalErrorText)})
				_, req := newAuthRequest(http.MethodPost, "/auth/tokens/refresh", `{"refreshToken":"refresh"}`)
				handler.RefreshRefreshToken(rec, req)
			},
			status:  http.StatusInternalServerError,
			message: "failed to refresh refresh token",
		},
		{
			name: "logout",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := NewAuthHandler(authHandlerServiceStub{logoutErr: errors.New(internalErrorText)})
				_, req := newAuthRequest(http.MethodDelete, "/auth/sessions/current", `{"refreshToken":"refresh"}`)
				handler.Logout(rec, req)
			},
			status:  http.StatusInternalServerError,
			message: "failed to logout user",
		},
		{
			name: "logout all",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := NewAuthHandler(authHandlerServiceStub{logoutAllErr: errors.New(internalErrorText)})
				_, req := newAuthRequest(http.MethodDelete, "/auth/sessions", `{"refreshToken":"refresh"}`)
				handler.LogoutAll(rec, req)
			},
			status:  http.StatusInternalServerError,
			message: "failed to logout user",
		},
		{
			name: "delete user",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := NewUserHandler(&userServiceStub{deleteErr: errors.New(internalErrorText)})
				req := authenticatedRequest(http.MethodDelete, "/users/me", nil)
				handler.DeleteCurrentUser(rec, req)
			},
			status:  http.StatusInternalServerError,
			message: "failed to delete user",
		},
		{
			name: "create game save",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := newGameHandler(&logicStub{}, &storageStub{saveErr: errors.New(internalErrorText)})
				req := authenticatedRequest(http.MethodPost, "/games", nil)
				handler.CreateGame(rec, req)
			},
			status:  http.StatusInternalServerError,
			message: "failed to save current game",
		},
		{
			name: "list games",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := newGameHandler(&logicStub{}, &storageStub{getErr: errors.New(internalErrorText)})
				req := authenticatedRequest(http.MethodGet, "/games", nil)
				handler.ListGames(rec, req)
			},
			status:  http.StatusInternalServerError,
			message: "failed to load current games",
		},
		{
			name: "list completed games",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := newGameHandler(&logicStub{}, &storageStub{getErr: errors.New(internalErrorText)})
				req := authenticatedRequest(http.MethodGet, "/games/history", nil)
				handler.ListCompletedGames(rec, req)
			},
			status:  http.StatusInternalServerError,
			message: "failed to load completed games",
		},
		{
			name: "leaderboard",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := newGameHandler(&logicStub{}, &storageStub{getErr: errors.New(internalErrorText)})
				req := authenticatedRequest(http.MethodGet, "/games/leaderboard?n=10", nil)
				handler.ListTopPlayers(rec, req)
			},
			status:  http.StatusInternalServerError,
			message: "failed to load leaderboard",
		},
		{
			name: "get game",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := newGameHandler(&logicStub{}, &storageStub{getErr: errors.New(internalErrorText)})
				req := authenticatedRequest(http.MethodGet, "/games/"+testUUID, nil)
				handler.GetGame(rec, req, testUUID)
			},
			status:  http.StatusInternalServerError,
			message: "failed to load current game",
		},
		{
			name: "join load",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := newGameHandler(&logicStub{}, &storageStub{getErr: errors.New(internalErrorText)})
				req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/join", nil)
				handler.JoinGame(rec, req, testUUID)
			},
			status:  http.StatusInternalServerError,
			message: "failed to load current game",
		},
		{
			name: "move load",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := newGameHandler(&logicStub{}, &storageStub{getErr: errors.New(internalErrorText)})
				req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{"field": emptyField()}))
				handler.MakeMove(rec, req, testUUID)
			},
			status:  http.StatusInternalServerError,
			message: "failed to load current game",
		},
		{
			name: "get user",
			handlerCall: func(rec *httptest.ResponseRecorder) {
				handler := NewUserHandler(&userServiceStub{err: errors.New(internalErrorText)})
				req := authenticatedRequest(http.MethodGet, "/users/"+testUUID, nil)
				handler.GetUser(rec, req, testUUID)
			},
			status:  http.StatusInternalServerError,
			message: "failed to load user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			tt.handlerCall(rec)

			assertStatusAndMessage(t, rec, tt.status, tt.message)
			assertBodyDoesNotContain(t, rec, internalErrorText)
		})
	}
}

func TestUnauthorizedAuthErrorsStayGeneric(t *testing.T) {
	handler := NewAuthHandler(authHandlerServiceStub{authErr: authservice.ErrInvalidCredentials})
	rec, req := newAuthRequest(http.MethodPost, "/auth/sessions", `{"login":"player","password":"wrong"}`)

	handler.Authenticate(rec, req)

	assertStatusAndMessage(t, rec, http.StatusUnauthorized, "unauthorized")
	assertBodyDoesNotContain(t, rec, "invalid credentials")
}

func assertBodyDoesNotContain(t *testing.T, rec *httptest.ResponseRecorder, value string) {
	t.Helper()

	if strings.Contains(rec.Body.String(), value) {
		t.Fatalf("response leaked %q: %s", value, rec.Body.String())
	}
}
