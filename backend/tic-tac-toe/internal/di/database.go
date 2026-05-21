package di

import (
	"context"

	"go.uber.org/fx"

	"tic-tac-toe/infrastructure/postgres/datasource"
)

func RegisterDatabaseLifecycle(lifecycle fx.Lifecycle, db datasource.Database) {
	lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			db.Close()
			return nil
		},
	})
}
