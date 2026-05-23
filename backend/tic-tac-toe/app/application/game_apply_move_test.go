package application

import (
	"errors"
	"testing"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/domain"
)

func TestApplyMoveAllowsSecondPlayerO(t *testing.T) {
	service := NewGameService()
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	previous := domain.Game{
		UUID:       "game-1",
		Field:      emptyField(),
		Mode:       domain.GameModePlayer,
		State:      domain.GameStatePlayerToMove,
		NextPlayer: domain.NewUserPlayerRef(userUUID),
		PlayerX:    domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174002")),
		PlayerO:    domain.NewUserPlayerRef(userUUID),
	}
	current := previous
	current.Field = cloneField(previous.Field)
	current.Field[0][0] = domain.CellO

	next, err := service.ApplyMove(previous, current, userUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if next.Field[0][0] != domain.CellO {
		t.Fatalf("expected O move to be accepted, got %#v", next.Field)
	}
	if next.NextPlayer.String() != "123e4567-e89b-42d3-a456-426614174002" {
		t.Fatalf("expected next turn for user-x, got %q", next.NextPlayer.String())
	}
}

func TestApplyMoveRejectsInvalidStatesAndUsers(t *testing.T) {
	service := NewGameService()
	userUUIDX := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174002")
	userUUIDO := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174003")
	base := domain.Game{
		UUID:       "game-1",
		Field:      emptyField(),
		Mode:       domain.GameModePlayer,
		State:      domain.GameStatePlayerToMove,
		NextPlayer: domain.NewUserPlayerRef(userUUIDX),
		PlayerX:    domain.NewUserPlayerRef(userUUIDX),
		PlayerO:    domain.NewUserPlayerRef(userUUIDO),
	}
	current := base
	current.Field = cloneField(base.Field)
	current.Field[0][0] = domain.CellX

	tests := []struct {
		name string
		game domain.Game
		user googleuuid.UUID
		err  error
	}{
		{name: "empty user", game: base, user: googleuuid.Nil, err: ErrInvalidUUID},
		{name: "waiting players", game: func() domain.Game { g := base; g.State = domain.GameStateWaitingPlayers; return g }(), user: userUUIDX, err: ErrGameNotJoinable},
		{name: "finished", game: func() domain.Game { g := base; g.State = domain.GameStateDraw; return g }(), user: userUUIDX, err: ErrGameAlreadyFinished},
		{name: "wrong turn", game: base, user: userUUIDO, err: ErrNotPlayerTurn},
		{name: "not player", game: func() domain.Game {
			g := base
			g.NextPlayer = domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174004"))
			g.PlayerX = domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174005"))
			g.PlayerO = domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174006"))
			return g
		}(), user: googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174004"), err: ErrUserNotGamePlayer},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.ApplyMove(tt.game, current, tt.user)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected %v, got %v", tt.err, err)
			}
		})
	}
}

func TestApplyMoveLegacyGameMetadataAndWin(t *testing.T) {
	service := NewGameService()
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174002")
	previous := domain.Game{UUID: "game-1", Field: domain.Field{
		{domain.CellX, domain.CellX, domain.CellEmpty},
		{domain.CellO, domain.CellO, domain.CellEmpty},
		{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
	}}
	current := previous
	current.Field = cloneField(previous.Field)
	current.Field[0][2] = domain.CellX

	next, err := service.ApplyMove(previous, current, userUUID)
	if err != nil {
		t.Fatalf("unexpected apply move error: %v", err)
	}
	if next.State != domain.GameStatePlayerWins || next.Winner.String() != userUUID.String() {
		t.Fatalf("unexpected winning game: %#v", next)
	}
}

func TestApplyMoveRejectsChangedPreviousMove(t *testing.T) {
	service := NewGameService()
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174002")
	previous := domain.Game{
		UUID:       "game-1",
		Field:      emptyField(),
		Mode:       domain.GameModePlayer,
		State:      domain.GameStatePlayerToMove,
		NextPlayer: domain.NewUserPlayerRef(userUUID),
		PlayerX:    domain.NewUserPlayerRef(userUUID),
		PlayerO:    domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174003")),
	}
	previous.Field[0][0] = domain.CellX
	current := previous
	current.Field = cloneField(previous.Field)
	current.Field[0][0] = domain.CellO

	_, err := service.ApplyMove(previous, current, userUUID)
	if !errors.Is(err, ErrPreviousMovesChanged) {
		t.Fatalf("expected ErrPreviousMovesChanged, got %v", err)
	}
}

func TestApplyMoveLetsComputerRespondAndReturnTurn(t *testing.T) {
	service := NewGameService()
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174002")
	previous := domain.Game{
		UUID:       "game-1",
		Field:      emptyField(),
		Mode:       domain.GameModeComputer,
		State:      domain.GameStatePlayerToMove,
		NextPlayer: domain.NewUserPlayerRef(userUUID),
		PlayerX:    domain.NewUserPlayerRef(userUUID),
		PlayerO:    domain.NewComputerPlayerRef(),
	}
	current := previous
	current.Field = cloneField(previous.Field)
	current.Field[0][0] = domain.CellX

	next, err := service.ApplyMove(previous, current, userUUID)
	if err != nil {
		t.Fatalf("unexpected apply move error: %v", err)
	}
	if next.NextPlayer.String() != userUUID.String() || next.State != domain.GameStatePlayerToMove {
		t.Fatalf("expected turn to return to user-x, got %#v", next)
	}

	computerMoves := 0
	for i := 0; i < domain.BoardSize; i++ {
		for j := 0; j < domain.BoardSize; j++ {
			if next.Field[i][j] == domain.CellO {
				computerMoves++
			}
		}
	}
	if computerMoves != 1 {
		t.Fatalf("expected exactly one computer move, got %d in %#v", computerMoves, next.Field)
	}
}
