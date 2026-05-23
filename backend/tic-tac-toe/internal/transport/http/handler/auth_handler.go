package handler

import (
	"context"
	"errors"
	"net/http"

	"tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/internal/transport/http/dto"
	"tic-tac-toe/internal/transport/http/messages"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

type AuthHandler struct {
	auth AuthService
}

func NewAuthHandler(authService auth.AuthService) *AuthHandler {
	return &AuthHandler{auth: authService}
}

// Authenticate validates user credentials and establishes an HttpOnly session cookie.
// @Summary Authenticate by login and password
// @Description Send login and password as JSON and receive a session cookie.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.AuthRequest true "Request body"
// @Success 200 {object} dto.AuthResponse "Authenticated user"
// @Failure 400 {object} dto.ErrorResponse "Invalid JSON"
// @Failure 401 {object} dto.ErrorResponse "Invalid or missing credentials"
// @Failure 415 {object} dto.ErrorResponse "Request body must be application/json"
// @Failure 500 {object} dto.ErrorResponse "Authentication failed"
// @Router /auth/sessions [post]
func (h *AuthHandler) Authenticate(w http.ResponseWriter, r *http.Request) {
	logHandler("%s %s auth request", r.Method, r.URL.Path)

	request, ok := decodeLoginRequest(w, r)
	if !ok {
		return
	}

	session, err := h.auth.SignIn(r.Context(), auth.SessionRequest{
		Login:    request.Login,
		Password: request.Password,
	})
	if handleAuthActionError(w, r, "auth", err, messages.FailedAuthenticateUser) {
		return
	}

	logHandler("%s %s auth completed", r.Method, r.URL.Path)
	setSessionCookie(w, r, session.SessionID, session.ExpiresAt)
	webresponse.WriteJSON(w, http.StatusOK, dto.AuthResponse{UUID: uuidFromString(session.UserUUID)})
}

// SignUp registers a new user.
// @Summary Register a new user
// @Description Creates a user by login and password. Login is trimmed, password is stored as a hash.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.SignUpRequest true "Request body"
// @Success 201 {object} dto.SignUpResponse "User registered"
// @Failure 400 {object} dto.ErrorResponse "Invalid JSON or empty login/password"
// @Failure 415 {object} dto.ErrorResponse "Request body must be application/json"
// @Failure 409 {object} dto.SignUpResponse "Login already exists"
// @Failure 500 {object} dto.ErrorResponse "Registration failed"
// @Router /users [post]
func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	logHandler("%s %s signup request", r.Method, r.URL.Path)

	request, err := decodeJSONBody[dto.SignUpRequest](r)
	if err != nil {
		if writeDecodeError(w, err) {
			return
		}
		if writeAuthError(w, r, "invalid body", messages.InvalidRequestBody, err, webresponse.WriteBadRequest) {
			return
		}
		return
	}

	success, err := h.auth.SignUp(r.Context(), auth.SignUpRequest{
		Login:    request.Login,
		Password: request.Password,
	})
	if errors.Is(err, auth.ErrInvalidSignUp) {
		if writeAuthError(w, r, "invalid signup data for login="+request.Login, "", err, webresponse.WriteBadRequest) {
			return
		}
		return
	}
	if writeAuthError(w, r, "signup failed for login="+request.Login, messages.FailedRegisterUser, err, webresponse.WriteInternalError) {
		return
	}
	if !success {
		logHandler("%s %s login already exists for login=%q", r.Method, r.URL.Path, request.Login)
		webresponse.WriteJSON(w, http.StatusConflict, dto.SignUpResponse{Success: false})
		return
	}

	logHandler("%s %s signup completed for login=%q", r.Method, r.URL.Path, request.Login)
	webresponse.WriteJSON(w, http.StatusCreated, dto.SignUpResponse{Success: true})
}

// RefreshAccessToken renews the current session cookie.
// @Summary Refresh session
// @Tags auth
// @Produce json
// @Success 200 {object} dto.AuthResponse "Authenticated user"
// @Failure 401 {object} dto.ErrorResponse "Invalid session"
// @Failure 500 {object} dto.ErrorResponse "Session renewal failed"
// @Router /auth/tokens/access [post]
func (h *AuthHandler) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	h.refreshSession(w, r, h.auth.RefreshSession, messages.FailedRefreshAccess)
}

// RefreshRefreshToken refreshes the current session cookie.
// @Summary Refresh session
// @Tags auth
// @Produce json
// @Success 200 {object} dto.AuthResponse "Authenticated user"
// @Failure 401 {object} dto.ErrorResponse "Invalid session"
// @Failure 500 {object} dto.ErrorResponse "Session renewal failed"
// @Router /auth/tokens/refresh [post]
func (h *AuthHandler) RefreshRefreshToken(w http.ResponseWriter, r *http.Request) {
	h.refreshSession(w, r, h.auth.RefreshSession, messages.FailedRefreshRefresh)
}

