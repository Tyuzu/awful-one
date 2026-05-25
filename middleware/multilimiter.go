package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// MultiLimiter manages multiple rate limiters for different endpoints
type MultiLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*RateLimiter
}

// NewMultiLimiter creates a new multi-limiter
func NewMultiLimiter() *MultiLimiter {
	return &MultiLimiter{
		limiters: make(map[string]*RateLimiter),
	}
}

// AddLimiter adds a rate limiter for a specific endpoint
func (ml *MultiLimiter) AddLimiter(endpoint string, requests int, window interface{}) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	var duration time.Duration
	if d, ok := window.(time.Duration); ok {
		duration = d
	} else if d, ok := window.(int); ok {
		duration = time.Duration(d) * time.Second
	} else {
		duration = 1 * time.Minute // default
	}

	// Create a RateLimiter with the specified requests per window
	requestsPerSecond := float64(requests) / duration.Seconds()
	limiter := NewRateLimiter(
		rate.Limit(requestsPerSecond),
		requests,
		5*time.Minute,
		10000,
	)

	ml.limiters[endpoint] = limiter
}

// GetLimiter retrieves the limiter for an endpoint, or returns nil if not configured
func (ml *MultiLimiter) GetLimiter(endpoint string) *RateLimiter {
	ml.mu.RLock()
	defer ml.mu.RUnlock()
	return ml.limiters[endpoint]
}

// GetMultiMiddleware returns a middleware that applies per-endpoint rate limiting
func (ml *MultiLimiter) GetMultiMiddleware(endpoint string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ml.mu.RLock()
			limiter, exists := ml.limiters[endpoint]
			ml.mu.RUnlock()

			// If no limit configured for this endpoint, allow all
			if !exists || limiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Extract client IP for rate limiting
			ip := extractClientIP(r)
			rateLimiter := limiter.getLimiter(ip)

			// Check rate limit
			if !rateLimiter.Allow() {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Stop stops all rate limiters
func (ml *MultiLimiter) Stop() {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	for _, limiter := range ml.limiters {
		if limiter != nil {
			limiter.Stop()
		}
	}
}
