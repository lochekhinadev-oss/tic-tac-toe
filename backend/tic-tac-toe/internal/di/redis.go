package di

import (
	"context"

	"go.uber.org/fx"

	"tic-tac-toe/infrastructure/rediscache"
)

func RegisterRedisLifecycle(lifecycle fx.Lifecycle, cache rediscache.LeaderboardCache) {
	if cache == nil {
		return
	}

	lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			return cache.Close()
		},
	})
}
