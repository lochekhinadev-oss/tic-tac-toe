package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrInvalidDatabaseRow = errors.New("invalid database row")

func requiredString(field string, value sql.NullString) (string, error) {
	if !value.Valid || value.String == "" {
		return "", fmt.Errorf("%w: %s is required", ErrInvalidDatabaseRow, field)
	}
	return value.String, nil
}

func optionalString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func requiredTime(field string, value sql.NullTime) (time.Time, error) {
	if !value.Valid || value.Time.IsZero() {
		return time.Time{}, fmt.Errorf("%w: %s is required", ErrInvalidDatabaseRow, field)
	}
	return value.Time, nil
}
