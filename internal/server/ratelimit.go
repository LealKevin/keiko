package server

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type RateLimiter struct {
	requests     map[string][]time.Time
	mu           sync.Mutex
	maxRequests  int
	windowSecs   int
	cleanupEvery time.Duration
	done         chan struct{}
}

func NewRateLimiter(maxRequests, windowSecs int) *RateLimiter {
	rl := &RateLimiter{
		requests:     make(map[string][]time.Time),
		maxRequests:  maxRequests,
		windowSecs:   windowSecs,
		cleanupEvery: 5 * time.Minute,
		done:         make(chan struct{}),
	}

	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) Close() {
	close(rl.done)
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		if !rl.allow(ip) {
			w.Header().Set("Retry-After", "60")
			writeError(w, http.StatusTooManyRequests, "Rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-time.Duration(rl.windowSecs) * time.Second)

	// Get requests for this IP
	requests := rl.requests[ip]

	// Filter to only requests within window
	var validRequests []time.Time
	for _, t := range requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	// Check if limit exceeded
	if len(validRequests) >= rl.maxRequests {
		rl.requests[ip] = validRequests
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[ip] = validRequests

	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupEvery)
	defer ticker.Stop()

	for {
		select {
		case <-rl.done:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			windowStart := now.Add(-time.Duration(rl.windowSecs) * time.Second)

			for ip, requests := range rl.requests {
				var validRequests []time.Time
				for _, t := range requests {
					if t.After(windowStart) {
						validRequests = append(validRequests, t)
					}
				}
				if len(validRequests) == 0 {
					delete(rl.requests, ip)
				} else {
					rl.requests[ip] = validRequests
				}
			}
			rl.mu.Unlock()
		}
	}
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies/load balancers)
	// Format: "client, proxy1, proxy2" - we want the first one
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take first IP if comma-separated
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
