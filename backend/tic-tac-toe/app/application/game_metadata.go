package application

import (
	"time"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/domain"
)

func (s *GameService) withDefaultGameMetadata(game domain.Game, userUUID googleuuid.UUID) domain.Game {
	game.Mode = domain.GameModeComputer
	game.State = domain.GameStatePlayerToMove
	if game.CreatedAt.IsZero() {
		game.CreatedAt = s.nowUTC()
	}
	game.NextPlayer = domain.NewUserPlayerRef(userUUID)
	game.PlayerX = domain.NewUserPlayerRef(userUUID)
	game.PlayerO = domain.NewComputerPlayerRef()
	return game
}

func (s *GameService) nowUTC() time.Time {
	if s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

func symbolForUser(game domain.Game, userUUID string) (int, error) {
	uuid, err := googleuuid.Parse(userUUID)
	if err != nil {
		return domain.CellEmpty, ErrInvalidUUID
	}
	switch {
	case game.PlayerX.Matches(uuid):
		return domain.CellX, nil
	case game.PlayerO.Matches(uuid):
		return domain.CellO, nil
	default:
		return domain.CellEmpty, ErrUserNotGamePlayer
	}
}

func opponentUUID(game domain.Game, userUUID string) string {
	uuid, err := googleuuid.Parse(userUUID)
	if err != nil {
		return ""
	}
	if game.PlayerX.Matches(uuid) {
		return game.PlayerO.String()
	}
	return game.PlayerX.String()
}

func isComputerPlayer(game domain.Game) bool {
	return game.Mode == domain.GameModeComputer || game.PlayerO.IsZero() || game.PlayerO.IsComputer()
}
