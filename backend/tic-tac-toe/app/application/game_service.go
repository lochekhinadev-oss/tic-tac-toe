package application

import (
	"errors"
	"time"

	"tic-tac-toe/app/domain"
)

var (
	ErrInvalidUUID          = errors.New("invalid uuid")
	ErrInvalidFieldSize     = errors.New("invalid field size")
	ErrInvalidCellValue     = errors.New("invalid cell value")
	ErrPreviousMovesChanged = errors.New("previous moves were changed")
	ErrInvalidUserMove      = errors.New("invalid user move")
	ErrGameAlreadyFinished  = errors.New("game is already finished")
	ErrInvalidGameMode      = errors.New("invalid game mode")
	ErrGameNotJoinable      = domain.ErrGameNotJoinable
	ErrNotPlayerTurn        = errors.New("not player turn")
	ErrUserNotGamePlayer    = errors.New("user is not a game player")
)

type GameService struct {
	now func() time.Time
}

func NewGameService() *GameService {
	return NewGameServiceWithClock(time.Now)
}

func NewGameServiceWithClock(now func() time.Time) *GameService {
	if now == nil {
		now = time.Now
	}
	return &GameService{now: now}
}

func (s *GameService) CreateGame(uuid string, creatorUUID string, mode domain.GameMode) (domain.Game, error) {
	logApplication("create game uuid=%q creator=%q mode=%q", uuid, creatorUUID, mode)

	if uuid == "" || creatorUUID == "" {
		logApplication("create game invalid uuid/creator uuid=%q creator=%q", uuid, creatorUUID)
		return domain.Game{}, ErrInvalidUUID
	}

	field := newEmptyDomainField()
	game := domain.Game{
		UUID:        uuid,
		Field:       field,
		Mode:        mode,
		CreatedAt:   s.nowUTC(),
		PlayerXUUID: creatorUUID,
	}

	switch mode {
	case domain.GameModeComputer:
		game.PlayerOUUID = domain.ComputerPlayerUUID
		game.State = domain.GameStatePlayerToMove
		game.NextPlayerUUID = creatorUUID
	case domain.GameModePlayer:
		game.State = domain.GameStateWaitingPlayers
	default:
		logApplication("create game invalid mode uuid=%q mode=%q", uuid, mode)
		return domain.Game{}, ErrInvalidGameMode
	}

	logApplication("create game ok uuid=%q creator=%q mode=%q", uuid, creatorUUID, mode)
	return game, nil
}

func (s *GameService) JoinGame(game domain.Game, userUUID string) (domain.Game, error) {
	logApplication("join game uuid=%q user=%q state=%s mode=%s", game.UUID, userUUID, game.State, game.Mode)

	if game.UUID == "" || userUUID == "" {
		logApplication("join game invalid uuid user=%q game=%q", userUUID, game.UUID)
		return domain.Game{}, ErrInvalidUUID
	}
	if game.Mode != domain.GameModePlayer || game.State != domain.GameStateWaitingPlayers || game.PlayerOUUID != "" {
		logApplication("join game not joinable uuid=%q user=%q", game.UUID, userUUID)
		return domain.Game{}, ErrGameNotJoinable
	}
	if game.PlayerXUUID == userUUID {
		logApplication("join game same player uuid=%q user=%q", game.UUID, userUUID)
		return domain.Game{}, ErrGameNotJoinable
	}

	game.PlayerOUUID = userUUID
	game.State = domain.GameStatePlayerToMove
	game.NextPlayerUUID = game.PlayerXUUID
	logApplication("join game ok uuid=%q user=%q", game.UUID, userUUID)
	return game, nil
}

func (s *GameService) ApplyMove(previous domain.Game, current domain.Game, userUUID string) (domain.Game, error) {
	logApplication("apply move uuid=%q user=%q state=%s mode=%s", previous.UUID, userUUID, previous.State, previous.Mode)

	if userUUID == "" {
		logApplication("apply move invalid user uuid=%q", previous.UUID)
		return domain.Game{}, ErrInvalidUUID
	}
	if previous.State == "" {
		previous = s.withDefaultGameMetadata(previous, userUUID)
	}

	if previous.State == domain.GameStateWaitingPlayers {
		logApplication("apply move not joinable uuid=%q", previous.UUID)
		return domain.Game{}, ErrGameNotJoinable
	}
	if previous.State == domain.GameStateDraw || previous.State == domain.GameStatePlayerWins {
		logApplication("apply move already finished uuid=%q state=%s", previous.UUID, previous.State)
		return domain.Game{}, ErrGameAlreadyFinished
	}
	if previous.NextPlayerUUID != userUUID {
		logApplication("apply move wrong turn uuid=%q next=%q user=%q", previous.UUID, previous.NextPlayerUUID, userUUID)
		return domain.Game{}, ErrNotPlayerTurn
	}

	symbol, err := symbolForUser(previous, userUUID)
	if err != nil {
		logApplication("apply move symbol failed uuid=%q user=%q: %v", previous.UUID, userUUID, err)
		return domain.Game{}, err
	}

	next, err := validatePlayerMove(previous, current, symbol)
	if err != nil {
		logApplication("apply move validation failed uuid=%q user=%q: %v", previous.UUID, userUUID, err)
		return domain.Game{}, err
	}

	next = s.updateGameState(next)
	if next.State != domain.GameStatePlayerToMove {
		logApplication("apply move finished uuid=%q user=%q state=%s winner=%s", next.UUID, userUUID, next.State, next.WinnerUUID)
		return next, nil
	}

	next.NextPlayerUUID = opponentUUID(next, userUUID)

	if next.Mode == domain.GameModeComputer && next.NextPlayerUUID == domain.ComputerPlayerUUID {
		computerGame, err := s.GetNextMove(next)
		if err != nil {
			logApplication("apply move computer move failed uuid=%q user=%q: %v", next.UUID, userUUID, err)
			return domain.Game{}, err
		}
		next.Field = computerGame.Field
		next = s.updateGameState(next)
		if next.State == domain.GameStatePlayerToMove {
			next.NextPlayerUUID = next.PlayerXUUID
		}
	}

	logApplication("apply move ok uuid=%q user=%q next=%q state=%s winner=%s", next.UUID, userUUID, next.NextPlayerUUID, next.State, next.WinnerUUID)
	return next, nil
}
