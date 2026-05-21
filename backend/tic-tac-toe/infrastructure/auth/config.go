package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"tic-tac-toe/internal/config"
)

const (
	appEnvProduction       = "production"
	defaultJWTKeyID        = "tic-tac-toe-main"
	defaultJWTIssuer       = "tic-tac-toe"
	defaultJWTAudience     = "tic-tac-toe-api"
	defaultAccessTokenTTL  = 15 * time.Minute
	defaultRefreshTokenTTL = 7 * 24 * time.Hour
)

type AuthConfig struct {
	AppEnv                string
	JWTPrivateKeyPath     string
	JWTPublicKeyPath      string
	JWTPrivateKeyPEM      string
	JWTPublicKeyPEM       string
	JWTPublicCertPEM      string
	JWTPreviousPublicKeys string
	JWTKeyID              string
	JWTIssuer             string
	JWTAudience           string
	AccessTokenTTL        time.Duration
	RefreshTokenTTL       time.Duration
}

func (c AuthConfig) Validate() error {
	if c.JWTPrivateKeyPath == "" && c.JWTPrivateKeyPEM == "" {
		return fmt.Errorf("JWT_PRIVATE_KEY_PATH must be set")
	}
	if c.JWTPublicKeyPath == "" && c.JWTPublicKeyPEM == "" && c.JWTPublicCertPEM == "" {
		return fmt.Errorf("JWT_PUBLIC_KEY_PATH must be set")
	}
	if c.JWTPrivateKeyPath != "" {
		if _, err := os.Stat(c.JWTPrivateKeyPath); err != nil {
			return fmt.Errorf("JWT_PRIVATE_KEY_PATH %q is not readable: %w", c.JWTPrivateKeyPath, err)
		}
	}
	if c.JWTPublicKeyPath != "" {
		if _, err := os.Stat(c.JWTPublicKeyPath); err != nil {
			return fmt.Errorf("JWT_PUBLIC_KEY_PATH %q is not readable: %w", c.JWTPublicKeyPath, err)
		}
	}
	if c.JWTPrivateKeyPEM == "" {
		return fmt.Errorf("jwt private key must not be empty")
	}
	if c.JWTPublicKeyPEM == "" && c.JWTPublicCertPEM == "" {
		return fmt.Errorf("jwt public key or certificate must not be empty")
	}
	if c.JWTKeyID == "" {
		return fmt.Errorf("jwt key id must not be empty")
	}
	if c.JWTIssuer == "" {
		return fmt.Errorf("jwt issuer must not be empty")
	}
	if c.JWTAudience == "" {
		return fmt.Errorf("jwt audience must not be empty")
	}
	if _, err := parseJWTPublicKeySpecs(c.JWTPreviousPublicKeys); err != nil {
		return fmt.Errorf("jwt previous public keys: %w", err)
	}
	if _, err := parseRSAPrivateKey(c.JWTPrivateKeyPEM); err != nil {
		return fmt.Errorf("jwt private key: %w", err)
	}
	if _, err := parseRSAPublicKeyOrCertificate(c.JWTPublicKeyPEM, c.JWTPublicCertPEM); err != nil {
		return fmt.Errorf("jwt public key: %w", err)
	}
	if c.AccessTokenTTL <= 0 {
		return fmt.Errorf("jwt access ttl must be positive")
	}
	if c.RefreshTokenTTL <= 0 {
		return fmt.Errorf("jwt refresh ttl must be positive")
	}
	return nil
}

func NewAuthConfig() AuthConfig {
	appEnv := config.String("APP_ENV", "development")
	privateKeyPEM := config.String("JWT_PRIVATE_KEY_PEM", "")
	publicKeyPEM := config.String("JWT_PUBLIC_KEY_PEM", "")
	publicCertPEM := config.String("JWT_PUBLIC_CERT_PEM", "")
	privateKeyPath := config.String("JWT_PRIVATE_KEY_PATH", "")
	publicKeyPath := config.String("JWT_PUBLIC_KEY_PATH", "")

	privateKeyPEM = valueFromEnvOrPath(privateKeyPEM, privateKeyPath)
	publicKeyPEM = valueFromEnvOrPath(publicKeyPEM, publicKeyPath)
	publicCertPEM = valueFromEnvOrFile(publicCertPEM, "JWT_PUBLIC_CERT_FILE")

	return AuthConfig{
		AppEnv:                appEnv,
		JWTPrivateKeyPath:     privateKeyPath,
		JWTPublicKeyPath:      publicKeyPath,
		JWTPrivateKeyPEM:      privateKeyPEM,
		JWTPublicKeyPEM:       publicKeyPEM,
		JWTPublicCertPEM:      publicCertPEM,
		JWTPreviousPublicKeys: config.String("JWT_PREVIOUS_PUBLIC_KEYS", ""),
		JWTKeyID:              config.String("JWT_KEY_ID", defaultJWTKeyID),
		JWTIssuer:             config.String("JWT_ISSUER", defaultJWTIssuer),
		JWTAudience:           config.String("JWT_AUDIENCE", defaultJWTAudience),
		AccessTokenTTL:        config.Duration("JWT_ACCESS_TTL", defaultAccessTokenTTL),
		RefreshTokenTTL:       config.Duration("JWT_REFRESH_TTL", defaultRefreshTokenTTL),
	}
}

func valueFromEnvOrPath(value string, path string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	if strings.TrimSpace(path) == "" {
		return ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func valueFromEnvOrFile(value string, fileKey string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}

	path := strings.TrimSpace(os.Getenv(fileKey))
	if path == "" {
		return ""
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func newDevelopmentCertificate() (string, string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", ""
	}

	now := time.Now().UTC()
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "tic-tac-toe.local"},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return "", ""
	}

	privateBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	certBlock := &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}
	return string(pem.EncodeToMemory(privateBlock)), string(pem.EncodeToMemory(certBlock))
}

func newDevelopmentKeyPair() (string, string) {
	privateKey, cert := newDevelopmentCertificate()
	publicKey, err := parseRSAPublicKeyFromCertificate(cert)
	if err != nil {
		return privateKey, ""
	}
	publicBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return privateKey, ""
	}
	return privateKey, string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicBytes}))
}
