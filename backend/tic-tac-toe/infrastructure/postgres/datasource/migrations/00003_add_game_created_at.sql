-- +goose Up
ALTER TABLE games
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE games
    DROP COLUMN IF EXISTS created_at;
