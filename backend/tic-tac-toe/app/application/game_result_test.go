package application

import (
	"testing"

	googleuuid "github.com/google/uuid"

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
				State:  domain.GameStatePlayerWins,
				Field:  emptyField(),
				Winner: domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174101")),
			},
			status:   domain.GameUserWon,
			winner:   "123e4567-e89b-42d3-a456-426614174101",
			finished: true,
		},
		{
			name: "state computer won",
			game: domain.Game{
				State:  domain.GameStatePlayerWins,
				Field:  emptyField(),
				Winner: domain.NewComputerPlayerRef(),
			},
			status:   domain.GameComputerWon,
			winner:   domain.NewComputerPlayerRef().String(),
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
			winner:   domain.NewComputerPlayerRef().String(),
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
				Mode:    domain.GameModePlayer,
				PlayerX: domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174201")),
				PlayerO: domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174202")),
			},
			status:   domain.GameUserWon,
			winner:   "123e4567-e89b-42d3-a456-426614174202",
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
			if result.Status != tt.status || result.Winner.String() != tt.winner || finished != tt.finished {
				t.Fatalf("expected (%v,%q,%v), got (%v,%q,%v)", tt.status, tt.winner, tt.finished, result.Status, result.Winner.String(), finished)
			}
		})
	}
}

func TestUpdateGameStateDrawAndOWins(t *testing.T) {
	service := &GameService{}
	xWins := domain.Game{
		Field:   domain.Field{{domain.CellX, domain.CellX, domain.CellX}, {domain.CellO, domain.CellO, domain.CellEmpty}, {domain.CellEmpty, domain.CellEmpty, domain.CellEmpty}},
		PlayerX: domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174301")),
	}
	got := service.updateGameState(xWins)
	if got.State != domain.GameStatePlayerWins || got.Winner.String() != "123e4567-e89b-42d3-a456-426614174301" || !got.NextPlayer.IsZero() {
		t.Fatalf("unexpected X win state: %#v", got)
	}

	oWins := domain.Game{
		Field:   domain.Field{{domain.CellO, domain.CellX, domain.CellX}, {domain.CellEmpty, domain.CellO, domain.CellX}, {domain.CellEmpty, domain.CellEmpty, domain.CellO}},
		PlayerO: domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174302")),
	}
	got = service.updateGameState(oWins)
	if got.State != domain.GameStatePlayerWins || got.Winner.String() != "123e4567-e89b-42d3-a456-426614174302" {
		t.Fatalf("unexpected O win state: %#v", got)
	}

	draw := domain.Game{Field: domain.Field{
		{domain.CellX, domain.CellO, domain.CellX},
		{domain.CellX, domain.CellO, domain.CellO},
		{domain.CellO, domain.CellX, domain.CellX},
	}}
	got = service.updateGameState(draw)
	if got.State != domain.GameStateDraw || !got.NextPlayer.IsZero() {
		t.Fatalf("unexpected draw state: %#v", got)
	}
}
