package di

import (
	"go.uber.org/fx"

	"tic-tac-toe/app/application"
	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/infrastructure/postgres/repository"
	"tic-tac-toe/infrastructure/rediscache"
	"tic-tac-toe/internal/transport/http/handler"
	"tic-tac-toe/internal/transport/http/middleware"
)

var ConfigModule = fx.Module(
	"config",
	fx.Provide(
		datasource.NewDatabaseConfig,
		auth.NewAuthConfig,
		rediscache.NewConfig,
		NewHTTPConfig,
	),
	fx.Invoke(ValidateConfigs),
)

var ApplicationModule = fx.Module(
	"application",
	fx.Provide(
		fx.Annotate(
			application.NewGameService,
			fx.As(new(handler.GameCommandService)),
		),
		fx.Annotate(
			application.NewUserService,
			fx.As(new(domain.UserService)),
		),
		fx.Annotate(
			application.NewAuthorizationService,
			fx.As(new(domain.AuthorizationService)),
		),
		fx.Annotate(
			application.NewRoutePermissionPolicy,
			fx.As(new(application.RequestPermissionResolver)),
		),
		fx.Annotate(
			application.NewRequestAuthorizationService,
			fx.As(new(application.RequestAuthorizer)),
		),
		AsUserQueryService,
	),
)

var InfrastructureModule = fx.Module(
	"infrastructure",
	fx.Provide(
		datasource.NewDatabase,
		fx.Annotate(
			rediscache.NewClient,
			fx.As(new(rediscache.LeaderboardCache)),
		),
		repository.NewGameRepository,
		AsGameCommandStorage,
		AsGameQueryService,
		repository.NewUserRepository,
		repository.NewAuthorizationRepository,
		repository.NewAuthSessionRepository,
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
		handler.NewGameHandler,
		handler.NewAuthHandler,
		handler.NewUserHandler,
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
	fx.Invoke(RegisterRedisLifecycle),
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

func AsGameCommandStorage(repo *repository.GameRepository) handler.GameCommandStorage {
	return repo
}

func AsGameQueryService(repo *repository.GameRepository) handler.GameQueryService {
	return repo
}

func AsUserQueryService(userService domain.UserService) handler.UserQueryService {
	return userService
}
