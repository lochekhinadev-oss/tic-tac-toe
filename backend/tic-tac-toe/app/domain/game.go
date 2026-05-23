package domain

import "time"

type GameMode string

const (
	GameModeComputer GameMode = "computer"
	GameModePlayer   GameMode = "player"
)

type GameState string

const (
	GameStateWaitingPlayers GameState = "waiting_players"
	GameStatePlayerToMove   GameState = "player_to_move"
	GameStateDraw           GameState = "draw"
	GameStatePlayerWins     GameState = "player_wins"
)

type Game struct {
	UUID       string    `db:"uuid"`
	Field      Field     `db:"field"`
	Mode       GameMode  `db:"mode"`
	State      GameState `db:"state"`
	CreatedAt  time.Time `db:"created_at"`
	NextPlayer PlayerRef `db:"next_player_uuid"`
	Winner     PlayerRef `db:"winner_uuid"`
	PlayerX    PlayerRef `db:"player_x_uuid"`
	PlayerO    PlayerRef `db:"player_o_uuid"`
}

type WonGameInfo struct {
	UserUUID string  `db:"user_uuid"`
	Login    string  `db:"login"`
	WinRatio float64 `db:"win_ratio"`
}
