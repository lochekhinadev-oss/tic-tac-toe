package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"tic-tac-toe/internal/transport/http/messages"
	webresponse "tic-tac-toe/internal/transport/http/response"
)

type IPRateLimiterConfig struct {
	Limit  int
	Window time.Duration
}

type IPRateLimiter struct {
	config IPRateLimiterConfig
	now    func() time.Time

	mu          sync.Mutex
	requests    map[string][]time.Time
	lastCleanup time.Time
}

func NewIPRateLimiter(config IPRateLimiterConfig) *IPRateLimiter {
	return &IPRateLimiter{
		config:   config,
		now:      time.Now,
		requests: make(map[string][]time.Time),
	}
}

func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			webresponse.WriteTooManyRequests(w, messages.TooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (l *IPRateLimiter) allow(key string) bool {
	if l == nil || l.config.Limit <= 0 || l.config.Window <= 0 {
		return true
	}

	now := l.now()
	windowStart := now.Add(-l.config.Window)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanupExpiredLocked(windowStart, now)
	events := l.requests[key]
	keepFrom := 0
	for keepFrom < len(events) && events[keepFrom].Before(windowStart) {
		keepFrom++
	}
	events = events[keepFrom:]

	if len(events) >= l.config.Limit {
		l.requests[key] = events
		return false
	}

	l.requests[key] = append(events, now)
	return true
}

func (l *IPRateLimiter) cleanupExpiredLocked(windowStart time.Time, now time.Time) {
	if !l.lastCleanup.IsZero() && now.Sub(l.lastCleanup) < l.config.Window {
		return
	}
	for key, events := range l.requests {
		keepFrom := 0
		for keepFrom < len(events) && events[keepFrom].Before(windowStart) {
			keepFrom++
		}
		events = events[keepFrom:]
		if len(events) == 0 {
			delete(l.requests, key)
			continue
		}
		l.requests[key] = events
	}
	l.lastCleanup = now
}

func clientIP(r *http.Request) string {
	if ip := net.ParseIP(r.RemoteAddr); ip != nil {
		return ip.String()
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
