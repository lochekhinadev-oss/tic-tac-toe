package middleware

import (
	"net/http"

	chi "github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	observability "tic-tac-toe/internal/logging"
)

func requestLogFields(r *http.Request) []any {
	fields := append(observability.Fields(), []any{
		"request_id", chimiddleware.GetReqID(r.Context()),
		"method", r.Method,
		"route", routeLabel(r),
		"path", r.URL.Path,
	}...)
	return fields
}

func authLogFields(r *http.Request, userUUID string) []any {
	fields := requestLogFields(r)
	if userUUID != "" {
		fields = append(fields, "user_uuid", userUUID)
	}
	return fields
}

func routeLabel(r *http.Request) string {
	if ctx := chi.RouteContext(r.Context()); ctx != nil {
		if pattern := ctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return r.URL.Path
}
