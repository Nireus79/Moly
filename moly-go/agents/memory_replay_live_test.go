package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// TestLiveMemoryReplay is step 4 of docs/MEMORY_REDESIGN.md: it replays saved conversations (MOLY_REPLAY_FILES=a.json,b.json: transcripts
// of scripts/dynamic/run.py, with the people and the goal the OLD pipeline stored) through the update call and the store, and prints
// the memory the new path ends with next to what the old one stored. Moly's own questions come from the transcript, so the memory
// meets the same dialogue. It only reports; the differences are read by hand.
func TestLiveMemoryReplay(t *testing.T) {
	files := os.Getenv("MOLY_REPLAY_FILES")
	if os.Getenv("MOLY_LIVE_LLM") != "1" || files == "" {
		t.Skip("set MOLY_LIVE_LLM=1 and MOLY_REPLAY_FILES")
	}
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]string{"name_question": "name", "gap_question": "gap", "clarity_question": "clarity", "principle_question": "principle", "confirm_question": "confirm"}
	for _, f := range strings.Split(files, ",") {
		raw, err := os.ReadFile(strings.TrimSpace(f))
		if err != nil {
			t.Fatal(err)
		}
		var tr struct {
			ID    string `json:"id"`
			Turns []struct {
				User  string `json:"user"`
				Exit  string `json:"exit"`
				Reply string `json:"reply"`
			} `json:"turns"`
			State struct {
				Memory struct {
					People []string `json:"people"`
					Goal   string   `json:"goal"`
				} `json:"memory"`
			} `json:"state"`
		}
		if err := json.Unmarshal(raw, &tr); err != nil {
			t.Fatal(err)
		}
		store, conn := seedMemory(t, models.Memory{})
		_ = conn
		var recent []MemoryRecent
		var lines []string
		for i, turn := range tr.Turns {
			snap, _ := store.Snapshot("u1", "c1")
			started := time.Now()
			up, err := RunMemoryUpdate(context.Background(), c, snap, lastN(recent, 3), turn.User)
			if err != nil {
				lines = append(lines, fmt.Sprintf("turn %d: update failed: %v", i+1, err))
				recent = append(recent, MemoryRecent{Role: "user", Text: turn.User}, MemoryRecent{Role: "moly", Text: turn.Reply})
				continue
			}
			sources := []string{turn.User}
			for _, r := range lastN(recent, 3) {
				sources = append(sources, r.Text)
			}
			res, _ := store.Apply(database.ApplyOptions{UserID: "u1", ConversationID: "c1", MessageID: fmt.Sprintf("m%d", i+1), Sources: sources}, up.Ops)
			rej := ""
			for _, r := range res {
				if !r.Applied {
					rej += fmt.Sprintf(" [rejected %s: %s]", r.Op.Op, r.Reason)
				}
			}
			lines = append(lines, fmt.Sprintf("turn %2d (%v) next=%s/%s old-exit=%s%s", i+1, time.Since(started).Round(time.Second), up.Next.Action, up.Next.Kind, turn.Exit, rej))
			if kind, ok := kinds[turn.Exit]; ok {
				about := ""
				if kind == "name" {
					m, _ := store.Snapshot("u1", "c1")
					for _, p := range m.People {
						if p.NameStatus == "unnamed" {
							about = p.ID
						}
					}
				}
				topic := ""
				if up.Next.Action == "ask" && up.Next.Kind == kind {
					topic = up.Next.Topic
				}
				store.OpenQuestion("u1", "c1", fmt.Sprintf("m%d", i+1), kind, about, turn.Reply, topic)
			}
			recent = append(recent, MemoryRecent{Role: "user", Text: turn.User}, MemoryRecent{Role: "moly", Text: turn.Reply})
		}
		final, _ := store.Snapshot("u1", "c1")
		var people []string
		for _, p := range final.People {
			var roles []string
			for _, r := range p.Roles {
				who := r.Of
				former := ""
				if r.Former {
					former = "former "
				}
				roles = append(roles, former+r.Role+" of "+who)
			}
			people = append(people, fmt.Sprintf("%s %s/%s [%s] traits=%v", p.ID, p.Name, p.Label, strings.Join(roles, "; "), p.Traits))
		}
		t.Logf("=== %s (%s)\n%s\nNEW people:\n  %s\nNEW goal: %s\nNEW facts: %d\nOLD people: %v\nOLD goal: %s", tr.ID, filepath.Base(f), strings.Join(lines, "\n"),
			strings.Join(people, "\n  "), final.Goal, len(final.Facts), tr.State.Memory.People, tr.State.Memory.Goal)
	}
}

func lastN(r []MemoryRecent, n int) []MemoryRecent {
	if len(r) <= n {
		return r
	}
	return r[len(r)-n:]
}
