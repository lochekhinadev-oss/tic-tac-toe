package application

import (
	"log/slog"

	observability "tic-tac-toe/internal/logging"
)

const applicationLogPrefix = "[app/application]"

func logApplication(action string, args ...any) {
	fields := append(observability.Fields(), args...)
	slog.Info(applicationLogPrefix+" "+action, fields...)
}
