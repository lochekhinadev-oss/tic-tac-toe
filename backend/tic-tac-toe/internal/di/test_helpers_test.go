package di

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"tic-tac-toe/app/domain"
	authservice "tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/infrastructure/postgres/datasource"
)

type databaseStub struct {
	datasource.Database
	closed  bool
	pingErr error
}

func (d *databaseStub) Close() {
	d.closed = true
}

func (d *databaseStub) Ping(context.Context) error {
	return d.pingErr
}

type authStub struct{}

func (authStub) SignUp(context.Context, authservice.SignUpRequest) (bool, error) {
	return true, nil
}

func (authStub) Authenticate(context.Context, authservice.JwtRequest) (authservice.JwtResponse, error) {
	return authservice.JwtResponse{Type: "Bearer", AccessToken: "access", RefreshToken: "refresh"}, nil
}

func (authStub) RefreshAccessToken(context.Context, string) (authservice.JwtResponse, error) {
	return authservice.JwtResponse{Type: "Bearer", AccessToken: "access-2", RefreshToken: "refresh"}, nil
}

func (authStub) RefreshRefreshToken(context.Context, string) (authservice.JwtResponse, error) {
	return authservice.JwtResponse{Type: "Bearer", AccessToken: "access-2", RefreshToken: "refresh-2"}, nil
}

func (authStub) Logout(context.Context, string) error { return nil }

func (authStub) LogoutAll(context.Context, string) error { return nil }

func (authStub) AuthenticateToken(context.Context, string) (string, error) {
	return "user-1", nil
}

type deniedAuthStub struct{}

func (deniedAuthStub) SignUp(context.Context, authservice.SignUpRequest) (bool, error) {
	return true, nil
}

func (deniedAuthStub) Authenticate(context.Context, authservice.JwtRequest) (authservice.JwtResponse, error) {
	return authservice.JwtResponse{}, nil
}

func (deniedAuthStub) RefreshAccessToken(context.Context, string) (authservice.JwtResponse, error) {
	return authservice.JwtResponse{}, nil
}

func (deniedAuthStub) RefreshRefreshToken(context.Context, string) (authservice.JwtResponse, error) {
	return authservice.JwtResponse{}, nil
}

func (deniedAuthStub) Logout(context.Context, string) error {
	return authservice.ErrInvalidToken
}

func (deniedAuthStub) LogoutAll(context.Context, string) error {
	return authservice.ErrInvalidToken
}

func (deniedAuthStub) AuthenticateToken(context.Context, string) (string, error) {
	return "", authservice.ErrInvalidToken
}

type gameLogicStub struct{}

func (gameLogicStub) CreateGame(uuid string, creatorUUID string, mode domain.GameMode) (domain.Game, error) {
	return domain.Game{
		UUID:           uuid,
		Field:          domain.Field{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}},
		Mode:           mode,
		State:          domain.GameStatePlayerToMove,
		NextPlayerUUID: creatorUUID,
		PlayerXUUID:    creatorUUID,
		PlayerOUUID:    domain.ComputerPlayerUUID,
	}, nil
}

func (gameLogicStub) JoinGame(game domain.Game, userUUID string) (domain.Game, error) {
	game.PlayerOUUID = userUUID
	game.State = domain.GameStatePlayerToMove
	return game, nil
}

func (gameLogicStub) ApplyMove(previous domain.Game, current domain.Game, _ string) (domain.Game, error) {
	previous.Field = current.Field
	return previous, nil
}

type gameStorageStub struct{}

func (gameStorageStub) SaveGame(context.Context, domain.Game) error { return nil }

func (gameStorageStub) SaveGameIfUnchanged(context.Context, domain.Game, domain.Game) error {
	return nil
}

func (gameStorageStub) GetGame(_ context.Context, uuid string) (domain.Game, error) {
	return domain.Game{
		UUID:           uuid,
		Field:          domain.Field{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}},
		Mode:           domain.GameModePlayer,
		State:          domain.GameStateWaitingPlayers,
		NextPlayerUUID: "user-1",
		PlayerXUUID:    "user-x",
	}, nil
}

func (gameStorageStub) ListActiveGames(context.Context) ([]domain.Game, error) {
	return []domain.Game{{UUID: "123e4567-e89b-42d3-a456-426614174000", Field: domain.Field{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}}}}, nil
}

func (gameStorageStub) ListCompletedGamesByUserUUID(context.Context, string) ([]domain.Game, error) {
	return []domain.Game{{UUID: "123e4567-e89b-42d3-a456-426614174000", Field: domain.Field{{1, 2, 1}, {2, 1, 2}, {2, 1, 1}}, State: domain.GameStatePlayerWins, WinnerUUID: "user-1"}}, nil
}

func (gameStorageStub) ListTopPlayers(context.Context, int) ([]domain.WonGameInfo, error) {
	return []domain.WonGameInfo{{UserUUID: "123e4567-e89b-42d3-a456-426614174000", Login: "player", WinRatio: 1}}, nil
}

func (gameStorageStub) JoinGame(ctx context.Context, uuid string, userUUID string) (domain.Game, error) {
	game, _ := gameStorageStub{}.GetGame(ctx, uuid)
	game.PlayerOUUID = userUUID
	game.State = domain.GameStatePlayerToMove
	return game, nil
}

type userServiceStub struct{}

func (userServiceStub) CreateUser(context.Context, domain.User) error { return nil }

func (userServiceStub) GetUserByLogin(context.Context, string) (domain.User, error) {
	return domain.User{UUID: "user-1", Login: "player"}, nil
}

func (userServiceStub) GetUserByUUID(context.Context, string) (domain.User, error) {
	return domain.User{UUID: "123e4567-e89b-42d3-a456-426614174000", Login: "player"}, nil
}

func (userServiceStub) UpdatePassword(context.Context, string, string) error { return nil }

func (userServiceStub) DeleteUser(context.Context, string) error { return nil }

func (userServiceStub) VerifyPassword(domain.User, string) (bool, bool) { return true, false }

func assertResponseMessage(t *testing.T, rec *httptest.ResponseRecorder, message string) {
	t.Helper()

	var payload struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Message != message {
		t.Fatalf("expected message %q, got %q", message, payload.Message)
	}
}

func assertResponseHasKey(t *testing.T, rec *httptest.ResponseRecorder, key string) {
	t.Helper()

	var payload map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := payload[key]; !ok {
		t.Fatalf("expected response key %q, got %#v", key, payload)
	}
}

func assertSecurityHeaders(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	headers := rec.Header()
	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options header, got %q", headers.Get("X-Content-Type-Options"))
	}
	if headers.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("expected X-Frame-Options header, got %q", headers.Get("X-Frame-Options"))
	}
	if headers.Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("expected Referrer-Policy header, got %q", headers.Get("Referrer-Policy"))
	}
}
