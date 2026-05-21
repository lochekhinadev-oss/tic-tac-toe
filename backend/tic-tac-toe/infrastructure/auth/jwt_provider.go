package auth

import (
	"crypto/rsa"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"tic-tac-toe/app/domain"
)

const (
	tokenTypeBearer = "Bearer"
	jwtTypeAccess   = "access"
	jwtTypeRefresh  = "refresh"
)

var ErrInvalidToken = errors.New("invalid token")

type JwtProvider struct {
	privateKey *rsa.PrivateKey
	publicKeys map[string]*rsa.PublicKey
	keyID      string
	issuer     string
	audience   string
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

type jwtClaims struct {
	UUID string `json:"uuid"`
	Type string `json:"typ"`
	jwt.RegisteredClaims
}

func NewJwtProvider(config AuthConfig) *JwtProvider {
	privateKey, _ := parseRSAPrivateKey(config.JWTPrivateKeyPEM)
	publicKey, _ := parseRSAPublicKeyOrCertificate(config.JWTPublicKeyPEM, config.JWTPublicCertPEM)

	publicKeys := map[string]*rsa.PublicKey{}
	if publicKey != nil {
		publicKeys[config.JWTKeyID] = publicKey
	}
	previousKeys, _ := parseJWTPublicKeySpecs(config.JWTPreviousPublicKeys)
	for _, key := range previousKeys {
		publicKeys[key.keyID] = key.key
	}

	return &JwtProvider{
		privateKey: privateKey,
		publicKeys: publicKeys,
		keyID:      config.JWTKeyID,
		issuer:     config.JWTIssuer,
		audience:   config.JWTAudience,
		accessTTL:  config.AccessTokenTTL,
		refreshTTL: config.RefreshTokenTTL,
		now:        time.Now,
	}
}

func (p *JwtProvider) GenerateAccessToken(user domain.User) (string, error) {
	return p.generateToken(user, jwtTypeAccess, p.accessTTL)
}

func (p *JwtProvider) GenerateRefreshToken(user domain.User) (string, error) {
	return p.generateToken(user, jwtTypeRefresh, p.refreshTTL)
}

func (p *JwtProvider) ValidateAccessToken(token string) error {
	_, err := p.parseToken(token, jwtTypeAccess)
	return err
}

func (p *JwtProvider) ValidateRefreshToken(token string) error {
	_, err := p.parseToken(token, jwtTypeRefresh)
	return err
}

func (p *JwtProvider) UUIDFromToken(token string) (string, error) {
	claims, err := p.parseToken(token, jwtTypeAccess)
	if err != nil {
		return "", err
	}
	return claims.UUID, nil
}

func (p *JwtProvider) generateToken(user domain.User, jwtType string, ttl time.Duration) (string, error) {
	if strings.TrimSpace(user.UUID) == "" {
		return "", ErrInvalidToken
	}
	if strings.TrimSpace(p.keyID) == "" || strings.TrimSpace(p.issuer) == "" || strings.TrimSpace(p.audience) == "" {
		return "", ErrInvalidToken
	}
	if p.privateKey == nil {
		return "", ErrInvalidToken
	}

	now := p.now().UTC()
	claims := jwtClaims{
		UUID: user.UUID,
		Type: jwtType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.UUID,
			Issuer:    p.issuer,
			Audience:  jwt.ClaimStrings{p.audience},
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = p.keyID
	return token.SignedString(p.privateKey)
}

func (p *JwtProvider) parseToken(token string, expectedType string) (jwtClaims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return jwtClaims{}, ErrInvalidToken
	}

	claims := jwtClaims{}
	parsed, err := jwt.ParseWithClaims(token, &claims, func(parsed *jwt.Token) (any, error) {
		method, ok := parsed.Method.(*jwt.SigningMethodRSA)
		if !ok || method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, ErrInvalidToken
		}
		kid, _ := parsed.Header["kid"].(string)
		if kid == "" {
			kid = p.keyID
		}
		key, ok := p.publicKeys[kid]
		if !ok || key == nil {
			return nil, ErrInvalidToken
		}
		return key, nil
	})
	if err != nil || parsed == nil || !parsed.Valid {
		return jwtClaims{}, ErrInvalidToken
	}
	if strings.TrimSpace(claims.UUID) == "" || strings.TrimSpace(claims.Subject) == "" || claims.Subject != claims.UUID {
		return jwtClaims{}, ErrInvalidToken
	}
	if claims.Issuer != p.issuer {
		return jwtClaims{}, ErrInvalidToken
	}
	if !claims.VerifyAudience(p.audience, true) {
		return jwtClaims{}, ErrInvalidToken
	}
	if claims.ID == "" {
		return jwtClaims{}, ErrInvalidToken
	}
	if expectedType != "" && claims.Type != expectedType {
		return jwtClaims{}, ErrInvalidToken
	}

	return claims, nil
}
