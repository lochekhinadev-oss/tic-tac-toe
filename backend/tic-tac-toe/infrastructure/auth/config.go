package auth

import (
	"fmt"
	"strings"
	"time"

	"tic-tac-toe/internal/config"
)

const (
	defaultSessionCookie = "tic-tac-toe.session"
	defaultSessionTTL    = 7 * 24 * time.Hour
)

type AuthConfig struct {
	SessionCookieName string
	SessionTTL        time.Duration
}

func (c AuthConfig) Validate() error {
	if strings.TrimSpace(c.SessionCookieName) == "" {
		return fmt.Errorf("session cookie name must not be empty")
	}
	if c.SessionTTL <= 0 {
		return fmt.Errorf("session ttl must be positive")
	}
	return nil
}

func NewAuthConfig() AuthConfig {
	return AuthConfig{
		SessionCookieName: config.String("SESSION_COOKIE_NAME", defaultSessionCookie),
		SessionTTL:        config.Duration("SESSION_TTL", defaultSessionTTL),
	}
}
