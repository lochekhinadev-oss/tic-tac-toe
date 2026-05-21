-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    uuid TEXT PRIMARY KEY,
    login TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS users;
