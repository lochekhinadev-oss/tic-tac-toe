package handler

import (
	"net/http"

	"tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/internal/transport/http/messages"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

// Authenticate validates user credentials and returns JWT tokens.
// @Summary Authenticate by login and password
// @Description Send login and password as JSON and receive Bearer access and refresh tokens.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.JwtRequest true "Request body"
// @Success 200 {object} dto.JwtResponse "JWT tokens"
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

	response, err := h.auth.Authenticate(r.Context(), auth.JwtRequest{
		Login:    request.Login,
		Password: request.Password,
	})
	if handleAuthActionError(w, r, "auth", err, messages.FailedAuthenticateUser) {
		return
	}

	logHandler("%s %s auth completed", r.Method, r.URL.Path)
	webresponse.WriteJSON(w, http.StatusOK, jwtResponseToDTO(response))
}
