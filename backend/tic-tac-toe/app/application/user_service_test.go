package application

import (
	"context"
	"errors"
	"testing"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/domain"
)

type userRepositoryStub struct {
	user            domain.User
	saveErr         error
	getErr          error
	updateErr       error
	deleteErr       error
	saved           domain.User
	updatedUUID     string
	updatedPassword string
	deletedUUID     string
}

func (r *userRepositoryStub) SaveUser(_ context.Context, user domain.User) error {
	r.saved = user
	return r.saveErr
}

func (r *userRepositoryStub) GetUserByLogin(context.Context, string) (domain.User, error) {
	return r.user, r.getErr
}

func (r *userRepositoryStub) GetUserByUUID(context.Context, googleuuid.UUID) (domain.User, error) {
	return r.user, r.getErr
}

func (r *userRepositoryStub) UpdateUserPassword(_ context.Context, uuid googleuuid.UUID, password string) error {
	r.updatedUUID = uuid.String()
	r.updatedPassword = password
	return r.updateErr
}

func (r *userRepositoryStub) DeleteUser(_ context.Context, uuid googleuuid.UUID) error {
	r.deletedUUID = uuid.String()
	return r.deleteErr
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
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	repo := &userRepositoryStub{user: domain.User{UUID: userUUID.String(), Login: "player"}}
	service := NewUserService(repo)

	if user, err := service.GetUserByLogin(context.Background(), "player"); err != nil || user.UUID != userUUID.String() {
		t.Fatalf("unexpected login lookup result: %#v %v", user, err)
	}
	if user, err := service.GetUserByUUID(context.Background(), userUUID); err != nil || user.Login != "player" {
		t.Fatalf("unexpected uuid lookup result: %#v %v", user, err)
	}
	if err := service.UpdatePassword(context.Background(), userUUID, "new-secret"); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if repo.updatedUUID != userUUID.String() || repo.updatedPassword == "" || repo.updatedPassword == "new-secret" {
		t.Fatalf("unexpected password update: %#v", repo)
	}
}

func TestUserServiceDeleteUserForwardsRepositoryCall(t *testing.T) {
	repo := &userRepositoryStub{}
	service := NewUserService(repo)

	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	if err := service.DeleteUser(context.Background(), userUUID); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if repo.deletedUUID != userUUID.String() {
		t.Fatalf("unexpected deleted uuid: %q", repo.deletedUUID)
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
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")
	if _, err := NewUserService(&userRepositoryStub{getErr: getErr}).GetUserByUUID(context.Background(), userUUID); !errors.Is(err, getErr) {
		t.Fatalf("expected get by uuid error, got %v", err)
	}
	if err := NewUserService(&userRepositoryStub{updateErr: updateErr}).UpdatePassword(context.Background(), userUUID, "secret"); !errors.Is(err, updateErr) {
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
