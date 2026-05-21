package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/internal/logging"
)

var ErrSessionNotFound = errors.New("auth session not found")

const (
	authSessionLogPrefix   = "[infrastructure/postgres/repository]"
	createAuthSessionQuery = `
INSERT INTO auth_sessions (
    refresh_jti_hash,
    user_uuid,
    created_at,
    expires_at,
    last_used_at
) VALUES ($1, $2, $3, $4, $5)`
	findActiveAuthSessionQuery = `
UPDATE auth_sessions
SET last_used_at = NOW()
WHERE refresh_jti_hash = $1
  AND revoked_at IS NULL
  AND expires_at > NOW()
RETURNING refresh_jti_hash, user_uuid, created_at, expires_at, last_used_at`
	revokeAuthSessionQuery = `
UPDATE auth_sessions
SET revoked_at = NOW()
WHERE refresh_jti_hash = $1
  AND revoked_at IS NULL`
	revokeAllAuthSessionsQuery = `
UPDATE auth_sessions
SET revoked_at = NOW()
WHERE user_uuid = $1
  AND revoked_at IS NULL`
)

type RefreshSession struct {
	RefreshJTIHash string
	UserUUID       string
	CreatedAt      time.Time
	ExpiresAt      time.Time
	LastUsedAt     time.Time
}

type AuthSessionRepository interface {
	CreateSession(ctx context.Context, session RefreshSession) error
	FindActiveSessionByRefreshJTIHash(ctx context.Context, refreshJTIHash string) (RefreshSession, error)
	RevokeSession(ctx context.Context, refreshJTIHash string) error
	RevokeSessionsByUserUUID(ctx context.Context, userUUID string) (int64, error)
}

type PostgresAuthSessionRepository struct {
	db datasource.Database
}

func NewAuthSessionRepository(db datasource.Database) AuthSessionRepository {
	return &PostgresAuthSessionRepository{db: db}
}

func (r *PostgresAuthSessionRepository) CreateSession(ctx context.Context, session RefreshSession) error {
	if session.RefreshJTIHash == "" || session.UserUUID == "" {
		return fmt.Errorf("auth session must have refresh hash and user uuid")
	}

	args := []any{session.RefreshJTIHash, session.UserUUID, session.CreatedAt, session.ExpiresAt, session.LastUsedAt}
	logAuthSession("create auth session", "user=%q", session.UserUUID)
	_, err := r.db.Exec(ctx, createAuthSessionQuery, args...)
	if err != nil {
		logAuthSession("create auth session failed", "user=%q: %v", session.UserUUID, err)
		return err
	}

	logAuthSession("create auth session ok", "user=%q", session.UserUUID)
	return nil
}

func (r *PostgresAuthSessionRepository) FindActiveSessionByRefreshJTIHash(ctx context.Context, refreshJTIHash string) (RefreshSession, error) {
	logAuthSession("find auth session", "by_refresh_hash=true")

	var scannedRefreshJTIHash sql.NullString
	var scannedUserUUID sql.NullString
	var createdAt sql.NullTime
	var expiresAt sql.NullTime
	var lastUsedAt sql.NullTime
	err := r.db.QueryRow(ctx, findActiveAuthSessionQuery, refreshJTIHash).Scan(
		&scannedRefreshJTIHash,
		&scannedUserUUID,
		&createdAt,
		&expiresAt,
		&lastUsedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		logAuthSession("find auth session not found", "by_refresh_hash=true")
		return RefreshSession{}, ErrSessionNotFound
	}
	if err != nil {
		logAuthSession("find auth session failed", "%v", err)
		return RefreshSession{}, err
	}

	session, err := buildRefreshSession(scannedRefreshJTIHash, scannedUserUUID, createdAt, expiresAt, lastUsedAt)
	if err != nil {
		logAuthSession("find auth session invalid row", "%v", err)
		return RefreshSession{}, err
	}

	logAuthSession("find auth session ok", "user=%q", session.UserUUID)
	return session, nil
}

func (r *PostgresAuthSessionRepository) RevokeSession(ctx context.Context, refreshJTIHash string) error {
	logAuthSession("revoke auth session", "by_refresh_hash=true")

	result, err := r.db.Exec(ctx, revokeAuthSessionQuery, refreshJTIHash)
	if err != nil {
		logAuthSession("revoke auth session failed", "%v", err)
		return err
	}
	if result.RowsAffected() == 0 {
		logAuthSession("revoke auth session not found", "by_refresh_hash=true")
		return ErrSessionNotFound
	}

	logAuthSession("revoke auth session ok", "by_refresh_hash=true")
	return nil
}

func (r *PostgresAuthSessionRepository) RevokeSessionsByUserUUID(ctx context.Context, userUUID string) (int64, error) {
	logAuthSession("revoke auth sessions", "user=%q", userUUID)

	result, err := r.db.Exec(ctx, revokeAllAuthSessionsQuery, userUUID)
	if err != nil {
		logAuthSession("revoke auth sessions failed", "user=%q: %v", userUUID, err)
		return 0, err
	}

	rows := result.RowsAffected()
	logAuthSession("revoke auth sessions ok", "user=%q rows=%d", userUUID, rows)
	return rows, nil
}

func logAuthSession(action string, format string, args ...any) {
	log.Printf(authSessionLogPrefix+" "+action+" "+format, args...)
}

func buildRefreshSession(refreshJTIHash sql.NullString, userUUID sql.NullString, createdAt sql.NullTime, expiresAt sql.NullTime, lastUsedAt sql.NullTime) (RefreshSession, error) {
	hash, err := requiredString("auth_sessions.refresh_jti_hash", refreshJTIHash)
	if err != nil {
		return RefreshSession{}, err
	}
	uuid, err := requiredString("auth_sessions.user_uuid", userUUID)
	if err != nil {
		return RefreshSession{}, err
	}
	created, err := requiredTime("auth_sessions.created_at", createdAt)
	if err != nil {
		return RefreshSession{}, err
	}
	expires, err := requiredTime("auth_sessions.expires_at", expiresAt)
	if err != nil {
		return RefreshSession{}, err
	}
	lastUsed, err := requiredTime("auth_sessions.last_used_at", lastUsedAt)
	if err != nil {
		return RefreshSession{}, err
	}
	return RefreshSession{
		RefreshJTIHash: hash,
		UserUUID:       uuid,
		CreatedAt:      created,
		ExpiresAt:      expires,
		LastUsedAt:     lastUsed,
	}, nil
}
