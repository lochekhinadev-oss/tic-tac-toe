package application

import (
	"testing"

	"tic-tac-toe/app/domain"
)

func TestCheckGameFinished(t *testing.T) {
	service := NewGameService()

	tests := []struct {
		name     string
		game     domain.Game
		status   domain.GameStatus
		winner   string
		finished bool
	}{
		{name: "invalid field", game: domain.Game{Field: domain.Field{{0}}}},
		{
			name: "state waiting players",
			game: domain.Game{
				State: domain.GameStateWaitingPlayers,
				Field: emptyField(),
			},
			status:   domain.GameWaitingPlayers,
			finished: false,
		},
		{
			name: "state draw",
			game: domain.Game{
				State: domain.GameStateDraw,
				Field: emptyField(),
			},
			status:   domain.GameDraw,
			finished: true,
		},
		{
			name: "state user won",
			game: domain.Game{
				State:      domain.GameStatePlayerWins,
				Field:      emptyField(),
				WinnerUUID: "user-x",
			},
			status:   domain.GameUserWon,
			winner:   "user-x",
			finished: true,
		},
		{
			name: "state computer won",
			game: domain.Game{
				State:      domain.GameStatePlayerWins,
				Field:      emptyField(),
				WinnerUUID: domain.ComputerPlayerUUID,
			},
			status:   domain.GameComputerWon,
			winner:   domain.ComputerPlayerUUID,
			finished: true,
		},
		{
			name: "user won",
			game: domain.Game{Field: domain.Field{
				{domain.CellUser, domain.CellUser, domain.CellUser},
				{domain.CellComputer, domain.CellEmpty, domain.CellComputer},
				{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
			}},
			status:   domain.GameUserWon,
			finished: true,
		},
		{
			name: "computer won",
			game: domain.Game{Field: domain.Field{
				{domain.CellComputer, domain.CellUser, domain.CellUser},
				{domain.CellEmpty, domain.CellComputer, domain.CellUser},
				{domain.CellEmpty, domain.CellEmpty, domain.CellComputer},
			}},
			status:   domain.GameComputerWon,
			winner:   domain.ComputerPlayerUUID,
			finished: true,
		},
		{
			name: "second pvp player won",
			game: domain.Game{
				Field: domain.Field{
					{domain.CellO, domain.CellX, domain.CellX},
					{domain.CellEmpty, domain.CellO, domain.CellX},
					{domain.CellEmpty, domain.CellEmpty, domain.CellO},
				},
				Mode:        domain.GameModePlayer,
				PlayerXUUID: "user-x",
				PlayerOUUID: "user-o",
			},
			status:   domain.GameUserWon,
			winner:   "user-o",
			finished: true,
		},
		{
			name: "draw",
			game: domain.Game{Field: domain.Field{
				{domain.CellUser, domain.CellComputer, domain.CellUser},
				{domain.CellUser, domain.CellComputer, domain.CellComputer},
				{domain.CellComputer, domain.CellUser, domain.CellUser},
			}},
			status:   domain.GameDraw,
			finished: true,
		},
		{
			name: "in progress",
			game: domain.Game{Field: domain.Field{
				{domain.CellUser, domain.CellComputer, domain.CellEmpty},
				{domain.CellEmpty, domain.CellComputer, domain.CellEmpty},
				{domain.CellEmpty, domain.CellUser, domain.CellEmpty},
			}},
			status:   domain.GameInProgress,
			finished: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, finished := service.CheckGameFinished(tt.game)
			if result.Status != tt.status || result.WinnerUUID != tt.winner || finished != tt.finished {
				t.Fatalf("expected (%v,%q,%v), got (%v,%q,%v)", tt.status, tt.winner, tt.finished, result.Status, result.WinnerUUID, finished)
			}
		})
	}
}

func TestUpdateGameStateDrawAndOWins(t *testing.T) {
	service := &GameService{}
	xWins := domain.Game{
		Field:       domain.Field{{domain.CellX, domain.CellX, domain.CellX}, {domain.CellO, domain.CellO, domain.CellEmpty}, {domain.CellEmpty, domain.CellEmpty, domain.CellEmpty}},
		PlayerXUUID: "user-x",
	}
	got := service.updateGameState(xWins)
	if got.State != domain.GameStatePlayerWins || got.WinnerUUID != "user-x" || got.NextPlayerUUID != "" {
		t.Fatalf("unexpected X win state: %#v", got)
	}

	oWins := domain.Game{
		Field:       domain.Field{{domain.CellO, domain.CellX, domain.CellX}, {domain.CellEmpty, domain.CellO, domain.CellX}, {domain.CellEmpty, domain.CellEmpty, domain.CellO}},
		PlayerOUUID: "user-o",
	}
	got = service.updateGameState(oWins)
	if got.State != domain.GameStatePlayerWins || got.WinnerUUID != "user-o" {
		t.Fatalf("unexpected O win state: %#v", got)
	}

	draw := domain.Game{Field: domain.Field{
		{domain.CellX, domain.CellO, domain.CellX},
		{domain.CellX, domain.CellO, domain.CellO},
		{domain.CellO, domain.CellX, domain.CellX},
	}}
	got = service.updateGameState(draw)
	if got.State != domain.GameStateDraw || got.NextPlayerUUID != "" {
		t.Fatalf("unexpected draw state: %#v", got)
	}
}
