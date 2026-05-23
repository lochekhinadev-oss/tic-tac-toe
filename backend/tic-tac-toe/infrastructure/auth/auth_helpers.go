package auth

import (
	"log/slog"

	observability "tic-tac-toe/internal/logging"
)

const authLogPrefix = "[infrastructure/auth]"
const SessionCookieName = "tic-tac-toe.session"

func logAuth(action string, args ...any) {
	fields := append(observability.Fields(), args...)
	slog.Info(authLogPrefix+" "+action, fields...)
}
