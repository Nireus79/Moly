package main

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// Firefox drops a request that receives no data for five minutes (network.http.response.timeout); a reply from a slow local
// model can take longer, and the person sees neither an answer nor an error. For requests from a Firefox add-on the answer
// to a slow call therefore starts early: status 200, then a space every minute (leading white space is valid JSON) until the
// handler has its answer. Chrome has no such limit and is left alone. The price: a handler that fails after the first space
// can no longer change the status, so its error body arrives with status 200.

var keepAliveEvery = 60 * time.Second

func needsKeepAlive(r *http.Request) bool {
	return r.URL.Path == "/api/message-processor" && strings.HasPrefix(r.Header.Get("Origin"), "moz-extension://")
}

type keepAliveWriter struct {
	w         http.ResponseWriter
	mu        sync.Mutex
	hdr       http.Header // what the handler sets; copied to the real headers when the handler commits
	committed bool        // the real status line is out
	early     bool        // ... and the keep-alive sent it
}

func (k *keepAliveWriter) Header() http.Header { return k.hdr }

func (k *keepAliveWriter) commit(code int) {
	for name, values := range k.hdr {
		k.w.Header()[name] = values
	}
	k.w.WriteHeader(code)
	k.committed = true
}

func (k *keepAliveWriter) WriteHeader(code int) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if !k.committed {
		k.commit(code)
	}
}

func (k *keepAliveWriter) Write(b []byte) (int, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if !k.committed {
		k.commit(http.StatusOK)
	}
	return k.w.Write(b)
}

func (k *keepAliveWriter) beat() {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.committed && !k.early {
		return // the handler has begun its real answer
	}
	if !k.committed {
		k.w.Header().Set("Content-Type", "application/json")
		k.w.Header().Set("Access-Control-Allow-Origin", k.hdr.Get("Access-Control-Allow-Origin"))
		k.w.Header().Set("Vary", "Origin")
		k.w.WriteHeader(http.StatusOK)
		k.committed, k.early = true, true
	}
	k.w.Write([]byte(" "))
	if f, ok := k.w.(http.Flusher); ok {
		f.Flush()
	}
}

// withKeepAlive runs next and, while it has not answered, keeps the connection busy.
func withKeepAlive(w http.ResponseWriter, r *http.Request, next http.Handler) {
	k := &keepAliveWriter{w: w, hdr: http.Header{}}
	for name, values := range w.Header() {
		k.hdr[name] = values // CORS and content type already set by the middleware
	}
	done := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		t := time.NewTicker(keepAliveEvery)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				k.beat()
			}
		}
	}()
	defer func() {
		close(done)
		<-finished
	}()
	next.ServeHTTP(k, r)
}
