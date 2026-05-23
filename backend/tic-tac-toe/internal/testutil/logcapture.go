package testutil

import (
	"bytes"
	"log/slog"
	"testing"
)

func CaptureSlog(t *testing.T, level slog.Level) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level})))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	return &buf
}
