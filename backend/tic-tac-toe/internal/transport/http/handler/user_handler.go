package handler

import (
	"errors"
	"net/http"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/internal/transport/http/dto"
	"tic-tac-toe/internal/transport/http/messages"
	"tic-tac-toe/internal/transport/http/middleware"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

type UserHandler struct {
	users UserQueryService
}

func NewUserHandler(users UserQueryService) *UserHandler {
	return &UserHandler{users: users}
}

// GetUser returns user information by UUID.
// @Summary Get user by UUID
// @Description Returns public user information. Request body is not used.
// @Tags users
// @Produce json
// @Security SessionCookieAuth
// @Param uuid path string true "User UUID" Format(uuid) default(123e4567-e89b-42d3-a456-426614174000)
// @Success 200 {object} dto.UserResponse "User"
// @Failure 400 {object} dto.ErrorResponse "Invalid UUID"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid session cookie"
// @Failure 404 {object} dto.ErrorResponse "User not found"
// @Failure 500 {object} dto.ErrorResponse "User was not loaded"
// @Router /users/{uuid} [get]
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request, uuid googleuuid.UUID) {
	logHandler("%s %s get user request uuid=%s", r.Method, r.URL.Path, uuid)
	if uuid == googleuuid.Nil {
		webresponse.WriteBadRequest(w, messages.InvalidUUID)
		return
	}

	user, err := h.users.GetUserByUUID(r.Context(), uuid)
	if errors.Is(err, domain.ErrUserNotFound) {
		logHandler("%s %s user not found uuid=%s", r.Method, r.URL.Path, uuid)
		webresponse.WriteNotFound(w, messages.UserNotFound)
		return
	}
	if h.writeUserError(w, r, "failed to load user uuid="+uuid.String(), messages.FailedLoadUser, err) {
		return
	}

	logHandler("%s %s loaded user uuid=%s login=%q", r.Method, r.URL.Path, user.UUID, user.Login)
	webresponse.WriteJSON(w, http.StatusOK, dto.UserResponse{UUID: uuidFromString(user.UUID), Login: user.Login})
}

// GetCurrentUser returns user information by session cookie.
// @Summary Get current user
// @Description Returns public user information for the authenticated session cookie.
// @Tags users
// @Produce json
// @Security SessionCookieAuth
// @Success 200 {object} dto.UserResponse "User"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid session cookie"
// @Failure 404 {object} dto.ErrorResponse "User not found"
// @Failure 500 {object} dto.ErrorResponse "User was not loaded"
// @Router /users/me [get]
func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	uuid := middleware.UserUUIDFromContext(r.Context())
	if uuid == "" {
		webresponse.WriteUnauthorized(w)
		return
	}
	parsedUUID, err := parseUUID(uuid)
	if err != nil {
		webresponse.WriteUnauthorized(w)
		return
	}
	h.GetUser(w, r, parsedUUID)
}

// DeleteCurrentUser soft-deletes the authenticated user.
// @Summary Delete current user
// @Description Soft-deletes the authenticated user account by marking it deleted and making the login unavailable for future auth.
// @Tags users
// @Produce json
// @Security SessionCookieAuth
// @Success 204 "User deleted"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid session cookie"
// @Failure 404 {object} dto.ErrorResponse "User not found"
// @Failure 500 {object} dto.ErrorResponse "User was not deleted"
// @Router /users/me [delete]
func (h *UserHandler) DeleteCurrentUser(w http.ResponseWriter, r *http.Request) {
	uuid := middleware.UserUUIDFromContext(r.Context())
	logHandler("%s %s delete user request uuid=%s", r.Method, r.URL.Path, uuid)

	if uuid == "" {
		logHandler("%s %s unauthorized delete user request", r.Method, r.URL.Path)
		webresponse.WriteUnauthorized(w)
		return
	}

	parsedUUID, err := parseUUID(uuid)
	if err != nil {
		webresponse.WriteUnauthorized(w)
		return
	}

	if err := h.users.DeleteUser(r.Context(), parsedUUID); errors.Is(err, domain.ErrUserNotFound) {
		logHandler("%s %s user not found uuid=%s", r.Method, r.URL.Path, uuid)
		webresponse.WriteNotFound(w, messages.UserNotFound)
		return
	} else if h.writeUserError(w, r, "failed to delete user uuid="+uuid, messages.FailedDeleteUser, err) {
		return
	}

	clearSessionCookie(w, r)
	logHandler("%s %s deleted user uuid=%s", r.Method, r.URL.Path, uuid)
	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) writeUserError(w http.ResponseWriter, r *http.Request, logMessage, responseMessage string, err error) bool {
	if err == nil {
		return false
	}

	logHandler("%s %s %s: %v", r.Method, r.URL.Path, logMessage, err)
	webresponse.WriteInternalError(w, responseMessage)
	return true
}
