package di

import (
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"

	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/internal/transport/http/dto"
	"tic-tac-toe/internal/transport/http/messages"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

var startedAt = time.Now()

func registerSystemRoutes(router chi.Router, db datasource.Database) {
	router.Get("/healthz", healthz)
	router.Get("/readyz", readyz(db))
	router.Get("/metrics", metrics)
	router.Get("/swagger", swaggerUI)
	router.Get("/openapi.yaml", openAPIYAML)
	router.Get("/swagger/doc.json", openAPIJSON)
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	webresponse.WriteJSON(w, http.StatusOK, dto.HealthResponse{Status: "ok"})
}

func readyz(db datasource.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			webresponse.WriteInternalError(w, messages.DatabaseNotReady)
			return
		}

		webresponse.WriteJSON(w, http.StatusOK, dto.HealthResponse{Status: "ready"})
	}
}

func metrics(w http.ResponseWriter, _ *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	webresponse.WriteJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"uptimeSec":     int64(time.Since(startedAt).Seconds()),
		"goroutines":    runtime.NumGoroutine(),
		"allocBytes":    mem.Alloc,
		"totalAlloc":    mem.TotalAlloc,
		"sysBytes":      mem.Sys,
		"heapObjects":   mem.HeapObjects,
		"gcCycles":      mem.NumGC,
		"lastGCPauseNs": mem.PauseNs[(mem.NumGC+255)%256],
	})
}
