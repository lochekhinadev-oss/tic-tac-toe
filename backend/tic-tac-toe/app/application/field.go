package application

import "tic-tac-toe/app/domain"

func validateFieldShape(field domain.Field) error {
	if len(field) != domain.BoardSize {
		return ErrInvalidFieldSize
	}

	for i := 0; i < domain.BoardSize; i++ {
		if len(field[i]) != domain.BoardSize {
			return ErrInvalidFieldSize
		}

		for j := 0; j < domain.BoardSize; j++ {
			switch field[i][j] {
			case domain.CellEmpty, domain.CellUser, domain.CellComputer:
			default:
				return ErrInvalidCellValue
			}
		}
	}

	return nil
}

func cloneField(field domain.Field) domain.Field {
	result := make(domain.Field, len(field))
	for i := range field {
		result[i] = make([]int, len(field[i]))
		copy(result[i], field[i])
	}
	return result
}

func newEmptyDomainField() domain.Field {
	field := make(domain.Field, domain.BoardSize)
	for i := 0; i < domain.BoardSize; i++ {
		field[i] = make([]int, domain.BoardSize)
	}
	return field
}
