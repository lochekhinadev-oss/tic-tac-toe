package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidSignUp      = errors.New("login or password is invalid")
)

const (
	minLoginLength    = 3
	maxLoginLength    = 32
	minPasswordLength = 4
	maxPasswordLength = 128
)

type service struct {
	users         domain.UserService
	authorization domain.AuthorizationService
	sessions      repository.AuthSessionRepository
	sessionTTL    time.Duration
}

func NewAuthService(users domain.UserService, authorization domain.AuthorizationService, sessions repository.AuthSessionRepository, config AuthConfig) *service {
	return &service{
		users:         users,
		authorization: authorization,
		sessions:      sessions,
		sessionTTL:    config.SessionTTL,
	}
}

func (s *service) SignUp(ctx context.Context, request SignUpRequest) (bool, error) {
	request.Login = strings.TrimSpace(request.Login)
	logAuth("sign up request login=%q", request.Login)
	if !validCredentials(request.Login, request.Password) {
		logAuth("sign up invalid credentials login=%q", request.Login)
		return false, ErrInvalidSignUp
	}

	err := s.users.CreateUser(ctx, domain.User{
		Login:    request.Login,
		Password: request.Password,
	})
	if errors.Is(err, repository.ErrLoginAlreadyExists) {
		logAuth("sign up duplicate login=%q", request.Login)
		return false, nil
	}
	if err != nil {
		logAuth("sign up failed login=%q: %v", request.Login, err)
		return false, err
	}

	logAuth("sign up ok login=%q", request.Login)
	return true, nil
}

func (s *service) SignIn(ctx context.Context, request SessionRequest) (SessionResponse, error) {
	logAuth("sign in request")

	request.Login = strings.TrimSpace(request.Login)
	if !validCredentials(request.Login, request.Password) {
		logAuth("sign in invalid credentials payload")
		return SessionResponse{}, ErrInvalidCredentials
	}

	user, err := s.users.GetUserByLogin(ctx, request.Login)
	if errors.Is(err, domain.ErrUserNotFound) {
		logAuth("sign in user not found login=%q", request.Login)
		return SessionResponse{}, ErrInvalidCredentials
	}
	if err != nil {
		logAuth("sign in load user failed login=%q: %v", request.Login, err)
		return SessionResponse{}, err
	}

	ok, needsUpdate := s.users.VerifyPassword(user, request.Password)
	if !ok {
		logAuth("sign in invalid credentials login=%q uuid=%q", request.Login, user.UUID)
		return SessionResponse{}, ErrInvalidCredentials
	}
	if needsUpdate {
		if err := s.users.UpdatePassword(ctx, user.UUID, request.Password); err != nil {
			logAuth("sign in password migration failed uuid=%q: %v", user.UUID, err)
			return SessionResponse{}, err
		}
	}

	if s.authorization != nil {
		if err := s.authorization.GrantDefaultRole(ctx, user.UUID); err != nil {
			logAuth("sign in default role grant failed uuid=%q: %v", user.UUID, err)
			return SessionResponse{}, err
		}
	}

	session, err := s.issueCookieSession(ctx, user)
	if err != nil {
		logAuth("sign in session generation failed uuid=%q: %v", user.UUID, err)
		return SessionResponse{}, err
	}

	logAuth("sign in ok login=%q uuid=%q", request.Login, user.UUID)
	return session, nil
}

func (s *service) RefreshSession(ctx context.Context, sessionID string) (SessionResponse, error) {
	logAuth("refresh session request")
	session, user, err := s.activeSessionBySessionID(ctx, sessionID)
	if err != nil {
		return SessionResponse{}, err
	}

	if err := s.sessions.RevokeSession(ctx, session.RefreshJTIHash); err != nil {
		logAuth("refresh session revoke old failed user=%q: %v", user.UUID, err)
		return SessionResponse{}, err
	}

	refreshed, err := s.issueCookieSession(ctx, user)
	if err != nil {
		logAuth("refresh session create new failed user=%q: %v", user.UUID, err)
		return SessionResponse{}, err
	}

	logAuth("refresh session ok user=%q old_revoked=%t new_created=%t", user.UUID, session.RefreshJTIHash != "", refreshed.SessionID != "")
	return refreshed, nil
}

