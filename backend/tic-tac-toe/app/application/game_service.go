package application

import (
	"errors"
	"time"

	googleuuid "github.com/google/uuid"

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

func (s *GameService) CreateGame(uuid googleuuid.UUID, creatorUUID googleuuid.UUID, mode domain.GameMode) (domain.Game, error) {
	logApplication("create game uuid=%q creator=%q mode=%q", uuid, creatorUUID, mode)

	if uuid == googleuuid.Nil || creatorUUID == googleuuid.Nil {
		logApplication("create game invalid uuid/creator uuid=%q creator=%q", uuid, creatorUUID)
		return domain.Game{}, ErrInvalidUUID
	}

	field := newEmptyDomainField()
	game := domain.Game{
		UUID:      uuid.String(),
		Field:     field,
		Mode:      mode,
		CreatedAt: s.nowUTC(),
		PlayerX:   domain.NewUserPlayerRef(creatorUUID),
	}

	switch mode {
	case domain.GameModeComputer:
		game.PlayerO = domain.NewComputerPlayerRef()
		game.State = domain.GameStatePlayerToMove
		game.NextPlayer = domain.NewUserPlayerRef(creatorUUID)
	case domain.GameModePlayer:
		game.State = domain.GameStateWaitingPlayers
	default:
		logApplication("create game invalid mode uuid=%q mode=%q", uuid, mode)
		return domain.Game{}, ErrInvalidGameMode
	}

	logApplication("create game ok uuid=%q creator=%q mode=%q", uuid, creatorUUID, mode)
	return game, nil
}

func (s *GameService) JoinGame(game domain.Game, userUUID googleuuid.UUID) (domain.Game, error) {
	logApplication("join game uuid=%q user=%q state=%s mode=%s", game.UUID, userUUID, game.State, game.Mode)

	if game.UUID == "" || userUUID == googleuuid.Nil {
		logApplication("join game invalid uuid user=%q game=%q", userUUID, game.UUID)
		return domain.Game{}, ErrInvalidUUID
	}
	if game.Mode != domain.GameModePlayer || game.State != domain.GameStateWaitingPlayers || !game.PlayerO.IsZero() {
		logApplication("join game not joinable uuid=%q user=%q", game.UUID, userUUID)
		return domain.Game{}, ErrGameNotJoinable
	}
	if game.PlayerX.Matches(userUUID) {
		logApplication("join game same player uuid=%q user=%q", game.UUID, userUUID)
		return domain.Game{}, ErrGameNotJoinable
	}

	game.PlayerO = domain.NewUserPlayerRef(userUUID)
	game.State = domain.GameStatePlayerToMove
	game.NextPlayer = game.PlayerX
	logApplication("join game ok uuid=%q user=%q", game.UUID, userUUID)
	return game, nil
}

func (s *GameService) ApplyMove(previous domain.Game, current domain.Game, userUUID googleuuid.UUID) (domain.Game, error) {
	logApplication("apply move uuid=%q user=%q state=%s mode=%s", previous.UUID, userUUID, previous.State, previous.Mode)

	if userUUID == googleuuid.Nil {
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
	if !previous.NextPlayer.Matches(userUUID) {
		logApplication("apply move wrong turn uuid=%q next=%q user=%q", previous.UUID, previous.NextPlayer.String(), userUUID)
		return domain.Game{}, ErrNotPlayerTurn
	}

	symbol, err := symbolForUser(previous, userUUID.String())
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
		logApplication("apply move finished uuid=%q user=%q state=%s winner=%s", next.UUID, userUUID, next.State, next.Winner.String())
		return next, nil
	}

	nextPlayer := opponentUUID(next, userUUID.String())
	if nextPlayer != "" {
		next.NextPlayer = domain.PlayerRefFromString(nextPlayer)
	} else {
		next.NextPlayer = domain.PlayerRef{}
	}

	if next.Mode == domain.GameModeComputer && next.NextPlayer.IsComputer() {
		computerGame, err := s.GetNextMove(next)
		if err != nil {
			logApplication("apply move computer move failed uuid=%q user=%q: %v", next.UUID, userUUID, err)
			return domain.Game{}, err
		}
		next.Field = computerGame.Field
		next = s.updateGameState(next)
		if next.State == domain.GameStatePlayerToMove {
			next.NextPlayer = next.PlayerX
		}
	}

	logApplication("apply move ok uuid=%q user=%q next=%q state=%s winner=%s", next.UUID, userUUID, next.NextPlayer.String(), next.State, next.Winner.String())
	return next, nil
}
