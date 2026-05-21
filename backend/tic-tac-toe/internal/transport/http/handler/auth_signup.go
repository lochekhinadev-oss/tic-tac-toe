package handler

import (
	"errors"
	"net/http"

	"tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/internal/transport/http/dto"
	"tic-tac-toe/internal/transport/http/messages"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

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
