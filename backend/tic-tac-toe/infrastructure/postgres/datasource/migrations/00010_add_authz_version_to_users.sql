-- +goose Up
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS authz_version BIGINT NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE users
    DROP COLUMN IF EXISTS authz_version;
