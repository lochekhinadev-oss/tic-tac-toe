package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/repository"
)

func newSessionAuthService(users domain.UserService, authorization domain.AuthorizationService, sessions repository.AuthSessionRepository) *service {
	return NewAuthService(users, authorization, sessions, testAuthConfig())
}

func TestAuthServiceSignUp(t *testing.T) {
	t.Run("creates user", func(t *testing.T) {
		users := &userServiceStub{}
		auth := newSessionAuthService(users, nil, newSessionStoreStub())

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
		auth := newSessionAuthService(&userServiceStub{createErr: repository.ErrLoginAlreadyExists}, nil, newSessionStoreStub())

		ok, err := auth.SignUp(context.Background(), SignUpRequest{Login: "player", Password: "secret"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Fatal("expected duplicate login to return false")
		}
	})

	t.Run("rejects empty credentials", func(t *testing.T) {
		auth := newSessionAuthService(&userServiceStub{}, nil, newSessionStoreStub())

		ok, err := auth.SignUp(context.Background(), SignUpRequest{Login: " ", Password: ""})
		if !errors.Is(err, ErrInvalidSignUp) {
			t.Fatalf("expected ErrInvalidSignUp, got %v", err)
		}
		if ok {
			t.Fatal("expected failed registration")
		}
	})
}

func TestAuthServiceSignInCreatesSession(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	store := newSessionStoreStub()
	authz := &authorizationServiceStub{}
	auth := newSessionAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, authz, store)

	response, err := auth.SignIn(context.Background(), SessionRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.UserUUID != "user-1" || response.SessionID == "" || response.ExpiresAt.IsZero() {
		t.Fatalf("unexpected session response: %#v", response)
	}

	session, err := store.FindActiveSessionByRefreshJTIHash(context.Background(), hashTokenID(response.SessionID))
	if err != nil {
		t.Fatalf("expected active session, got %v", err)
	}
	if session.UserUUID != "user-1" {
		t.Fatalf("unexpected stored session: %#v", session)
	}
	if authz.grantDefaultRoleUUID != "user-1" {
		t.Fatalf("expected default role grant for user-1, got %q", authz.grantDefaultRoleUUID)
	}
}

func TestAuthServiceSignInMigratesPlaintextPassword(t *testing.T) {
	users := &userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: "secret"},
	}
	auth := newSessionAuthService(users, nil, newSessionStoreStub())

	response, err := auth.SignIn(context.Background(), SessionRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.UserUUID != "user-1" || response.SessionID == "" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if users.updatedUUID != "user-1" {
		t.Fatalf("expected plaintext password migration, got updated=%q", users.updatedUUID)
	}
	if !isPasswordHash("secret", users.user.Password) {
		t.Fatal("expected migrated password hash")
	}
}

func TestAuthServiceSignInRejectsInvalidCredentials(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	auth := newSessionAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, nil, newSessionStoreStub())

	_, err = auth.SignIn(context.Background(), SessionRequest{Login: "player", Password: "wrong"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthServiceSignInRejectsEmptyCredentials(t *testing.T) {
	auth := newSessionAuthService(&userServiceStub{}, nil, newSessionStoreStub())

	_, err := auth.SignIn(context.Background(), SessionRequest{Login: "player", Password: ""})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthServiceAuthenticateSession(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	store := newSessionStoreStub()
	auth := newSessionAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, nil, store)

	response, err := auth.SignIn(context.Background(), SessionRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}

	uuid, err := auth.AuthenticateSession(context.Background(), response.SessionID)
	if err != nil {
		t.Fatalf("unexpected session auth error: %v", err)
	}
	if uuid != "user-1" {
		t.Fatalf("expected user-1, got %q", uuid)
	}

	users := &userServiceStub{getErr: domain.ErrUserNotFound}
	auth = newSessionAuthService(users, nil, store)
	if _, err := auth.AuthenticateSession(context.Background(), response.SessionID); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for missing user, got %v", err)
	}
}

func TestAuthServiceRefreshSessionRotatesSession(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	store := newSessionStoreStub()
	auth := newSessionAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, nil, store)

	first, err := auth.SignIn(context.Background(), SessionRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}

	refreshed, err := auth.RefreshSession(context.Background(), first.SessionID)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.SessionID == first.SessionID {
		t.Fatal("expected rotated session id")
	}
	if !store.revoked[hashTokenID(first.SessionID)] {
		t.Fatal("expected old session to be revoked")
	}
	if _, err := store.FindActiveSessionByRefreshJTIHash(context.Background(), hashTokenID(refreshed.SessionID)); err != nil {
		t.Fatalf("expected new active session, got %v", err)
	}
}

func TestAuthServiceLogoutRevokesSession(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	store := newSessionStoreStub()
	auth := newSessionAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, nil, store)

	response, err := auth.SignIn(context.Background(), SessionRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("sign in: %v", err)
	}

	if err := auth.Logout(context.Background(), response.SessionID); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if !store.revoked[hashTokenID(response.SessionID)] {
		t.Fatal("expected session to be revoked")
	}
	if err := auth.Logout(context.Background(), response.SessionID); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on repeated logout, got %v", err)
	}
}

func TestAuthServiceLogoutAllRevokesAllSessions(t *testing.T) {
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	store := newSessionStoreStub()
	auth := newSessionAuthService(&userServiceStub{
		user: domain.User{UUID: "user-1", Login: "player", Password: hash},
	}, nil, store)

	first, err := auth.SignIn(context.Background(), SessionRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("first sign in: %v", err)
	}
	second, err := auth.SignIn(context.Background(), SessionRequest{Login: "player", Password: "secret"})
	if err != nil {
		t.Fatalf("second sign in: %v", err)
	}

	if err := auth.LogoutAll(context.Background(), first.SessionID); err != nil {
		t.Fatalf("logout all: %v", err)
	}
	if !store.revoked[hashTokenID(first.SessionID)] || !store.revoked[hashTokenID(second.SessionID)] {
		t.Fatal("expected all sessions to be revoked")
	}
	if err := auth.LogoutAll(context.Background(), first.SessionID); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on repeated logout all, got %v", err)
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

func TestAuthConfigValidateSessionsOnly(t *testing.T) {
	valid := AuthConfig{SessionCookieName: defaultSessionCookie, SessionTTL: time.Hour}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	cases := []struct {
		name   string
		config AuthConfig
	}{
		{name: "session name", config: AuthConfig{SessionTTL: time.Hour}},
		{name: "session ttl", config: AuthConfig{SessionCookieName: defaultSessionCookie}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.config.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNewAuthConfigLoadsSessionDefaults(t *testing.T) {
	t.Setenv("SESSION_COOKIE_NAME", "tic-tac-toe.session")
	t.Setenv("SESSION_TTL", "1h")

	config := NewAuthConfig()
	if config.SessionCookieName != "tic-tac-toe.session" {
		t.Fatalf("unexpected session cookie name: %#v", config)
	}
	if config.SessionTTL != time.Hour {
		t.Fatalf("unexpected session ttl: %#v", config)
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("expected valid session config, got %v", err)
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
