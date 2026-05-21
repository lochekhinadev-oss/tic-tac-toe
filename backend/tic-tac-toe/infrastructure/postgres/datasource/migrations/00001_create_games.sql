-- +goose Up
CREATE TABLE IF NOT EXISTS games (
    uuid TEXT PRIMARY KEY,
    field JSONB NOT NULL,
    mode TEXT NOT NULL DEFAULT 'computer',
    state TEXT NOT NULL DEFAULT 'player_to_move',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    next_player_uuid TEXT NOT NULL DEFAULT '',
    winner_uuid TEXT NOT NULL DEFAULT '',
    player_x_uuid TEXT NOT NULL DEFAULT '',
    player_o_uuid TEXT NOT NULL DEFAULT ''
);

-- +goose Down
DROP TABLE IF EXISTS games;
