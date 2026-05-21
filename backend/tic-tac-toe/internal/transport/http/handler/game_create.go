package handler

import (
	"errors"
	"net/http"

	"tic-tac-toe/app/application"
	"tic-tac-toe/app/domain"
	"tic-tac-toe/internal/transport/http/messages"
	"tic-tac-toe/internal/transport/http/middleware"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

// CreateGame creates a new game against the computer or another player.
// @Summary Create a new game
// @Description Creates a computer game by default when the body is empty or mode is omitted. Supported modes are "computer" and "player".
// @Tags games
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateGameRequest false "Request body. Can be omitted."
// @Success 201 {object} dto.GameResponse "Created game"
// @Failure 400 {object} dto.ErrorResponse "Invalid request body or mode"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid Bearer token"
// @Failure 415 {object} dto.ErrorResponse "Request body must be application/json"
// @Failure 500 {object} dto.ErrorResponse "Game was not saved"
// @Router /games [post]
func (h *GameHandler) CreateGame(w http.ResponseWriter, r *http.Request) {
	logHandler("%s %s create game request", r.Method, r.URL.Path)

	userUUID := middleware.UserUUIDFromContext(r.Context())
	if userUUID == "" {
		logHandler("%s %s unauthorized create game request", r.Method, r.URL.Path)
		webresponse.WriteUnauthorized(w)
		return
	}

	request, err := h.decodeCreateRequest(r)
	if err != nil {
		logHandler("%s %s invalid create game body: %v", r.Method, r.URL.Path, err)
		if writeDecodeError(w, err) {
			return
		}
		webresponse.WriteBadRequest(w, messages.InvalidRequestBody)
		return
	}
	mode := domain.GameMode(request.Mode)
	if mode == "" {
		mode = domain.GameModeComputer
	}

	uuid, err := newUUID()
	if err != nil {
		logHandler("%s %s failed to generate game uuid: %v", r.Method, r.URL.Path, err)
		webresponse.WriteInternalError(w, messages.FailedCreateGame)
		return
	}

	game, err := h.logic.CreateGame(uuid, userUUID, mode)
	if errors.Is(err, application.ErrInvalidGameMode) {
		if h.writeCreateError(w, r, "invalid game mode="+string(mode)+" user="+userUUID, err.Error(), err, webresponse.WriteBadRequest) {
			return
		}
		return
	}
	if h.writeCreateError(w, r, "create game failed for user="+userUUID+" mode="+string(mode), messages.FailedCreateGame, err, webresponse.WriteBadRequest) {
		return
	}

	if h.writeCreateError(w, r, "save game failed for uuid="+uuid, messages.FailedSaveCurrentGame, h.storage.SaveGame(r.Context(), game), webresponse.WriteInternalError) {
		return
	}

	logHandler("%s %s created game uuid=%s mode=%s", r.Method, r.URL.Path, game.UUID, game.Mode)
	webresponse.WriteJSON(w, http.StatusCreated, gameResponse(game))
}

func (h *GameHandler) writeCreateError(w http.ResponseWriter, r *http.Request, logMessage, responseMessage string, err error, write func(http.ResponseWriter, string)) bool {
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
