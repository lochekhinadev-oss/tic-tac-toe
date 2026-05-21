package handler

import (
	"errors"
	"net/http"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/internal/transport/http/messages"
	"tic-tac-toe/internal/transport/http/middleware"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

// JoinGame joins a waiting player-vs-player game.
// @Summary Join a game
// @Description Joins an existing player-vs-player game that is waiting for a second player. Request body is not used.
// @Tags games
// @Produce json
// @Security BearerAuth
// @Param uuid path string true "Game UUID" Format(uuid) default(123e4567-e89b-42d3-a456-426614174000)
// @Success 200 {object} dto.GameResponse "Joined game"
// @Failure 400 {object} dto.ErrorResponse "Game cannot be joined"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid Bearer token"
// @Failure 404 {object} dto.ErrorResponse "Game not found"
// @Failure 500 {object} dto.ErrorResponse "Join was not saved"
// @Router /games/{uuid}/join [post]
func (h *GameHandler) JoinGame(w http.ResponseWriter, r *http.Request, uuid string) {
	logHandler("%s %s join game request uuid=%s", r.Method, r.URL.Path, uuid)

	if err := validateUUID(uuid); err != nil {
		logHandler("%s %s invalid uuid=%s: %v", r.Method, r.URL.Path, uuid, err)
		webresponse.WriteBadRequest(w, messages.InvalidUUID)
		return
	}

	userUUID := middleware.UserUUIDFromContext(r.Context())
	if userUUID == "" {
		logHandler("%s %s unauthorized join game request uuid=%s", r.Method, r.URL.Path, uuid)
		webresponse.WriteUnauthorized(w)
		return
	}

	game, err := h.storage.GetGame(r.Context(), uuid)
	if errors.Is(err, domain.ErrGameNotFound) {
		logHandler("%s %s game not found uuid=%s", r.Method, r.URL.Path, uuid)
		webresponse.WriteNotFound(w, messages.GameNotFound)
		return
	}
	if h.writeJoinError(w, r, "failed to load game uuid="+uuid, messages.FailedLoadCurrentGame, err) {
		return
	}

	_, err = h.logic.JoinGame(game, userUUID)
	if err != nil {
		logHandler("%s %s join validation failed uuid=%s user=%s: %v", r.Method, r.URL.Path, uuid, userUUID, err)
		webresponse.WriteBadRequest(w, err.Error())
		return
	}

	game, err = h.storage.JoinGame(r.Context(), uuid, userUUID)
	if errors.Is(err, domain.ErrGameNotJoinable) {
		logHandler("%s %s game not joinable uuid=%s user=%s", r.Method, r.URL.Path, uuid, userUUID)
		webresponse.WriteBadRequest(w, messages.GameNotJoinable)
		return
	}
	if h.writeJoinError(w, r, "failed to save join for uuid="+uuid+" user="+userUUID, messages.FailedSaveCurrentGame, err) {
		return
	}

	logHandler("%s %s joined game uuid=%s user=%s", r.Method, r.URL.Path, uuid, userUUID)
	h.writeGame(w, game)
}

func (h *GameHandler) writeJoinError(w http.ResponseWriter, r *http.Request, logMessage, responseMessage string, err error) bool {
	if err == nil {
		return false
	}

	logHandler("%s %s %s: %v", r.Method, r.URL.Path, logMessage, err)
	webresponse.WriteInternalError(w, responseMessage)
	return true
}
