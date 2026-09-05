package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(5, 10) // 5 req/sec, burst of 10
	defer rl.Close()

	testCases := []struct {
		name       string
		identifier string
		count      int
		expected   int
	}{
		{
			name:       "Within burst limit",
			identifier: "user1",
			count:      5,
			expected:   5,
		},
		{
			name:       "Exceed burst limit",
			identifier: "user2",
			count:      15,
			expected:   10, // Should be capped at burst size
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allowed := 0
			for i := 0; i < tc.count; i++ {
				if rl.Allow(tc.identifier) {
					allowed++
				}
			}

			if allowed != tc.expected {
				t.Errorf("Expected %d allowed, got %d", tc.expected, allowed)
			}
		})
	}
}

func TestRateLimiterTokenRefill(t *testing.T) {
	rl := NewRateLimiter(10, 5) // 10 req/sec, burst of 5
	defer rl.Close()

	identifier := "test_user"

	// Use up the burst
	for i := 0; i < 5; i++ {
		if !rl.Allow(identifier) {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	// Next request should be denied
	if rl.Allow(identifier) {
		t.Error("Request should be denied after burst exhausted")
	}

	// Wait for token refill
	time.Sleep(100 * time.Millisecond) // ~1 token refilled at 10 req/sec

	// Should allow one more request
	if !rl.Allow(identifier) {
		t.Error("Request should be allowed after refill")
	}
}

func TestRateLimiterMultipleUsers(t *testing.T) {
	rl := NewRateLimiter(5, 10)
	defer rl.Close()

	// Different users should have independent limits
	user1Allowed := 0
	user2Allowed := 0

	for i := 0; i < 10; i++ {
		if rl.Allow("user1") {
			user1Allowed++
		}
		if rl.Allow("user2") {
			user2Allowed++
		}
	}

	if user1Allowed != 10 || user2Allowed != 10 {
		t.Errorf("Users should have independent limits: user1=%d, user2=%d", user1Allowed, user2Allowed)
	}
}

func TestEndpointRateLimits(t *testing.T) {
	limits := NewEndpointRateLimits()
	defer limits.Close()

	identifier := "test_client"

	// Chat limit should be more restrictive
	chatAllowed := 0
	for i := 0; i < 10; i++ {
		if limits.ChatLimit.Allow(identifier) {
			chatAllowed++
		}
	}

	// API limit should allow more
	apiAllowed := 0
	for i := 0; i < 50; i++ {
		if limits.APILimit.Allow(identifier) {
			apiAllowed++
		}
	}

	if chatAllowed != 10 {
		t.Errorf("Chat limit should allow burst of 10, got %d", chatAllowed)
	}

	if apiAllowed != 50 {
		t.Errorf("API limit should allow burst of 50, got %d", apiAllowed)
	}
}

func TestGetClientIP(t *testing.T) {
	testCases := []struct {
		name     string
		setup    func(*http.Request)
		expected string
	}{
		{
			name: "X-Forwarded-For header",
			setup: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "192.168.1.100")
			},
			expected: "192.168.1.100",
		},
		{
			name: "X-Real-IP header",
			setup: func(r *http.Request) {
				r.Header.Set("X-Real-IP", "10.0.0.1")
			},
			expected: "10.0.0.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			tc.setup(r)
			r.RemoteAddr = "127.0.0.1:1234"

			ip := getClientIP(r)
			if ip != tc.expected {
				t.Errorf("Expected IP %s, got %s", tc.expected, ip)
			}
		})
	}
}

func TestIsIPAllowed(t *testing.T) {
	blocklist := []string{"192.168.1.1", "10.0.0.1"}

	testCases := []struct {
		name     string
		ip       string
		expected bool
	}{
		{
			name:     "Blocked IP",
			ip:       "192.168.1.1",
			expected: false,
		},
		{
			name:     "Allowed IP",
			ip:       "8.8.8.8",
			expected: true,
		},
		{
			name:     "Another blocked IP",
			ip:       "10.0.0.1",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			allowed := IsIPAllowed(tc.ip, blocklist)
			if allowed != tc.expected {
				t.Errorf("Expected IP allowed=%v, got %v", tc.expected, allowed)
			}
		})
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	rl := NewRateLimiter(5, 10)
	defer rl.Close()

	// Create a bucket
	rl.Allow("cleanup_test")

	rl.mu.RLock()
	initialCount := len(rl.buckets)
	rl.mu.RUnlock()

	if initialCount != 1 {
		t.Errorf("Expected 1 bucket, got %d", initialCount)
	}
}

func TestRateLimiterConcurrency(t *testing.T) {
	rl := NewRateLimiter(100, 100)
	defer rl.Close()

	// Test concurrent requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(userID string) {
			for j := 0; j < 10; j++ {
				rl.Allow(userID)
			}
			done <- true
		}("user_" + string(rune(i)))
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
