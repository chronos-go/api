package ratelimit

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type entry struct {
	count int
	reset time.Time
}

type Limiter struct {
	mu          sync.Mutex
	entries     map[string]entry
	limit       int
	window      time.Duration
	trustProxy  bool
	nowFn       func() time.Time
	lastCleanup time.Time
}

func New(limit int, window time.Duration, trustProxy bool) (*Limiter, error) {
	if limit <= 0 || window <= 0 {
		return nil, fmt.Errorf("rate limit and window must be positive")
	}
	return &Limiter{entries: make(map[string]entry), limit: limit, window: window, trustProxy: trustProxy, nowFn: time.Now}, nil
}

func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := l.clientIP(r)
		now := l.nowFn()
		l.mu.Lock()
		if l.lastCleanup.IsZero() || now.Sub(l.lastCleanup) >= l.window {
			for candidate, value := range l.entries {
				if !now.Before(value.reset) {
					delete(l.entries, candidate)
				}
			}
			l.lastCleanup = now
		}
		current, ok := l.entries[key]
		if !ok || !now.Before(current.reset) {
			current = entry{reset: now.Add(l.window)}
		}
		current.count++
		l.entries[key] = current
		allowed := current.count <= l.limit
		remaining := l.limit - current.count
		if remaining < 0 {
			remaining = 0
		}
		l.mu.Unlock()

		w.Header().Set("X-RateLimit-Limit", fmt.Sprint(l.limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprint(remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprint(current.reset.Unix()))
		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", fmt.Sprint(max(1, int(current.reset.Sub(now).Seconds()))))
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "too many requests"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *Limiter) clientIP(r *http.Request) string {
	if l.trustProxy {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			return strings.TrimSpace(strings.Split(forwarded, ",")[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
