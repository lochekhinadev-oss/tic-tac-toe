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
	logApplication("create game", "uuid", uuid, "creator_uuid", creatorUUID, "mode", mode)

	if uuid == googleuuid.Nil || creatorUUID == googleuuid.Nil {
		logApplication("create game invalid uuid", "uuid", uuid, "creator_uuid", creatorUUID)
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
		logApplication("create game invalid mode", "uuid", uuid, "mode", mode)
		return domain.Game{}, ErrInvalidGameMode
	}

	logApplication("create game ok", "uuid", uuid, "creator_uuid", creatorUUID, "mode", mode)
	return game, nil
}

func (s *GameService) JoinGame(game domain.Game, userUUID googleuuid.UUID) (domain.Game, error) {
	logApplication("join game", "uuid", game.UUID, "user_uuid", userUUID, "state", game.State, "mode", game.Mode)

	if game.UUID == "" || userUUID == googleuuid.Nil {
		logApplication("join game invalid uuid", "user_uuid", userUUID, "game_uuid", game.UUID)
		return domain.Game{}, ErrInvalidUUID
	}
	if game.Mode != domain.GameModePlayer || game.State != domain.GameStateWaitingPlayers || !game.PlayerO.IsZero() {
		logApplication("join game not joinable", "uuid", game.UUID, "user_uuid", userUUID)
		return domain.Game{}, ErrGameNotJoinable
	}
	if game.PlayerX.Matches(userUUID) {
		logApplication("join game same player", "uuid", game.UUID, "user_uuid", userUUID)
		return domain.Game{}, ErrGameNotJoinable
	}

	game.PlayerO = domain.NewUserPlayerRef(userUUID)
	game.State = domain.GameStatePlayerToMove
	game.NextPlayer = game.PlayerX
	logApplication("join game ok", "uuid", game.UUID, "user_uuid", userUUID)
	return game, nil
}

func (s *GameService) ApplyMove(previous domain.Game, current domain.Game, userUUID googleuuid.UUID) (domain.Game, error) {
	logApplication("apply move", "uuid", previous.UUID, "user_uuid", userUUID, "state", previous.State, "mode", previous.Mode)

	if userUUID == googleuuid.Nil {
		logApplication("apply move invalid user uuid", "uuid", previous.UUID)
		return domain.Game{}, ErrInvalidUUID
	}
	if previous.State == "" {
		previous = s.withDefaultGameMetadata(previous, userUUID)
	}

	if previous.State == domain.GameStateWaitingPlayers {
		logApplication("apply move not joinable", "uuid", previous.UUID)
		return domain.Game{}, ErrGameNotJoinable
	}
	if previous.State == domain.GameStateDraw || previous.State == domain.GameStatePlayerWins {
		logApplication("apply move already finished", "uuid", previous.UUID, "state", previous.State)
		return domain.Game{}, ErrGameAlreadyFinished
	}
	if !previous.NextPlayer.Matches(userUUID) {
		logApplication("apply move wrong turn", "uuid", previous.UUID, "next", previous.NextPlayer.String(), "user_uuid", userUUID)
		return domain.Game{}, ErrNotPlayerTurn
	}

	symbol, err := symbolForUser(previous, userUUID.String())
	if err != nil {
		logApplication("apply move symbol failed", "uuid", previous.UUID, "user_uuid", userUUID, "error", err)
		return domain.Game{}, err
	}

	next, err := validatePlayerMove(previous, current, symbol)
	if err != nil {
		logApplication("apply move validation failed", "uuid", previous.UUID, "user_uuid", userUUID, "error", err)
		return domain.Game{}, err
	}

	next = s.updateGameState(next)
	if next.State != domain.GameStatePlayerToMove {
		logApplication("apply move finished", "uuid", next.UUID, "user_uuid", userUUID, "state", next.State, "winner", next.Winner.String())
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
			logApplication("apply move computer move failed", "uuid", next.UUID, "user_uuid", userUUID, "error", err)
			return domain.Game{}, err
		}
		next.Field = computerGame.Field
		next = s.updateGameState(next)
		if next.State == domain.GameStatePlayerToMove {
			next.NextPlayer = next.PlayerX
		}
	}

	logApplication("apply move ok", "uuid", next.UUID, "user_uuid", userUUID, "next", next.NextPlayer.String(), "state", next.State, "winner", next.Winner.String())
	return next, nil
}
