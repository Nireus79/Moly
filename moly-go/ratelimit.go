package main

import (
	"encoding/json"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// Rate limiting: one token bucket per (class of endpoint, client address). A request takes one token; an empty bucket
// answers 429 with Retry-After. The client is the connection's address (not a header the client can set, and not the
// token: a caller could rotate fake tokens to get a fresh bucket each time). Behind a reverse proxy every caller shares
// the proxy's address, so the limits are set for one household, not for a public service.

type limitSpec struct {
	perSecond float64 // refill rate
	burst     float64 // bucket size
}

// Classes of endpoints, from the cost of a request.
const (
	classAuth    = "auth"    // register, login: guessing passwords or flooding sign-ups
	classModel   = "model"   // endpoints that start model calls (a message turn makes 10-12 of them)
	classGeneral = "general" // everything else
)

var limitSpecs = map[string]limitSpec{
	classAuth:    {perSecond: 1.0 / 12, burst: 5}, // five at once, then one every 12 seconds
	classModel:   {perSecond: 1.0 / 3, burst: 5},  // five at once, then one every 3 seconds
	classGeneral: {perSecond: 5, burst: 100},
}

// limitClass decides the class of a request by its path. The health check is not limited: the extension polls it to find the
// backend before anyone has logged in.
func limitClass(path string) (class string, limited bool) {
	switch path {
	case "/api/status":
		return "", false
	case "/api/auth/register", "/api/auth/login", "/api/auth/reset", "/api/auth/password":
		return classAuth, true
	case "/api/message-processor", "/api/clarification/respond":
		return classModel, true
	}
	return classGeneral, true
}

type tokenBucket struct {
	tokens float64
	last   time.Time
}

type RateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*tokenBucket
	now       func() time.Time
	lastSweep time.Time
	specs     map[string]limitSpec
}

const (
	maxBuckets    = 10000
	sweepInterval = 5 * time.Minute
)

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{buckets: map[string]*tokenBucket{}, now: time.Now, specs: limitSpecs}
}

// Allow takes a token from the bucket of (class, client). When the bucket is empty it returns false and how long until a
// token is available.
func (rl *RateLimiter) Allow(class, client string) (bool, time.Duration) {
	spec, ok := rl.specs[class]
	if !ok {
		return true, 0
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := rl.now()
	rl.sweep(now)

	key := class + "|" + client
	b := rl.buckets[key]
	if b == nil {
		if len(rl.buckets) >= maxBuckets {
			// Flooded with new addresses: refuse new ones rather than grow without bound.
			return false, time.Duration(float64(time.Second) / spec.perSecond)
		}
		b = &tokenBucket{tokens: spec.burst, last: now}
		rl.buckets[key] = b
	}
	b.tokens = math.Min(spec.burst, b.tokens+now.Sub(b.last).Seconds()*spec.perSecond)
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	return false, time.Duration((1 - b.tokens) / spec.perSecond * float64(time.Second))
}

// sweep drops buckets that have refilled completely (they carry no information). Called with the lock held, at most every
// sweepInterval, so no background goroutine is needed.
func (rl *RateLimiter) sweep(now time.Time) {
	if now.Sub(rl.lastSweep) < sweepInterval {
		return
	}
	rl.lastSweep = now
	for key, b := range rl.buckets {
		for class, spec := range rl.specs {
			if len(key) > len(class) && key[:len(class)+1] == class+"|" &&
				b.tokens+now.Sub(b.last).Seconds()*spec.perSecond >= spec.burst {
				delete(rl.buckets, key)
			}
		}
	}
}

// clientAddress is the address of the connection.
func clientAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// rateLimitMiddleware answers 429 to a client whose bucket is empty. MOLY_RATE_LIMIT=off turns it off (load tests).
func rateLimitMiddleware(rl *RateLimiter, next http.Handler) http.Handler {
	if os.Getenv("MOLY_RATE_LIMIT") == "off" {
		log.Printf("[RateLimit] disabled by MOLY_RATE_LIMIT=off")
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		class, limited := limitClass(r.URL.Path)
		if !limited {
			next.ServeHTTP(w, r)
			return
		}
		if ok, wait := rl.Allow(class, clientAddress(r)); !ok {
			secs := int(math.Ceil(wait.Seconds()))
			log.Printf("[RateLimit] 429 %s %s from %s (class %s), retry in %ds", r.Method, r.URL.Path, clientAddress(r), class, secs)
			w.Header().Set("Retry-After", strconv.Itoa(secs))
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "Too many requests", "retryAfterSeconds": secs})
			return
		}
		next.ServeHTTP(w, r)
	})
}
