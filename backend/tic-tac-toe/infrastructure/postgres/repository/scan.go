package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/infrastructure/postgres/mapper"
	observability "tic-tac-toe/internal/logging"
)

const repositoryLogPrefix = "[infrastructure/postgres/repository]"

var ErrInvalidDatabaseRow = errors.New("invalid database row")

func logRepository(action string, args ...any) {
	fields := append(observability.Fields(), args...)
	slog.Info(repositoryLogPrefix+" "+action, fields...)
}

func requiredString(field string, value sql.NullString) (string, error) {
	if !value.Valid || value.String == "" {
		return "", fmt.Errorf("%w: %s is required", ErrInvalidDatabaseRow, field)
	}
	return value.String, nil
}

func optionalString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func requiredTime(field string, value sql.NullTime) (time.Time, error) {
	if !value.Valid || value.Time.IsZero() {
		return time.Time{}, fmt.Errorf("%w: %s is required", ErrInvalidDatabaseRow, field)
	}
	return value.Time, nil
}

func requiredInt64(field string, value sql.NullInt64) (int64, error) {
	if !value.Valid {
		return 0, fmt.Errorf("%w: %s is required", ErrInvalidDatabaseRow, field)
	}
	return value.Int64, nil
}

func scanGames(rows pgx.Rows, operation string) ([]domain.Game, error) {
	var games []domain.Game
	for rows.Next() {
		game, err := scanGameRow(rows)
		if err != nil {
			logRepository("scan failed", "operation", operation, "error", err)
			return nil, err
		}
		games = append(games, game)
	}

	if err := rows.Err(); err != nil {
		logRepository("rows error", "operation", operation, "error", err)
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
		logRepository("game unmarshal failed", "uuid", gameUUID, "error", err)
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

func scanTopPlayers(rows pgx.Rows, operation string) ([]domain.WonGameInfo, error) {
	var players []domain.WonGameInfo
	for rows.Next() {
		var userUUID sql.NullString
		var login sql.NullString
		var winRatio sql.NullFloat64
		if err := rows.Scan(&userUUID, &login, &winRatio); err != nil {
			logRepository("scan failed", "operation", operation, "error", err)
			return nil, err
		}
		player, err := buildWonGameInfo(userUUID, login, winRatio)
		if err != nil {
			logRepository("invalid row", "operation", operation, "error", err)
			return nil, err
		}
		players = append(players, player)
	}

	if err := rows.Err(); err != nil {
		logRepository("rows error", "operation", operation, "error", err)
		return nil, err
	}

	return players, nil
}

func buildWonGameInfo(userUUID sql.NullString, login sql.NullString, winRatio sql.NullFloat64) (domain.WonGameInfo, error) {
	uuid, err := requiredString("leaderboard.user_uuid", userUUID)
	if err != nil {
		return domain.WonGameInfo{}, err
	}
	userLogin, err := requiredString("leaderboard.login", login)
	if err != nil {
		return domain.WonGameInfo{}, err
	}
	if !winRatio.Valid {
		return domain.WonGameInfo{}, fmt.Errorf("%w: leaderboard.win_ratio is required", ErrInvalidDatabaseRow)
	}
	return domain.WonGameInfo{UserUUID: uuid, Login: userLogin, WinRatio: winRatio.Float64}, nil
}
