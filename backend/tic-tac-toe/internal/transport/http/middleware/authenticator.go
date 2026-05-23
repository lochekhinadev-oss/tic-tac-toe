package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/application"
	"tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/internal/transport/http/messages"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

type TokenAuthenticator interface {
	AuthenticateSession(ctx context.Context, sessionID string) (string, error)
}

type UserAuthenticator struct {
	auth       TokenAuthenticator
	authorizer application.RequestAuthorizer
}

func NewUserAuthenticator(auth TokenAuthenticator, authorizer application.RequestAuthorizer) *UserAuthenticator {
	return &UserAuthenticator{auth: auth, authorizer: authorizer}
}

func (a *UserAuthenticator) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("auth check", "method", r.Method, "path", r.URL.Path)

		uuid, err := a.authenticateRequest(r)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidToken) {
				slog.Info("auth rejected", "method", r.Method, "path", r.URL.Path, "reason", err)
				webresponse.WriteUnauthorized(w)
				return
			}

			writeAuthMiddlewareError(w, r, "auth failed", err, func(w http.ResponseWriter) {
				webresponse.WriteInternalError(w, messages.FailedAuthenticateUser)
			})
			return
		}

		userUUID, err := googleuuid.Parse(uuid)
		if err != nil {
			writeAuthMiddlewareError(w, r, "invalid authenticated user uuid", err, func(w http.ResponseWriter) {
				webresponse.WriteInternalError(w, messages.FailedAuthenticateUser)
			})
			return
		}

		allowed, err := a.authorizeRequest(r, userUUID)
		if err != nil {
			writeAuthMiddlewareError(w, r, "authz failed", err, func(w http.ResponseWriter) {
				webresponse.WriteInternalError(w, messages.FailedAuthorizeUser)
			})
			return
		}
		if !allowed {
			slog.Info("authz rejected", "method", r.Method, "path", r.URL.Path, "user_uuid", uuid, "reason", "request not permitted")
			webresponse.WriteForbidden(w)
			return
		}

		slog.Debug("auth ok", "method", r.Method, "path", r.URL.Path, "user_uuid", uuid)
		next.ServeHTTP(w, r.WithContext(WithUserUUID(r.Context(), uuid)))
	})
}

func (a *UserAuthenticator) authenticateRequest(r *http.Request) (string, error) {
	if cookie, err := r.Cookie(auth.SessionCookieName); err == nil && cookie != nil && cookie.Value != "" {
		return a.auth.AuthenticateSession(r.Context(), cookie.Value)
	}

	return "", auth.ErrInvalidToken
}

func (a *UserAuthenticator) authorizeRequest(r *http.Request, userUUID googleuuid.UUID) (bool, error) {
	if a.authorizer == nil {
		return true, nil
	}
	return a.authorizer.AuthorizeRequest(r.Context(), userUUID, r.Method, r.URL.Path)
}

func writeAuthMiddlewareError(w http.ResponseWriter, r *http.Request, logMessage string, err error, write func(http.ResponseWriter)) {
	slog.Error(logMessage, "method", r.Method, "path", r.URL.Path, "error", err)
	write(w)
}
