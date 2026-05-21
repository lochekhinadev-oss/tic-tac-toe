package di

import (
	"go.uber.org/fx"

	"tic-tac-toe/app/application"
	"tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/infrastructure/postgres/repository"
	"tic-tac-toe/internal/transport/http/handler"
	authhandler "tic-tac-toe/internal/transport/http/handler/auth"
	gamehandler "tic-tac-toe/internal/transport/http/handler/game"
	userhandler "tic-tac-toe/internal/transport/http/handler/user"
	"tic-tac-toe/internal/transport/http/middleware"
)

var ConfigModule = fx.Module(
	"config",
	fx.Provide(
		datasource.NewDatabaseConfig,
		auth.NewAuthConfig,
		NewHTTPConfig,
	),
	fx.Invoke(ValidateConfigs),
)

var ApplicationModule = fx.Module(
	"application",
	fx.Provide(
		fx.Annotate(
			application.NewGameService,
			fx.As(new(handler.GameLogic)),
		),
		application.NewUserService,
	),
)

var InfrastructureModule = fx.Module(
	"infrastructure",
	fx.Provide(
		datasource.NewDatabase,
		fx.Annotate(
			repository.NewGameRepository,
			fx.As(new(handler.GameStorage)),
		),
		repository.NewUserRepository,
		repository.NewAuthSessionRepository,
		auth.NewJwtProvider,
		fx.Annotate(
			auth.NewAuthService,
			fx.As(new(auth.AuthService)),
		),
	),
)

var WebModule = fx.Module(
	"web",
	fx.Provide(
		NewTokenAuthenticator,
		gamehandler.New,
		authhandler.New,
		userhandler.New,
		middleware.NewUserAuthenticator,
		NewRouter,
	),
)

var HTTPModule = fx.Module(
	"http",
	fx.Provide(NewHTTPServer),
	fx.Invoke(RegisterHTTPServer),
)

var LifecycleModule = fx.Module(
	"lifecycle",
	fx.Invoke(RegisterDatabaseLifecycle),
)

var AppModule = fx.Options(
	ConfigModule,
	ApplicationModule,
	InfrastructureModule,
	WebModule,
	HTTPModule,
	LifecycleModule,
)

func NewApp(options ...fx.Option) *fx.App {
	appOptions := []fx.Option{AppModule}
	appOptions = append(appOptions, options...)
	return fx.New(appOptions...)
}

func NewTokenAuthenticator(authService auth.AuthService) middleware.TokenAuthenticator {
	return authService
}
