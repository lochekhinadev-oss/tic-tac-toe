package di

import (
	"github.com/go-chi/chi/v5"

	"tic-tac-toe/internal/transport/http/handler"
	"tic-tac-toe/internal/transport/http/middleware"
)

func registerProtectedRoutes(router chi.Router, gameHandler *handler.GameHandler, userHandler *handler.UserHandler, authenticator *middleware.UserAuthenticator) {
	router.Group(func(protected chi.Router) {
		protected.Use(authenticator.Protect)

		protected.Route("/games", func(games chi.Router) {
			games.Post("/", gameHandler.CreateGame)
			games.Get("/", gameHandler.ListGames)
			games.Get("/history", gameHandler.ListCompletedGames)
			games.Get("/leaderboard", gameHandler.ListTopPlayers)
			games.Route("/{uuid}", func(game chi.Router) {
				game.Get("/", withUUID(gameHandler.GetGame))
				game.Post("/join", withUUID(gameHandler.JoinGame))
				game.Post("/move", withUUID(gameHandler.MakeMove))
			})
		})

		protected.Get("/users/me", userHandler.GetCurrentUser)
		protected.Get("/users/{uuid}", withUUID(userHandler.GetUser))
	})
}
