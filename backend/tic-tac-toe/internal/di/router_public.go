package di

import (
	"github.com/go-chi/chi/v5"

	"tic-tac-toe/internal/transport/http/handler"
)

func registerPublicRoutes(router chi.Router, authHandler *handler.AuthHandler) {
	router.Post("/users", authHandler.SignUp)
	router.Post("/auth/sessions", authHandler.Authenticate)
	router.Post("/auth/tokens/access", authHandler.RefreshAccessToken)
	router.Post("/auth/tokens/refresh", authHandler.RefreshRefreshToken)
	router.Delete("/auth/sessions/current", authHandler.Logout)
	router.Delete("/auth/sessions", authHandler.LogoutAll)

	// Legacy aliases kept to avoid breaking existing clients.
	router.Post("/signup", authHandler.SignUp)
	router.Post("/auth", authHandler.Authenticate)
	router.Post("/auth/refresh/access", authHandler.RefreshAccessToken)
	router.Post("/auth/refresh", authHandler.RefreshRefreshToken)
	router.Post("/auth/logout", authHandler.Logout)
	router.Post("/auth/logout/all", authHandler.LogoutAll)
}
