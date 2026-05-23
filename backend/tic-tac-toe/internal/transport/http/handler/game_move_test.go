package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/repository"
)

func TestGameHandlerMakeMove(t *testing.T) {
	t.Run("rejects unsupported method", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID, nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusMethodNotAllowed, "method not allowed")
	})

	t.Run("rejects invalid uuid", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := authenticatedRequest(http.MethodPost, "/games/not-a-uuid/move", bytes.NewBufferString(`{}`))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, "invalid uuid")
	})

	t.Run("rejects invalid body", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", bytes.NewBufferString(`{`))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, "invalid request body")
	})

	t.Run("rejects unknown fields", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{
			"field": emptyField(),
			"extra": "value",
		}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, "invalid request body")
	})

	t.Run("rejects uuid mismatch", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{
			"uuid":  "123e4567-e89b-42d3-a456-426614174001",
			"field": emptyField(),
		}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, "uuid in path does not match request body")
	})

	t.Run("returns internal error when storage load fails unexpectedly", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{getErr: errors.New("boom")})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{
			"field": emptyField(),
		}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusInternalServerError, "failed to load current game")
	})

	t.Run("returns validation error", func(t *testing.T) {
		handler := newGameHandler(&logicStub{validateErr: errors.New("bad move")}, &storageStub{
			game: domain.Game{UUID: testUUID, Field: emptyDomainField()},
		})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{
			"field": emptyField(),
		}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, "bad move")
	})

	t.Run("returns finished game and saves it", func(t *testing.T) {
		storage := &storageStub{game: domain.Game{UUID: testUUID, Field: emptyDomainField()}}
		handler := newGameHandler(&logicStub{}, storage)
		field := emptyField()
		field[0][0] = domain.CellUser
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{
			"field": field,
		}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		if storage.savedGame.UUID != testUUID || storage.savedGame.Field[0][0] != domain.CellUser {
			t.Fatalf("unexpected saved game: %#v", storage.savedGame)
		}
	})

	t.Run("returns error when save finished game fails", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{
			game:    domain.Game{UUID: testUUID, Field: emptyDomainField()},
			saveErr: errors.New("save failed"),
		})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{
			"field": emptyFieldWithUserMove(),
		}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusInternalServerError, "failed to save current game")
	})

	t.Run("returns conflict when game changed concurrently", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{
			game:    domain.Game{UUID: testUUID, Field: emptyDomainField()},
			saveErr: domain.ErrGameConflict,
		})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{
			"field": emptyFieldWithUserMove(),
		}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusConflict, "game was changed by another request")
	})

	t.Run("returns next move error", func(t *testing.T) {
		handler := newGameHandler(&logicStub{nextMoveErr: errors.New("next move failed")}, &storageStub{
			game: domain.Game{UUID: testUUID, Field: emptyDomainField()},
		})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{
			"field": emptyFieldWithUserMove(),
		}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, "next move failed")
	})

	t.Run("returns error when saving next move fails", func(t *testing.T) {
		handler := newGameHandler(&logicStub{
			nextGame: domain.Game{UUID: testUUID, Field: emptyDomainFieldWithUserMove()},
		}, &storageStub{
			game:    domain.Game{UUID: testUUID, Field: emptyDomainField()},
			saveErr: errors.New("save failed"),
		})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{
			"field": emptyFieldWithUserMove(),
		}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusInternalServerError, "failed to save current game")
	})

	t.Run("returns next game", func(t *testing.T) {
		next := domain.Game{
			UUID: testUUID,
			Field: domain.Field{
				{domain.CellUser, domain.CellComputer, domain.CellEmpty},
				{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
				{domain.CellEmpty, domain.CellEmpty, domain.CellEmpty},
			},
		}
		handler := newGameHandler(&logicStub{nextGame: next}, &storageStub{
			game: domain.Game{UUID: testUUID, Field: emptyDomainField()},
		})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{
			"field": emptyFieldWithUserMove(),
		}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}

		var payload struct {
			UUID  googleuuid.UUID `json:"uuid"`
			Field [][]int         `json:"field"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if payload.UUID.String() != testUUID || payload.Field[0][1] != domain.CellComputer {
			t.Fatalf("unexpected payload: %#v", payload)
		}
	})

	t.Run("makes move", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{game: domain.Game{
			UUID:       testUUID,
			Field:      emptyDomainField(),
			State:      domain.GameStatePlayerToMove,
			NextPlayer: domain.NewUserPlayerRef(googleuuid.MustParse(testUserUUID)),
			PlayerX:    domain.NewUserPlayerRef(googleuuid.MustParse(testUserUUID)),
			PlayerO:    domain.NewComputerPlayerRef(),
			Mode:       domain.GameModeComputer,
		}})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{
			"field": emptyFieldWithUserMove(),
		}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{getErr: repository.ErrGameNotFound})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{
			"field": emptyFieldWithUserMove(),
		}))
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusNotFound, "game not found")
	})

	t.Run("rejects unauthenticated request", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{})
		req := httptest.NewRequest(http.MethodPost, "/games/"+testUUID+"/move", marshalBody(t, map[string]any{"field": emptyField()}))
		rec := httptest.NewRecorder()

		handler.MakeMove(rec, req, googleuuid.MustParse(testUUID))

		assertStatusAndMessage(t, rec, http.StatusUnauthorized, "unauthorized")
	})
}
