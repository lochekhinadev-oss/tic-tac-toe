package application

import "tic-tac-toe/app/domain"

func emptyField() domain.Field {
	field := make(domain.Field, domain.BoardSize)
	for i := 0; i < domain.BoardSize; i++ {
		field[i] = make([]int, domain.BoardSize)
	}
	return field
}
