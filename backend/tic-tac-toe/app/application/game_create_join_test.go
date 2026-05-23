package application

import (
	"errors"
	"testing"
	"time"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/domain"
)

func TestCreateAndJoinGame(t *testing.T) {
	service := NewGameService()
	gameUUID1 := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	gameUUID2 := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174002")
	userUUIDX := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174101")
	userUUIDO := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174102")

	computerGame, err := service.CreateGame(gameUUID1, userUUIDX, domain.GameModeComputer)
	if err != nil {
		t.Fatalf("unexpected create computer error: %v", err)
	}
	if computerGame.State != domain.GameStatePlayerToMove || !computerGame.PlayerO.IsComputer() {
		t.Fatalf("unexpected computer game: %#v", computerGame)
	}

	playerGame, err := service.CreateGame(gameUUID2, userUUIDX, domain.GameModePlayer)
	if err != nil {
		t.Fatalf("unexpected create player error: %v", err)
	}
	if playerGame.State != domain.GameStateWaitingPlayers || playerGame.PlayerX.String() != userUUIDX.String() {
		t.Fatalf("unexpected player game: %#v", playerGame)
	}

	joined, err := service.JoinGame(playerGame, userUUIDO)
	if err != nil {
		t.Fatalf("unexpected join error: %v", err)
	}
	if joined.PlayerO.String() != userUUIDO.String() || joined.NextPlayer.String() != userUUIDX.String() {
		t.Fatalf("unexpected joined game: %#v", joined)
	}
}

func TestCreateAndJoinGameRejectInvalidInputs(t *testing.T) {
	service := NewGameService()
	gameUUID1 := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	userUUIDX := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174101")
	userUUIDO := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174102")

	if _, err := service.CreateGame(googleuuid.Nil, userUUIDX, domain.GameModeComputer); !errors.Is(err, ErrInvalidUUID) {
		t.Fatalf("expected ErrInvalidUUID, got %v", err)
	}
	if _, err := service.CreateGame(gameUUID1, googleuuid.Nil, domain.GameModeComputer); !errors.Is(err, ErrInvalidUUID) {
		t.Fatalf("expected ErrInvalidUUID, got %v", err)
	}
	if _, err := service.CreateGame(gameUUID1, userUUIDX, "unknown"); !errors.Is(err, ErrInvalidGameMode) {
		t.Fatalf("expected ErrInvalidGameMode, got %v", err)
	}
	if _, err := service.JoinGame(domain.Game{}, userUUIDO); !errors.Is(err, ErrInvalidUUID) {
		t.Fatalf("expected ErrInvalidUUID, got %v", err)
	}
	if _, err := service.JoinGame(domain.Game{UUID: gameUUID1.String()}, googleuuid.Nil); !errors.Is(err, ErrInvalidUUID) {
		t.Fatalf("expected ErrInvalidUUID, got %v", err)
	}
	if _, err := service.JoinGame(domain.Game{
		UUID:    gameUUID1.String(),
		Mode:    domain.GameModePlayer,
		State:   domain.GameStateWaitingPlayers,
		PlayerX: domain.NewUserPlayerRef(userUUIDX),
	}, userUUIDX); !errors.Is(err, ErrGameNotJoinable) {
		t.Fatalf("expected ErrGameNotJoinable, got %v", err)
	}
	if _, err := service.JoinGame(domain.Game{
		UUID:    gameUUID1.String(),
		Mode:    domain.GameModeComputer,
		State:   domain.GameStatePlayerToMove,
		PlayerX: domain.NewUserPlayerRef(userUUIDX),
		PlayerO: domain.NewComputerPlayerRef(),
	}, userUUIDO); !errors.Is(err, ErrGameNotJoinable) {
		t.Fatalf("expected ErrGameNotJoinable, got %v", err)
	}
}

func TestCreateGameUsesConfiguredClock(t *testing.T) {
	now := time.Date(2026, time.May, 16, 12, 30, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	service := NewGameServiceWithClock(func() time.Time { return now })
	gameUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174101")

	game, err := service.CreateGame(gameUUID, userUUID, domain.GameModeComputer)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	if !game.CreatedAt.Equal(now.UTC()) {
		t.Fatalf("expected created at %s, got %s", now.UTC(), game.CreatedAt)
	}
}

func TestNewGameServiceWithNilClockUsesCurrentTime(t *testing.T) {
	service := NewGameServiceWithClock(nil)
	gameUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174101")
	game, err := service.CreateGame(gameUUID, userUUID, domain.GameModeComputer)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if game.CreatedAt.IsZero() {
		t.Fatal("expected created at to be set")
	}
}