func (s *service) Logout(ctx context.Context, sessionID string) error {
	logAuth("logout request")
	session, user, err := s.activeSessionBySessionID(ctx, sessionID)
	if err != nil {
		return err
	}
	if err := s.sessions.RevokeSession(ctx, session.RefreshJTIHash); err != nil {
		return err
	}
	logAuth("logout session ok user=%q session_revoked=%t", user.UUID, session.RefreshJTIHash != "")
	return nil
}

func (s *service) LogoutAll(ctx context.Context, sessionID string) error {
	logAuth("logout all request")
	session, user, err := s.activeSessionBySessionID(ctx, sessionID)
	if err != nil {
		return err
	}
	rows, err := s.sessions.RevokeSessionsByUserUUID(ctx, user.UUID)
	if err != nil {
		return err
	}
	logAuth("logout all sessions ok user=%q session_found=%t revoked=%d", user.UUID, session.RefreshJTIHash != "", rows)
	return nil
}

func (s *service) AuthenticateSession(ctx context.Context, sessionID string) (string, error) {
	session, user, err := s.activeSessionBySessionID(ctx, sessionID)
	if err != nil {
		return "", err
	}
	logAuth("authenticate session ok user=%q session_found=%t", user.UUID, session.RefreshJTIHash != "")
	return user.UUID, nil
}

func (s *service) issueCookieSession(ctx context.Context, user domain.User) (SessionResponse, error) {
	if strings.TrimSpace(user.UUID) == "" {
		return SessionResponse{}, ErrInvalidCredentials
	}

	sessionID := uuid.NewString()
	now := time.Now().UTC()
	sessionTTL := s.sessionTTL
	if sessionTTL <= 0 {
		sessionTTL = defaultSessionTTL
	}
	session := repository.RefreshSession{
		RefreshJTIHash: hashTokenID(sessionID),
		UserUUID:       user.UUID,
		CreatedAt:      now,
		ExpiresAt:      now.Add(sessionTTL),
		LastUsedAt:     now,
	}
	if err := s.sessions.CreateSession(ctx, session); err != nil {
		return SessionResponse{}, err
	}
	return SessionResponse{
		UserUUID:  user.UUID,
		SessionID: sessionID,
		ExpiresAt: session.ExpiresAt,
	}, nil
}

func (s *service) activeSessionBySessionID(ctx context.Context, sessionID string) (repository.RefreshSession, domain.User, error) {
	if strings.TrimSpace(sessionID) == "" {
		return repository.RefreshSession{}, domain.User{}, ErrInvalidToken
	}

	session, err := s.sessions.FindActiveSessionByRefreshJTIHash(ctx, hashTokenID(sessionID))
	if errors.Is(err, repository.ErrSessionNotFound) {
		logAuth("session not found")
		return repository.RefreshSession{}, domain.User{}, ErrInvalidToken
	}
	if err != nil {
		logAuth("load session failed: %v", err)
		return repository.RefreshSession{}, domain.User{}, err
	}

	user, err := s.users.GetUserByUUID(ctx, session.UserUUID)
	if errors.Is(err, domain.ErrUserNotFound) {
		logAuth("session user missing uuid=%q", session.UserUUID)
		return repository.RefreshSession{}, domain.User{}, ErrInvalidToken
	}
	if err != nil {
		logAuth("load user for session failed uuid=%q: %v", session.UserUUID, err)
		return repository.RefreshSession{}, domain.User{}, err
	}

	return session, user, nil
}

func validCredentials(login string, password string) bool {
	return validLogin(login) && len(password) >= minPasswordLength && len(password) <= maxPasswordLength
}

func validLogin(login string) bool {
	if login != strings.TrimSpace(login) || len(login) < minLoginLength || len(login) > maxLoginLength {
		return false
	}
	for _, char := range login {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' {
			continue
		}
		if char == '_' || char == '-' || char == '.' {
			continue
		}
		return false
	}
	return true
}

func hashTokenID(tokenID string) string {
	sum := sha256.Sum256([]byte(tokenID))
	return hex.EncodeToString(sum[:])
}
