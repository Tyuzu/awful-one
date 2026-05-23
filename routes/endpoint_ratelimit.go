package routes

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// EndpointRateLimiter provides per-endpoint and per-user rate limiting
type EndpointRateLimiter struct {
	mu              sync.RWMutex
	userBuckets     map[string]*TokenBucket
	ipBuckets       map[string]*TokenBucket
	endpointLimits  map[string]RateLimitConfig
	cleanupTicker   *time.Ticker
	defaultCapacity int
	defaultRefill   time.Duration
}

// RateLimitConfig defines rate limit parameters for an endpoint
type RateLimitConfig struct {
	RequestsPerWindow int
	Window            time.Duration
	ByUser            bool // If true, limit per user, else by IP
	ByIP              bool // If true, also limit by IP regardless of user limit
}

// TokenBucket implements token bucket algorithm
type TokenBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewEndpointRateLimiter creates a new endpoint rate limiter
func NewEndpointRateLimiter() *EndpointRateLimiter {
	erl := &EndpointRateLimiter{
		userBuckets:     make(map[string]*TokenBucket),
		ipBuckets:       make(map[string]*TokenBucket),
		endpointLimits:  make(map[string]RateLimitConfig),
		defaultCapacity: 100,
		defaultRefill:   1 * time.Minute,
		cleanupTicker:   time.NewTicker(5 * time.Minute),
	}

	// Cleanup old buckets periodically
	go erl.cleanupOldBuckets()

	return erl
}

// AddEndpointLimit adds a rate limit for a specific endpoint
func (erl *EndpointRateLimiter) AddEndpointLimit(endpoint string, config RateLimitConfig) {
	erl.mu.Lock()
	defer erl.mu.Unlock()
	erl.endpointLimits[endpoint] = config
}

// ConfigureCommonEndpoints sets up rate limits for standard purchase endpoints
func (erl *EndpointRateLimiter) ConfigureCommonEndpoints() {
	// Ticket purchases: 10 per minute per user
	erl.AddEndpointLimit("/api/v1/ticket/event/*/buy", RateLimitConfig{
		RequestsPerWindow: 10,
		Window:            1 * time.Minute,
		ByUser:            true,
		ByIP:              true,
	})

	// Merchandise purchases: 10 per minute per user
	erl.AddEndpointLimit("/api/v1/merch/*/confirm-purchase", RateLimitConfig{
		RequestsPerWindow: 10,
		Window:            1 * time.Minute,
		ByUser:            true,
		ByIP:              true,
	})

	// Topup: 5 per minute per user
	erl.AddEndpointLimit("/api/v1/pay/topup", RateLimitConfig{
		RequestsPerWindow: 5,
		Window:            1 * time.Minute,
		ByUser:            true,
		ByIP:              true,
	})

	// Payment: 5 per minute per user
	erl.AddEndpointLimit("/api/v1/pay/transfer", RateLimitConfig{
		RequestsPerWindow: 5,
		Window:            1 * time.Minute,
		ByUser:            true,
		ByIP:              true,
	})

	// Cart: 20 per minute per user
	erl.AddEndpointLimit("/api/v1/cart", RateLimitConfig{
		RequestsPerWindow: 20,
		Window:            1 * time.Minute,
		ByUser:            true,
	})

	// Form submissions: 5 per hour per IP
	erl.AddEndpointLimit("/api/v1/contact", RateLimitConfig{
		RequestsPerWindow: 5,
		Window:            1 * time.Hour,
		ByIP:              true,
	})
}

// Allow checks if a request is allowed
func (erl *EndpointRateLimiter) Allow(endpoint string, userID string, ip string) bool {
	erl.mu.RLock()
	config, exists := erl.endpointLimits[endpoint]
	erl.mu.RUnlock()

	if !exists {
		return true // No limit configured
	}

	// Check user limit if configured
	if config.ByUser && userID != "" {
		if !erl.checkBucket(userID, "user", config) {
			return false
		}
	}

	// Check IP limit if configured
	if config.ByIP {
		if !erl.checkBucket(ip, "ip", config) {
			return false
		}
	}

	return true
}

// checkBucket checks if a token is available in the bucket
func (erl *EndpointRateLimiter) checkBucket(key string, bucketType string, config RateLimitConfig) bool {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	var buckets map[string]*TokenBucket
	if bucketType == "user" {
		buckets = erl.userBuckets
	} else {
		buckets = erl.ipBuckets
	}

	bucket, exists := buckets[key]
	if !exists {
		refillRate := float64(config.RequestsPerWindow) / config.Window.Seconds()
		bucket = &TokenBucket{
			tokens:     float64(config.RequestsPerWindow),
			capacity:   float64(config.RequestsPerWindow),
			refillRate: refillRate,
			lastRefill: time.Now(),
		}
		buckets[key] = bucket
	}

	// Refill tokens
	bucket.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens = min(bucket.capacity, bucket.tokens+bucket.refillRate*elapsed)
	bucket.lastRefill = now
	bucket.mu.Unlock()

	// Check if we have a token
	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	if bucket.tokens >= 1 {
		bucket.tokens--
		return true
	}

	return false
}

// cleanupOldBuckets removes unused buckets
func (erl *EndpointRateLimiter) cleanupOldBuckets() {
	for range erl.cleanupTicker.C {
		erl.mu.Lock()
		now := time.Now()

		// Clean user buckets
		for key, bucket := range erl.userBuckets {
			bucket.mu.Lock()
			if now.Sub(bucket.lastRefill) > 10*time.Minute {
				delete(erl.userBuckets, key)
			}
			bucket.mu.Unlock()
		}

		// Clean IP buckets
		for key, bucket := range erl.ipBuckets {
			bucket.mu.Lock()
			if now.Sub(bucket.lastRefill) > 10*time.Minute {
				delete(erl.ipBuckets, key)
			}
			bucket.mu.Unlock()
		}

		erl.mu.Unlock()
	}
}

// Middleware returns an HTTP middleware for rate limiting
func (erl *EndpointRateLimiter) Middleware(endpoint string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract user ID and IP
			userID := extractUserID(r)
			ip := extractClientIP(r)

			// Check rate limit
			if !erl.Allow(endpoint, userID, ip) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitHandler wraps a handler with rate limiting
func (erl *EndpointRateLimiter) RateLimitHandler(endpoint string, handler http.Handler) http.Handler {
	return erl.Middleware(endpoint)(handler)
}

// Helper functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func extractUserID(r *http.Request) string {
	if userID, ok := r.Context().Value("userID").(string); ok {
		return userID
	}
	return ""
}

func extractClientIP(r *http.Request) string {
	// Check XFF header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// Stop stops the cleanup ticker
func (erl *EndpointRateLimiter) Stop() {
	erl.cleanupTicker.Stop()
}
