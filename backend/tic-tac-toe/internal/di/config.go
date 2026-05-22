package di

import (
	"errors"
	"fmt"

	"tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/infrastructure/rediscache"
)

func ValidateConfigs(databaseConfig datasource.DatabaseConfig, authConfig auth.AuthConfig, redisConfig rediscache.Config, httpConfig HTTPConfig) error {
	return errors.Join(
		joinConfigError("database", databaseConfig.Validate()),
		joinConfigError("auth", authConfig.Validate()),
		joinConfigError("redis", redisConfig.Validate()),
		joinConfigError("http", httpConfig.Validate()),
	)
}

func joinConfigError(scope string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s config: %w", scope, err)
}
