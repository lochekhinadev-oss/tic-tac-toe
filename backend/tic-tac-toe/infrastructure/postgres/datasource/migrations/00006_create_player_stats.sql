-- +goose Up
CREATE TABLE IF NOT EXISTS player_stats (
    user_uuid TEXT PRIMARY KEY REFERENCES users(uuid) ON DELETE CASCADE,
    wins BIGINT NOT NULL DEFAULT 0,
    losses BIGINT NOT NULL DEFAULT 0,
    draws BIGINT NOT NULL DEFAULT 0,
    completed_games BIGINT NOT NULL DEFAULT 0,
    win_ratio DOUBLE PRECISION GENERATED ALWAYS AS (
        CASE
            WHEN losses + draws = 0 THEN wins::DOUBLE PRECISION
            ELSE wins::DOUBLE PRECISION / (losses + draws)
        END
    ) STORED,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS game_stats_applied (
    game_uuid TEXT PRIMARY KEY REFERENCES games(uuid) ON DELETE CASCADE,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

WITH player_results AS (
    SELECT
        uuid AS game_uuid,
        player_x_uuid AS user_uuid,
        CASE WHEN winner_uuid = player_x_uuid THEN 1 ELSE 0 END AS wins,
        CASE WHEN state = 'player_wins' AND winner_uuid <> player_x_uuid THEN 1 ELSE 0 END AS losses,
        CASE WHEN state = 'draw' THEN 1 ELSE 0 END AS draws
    FROM games
    WHERE state IN ('player_wins', 'draw')
      AND player_x_uuid <> ''
      AND player_x_uuid <> 'computer'
    UNION ALL
    SELECT
        uuid AS game_uuid,
        player_o_uuid AS user_uuid,
        CASE WHEN winner_uuid = player_o_uuid THEN 1 ELSE 0 END AS wins,
        CASE WHEN state = 'player_wins' AND winner_uuid <> player_o_uuid THEN 1 ELSE 0 END AS losses,
        CASE WHEN state = 'draw' THEN 1 ELSE 0 END AS draws
    FROM games
    WHERE state IN ('player_wins', 'draw')
      AND player_o_uuid <> ''
      AND player_o_uuid <> 'computer'
),
inserted_games AS (
    INSERT INTO game_stats_applied (game_uuid)
    SELECT DISTINCT game_uuid
    FROM player_results
    ON CONFLICT (game_uuid) DO NOTHING
    RETURNING game_uuid
),
stats AS (
    SELECT
        player_results.user_uuid,
        SUM(player_results.wins) AS wins,
        SUM(player_results.losses) AS losses,
        SUM(player_results.draws) AS draws,
        COUNT(*) AS completed_games
    FROM player_results
    JOIN inserted_games ON inserted_games.game_uuid = player_results.game_uuid
    GROUP BY player_results.user_uuid
)
INSERT INTO player_stats (user_uuid, wins, losses, draws, completed_games, updated_at)
SELECT user_uuid, wins, losses, draws, completed_games, now()
FROM stats
ON CONFLICT (user_uuid) DO UPDATE SET
    wins = player_stats.wins + EXCLUDED.wins,
    losses = player_stats.losses + EXCLUDED.losses,
    draws = player_stats.draws + EXCLUDED.draws,
    completed_games = player_stats.completed_games + EXCLUDED.completed_games,
    updated_at = now();

CREATE INDEX IF NOT EXISTS player_stats_leaderboard_idx
    ON player_stats (win_ratio DESC, user_uuid)
    WHERE completed_games > 0;

-- +goose Down
DROP INDEX IF EXISTS player_stats_leaderboard_idx;
DROP TABLE IF EXISTS game_stats_applied;
DROP TABLE IF EXISTS player_stats;
