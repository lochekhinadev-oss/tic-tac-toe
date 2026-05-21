package application

import (
	"errors"
	"testing"
	"time"

	"tic-tac-toe/app/domain"
)

func TestCreateAndJoinGame(t *testing.T) {
	service := NewGameService()

	computerGame, err := service.CreateGame("game-1", "user-x", domain.GameModeComputer)
	if err != nil {
		t.Fatalf("unexpected create computer error: %v", err)
	}
	if computerGame.State != domain.GameStatePlayerToMove || computerGame.PlayerOUUID != domain.ComputerPlayerUUID {
		t.Fatalf("unexpected computer game: %#v", computerGame)
	}

	playerGame, err := service.CreateGame("game-2", "user-x", domain.GameModePlayer)
	if err != nil {
		t.Fatalf("unexpected create player error: %v", err)
	}
	if playerGame.State != domain.GameStateWaitingPlayers || playerGame.PlayerXUUID != "user-x" {
		t.Fatalf("unexpected player game: %#v", playerGame)
	}

	joined, err := service.JoinGame(playerGame, "user-o")
	if err != nil {
		t.Fatalf("unexpected join error: %v", err)
	}
	if joined.PlayerOUUID != "user-o" || joined.NextPlayerUUID != "user-x" {
		t.Fatalf("unexpected joined game: %#v", joined)
	}
}

func TestCreateAndJoinGameRejectInvalidInputs(t *testing.T) {
	service := NewGameService()

	if _, err := service.CreateGame("", "user-x", domain.GameModeComputer); !errors.Is(err, ErrInvalidUUID) {
		t.Fatalf("expected ErrInvalidUUID, got %v", err)
	}
	if _, err := service.CreateGame("game-1", "", domain.GameModeComputer); !errors.Is(err, ErrInvalidUUID) {
		t.Fatalf("expected ErrInvalidUUID, got %v", err)
	}
	if _, err := service.CreateGame("game-1", "user-x", "unknown"); !errors.Is(err, ErrInvalidGameMode) {
		t.Fatalf("expected ErrInvalidGameMode, got %v", err)
	}
	if _, err := service.JoinGame(domain.Game{}, "user-o"); !errors.Is(err, ErrInvalidUUID) {
		t.Fatalf("expected ErrInvalidUUID, got %v", err)
	}
	if _, err := service.JoinGame(domain.Game{UUID: "game-1"}, ""); !errors.Is(err, ErrInvalidUUID) {
		t.Fatalf("expected ErrInvalidUUID, got %v", err)
	}
	if _, err := service.JoinGame(domain.Game{
		UUID:        "game-1",
		Mode:        domain.GameModePlayer,
		State:       domain.GameStateWaitingPlayers,
		PlayerXUUID: "user-x",
	}, "user-x"); !errors.Is(err, ErrGameNotJoinable) {
		t.Fatalf("expected ErrGameNotJoinable, got %v", err)
	}
	if _, err := service.JoinGame(domain.Game{
		UUID:        "game-1",
		Mode:        domain.GameModeComputer,
		State:       domain.GameStatePlayerToMove,
		PlayerXUUID: "user-x",
		PlayerOUUID: domain.ComputerPlayerUUID,
	}, "user-o"); !errors.Is(err, ErrGameNotJoinable) {
		t.Fatalf("expected ErrGameNotJoinable, got %v", err)
	}
}

func TestCreateGameUsesConfiguredClock(t *testing.T) {
	now := time.Date(2026, time.May, 16, 12, 30, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	service := NewGameServiceWithClock(func() time.Time { return now })

	game, err := service.CreateGame("game-1", "user-x", domain.GameModeComputer)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	if !game.CreatedAt.Equal(now.UTC()) {
		t.Fatalf("expected created at %s, got %s", now.UTC(), game.CreatedAt)
	}
}

func TestNewGameServiceWithNilClockUsesCurrentTime(t *testing.T) {
	service := NewGameServiceWithClock(nil)
	game, err := service.CreateGame("game-1", "user-x", domain.GameModeComputer)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if game.CreatedAt.IsZero() {
		t.Fatal("expected created at to be set")
	}
}
