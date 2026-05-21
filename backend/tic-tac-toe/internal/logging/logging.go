package log

import (
	"fmt"
	"log/slog"
	"os"
)

func init() {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
}

func Printf(format string, args ...any) {
	slog.Default().Info(fmt.Sprintf(format, args...))
}
