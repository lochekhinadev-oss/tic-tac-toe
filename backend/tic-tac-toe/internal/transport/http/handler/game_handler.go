package handler

import (
	"errors"
	"net/http"
	"strconv"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/application"
	"tic-tac-toe/app/domain"
	"tic-tac-toe/internal/transport/http/dto"
	"tic-tac-toe/internal/transport/http/messages"
	"tic-tac-toe/internal/transport/http/middleware"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

type GameHandler struct {
	commands GameCommandService
	storage  GameCommandStorage
	queries  GameQueryService
}

const maxTopPlayersLimit = 100

func NewGameHandler(commands GameCommandService, storage GameCommandStorage, queries GameQueryService) *GameHandler {
	return &GameHandler{
		commands: commands,
		storage:  storage,
		queries:  queries,
	}
}

func newGameHandler(commands GameCommandService, storage GameStorage) *GameHandler {
	return NewGameHandler(commands, storage, storage)
}

// CreateGame creates a new game against the computer or another player.
// @Summary Create a new game
// @Description Creates a computer game by default when the body is empty or mode is omitted. Supported modes are "computer" and "player".
// @Tags games
// @Accept json
// @Produce json
// @Security SessionCookieAuth
// @Param request body dto.CreateGameRequest false "Request body. Can be omitted."
// @Success 201 {object} dto.GameResponse "Created game"
// @Failure 400 {object} dto.ErrorResponse "Invalid request body or mode"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid session cookie"
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
	creatorUUID, ok := mustParseUUID(userUUID)
	if !ok {
		logHandler("%s %s invalid authenticated user uuid=%s", r.Method, r.URL.Path, userUUID)
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

	uuid, err := googleuuid.NewRandom()
	if err != nil {
		logHandler("%s %s failed to generate game uuid: %v", r.Method, r.URL.Path, err)
		webresponse.WriteInternalError(w, messages.FailedCreateGame)
		return
	}

	game, err := h.commands.CreateGame(uuid, creatorUUID, mode)
	if errors.Is(err, application.ErrInvalidGameMode) {
		if h.writeCreateError(w, r, "invalid game mode="+string(mode)+" user="+userUUID, err.Error(), err, webresponse.WriteBadRequest) {
			return
		}
		return
	}
	if h.writeCreateError(w, r, "create game failed for user="+userUUID+" mode="+string(mode), messages.FailedCreateGame, err, webresponse.WriteBadRequest) {
		return
	}

	if h.writeCreateError(w, r, "save game failed for uuid="+uuid.String(), messages.FailedSaveCurrentGame, h.storage.SaveGame(r.Context(), game), webresponse.WriteInternalError) {
		return
	}

	logHandler("%s %s created game uuid=%s mode=%s", r.Method, r.URL.Path, game.UUID, game.Mode)
	webresponse.WriteJSON(w, http.StatusCreated, gameResponse(game))
}

// JoinGame joins a waiting player-vs-player game.
// @Summary Join a game
// @Description Joins an existing player-vs-player game that is waiting for a second player. Request body is not used.
// @Tags games
// @Produce json
// @Security SessionCookieAuth
// @Param uuid path string true "Game UUID" Format(uuid) default(123e4567-e89b-42d3-a456-426614174000)
// @Success 200 {object} dto.GameResponse "Joined game"
// @Failure 400 {object} dto.ErrorResponse "Game cannot be joined"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid session cookie"
// @Failure 404 {object} dto.ErrorResponse "Game not found"
// @Failure 500 {object} dto.ErrorResponse "Join was not saved"
// @Router /games/{uuid}/join [post]
func (h *GameHandler) JoinGame(w http.ResponseWriter, r *http.Request, uuid googleuuid.UUID) {
	logHandler("%s %s join game request uuid=%s", r.Method, r.URL.Path, uuid)
	if uuid == googleuuid.Nil {
		webresponse.WriteBadRequest(w, messages.InvalidUUID)
		return
	}

	userUUID, ok := mustParseUUID(middleware.UserUUIDFromContext(r.Context()))
	if !ok {
		logHandler("%s %s unauthorized join game request uuid=%s", r.Method, r.URL.Path, uuid)
		webresponse.WriteUnauthorized(w)
		return
	}

	game, err := h.queries.GetGame(r.Context(), uuid)
	if errors.Is(err, domain.ErrGameNotFound) {
		logHandler("%s %s game not found uuid=%s", r.Method, r.URL.Path, uuid)
		webresponse.WriteNotFound(w, messages.GameNotFound)
		return
	}
	if h.writeJoinError(w, r, "failed to load game uuid="+uuid.String(), messages.FailedLoadCurrentGame, err) {
		return
	}

	_, err = h.commands.JoinGame(game, userUUID)
	if err != nil {
		logHandler("%s %s join validation failed uuid=%s user=%s: %v", r.Method, r.URL.Path, uuid.String(), userUUID.String(), err)
		webresponse.WriteBadRequest(w, err.Error())
		return
	}

	game, err = h.storage.JoinGame(r.Context(), uuid, userUUID)
	if errors.Is(err, domain.ErrGameNotJoinable) {
		logHandler("%s %s game not joinable uuid=%s user=%s", r.Method, r.URL.Path, uuid.String(), userUUID.String())
		webresponse.WriteBadRequest(w, messages.GameNotJoinable)
		return
	}
	if h.writeJoinError(w, r, "failed to save join for uuid="+uuid.String()+" user="+userUUID.String(), messages.FailedSaveCurrentGame, err) {
		return
	}

	logHandler("%s %s joined game uuid=%s user=%s", r.Method, r.URL.Path, uuid.String(), userUUID.String())
	h.writeGame(w, game)
}

// MakeMove applies a move to the current game.
// @Summary Apply a move
// @Description The request body UUID is optional. If it is present, it must match the UUID from the path.
// @Tags games
// @Accept json
// @Produce json
// @Security SessionCookieAuth
// @Param uuid path string true "Game UUID" Format(uuid) default(123e4567-e89b-42d3-a456-426614174000)
// @Param request body dto.GameRequest true "Request body"
// @Success 200 {object} dto.GameResponse "Updated game"
// @Failure 400 {object} dto.ErrorResponse "Invalid body, invalid move, or UUID mismatch"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid session cookie"
// @Failure 404 {object} dto.ErrorResponse "Game not found"
// @Failure 415 {object} dto.ErrorResponse "Request body must be application/json"
// @Failure 500 {object} dto.ErrorResponse "Move was not saved"
// @Router /games/{uuid}/move [post]
func (h *GameHandler) MakeMove(w http.ResponseWriter, r *http.Request, uuid googleuuid.UUID) {
	logHandler("%s %s make move request uuid=%s", r.Method, r.URL.Path, uuid)
	if uuid == googleuuid.Nil {
		webresponse.WriteBadRequest(w, messages.InvalidUUID)
		return
	}

	userUUID, ok := mustParseUUID(middleware.UserUUIDFromContext(r.Context()))
	if !ok {
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
	if request.UUID != googleuuid.Nil && request.UUID.String() != uuid.String() {
		logHandler("%s %s uuid mismatch path=%s body=%s", r.Method, r.URL.Path, uuid, request.UUID.String())
		webresponse.WriteBadRequest(w, messages.UUIDMismatch)
		return
	}
	request.UUID = uuid

	previousGame, err := h.queries.GetGame(r.Context(), uuid)
	if errors.Is(err, domain.ErrGameNotFound) {
		logHandler("%s %s game not found uuid=%s", r.Method, r.URL.Path, uuid)
		webresponse.WriteNotFound(w, messages.GameNotFound)
		return
	}
	if h.writeMoveError(w, r, "failed to load game uuid="+uuid.String(), messages.FailedLoadCurrentGame, err, webresponse.WriteInternalError) {
		return
	}

	currentGame := domain.Game{UUID: request.UUID.String(), Field: domain.Field(request.Field)}
	nextGame, err := h.commands.ApplyMove(previousGame, currentGame, userUUID)
	if h.writeMoveError(w, r, "apply move failed uuid="+uuid.String()+" user="+userUUID.String(), "", err, webresponse.WriteBadRequest) {
		return
	}
	err = h.storage.SaveGameIfUnchanged(r.Context(), previousGame, nextGame)
	if errors.Is(err, domain.ErrGameConflict) {
		logHandler("%s %s move conflict uuid=%s user=%s: %v", r.Method, r.URL.Path, uuid.String(), userUUID.String(), err)
		webresponse.WriteConflict(w, messages.GameConflict)
		return
	}
	if h.writeMoveError(w, r, "save move failed uuid="+uuid.String()+" user="+userUUID.String(), messages.FailedSaveCurrentGame, err, webresponse.WriteInternalError) {
		return
	}

	logHandler("%s %s move applied uuid=%s user=%s next_state=%s winner=%s", r.Method, r.URL.Path, uuid.String(), userUUID.String(), nextGame.State, nextGame.Winner.String())
	h.writeGame(w, nextGame)
}

// ListGames returns active games available to join.
// @Summary List available games
// @Description Returns active games visible to authenticated users. Request body is not used.
// @Tags games
// @Produce json
// @Security SessionCookieAuth
// @Success 200 {object} dto.GamesResponse "Active games"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid session cookie"
// @Failure 500 {object} dto.ErrorResponse "Games were not loaded"
// @Router /games [get]
func (h *GameHandler) ListGames(w http.ResponseWriter, r *http.Request) {
	logHandler("%s %s list games request", r.Method, r.URL.Path)

	games, err := h.queries.ListActiveGames(r.Context())
	if h.writeLoadError(w, r, "failed to load current games", messages.FailedLoadCurrentGames, err) {
		return
	}

	logHandler("%s %s returned %d active games", r.Method, r.URL.Path, len(games))
	webresponse.WriteJSON(w, http.StatusOK, gamesResponse(games))
}

// ListCompletedGames returns completed games for the authenticated user.
// @Summary List completed games
// @Description Returns games completed by the authenticated user. Includes wins by the user and draws involving the user.
// @Tags games
// @Produce json
// @Security SessionCookieAuth
// @Success 200 {object} dto.GameHistoryResponse "Completed games"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid session cookie"
// @Failure 500 {object} dto.ErrorResponse "Games were not loaded"
// @Router /games/history [get]
func (h *GameHandler) ListCompletedGames(w http.ResponseWriter, r *http.Request) {
	logHandler("%s %s list completed games request", r.Method, r.URL.Path)

	userUUID := middleware.UserUUIDFromContext(r.Context())
	if userUUID == "" {
		logHandler("%s %s unauthorized completed games request", r.Method, r.URL.Path)
		webresponse.WriteUnauthorized(w)
		return
	}
	parsedUserUUID, ok := mustParseUUID(userUUID)
	if !ok {
		logHandler("%s %s invalid authenticated user uuid=%s", r.Method, r.URL.Path, userUUID)
		webresponse.WriteUnauthorized(w)
		return
	}

	games, err := h.queries.ListCompletedGamesByUserUUID(r.Context(), parsedUserUUID)
	if h.writeLoadError(w, r, "failed to load completed games", messages.FailedLoadCompleted, err) {
		return
	}

	logHandler("%s %s returned %d completed games user=%s", r.Method, r.URL.Path, len(games), userUUID)
	webresponse.WriteJSON(w, http.StatusOK, gameHistoryResponse(games))
}

// ListTopPlayers returns the leaderboard.
// @Summary List top players
// @Description Returns the best players sorted by win ratio in descending order.
// @Tags games
// @Produce json
// @Security SessionCookieAuth
// @Param n query int false "Number of top players" minimum(1) maximum(100) default(10)
// @Success 200 {object} dto.LeaderboardResponse "Top players"
// @Failure 400 {object} dto.ErrorResponse "Invalid limit"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid session cookie"
// @Failure 500 {object} dto.ErrorResponse "Leaderboard was not loaded"
// @Router /games/leaderboard [get]
func (h *GameHandler) ListTopPlayers(w http.ResponseWriter, r *http.Request) {
	logHandler("%s %s leaderboard request", r.Method, r.URL.Path)

	limit, err := parseTopPlayersLimit(r)
	if err != nil {
		logHandler("%s %s invalid leaderboard limit: %v", r.Method, r.URL.Path, err)
		webresponse.WriteBadRequest(w, messages.InvalidLeaderboardLimit)
		return
	}

	players, err := h.queries.ListTopPlayers(r.Context(), limit)
	if h.writeLoadError(w, r, "failed to load leaderboard", messages.FailedLoadLeaderboard, err) {
		return
	}

	logHandler("%s %s returned %d leaderboard players", r.Method, r.URL.Path, len(players))
	webresponse.WriteJSON(w, http.StatusOK, leaderboardResponse(players))
}

// GetGame returns current game state by UUID.
// @Summary Get game by UUID
// @Description Returns the current saved state of one game. Request body is not used.
// @Tags games
// @Produce json
// @Security SessionCookieAuth
// @Param uuid path string true "Game UUID" Format(uuid) default(123e4567-e89b-42d3-a456-426614174000)
// @Success 200 {object} dto.GameResponse "Game state"
// @Failure 400 {object} dto.ErrorResponse "Invalid UUID"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid session cookie"
// @Failure 404 {object} dto.ErrorResponse "Game not found"
// @Failure 500 {object} dto.ErrorResponse "Game was not loaded"
// @Router /games/{uuid} [get]
func (h *GameHandler) GetGame(w http.ResponseWriter, r *http.Request, uuid googleuuid.UUID) {
	logHandler("%s %s get game request uuid=%s", r.Method, r.URL.Path, uuid)
	if uuid == googleuuid.Nil {
		webresponse.WriteBadRequest(w, messages.InvalidUUID)
		return
	}

	game, err := h.queries.GetGame(r.Context(), uuid)
	if errors.Is(err, domain.ErrGameNotFound) {
		logHandler("%s %s game not found uuid=%s", r.Method, r.URL.Path, uuid)
		webresponse.WriteNotFound(w, messages.GameNotFound)
		return
	}
	if h.writeLoadError(w, r, "failed to load current game uuid="+uuid.String(), messages.FailedLoadCurrentGame, err) {
		return
	}

	logHandler("%s %s loaded game uuid=%s state=%s", r.Method, r.URL.Path, uuid, game.State)
	h.writeGame(w, game)
}

func (h *GameHandler) writeGame(w http.ResponseWriter, game domain.Game) {
	webresponse.WriteJSON(w, http.StatusOK, gameResponse(game))
}

func (h *GameHandler) decodeRequest(r *http.Request) (dto.GameRequest, error) {
	return decodeJSONBody[dto.GameRequest](r)
}

func (h *GameHandler) decodeCreateRequest(r *http.Request) (dto.CreateGameRequest, error) {
	return decodeOptionalJSONBody(r, dto.CreateGameRequest{Mode: string(domain.GameModeComputer)})
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

func (h *GameHandler) writeJoinError(w http.ResponseWriter, r *http.Request, logMessage, responseMessage string, err error) bool {
	if err == nil {
		return false
	}

	logHandler("%s %s %s: %v", r.Method, r.URL.Path, logMessage, err)
	webresponse.WriteInternalError(w, responseMessage)
	return true
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

func (h *GameHandler) writeLoadError(w http.ResponseWriter, r *http.Request, logMessage, responseMessage string, err error) bool {
	if err == nil {
		return false
	}

	logHandler("%s %s %s: %v", r.Method, r.URL.Path, logMessage, err)
	webresponse.WriteInternalError(w, responseMessage)
	return true
}

func parseTopPlayersLimit(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("n")
	if raw == "" {
		raw = "10"
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 || limit > maxTopPlayersLimit {
		return 0, errors.New(messages.InvalidLeaderboardLimit)
	}
	return limit, nil
}
