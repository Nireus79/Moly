package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestActivityTrackerCountsUseNotTheServicesOwnCalls(t *testing.T) {
	clock := time.Unix(1000, 0)
	tr := newActivityTracker()
	tr.now = func() time.Time { return clock }
	tr.lastEnd.Store(clock.UnixNano())

	release := make(chan struct{})
	entered := make(chan struct{})
	h := tr.wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/chat" {
			close(entered)
			<-release
		}
	}))

	done := make(chan struct{})
	go func() {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/chat", nil))
		close(done)
	}()
	<-entered
	clock = clock.Add(10 * time.Minute)
	if got := tr.IdleFor(); got != 0 {
		t.Fatalf("a request is in flight, idle should be 0, got %v", got)
	}
	close(release)
	<-done
	// The reply request ended at +10 min: idle counts from there.
	clock = clock.Add(90 * time.Second)
	if got := tr.IdleFor(); got != 90*time.Second {
		t.Fatalf("idle after the request ended = %v, want 90s", got)
	}
	// The service's own questions and the status check do not restart the idle clock.
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/status", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/internal/idle", nil))
	if got := tr.IdleFor(); got != 90*time.Second {
		t.Fatalf("status and internal calls must not count as use, idle = %v", got)
	}
}

func TestIdleEndpointNeedsTheToken(t *testing.T) {
	p := &securityPolicy{token: "t"}
	if !p.needsToken("/internal/idle") {
		t.Fatal("/internal/idle must need the session token")
	}
}
