package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

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

func (s *userServiceStub) GetUserByUUID(context.Context, string) (domain.User, error) {
	return s.user, s.getErr
}

func (s *userServiceStub) UpdatePassword(_ context.Context, uuid string, password string) error {
	s.updatedUUID = uuid
	hash, err := hashPassword(password)
	if err != nil {
		return err
	}
	s.user.Password = hash
	return s.updateErr
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

func TestAuthServiceSignUp(t *testing.T) {
	t.Run("creates user", func(t *testing.T) {
		users := &userServiceStub{}
		auth := NewAuthService(users, newSessionStoreStub(), NewJwtProvider(testAuthConfig()))

		ok, err := auth.SignUp(context.Background(), SignUpRequest{Login: " player ", Password: "secret"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatal("expected successful registration")
		}
		if users.created.Login != "player" || users.created.Password != "secret" {
			t.Fatalf("unexpected created user: %#v", users.created)
		}
	})

	t.Run("returns false for duplicate login", func(t *testing.T) {
		auth := NewAuthService(&userServiceStub{createErr: repository.ErrLoginAlreadyExists}, newSessionStoreStub(), NewJwtProvider(testAuthConfig()))

		ok, err := auth.SignUp(context.Background(), SignUpRequest{Login: "player", Password: "secret"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Fatal("expected duplicate login to return false")
		}
	})

	t.Run("rejects empty credentials", func(t *testing.T) {
		auth := NewAuthService(&userServiceStub{}, newSessionStoreStub(), NewJwtProvider(testAuthConfig()))

		ok, err := auth.SignUp(context.Background(), SignUpRequest{Login: " ", Password: ""})
		if !errors.Is(err, ErrInvalidSignUp) {
			t.Fatalf("expected ErrInvalidSignUp, got %v", err)
		}
		if ok {
			t.Fatal("expected failed registration")
		}
	})
}

func TestAuthServiceAuthenticate(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	auth := NewAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, newSessionStoreStub(), NewJwtProvider(testAuthConfig()))

	response, err := auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Type != tokenTypeBearer || response.AccessToken == "" || response.RefreshToken == "" {
		t.Fatalf("unexpected jwt response: %#v", response)
	}

	uuid, err := auth.AuthenticateToken(context.Background(), "Bearer "+response.AccessToken)
	if err != nil {
		t.Fatalf("unexpected token auth error: %v", err)
	}
	if uuid != "user-1" {
		t.Fatalf("expected user-1, got %q", uuid)
	}
}

func TestAuthServiceAuthenticateMigratesPlaintextPassword(t *testing.T) {
	users := &userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: "secret"},
	}
	auth := NewAuthService(users, newSessionStoreStub(), NewJwtProvider(testAuthConfig()))

	response, err := auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	uuid, err := auth.AuthenticateToken(context.Background(), "Bearer "+response.AccessToken)
	if err != nil {
		t.Fatalf("unexpected token auth error: %v", err)
	}
	if uuid != "user-1" || users.updatedUUID != "user-1" {
		t.Fatalf("expected plaintext password migration, got uuid=%q updated=%q", uuid, users.updatedUUID)
	}
	if !isPasswordHash("secret", users.user.Password) {
		t.Fatal("expected migrated password hash")
	}
}

func TestAuthServiceAuthenticateReportsSessionCreationError(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	auth := NewAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, &sessionStoreStub{createErr: errors.New("create failed")}, NewJwtProvider(testAuthConfig()))

	_, err = auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "secret"})
	if err == nil {
		t.Fatal("expected session creation error")
	}
}

