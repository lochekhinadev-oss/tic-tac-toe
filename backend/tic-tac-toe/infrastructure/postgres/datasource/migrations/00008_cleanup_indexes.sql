-- +goose Up
CREATE INDEX IF NOT EXISTS users_deleted_at_idx
    ON users (deleted_at)
    WHERE deleted_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS auth_sessions_cleanup_idx
    ON auth_sessions (revoked_at, expires_at);

CREATE INDEX IF NOT EXISTS games_cleanup_idx
    ON games (state, created_at);

-- +goose Down
DROP INDEX IF EXISTS games_cleanup_idx;
DROP INDEX IF EXISTS auth_sessions_cleanup_idx;
DROP INDEX IF EXISTS users_deleted_at_idx;
