package application

import (
	"tic-tac-toe/app/domain"
)

func (s *GameService) GetNextMove(game domain.Game) (domain.Game, error) {
	logApplication("get next move", "uuid", game.UUID)

	if game.UUID == "" {
		logApplication("get next move invalid uuid")
		return domain.Game{}, ErrInvalidUUID
	}

	if err := validateFieldShape(game.Field); err != nil {
		logApplication("get next move invalid field", "uuid", game.UUID, "error", err)
		return domain.Game{}, err
	}

	if _, finished := s.CheckGameFinished(game); finished {
		logApplication("get next move finished game", "uuid", game.UUID)
		return domain.Game{}, ErrGameAlreadyFinished
	}

	bestScore := -1000
	bestRow := -1
	bestCol := -1

	for i := 0; i < domain.BoardSize; i++ {
		for j := 0; j < domain.BoardSize; j++ {
			if game.Field[i][j] == domain.CellEmpty {
				game.Field[i][j] = domain.CellComputer
				score := minimax(game.Field, false)
				game.Field[i][j] = domain.CellEmpty

				if score > bestScore {
					bestScore = score
					bestRow = i
					bestCol = j
				}
			}
		}
	}

	if bestRow == -1 || bestCol == -1 {
		logApplication("get next move no move found", "uuid", game.UUID)
		return game, nil
	}

	nextField := cloneField(game.Field)
	nextField[bestRow][bestCol] = domain.CellComputer

	next := domain.Game{
		UUID:       game.UUID,
		Field:      nextField,
		Mode:       game.Mode,
		State:      game.State,
		CreatedAt:  game.CreatedAt,
		NextPlayer: game.NextPlayer,
		Winner:     game.Winner,
		PlayerX:    game.PlayerX,
		PlayerO:    game.PlayerO,
	}
	logApplication("get next move ok", "uuid", game.UUID, "row", bestRow, "col", bestCol)
	return next, nil
}

func evaluate(field domain.Field) int {
	if hasWinner(field, domain.CellComputer) {
		return 10
	}

	if hasWinner(field, domain.CellUser) {
		return -10
	}

	return 0
}

func minimax(field domain.Field, isMaximizing bool) int {
	score := evaluate(field)

	if score == 10 || score == -10 {
		return score
	}

	if isBoardFull(field) {
		return 0
	}

	if isMaximizing {
		best := -1000

		for i := 0; i < domain.BoardSize; i++ {
			for j := 0; j < domain.BoardSize; j++ {
				if field[i][j] == domain.CellEmpty {
					field[i][j] = domain.CellComputer
					value := minimax(field, false)
					field[i][j] = domain.CellEmpty

					if value > best {
						best = value
					}
				}
			}
		}

		return best
	}

	best := 1000

	for i := 0; i < domain.BoardSize; i++ {
		for j := 0; j < domain.BoardSize; j++ {
			if field[i][j] == domain.CellEmpty {
				field[i][j] = domain.CellUser
				value := minimax(field, true)
				field[i][j] = domain.CellEmpty

				if value < best {
					best = value
				}
			}
		}
	}

	return best
}
