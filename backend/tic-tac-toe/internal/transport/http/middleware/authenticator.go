package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/internal/transport/http/messages"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

type TokenAuthenticator interface {
	AuthenticateToken(ctx context.Context, header string) (string, error)
}

type UserAuthenticator struct {
	auth TokenAuthenticator
}

func NewUserAuthenticator(auth TokenAuthenticator) *UserAuthenticator {
	return &UserAuthenticator{auth: auth}
}

func (a *UserAuthenticator) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("auth check", "method", r.Method, "path", r.URL.Path)

		uuid, err := a.auth.AuthenticateToken(r.Context(), r.Header.Get("Authorization"))
		if err != nil {
			if errors.Is(err, auth.ErrInvalidAuthHeader) || errors.Is(err, auth.ErrInvalidToken) {
				slog.Info("auth rejected", "method", r.Method, "path", r.URL.Path, "reason", err)
				webresponse.WriteUnauthorized(w)
				return
			}

			writeAuthMiddlewareError(w, r, "auth failed", err, func(w http.ResponseWriter) {
				webresponse.WriteInternalError(w, messages.FailedAuthenticateUser)
			})
			return
		}

		slog.Debug("auth ok", "method", r.Method, "path", r.URL.Path, "user_uuid", uuid)
		next.ServeHTTP(w, r.WithContext(WithUserUUID(r.Context(), uuid)))
	})
}

func writeAuthMiddlewareError(w http.ResponseWriter, r *http.Request, logMessage string, err error, write func(http.ResponseWriter)) {
	slog.Error(logMessage, "method", r.Method, "path", r.URL.Path, "error", err)
	write(w)
}
