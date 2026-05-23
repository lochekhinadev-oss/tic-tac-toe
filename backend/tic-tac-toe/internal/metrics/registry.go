package metrics

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registryMu sync.RWMutex
	registry   *prometheus.Registry

	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	authEventsTotal     *prometheus.CounterVec
	gameEventsTotal     *prometheus.CounterVec
)

func init() {
	resetLocked()
}

func resetLocked() {
	registry = prometheus.NewRegistry()

	httpRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tic_tac_toe_http_requests_total",
		Help: "Total number of HTTP requests observed by route, method, and status.",
	}, []string{"route", "method", "status"})

	httpRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "tic_tac_toe_http_request_duration_seconds",
		Help:    "HTTP request durations in seconds.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"route", "method"})

	authEventsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tic_tac_toe_auth_events_total",
		Help: "Total number of auth-related events.",
	}, []string{"event"})

	gameEventsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "tic_tac_toe_game_events_total",
		Help: "Total number of game-related events.",
	}, []string{"event"})

	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(httpRequestsTotal)
	registry.MustRegister(httpRequestDuration)
	registry.MustRegister(authEventsTotal)
	registry.MustRegister(gameEventsTotal)
}

func ResetForTests() {
	registryMu.Lock()
	defer registryMu.Unlock()
	resetLocked()
}

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		registryMu.RLock()
		currentRegistry := registry
		registryMu.RUnlock()

		promhttp.HandlerFor(currentRegistry, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	})
}

func ObserveHTTPRequest(route string, method string, status int, duration time.Duration) {
	if route == "" {
		route = "unknown"
	}
	if method == "" {
		method = "UNKNOWN"
	}

	registryMu.RLock()
	counter := httpRequestsTotal
	histogram := httpRequestDuration
	registryMu.RUnlock()

	counter.WithLabelValues(route, method, strconv.Itoa(status)).Inc()
	histogram.WithLabelValues(route, method).Observe(duration.Seconds())
}

func ObserveAuthEvent(event string) {
	if event == "" {
		event = "unknown"
	}

	registryMu.RLock()
	counter := authEventsTotal
	registryMu.RUnlock()

	counter.WithLabelValues(event).Inc()
}

func ObserveGameEvent(event string) {
	if event == "" {
		event = "unknown"
	}

	registryMu.RLock()
	counter := gameEventsTotal
	registryMu.RUnlock()

	counter.WithLabelValues(event).Inc()
}
