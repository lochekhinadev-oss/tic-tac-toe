package metrics

import (
	"sort"
	"sync"
)

type AuthEventStat struct {
	Event string `json:"event"`
	Count int64  `json:"count"`
}

var (
	authEventMu      sync.Mutex
	authEventMetrics = make(map[string]int64)
)

func ObserveAuthEvent(event string) {
	if event == "" {
		event = "unknown"
	}

	authEventMu.Lock()
	defer authEventMu.Unlock()
	authEventMetrics[event]++
}

func SnapshotAuthEventStats() []AuthEventStat {
	authEventMu.Lock()
	defer authEventMu.Unlock()

	stats := make([]AuthEventStat, 0, len(authEventMetrics))
	for event, count := range authEventMetrics {
		stats = append(stats, AuthEventStat{Event: event, Count: count})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Event < stats[j].Event
	})

	return stats
}

func ResetAuthEventStats() {
	authEventMu.Lock()
	defer authEventMu.Unlock()
	authEventMetrics = make(map[string]int64)
}
