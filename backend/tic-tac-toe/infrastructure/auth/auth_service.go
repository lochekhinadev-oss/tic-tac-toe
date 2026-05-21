package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidAuthHeader  = errors.New("invalid authorization header")
	ErrInvalidSignUp      = errors.New("login or password is invalid")
	ErrRateLimited        = errors.New("too many requests")
)

const (
	minLoginLength    = 3
	maxLoginLength    = 32
	minPasswordLength = 4
	maxPasswordLength = 128
)

type JwtAuthService struct {
	users    domain.UserService
	sessions repository.AuthSessionRepository
	jwt      *JwtProvider
	limiter  *authActionLimiter
}

func NewAuthService(users domain.UserService, sessions repository.AuthSessionRepository, jwt *JwtProvider) *JwtAuthService {
	return &JwtAuthService{
		users:    users,
		sessions: sessions,
		jwt:      jwt,
		limiter:  newAuthActionLimiter(10, time.Minute),
	}
}

func (s *JwtAuthService) SignUp(ctx context.Context, request SignUpRequest) (bool, error) {
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

func (s *JwtAuthService) Authenticate(ctx context.Context, request JwtRequest) (JwtResponse, error) {
	logAuth("authenticate request")

	request.Login = strings.TrimSpace(request.Login)
	if !s.allowAction(authRateLimitKey("login", request.Login)) {
		logAuth("authenticate rate limited login=%q", request.Login)
		return JwtResponse{}, ErrRateLimited
	}
	if !validCredentials(request.Login, request.Password) {
		logAuth("authenticate invalid credentials payload")
		return JwtResponse{}, ErrInvalidCredentials
	}

	user, err := s.users.GetUserByLogin(ctx, request.Login)
	if errors.Is(err, domain.ErrUserNotFound) {
		logAuth("authenticate user not found login=%q", request.Login)
		return JwtResponse{}, ErrInvalidCredentials
	}
	if err != nil {
		logAuth("authenticate load user failed login=%q: %v", request.Login, err)
		return JwtResponse{}, err
	}
	ok, needsUpdate := s.users.VerifyPassword(user, request.Password)
	if !ok {
		logAuth("authenticate invalid credentials login=%q uuid=%q", request.Login, user.UUID)
		return JwtResponse{}, ErrInvalidCredentials
	}
	if needsUpdate {
		if err := s.users.UpdatePassword(ctx, user.UUID, request.Password); err != nil {
			logAuth("authenticate password migration failed uuid=%q: %v", user.UUID, err)
			return JwtResponse{}, err
		}
	}

	response, _, err := s.issueSession(ctx, user)
	if err != nil {
		logAuth("authenticate token generation failed uuid=%q: %v", user.UUID, err)
		return JwtResponse{}, err
	}

	logAuth("authenticate ok login=%q uuid=%q", request.Login, user.UUID)
	return response, nil
}

func (s *JwtAuthService) RefreshAccessToken(ctx context.Context, refreshToken string) (JwtResponse, error) {
	logAuth("refresh access token request")
	if !s.allowAction(authRateLimitKey("refresh", refreshToken)) {
		logAuth("refresh access rate limited")
		return JwtResponse{}, ErrRateLimited
	}
	session, user, err := s.activeSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return JwtResponse{}, err
	}

	accessToken, err := s.jwt.GenerateAccessToken(user)
	if err != nil {
		return JwtResponse{}, err
	}
	logAuth("refresh access ok user=%q session_found=%t", user.UUID, session.RefreshJTIHash != "")
	return JwtResponse{Type: tokenTypeBearer, AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *JwtAuthService) RefreshRefreshToken(ctx context.Context, refreshToken string) (JwtResponse, error) {
	logAuth("refresh refresh token request")
	if !s.allowAction(authRateLimitKey("refresh-rotation", refreshToken)) {
		logAuth("refresh rotation rate limited")
		return JwtResponse{}, ErrRateLimited
	}
	session, user, err := s.activeSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return JwtResponse{}, err
	}

	response, newSession, err := s.issueSession(ctx, user)
	if err != nil {
		return JwtResponse{}, err
	}

	if err := s.sessions.RevokeSession(ctx, session.RefreshJTIHash); err != nil {
		logAuth("refresh refresh revoke old session failed user=%q: %v", user.UUID, err)
		_ = s.sessions.RevokeSession(ctx, newSession.RefreshJTIHash)
		return JwtResponse{}, err
	}

	logAuth("refresh refresh ok user=%q old_revoked=%t new_created=%t", user.UUID, session.RefreshJTIHash != "", newSession.RefreshJTIHash != "")
	return response, nil
}

func (s *JwtAuthService) Logout(ctx context.Context, refreshToken string) error {
	logAuth("logout request")
	if !s.allowAction(authRateLimitKey("logout", refreshToken)) {
		logAuth("logout rate limited")
		return ErrRateLimited
	}
	session, user, err := s.activeSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return err
	}

	if err := s.sessions.RevokeSession(ctx, session.RefreshJTIHash); err != nil {
		return err
	}

	logAuth("logout ok user=%q session_revoked=%t", user.UUID, session.RefreshJTIHash != "")
	return nil
}

