package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"tic-tac-toe/app/domain"
)

func TestJwtProviderRejectsUnexpectedSigningAlgorithm(t *testing.T) {
	provider := NewJwtProvider(testAuthConfig())
	now := time.Now().UTC()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims{
		UUID: "user-1",
		Type: jwtTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	})
	signedToken, err := token.SignedString([]byte("hmac-secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	err = provider.ValidateAccessToken(signedToken)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJwtProviderAcceptsPreviousSigningKey(t *testing.T) {
	currentPrivate, currentPublic := newDevelopmentKeyPair()
	legacyPrivate, legacyPublic := newDevelopmentKeyPair()
	legacyKey, err := parseRSAPrivateKey(legacyPrivate)
	if err != nil {
		t.Fatalf("parse legacy private key: %v", err)
	}
	provider := NewJwtProvider(newTestAuthConfig(currentPrivate, currentPublic, "current", "legacy:"+base64.StdEncoding.EncodeToString([]byte(legacyPublic))))

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwtClaims{
		UUID: "user-1",
		Type: jwtTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			Issuer:    "issuer",
			Audience:  jwt.ClaimStrings{"audience"},
			ID:        "token-id",
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().UTC()),
		},
	})
	token.Header["kid"] = "legacy"
	signedToken, err := token.SignedString(legacyKey)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	if err := provider.ValidateAccessToken(signedToken); err != nil {
		t.Fatalf("expected legacy key token to validate, got %v", err)
	}
}

func TestJwtProviderValidateRefreshToken(t *testing.T) {
	provider := NewJwtProvider(testAuthConfig())
	token, err := provider.GenerateRefreshToken(domain.User{UUID: "user-1"})
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	if err := provider.ValidateRefreshToken(token); err != nil {
		t.Fatalf("expected refresh token to validate, got %v", err)
	}
}

func TestJwtProviderValidatesTokenWithCertificatePublicKey(t *testing.T) {
	privateKeyPEM, certPEM := newTestCertificate(t)
	provider := NewJwtProvider(AuthConfig{
		JWTPrivateKeyPEM: privateKeyPEM,
		JWTPublicCertPEM: certPEM,
		JWTKeyID:         "cert-key",
		JWTIssuer:        "issuer",
		JWTAudience:      "audience",
		AccessTokenTTL:   time.Hour,
		RefreshTokenTTL:  time.Hour,
	})

	token, err := provider.GenerateAccessToken(domain.User{UUID: "user-1"})
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	if err := provider.ValidateAccessToken(token); err != nil {
		t.Fatalf("expected token to validate with certificate public key, got %v", err)
	}
}

func TestJwtProviderRejectsInvalidInputs(t *testing.T) {
	provider := NewJwtProvider(testAuthConfig())

	if _, err := provider.GenerateAccessToken(domain.User{}); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token for empty user uuid, got %v", err)
	}
	if _, err := provider.UUIDFromToken(" "); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token for empty token, got %v", err)
	}
}

func TestJwtProviderRejectsInvalidClaims(t *testing.T) {
	privateKey, publicKey := newDevelopmentKeyPair()
	provider := NewJwtProvider(newTestAuthConfig(privateKey, publicKey, "kid", ""))
	now := time.Now().UTC()

	cases := []struct {
		name  string
		token string
	}{
		{
			name: "issuer",
			token: mustSignJWT(t, provider, jwtClaims{
				UUID: "user-1",
				Type: jwtTypeAccess,
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "user-1",
					Issuer:    "bad-issuer",
					Audience:  jwt.ClaimStrings{"audience"},
					ID:        "token-id",
					IssuedAt:  jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
					NotBefore: jwt.NewNumericDate(now),
				},
			}),
		},
		{
			name: "audience",
			token: mustSignJWT(t, provider, jwtClaims{
				UUID: "user-1",
				Type: jwtTypeAccess,
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "user-1",
					Issuer:    "issuer",
					Audience:  jwt.ClaimStrings{"bad-audience"},
					ID:        "token-id",
					IssuedAt:  jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
					NotBefore: jwt.NewNumericDate(now),
				},
			}),
		},
		{
			name: "subject mismatch",
			token: mustSignJWT(t, provider, jwtClaims{
				UUID: "user-1",
				Type: jwtTypeAccess,
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "user-2",
					Issuer:    "issuer",
					Audience:  jwt.ClaimStrings{"audience"},
					ID:        "token-id",
					IssuedAt:  jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
					NotBefore: jwt.NewNumericDate(now),
				},
			}),
		},
		{
			name: "missing id",
			token: mustSignJWT(t, provider, jwtClaims{
				UUID: "user-1",
				Type: jwtTypeAccess,
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:   "user-1",
					Issuer:    "issuer",
					Audience:  jwt.ClaimStrings{"audience"},
					IssuedAt:  jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
					NotBefore: jwt.NewNumericDate(now),
				},
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := provider.ValidateAccessToken(tc.token); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("expected invalid token, got %v", err)
			}
		})
	}
}

func TestJwtProviderRejectsWrongTokenType(t *testing.T) {
	provider := NewJwtProvider(testAuthConfig())
	user := domain.User{UUID: "user-1"}
	accessToken, err := provider.GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	if err := provider.ValidateRefreshToken(accessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token for wrong type, got %v", err)
	}

	refreshToken, err := provider.GenerateRefreshToken(user)
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	if _, err := provider.UUIDFromToken(refreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected UUIDFromToken to reject refresh token, got %v", err)
	}
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
