package handler

import (
	"errors"
	"net/http"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/internal/transport/http/messages"
	"tic-tac-toe/internal/transport/http/middleware"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

// MakeMove applies a move to the current game.
// @Summary Apply a move
// @Description The request body UUID is optional. If it is present, it must match the UUID from the path.
// @Tags games
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param uuid path string true "Game UUID" Format(uuid) default(123e4567-e89b-42d3-a456-426614174000)
// @Param request body dto.GameRequest true "Request body"
// @Success 200 {object} dto.GameResponse "Updated game"
// @Failure 400 {object} dto.ErrorResponse "Invalid body, invalid move, or UUID mismatch"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid Bearer token"
// @Failure 404 {object} dto.ErrorResponse "Game not found"
// @Failure 415 {object} dto.ErrorResponse "Request body must be application/json"
// @Failure 500 {object} dto.ErrorResponse "Move was not saved"
// @Router /games/{uuid}/move [post]
func (h *GameHandler) MakeMove(w http.ResponseWriter, r *http.Request, uuid string) {
	logHandler("%s %s make move request uuid=%s", r.Method, r.URL.Path, uuid)

	if err := validateUUID(uuid); err != nil {
		logHandler("%s %s invalid uuid=%s: %v", r.Method, r.URL.Path, uuid, err)
		webresponse.WriteBadRequest(w, messages.InvalidUUID)
		return
	}

	userUUID := middleware.UserUUIDFromContext(r.Context())
	if userUUID == "" {
		logHandler("%s %s unauthorized move request uuid=%s", r.Method, r.URL.Path, uuid)
		webresponse.WriteUnauthorized(w)
		return
	}

	request, err := h.decodeRequest(r)
	if err != nil {
		logHandler("%s %s invalid move body uuid=%s: %v", r.Method, r.URL.Path, uuid, err)
		if writeDecodeError(w, err) {
			return
		}
		webresponse.WriteBadRequest(w, messages.InvalidRequestBody)
		return
	}
	if request.UUID != googleuuid.Nil && request.UUID.String() != uuid {
		logHandler("%s %s uuid mismatch path=%s body=%s", r.Method, r.URL.Path, uuid, request.UUID.String())
		webresponse.WriteBadRequest(w, messages.UUIDMismatch)
		return
	}
	request.UUID = googleuuid.MustParse(uuid)

	previousGame, err := h.storage.GetGame(r.Context(), uuid)
	if errors.Is(err, domain.ErrGameNotFound) {
		logHandler("%s %s game not found uuid=%s", r.Method, r.URL.Path, uuid)
		webresponse.WriteNotFound(w, messages.GameNotFound)
		return
	}
	if h.writeMoveError(w, r, "failed to load game uuid="+uuid, messages.FailedLoadCurrentGame, err, webresponse.WriteInternalError) {
		return
	}

	currentGame := domain.Game{UUID: request.UUID.String(), Field: domain.Field(request.Field)}
	nextGame, err := h.logic.ApplyMove(previousGame, currentGame, userUUID)
	if h.writeMoveError(w, r, "apply move failed uuid="+uuid+" user="+userUUID, "", err, webresponse.WriteBadRequest) {
		return
	}
	err = h.storage.SaveGameIfUnchanged(r.Context(), previousGame, nextGame)
	if errors.Is(err, domain.ErrGameConflict) {
		logHandler("%s %s move conflict uuid=%s user=%s: %v", r.Method, r.URL.Path, uuid, userUUID, err)
		webresponse.WriteConflict(w, messages.GameConflict)
		return
	}
	if h.writeMoveError(w, r, "save move failed uuid="+uuid+" user="+userUUID, messages.FailedSaveCurrentGame, err, webresponse.WriteInternalError) {
		return
	}

	logHandler("%s %s move applied uuid=%s user=%s next_state=%s winner=%s", r.Method, r.URL.Path, uuid, userUUID, nextGame.State, nextGame.WinnerUUID)
	h.writeGame(w, nextGame)
}

func (h *GameHandler) writeMoveError(w http.ResponseWriter, r *http.Request, logMessage, responseMessage string, err error, write func(http.ResponseWriter, string)) bool {
	if err == nil {
		return false
	}

	if responseMessage == "" {
		responseMessage = err.Error()
	}
	logHandler("%s %s %s: %v", r.Method, r.URL.Path, logMessage, err)
	write(w, responseMessage)
	return true
}
