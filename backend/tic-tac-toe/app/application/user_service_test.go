package application

import (
	"context"
	"errors"
	"testing"

	"tic-tac-toe/app/domain"
)

type userRepositoryStub struct {
	user            domain.User
	saveErr         error
	getErr          error
	updateErr       error
	saved           domain.User
	updatedUUID     string
	updatedPassword string
}

func (r *userRepositoryStub) SaveUser(_ context.Context, user domain.User) error {
	r.saved = user
	return r.saveErr
}

func (r *userRepositoryStub) GetUserByLogin(context.Context, string) (domain.User, error) {
	return r.user, r.getErr
}

func (r *userRepositoryStub) GetUserByUUID(context.Context, string) (domain.User, error) {
	return r.user, r.getErr
}

func (r *userRepositoryStub) UpdateUserPassword(_ context.Context, uuid string, password string) error {
	r.updatedUUID = uuid
	r.updatedPassword = password
	return r.updateErr
}

func TestUserServiceCreateAndVerifyPassword(t *testing.T) {
	repo := &userRepositoryStub{}
	service := NewUserService(repo)

	if err := service.CreateUser(context.Background(), domain.User{Login: "player", Password: "secret"}); err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if repo.saved.UUID == "" || repo.saved.Login != "player" || repo.saved.Password == "secret" {
		t.Fatalf("unexpected saved user: %#v", repo.saved)
	}

	ok, needsUpdate := service.VerifyPassword(repo.saved, "secret")
	if !ok || needsUpdate {
		t.Fatalf("expected bcrypt password to verify without update, got ok=%v update=%v", ok, needsUpdate)
	}
}

func TestUserServiceForwardsRepositoryCalls(t *testing.T) {
	repo := &userRepositoryStub{user: domain.User{UUID: "user-1", Login: "player"}}
	service := NewUserService(repo)

	if user, err := service.GetUserByLogin(context.Background(), "player"); err != nil || user.UUID != "user-1" {
		t.Fatalf("unexpected login lookup result: %#v %v", user, err)
	}
	if user, err := service.GetUserByUUID(context.Background(), "user-1"); err != nil || user.Login != "player" {
		t.Fatalf("unexpected uuid lookup result: %#v %v", user, err)
	}
	if err := service.UpdatePassword(context.Background(), "user-1", "new-secret"); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if repo.updatedUUID != "user-1" || repo.updatedPassword == "" || repo.updatedPassword == "new-secret" {
		t.Fatalf("unexpected password update: %#v", repo)
	}
}

func TestUserServiceErrorsAndLegacyPassword(t *testing.T) {
	saveErr := errors.New("save failed")
	getErr := errors.New("get failed")
	updateErr := errors.New("update failed")

	if err := NewUserService(&userRepositoryStub{saveErr: saveErr}).CreateUser(context.Background(), domain.User{Password: "secret"}); !errors.Is(err, saveErr) {
		t.Fatalf("expected save error, got %v", err)
	}
	if _, err := NewUserService(&userRepositoryStub{getErr: getErr}).GetUserByLogin(context.Background(), "player"); !errors.Is(err, getErr) {
		t.Fatalf("expected get by login error, got %v", err)
	}
	if _, err := NewUserService(&userRepositoryStub{getErr: getErr}).GetUserByUUID(context.Background(), "user-1"); !errors.Is(err, getErr) {
		t.Fatalf("expected get by uuid error, got %v", err)
	}
	if err := NewUserService(&userRepositoryStub{updateErr: updateErr}).UpdatePassword(context.Background(), "user-1", "secret"); !errors.Is(err, updateErr) {
		t.Fatalf("expected update error, got %v", err)
	}

	service := NewUserService(&userRepositoryStub{})
	ok, needsUpdate := service.VerifyPassword(domain.User{Password: "secret"}, "secret")
	if !ok || !needsUpdate {
		t.Fatalf("expected plaintext legacy password to require update, got ok=%v update=%v", ok, needsUpdate)
	}
	ok, needsUpdate = service.VerifyPassword(domain.User{Password: legacyHashPassword("secret")}, "secret")
	if !ok || !needsUpdate {
		t.Fatalf("expected sha legacy password to require update, got ok=%v update=%v", ok, needsUpdate)
	}
	ok, needsUpdate = service.VerifyPassword(domain.User{Password: "different"}, "secret")
	if ok || needsUpdate {
		t.Fatalf("expected invalid password, got ok=%v update=%v", ok, needsUpdate)
	}
}
