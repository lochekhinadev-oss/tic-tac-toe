package metrics

import (
	"sort"
	"sync"
)

type GameEventStat struct {
	Event string `json:"event"`
	Count int64  `json:"count"`
}

var (
	gameEventMu      sync.Mutex
	gameEventMetrics = make(map[string]int64)
)

func ObserveGameEvent(event string) {
	if event == "" {
		event = "unknown"
	}

	gameEventMu.Lock()
	defer gameEventMu.Unlock()
	gameEventMetrics[event]++
}

func SnapshotGameEventStats() []GameEventStat {
	gameEventMu.Lock()
	defer gameEventMu.Unlock()

	stats := make([]GameEventStat, 0, len(gameEventMetrics))
	for event, count := range gameEventMetrics {
		stats = append(stats, GameEventStat{Event: event, Count: count})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Event < stats[j].Event
	})

	return stats
}

func ResetGameEventStats() {
	gameEventMu.Lock()
	defer gameEventMu.Unlock()
	gameEventMetrics = make(map[string]int64)
}
