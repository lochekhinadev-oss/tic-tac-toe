package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"

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

func (s *userServiceStub) DeleteUser(context.Context, string) error {
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

func mustSignJWT(t *testing.T, provider *JwtProvider, claims jwtClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = provider.keyID
	signed, err := token.SignedString(provider.privateKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func newTestAuthConfig(privateKeyPEM string, publicKeyPEM string, keyID string, previousPublicKeys string) AuthConfig {
	return AuthConfig{
		JWTPrivateKeyPEM:      privateKeyPEM,
		JWTPublicKeyPEM:       publicKeyPEM,
		JWTPreviousPublicKeys: previousPublicKeys,
		JWTKeyID:              keyID,
		JWTIssuer:             "issuer",
		JWTAudience:           "audience",
		AccessTokenTTL:        time.Hour,
		RefreshTokenTTL:       time.Hour,
	}
}

func testAuthConfig() AuthConfig {
	privateKeyPEM, publicKeyPEM := newDevelopmentKeyPair()
	return newTestAuthConfig(privateKeyPEM, publicKeyPEM, "test-key", "")
}

func newTestCertificate(t *testing.T) (string, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "tic-tac-toe.test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return string(privatePEM), string(certPEM)
}
