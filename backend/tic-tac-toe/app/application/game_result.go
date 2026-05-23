package application

import "tic-tac-toe/app/domain"

func (s *GameService) CheckGameFinished(game domain.Game) (domain.GameResult, bool) {
	switch game.State {
	case domain.GameStateWaitingPlayers:
		return domain.GameResult{Status: domain.GameWaitingPlayers}, false
	case domain.GameStateDraw:
		return domain.GameResult{Status: domain.GameDraw}, true
	case domain.GameStatePlayerWins:
		if game.Winner.IsComputer() {
			return domain.GameResult{Status: domain.GameComputerWon, Winner: game.Winner}, true
		}
		return domain.GameResult{Status: domain.GameUserWon, Winner: game.Winner}, true
	}

	if err := validateFieldShape(game.Field); err != nil {
		return domain.GameResult{Status: domain.GameInProgress}, false
	}

	if hasWinner(game.Field, domain.CellUser) {
		return domain.GameResult{Status: domain.GameUserWon, Winner: game.PlayerX}, true
	}

	if hasWinner(game.Field, domain.CellComputer) {
		winner := game.PlayerO
		if winner.IsZero() && isComputerPlayer(game) {
			winner = domain.NewComputerPlayerRef()
		}
		if isComputerPlayer(game) {
			return domain.GameResult{Status: domain.GameComputerWon, Winner: winner}, true
		}
		return domain.GameResult{Status: domain.GameUserWon, Winner: winner}, true
	}

	if isBoardFull(game.Field) {
		return domain.GameResult{Status: domain.GameDraw}, true
	}

	return domain.GameResult{Status: domain.GameInProgress}, false
}

func (s *GameService) updateGameState(game domain.Game) domain.Game {
	if hasWinner(game.Field, domain.CellX) {
		game.State = domain.GameStatePlayerWins
		game.Winner = game.PlayerX
		game.NextPlayer = domain.PlayerRef{}
		return game
	}

	if hasWinner(game.Field, domain.CellO) {
		game.State = domain.GameStatePlayerWins
		game.Winner = game.PlayerO
		game.NextPlayer = domain.PlayerRef{}
		return game
	}

	if isBoardFull(game.Field) {
		game.State = domain.GameStateDraw
		game.Winner = domain.PlayerRef{}
		game.NextPlayer = domain.PlayerRef{}
		return game
	}

	game.State = domain.GameStatePlayerToMove
	game.Winner = domain.PlayerRef{}
	return game
}

func hasWinner(field domain.Field, player int) bool {
	for i := 0; i < domain.BoardSize; i++ {
		if field[i][0] == player && field[i][1] == player && field[i][2] == player {
			return true
		}
	}

	for j := 0; j < domain.BoardSize; j++ {
		if field[0][j] == player && field[1][j] == player && field[2][j] == player {
			return true
		}
	}

	if field[0][0] == player && field[1][1] == player && field[2][2] == player {
		return true
	}

	if field[0][2] == player && field[1][1] == player && field[2][0] == player {
		return true
	}

	return false
}

func isBoardFull(field domain.Field) bool {
	for i := 0; i < domain.BoardSize; i++ {
		for j := 0; j < domain.BoardSize; j++ {
			if field[i][j] == domain.CellEmpty {
				return false
			}
		}
	}
	return true
}
