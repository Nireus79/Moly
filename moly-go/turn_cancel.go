package main

import (
	"context"
	"net/http"
	"sync"

	"moly/tools"
)

// Stop: a person who presses Stop ends the turn they started. Without it the turn runs on to the end, up to a dozen paid model calls, and then
// saves a turn the person never saw (found with a tester's report, 2026-10-29: the text box was locked until the reply came).
// A closed connection does NOT stop a turn: on the phone the engine goes on while Android closes the screen (turn_status.go). Only
// POST /api/turn-stop does. The model calls made for the user while the turn runs are cancelled, the turn fails, and the memory is
// rolled back (rollbackUnlessCommitted). Work that outlives the turn (the summary, the title) is not cancelled: the entry is removed
// when the handler returns.
var runningTurns stoppableTurns

type stoppableTurns struct{ m sync.Map } // user id -> *stoppable

type stoppable struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// begin registers the running turn of a user. The returned function ends it; call it when the handler returns.
func (s *stoppableTurns) begin(userID string) func() {
	if userID == "" {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	e := &stoppable{ctx: ctx, cancel: cancel}
	s.m.Store(userID, e)
	return func() { s.m.CompareAndDelete(userID, e) }
}

// stop cancels the running turn of a user; it reports whether there was one.
func (s *stoppableTurns) stop(userID string) bool {
	v, ok := s.m.Load(userID)
	if ok {
		v.(*stoppable).cancel()
	}
	return ok
}

// stopsWithTurn wraps the model provider of one user so that its calls end when the running turn is stopped.
type stopsWithTurn struct {
	inner  tools.LLMProvider
	userID string
}

func (c stopsWithTurn) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	v, ok := runningTurns.m.Load(c.userID)
	if !ok {
		return c.inner.Call(ctx, req)
	}
	merged, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := context.AfterFunc(v.(*stoppable).ctx, cancel)
	defer stop()
	return c.inner.Call(merged, req)
}

// TurnStopHandler: POST /api/turn-stop
func (srv *APIServer) TurnStopHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := extractAndValidateToken(r, srv.database)
	if err != nil {
		denyJSON(w, http.StatusUnauthorized, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]bool{"stopped": runningTurns.stop(userID)})
}
