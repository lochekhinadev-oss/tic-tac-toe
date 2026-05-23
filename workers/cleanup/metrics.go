package main

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type cleanupMetrics struct {
	registryMu sync.RWMutex
	registry   *prometheus.Registry

	runsTotal            *prometheus.CounterVec
	deletedRowsTotal     *prometheus.CounterVec
	durationSeconds      *prometheus.HistogramVec
	lastRunTimestamp     prometheus.Gauge
	lastSuccessTimestamp prometheus.Gauge
}

func newCleanupMetrics() *cleanupMetrics {
	m := &cleanupMetrics{}
	m.reset()
	return m
}

func (m *cleanupMetrics) reset() {
	m.registryMu.Lock()
	defer m.registryMu.Unlock()

	registry := prometheus.NewRegistry()

	m.runsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tic_tac_toe_cleanup_runs_total",
		Help: "Total number of cleanup runs by result.",
	}, []string{"result"})
	m.deletedRowsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tic_tac_toe_cleanup_deleted_rows_total",
		Help: "Total number of deleted rows by entity.",
	}, []string{"entity"})
	m.durationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "tic_tac_toe_cleanup_duration_seconds",
		Help:    "Cleanup run duration in seconds.",
		Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
	}, []string{"result"})
	m.lastRunTimestamp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tic_tac_toe_cleanup_last_run_timestamp_seconds",
		Help: "Unix timestamp of the last cleanup run.",
	})
	m.lastSuccessTimestamp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tic_tac_toe_cleanup_last_success_timestamp_seconds",
		Help: "Unix timestamp of the last successful cleanup run.",
	})

	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(m.runsTotal)
	registry.MustRegister(m.deletedRowsTotal)
	registry.MustRegister(m.durationSeconds)
	registry.MustRegister(m.lastRunTimestamp)
	registry.MustRegister(m.lastSuccessTimestamp)

	m.registry = registry
}

func (m *cleanupMetrics) record(startedAt time.Time, result cleanupResult, err error) {
	if m == nil {
		return
	}

	duration := time.Since(startedAt).Seconds()
	resultLabel := "success"
	if err != nil {
		resultLabel = "failure"
	}

	m.registryMu.RLock()
	runsTotal := m.runsTotal
	deletedRowsTotal := m.deletedRowsTotal
	durationSeconds := m.durationSeconds
	lastRunTimestamp := m.lastRunTimestamp
	lastSuccessTimestamp := m.lastSuccessTimestamp
	m.registryMu.RUnlock()

	runsTotal.WithLabelValues(resultLabel).Inc()
	durationSeconds.WithLabelValues(resultLabel).Observe(duration)
	lastRunTimestamp.Set(float64(startedAt.Unix()))

	if err == nil {
		lastSuccessTimestamp.Set(float64(time.Now().UTC().Unix()))
	}

	if result.UsersDeleted > 0 {
		deletedRowsTotal.WithLabelValues("users").Add(float64(result.UsersDeleted))
	}
	if result.GamesDeleted > 0 {
		deletedRowsTotal.WithLabelValues("games").Add(float64(result.GamesDeleted))
	}
}

func (m *cleanupMetrics) handler() http.Handler {
	if m == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
	}

	m.registryMu.RLock()
	registry := m.registry
	m.registryMu.RUnlock()

	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}
