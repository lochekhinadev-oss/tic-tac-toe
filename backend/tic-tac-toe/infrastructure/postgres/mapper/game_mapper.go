package mapper

import (
	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/datasource"
)

func ToDatasourceGame(game domain.Game) datasource.Game {
	return datasource.Game{
		UUID:           game.UUID,
		Field:          toDatasourceField(game.Field),
		Mode:           string(game.Mode),
		State:          string(game.State),
		CreatedAt:      game.CreatedAt,
		NextPlayerUUID: game.NextPlayerUUID,
		WinnerUUID:     game.WinnerUUID,
		PlayerXUUID:    game.PlayerXUUID,
		PlayerOUUID:    game.PlayerOUUID,
	}
}

func ToDomainGame(game datasource.Game) domain.Game {
	return domain.Game{
		UUID:           game.UUID,
		Field:          toDomainField(game.Field),
		Mode:           domain.GameMode(game.Mode),
		State:          domain.GameState(game.State),
		CreatedAt:      game.CreatedAt,
		NextPlayerUUID: game.NextPlayerUUID,
		WinnerUUID:     game.WinnerUUID,
		PlayerXUUID:    game.PlayerXUUID,
		PlayerOUUID:    game.PlayerOUUID,
	}
}

func toDatasourceField(field domain.Field) datasource.Field {
	result := make(datasource.Field, len(field))
	for i := range field {
		result[i] = make([]int, len(field[i]))
		copy(result[i], field[i])
	}

	return result
}

func toDomainField(field datasource.Field) domain.Field {
	result := make(domain.Field, len(field))
	for i := range field {
		result[i] = make([]int, len(field[i]))
		copy(result[i], field[i])
	}

	return result
}
