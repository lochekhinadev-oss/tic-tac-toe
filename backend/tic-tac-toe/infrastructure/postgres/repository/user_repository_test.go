package repository

import (
	"context"
	"errors"
	"testing"

	googleuuid "github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"tic-tac-toe/app/domain"
)

func TestUserRepositorySaveGetAndUpdate(t *testing.T) {
	user := sampleUser()
	db := &databaseStub{}
	repo := NewUserRepository(db)

	if err := repo.SaveUser(context.Background(), user); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}
	assertSavedArgs(t, db.savedArgs, user.UUID, user.Login, user.Password, domain.DefaultPlayerRole)

	db.savedArgs = []any{user.UUID, user.Login, user.Password}
	user, err := repo.GetUserByLogin(context.Background(), "player")
	if err != nil {
		t.Fatalf("unexpected get by login error: %v", err)
	}
	assertUser(t, user, sampleUser())

	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	user, err = repo.GetUserByUUID(context.Background(), userUUID)
	if err != nil {
		t.Fatalf("unexpected get by uuid error: %v", err)
	}
	assertPassword(t, user.Password, sampleUser().Password)

	if err := repo.UpdateUserPassword(context.Background(), userUUID, "new-hash"); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	assertSavedArgs(t, db.savedArgs, userUUID.String(), "new-hash")

	if err := repo.DeleteUser(context.Background(), userUUID); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	assertSavedArgs(t, db.savedArgs, userUUID.String())
}

func TestUserRepositoryNotFound(t *testing.T) {
	repo := NewUserRepository(&databaseStub{queryErr: pgx.ErrNoRows})
	if _, err := repo.GetUserByLogin(context.Background(), "missing"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
	if _, err := repo.GetUserByUUID(context.Background(), googleuuid.Nil); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserRepositoryUsesParameterizedQueries(t *testing.T) {
	t.Run("save user", func(t *testing.T) {
		db := &databaseStub{}
		repo := NewUserRepository(db)

		err := repo.SaveUser(context.Background(), domain.User{UUID: "user-1", Login: sqlInjectionPayload, Password: "hash"})
		if err != nil {
			t.Fatalf("unexpected save error: %v", err)
		}

		assertQueryDoesNotContainPayload(t, db.lastExecQuery)
		assertArgsContainPayload(t, db.savedArgs)
	})

	t.Run("get by login", func(t *testing.T) {
		db := &databaseStub{}
		repo := NewUserRepository(db)

		_, err := repo.GetUserByLogin(context.Background(), sqlInjectionPayload)
		if err != nil {
			t.Fatalf("unexpected get error: %v", err)
		}

		assertQueryDoesNotContainPayload(t, db.lastQueryRowQuery)
		assertArgsContainPayload(t, db.lastQueryRowArgs)
	})

	t.Run("update password", func(t *testing.T) {
		db := &databaseStub{}
		repo := NewUserRepository(db)

		userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
		err := repo.UpdateUserPassword(context.Background(), userUUID, sqlInjectionPayload)
		if err != nil {
			t.Fatalf("unexpected update error: %v", err)
		}

		assertQueryDoesNotContainPayload(t, db.lastExecQuery)
		assertArgsContainPayload(t, db.savedArgs)
	})

	t.Run("delete user", func(t *testing.T) {
		db := &databaseStub{}
		repo := NewUserRepository(db)
		userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")

		err := repo.DeleteUser(context.Background(), userUUID)
		if err != nil {
			t.Fatalf("unexpected delete error: %v", err)
		}

		assertQueryDoesNotContainPayload(t, db.lastExecQuery)
		if len(db.savedArgs) != 1 || db.savedArgs[0] != userUUID.String() {
			t.Fatalf("expected uuid argument, got %#v", db.savedArgs)
		}
	})
}

func sampleUser() domain.User {
	return domain.User{
		UUID:     "user-1",
		Login:    "player",
		Password: "hash",
	}
}

func assertUser(t *testing.T, got domain.User, want domain.User) {
	t.Helper()

	if got.UUID != want.UUID || got.Login != want.Login {
		t.Fatalf("unexpected user: %#v", got)
	}
}

func assertPassword(t *testing.T, got string, want string) {
	t.Helper()

	if got != want {
		t.Fatalf("unexpected password: %q", got)
	}
}

func assertSavedArgs(t *testing.T, got []any, want ...any) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("unexpected args: %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected args: %#v", got)
		}
	}
}
