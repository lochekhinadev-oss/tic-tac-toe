package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/repository"
)

func TestGameHandlerListGames(t *testing.T) {
	t.Run("lists games", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{game: domain.Game{UUID: testUUID, Field: emptyDomainField()}})
		req := authenticatedRequest(http.MethodGet, "/games", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("returns storage error", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{getErr: errors.New("load failed")})
		req := authenticatedRequest(http.MethodGet, "/games", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusInternalServerError, "failed to load current games")
	})
}

func TestGameHandlerListCompletedGames(t *testing.T) {
	t.Run("lists completed games", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{history: []domain.Game{{
			UUID:       testUUID,
			Field:      emptyDomainField(),
			State:      domain.GameStatePlayerWins,
			WinnerUUID: "user-1",
		}}})
		req := authenticatedRequest(http.MethodGet, "/games/history", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var payload struct {
			Games []struct {
				UUID string `json:"uuid"`
			} `json:"games"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(payload.Games) != 1 || payload.Games[0].UUID != testUUID {
			t.Fatalf("unexpected history payload: %#v", payload)
		}
	})

	t.Run("returns storage error", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{getErr: errors.New("load failed")})
		req := authenticatedRequest(http.MethodGet, "/games/history", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusInternalServerError, "failed to load completed games")
	})

	t.Run("rejects unauthenticated request", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := httptest.NewRequest(http.MethodGet, "/games/history", nil)
		rec := httptest.NewRecorder()

		handler.ListCompletedGames(rec, req)

		assertStatusAndMessage(t, rec, http.StatusUnauthorized, "unauthorized")
	})
}

func TestGameHandlerListTopPlayers(t *testing.T) {
	t.Run("lists top players", func(t *testing.T) {
		storage := &storageStub{top: []domain.WonGameInfo{{UserUUID: testUUID, Login: "player", WinRatio: 2.5}}}
		handler := newGameHandler(&logicStub{}, storage)
		req := authenticatedRequest(http.MethodGet, "/games/leaderboard?n=3", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		if storage.lastLimit != 3 {
			t.Fatalf("expected limit 3, got %d", storage.lastLimit)
		}
		var payload struct {
			Players []struct {
				UUID     string  `json:"uuid"`
				Login    string  `json:"login"`
				WinRatio float64 `json:"winRatio"`
			} `json:"players"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(payload.Players) != 1 || payload.Players[0].UUID != testUUID || payload.Players[0].Login != "player" || payload.Players[0].WinRatio != 2.5 {
			t.Fatalf("unexpected leaderboard payload: %#v", payload)
		}
	})

	t.Run("rejects invalid limit", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := authenticatedRequest(http.MethodGet, "/games/leaderboard?n=0", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, "n must be an integer between 1 and 100")
	})

	t.Run("rejects injection-like limit", func(t *testing.T) {
		storage := &storageStub{}
		handler := newGameHandler(&logicStub{}, storage)
		req := authenticatedRequest(http.MethodGet, "/games/leaderboard?n=1%20OR%201=1", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, "n must be an integer between 1 and 100")
		if storage.lastLimit != 0 {
			t.Fatalf("expected storage not to be called, got limit %d", storage.lastLimit)
		}
	})

	t.Run("rejects sql statement limit", func(t *testing.T) {
		storage := &storageStub{}
		handler := newGameHandler(&logicStub{}, storage)
		req := authenticatedRequest(http.MethodGet, "/games/leaderboard?n=100%3BDROP%20TABLE%20users", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, "n must be an integer between 1 and 100")
		if storage.lastLimit != 0 {
			t.Fatalf("expected storage not to be called, got limit %d", storage.lastLimit)
		}
	})

	t.Run("rejects too large limit", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := authenticatedRequest(http.MethodGet, "/games/leaderboard?n=101", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, "n must be an integer between 1 and 100")
	})

	t.Run("returns storage error", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{getErr: errors.New("load failed")})
		req := authenticatedRequest(http.MethodGet, "/games/leaderboard?n=5", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusInternalServerError, "failed to load leaderboard")
	})
}

func TestGameHandlerGetGame(t *testing.T) {
	t.Run("gets game", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{game: domain.Game{UUID: testUUID, Field: emptyDomainField()}})
		req := authenticatedRequest(http.MethodGet, "/games/"+testUUID, nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{getErr: repository.ErrGameNotFound})
		req := authenticatedRequest(http.MethodGet, "/games/"+testUUID, nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusNotFound, "game not found")
	})

	t.Run("rejects invalid uuid", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := authenticatedRequest(http.MethodGet, "/games/not-a-uuid", nil)
		rec := httptest.NewRecorder()

		handler.GetGame(rec, req, "not-a-uuid")

		assertStatusAndMessage(t, rec, http.StatusBadRequest, "invalid uuid")
	})
}
