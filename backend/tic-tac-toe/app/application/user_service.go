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
	logApplication("create user login=%q uuid=%q", user.Login, user.UUID)

	if user.UUID == "" {
		uuid, err := newUserUUID()
		if err != nil {
			logApplication("create user failed to generate uuid login=%q: %v", user.Login, err)
			return err
		}
		user.UUID = uuid
	}

	passwordHash, err := hashPassword(user.Password)
	if err != nil {
		logApplication("create user failed to hash password login=%q uuid=%q: %v", user.Login, user.UUID, err)
		return err
	}
	user.Password = passwordHash

	if err := s.repository.SaveUser(ctx, user); err != nil {
		logApplication("create user failed login=%q uuid=%q: %v", user.Login, user.UUID, err)
		return err
	}

	logApplication("create user ok login=%q uuid=%q", user.Login, user.UUID)
	return nil
}

func (s *UserService) GetUserByLogin(ctx context.Context, login string) (domain.User, error) {
	logApplication("get user by login login=%q", login)
	user, err := s.repository.GetUserByLogin(ctx, login)
	if err != nil {
		logApplication("get user by login failed login=%q: %v", login, err)
		return domain.User{}, err
	}
	logApplication("get user by login ok login=%q uuid=%q", login, user.UUID)
	return user, nil
}

func (s *UserService) GetUserByUUID(ctx context.Context, uuid googleuuid.UUID) (domain.User, error) {
	logApplication("get user by uuid uuid=%q", uuid)
	user, err := s.repository.GetUserByUUID(ctx, uuid)
	if err != nil {
		logApplication("get user by uuid failed uuid=%q: %v", uuid, err)
		return domain.User{}, err
	}
	logApplication("get user by uuid ok uuid=%q login=%q", uuid, user.Login)
	return user, nil
}

func (s *UserService) UpdatePassword(ctx context.Context, uuid googleuuid.UUID, password string) error {
	logApplication("update password uuid=%q", uuid)
	hash, err := hashPassword(password)
	if err != nil {
		logApplication("update password hash failed uuid=%q: %v", uuid, err)
		return err
	}

	if err := s.repository.UpdateUserPassword(ctx, uuid, hash); err != nil {
		logApplication("update password failed uuid=%q: %v", uuid, err)
		return err
	}

	logApplication("update password ok uuid=%q", uuid)
	return nil
}

func (s *UserService) DeleteUser(ctx context.Context, uuid googleuuid.UUID) error {
	logApplication("delete user uuid=%q", uuid)
	if err := s.repository.DeleteUser(ctx, uuid); err != nil {
		logApplication("delete user failed uuid=%q: %v", uuid, err)
		return err
	}

	logApplication("delete user ok uuid=%q", uuid)
	return nil
}

func (s *UserService) VerifyPassword(user domain.User, password string) (bool, bool) {
	logApplication("verify password uuid=%q login=%q", user.UUID, user.Login)
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
