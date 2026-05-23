package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/application"
	"tic-tac-toe/app/domain"
	"tic-tac-toe/internal/transport/http/middleware"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

const testUUID = "123e4567-e89b-42d3-a456-426614174000"
const testUserUUID = "123e4567-e89b-42d3-a456-426614174001"

type logicStub struct {
	nextGame    domain.Game
	nextMoveErr error
	validateErr error
}

func (l *logicStub) CreateGame(uuid googleuuid.UUID, creatorUUID googleuuid.UUID, mode domain.GameMode) (domain.Game, error) {
	if mode != domain.GameModeComputer && mode != domain.GameModePlayer {
		return domain.Game{}, application.ErrInvalidGameMode
	}
	return domain.Game{
		UUID:       uuid.String(),
		Field:      emptyDomainField(),
		Mode:       mode,
		State:      domain.GameStatePlayerToMove,
		NextPlayer: domain.NewUserPlayerRef(creatorUUID),
		PlayerX:    domain.NewUserPlayerRef(creatorUUID),
		PlayerO:    domain.NewComputerPlayerRef(),
	}, nil
}

func (l *logicStub) JoinGame(game domain.Game, userUUID googleuuid.UUID) (domain.Game, error) {
	if game.Mode != domain.GameModePlayer || game.State != domain.GameStateWaitingPlayers || game.PlayerX.Matches(userUUID) {
		return domain.Game{}, application.ErrGameNotJoinable
	}
	game.PlayerO = domain.NewUserPlayerRef(userUUID)
	game.State = domain.GameStatePlayerToMove
	return game, nil
}

func (l *logicStub) ApplyMove(previous domain.Game, current domain.Game, userUUID googleuuid.UUID) (domain.Game, error) {
	if l.validateErr != nil {
		return domain.Game{}, l.validateErr
	}
	if l.nextGame.UUID == "" {
		return current, l.nextMoveErr
	}
	return l.nextGame, l.nextMoveErr
}

type storageStub struct {
	game      domain.Game
	history   []domain.Game
	top       []domain.WonGameInfo
	getErr    error
	saveErr   error
	savedGame domain.Game
	lastLimit int
}

func (s *storageStub) SaveGame(_ context.Context, game domain.Game) error {
	s.savedGame = game
	return s.saveErr
}

func (s *storageStub) SaveGameIfUnchanged(_ context.Context, _ domain.Game, game domain.Game) error {
	s.savedGame = game
	return s.saveErr
}

func (s *storageStub) GetGame(_ context.Context, uuid googleuuid.UUID) (domain.Game, error) {
	return s.game, s.getErr
}

func (s *storageStub) ListActiveGames(context.Context) ([]domain.Game, error) {
	return []domain.Game{s.game}, s.getErr
}

func (s *storageStub) ListCompletedGamesByUserUUID(context.Context, googleuuid.UUID) ([]domain.Game, error) {
	if s.history != nil {
		return s.history, s.getErr
	}
	return []domain.Game{s.game}, s.getErr
}

func (s *storageStub) ListTopPlayers(_ context.Context, limit int) ([]domain.WonGameInfo, error) {
	s.lastLimit = limit
	if s.top != nil {
		return s.top, s.getErr
	}
	return []domain.WonGameInfo{{UserUUID: testUUID, Login: "player", WinRatio: 1}}, s.getErr
}

func (s *storageStub) JoinGame(context.Context, googleuuid.UUID, googleuuid.UUID) (domain.Game, error) {
	if s.getErr != nil {
		return domain.Game{}, s.getErr
	}
	return s.game, s.saveErr
}

var _ GameStorage = (*storageStub)(nil)

func TestNewGameHandler(t *testing.T) {
	storage := &storageStub{}
	handler := NewGameHandler(&logicStub{}, storage, storage)
	if handler == nil {
		t.Fatal("expected handler")
	}
}

func assertStatusAndMessage(t *testing.T, rec *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()

	if rec.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, rec.Code)
	}

	var payload struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Message != expectedMessage {
		t.Fatalf("expected message %q, got %q", expectedMessage, payload.Message)
	}
}

func marshalBody(t *testing.T, payload any) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	return &buf
}

func authenticatedRequest(method string, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req.WithContext(middleware.WithUserUUID(req.Context(), testUserUUID))
}

func serveGameHandler(handler *GameHandler, rec *httptest.ResponseRecorder, req *http.Request) {
	newGameRouter(handler).ServeHTTP(rec, req)
}

func newGameRouter(handler *GameHandler) chi.Router {
	router := chi.NewRouter()
	router.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		webresponse.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	router.Route("/games", func(games chi.Router) {
		games.Post("/", handler.CreateGame)
		games.Get("/", handler.ListGames)
		games.Get("/history", handler.ListCompletedGames)
		games.Get("/leaderboard", handler.ListTopPlayers)
		games.Route("/{uuid}", func(game chi.Router) {
			game.Get("/", withRouteUUID(handler.GetGame))
			game.Post("/join", withRouteUUID(handler.JoinGame))
			game.Post("/move", withRouteUUID(handler.MakeMove))
		})
	})

	return router
}

func withRouteUUID(next func(http.ResponseWriter, *http.Request, googleuuid.UUID)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid, err := googleuuid.Parse(chi.URLParam(r, "uuid"))
		if err != nil {
			webresponse.WriteBadRequest(w, "invalid uuid")
			return
		}
		next(w, r, uuid)
	}
}

func emptyField() [][]int {
	return [][]int{
		{0, 0, 0},
		{0, 0, 0},
		{0, 0, 0},
	}
}

func emptyFieldWithUserMove() [][]int {
	field := emptyField()
	field[0][0] = domain.CellUser
	return field
}

func emptyDomainField() domain.Field {
	return domain.Field{
		{0, 0, 0},
		{0, 0, 0},
		{0, 0, 0},
	}
}

func emptyDomainFieldWithUserMove() domain.Field {
	field := emptyDomainField()
	field[0][0] = domain.CellUser
	return field
}
