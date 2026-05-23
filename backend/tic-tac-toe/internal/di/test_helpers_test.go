package di

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	googleuuid "github.com/google/uuid"

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

func (authStub) Can(context.Context, googleuuid.UUID, domain.Permission) (bool, error) {
	return true, nil
}

func (authStub) AuthorizeRequest(context.Context, googleuuid.UUID, string, string) (bool, error) {
	return true, nil
}

func (authStub) SignUp(context.Context, authservice.SignUpRequest) (bool, error) {
	return true, nil
}

func (authStub) SignIn(context.Context, authservice.SessionRequest) (authservice.SessionResponse, error) {
	return authservice.SessionResponse{UserUUID: "123e4567-e89b-42d3-a456-426614174000", SessionID: "session-1"}, nil
}

func (authStub) RefreshSession(context.Context, string) (authservice.SessionResponse, error) {
	return authservice.SessionResponse{UserUUID: "123e4567-e89b-42d3-a456-426614174000", SessionID: "session-2"}, nil
}

func (authStub) Logout(context.Context, string) error { return nil }

func (authStub) LogoutAll(context.Context, string) error { return nil }

func (authStub) AuthenticateSession(context.Context, string) (string, error) {
	return "123e4567-e89b-42d3-a456-426614174000", nil
}

type deniedAuthStub struct{}

func (deniedAuthStub) Can(context.Context, googleuuid.UUID, domain.Permission) (bool, error) {
	return false, nil
}

func (deniedAuthStub) AuthorizeRequest(context.Context, googleuuid.UUID, string, string) (bool, error) {
	return false, nil
}

func (deniedAuthStub) SignUp(context.Context, authservice.SignUpRequest) (bool, error) {
	return true, nil
}

func (deniedAuthStub) SignIn(context.Context, authservice.SessionRequest) (authservice.SessionResponse, error) {
	return authservice.SessionResponse{}, nil
}

func (deniedAuthStub) RefreshSession(context.Context, string) (authservice.SessionResponse, error) {
	return authservice.SessionResponse{}, nil
}

func (deniedAuthStub) Logout(context.Context, string) error {
	return authservice.ErrInvalidToken
}

func (deniedAuthStub) LogoutAll(context.Context, string) error {
	return authservice.ErrInvalidToken
}

func (deniedAuthStub) AuthenticateSession(context.Context, string) (string, error) {
	return "", authservice.ErrInvalidToken
}

type gameLogicStub struct{}

func (gameLogicStub) CreateGame(uuid googleuuid.UUID, creatorUUID googleuuid.UUID, mode domain.GameMode) (domain.Game, error) {
	return domain.Game{
		UUID:       uuid.String(),
		Field:      domain.Field{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}},
		Mode:       mode,
		State:      domain.GameStatePlayerToMove,
		NextPlayer: domain.NewUserPlayerRef(creatorUUID),
		PlayerX:    domain.NewUserPlayerRef(creatorUUID),
		PlayerO:    domain.NewComputerPlayerRef(),
	}, nil
}

func (gameLogicStub) JoinGame(game domain.Game, userUUID googleuuid.UUID) (domain.Game, error) {
	game.PlayerO = domain.NewUserPlayerRef(userUUID)
	game.State = domain.GameStatePlayerToMove
	return game, nil
}

func (gameLogicStub) ApplyMove(previous domain.Game, current domain.Game, _ googleuuid.UUID) (domain.Game, error) {
	previous.Field = current.Field
	return previous, nil
}

type gameStorageStub struct{}

func (gameStorageStub) SaveGame(context.Context, domain.Game) error { return nil }

func (gameStorageStub) SaveGameIfUnchanged(context.Context, domain.Game, domain.Game) error {
	return nil
}

func (gameStorageStub) GetGame(_ context.Context, uuid googleuuid.UUID) (domain.Game, error) {
	return domain.Game{
		UUID:       uuid.String(),
		Field:      domain.Field{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}},
		Mode:       domain.GameModePlayer,
		State:      domain.GameStateWaitingPlayers,
		NextPlayer: domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174000")),
		PlayerX:    domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")),
	}, nil
}

func (gameStorageStub) ListActiveGames(context.Context) ([]domain.Game, error) {
	return []domain.Game{{UUID: "123e4567-e89b-42d3-a456-426614174000", Field: domain.Field{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}}}}, nil
}

func (gameStorageStub) ListCompletedGamesByUserUUID(context.Context, googleuuid.UUID) ([]domain.Game, error) {
	return []domain.Game{{UUID: "123e4567-e89b-42d3-a456-426614174000", Field: domain.Field{{1, 2, 1}, {2, 1, 2}, {2, 1, 1}}, State: domain.GameStatePlayerWins, Winner: domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174000"))}}, nil
}

func (gameStorageStub) ListTopPlayers(context.Context, int) ([]domain.WonGameInfo, error) {
	return []domain.WonGameInfo{{UserUUID: "123e4567-e89b-42d3-a456-426614174000", Login: "player", WinRatio: 1}}, nil
}

func (gameStorageStub) JoinGame(ctx context.Context, uuid googleuuid.UUID, userUUID googleuuid.UUID) (domain.Game, error) {
	game, _ := gameStorageStub{}.GetGame(ctx, uuid)
	game.PlayerO = domain.NewUserPlayerRef(userUUID)
	game.State = domain.GameStatePlayerToMove
	return game, nil
}

type userServiceStub struct{}

func (userServiceStub) CreateUser(context.Context, domain.User) error { return nil }

func (userServiceStub) GetUserByLogin(context.Context, string) (domain.User, error) {
	return domain.User{UUID: "123e4567-e89b-42d3-a456-426614174000", Login: "player"}, nil
}

func (userServiceStub) GetUserByUUID(context.Context, googleuuid.UUID) (domain.User, error) {
	return domain.User{UUID: "123e4567-e89b-42d3-a456-426614174000", Login: "player"}, nil
}

func (userServiceStub) UpdatePassword(context.Context, googleuuid.UUID, string) error { return nil }

func (userServiceStub) DeleteUser(context.Context, googleuuid.UUID) error { return nil }

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
