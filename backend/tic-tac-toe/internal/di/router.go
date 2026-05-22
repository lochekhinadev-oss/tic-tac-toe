package di

import (
	"github.com/go-chi/chi/v5"

	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/internal/transport/http/handler"
	"tic-tac-toe/internal/transport/http/middleware"
)

func NewRouter(gameHandler *handler.GameHandler, authHandler *handler.AuthHandler, userHandler *handler.UserHandler, authenticator *middleware.UserAuthenticator, db datasource.Database) chi.Router {
	router := chi.NewRouter()

	registerRouterMiddleware(router)
	registerSystemRoutes(router, db)
	registerPublicRoutes(router, authHandler)
	registerProtectedRoutes(router, gameHandler, userHandler, authenticator)

	return router
}