// Logout revokes the active session.
// @Summary Logout
// @Tags auth
// @Produce json
// @Success 204 "Logged out"
// @Failure 401 {object} dto.ErrorResponse "Invalid session"
// @Failure 500 {object} dto.ErrorResponse "Logout failed"
// @Router /auth/sessions/current [delete]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	logHandler("%s %s logout request", r.Method, r.URL.Path)

	sessionID, ok := decodeSessionID(w, r, "logout")
	if !ok {
		return
	}

	if handleSessionActionError(w, r, "logout", h.auth.Logout(r.Context(), sessionID), messages.FailedLogoutUser) {
		return
	}

	clearSessionCookie(w, r)
	logHandler("%s %s logout completed", r.Method, r.URL.Path)
	w.WriteHeader(http.StatusNoContent)
}

// LogoutAll revokes all sessions for the authenticated user.
// @Summary Logout from all sessions
// @Tags auth
// @Produce json
// @Success 204 "Logged out from all sessions"
// @Failure 401 {object} dto.ErrorResponse "Invalid session"
// @Failure 500 {object} dto.ErrorResponse "Logout failed"
// @Router /auth/sessions [delete]
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	logHandler("%s %s logout all request", r.Method, r.URL.Path)

	sessionID, ok := decodeSessionID(w, r, "logout all")
	if !ok {
		return
	}

	if handleSessionActionError(w, r, "logout all", h.auth.LogoutAll(r.Context(), sessionID), messages.FailedLogoutUser) {
		return
	}

	clearSessionCookie(w, r)
	logHandler("%s %s logout all completed", r.Method, r.URL.Path)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) refreshSession(w http.ResponseWriter, r *http.Request, refresh func(context.Context, string) (auth.SessionResponse, error), internalMessage string) {
	logHandler("%s %s session refresh request", r.Method, r.URL.Path)

	sessionID, ok := decodeSessionID(w, r, "refresh")
	if !ok {
		return
	}

	response, err := refresh(r.Context(), sessionID)
	if handleSessionActionError(w, r, "refresh", err, internalMessage) {
		return
	}

	setSessionCookie(w, r, response.SessionID, response.ExpiresAt)
	webresponse.WriteJSON(w, http.StatusOK, dto.AuthResponse{UUID: uuidFromString(response.UserUUID)})
}

func decodeLoginRequest(w http.ResponseWriter, r *http.Request) (dto.AuthRequest, bool) {
	request, err := decodeJSONBody[dto.AuthRequest](r)
	if err != nil {
		if writeDecodeError(w, err) {
			return dto.AuthRequest{}, false
		}
		if writeAuthError(w, r, "invalid auth body", messages.InvalidRequestBody, err, webresponse.WriteBadRequest) {
			return dto.AuthRequest{}, false
		}
		return dto.AuthRequest{}, false
	}
	return request, true
}

func decodeSessionID(w http.ResponseWriter, r *http.Request, action string) (string, bool) {
	if cookie, err := r.Cookie(auth.SessionCookieName); err == nil && cookie != nil && cookie.Value != "" {
		return cookie.Value, true
	}

	logHandler("%s %s missing %s session", r.Method, r.URL.Path, action)
	webresponse.WriteUnauthorized(w)
	return "", false
}

func handleAuthActionError(w http.ResponseWriter, r *http.Request, action string, err error, internalMessage string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, auth.ErrInvalidCredentials) || errors.Is(err, auth.ErrInvalidToken) {
		logHandler("%s %s unauthorized %s request: %v", r.Method, r.URL.Path, action, err)
		webresponse.WriteUnauthorized(w)
		return true
	}
	if writeAuthError(w, r, action+" failed", internalMessage, err, webresponse.WriteInternalError) {
		return true
	}
	return false
}

func handleSessionActionError(w http.ResponseWriter, r *http.Request, action string, err error, internalMessage string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, auth.ErrInvalidToken) {
		logHandler("%s %s invalid %s session: %v", r.Method, r.URL.Path, action, err)
		webresponse.WriteUnauthorized(w)
		return true
	}
	if writeAuthError(w, r, action+" failed", internalMessage, err, webresponse.WriteInternalError) {
		return true
	}
	return false
}

func writeAuthError(w http.ResponseWriter, r *http.Request, logMessage, responseMessage string, err error, write func(http.ResponseWriter, string)) bool {
	if err == nil {
		return false
	}

	if responseMessage == "" {
		responseMessage = err.Error()
	}
	logHandlerError("%s %s %s: %v", r.Method, r.URL.Path, logMessage, err)
	write(w, responseMessage)
	return true
}
