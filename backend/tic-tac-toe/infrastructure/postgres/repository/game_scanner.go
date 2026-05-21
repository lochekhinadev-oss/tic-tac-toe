package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/infrastructure/postgres/mapper"
)

func scanGames(rows pgx.Rows, operation string) ([]domain.Game, error) {
	var games []domain.Game
	for rows.Next() {
		game, err := scanGameRow(rows)
		if err != nil {
			logRepository("%s scan failed: %v", operation, err)
			return nil, err
		}
		games = append(games, game)
	}

	if err := rows.Err(); err != nil {
		logRepository("%s rows error: %v", operation, err)
		return nil, err
	}

	return games, nil
}

func scanGameRow(row pgx.Row) (domain.Game, error) {
	var uuid sql.NullString
	var mode sql.NullString
	var state sql.NullString
	var createdAt sql.NullTime
	var nextPlayerUUID sql.NullString
	var winnerUUID sql.NullString
	var playerXUUID sql.NullString
	var playerOUUID sql.NullString
	var field []byte
	if err := row.Scan(
		&uuid,
		&field,
		&mode,
		&state,
		&createdAt,
		&nextPlayerUUID,
		&winnerUUID,
		&playerXUUID,
		&playerOUUID,
	); err != nil {
		return domain.Game{}, err
	}

	game, err := buildDatasourceGame(uuid, field, mode, state, createdAt, nextPlayerUUID, winnerUUID, playerXUUID, playerOUUID)
	if err != nil {
		return domain.Game{}, err
	}

	return mapper.ToDomainGame(game), nil
}

func buildDatasourceGame(uuid sql.NullString, rawField []byte, mode sql.NullString, state sql.NullString, createdAt sql.NullTime, nextPlayerUUID sql.NullString, winnerUUID sql.NullString, playerXUUID sql.NullString, playerOUUID sql.NullString) (datasource.Game, error) {
	gameUUID, err := requiredString("games.uuid", uuid)
	if err != nil {
		return datasource.Game{}, err
	}
	gameMode, err := requiredString("games.mode", mode)
	if err != nil {
		return datasource.Game{}, err
	}
	if gameMode != string(domain.GameModeComputer) && gameMode != string(domain.GameModePlayer) {
		return datasource.Game{}, fmt.Errorf("%w: games.mode has invalid value %q", ErrInvalidDatabaseRow, gameMode)
	}
	gameState, err := requiredString("games.state", state)
	if err != nil {
		return datasource.Game{}, err
	}
	if gameState != string(domain.GameStateWaitingPlayers) &&
		gameState != string(domain.GameStatePlayerToMove) &&
		gameState != string(domain.GameStateDraw) &&
		gameState != string(domain.GameStatePlayerWins) {
		return datasource.Game{}, fmt.Errorf("%w: games.state has invalid value %q", ErrInvalidDatabaseRow, gameState)
	}
	gameCreatedAt, err := requiredTime("games.created_at", createdAt)
	if err != nil {
		return datasource.Game{}, err
	}
	if len(rawField) == 0 {
		return datasource.Game{}, fmt.Errorf("%w: games.field is required", ErrInvalidDatabaseRow)
	}

	var field datasource.Field
	if err := json.Unmarshal(rawField, &field); err != nil {
		logRepository("game unmarshal failed uuid=%q: %v", gameUUID, err)
		return datasource.Game{}, err
	}
	if err := validateDatasourceField(field); err != nil {
		return datasource.Game{}, err
	}

	return datasource.Game{
		UUID:           gameUUID,
		Field:          field,
		Mode:           gameMode,
		State:          gameState,
		CreatedAt:      gameCreatedAt,
		NextPlayerUUID: optionalString(nextPlayerUUID),
		WinnerUUID:     optionalString(winnerUUID),
		PlayerXUUID:    optionalString(playerXUUID),
		PlayerOUUID:    optionalString(playerOUUID),
	}, nil
}

func validateDatasourceField(field datasource.Field) error {
	if len(field) != domain.BoardSize {
		return fmt.Errorf("%w: games.field must have %d rows", ErrInvalidDatabaseRow, domain.BoardSize)
	}
	for row := range field {
		if len(field[row]) != domain.BoardSize {
			return fmt.Errorf("%w: games.field row %d must have %d cells", ErrInvalidDatabaseRow, row, domain.BoardSize)
		}
		for column, cell := range field[row] {
			if cell != domain.CellEmpty && cell != domain.CellX && cell != domain.CellO {
				return fmt.Errorf("%w: games.field cell [%d][%d] has invalid value %d", ErrInvalidDatabaseRow, row, column, cell)
			}
		}
	}
	return nil
}
