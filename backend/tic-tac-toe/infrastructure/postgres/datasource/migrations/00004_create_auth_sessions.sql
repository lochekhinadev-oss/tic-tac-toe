-- +goose Up
CREATE TABLE IF NOT EXISTS auth_sessions (
    refresh_jti_hash TEXT PRIMARY KEY,
    user_uuid TEXT NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS auth_sessions_user_uuid_idx ON auth_sessions (user_uuid);

-- +goose Down
DROP TABLE IF EXISTS auth_sessions;
