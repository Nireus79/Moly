package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// How the last message of a user ended, for a screen that lost its connection while the reply was being written (the phone app:
// Android closes the screen while the engine, a foreground service, goes on). The engine saves a turn only when it succeeds, so
// without this a failed turn leaves nothing behind and the screen would wait for a reply that never comes.
//
// State: "none" (no message since the engine started: a turn that was running is gone), "running", "done", "failed".
// The record lives in memory only.

const (
	turnNone    = "none"
	turnRunning = "running"
	turnDone    = "done"
	turnFailed  = "failed"
)

type turnRecord struct {
	State     string `json:"state"`
	StartedAt int64  `json:"startedAt"` // Unix milliseconds
	Error     string `json:"error,omitempty"`
	id        uint64 // which turn this record belongs to
}

type turnTracker struct {
	last sync.Map // userID -> turnRecord
	seq  atomic.Uint64
}

var turns = &turnTracker{}

func (t *turnTracker) get(userID string) turnRecord {
	if v, ok := t.last.Load(userID); ok {
		return v.(turnRecord)
	}
	return turnRecord{State: turnNone}
}

// recordedWriter keeps the status code and the beginning of the body (an error text) of what a handler answered.
type recordedWriter struct {
	http.ResponseWriter
	status int
	body   []byte
}

func (w *recordedWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *recordedWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if room := 1024 - len(w.body); room > 0 {
		if len(b) < room {
			room = len(b)
		}
		w.body = append(w.body, b[:room]...)
	}
	return w.ResponseWriter.Write(b)
}

func (w *recordedWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// track runs one turn and records how it ended. Only the newest turn of a user is kept.
func (t *turnTracker) track(userID string, w http.ResponseWriter, r *http.Request, turn http.HandlerFunc) {
	started, id := time.Now().UnixMilli(), t.seq.Add(1)
	t.last.Store(userID, turnRecord{State: turnRunning, StartedAt: started, id: id})
	rw := &recordedWriter{ResponseWriter: w}
	finish := func(rec turnRecord) {
		if cur, ok := t.last.Load(userID); ok && cur.(turnRecord).id == id {
			rec.StartedAt, rec.id = started, id
			t.last.Store(userID, rec)
		}
	}
	defer func() {
		if p := recover(); p != nil {
			finish(turnRecord{State: turnFailed, Error: "Moly stopped while writing the reply."})
			panic(p)
		}
	}()
	turn(rw, r)
	if rw.status == 0 || rw.status < 400 {
		finish(turnRecord{State: turnDone})
		return
	}
	finish(turnRecord{State: turnFailed, Error: errorText(rw.body, rw.status)})
}

// errorText is the "error" of a JSON error answer, or a general sentence.
func errorText(body []byte, status int) string {
	var e struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error != "" {
		return e.Error
	}
	if status == http.StatusServiceUnavailable || status == http.StatusLocked {
		return "Moly is still starting, or the AI model is not answering."
	}
	return "Moly could not answer."
}

// TurnStatusHandler: GET /api/turn-status
func (srv *APIServer) TurnStatusHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := extractAndValidateToken(r, srv.database)
	if err != nil {
		denyJSON(w, http.StatusUnauthorized, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, turns.get(userID))
}