func TestAuthServiceAuthenticateRejectsInvalidCredentials(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	auth := NewAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, newSessionStoreStub(), NewJwtProvider(testAuthConfig()))

	_, err = auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "wrong"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func isPasswordHash(password string, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func legacyHashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func TestAuthServiceAuthenticateRejectsEmptyCredentials(t *testing.T) {
	auth := NewAuthService(&userServiceStub{}, newSessionStoreStub(), NewJwtProvider(testAuthConfig()))

	_, err := auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: ""})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidCredentials(t *testing.T) {
	cases := []struct {
		name     string
		login    string
		password string
		want     bool
	}{
		{name: "valid", login: "player_01", password: "0000", want: true},
		{name: "valid punctuation", login: "player-01.dev", password: "secret", want: true},
		{name: "short login", login: "ab", password: "secret"},
		{name: "long login", login: "player_player_player_player_player_1", password: "secret"},
		{name: "login spaces", login: "player one", password: "secret"},
		{name: "login sql payload", login: "admin' OR '1'='1", password: "secret"},
		{name: "trimmed login required", login: " player", password: "secret"},
		{name: "short password", login: "player", password: "123"},
		{name: "long password", login: "player", password: strings.Repeat("a", maxPasswordLength+1)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validCredentials(tc.login, tc.password); got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}

func TestAuthServiceAuthenticateReportsPasswordUpdateError(t *testing.T) {
	users := &userServiceStub{
		user:      domain.User{UUID: "user-1", Login: "player", Password: "secret"},
		updateErr: errors.New("update failed"),
	}
	auth := NewAuthService(users, newSessionStoreStub(), NewJwtProvider(testAuthConfig()))

	_, err := auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "secret"})
	if err == nil {
		t.Fatal("expected password update error")
	}
}

func TestAuthServiceUserByToken(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	provider := NewJwtProvider(testAuthConfig())
	users := &userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}
	auth := NewAuthService(users, newSessionStoreStub(), provider)

	accessToken, err := provider.GenerateAccessToken(domain.User{UUID: "user-1"})
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	user, err := auth.userByToken(context.Background(), accessToken)
	if err != nil {
		t.Fatalf("unexpected userByToken error: %v", err)
	}
	if user.UUID != "user-1" || user.Login != "player" {
		t.Fatalf("unexpected user: %#v", user)
	}

	users.getErr = domain.ErrUserNotFound
	if _, err := auth.userByToken(context.Background(), accessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token for missing user, got %v", err)
	}
}

