package application

import (
	"time"

	"tic-tac-toe/app/domain"
)

func (s *GameService) withDefaultGameMetadata(game domain.Game, userUUID string) domain.Game {
	game.Mode = domain.GameModeComputer
	game.State = domain.GameStatePlayerToMove
	if game.CreatedAt.IsZero() {
		game.CreatedAt = s.nowUTC()
	}
	game.NextPlayerUUID = userUUID
	game.PlayerXUUID = userUUID
	game.PlayerOUUID = domain.ComputerPlayerUUID
	return game
}

func (s *GameService) nowUTC() time.Time {
	if s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

func symbolForUser(game domain.Game, userUUID string) (int, error) {
	switch userUUID {
	case game.PlayerXUUID:
		return domain.CellX, nil
	case game.PlayerOUUID:
		return domain.CellO, nil
	default:
		return domain.CellEmpty, ErrUserNotGamePlayer
	}
}

func opponentUUID(game domain.Game, userUUID string) string {
	if userUUID == game.PlayerXUUID {
		return game.PlayerOUUID
	}
	return game.PlayerXUUID
}

func isComputerPlayer(game domain.Game) bool {
	return game.Mode == domain.GameModeComputer || game.PlayerOUUID == "" || game.PlayerOUUID == domain.ComputerPlayerUUID
}
