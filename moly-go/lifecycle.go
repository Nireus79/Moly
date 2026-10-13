package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Lifecycle for production mode: browsers (through the native host) attach and send heartbeats; when none has been
// alive for idleGrace the backend shuts itself down (MOLY_IDLE_EXIT=1). Several browsers share one backend.

const (
	hostTimeout = 30 * time.Second // a host that sent no heartbeat for this long is gone
	idleGrace   = 30 * time.Second // exit this long after the last host is gone (or after start, if none ever attached)
)

// startupSwitch answers 503 "starting" until the real handler is set, so the port can be taken before the database is open.
type startupSwitch struct {
	mu sync.RWMutex
	h  http.Handler
}

func (s *startupSwitch) Set(h http.Handler) {
	s.mu.Lock()
	s.h = h
	s.mu.Unlock()
}

func (s *startupSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	h := s.h
	s.mu.RUnlock()
	if h == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"starting"}`))
		return
	}
	h.ServeHTTP(w, r)
}

type hostRegistry struct {
	mu    sync.Mutex
	hosts map[string]time.Time // host ID -> last heartbeat
	since time.Time            // when there was last a live host (or the start)
	now   func() time.Time
}

func newHostRegistry() *hostRegistry {
	return &hostRegistry{hosts: map[string]time.Time{}, since: time.Now(), now: time.Now}
}

func (r *hostRegistry) liveLocked() int {
	n := 0
	for id, beat := range r.hosts {
		if r.now().Sub(beat) > hostTimeout {
			delete(r.hosts, id)
			continue
		}
		n++
	}
	return n
}

func (r *hostRegistry) Attach() string {
	raw := make([]byte, 8)
	rand.Read(raw)
	id := hex.EncodeToString(raw)
	r.mu.Lock()
	r.hosts[id] = r.now()
	r.since = r.now()
	r.mu.Unlock()
	return id
}

func (r *hostRegistry) Heartbeat(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.hosts[id]; !ok {
		return false
	}
	r.hosts[id] = r.now()
	r.since = r.now()
	return true
}

func (r *hostRegistry) Detach(id string) {
	r.mu.Lock()
	delete(r.hosts, id)
	r.since = r.now() // a full grace period after a clean detach: a reload or a restarted service worker can reattach
	r.mu.Unlock()
}

// Idle reports whether no host has been alive for idleGrace.
func (r *hostRegistry) Idle() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.liveLocked() > 0 {
		r.since = r.now()
		return false
	}
	return r.now().Sub(r.since) >= idleGrace
}

// register adds /internal/attach, /internal/heartbeat and /internal/detach (all need the token, see needsToken).
func (r *hostRegistry) register(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/attach", func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"hostId": r.Attach()})
	})
	mux.HandleFunc("POST /internal/heartbeat", func(w http.ResponseWriter, req *http.Request) {
		if !r.Heartbeat(req.URL.Query().Get("id")) {
			denyJSON(w, http.StatusNotFound, "unknown host")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /internal/detach", func(w http.ResponseWriter, req *http.Request) {
		r.Detach(req.URL.Query().Get("id"))
		w.WriteHeader(http.StatusNoContent)
	})
}

// registerShutdown adds POST /internal/shutdown (token needed, see needsToken): "moly --stop" and the uninstaller use it,
// because the process cannot be found from outside (it is non-dumpable, so /proc/<pid>/exe is unreadable).
func registerShutdown(mux *http.ServeMux, quit chan<- struct{}) {
	mux.HandleFunc("POST /internal/shutdown", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		select {
		case quit <- struct{}{}:
		default:
		}
	})
}
