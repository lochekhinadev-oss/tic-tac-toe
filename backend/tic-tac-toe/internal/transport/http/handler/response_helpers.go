package handler

import (
	googleuuid "github.com/google/uuid"
	"net/http"
	"strings"
	"time"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/internal/transport/http/dto"
)

func gameResponse(game domain.Game) dto.GameResponse {
	return dto.GameResponse{
		UUID:           uuidFromString(game.UUID),
		Field:          [][]int(game.Field),
		Mode:           string(game.Mode),
		State:          string(game.State),
		CreatedAt:      game.CreatedAt,
		NextPlayerUUID: game.NextPlayer.String(),
		WinnerUUID:     game.Winner.String(),
		PlayerXUUID:    game.PlayerX.String(),
		PlayerOUUID:    game.PlayerO.String(),
	}
}

func gamesResponse(games []domain.Game) dto.GamesResponse {
	response := dto.GamesResponse{Games: make([]dto.GameResponse, len(games))}
	for i := range games {
		response.Games[i] = gameResponse(games[i])
	}
	return response
}

func gameHistoryResponse(games []domain.Game) dto.GameHistoryResponse {
	return dto.GameHistoryResponse{Games: gamesResponse(games).Games}
}

func leaderboardResponse(players []domain.WonGameInfo) dto.LeaderboardResponse {
	response := dto.LeaderboardResponse{Players: make([]dto.LeaderboardEntry, len(players))}
	for i := range players {
		response.Players[i] = dto.LeaderboardEntry{
			UUID:     uuidFromString(players[i].UserUUID),
			Login:    players[i].Login,
			WinRatio: players[i].WinRatio,
		}
	}
	return response
}

func uuidFromString(value string) googleuuid.UUID {
	parsed, err := googleuuid.Parse(value)
	if err != nil {
		return googleuuid.Nil
	}
	return parsed
}

func newUUID() (string, error) {
	id, err := googleuuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, sessionID string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Domain:   "",
		Expires:  expiresAt.UTC(),
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		Secure:   isRequestSecure(r),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   "",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		Secure:   isRequestSecure(r),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func isRequestSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}

	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		return false
	}

	for _, part := range strings.Split(proto, ",") {
		if strings.EqualFold(strings.TrimSpace(part), "https") {
			return true
		}
	}

	return false
}
