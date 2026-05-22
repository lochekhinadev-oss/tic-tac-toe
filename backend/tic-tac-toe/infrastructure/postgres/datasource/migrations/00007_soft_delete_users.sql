-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_login_key;

CREATE UNIQUE INDEX IF NOT EXISTS users_login_active_idx
    ON users (login)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS users_login_active_idx;

ALTER TABLE users ADD CONSTRAINT users_login_key UNIQUE (login);

ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;
