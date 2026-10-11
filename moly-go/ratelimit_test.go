package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func fakeClock(rl *RateLimiter) *time.Time {
	now := time.Unix(1_000_000, 0)
	rl.now = func() time.Time { return now }
	return &now
}

func TestRateLimiterAllowsABurstThenRefuses(t *testing.T) {
	rl := NewRateLimiter()
	fakeClock(rl)
	for i := 0; i < 5; i++ {
		if ok, _ := rl.Allow(classAuth, "1.2.3.4"); !ok {
			t.Fatalf("request %d of the burst was refused", i+1)
		}
	}
	ok, wait := rl.Allow(classAuth, "1.2.3.4")
	if ok || wait <= 0 || wait > 13*time.Second {
		t.Fatalf("sixth request: ok=%v wait=%v, want refused with a wait of up to 12 s", ok, wait)
	}
}

func TestRateLimiterRefillsAndKeepsClientsApart(t *testing.T) {
	rl := NewRateLimiter()
	now := fakeClock(rl)
	for i := 0; i < 5; i++ {
		rl.Allow(classAuth, "a")
	}
	if ok, _ := rl.Allow(classAuth, "b"); !ok {
		t.Fatal("another client has its own bucket")
	}
	if ok, _ := rl.Allow(classModel, "a"); !ok {
		t.Fatal("another class has its own bucket")
	}
	*now = now.Add(13 * time.Second)
	if ok, _ := rl.Allow(classAuth, "a"); !ok {
		t.Fatal("a token comes back after 12 seconds")
	}
	if ok, _ := rl.Allow(classAuth, "a"); ok {
		t.Fatal("only one token came back")
	}
}

func TestRateLimiterDropsRefilledBucketsAndCapsTheirNumber(t *testing.T) {
	rl := NewRateLimiter()
	now := fakeClock(rl)
	rl.Allow(classGeneral, "a")
	*now = now.Add(time.Hour)
	rl.Allow(classGeneral, "b") // triggers the sweep
	if len(rl.buckets) != 1 {
		t.Fatalf("the refilled bucket is dropped, have %d buckets", len(rl.buckets))
	}
	rl2 := NewRateLimiter()
	fakeClock(rl2)
	for i := 0; i < maxBuckets; i++ {
		rl2.buckets[string(rune(i))+"x"] = &tokenBucket{tokens: 1, last: time.Unix(1_000_000, 0)}
	}
	if ok, _ := rl2.Allow(classGeneral, "new"); ok {
		t.Fatal("a new client is refused when the table is full")
	}
}

func TestRateLimitMiddlewareAnswers429WithRetryAfter(t *testing.T) {
	rl := NewRateLimiter()
	fakeClock(rl)
	h := rateLimitMiddleware(rl, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	do := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", path, nil)
		req.RemoteAddr = "9.9.9.9:5555"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	for i := 0; i < 5; i++ {
		if c := do("/api/auth/login").Code; c != 200 {
			t.Fatalf("login %d: %d", i+1, c)
		}
	}
	rec := do("/api/auth/login")
	if rec.Code != 429 || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("sixth login: code=%d retry-after=%q", rec.Code, rec.Header().Get("Retry-After"))
	}
	for i := 0; i < 50; i++ {
		if c := do("/api/status").Code; c != 200 {
			t.Fatalf("the health check is never limited, got %d", c)
		}
	}
	if c := do("/api/contacts").Code; c != 200 {
		t.Fatalf("another class is unaffected, got %d", c)
	}
}

func TestLimitClassByPath(t *testing.T) {
	for path, want := range map[string]string{
		"/api/auth/login": classAuth, "/api/auth/register": classAuth,
		"/api/message-processor": classModel, "/api/clarification/respond": classModel,
		"/api/contacts": classGeneral, "/api/auth/verify": classGeneral,
	} {
		if got, limited := limitClass(path); !limited || got != want {
			t.Errorf("%s: class %q limited=%v, want %q", path, got, limited, want)
		}
	}
	if _, limited := limitClass("/api/status"); limited {
		t.Error("/api/status is not limited")
	}
}
