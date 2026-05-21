package application

import (
	"errors"
	"testing"

	"tic-tac-toe/app/domain"
)

func TestApplyMoveAllowsSecondPlayerO(t *testing.T) {
	service := NewGameService()
	previous := domain.Game{
		UUID:           "game-1",
		Field:          emptyField(),
		Mode:           domain.GameModePlayer,
		State:          domain.GameStatePlayerToMove,
		NextPlayerUUID: "user-o",
		PlayerXUUID:    "user-x",
		PlayerOUUID:    "user-o",
	}
	current := previous
	current.Field = cloneField(previous.Field)
	current.Field[0][0] = domain.CellO

	next, err := service.ApplyMove(previous, current, "user-o")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if next.Field[0][0] != domain.CellO {
		t.Fatalf("expected O move to be accepted, got %#v", next.Field)
	}
	if next.NextPlayerUUID != "user-x" {
		t.Fatalf("expected next turn for user-x, got %q", next.NextPlayerUUID)
	}
}

func TestApplyMoveRejectsInvalidStatesAndUsers(t *testing.T) {
	service := NewGameService()
	base := domain.Game{
		UUID:           "game-1",
		Field:          emptyField(),
		Mode:           domain.GameModePlayer,
		State:          domain.GameStatePlayerToMove,
		NextPlayerUUID: "user-x",
		PlayerXUUID:    "user-x",
		PlayerOUUID:    "user-o",
	}
	current := base
	current.Field = cloneField(base.Field)
	current.Field[0][0] = domain.CellX

	tests := []struct {
		name string
		game domain.Game
		user string
		err  error
	}{
		{name: "empty user", game: base, user: "", err: ErrInvalidUUID},
		{name: "waiting players", game: func() domain.Game { g := base; g.State = domain.GameStateWaitingPlayers; return g }(), user: "user-x", err: ErrGameNotJoinable},
		{name: "finished", game: func() domain.Game { g := base; g.State = domain.GameStateDraw; return g }(), user: "user-x", err: ErrGameAlreadyFinished},
		{name: "wrong turn", game: base, user: "user-o", err: ErrNotPlayerTurn},
		{name: "not player", game: func() domain.Game { g := base; g.NextPlayerUUID = "stranger"; return g }(), user: "stranger", err: ErrUserNotGamePlayer},
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
	previous := domain.Game{UUID: "game-1", Field: domain.Field{
		{domain.CellX, domain.CellX, domain.CellEmpty},
		{domain.CellO, domain.CellO, domain.CellEmpty},
		{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
	}}
	current := previous
	current.Field = cloneField(previous.Field)
	current.Field[0][2] = domain.CellX

	next, err := service.ApplyMove(previous, current, "user-x")
	if err != nil {
		t.Fatalf("unexpected apply move error: %v", err)
	}
	if next.State != domain.GameStatePlayerWins || next.WinnerUUID != "user-x" {
		t.Fatalf("unexpected winning game: %#v", next)
	}
}

func TestApplyMoveRejectsChangedPreviousMove(t *testing.T) {
	service := NewGameService()
	previous := domain.Game{
		UUID:           "game-1",
		Field:          emptyField(),
		Mode:           domain.GameModePlayer,
		State:          domain.GameStatePlayerToMove,
		NextPlayerUUID: "user-x",
		PlayerXUUID:    "user-x",
		PlayerOUUID:    "user-o",
	}
	previous.Field[0][0] = domain.CellX
	current := previous
	current.Field = cloneField(previous.Field)
	current.Field[0][0] = domain.CellO

	_, err := service.ApplyMove(previous, current, "user-x")
	if !errors.Is(err, ErrPreviousMovesChanged) {
		t.Fatalf("expected ErrPreviousMovesChanged, got %v", err)
	}
}

func TestApplyMoveLetsComputerRespondAndReturnTurn(t *testing.T) {
	service := NewGameService()
	previous := domain.Game{
		UUID:           "game-1",
		Field:          emptyField(),
		Mode:           domain.GameModeComputer,
		State:          domain.GameStatePlayerToMove,
		NextPlayerUUID: "user-x",
		PlayerXUUID:    "user-x",
		PlayerOUUID:    domain.ComputerPlayerUUID,
	}
	current := previous
	current.Field = cloneField(previous.Field)
	current.Field[0][0] = domain.CellX

	next, err := service.ApplyMove(previous, current, "user-x")
	if err != nil {
		t.Fatalf("unexpected apply move error: %v", err)
	}
	if next.NextPlayerUUID != "user-x" || next.State != domain.GameStatePlayerToMove {
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
