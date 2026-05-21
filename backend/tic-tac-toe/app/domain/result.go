package domain

type GameStatus int

const (
	GameInProgress GameStatus = iota
	GameWaitingPlayers
	GameDraw
	GameUserWon
	GameComputerWon
)

type GameResult struct {
	Status     GameStatus
	WinnerUUID string
}
