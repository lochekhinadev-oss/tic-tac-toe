package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"strings"

	_ "tic-tac-toe/internal/logging"
)

const authLogPrefix = "[infrastructure/auth]"

func logAuth(format string, args ...any) {
	log.Printf(authLogPrefix+" "+format, args...)
}

const bearerPrefix = "Bearer "

func parseBearerAuthorizationHeader(header string) (string, error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", ErrInvalidAuthHeader
	}

	if !strings.HasPrefix(strings.ToLower(header), strings.ToLower(bearerPrefix)) {
		return "", ErrInvalidAuthHeader
	}

	token := strings.TrimSpace(header[len(bearerPrefix):])
	if token == "" {
		return "", ErrInvalidAuthHeader
	}
	return token, nil
}

type jwtPublicKeySpec struct {
	keyID string
	key   *rsa.PublicKey
}

func parseJWTPublicKeySpecs(raw string) ([]jwtPublicKeySpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	specs := make([]jwtPublicKeySpec, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		keyID, encodedPEM, ok := strings.Cut(part, ":")
		if !ok {
			return nil, fmt.Errorf("invalid key spec %q, expected kid:base64-pem", part)
		}
		keyID = strings.TrimSpace(keyID)
		encodedPEM = strings.TrimSpace(encodedPEM)
		if keyID == "" || encodedPEM == "" {
			return nil, fmt.Errorf("invalid key spec %q, key id and public key must not be empty", part)
		}

		pemBytes, err := base64.StdEncoding.DecodeString(encodedPEM)
		if err != nil {
			return nil, fmt.Errorf("decode key %q: %w", keyID, err)
		}
		key, err := parseRSAPublicKey(string(pemBytes))
		if err != nil {
			key, err = parseRSAPublicKeyFromCertificate(string(pemBytes))
		}
		if err != nil {
			return nil, fmt.Errorf("parse key %q: %w", keyID, err)
		}
		specs = append(specs, jwtPublicKeySpec{keyID: keyID, key: key})
	}
	return specs, nil
}

func parseRSAPrivateKey(raw string) (*rsa.PrivateKey, error) {
	block, err := decodePEMBlock(raw)
	if err != nil {
		return nil, err
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unsupported private key format")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key must be RSA")
	}
	return key, nil
}

func parseRSAPublicKeyOrCertificate(publicKeyPEM string, certPEM string) (*rsa.PublicKey, error) {
	if strings.TrimSpace(certPEM) != "" {
		return parseRSAPublicKeyFromCertificate(certPEM)
	}
	return parseRSAPublicKey(publicKeyPEM)
}

func parseRSAPublicKey(raw string) (*rsa.PublicKey, error) {
	block, err := decodePEMBlock(raw)
	if err != nil {
		return nil, err
	}

	if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unsupported public key format")
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key must be RSA")
	}
	return key, nil
}

func parseRSAPublicKeyFromCertificate(raw string) (*rsa.PublicKey, error) {
	block, err := decodePEMBlock(raw)
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}
	key, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("certificate public key must be RSA")
	}
	return key, nil
}

func decodePEMBlock(raw string) (*pem.Block, error) {
	raw = strings.ReplaceAll(strings.TrimSpace(raw), `\n`, "\n")
	if raw == "" {
		return nil, fmt.Errorf("pem must not be empty")
	}

	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, fmt.Errorf("invalid pem")
	}
	return block, nil
}
