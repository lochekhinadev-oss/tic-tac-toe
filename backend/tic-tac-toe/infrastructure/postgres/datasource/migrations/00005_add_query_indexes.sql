-- +goose Up
CREATE INDEX IF NOT EXISTS games_active_idx
    ON games (mode, state, uuid)
    WHERE player_o_uuid = '';

CREATE INDEX IF NOT EXISTS games_completed_winner_idx
    ON games (winner_uuid, created_at DESC, uuid)
    WHERE state = 'player_wins';

CREATE INDEX IF NOT EXISTS games_completed_player_x_draw_idx
    ON games (player_x_uuid, created_at DESC, uuid)
    WHERE state = 'draw';

CREATE INDEX IF NOT EXISTS games_completed_player_o_draw_idx
    ON games (player_o_uuid, created_at DESC, uuid)
    WHERE state = 'draw';

CREATE INDEX IF NOT EXISTS games_leaderboard_idx
    ON games (state, winner_uuid, player_x_uuid, player_o_uuid);

CREATE INDEX IF NOT EXISTS auth_sessions_active_refresh_idx
    ON auth_sessions (refresh_jti_hash, expires_at)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS auth_sessions_active_user_idx
    ON auth_sessions (user_uuid)
    WHERE revoked_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS auth_sessions_active_user_idx;
DROP INDEX IF EXISTS auth_sessions_active_refresh_idx;
DROP INDEX IF EXISTS games_leaderboard_idx;
DROP INDEX IF EXISTS games_completed_player_o_draw_idx;
DROP INDEX IF EXISTS games_completed_player_x_draw_idx;
DROP INDEX IF EXISTS games_completed_winner_idx;
DROP INDEX IF EXISTS games_active_idx;
