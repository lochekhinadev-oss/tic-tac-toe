package di

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	googleuuid "github.com/google/uuid"

	"tic-tac-toe/docs"
	"tic-tac-toe/infrastructure/postgres/datasource"
	appmetrics "tic-tac-toe/internal/metrics"
	"tic-tac-toe/internal/transport/http/handler"
	"tic-tac-toe/internal/transport/http/messages"
	"tic-tac-toe/internal/transport/http/middleware"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

const (
	maxRequestBodyBytes = 1 << 20
	swaggerHTML         = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Tic-Tac-Toe API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #f7f7f7; }
    .swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      SwaggerUIBundle({
        url: "/openapi.yaml",
        dom_id: "#swagger-ui",
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`
)

func registerRouterMiddleware(router chi.Router) {
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(middleware.SecurityHeaders)
	router.Use(middleware.BodySizeLimit(maxRequestBodyBytes))
	router.Use(middleware.RequestLogger)
	router.Use(chimiddleware.Recoverer)
	router.NotFound(notFound)
	router.MethodNotAllowed(methodNotAllowed)
}

func registerSystemRoutes(router chi.Router, db datasource.Database) {
	router.Get("/healthz", healthz)
	router.Get("/readyz", readyz(db))
	router.Get("/metrics", metrics)
	router.Get("/swagger", swaggerUI)
	router.Get("/openapi.yaml", openAPIYAML)
	router.Get("/swagger/doc.json", openAPIJSON)
}

func registerPublicRoutes(router chi.Router, authHandler *handler.AuthHandler) {
	router.Post("/users", authHandler.SignUp)
	router.Post("/auth/sessions", authHandler.Authenticate)
	router.Delete("/auth/sessions/current", authHandler.Logout)
	router.Delete("/auth/sessions", authHandler.LogoutAll)
}

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
		protected.Delete("/users/me", userHandler.DeleteCurrentUser)
		protected.Get("/users/{uuid}", withUUID(userHandler.GetUser))
	})
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	webresponse.WriteJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: "ok"})
}

func readyz(db datasource.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			webresponse.WriteInternalError(w, messages.DatabaseNotReady)
			return
		}

		webresponse.WriteJSON(w, http.StatusOK, struct {
			Status string `json:"status"`
		}{Status: "ready"})
	}
}

func metrics(w http.ResponseWriter, r *http.Request) {
	appmetrics.Handler().ServeHTTP(w, r)
}

func swaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(swaggerHTML))
}

func openAPIYAML(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(docs.OpenAPIYAML)
}

func openAPIJSON(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(docs.OpenAPIJSON)
}

func withUUID(next func(http.ResponseWriter, *http.Request, googleuuid.UUID)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawUUID := chi.URLParam(r, "uuid")
		parsedUUID, err := googleuuid.Parse(rawUUID)
		if err != nil {
			webresponse.WriteBadRequest(w, messages.InvalidUUID)
			return
		}
		next(w, r, parsedUUID)
	}
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	webresponse.WriteNotFound(w, messages.NotFound)
}

func methodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	webresponse.WriteMethodNotAllowed(w, messages.MethodNotAllowed)
}
