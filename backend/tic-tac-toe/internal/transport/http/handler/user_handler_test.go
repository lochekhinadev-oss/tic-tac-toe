package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/internal/transport/http/middleware"
)

type userServiceStub struct {
	user domain.User
	err  error
}

func (s *userServiceStub) CreateUser(context.Context, domain.User) error {
	return s.err
}

func (s *userServiceStub) GetUserByLogin(context.Context, string) (domain.User, error) {
	return s.user, s.err
}

func (s *userServiceStub) GetUserByUUID(context.Context, string) (domain.User, error) {
	return s.user, s.err
}

func (s *userServiceStub) UpdatePassword(context.Context, string, string) error {
	return s.err
}

func (s *userServiceStub) VerifyPassword(domain.User, string) (bool, bool) {
	return true, false
}

func TestUserHandlerGetUser(t *testing.T) {
	handler := NewUserHandler(&userServiceStub{
		user: domain.User{UUID: testUUID, Login: "player"},
	})
	req := authenticatedRequest(http.MethodGet, "/users/"+testUUID, nil)
	rec := httptest.NewRecorder()

	handler.GetUser(rec, req, testUUID)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload struct {
		UUID  googleuuid.UUID `json:"uuid"`
		Login string          `json:"login"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.UUID.String() != testUUID || payload.Login != "player" {
		t.Fatalf("unexpected user response: %#v", payload)
	}
}

func TestUserHandlerGetUserErrors(t *testing.T) {
	handler := NewUserHandler(&userServiceStub{err: domain.ErrUserNotFound})
	req := authenticatedRequest(http.MethodGet, "/users/"+testUUID, nil)
	rec := httptest.NewRecorder()

	handler.GetUser(rec, req, testUUID)
	assertStatusAndMessage(t, rec, http.StatusNotFound, "user not found")

	handler = NewUserHandler(&userServiceStub{err: errors.New("db failed")})
	rec = httptest.NewRecorder()
	handler.GetUser(rec, req, testUUID)
	assertStatusAndMessage(t, rec, http.StatusInternalServerError, "failed to load user")

	rec = httptest.NewRecorder()
	handler.GetUser(rec, req, "not-a-uuid")
	assertStatusAndMessage(t, rec, http.StatusBadRequest, "invalid uuid")
}

func TestUserHandlerGetCurrentUser(t *testing.T) {
	handler := NewUserHandler(&userServiceStub{
		user: domain.User{UUID: testUUID, Login: "player"},
	})
	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	req = req.WithContext(middleware.WithUserUUID(req.Context(), testUUID))
	rec := httptest.NewRecorder()

	handler.GetCurrentUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload struct {
		UUID  googleuuid.UUID `json:"uuid"`
		Login string          `json:"login"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.UUID.String() != testUUID || payload.Login != "player" {
		t.Fatalf("unexpected user response: %#v", payload)
	}
}
