package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// Activity for the phone app: its service stops the engine when nobody has used it for a while (the "Moly is running" notice
// goes away with it). It asks GET /internal/idle (token needed, see needsToken): how many requests are being served and how
// long it has been since the last one ended. The work Moly does after a reply is written (the summary, the title) starts
// before that request ends and finishes within seconds of it, which the service's wait covers.

type activityTracker struct {
	inFlight atomic.Int64
	lastEnd  atomic.Int64 // unix nanoseconds of the last request that ended (or of the start)
	now      func() time.Time
}

func newActivityTracker() *activityTracker {
	t := &activityTracker{now: time.Now}
	t.lastEnd.Store(t.now().UnixNano())
	return t
}

// counted says whether a request is the person's use of Moly; the status check and the internal calls (the service's own
// questions and heartbeats) are not.
func counted(path string) bool {
	return path != "/api/status" && !strings.HasPrefix(path, "/internal/")
}

func (t *activityTracker) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !counted(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		t.inFlight.Add(1)
		defer func() {
			t.lastEnd.Store(t.now().UnixNano())
			t.inFlight.Add(-1)
		}()
		next.ServeHTTP(w, r)
	})
}

// IdleFor is how long nothing has been served: zero while a request is in flight.
func (t *activityTracker) IdleFor() time.Duration {
	if t.inFlight.Load() > 0 {
		return 0
	}
	d := t.now().Sub(time.Unix(0, t.lastEnd.Load()))
	if d < 0 {
		return 0
	}
	return d
}

func (t *activityTracker) register(mux *http.ServeMux) {
	mux.HandleFunc("GET /internal/idle", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"inFlight":    t.inFlight.Load(),
			"idleSeconds": int64(t.IdleFor() / time.Second),
		})
	})
}
