package agents

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"moly/database"
	"moly/models"
	"moly/tools"
)

var seedCounter atomic.Int64

// seedMemory writes a case's memory into a fresh store, keeping its ids (p3 stays p3, q1 stays q1), so the update can be applied to it.
func seedMemory(t testing.TB, mem models.Memory) (*database.MemoryStore, *sql.DB) {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	db, err := database.OpenIndependent(filepath.Join(t.TempDir(), fmt.Sprintf("m%d.db", seedCounter.Add(1))), key)
	if err != nil {
		t.Fatal(err)
	}
	c := db.GetConnection()
	now := time.Now().Unix()
	must := func(q string, args ...interface{}) {
		if _, err := c.Exec(q, args...); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	must(`INSERT INTO users (id, created_at, last_active) VALUES ('u1', ?, ?)`, now, now)
	must(`INSERT INTO conversations (id, user_id, name, created_at, updated_at) VALUES ('c1', 'u1', 'n', ?, ?)`, now, now)
	num := func(id string) int64 {
		var n int64
		fmt.Sscanf(id[1:], "%d", &n)
		return n
	}
	for _, p := range mem.People {
		must(`INSERT INTO memory_people (id, user_id, conversation_id, name, label, name_status, created_at, updated_at) VALUES (?, 'u1', 'c1', ?, ?, ?, ?, ?)`,
			num(p.ID), p.Name, p.Label, p.NameStatus, now, now)
	}
	for _, p := range mem.People {
		for _, r := range p.Roles {
			var of interface{}
			if r.Of != models.MemoryUser {
				of = num(r.Of)
			}
			must(`INSERT INTO memory_roles (user_id, person_id, role, of_person_id, former, created_at) VALUES ('u1', ?, ?, ?, ?, ?)`, num(p.ID), r.Role, of, r.Former, now)
		}
		for _, tr := range p.Traits {
			must(`INSERT INTO memory_traits (user_id, person_id, trait, created_at) VALUES ('u1', ?, ?, ?)`, num(p.ID), tr, now)
		}
	}
	for _, q := range mem.Questions {
		var about interface{}
		if q.About != "" {
			about = num(q.About)
		}
		must(`INSERT INTO memory_questions (id, user_id, conversation_id, kind, about_person_id, text, topic, state, created_at) VALUES (?, 'u1', 'c1', ?, ?, ?, ?, ?, ?)`,
			num(q.ID), q.Kind, about, q.Text, q.Topic, q.State, now)
	}
	if mem.Goal != "" {
		must(`INSERT INTO memory_goals (conversation_id, user_id, goal, updated_at) VALUES ('c1', 'u1', ?, ?)`, mem.Goal, now)
	}
	return database.NewMemoryStore(c), c
}

// TestLiveMemoryUpdateBattery runs the memory update call on every case of testdata/memory_cases.json
// (MOLY_LIVE_LLM=1; MOLY_LIVE_REPLAYS=n, default 2; MOLY_LIVE_GROUPS=identity,roles to run some groups; MOLY_MODEL_REPLY=sonnet to use
// Sonnet for the reply role). It reports the rate per case and per group, and prints every miss with what the call returned.
func TestLiveMemoryUpdateBattery(t *testing.T) {
	if os.Getenv("MOLY_LIVE_LLM") != "1" {
		t.Skip("set MOLY_LIVE_LLM=1 to run against a real model")
	}
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Fatal(err)
	}
	replays := 2
	if v := os.Getenv("MOLY_LIVE_REPLAYS"); v != "" {
		fmt.Sscan(v, &replays)
	}
	groups := map[string]bool{}
	for _, g := range strings.Split(os.Getenv("MOLY_LIVE_GROUPS"), ",") {
		if g = strings.TrimSpace(g); g != "" {
			groups[g] = true
		}
	}
	type outcome struct {
		ok, stored bool // the proposed operations meet the case; the operations the store accepted meet it
		why, raw   string
		rejected   int
		took       time.Duration
	}
	var cases []memoryCase
	for _, mc := range loadMemoryCases(t, "testdata/memory_cases.json") {
		if len(groups) == 0 || groups[mc.Group] {
			cases = append(cases, mc)
		}
	}
	results := make([][]outcome, len(cases))
	var wg sync.WaitGroup
	parallel := 4
	if os.Getenv("MOLY_LIVE_SEQ") == "1" {
		parallel = 1 // one call at a time: the durations then mean something
	}
	sem := make(chan struct{}, parallel)
	for i := range cases {
		results[i] = make([]outcome, replays)
		for r := 0; r < replays; r++ {
			wg.Add(1)
			go func(i, r int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				mc := cases[i]
				var mem models.Memory
				if err := json.Unmarshal(mc.Memory, &mem); err != nil {
					results[i][r] = outcome{why: "bad memory: " + err.Error()}
					return
				}
				var recent []MemoryRecent
				for _, m := range mc.Recent {
					recent = append(recent, MemoryRecent{Role: m["role"], Text: m["text"]})
				}
				started := time.Now()
				up, err := RunMemoryUpdate(context.Background(), c, mem, recent, mc.Message)
				took := time.Since(started)
				if err != nil {
					results[i][r] = outcome{why: err.Error()}
					return
				}
				var ops []map[string]interface{}
				raw, _ := json.Marshal(up.Ops)
				json.Unmarshal(raw, &ops)
				ok, why := scoreMemoryCase(mc, ops, map[string]string{"action": up.Next.Action, "kind": up.Next.Kind})
				nx, _ := json.Marshal(up.Next)
				// the same operations through the store's checks: what would really be stored
				store, _ := seedMemory(t, mem)
				sources := []string{mc.Message}
				for _, m := range recent {
					sources = append(sources, m.Text)
				}
				applied, rejected := []map[string]interface{}{}, 0
				if res, aerr := store.Apply(database.ApplyOptions{UserID: "u1", ConversationID: "c1", MessageID: "m", Sources: sources}, up.Ops); aerr == nil {
					for _, rs := range res {
						if !rs.Applied {
							rejected++
							continue
						}
						one, _ := json.Marshal(rs.Op)
						var m map[string]interface{}
						json.Unmarshal(one, &m)
						applied = append(applied, m)
					}
				}
				after, _ := store.Snapshot("u1", "c1")
				guarded, _ := GuardNext(after, up.Next) // the question discipline in code
				stored, _ := scoreMemoryCase(mc, applied, map[string]string{"action": guarded.Action, "kind": guarded.Kind})
				results[i][r] = outcome{ok: ok, stored: stored, why: why, raw: string(raw) + " next=" + string(nx), rejected: rejected, took: took}
			}(i, r)
		}
	}
	wg.Wait()
	var took []time.Duration
	storedRight, rejectedTotal := 0, 0
	for i := range results {
		for _, o := range results[i] {
			if o.took > 0 {
				took = append(took, o.took)
			}
			if o.stored {
				storedRight++
			}
			rejectedTotal += o.rejected
		}
	}
	sort.Slice(took, func(a, b int) bool { return took[a] < took[b] })
	if len(took) > 0 {
		t.Logf("latency: median %v, 90th %v, longest %v (%d calls, %d at a time)", took[len(took)/2], took[len(took)*9/10], took[len(took)-1], len(took), parallel)
	}
	t.Logf("after the store's checks: %d/%d meet their case; the store rejected %d operations", storedRight, len(cases)*replays, rejectedTotal)
	perGroup := map[string][2]int{}
	for i, mc := range cases {
		ok := 0
		for _, o := range results[i] {
			if o.ok {
				ok++
			}
		}
		g := perGroup[mc.Group]
		g[0] += ok
		g[1] += replays
		perGroup[mc.Group] = g
		if ok < replays {
			for _, o := range results[i] {
				if !o.ok {
					t.Logf("MISS %s (%d/%d): %s\n      got %s", mc.ID, ok, replays, o.why, o.raw)
					break
				}
			}
		}
	}
	var names []string
	total, right := 0, 0
	for g, v := range perGroup {
		names = append(names, g)
		right += v[0]
		total += v[1]
	}
	sort.Strings(names)
	for _, g := range names {
		t.Logf("group %-10s %d/%d", g, perGroup[g][0], perGroup[g][1])
	}
	t.Logf("memory update battery: %d/%d", right, total)
}