func TestAuthConfigValidate(t *testing.T) {
	privateKey, publicKey := newDevelopmentKeyPair()
	valid := newTestAuthConfig(privateKey, publicKey, "kid", "")
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	cases := []struct {
		name   string
		config AuthConfig
	}{
		{name: "private key", config: AuthConfig{JWTPublicKeyPEM: publicKey, JWTKeyID: "kid", JWTIssuer: "issuer", JWTAudience: "audience", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour}},
		{name: "public key", config: AuthConfig{JWTPrivateKeyPEM: privateKey, JWTKeyID: "kid", JWTIssuer: "issuer", JWTAudience: "audience", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour}},
		{name: "key id", config: AuthConfig{JWTPrivateKeyPEM: privateKey, JWTPublicKeyPEM: publicKey, JWTIssuer: "issuer", JWTAudience: "audience", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour}},
		{name: "issuer", config: AuthConfig{JWTPrivateKeyPEM: privateKey, JWTPublicKeyPEM: publicKey, JWTKeyID: "kid", JWTAudience: "audience", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour}},
		{name: "audience", config: AuthConfig{JWTPrivateKeyPEM: privateKey, JWTPublicKeyPEM: publicKey, JWTKeyID: "kid", JWTIssuer: "issuer", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour}},
		{name: "previous keys", config: AuthConfig{JWTPrivateKeyPEM: privateKey, JWTPublicKeyPEM: publicKey, JWTKeyID: "kid", JWTIssuer: "issuer", JWTAudience: "audience", JWTPreviousPublicKeys: "bad", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour}},
		{name: "access ttl", config: AuthConfig{JWTPrivateKeyPEM: privateKey, JWTPublicKeyPEM: publicKey, JWTKeyID: "kid", JWTIssuer: "issuer", JWTAudience: "audience", AccessTokenTTL: 0, RefreshTokenTTL: time.Hour}},
		{name: "refresh ttl", config: AuthConfig{JWTPrivateKeyPEM: privateKey, JWTPublicKeyPEM: publicKey, JWTKeyID: "kid", JWTIssuer: "issuer", JWTAudience: "audience", AccessTokenTTL: time.Minute, RefreshTokenTTL: 0}},
		{name: "missing private key file", config: AuthConfig{JWTPrivateKeyPath: "/missing/private.pem", JWTPublicKeyPEM: publicKey, JWTPrivateKeyPEM: privateKey, JWTKeyID: "kid", JWTIssuer: "issuer", JWTAudience: "audience", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour}},
		{name: "missing public key file", config: AuthConfig{JWTPrivateKeyPEM: privateKey, JWTPublicKeyPath: "/missing/public.pem", JWTPublicKeyPEM: publicKey, JWTKeyID: "kid", JWTIssuer: "issuer", JWTAudience: "audience", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.config.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNewAuthConfigLoadsRSAKeysFromPaths(t *testing.T) {
	privateKey, publicKey := newDevelopmentKeyPair()
	dir := t.TempDir()
	privatePath := filepath.Join(dir, "private.pem")
	publicPath := filepath.Join(dir, "public.pem")
	if err := os.WriteFile(privatePath, []byte(privateKey), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.WriteFile(publicPath, []byte(publicKey), 0o644); err != nil {
		t.Fatalf("write public key: %v", err)
	}

	t.Setenv("JWT_PRIVATE_KEY_PATH", privatePath)
	t.Setenv("JWT_PUBLIC_KEY_PATH", publicPath)

	config := NewAuthConfig()
	if config.JWTPrivateKeyPath != privatePath || config.JWTPublicKeyPath != publicPath {
		t.Fatalf("unexpected key paths: %#v", config)
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("expected valid path config, got %v", err)
	}
}

func TestAuthServiceRefreshTokens(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	sessions := newSessionStoreStub()
	provider := NewJwtProvider(testAuthConfig())
	auth := NewAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, sessions, provider)

	response, err := auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("unexpected auth error: %v", err)
	}

	refreshedAccess, err := auth.RefreshAccessToken(context.Background(), response.RefreshToken)
	if err != nil {
		t.Fatalf("unexpected access refresh error: %v", err)
	}
	if refreshedAccess.Type != tokenTypeBearer || refreshedAccess.AccessToken == "" || refreshedAccess.RefreshToken != response.RefreshToken {
		t.Fatalf("unexpected access refresh response: %#v", refreshedAccess)
	}

	refreshedRefresh, err := auth.RefreshRefreshToken(context.Background(), response.RefreshToken)
	if err != nil {
		t.Fatalf("unexpected refresh refresh error: %v", err)
	}
	if refreshedRefresh.Type != tokenTypeBearer || refreshedRefresh.AccessToken == "" || refreshedRefresh.RefreshToken == "" {
		t.Fatalf("unexpected refresh refresh response: %#v", refreshedRefresh)
	}
	if refreshedRefresh.RefreshToken == response.RefreshToken {
		t.Fatal("expected refresh rotation to issue a new refresh token")
	}
	refreshedRefreshClaims, err := provider.parseToken(refreshedRefresh.RefreshToken, jwtTypeRefresh)
	if err != nil {
		t.Fatalf("failed to parse rotated refresh token: %v", err)
	}
	if _, err := sessions.FindActiveSessionByRefreshJTIHash(context.Background(), hashTokenID(refreshedRefreshClaims.ID)); err != nil {
		t.Fatalf("expected rotated refresh session to be active: %v", err)
	}
	oldClaims, err := provider.parseToken(response.RefreshToken, jwtTypeRefresh)
	if err != nil {
		t.Fatalf("failed to parse old refresh token: %v", err)
	}
	if _, err := sessions.FindActiveSessionByRefreshJTIHash(context.Background(), hashTokenID(oldClaims.ID)); !errors.Is(err, repository.ErrSessionNotFound) {
		t.Fatalf("expected old refresh session to be revoked, got %v", err)
	}
}

func TestAuthServiceLogoutRevokesRefreshSession(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	sessions := newSessionStoreStub()
	auth := NewAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, sessions, NewJwtProvider(testAuthConfig()))

	response, err := auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("unexpected auth error: %v", err)
	}
	if err := auth.Logout(context.Background(), response.RefreshToken); err != nil {
		t.Fatalf("unexpected logout error: %v", err)
	}
	if _, err := auth.RefreshAccessToken(context.Background(), response.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token after logout, got %v", err)
	}
}

func TestAuthServiceLogoutAllRevokesAllUserSessions(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	sessions := newSessionStoreStub()
	provider := NewJwtProvider(testAuthConfig())
	auth := NewAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, sessions, provider)

	first, err := auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("unexpected auth error: %v", err)
	}
	second, err := auth.RefreshRefreshToken(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatalf("unexpected refresh error: %v", err)
	}
	if err := auth.LogoutAll(context.Background(), second.RefreshToken); err != nil {
		t.Fatalf("unexpected logout all error: %v", err)
	}
	if _, err := auth.RefreshAccessToken(context.Background(), second.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token after logout all, got %v", err)
	}
}

