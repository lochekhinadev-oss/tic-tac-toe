package auth

import (
	"context"
	"time"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/repository"
)

type sessionStoreStub struct {
	sessions  map[string]repository.RefreshSession
	revoked   map[string]bool
	createErr error
}

type failingSessionStoreStub struct {
	*sessionStoreStub
	revokeErrFor map[string]error
}

func newSessionStoreStub() *sessionStoreStub {
	return &sessionStoreStub{
		sessions: make(map[string]repository.RefreshSession),
		revoked:  make(map[string]bool),
	}
}

func (s *sessionStoreStub) CreateSession(_ context.Context, session repository.RefreshSession) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.sessions[session.RefreshJTIHash] = session
	s.revoked[session.RefreshJTIHash] = false
	return nil
}

func (s *sessionStoreStub) FindActiveSessionByRefreshJTIHash(_ context.Context, refreshJTIHash string) (repository.RefreshSession, error) {
	session, ok := s.sessions[refreshJTIHash]
	if !ok || s.revoked[refreshJTIHash] {
		return repository.RefreshSession{}, repository.ErrSessionNotFound
	}
	return session, nil
}

func (s *sessionStoreStub) RevokeSession(_ context.Context, refreshJTIHash string) error {
	if _, ok := s.sessions[refreshJTIHash]; !ok || s.revoked[refreshJTIHash] {
		return repository.ErrSessionNotFound
	}
	s.revoked[refreshJTIHash] = true
	return nil
}

func (s *failingSessionStoreStub) RevokeSession(ctx context.Context, refreshJTIHash string) error {
	if err, ok := s.revokeErrFor[refreshJTIHash]; ok {
		return err
	}
	return s.sessionStoreStub.RevokeSession(ctx, refreshJTIHash)
}

func (s *sessionStoreStub) RevokeSessionsByUserUUID(_ context.Context, userUUID string) (int64, error) {
	var revoked int64
	for key, session := range s.sessions {
		if session.UserUUID == userUUID && !s.revoked[key] {
			s.revoked[key] = true
			revoked++
		}
	}
	return revoked, nil
}

type userServiceStub struct {
	user        domain.User
	createErr   error
	getErr      error
	updateErr   error
	deleteErr   error
	created     domain.User
	updatedUUID string
}

func (s *userServiceStub) CreateUser(_ context.Context, user domain.User) error {
	s.created = user
	return s.createErr
}

func (s *userServiceStub) GetUserByLogin(context.Context, string) (domain.User, error) {
	return s.user, s.getErr
}

func (s *userServiceStub) GetUserByUUID(context.Context, googleuuid.UUID) (domain.User, error) {
	return s.user, s.getErr
}

func (s *userServiceStub) UpdatePassword(_ context.Context, uuid googleuuid.UUID, password string) error {
	s.updatedUUID = uuid.String()
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	s.user.Password = hash
	return s.updateErr
}

func (s *userServiceStub) DeleteUser(context.Context, googleuuid.UUID) error {
	return s.deleteErr
}

func (s *userServiceStub) VerifyPassword(user domain.User, password string) (bool, bool) {
	if isPasswordHash(password, user.Password) {
		return true, false
	}
	if user.Password == password || user.Password == legacyHashPassword(password) {
		return true, true
	}
	return false, false
}

type authorizationServiceStub struct {
	grantDefaultRoleErr  error
	grantDefaultRoleUUID string
}

func (s *authorizationServiceStub) GrantDefaultRole(_ context.Context, userUUID googleuuid.UUID) error {
	s.grantDefaultRoleUUID = userUUID.String()
	return s.grantDefaultRoleErr
}

func (s *authorizationServiceStub) GrantRoleToUser(context.Context, googleuuid.UUID, string) error {
	return nil
}

func (s *authorizationServiceStub) RevokeRoleFromUser(context.Context, googleuuid.UUID, string) error {
	return nil
}

func (s *authorizationServiceStub) LoadPrincipal(context.Context, googleuuid.UUID) (domain.Principal, error) {
	return domain.Principal{}, nil
}

func (s *authorizationServiceStub) Can(context.Context, googleuuid.UUID, domain.Permission) (bool, error) {
	return true, nil
}

func testAuthConfig() AuthConfig {
	return AuthConfig{
		SessionCookieName: defaultSessionCookie,
		SessionTTL:        time.Hour,
	}
}
