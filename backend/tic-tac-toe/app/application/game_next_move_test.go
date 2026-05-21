package application

import (
	"errors"
	"testing"

	"tic-tac-toe/app/domain"
)

func TestGetNextMove(t *testing.T) {
	service := NewGameService()

	t.Run("returns error for empty uuid", func(t *testing.T) {
		_, err := service.GetNextMove(domain.Game{Field: emptyField()})
		if !errors.Is(err, ErrInvalidUUID) {
			t.Fatalf("expected ErrInvalidUUID, got %v", err)
		}
	})

	t.Run("returns error for invalid field", func(t *testing.T) {
		_, err := service.GetNextMove(domain.Game{
			UUID:  "game-1",
			Field: domain.Field{{0, 0}, {0, 0}},
		})
		if !errors.Is(err, ErrInvalidFieldSize) {
			t.Fatalf("expected ErrInvalidFieldSize, got %v", err)
		}
	})

	t.Run("returns error when game already finished", func(t *testing.T) {
		_, err := service.GetNextMove(domain.Game{
			UUID: "game-1",
			Field: domain.Field{
				{domain.CellUser, domain.CellUser, domain.CellUser},
				{domain.CellEmpty, domain.CellComputer, domain.CellEmpty},
				{domain.CellComputer, domain.CellEmpty, domain.CellEmpty},
			},
		})
		if !errors.Is(err, ErrGameAlreadyFinished) {
			t.Fatalf("expected ErrGameAlreadyFinished, got %v", err)
		}
	})

	t.Run("returns computer move without mutating input field", func(t *testing.T) {
		game := domain.Game{
			UUID: "game-1",
			Field: domain.Field{
				{domain.CellUser, domain.CellComputer, domain.CellUser},
				{domain.CellComputer, domain.CellEmpty, domain.CellEmpty},
				{domain.CellEmpty, domain.CellUser, domain.CellComputer},
			},
		}

		original := cloneField(game.Field)
		next, err := service.GetNextMove(game)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if next.UUID != game.UUID {
			t.Fatalf("expected same uuid, got %q", next.UUID)
		}

		if next.Field[1][1] != domain.CellComputer {
			t.Fatalf("expected winning/blocking move at [1][1], got %v", next.Field)
		}

		if game.Field[1][1] != original[1][1] {
			t.Fatal("expected input field not to be mutated")
		}
	})
}