func TestAuthServiceLogoutRejectsRepeatedLogout(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	sessions := newSessionStoreStub()
	auth := NewAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, sessions, NewJwtProvider(testAuthConfig()))

	response, err := auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("unexpected auth error: %v", err)
	}
	if err := auth.Logout(context.Background(), response.RefreshToken); err != nil {
		t.Fatalf("unexpected logout error: %v", err)
	}
	if err := auth.Logout(context.Background(), response.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token on repeated logout, got %v", err)
	}
}

func TestAuthServiceLogoutAllRejectsRepeatedLogoutAll(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	sessions := newSessionStoreStub()
	auth := NewAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, sessions, NewJwtProvider(testAuthConfig()))

	response, err := auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("unexpected auth error: %v", err)
	}
	if err := auth.LogoutAll(context.Background(), response.RefreshToken); err != nil {
		t.Fatalf("unexpected logout all error: %v", err)
	}
	if err := auth.LogoutAll(context.Background(), response.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token on repeated logout all, got %v", err)
	}
}

func TestAuthServiceAuthenticateRateLimitsRepeatedLogin(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	auth := NewAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, newSessionStoreStub(), NewJwtProvider(testAuthConfig()))
	auth.limiter = newAuthActionLimiter(1, time.Hour)

	if _, err := auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "secret"}); err != nil {
		t.Fatalf("unexpected first auth error: %v", err)
	}
	if _, err := auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "secret"}); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected rate limit error, got %v", err)
	}
}

func TestAuthServiceAuthenticateTokenRejectsNonBearerHeader(t *testing.T) {
	auth := NewAuthService(&userServiceStub{}, newSessionStoreStub(), NewJwtProvider(testAuthConfig()))

	_, err := auth.AuthenticateToken(context.Background(), "Basic token")
	if !errors.Is(err, ErrInvalidAuthHeader) {
		t.Fatalf("expected ErrInvalidAuthHeader, got %v", err)
	}
}

func TestParseBearerAuthorizationHeader(t *testing.T) {
	token, err := parseBearerAuthorizationHeader(" bearer token-value ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "token-value" {
		t.Fatalf("expected token-value, got %q", token)
	}

	_, err = parseBearerAuthorizationHeader("Bearer " + strings.Repeat(" ", 2))
	if !errors.Is(err, ErrInvalidAuthHeader) {
		t.Fatalf("expected ErrInvalidAuthHeader, got %v", err)
	}
}

