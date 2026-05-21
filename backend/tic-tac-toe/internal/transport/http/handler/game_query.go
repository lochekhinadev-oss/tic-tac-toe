package handler

import (
	"errors"
	"net/http"
	"strconv"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/internal/transport/http/messages"
	"tic-tac-toe/internal/transport/http/middleware"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

const maxTopPlayersLimit = 100

// ListGames returns active games available to join.
// @Summary List available games
// @Description Returns active games visible to authenticated users. Request body is not used.
// @Tags games
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.GamesResponse "Active games"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid Bearer token"
// @Failure 500 {object} dto.ErrorResponse "Games were not loaded"
// @Router /games [get]
func (h *GameHandler) ListGames(w http.ResponseWriter, r *http.Request) {
	logHandler("%s %s list games request", r.Method, r.URL.Path)

	games, err := h.storage.ListActiveGames(r.Context())
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
// @Security BearerAuth
// @Success 200 {object} dto.GameHistoryResponse "Completed games"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid Bearer token"
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

	games, err := h.storage.ListCompletedGamesByUserUUID(r.Context(), userUUID)
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
// @Security BearerAuth
// @Param n query int false "Number of top players" minimum(1) maximum(100) default(10)
// @Success 200 {object} dto.LeaderboardResponse "Top players"
// @Failure 400 {object} dto.ErrorResponse "Invalid limit"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid Bearer token"
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

	players, err := h.storage.ListTopPlayers(r.Context(), limit)
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
// @Security BearerAuth
// @Param uuid path string true "Game UUID" Format(uuid) default(123e4567-e89b-42d3-a456-426614174000)
// @Success 200 {object} dto.GameResponse "Game state"
// @Failure 400 {object} dto.ErrorResponse "Invalid UUID"
// @Failure 401 {object} dto.ErrorResponse "Missing or invalid Bearer token"
// @Failure 404 {object} dto.ErrorResponse "Game not found"
// @Failure 500 {object} dto.ErrorResponse "Game was not loaded"
// @Router /games/{uuid} [get]
func (h *GameHandler) GetGame(w http.ResponseWriter, r *http.Request, uuid string) {
	logHandler("%s %s get game request uuid=%s", r.Method, r.URL.Path, uuid)

	if err := validateUUID(uuid); err != nil {
		logHandler("%s %s invalid uuid=%s: %v", r.Method, r.URL.Path, uuid, err)
		webresponse.WriteBadRequest(w, messages.InvalidUUID)
		return
	}

	game, err := h.storage.GetGame(r.Context(), uuid)
	if errors.Is(err, domain.ErrGameNotFound) {
		logHandler("%s %s game not found uuid=%s", r.Method, r.URL.Path, uuid)
		webresponse.WriteNotFound(w, messages.GameNotFound)
		return
	}
	if h.writeLoadError(w, r, "failed to load current game uuid="+uuid, messages.FailedLoadCurrentGame, err) {
		return
	}

	logHandler("%s %s loaded game uuid=%s state=%s", r.Method, r.URL.Path, uuid, game.State)
	h.writeGame(w, game)
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

func (h *GameHandler) writeLoadError(w http.ResponseWriter, r *http.Request, logMessage, responseMessage string, err error) bool {
	if err == nil {
		return false
	}

	logHandler("%s %s %s: %v", r.Method, r.URL.Path, logMessage, err)
	webresponse.WriteInternalError(w, responseMessage)
	return true
}
