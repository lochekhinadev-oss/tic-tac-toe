package di

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	googleuuid "github.com/google/uuid"

	"tic-tac-toe/internal/transport/http/messages"
	"tic-tac-toe/internal/transport/http/middleware"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

const (
	maxRequestBodyBytes = 1 << 20
	rateLimitRequests   = 120
	rateLimitWindow     = time.Minute
)

func registerRouterMiddleware(router chi.Router) {
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(middleware.SecurityHeaders)
	router.Use(middleware.BodySizeLimit(maxRequestBodyBytes))
	router.Use(middleware.NewIPRateLimiter(middleware.IPRateLimiterConfig{
		Limit:  rateLimitRequests,
		Window: rateLimitWindow,
	}).Middleware)
	router.Use(middleware.RequestLogger)
	router.Use(chimiddleware.Recoverer)
	router.NotFound(notFound)
	router.MethodNotAllowed(methodNotAllowed)
}

func withUUID(next func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawUUID := chi.URLParam(r, "uuid")
		parsedUUID, err := googleuuid.Parse(rawUUID)
		if err != nil {
			webresponse.WriteBadRequest(w, messages.InvalidUUID)
			return
		}
		next(w, r, parsedUUID.String())
	}
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	webresponse.WriteNotFound(w, messages.NotFound)
}

func methodNotAllowed(w http.ResponseWriter, _ *http.Request) {
	webresponse.WriteMethodNotAllowed(w, messages.MethodNotAllowed)
}
