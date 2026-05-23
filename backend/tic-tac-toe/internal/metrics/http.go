package metrics

import (
	"sort"
	"sync"
	"time"
)

type HTTPRequestStat struct {
	Route             string `json:"route"`
	Method            string `json:"method"`
	Status            int    `json:"status"`
	Count             int64  `json:"count"`
	DurationMS        int64  `json:"durationMs"`
	AverageDurationMS int64  `json:"averageDurationMs"`
}

type httpRequestMetricKey struct {
	Route  string
	Method string
	Status int
}

type httpRequestMetricValue struct {
	Count        int64
	DurationSum  time.Duration
	LastDuration time.Duration
}

var (
	httpRequestMu      sync.Mutex
	httpRequestMetrics = make(map[httpRequestMetricKey]*httpRequestMetricValue)
)

func ObserveHTTPRequest(route string, method string, status int, duration time.Duration) {
	if route == "" {
		route = "unknown"
	}
	if method == "" {
		method = "UNKNOWN"
	}

	key := httpRequestMetricKey{Route: route, Method: method, Status: status}

	httpRequestMu.Lock()
	defer httpRequestMu.Unlock()

	entry, ok := httpRequestMetrics[key]
	if !ok {
		entry = &httpRequestMetricValue{}
		httpRequestMetrics[key] = entry
	}
	entry.Count++
	entry.DurationSum += duration
	entry.LastDuration = duration
}

func SnapshotHTTPRequestStats() []HTTPRequestStat {
	httpRequestMu.Lock()
	defer httpRequestMu.Unlock()

	stats := make([]HTTPRequestStat, 0, len(httpRequestMetrics))
	for key, value := range httpRequestMetrics {
		avg := int64(0)
		if value.Count > 0 {
			avg = int64((value.DurationSum / time.Duration(value.Count)).Milliseconds())
		}
		stats = append(stats, HTTPRequestStat{
			Route:             key.Route,
			Method:            key.Method,
			Status:            key.Status,
			Count:             value.Count,
			DurationMS:        value.DurationSum.Milliseconds(),
			AverageDurationMS: avg,
		})
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Route != stats[j].Route {
			return stats[i].Route < stats[j].Route
		}
		if stats[i].Method != stats[j].Method {
			return stats[i].Method < stats[j].Method
		}
		return stats[i].Status < stats[j].Status
	})

	return stats
}

func ResetHTTPRequestStats() {
	httpRequestMu.Lock()
	defer httpRequestMu.Unlock()
	httpRequestMetrics = make(map[httpRequestMetricKey]*httpRequestMetricValue)
}
