package repository

const gameColumns = `uuid, field, mode, state, created_at, next_player_uuid, winner_uuid, player_x_uuid, player_o_uuid`

const saveGameQuery = `
		INSERT INTO games (
			uuid, field, mode, state, created_at, next_player_uuid, winner_uuid, player_x_uuid, player_o_uuid
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (uuid) DO UPDATE SET field = EXCLUDED.field
			, mode = EXCLUDED.mode
			, state = EXCLUDED.state
			, next_player_uuid = EXCLUDED.next_player_uuid
			, winner_uuid = EXCLUDED.winner_uuid
			, player_x_uuid = EXCLUDED.player_x_uuid
			, player_o_uuid = EXCLUDED.player_o_uuid
	`

const saveGameIfUnchangedQuery = `
		UPDATE games
		SET field = $2::jsonb
			, mode = $3
			, state = $4
			, next_player_uuid = $5
			, winner_uuid = $6
			, player_x_uuid = $7
			, player_o_uuid = $8
		WHERE uuid = $1
		  AND field = $9::jsonb
		  AND mode = $10
		  AND state = $11
		  AND next_player_uuid = $12
		  AND winner_uuid = $13
		  AND player_x_uuid = $14
		  AND player_o_uuid = $15
		`

const applyCompletedGameStatsQuery = `
	WITH applied AS (
		INSERT INTO game_stats_applied (game_uuid)
		VALUES ($1)
		ON CONFLICT (game_uuid) DO NOTHING
		RETURNING game_uuid
	),
	player_results AS (
		SELECT $2::text AS user_uuid,
		       CASE WHEN $4 = 'player_wins' AND $5 = $2 THEN 1 ELSE 0 END AS wins,
		       CASE WHEN $4 = 'player_wins' AND $5 <> $2 THEN 1 ELSE 0 END AS losses,
		       CASE WHEN $4 = 'draw' THEN 1 ELSE 0 END AS draws
		WHERE EXISTS (SELECT 1 FROM applied)
		  AND $2 <> ''
		  AND $2 <> 'computer'
		UNION ALL
		SELECT $3::text AS user_uuid,
		       CASE WHEN $4 = 'player_wins' AND $5 = $3 THEN 1 ELSE 0 END AS wins,
		       CASE WHEN $4 = 'player_wins' AND $5 <> $3 THEN 1 ELSE 0 END AS losses,
		       CASE WHEN $4 = 'draw' THEN 1 ELSE 0 END AS draws
		WHERE EXISTS (SELECT 1 FROM applied)
		  AND $3 <> ''
		  AND $3 <> 'computer'
	)
	INSERT INTO player_stats (user_uuid, wins, losses, draws, completed_games, updated_at)
	SELECT user_uuid, wins, losses, draws, 1, now()
	FROM player_results
	ON CONFLICT (user_uuid) DO UPDATE SET
		wins = player_stats.wins + EXCLUDED.wins,
		losses = player_stats.losses + EXCLUDED.losses,
		draws = player_stats.draws + EXCLUDED.draws,
		completed_games = player_stats.completed_games + EXCLUDED.completed_games,
		updated_at = now()
	`

const getGameQuery = `SELECT ` + gameColumns + `
	 FROM games WHERE uuid = $1`

const listActiveGamesQuery = `SELECT ` + gameColumns + `
	 FROM games
	 WHERE mode = $1 AND state = $2 AND player_o_uuid = ''
	 ORDER BY uuid`

const listCompletedGamesByUserUUIDQuery = `SELECT ` + gameColumns + `
	 FROM games
	 WHERE (state = $2 AND winner_uuid = $1)
	    OR (state = $3 AND (player_x_uuid = $1 OR player_o_uuid = $1))
	 ORDER BY created_at DESC, uuid`

const listTopPlayersQuery = `
	SELECT users.uuid,
	       users.login,
	       player_stats.win_ratio
	FROM player_stats
	JOIN users ON users.uuid = player_stats.user_uuid
	WHERE player_stats.completed_games > 0
	ORDER BY win_ratio DESC, users.uuid
	LIMIT $1`

const joinGameQuery = `UPDATE games
	 SET player_o_uuid = $2,
	     state = $3,
	     next_player_uuid = player_x_uuid
	 WHERE uuid = $1
	   AND mode = $4
	   AND state = $5
	   AND player_o_uuid = ''
	   AND player_x_uuid <> $2
	 RETURNING ` + gameColumns
