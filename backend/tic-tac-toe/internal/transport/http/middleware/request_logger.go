package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"tic-tac-toe/internal/metrics"
)

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)
		duration := time.Since(start)

		fields := append(
			requestLogFields(r),
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration", duration.Round(time.Millisecond).String(),
		)
		metrics.ObserveHTTPRequest(routeLabel(r), r.Method, ww.Status(), duration)
		slog.Info("http request", fields...)
	})
}
