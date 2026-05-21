package repository

import (
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"

	"tic-tac-toe/app/domain"
)

func scanTopPlayers(rows pgx.Rows, operation string) ([]domain.WonGameInfo, error) {
	var players []domain.WonGameInfo
	for rows.Next() {
		var userUUID sql.NullString
		var login sql.NullString
		var winRatio sql.NullFloat64
		if err := rows.Scan(&userUUID, &login, &winRatio); err != nil {
			logRepository("%s scan failed: %v", operation, err)
			return nil, err
		}
		player, err := buildWonGameInfo(userUUID, login, winRatio)
		if err != nil {
			logRepository("%s invalid row: %v", operation, err)
			return nil, err
		}
		players = append(players, player)
	}

	if err := rows.Err(); err != nil {
		logRepository("%s rows error: %v", operation, err)
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
