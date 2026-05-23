package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	googleuuid "github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"tic-tac-toe/app/domain"
)

type UserService struct {
	repository domain.UserRepository
}

func NewUserService(repository domain.UserRepository) domain.UserService {
	return &UserService{repository: repository}
}

func (s *UserService) CreateUser(ctx context.Context, user domain.User) error {
	logApplication("create user", "login", user.Login, "uuid", user.UUID)

	if user.UUID == "" {
		uuid, err := newUserUUID()
		if err != nil {
			logApplication("create user failed to generate uuid", "login", user.Login, "error", err)
			return err
		}
		user.UUID = uuid
	}

	passwordHash, err := hashPassword(user.Password)
	if err != nil {
		logApplication("create user failed to hash password", "login", user.Login, "uuid", user.UUID, "error", err)
		return err
	}
	user.Password = passwordHash

	if err := s.repository.SaveUser(ctx, user); err != nil {
		logApplication("create user failed", "login", user.Login, "uuid", user.UUID, "error", err)
		return err
	}

	logApplication("create user ok", "login", user.Login, "uuid", user.UUID)
	return nil
}

func (s *UserService) GetUserByLogin(ctx context.Context, login string) (domain.User, error) {
	logApplication("get user by login", "login", login)
	user, err := s.repository.GetUserByLogin(ctx, login)
	if err != nil {
		logApplication("get user by login failed", "login", login, "error", err)
		return domain.User{}, err
	}
	logApplication("get user by login ok", "login", login, "uuid", user.UUID)
	return user, nil
}

func (s *UserService) GetUserByUUID(ctx context.Context, uuid googleuuid.UUID) (domain.User, error) {
	logApplication("get user by uuid", "uuid", uuid)
	user, err := s.repository.GetUserByUUID(ctx, uuid)
	if err != nil {
		logApplication("get user by uuid failed", "uuid", uuid, "error", err)
		return domain.User{}, err
	}
	logApplication("get user by uuid ok", "uuid", uuid, "login", user.Login)
	return user, nil
}

func (s *UserService) UpdatePassword(ctx context.Context, uuid googleuuid.UUID, password string) error {
	logApplication("update password", "uuid", uuid)
	hash, err := hashPassword(password)
	if err != nil {
		logApplication("update password hash failed", "uuid", uuid, "error", err)
		return err
	}

	if err := s.repository.UpdateUserPassword(ctx, uuid, hash); err != nil {
		logApplication("update password failed", "uuid", uuid, "error", err)
		return err
	}

	logApplication("update password ok", "uuid", uuid)
	return nil
}

func (s *UserService) DeleteUser(ctx context.Context, uuid googleuuid.UUID) error {
	logApplication("delete user", "uuid", uuid)
	if err := s.repository.DeleteUser(ctx, uuid); err != nil {
		logApplication("delete user failed", "uuid", uuid, "error", err)
		return err
	}

	logApplication("delete user ok", "uuid", uuid)
	return nil
}

func (s *UserService) VerifyPassword(user domain.User, password string) (bool, bool) {
	logApplication("verify password", "uuid", user.UUID, "login", user.Login)
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) == nil {
		return true, false
	}

	if user.Password == password || user.Password == legacyHashPassword(password) {
		return true, true
	}

	return false, false
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func legacyHashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func newUserUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
