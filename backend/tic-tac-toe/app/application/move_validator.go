package application

import (
	"fmt"

	"tic-tac-toe/app/domain"
)

func validatePlayerMove(previous domain.Game, current domain.Game, symbol int) (domain.Game, error) {
	if err := validateMoveInputs(previous, current); err != nil {
		return domain.Game{}, err
	}
	if err := validateMoveDiff(previous.Field, current.Field, symbol); err != nil {
		return domain.Game{}, err
	}

	next := previous
	next.Field = cloneField(current.Field)
	return next, nil
}

func validateMoveInputs(previous domain.Game, current domain.Game) error {
	if previous.UUID == "" || current.UUID == "" || previous.UUID != current.UUID {
		return ErrInvalidUUID
	}
	if err := validateFieldShape(previous.Field); err != nil {
		return fmt.Errorf("previous field: %w", err)
	}
	if err := validateFieldShape(current.Field); err != nil {
		return fmt.Errorf("current field: %w", err)
	}
	return nil
}

func validateMoveDiff(previous domain.Field, current domain.Field, symbol int) error {
	diffCount := 0
	for i := 0; i < domain.BoardSize; i++ {
		for j := 0; j < domain.BoardSize; j++ {
			prev := previous[i][j]
			curr := current[i][j]
			if prev == curr {
				continue
			}
			if prev != domain.CellEmpty {
				return ErrPreviousMovesChanged
			}
			if curr != symbol {
				return ErrInvalidUserMove
			}
			diffCount++
		}
	}
	if diffCount != 1 {
		return ErrInvalidUserMove
	}
	return nil
}
