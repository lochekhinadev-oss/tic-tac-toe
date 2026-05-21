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

// RefreshAccessToken refreshes an access token using a refresh token.
// @Summary Refresh access token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshJwtRequest true "Request body"
// @Success 200 {object} dto.JwtResponse "JWT tokens"
// @Failure 400 {object} dto.ErrorResponse "Invalid JSON"
// @Failure 401 {object} dto.ErrorResponse "Invalid refresh token"
// @Failure 415 {object} dto.ErrorResponse "Request body must be application/json"
// @Failure 500 {object} dto.ErrorResponse "Refresh failed"
// @Router /auth/tokens/access [post]
func (h *AuthHandler) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	h.refreshToken(w, r, h.auth.RefreshAccessToken, messages.FailedRefreshAccess)
}

// RefreshRefreshToken refreshes a refresh token using a valid refresh token.
// @Summary Refresh refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshJwtRequest true "Request body"
// @Success 200 {object} dto.JwtResponse "JWT tokens"
// @Failure 400 {object} dto.ErrorResponse "Invalid JSON"
// @Failure 401 {object} dto.ErrorResponse "Invalid refresh token"
// @Failure 415 {object} dto.ErrorResponse "Request body must be application/json"
// @Failure 500 {object} dto.ErrorResponse "Refresh failed"
// @Router /auth/tokens/refresh [post]
func (h *AuthHandler) RefreshRefreshToken(w http.ResponseWriter, r *http.Request) {
	h.refreshToken(w, r, h.auth.RefreshRefreshToken, messages.FailedRefreshRefresh)
}

// Logout revokes the active refresh session.
// @Summary Logout
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshJwtRequest true "Request body"
// @Success 204 "Logged out"
// @Failure 400 {object} dto.ErrorResponse "Invalid JSON"
// @Failure 401 {object} dto.ErrorResponse "Invalid refresh token"
// @Failure 415 {object} dto.ErrorResponse "Request body must be application/json"
// @Failure 500 {object} dto.ErrorResponse "Logout failed"
// @Router /auth/sessions/current [delete]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	logHandler("%s %s logout request", r.Method, r.URL.Path)

	request, ok := decodeRefreshTokenRequest(w, r, "logout")
	if !ok {
		return
	}

	if handleRefreshActionError(w, r, "logout", h.auth.Logout(r.Context(), request.RefreshToken), messages.FailedLogoutUser) {
		return
	}

	logHandler("%s %s logout completed", r.Method, r.URL.Path)
	w.WriteHeader(http.StatusNoContent)
}

// LogoutAll revokes all refresh sessions for the authenticated user.
// @Summary Logout from all sessions
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshJwtRequest true "Request body"
// @Success 204 "Logged out from all sessions"
// @Failure 400 {object} dto.ErrorResponse "Invalid JSON"
// @Failure 401 {object} dto.ErrorResponse "Invalid refresh token"
// @Failure 415 {object} dto.ErrorResponse "Request body must be application/json"
// @Failure 500 {object} dto.ErrorResponse "Logout failed"
// @Router /auth/sessions [delete]
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	logHandler("%s %s logout all request", r.Method, r.URL.Path)

	request, ok := decodeRefreshTokenRequest(w, r, "logout all")
	if !ok {
		return
	}

	if handleRefreshActionError(w, r, "logout all", h.auth.LogoutAll(r.Context(), request.RefreshToken), messages.FailedLogoutUser) {
		return
	}

	logHandler("%s %s logout all completed", r.Method, r.URL.Path)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) refreshToken(w http.ResponseWriter, r *http.Request, refresh func(context.Context, string) (auth.JwtResponse, error), internalMessage string) {
	logHandler("%s %s jwt refresh request", r.Method, r.URL.Path)

	request, ok := decodeRefreshTokenRequest(w, r, "refresh")
	if !ok {
		return
	}

	response, err := refresh(r.Context(), request.RefreshToken)
	if handleRefreshActionError(w, r, "refresh", err, internalMessage) {
		return
	}

	webresponse.WriteJSON(w, http.StatusOK, jwtResponseToDTO(response))
}

func decodeLoginRequest(w http.ResponseWriter, r *http.Request) (dto.JwtRequest, bool) {
	request, err := decodeJSONBody[dto.JwtRequest](r)
	if err != nil {
		if writeDecodeError(w, err) {
			return dto.JwtRequest{}, false
		}
		if writeAuthError(w, r, "invalid auth body", messages.InvalidRequestBody, err, webresponse.WriteBadRequest) {
			return dto.JwtRequest{}, false
		}
		return dto.JwtRequest{}, false
	}
	return request, true
}

func decodeRefreshTokenRequest(w http.ResponseWriter, r *http.Request, action string) (dto.RefreshJwtRequest, bool) {
	request, err := decodeJSONBody[dto.RefreshJwtRequest](r)
	if err != nil {
		if writeDecodeError(w, err) {
			return dto.RefreshJwtRequest{}, false
		}
		if writeAuthError(w, r, "invalid "+action+" body", messages.InvalidRequestBody, err, webresponse.WriteBadRequest) {
			return dto.RefreshJwtRequest{}, false
		}
		return dto.RefreshJwtRequest{}, false
	}
	return request, true
}

func handleAuthActionError(w http.ResponseWriter, r *http.Request, action string, err error, internalMessage string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, auth.ErrRateLimited) {
		logHandler("%s %s %s rate limited: %v", r.Method, r.URL.Path, action, err)
		webresponse.WriteTooManyRequests(w, messages.TooManyRequests)
		return true
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

func handleRefreshActionError(w http.ResponseWriter, r *http.Request, action string, err error, internalMessage string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, auth.ErrRateLimited) {
		logHandler("%s %s %s rate limited: %v", r.Method, r.URL.Path, action, err)
		webresponse.WriteTooManyRequests(w, messages.TooManyRequests)
		return true
	}
	if errors.Is(err, auth.ErrInvalidToken) {
		logHandler("%s %s invalid %s token: %v", r.Method, r.URL.Path, action, err)
		webresponse.WriteUnauthorized(w)
		return true
	}
	if writeAuthError(w, r, action+" failed", internalMessage, err, webresponse.WriteInternalError) {
		return true
	}
	return false
}
