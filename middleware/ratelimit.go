package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/time/rate"
)

// visitorEntry tracks a rate limiter and its last access time for TTL-based cleanup
type visitorEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter is a middleware struct with configuration and visitor state
type RateLimiter struct {
	visitors     map[string]*visitorEntry
	mu           sync.Mutex
	rate         rate.Limit
	burst        int
	cleanupAfter time.Duration
	maxEntries   int
	stopChan     chan struct{}
}

// NewRateLimiter initializes a new RateLimiter with a background cleanup goroutine
func NewRateLimiter(r rate.Limit, b int, cleanupAfter time.Duration, maxEntries int) *RateLimiter {
	rl := &RateLimiter{
		visitors:     make(map[string]*visitorEntry),
		rate:         r,
		burst:        b,
		cleanupAfter: cleanupAfter,
		maxEntries:   maxEntries,
		stopChan:     make(chan struct{}),
	}
	// Start single cleanup goroutine instead of one per IP
	go rl.cleanupLoop()
	return rl
}

// cleanupLoop periodically removes stale entries from the visitors map
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupAfter / 2) // cleanup twice per TTL
	defer ticker.Stop()

	for {
		select {
		case <-rl.stopChan:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, entry := range rl.visitors {
				if now.Sub(entry.lastSeen) > rl.cleanupAfter {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Stop gracefully shuts down the cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}

// getLimiter returns an existing limiter or creates a new one, updating lastSeen time
func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if entry, exists := rl.visitors[ip]; exists {
		entry.lastSeen = time.Now()
		return entry.limiter
	}

	// Enforce max entries to avoid memory abuse
	if len(rl.visitors) >= rl.maxEntries {
		// Return a strict fallback limiter without storing it
		return rate.NewLimiter(rate.Limit(0.1), 1)
	}

	limiter := rate.NewLimiter(rl.rate, rl.burst)
	rl.visitors[ip] = &visitorEntry{
		limiter:  limiter,
		lastSeen: time.Now(),
	}

	return limiter
}

// extractClientIP tries to determine the client's real IP address.
// Note: X-Forwarded-For can be spoofed; only trust it if behind a trusted proxy.
func extractClientIP(r *http.Request) string {
	// Only use X-Forwarded-For if you control the proxies. Comment out if untrusted.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Fallback: use RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// Limit is the httprouter middleware for rate limiting
func (rl *RateLimiter) Limit(next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ip := extractClientIP(r)
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next(w, r, ps)
	}
}

// LimitHandler adapts the Limit middleware for use with http.Handler
func (rl *RateLimiter) LimitHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractClientIP(r)
		limiter := rl.getLimiter(ip)

		if !limiter.Allow() {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Allow checks if a request is allowed based on rate limiting
func (rl *RateLimiter) Allow(r *http.Request) bool {
	ip := extractClientIP(r)
	limiter := rl.getLimiter(ip)
	return limiter.Allow()
}
