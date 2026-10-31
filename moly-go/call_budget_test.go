package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"moly/database"
)

// callsBudget is the most model calls one message may cost. The numbers are today's (docs/NEXT_VERSION.md, D9 step A0) and only go down:
// when a change removes calls, lower the limit in the same commit, so the saving cannot creep back.
var callsBudget = map[string]int{
	"greeting":     5,
	"goal message": 8,
	"answer":       12,
	"pasted chat":  12,
}

// settledCalls waits until the model calls stop, because a reply starts background work (the memory summary, the title) that ends after it.
func settledCalls(calls *callLog) int {
	last := -1
	for i := 0; i < 100; i++ {
		calls.mu.Lock()
		n := len(calls.users)
		calls.mu.Unlock()
		if n == last {
			return n
		}
		last = n
		time.Sleep(300 * time.Millisecond)
	}
	return last
}

// TestCallBudget counts the model calls of one message of each common kind, background work included, and fails above the budget.
func TestCallBudget(t *testing.T) {
	// Its own database (Init is one per process): the memory tests number people from p1 in the shared one, so a test that runs before them must not leave people there.
	db, err := database.OpenIndependent(filepath.Join(t.TempDir(), "budget.db"), make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	calls := &callLog{}
	srv, err := NewAPIServer(scenarioSource{llm: scriptedLLM{}, calls: calls}, db)
	if err != nil {
		t.Fatal(err)
	}
	userID := fmt.Sprintf("budget_%d", time.Now().UnixNano())
	token := "token_" + userID
	harnessUser(t, db, userID, token)

	conv := ""
	step := func(kind, message string) {
		t.Helper()
		from := settledCalls(calls)
		reply, convID, code := sendMessage(t, srv, token, conv, message)
		conv = convID
		if code != http.StatusOK || strings.TrimSpace(reply) == "" {
			t.Fatalf("%s: status %d, reply %q", kind, code, reply)
		}
		n := settledCalls(calls) - from
		t.Logf("CALLS %-14s %d (budget %d)", kind, n, callsBudget[kind])
		if n > callsBudget[kind] {
			t.Errorf("%s made %d model calls, the budget is %d", kind, n, callsBudget[kind])
		}
	}
	step("greeting", "Hi Moly")
	step("goal message", "I want to write a first message to a girl I saw on fetlife.")
	step("answer", "Her name is Christine")
	step("pasted chat", "[12/10/2026, 18:20] Anna: Hey, did you get my message about Saturday?\n[12/10/2026, 18:22] Nikos: Yes. I can't make it, work again.\n[12/10/2026, 18:23] Anna: You always say that. I'm tired of planning everything alone.")
}
