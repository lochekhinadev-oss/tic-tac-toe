package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"tic-tac-toe/app/domain"
)

func TestBuildDatasourceUserRejectsInvalidRows(t *testing.T) {
	_, err := buildDatasourceUser(sql.NullString{}, sql.NullString{String: "player", Valid: true}, sql.NullString{String: "hash", Valid: true})
	if !errors.Is(err, ErrInvalidDatabaseRow) {
		t.Fatalf("expected ErrInvalidDatabaseRow, got %v", err)
	}
}

func TestBuildRefreshSessionRejectsInvalidRows(t *testing.T) {
	now := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)

	_, err := buildRefreshSession(
		sql.NullString{String: "hash", Valid: true},
		sql.NullString{String: "user-1", Valid: true},
		sql.NullTime{Time: now, Valid: true},
		sql.NullTime{},
		sql.NullTime{Time: now, Valid: true},
	)
	if !errors.Is(err, ErrInvalidDatabaseRow) {
		t.Fatalf("expected ErrInvalidDatabaseRow, got %v", err)
	}
}

func TestBuildDatasourceGameRejectsInvalidRows(t *testing.T) {
	field, err := json.Marshal(domain.Field{
		{domain.CellX, domain.CellEmpty, domain.CellEmpty},
		{domain.CellEmpty, domain.CellO, domain.CellEmpty},
		{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
	})
	if err != nil {
		t.Fatalf("marshal field: %v", err)
	}

	tests := []struct {
		name  string
		field []byte
		mode  sql.NullString
		state sql.NullString
	}{
		{
			name:  "missing field",
			field: nil,
			mode:  sql.NullString{String: string(domain.GameModeComputer), Valid: true},
			state: sql.NullString{String: string(domain.GameStatePlayerToMove), Valid: true},
		},
		{
			name:  "invalid mode",
			field: field,
			mode:  sql.NullString{String: "bad", Valid: true},
			state: sql.NullString{String: string(domain.GameStatePlayerToMove), Valid: true},
		},
		{
			name:  "invalid state",
			field: field,
			mode:  sql.NullString{String: string(domain.GameModeComputer), Valid: true},
			state: sql.NullString{String: "bad", Valid: true},
		},
		{
			name:  "null uuid",
			field: field,
			mode:  sql.NullString{String: string(domain.GameModeComputer), Valid: true},
			state: sql.NullString{String: string(domain.GameStatePlayerToMove), Valid: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uuid := sql.NullString{String: "game-1", Valid: true}
			if tt.name == "null uuid" {
				uuid = sql.NullString{}
			}

			_, err := buildDatasourceGame(
				uuid,
				tt.field,
				tt.mode,
				tt.state,
				sql.NullTime{Time: time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC), Valid: true},
				sql.NullString{},
				sql.NullString{},
				sql.NullString{String: "user-x", Valid: true},
				sql.NullString{String: "computer", Valid: true},
			)
			if !errors.Is(err, ErrInvalidDatabaseRow) {
				t.Fatalf("expected ErrInvalidDatabaseRow, got %v", err)
			}
		})
	}
}

func TestBuildWonGameInfoRejectsInvalidRows(t *testing.T) {
	_, err := buildWonGameInfo(
		sql.NullString{String: "user-1", Valid: true},
		sql.NullString{},
		sql.NullFloat64{Float64: 1, Valid: true},
	)
	if !errors.Is(err, ErrInvalidDatabaseRow) {
		t.Fatalf("expected ErrInvalidDatabaseRow, got %v", err)
	}
}
