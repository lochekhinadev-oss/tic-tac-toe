package dto

import (
	"time"

	"github.com/google/uuid"
)

type GameRequest struct {
	// Game UUID. Optional in request body; if present, it must match the path UUID.
	UUID uuid.UUID `json:"uuid" format:"uuid" example:"123e4567-e89b-42d3-a456-426614174000"`
	// Current 3x3 field. 0 is empty, 1 is X, 2 is O.
	Field [][]int `json:"field"`
}

type CreateGameRequest struct {
	// Game mode. Use "computer" to play against AI or "player" to wait for another user.
	Mode string `json:"mode" enums:"computer,player" default:"computer" example:"computer"`
}

type GameResponse struct {
	// Game UUID.
	UUID uuid.UUID `json:"uuid" format:"uuid" example:"123e4567-e89b-42d3-a456-426614174000"`
	// Current 3x3 field. 0 is empty, 1 is X, 2 is O.
	Field [][]int `json:"field"`
	// Game mode.
	Mode string `json:"mode,omitempty" enums:"computer,player" example:"computer"`
	// Game state.
	State string `json:"state,omitempty" enums:"waiting_players,player_to_move,draw,player_wins" example:"player_to_move"`
	// Game creation date.
	CreatedAt time.Time `json:"created_at" format:"date-time" example:"2026-05-15T20:00:00Z"`
	// UUID of the player who should move next, or "computer".
	NextPlayerUUID string `json:"next_player_uuid,omitempty" example:"123e4567-e89b-42d3-a456-426614174000"`
	// Winner UUID when state is player_wins.
	WinnerUUID string `json:"winner_uuid,omitempty" example:"123e4567-e89b-42d3-a456-426614174000"`
	// UUID of the player using X.
	PlayerXUUID string `json:"player_x_uuid,omitempty" example:"123e4567-e89b-42d3-a456-426614174000"`
	// UUID of the player using O, or "computer".
	PlayerOUUID string `json:"player_o_uuid,omitempty" example:"computer"`
}

type ErrorResponse struct {
	// Error message.
	Message string `json:"message" example:"invalid request body"`
}

type GamesResponse struct {
	// Active games.
	Games []GameResponse `json:"games"`
}

type GameHistoryResponse struct {
	// Completed games.
	Games []GameResponse `json:"games"`
}

type LeaderboardEntry struct {
	// User UUID.
	UUID uuid.UUID `json:"uuid" format:"uuid" example:"123e4567-e89b-42d3-a456-426614174000"`
	// User login.
	Login string `json:"login" example:"player"`
	// Ratio of wins to losses and draws.
	WinRatio float64 `json:"winRatio" example:"1.5"`
}

type LeaderboardResponse struct {
	// Best players.
	Players []LeaderboardEntry `json:"players"`
}