func (s *JwtAuthService) LogoutAll(ctx context.Context, refreshToken string) error {
	logAuth("logout all request")
	if !s.allowAction(authRateLimitKey("logout-all", refreshToken)) {
		logAuth("logout all rate limited")
		return ErrRateLimited
	}

	session, user, err := s.activeSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return err
	}

	rows, err := s.sessions.RevokeSessionsByUserUUID(ctx, user.UUID)
	if err != nil {
		return err
	}

	logAuth("logout all ok user=%q session_found=%t revoked=%d", user.UUID, session.RefreshJTIHash != "", rows)
	return nil
}

func (s *JwtAuthService) AuthenticateToken(_ context.Context, header string) (string, error) {
	token, err := parseBearerAuthorizationHeader(header)
	if err != nil {
		return "", err
	}
	uuid, err := s.jwt.UUIDFromToken(token)
	if err != nil {
		return "", ErrInvalidToken
	}
	return uuid, nil
}

func (s *JwtAuthService) userByToken(ctx context.Context, token string) (domain.User, error) {
	uuid, err := s.jwt.UUIDFromToken(token)
	if err != nil {
		return domain.User{}, ErrInvalidToken
	}

	user, err := s.users.GetUserByUUID(ctx, uuid)
	if errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, ErrInvalidToken
	}
	return user, err
}

func (s *JwtAuthService) jwtResponse(user domain.User, refreshToken string) (JwtResponse, error) {
	accessToken, err := s.jwt.GenerateAccessToken(user)
	if err != nil {
		return JwtResponse{}, err
	}
	if refreshToken == "" {
		refreshToken, err = s.jwt.GenerateRefreshToken(user)
		if err != nil {
			return JwtResponse{}, err
		}
	}
	return JwtResponse{Type: tokenTypeBearer, AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func (s *JwtAuthService) issueSession(ctx context.Context, user domain.User) (JwtResponse, repository.RefreshSession, error) {
	response, err := s.jwtResponse(user, "")
	if err != nil {
		return JwtResponse{}, repository.RefreshSession{}, err
	}

	refreshClaims, err := s.jwt.parseToken(response.RefreshToken, jwtTypeRefresh)
	if err != nil {
		return JwtResponse{}, repository.RefreshSession{}, err
	}

	now := time.Now().UTC()
	session := repository.RefreshSession{
		RefreshJTIHash: hashTokenID(refreshClaims.ID),
		UserUUID:       user.UUID,
		CreatedAt:      now,
		ExpiresAt:      refreshClaims.ExpiresAt.Time.UTC(),
		LastUsedAt:     now,
	}
	if err := s.sessions.CreateSession(ctx, session); err != nil {
		return JwtResponse{}, repository.RefreshSession{}, err
	}

	return response, session, nil
}

func (s *JwtAuthService) activeSessionByRefreshToken(ctx context.Context, refreshToken string) (repository.RefreshSession, domain.User, error) {
	claims, err := s.jwt.parseToken(refreshToken, jwtTypeRefresh)
	if err != nil {
		logAuth("invalid refresh token: %v", err)
		return repository.RefreshSession{}, domain.User{}, ErrInvalidToken
	}

	session, err := s.sessions.FindActiveSessionByRefreshJTIHash(ctx, hashTokenID(claims.ID))
	if errors.Is(err, repository.ErrSessionNotFound) {
		logAuth("refresh session not found user=%q", claims.UUID)
		return repository.RefreshSession{}, domain.User{}, ErrInvalidToken
	}
	if err != nil {
		logAuth("load refresh session failed user=%q: %v", claims.UUID, err)
		return repository.RefreshSession{}, domain.User{}, err
	}
	if session.UserUUID != claims.UUID {
		logAuth("refresh session user mismatch token_user=%q session_user=%q", claims.UUID, session.UserUUID)
		return repository.RefreshSession{}, domain.User{}, ErrInvalidToken
	}

	user, err := s.users.GetUserByUUID(ctx, session.UserUUID)
	if errors.Is(err, domain.ErrUserNotFound) {
		logAuth("refresh session user missing uuid=%q", session.UserUUID)
		return repository.RefreshSession{}, domain.User{}, ErrInvalidToken
	}
	if err != nil {
		logAuth("load user for refresh failed uuid=%q: %v", session.UserUUID, err)
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

func authRateLimitKey(action string, value string) string {
	if action == "login" {
		return action + ":" + strings.ToLower(strings.TrimSpace(value))
	}
	return action + ":" + hashTokenID(value)
}

func (s *JwtAuthService) allowAction(key string) bool {
	if s.limiter == nil {
		return true
	}
	return s.limiter.Allow(key)
}
