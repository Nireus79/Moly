package main

import (
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



// EndpointRateLimits defines rate limits for different endpoints
type EndpointRateLimits struct {
	ChatLimit     *RateLimiter
	APILimit      *RateLimiter
	ModelLimit    *RateLimiter
	ProviderLimit *RateLimiter
}


// Close closes all rate limiters
func (e *EndpointRateLimits) Close() {
	e.ChatLimit.Close()
	e.APILimit.Close()
	e.ModelLimit.Close()
	e.ProviderLimit.Close()
}