func TestParseBearerAuthorizationHeaderRejectsEmptyHeader(t *testing.T) {
	if _, err := parseBearerAuthorizationHeader("   "); !errors.Is(err, ErrInvalidAuthHeader) {
		t.Fatalf("expected ErrInvalidAuthHeader, got %v", err)
	}
}

func TestAuthServiceActiveSessionByRefreshTokenRejectsMissingSession(t *testing.T) {
	provider := NewJwtProvider(testAuthConfig())
	auth := NewAuthService(&userServiceStub{}, newSessionStoreStub(), provider)

	token, err := provider.GenerateRefreshToken(domain.User{UUID: "user-1"})
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	if _, _, err := auth.activeSessionByRefreshToken(context.Background(), token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token for missing session, got %v", err)
	}
}

func TestAuthServiceActiveSessionByRefreshTokenRejectsUserMismatch(t *testing.T) {
	provider := NewJwtProvider(testAuthConfig())
	auth := NewAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: "hash"},
	}, newSessionStoreStub(), provider)

	token, err := provider.GenerateRefreshToken(domain.User{UUID: "user-1"})
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	claims, err := provider.parseToken(token, jwtTypeRefresh)
	if err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}

	store := newSessionStoreStub()
	store.sessions[hashTokenID(claims.ID)] = repository.RefreshSession{
		RefreshJTIHash: hashTokenID(claims.ID),
		UserUUID:       "user-2",
	}
	auth.sessions = store

	if _, _, err := auth.activeSessionByRefreshToken(context.Background(), token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token for user mismatch, got %v", err)
	}
}

func TestAuthServiceActiveSessionByRefreshTokenReportsUserLoadError(t *testing.T) {
	provider := NewJwtProvider(testAuthConfig())
	users := &userServiceStub{
		user:   domain.User{UUID: "user-1", Login: "player", Password: "hash"},
		getErr: errors.New("load user failed"),
	}
	store := newSessionStoreStub()
	auth := NewAuthService(users, store, provider)

	token, err := provider.GenerateRefreshToken(domain.User{UUID: "user-1"})
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	claims, err := provider.parseToken(token, jwtTypeRefresh)
	if err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}
	store.sessions[hashTokenID(claims.ID)] = repository.RefreshSession{
		RefreshJTIHash: hashTokenID(claims.ID),
		UserUUID:       "user-1",
	}

	if _, _, err := auth.activeSessionByRefreshToken(context.Background(), token); err == nil {
		t.Fatal("expected user load error")
	}
}

func TestAuthServiceRefreshRefreshTokenRollsBackOnRevokeError(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	baseStore := newSessionStoreStub()
	provider := NewJwtProvider(testAuthConfig())
	auth := NewAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, baseStore, provider)

	response, err := auth.Authenticate(context.Background(), JwtRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("unexpected auth error: %v", err)
	}
	claims, err := provider.parseToken(response.RefreshToken, jwtTypeRefresh)
	if err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}
	oldHash := hashTokenID(claims.ID)

	errStore := &failingSessionStoreStub{
		sessionStoreStub: baseStore,
		revokeErrFor:     map[string]error{oldHash: errors.New("revoke failed")},
	}
	auth.sessions = errStore

	if _, err := auth.RefreshRefreshToken(context.Background(), response.RefreshToken); err == nil {
		t.Fatal("expected refresh rotation error")
	}
	if _, err := auth.RefreshAccessToken(context.Background(), response.RefreshToken); err != nil {
		t.Fatalf("expected old session to remain active after rollback, got %v", err)
	}
}

func TestAuthServiceAllowActionWithoutLimiter(t *testing.T) {
	auth := &JwtAuthService{}
	if !auth.allowAction("anything") {
		t.Fatal("expected allowAction to pass without limiter")
	}
}
