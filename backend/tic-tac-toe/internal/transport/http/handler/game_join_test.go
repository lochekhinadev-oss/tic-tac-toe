package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/application"
	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/repository"
)

func TestGameHandlerJoinGame(t *testing.T) {
	t.Run("joins waiting player game", func(t *testing.T) {
		game := domain.Game{
			UUID:    testUUID,
			Field:   emptyDomainField(),
			Mode:    domain.GameModePlayer,
			State:   domain.GameStateWaitingPlayers,
			PlayerX: domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174002")),
		}
		handler := newGameHandler(&logicStub{}, &storageStub{game: game})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/join", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{getErr: repository.ErrGameNotFound})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/join", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusNotFound, "game not found")
	})

	t.Run("not joinable", func(t *testing.T) {
		handler := newGameHandler(&logicStub{}, &storageStub{game: domain.Game{
			UUID:    testUUID,
			Field:   emptyDomainField(),
			Mode:    domain.GameModeComputer,
			State:   domain.GameStatePlayerToMove,
			PlayerX: domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174002")),
		}})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/join", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusBadRequest, application.ErrGameNotJoinable.Error())
	})

	t.Run("save error", func(t *testing.T) {
		game := domain.Game{
			UUID:    testUUID,
			Field:   emptyDomainField(),
			Mode:    domain.GameModePlayer,
			State:   domain.GameStateWaitingPlayers,
			PlayerX: domain.NewUserPlayerRef(googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174002")),
		}
		handler := newGameHandler(&logicStub{}, &storageStub{game: game, saveErr: errors.New("save failed")})
		req := authenticatedRequest(http.MethodPost, "/games/"+testUUID+"/join", nil)
		rec := httptest.NewRecorder()

		serveGameHandler(handler, rec, req)

		assertStatusAndMessage(t, rec, http.StatusInternalServerError, "failed to save current game")
	})
}
