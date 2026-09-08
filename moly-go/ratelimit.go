package main

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu                sync.RWMutex
	buckets           map[string]*tokenBucket
	requestsPerSecond float64
	burstSize         int
	cleanupTicker     *time.Ticker
}

// tokenBucket represents a single rate limit bucket
type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond float64, burstSize int) *RateLimiter {
	rl := &RateLimiter{
		buckets:           make(map[string]*tokenBucket),
		requestsPerSecond: requestsPerSecond,
		burstSize:         burstSize,
		cleanupTicker:     time.NewTicker(1 * time.Minute),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request from identifier is allowed
func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[identifier]
	if !exists {
		// New bucket
		bucket = &tokenBucket{
			tokens:     float64(rl.burstSize),
			lastRefill: time.Now(),
		}
		rl.buckets[identifier] = bucket
	}

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	tokensToAdd := elapsed * rl.requestsPerSecond

	if tokensToAdd > 0 {
		bucket.tokens = minFloat(float64(rl.burstSize), bucket.tokens+tokensToAdd)
		bucket.lastRefill = now
	}

	// Check if request is allowed
	if bucket.tokens >= 1.0 {
		bucket.tokens--
		Logger.WithFields(map[string]interface{}{
			"identifier": identifier,
			"remaining":  bucket.tokens,
		}).Debug("Rate limit allowed")
		return true
	}

	Logger.WithFields(map[string]interface{}{
		"identifier": identifier,
		"tokens":     bucket.tokens,
	}).Warn("Rate limit exceeded")
	return false
}

// cleanup removes old buckets that haven't been used
func (rl *RateLimiter) cleanup() {
	for range rl.cleanupTicker.C {
		rl.mu.Lock()

		now := time.Now()
		for id, bucket := range rl.buckets {
			// Remove buckets that haven't been refilled in 1 hour
			if now.Sub(bucket.lastRefill) > 1*time.Hour {
				delete(rl.buckets, id)
			}
		}

		rl.mu.Unlock()
	}
}

// Close stops the cleanup ticker
func (rl *RateLimiter) Close() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
}

// RateLimitMiddleware returns HTTP middleware for rate limiting
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get identifier from request (IP address by default)
			identifier := getClientIP(r)

			if !limiter.Allow(identifier) {
				w.Header().Set("Retry-After", "1")
				respondError(w, http.StatusTooManyRequests, "Rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}

	// Check X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// EndpointRateLimits defines rate limits for different endpoints
type EndpointRateLimits struct {
	ChatLimit     *RateLimiter
	APILimit      *RateLimiter
	ModelLimit    *RateLimiter
	ProviderLimit *RateLimiter
}

// NewEndpointRateLimits creates rate limiters for different endpoints
func NewEndpointRateLimits() *EndpointRateLimits {
	return &EndpointRateLimits{
		ChatLimit:     NewRateLimiter(5, 10),  // 5 req/sec, burst of 10
		APILimit:      NewRateLimiter(20, 50), // 20 req/sec, burst of 50
		ModelLimit:    NewRateLimiter(2, 5),   // 2 req/sec, burst of 5 (model operations)
		ProviderLimit: NewRateLimiter(10, 20), // 10 req/sec, burst of 20
	}
}

// Close closes all rate limiters
func (e *EndpointRateLimits) Close() {
	e.ChatLimit.Close()
	e.APILimit.Close()
	e.ModelLimit.Close()
	e.ProviderLimit.Close()
}

// IsIPAllowed checks if an IP is in the blocklist
func IsIPAllowed(ip string, blocklist []string) bool {
	for _, blocked := range blocklist {
		if ip == blocked {
			return false
		}
	}
	return true
}

// min returns the minimum of two floats
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
