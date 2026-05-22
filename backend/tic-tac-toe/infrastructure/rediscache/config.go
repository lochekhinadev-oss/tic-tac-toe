package rediscache

import (
	"fmt"
	"net/url"
	"strings"

	"tic-tac-toe/internal/config"
)

const defaultRedisURL = "redis://localhost:6379/0"

type Config struct {
	URL string
}

func NewConfig() Config {
	return Config{URL: config.String("REDIS_URL", defaultRedisURL)}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.URL) == "" {
		return fmt.Errorf("redis url must not be empty")
	}

	parsed, err := url.Parse(c.URL)
	if err != nil {
		return fmt.Errorf("invalid redis url: %w", err)
	}
	if parsed.Scheme != "redis" {
		return fmt.Errorf("unsupported redis url scheme: %s", parsed.Scheme)
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return fmt.Errorf("redis url must include host")
	}

	return nil
}
