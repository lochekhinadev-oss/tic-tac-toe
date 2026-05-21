package auth

import (
	"sync"
	"time"
)

type authActionLimiter struct {
	mu          sync.Mutex
	limit       int
	window      time.Duration
	events      map[string][]time.Time
	now         func() time.Time
	lastCleanup time.Time
}

func newAuthActionLimiter(limit int, window time.Duration) *authActionLimiter {
	return &authActionLimiter{
		limit:  limit,
		window: window,
		events: make(map[string][]time.Time),
		now:    time.Now,
	}
}

func (l *authActionLimiter) Allow(key string) bool {
	if l == nil || l.limit <= 0 || l.window <= 0 {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now().UTC()
	cutoff := now.Add(-l.window)
	l.cleanupExpiredLocked(cutoff, now)

	events := l.events[key][:0]
	for _, eventTime := range l.events[key] {
		if eventTime.After(cutoff) {
			events = append(events, eventTime)
		}
	}
	if len(events) >= l.limit {
		l.events[key] = events
		return false
	}

	l.events[key] = append(events, now)
	return true
}

func (l *authActionLimiter) cleanupExpiredLocked(cutoff time.Time, now time.Time) {
	if !l.lastCleanup.IsZero() && now.Sub(l.lastCleanup) < l.window {
		return
	}
	for key, events := range l.events {
		keep := events[:0]
		for _, eventTime := range events {
			if eventTime.After(cutoff) {
				keep = append(keep, eventTime)
			}
		}
		if len(keep) == 0 {
			delete(l.events, key)
			continue
		}
		l.events[key] = keep
	}
	l.lastCleanup = now
}
