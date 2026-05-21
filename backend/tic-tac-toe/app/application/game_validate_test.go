package application

import (
	"errors"
	"testing"

	"tic-tac-toe/app/domain"
)

func TestValidatePlayerMove(t *testing.T) {
	validPrevious := domain.Game{
		UUID: "game-1",
		Field: domain.Field{
			{domain.CellUser, domain.CellEmpty, domain.CellEmpty},
			{domain.CellEmpty, domain.CellComputer, domain.CellEmpty},
			{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
		},
	}

	t.Run("invalid uuid cases", func(t *testing.T) {
		_, err := validatePlayerMove(domain.Game{}, validPrevious, domain.CellUser)
		if !errors.Is(err, ErrInvalidUUID) {
			t.Fatalf("expected ErrInvalidUUID, got %v", err)
		}

		_, err = validatePlayerMove(validPrevious, domain.Game{UUID: "another", Field: emptyField()}, domain.CellUser)
		if !errors.Is(err, ErrInvalidUUID) {
			t.Fatalf("expected ErrInvalidUUID, got %v", err)
		}
	})

	t.Run("invalid previous field", func(t *testing.T) {
		_, err := validatePlayerMove(domain.Game{UUID: "game-1", Field: domain.Field{{0}}}, validPrevious, domain.CellUser)
		if !errors.Is(err, ErrInvalidFieldSize) {
			t.Fatalf("expected ErrInvalidFieldSize, got %v", err)
		}
	})

	t.Run("invalid current field", func(t *testing.T) {
		_, err := validatePlayerMove(validPrevious, domain.Game{UUID: "game-1", Field: domain.Field{{0}}}, domain.CellUser)
		if !errors.Is(err, ErrInvalidFieldSize) {
			t.Fatalf("expected ErrInvalidFieldSize, got %v", err)
		}
	})

	t.Run("invalid cell value", func(t *testing.T) {
		current := domain.Game{
			UUID: "game-1",
			Field: domain.Field{
				{domain.CellUser, 99, domain.CellEmpty},
				{domain.CellEmpty, domain.CellComputer, domain.CellEmpty},
				{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
			},
		}
		_, err := validatePlayerMove(validPrevious, current, domain.CellUser)
		if !errors.Is(err, ErrInvalidCellValue) {
			t.Fatalf("expected ErrInvalidCellValue, got %v", err)
		}
	})

	t.Run("previous moves changed", func(t *testing.T) {
		current := domain.Game{
			UUID: "game-1",
			Field: domain.Field{
				{domain.CellComputer, domain.CellEmpty, domain.CellEmpty},
				{domain.CellEmpty, domain.CellComputer, domain.CellEmpty},
				{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
			},
		}
		_, err := validatePlayerMove(validPrevious, current, domain.CellUser)
		if !errors.Is(err, ErrPreviousMovesChanged) {
			t.Fatalf("expected ErrPreviousMovesChanged, got %v", err)
		}
	})

	t.Run("invalid user move value", func(t *testing.T) {
		current := domain.Game{
			UUID: "game-1",
			Field: domain.Field{
				{domain.CellUser, domain.CellComputer, domain.CellEmpty},
				{domain.CellEmpty, domain.CellComputer, domain.CellEmpty},
				{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
			},
		}
		_, err := validatePlayerMove(validPrevious, current, domain.CellUser)
		if !errors.Is(err, ErrInvalidUserMove) {
			t.Fatalf("expected ErrInvalidUserMove, got %v", err)
		}
	})

	t.Run("invalid number of moves", func(t *testing.T) {
		current := domain.Game{
			UUID: "game-1",
			Field: domain.Field{
				{domain.CellUser, domain.CellUser, domain.CellEmpty},
				{domain.CellUser, domain.CellComputer, domain.CellEmpty},
				{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
			},
		}
		_, err := validatePlayerMove(validPrevious, current, domain.CellUser)
		if !errors.Is(err, ErrInvalidUserMove) {
			t.Fatalf("expected ErrInvalidUserMove, got %v", err)
		}
	})

	t.Run("valid move", func(t *testing.T) {
		current := domain.Game{
			UUID: "game-1",
			Field: domain.Field{
				{domain.CellUser, domain.CellUser, domain.CellEmpty},
				{domain.CellEmpty, domain.CellComputer, domain.CellEmpty},
				{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
			},
		}
		next, err := validatePlayerMove(validPrevious, current, domain.CellUser)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if next.Field[0][1] != domain.CellUser {
			t.Fatalf("expected accepted user move, got %#v", next.Field)
		}
	})
}
