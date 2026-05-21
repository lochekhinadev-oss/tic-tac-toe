package di

import (
	"tic-tac-toe/infrastructure/postgres/datasource"
	authhandler "tic-tac-toe/internal/transport/http/handler/auth"
	gamehandler "tic-tac-toe/internal/transport/http/handler/game"
	userhandler "tic-tac-toe/internal/transport/http/handler/user"
	"tic-tac-toe/internal/transport/http/middleware"

	"github.com/go-chi/chi/v5"
)

func NewRouter(gameHandler *gamehandler.Handler, authHandler *authhandler.Handler, userHandler *userhandler.Handler, authenticator *middleware.UserAuthenticator, db datasource.Database) chi.Router {
	router := chi.NewRouter()

	registerRouterMiddleware(router)
	registerSystemRoutes(router, db)
	registerPublicRoutes(router, authHandler)
	registerProtectedRoutes(router, gameHandler, userHandler, authenticator)

	return router
}
