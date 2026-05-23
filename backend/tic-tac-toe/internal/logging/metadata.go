package log

import "tic-tac-toe/internal/config"

const (
	defaultServiceName = "tic-tac-toe"
	defaultVersion     = "dev"
	defaultCommit      = "unknown"
	defaultEnvironment = "development"
)

func Fields() []any {
	return []any{
		"service", config.String("SERVICE_NAME", defaultServiceName),
		"version", config.String("VERSION", defaultVersion),
		"commit", config.String("COMMIT", defaultCommit),
		"env", config.String("APP_ENV", defaultEnvironment),
	}
}
