package di

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"tic-tac-toe/app/domain"
	authservice "tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/internal/transport/http/handler"
	"tic-tac-toe/internal/transport/http/middleware"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("expected non-nil app")
	}
}

func TestAppModule(t *testing.T) {
	if err := fx.ValidateApp(AppModule); err != nil {
		t.Fatalf("invalid fx app graph: %v", err)
	}
}

func TestNewRouter(t *testing.T) {
	gameHandler := handler.NewGameHandler(gameLogicStub{}, gameStorageStub{})
	authHandler := handler.NewAuthHandler(authStub{})
	userHandler := handler.NewUserHandler(userServiceStub{})
	db := &databaseStub{}
	router := NewRouter(gameHandler, authHandler, userHandler, middleware.NewUserAuthenticator(authStub{}), db)

	tests := []struct {
		method string
		path   string
		body   string
		status int
	}{
		{method: http.MethodGet, path: "/healthz", status: http.StatusOK},
		{method: http.MethodGet, path: "/readyz", status: http.StatusOK},
		{method: http.MethodGet, path: "/metrics", status: http.StatusOK},
		{method: http.MethodGet, path: "/swagger", status: http.StatusOK},
		{method: http.MethodGet, path: "/openapi.yaml", status: http.StatusOK},
		{method: http.MethodGet, path: "/swagger/doc.json", status: http.StatusOK},
		{method: http.MethodPost, path: "/users", body: `{"login":"player","password":"secret"}`, status: http.StatusCreated},
		{method: http.MethodPost, path: "/auth/sessions", body: `{"login":"player","password":"secret"}`, status: http.StatusOK},
		{method: http.MethodPost, path: "/auth/tokens/access", body: `{"refreshToken":"refresh"}`, status: http.StatusOK},
		{method: http.MethodPost, path: "/auth/tokens/refresh", body: `{"refreshToken":"refresh"}`, status: http.StatusOK},
		{method: http.MethodDelete, path: "/auth/sessions/current", body: `{"refreshToken":"refresh"}`, status: http.StatusNoContent},
		{method: http.MethodDelete, path: "/auth/sessions", body: `{"refreshToken":"refresh"}`, status: http.StatusNoContent},
		{method: http.MethodPost, path: "/signup", body: `{"login":"player","password":"secret"}`, status: http.StatusCreated},
		{method: http.MethodPost, path: "/auth", body: `{"login":"player","password":"secret"}`, status: http.StatusOK},
		{method: http.MethodPost, path: "/auth/refresh/access", body: `{"refreshToken":"refresh"}`, status: http.StatusOK},
		{method: http.MethodPost, path: "/auth/refresh", body: `{"refreshToken":"refresh"}`, status: http.StatusOK},
		{method: http.MethodPost, path: "/auth/logout", body: `{"refreshToken":"refresh"}`, status: http.StatusNoContent},
		{method: http.MethodPost, path: "/auth/logout/all", body: `{"refreshToken":"refresh"}`, status: http.StatusNoContent},
		{method: http.MethodGet, path: "/auth", status: http.StatusMethodNotAllowed},
		{method: http.MethodGet, path: "/missing", status: http.StatusNotFound},
		{method: http.MethodPost, path: "/games", status: http.StatusCreated},
		{method: http.MethodGet, path: "/games", status: http.StatusOK},
		{method: http.MethodGet, path: "/games/history", status: http.StatusOK},
		{method: http.MethodGet, path: "/games/leaderboard?n=10", status: http.StatusOK},
		{method: http.MethodGet, path: "/games/123e4567-e89b-42d3-a456-426614174000", status: http.StatusOK},
		{method: http.MethodPost, path: "/games/123e4567-e89b-42d3-a456-426614174000/join", status: http.StatusOK},
		{method: http.MethodPost, path: "/games/123e4567-e89b-42d3-a456-426614174000/move", body: `{"field":[[1,0,0],[0,0,0],[0,0,0]]}`, status: http.StatusOK},
		{method: http.MethodGet, path: "/users/123e4567-e89b-42d3-a456-426614174000", status: http.StatusOK},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
		if tt.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		router.ServeHTTP(rec, req)
		if rec.Code != tt.status {
			t.Fatalf("%s %s expected status %d, got %d body=%s", tt.method, tt.path, tt.status, rec.Code, rec.Body.String())
		}
	}
}

func TestRegisterDatabaseLifecycle(t *testing.T) {
	lifecycle := fxtest.NewLifecycle(t)
	db := &databaseStub{}

	RegisterDatabaseLifecycle(lifecycle, db)

	if err := lifecycle.Start(context.Background()); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}
	if err := lifecycle.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
	if !db.closed {
		t.Fatal("expected database to be closed")
	}
}

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

func (userServiceStub) VerifyPassword(domain.User, string) (bool, bool) { return true, false }

func TestNewHTTPServer(t *testing.T) {
	router := chi.NewRouter()
	server := NewHTTPServer(router, HTTPConfig{Addr: ":8080"})

	if server == nil {
		t.Fatal("expected non-nil server")
	}

	if server.Addr != ":8080" {
		t.Fatalf("expected :8080, got %q", server.Addr)
	}

	if server.Handler != router {
		t.Fatal("expected server to keep provided router")
	}

	if server.ReadHeaderTimeout <= 0 {
		t.Fatal("expected positive read header timeout")
	}
}

func TestRegisterHTTPServer(t *testing.T) {
	lifecycle := fxtest.NewLifecycle(t)
	server := &http.Server{
		Addr:    "127.0.0.1:0",
		Handler: http.NewServeMux(),
	}

	RegisterHTTPServer(lifecycle, server)

	if err := lifecycle.Start(context.Background()); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	if err := lifecycle.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected stop error: %v", err)
	}
}

func TestRegisterHTTPServerReturnsListenError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen test port: %v", err)
	}
	defer listener.Close()

	lifecycle := fxtest.NewLifecycle(t)
	server := &http.Server{
		Addr:    listener.Addr().String(),
		Handler: http.NewServeMux(),
	}

	RegisterHTTPServer(lifecycle, server)

	if err := lifecycle.Start(context.Background()); err == nil {
		t.Fatal("expected listen error")
	}
}
