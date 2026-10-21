package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func runTurn(t *testing.T, tr *turnTracker, user string, h http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	tr.track(user, rec, httptest.NewRequest(http.MethodPost, "/api/message-processor", nil), h)
	return rec
}

func TestTurnTrackerRecordsHowATurnEnds(t *testing.T) {
	tr := &turnTracker{}
	if got := tr.get("u1"); got.State != turnNone {
		t.Fatalf("before any message: %+v", got)
	}

	var during turnRecord
	runTurn(t, tr, "u1", func(w http.ResponseWriter, r *http.Request) {
		during = tr.get("u1")
		w.Write([]byte(`{"response":"hi"}`))
	})
	if during.State != turnRunning || during.StartedAt == 0 {
		t.Fatalf("while the turn runs: %+v", during)
	}
	if got := tr.get("u1"); got.State != turnDone || got.StartedAt != during.StartedAt {
		t.Fatalf("after success: %+v", got)
	}

	runTurn(t, tr, "u1", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "the safety check did not complete"})
	})
	if got := tr.get("u1"); got.State != turnFailed || got.Error != "the safety check did not complete" {
		t.Fatalf("after an error answer: %+v", got)
	}

	runTurn(t, tr, "u1", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusLocked) })
	if got := tr.get("u1"); got.State != turnFailed || got.Error == "" {
		t.Fatalf("an error without a body still gets a sentence: %+v", got)
	}

	if got := tr.get("u2"); got.State != turnNone {
		t.Fatalf("another user is not touched: %+v", got)
	}
}

func TestTurnTrackerPanicIsAFailure(t *testing.T) {
	tr := &turnTracker{}
	func() {
		defer func() { _ = recover() }()
		runTurn(t, tr, "u1", func(w http.ResponseWriter, r *http.Request) { panic("boom") })
	}()
	if got := tr.get("u1"); got.State != turnFailed {
		t.Fatalf("after a panic: %+v", got)
	}
}

func TestTurnTrackerKeepsOnlyTheNewestTurn(t *testing.T) {
	tr := &turnTracker{}
	runTurn(t, tr, "u1", func(w http.ResponseWriter, r *http.Request) {
		// a newer turn starts and ends while this one is still running
		runTurn(t, tr, "u1", func(w http.ResponseWriter, r *http.Request) {
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "newer failed"})
		})
		w.Write([]byte(`{}`))
	})
	if got := tr.get("u1"); got.State != turnFailed || got.Error != "newer failed" {
		t.Fatalf("the older turn must not overwrite the newer one: %+v", got)
	}
}
