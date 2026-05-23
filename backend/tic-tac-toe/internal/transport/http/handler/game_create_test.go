package handler

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"tic-tac-toe/app/application"
	"tic-tac-toe/app/domain"
)

func TestGameHandlerCreateGame(t *testing.T) {
	t.Run("creates computer game with empty body", func(t *testing.T) {
		storage := &storageStub{}
		handler := newGameHandler(&logicStub{}, storage)
		req := authenticatedRequest(http.MethodPost, "/games", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", rec.Code)
		}
		if storage.savedGame.Mode != domain.GameModeComputer || storage.savedGame.PlayerX.String() != testUserUUID {
			t.Fatalf("unexpected saved game: %#v", storage.savedGame)
		}
	})

	t.Run("rejects unauthenticated request", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := httptest.NewRequest(http.MethodPost, "/games", nil)
		rec := httptest.NewRecorder()

		handler.CreateGame(rec, req)

		assertStatusAndMessage(t, rec, http.StatusUnauthorized, "unauthorized")
	})

	t.Run("rejects invalid body", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := authenticatedRequest(http.MethodPost, "/games", bytes.NewBufferString(`{`))
		rec := httptest.NewRecorder()

		handler.CreateGame(rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, "invalid request body")
	})

	t.Run("rejects unsupported content type", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := authenticatedRequest(http.MethodPost, "/games", bytes.NewBufferString(`{"mode":"computer"}`))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()

		handler.CreateGame(rec, req)

		assertStatusAndMessage(t, rec, http.StatusUnsupportedMediaType, "content type must be application/json")
	})

	t.Run("handles save error", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{saveErr: errors.New("save failed")})
		req := authenticatedRequest(http.MethodPost, "/games", nil)
		rec := httptest.NewRecorder()

		handler.CreateGame(rec, req)

		assertStatusAndMessage(t, rec, http.StatusInternalServerError, "failed to save current game")
	})

	t.Run("rejects invalid mode", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := authenticatedRequest(http.MethodPost, "/games", marshalBody(t, map[string]any{"mode": "bad"}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, application.ErrInvalidGameMode.Error())
	})
}
