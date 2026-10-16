package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// The battery for the memory update call (docs/MEMORY_REDESIGN.md, sections 3a and 3b). The cases are data: a memory, the last messages, a
// new message, the operations the update must contain (a subset match), the operations it must not contain, and the next action.
// Every bug found live becomes a case; "heldOut" cases never appear in a prompt. This file only loads, checks and scores cases; the live
// run against the update call comes with step 3 of the plan.

type memoryCase struct {
	ID         string                   `json:"id"`
	Group      string                   `json:"group"`
	HeldOut    bool                     `json:"heldOut"`
	Note       string                   `json:"note"`
	Memory     json.RawMessage          `json:"memory"`
	Recent     []map[string]string      `json:"recent"`
	Message    string                   `json:"message"`
	ExpectOps  []map[string]interface{} `json:"expect_ops"`
	ForbidOps  []map[string]interface{} `json:"forbid_ops"`
	ExpectNext map[string]string        `json:"expect_next"`
	ForbidNext map[string]string        `json:"forbid_next"` // a next action that must NOT be proposed (a question that should not be asked)
}

type memoryFile struct {
	Version int          `json:"version"`
	Cases   []memoryCase `json:"cases"`
}

var memoryOpNames = map[string]bool{
	"add_person": true, "name_person": true, "set_role": true, "correct_role": true, "add_trait": true, "set_goal": true,
	"propose_goal_switch": true, "close_question": true, "add_fact": true, "merge_person": true,
}

var memoryNextActions = map[string]bool{"answer": true, "ask": true, "confirm": true, "result_now": true}

func loadMemoryCases(t testing.TB, path string) []memoryCase {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("battery file: %v", err)
	}
	var f memoryFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("battery file is not valid: %v", err)
	}
	return f.Cases
}

// opMatches is true when every field of want is present in got with the same value (text compared without case).
func opMatches(got, want map[string]interface{}) bool {
	for k, w := range want {
		g, ok := got[k]
		if !ok {
			return false
		}
		switch wv := w.(type) {
		case string:
			gs, ok := g.(string)
			if !ok || !strings.EqualFold(strings.TrimSpace(gs), strings.TrimSpace(wv)) {
				return false
			}
		default:
			if fmt.Sprint(g) != fmt.Sprint(w) {
				return false
			}
		}
	}
	return true
}

// scoreMemoryCase says whether an update's output meets a case, and why not. next may be nil when the case expects none.
func scoreMemoryCase(c memoryCase, ops []map[string]interface{}, next map[string]string) (bool, string) {
	for _, want := range c.ExpectOps {
		found := false
		for _, got := range ops {
			if opMatches(got, want) {
				found = true
				break
			}
		}
		if !found {
			return false, fmt.Sprintf("missing operation %v", want)
		}
	}
	for _, bad := range c.ForbidOps {
		for _, got := range ops {
			if opMatches(got, bad) {
				return false, fmt.Sprintf("forbidden operation %v", got)
			}
		}
	}
	for k, want := range c.ExpectNext {
		if !strings.EqualFold(next[k], want) {
			return false, fmt.Sprintf("next %s is %q, want %q", k, next[k], want)
		}
	}
	for k, bad := range c.ForbidNext {
		if strings.EqualFold(next[k], bad) {
			return false, fmt.Sprintf("next %s is %q, which must not be proposed", k, bad)
		}
	}
	return true, ""
}

func TestMemoryBatteryFileIsWellFormed(t *testing.T) {
	cases := loadMemoryCases(t, "testdata/memory_cases.json")
	if len(cases) < 30 {
		t.Fatalf("the battery holds %d cases; the plan wants at least 30 to start", len(cases))
	}
	seen := map[string]bool{}
	for _, c := range cases {
		if c.ID == "" || seen[c.ID] {
			t.Errorf("case id %q is empty or repeated", c.ID)
		}
		seen[c.ID] = true
		if strings.TrimSpace(c.Message) == "" || c.Group == "" {
			t.Errorf("%s: a message and a group are required", c.ID)
		}
		var mem struct {
			People []struct {
				ID string `json:"id"`
			} `json:"people"`
			Questions []struct {
				ID    string `json:"id"`
				About string `json:"about"`
			} `json:"questions"`
		}
		if err := json.Unmarshal(c.Memory, &mem); err != nil {
			t.Errorf("%s: memory is not valid: %v", c.ID, err)
			continue
		}
		people, questions := map[string]bool{"user": true}, map[string]bool{}
		for _, p := range mem.People {
			if p.ID == "" || people[p.ID] {
				t.Errorf("%s: person id %q is empty or repeated", c.ID, p.ID)
			}
			people[p.ID] = true
		}
		for _, q := range mem.Questions {
			questions[q.ID] = true
			if !people[q.About] && q.About != "" {
				t.Errorf("%s: question %s is about %q, who is not in the memory", c.ID, q.ID, q.About)
			}
		}
		for _, list := range [][]map[string]interface{}{c.ExpectOps, c.ForbidOps} {
			for _, op := range list {
				name, _ := op["op"].(string)
				if !memoryOpNames[name] {
					t.Errorf("%s: unknown operation %q", c.ID, name)
				}
				if p, ok := op["person"].(string); ok && !people[p] {
					t.Errorf("%s: operation refers to person %q, who is not in the memory", c.ID, p)
				}
				if q, ok := op["question"].(string); ok && !questions[q] {
					t.Errorf("%s: operation refers to question %q, which is not in the memory", c.ID, q)
				}
			}
		}
		if a, ok := c.ExpectNext["action"]; ok && !memoryNextActions[a] {
			t.Errorf("%s: unknown next action %q", c.ID, a)
		}
		if len(c.ExpectOps) == 0 && len(c.ForbidOps) == 0 && len(c.ExpectNext) == 0 && len(c.ForbidNext) == 0 {
			t.Errorf("%s: a case that expects and forbids nothing tests nothing", c.ID)
		}
	}
}

func TestMemoryBatteryScoring(t *testing.T) {
	c := memoryCase{
		ExpectOps:  []map[string]interface{}{{"op": "add_person", "label": "Father"}},
		ForbidOps:  []map[string]interface{}{{"op": "set_role", "role": "family"}},
		ExpectNext: map[string]string{"action": "ask"},
	}
	good := []map[string]interface{}{{"op": "add_person", "label": "father", "evidence": "his father"}}
	if ok, why := scoreMemoryCase(c, good, map[string]string{"action": "ask"}); !ok {
		t.Fatalf("a matching output scores right, got %s", why)
	}
	if ok, _ := scoreMemoryCase(c, nil, map[string]string{"action": "ask"}); ok {
		t.Fatal("a missing operation fails")
	}
	bad := append(good, map[string]interface{}{"op": "set_role", "role": "family"})
	if ok, _ := scoreMemoryCase(c, bad, map[string]string{"action": "ask"}); ok {
		t.Fatal("a forbidden operation fails")
	}
	if ok, _ := scoreMemoryCase(c, good, map[string]string{"action": "answer"}); ok {
		t.Fatal("a wrong next action fails")
	}
}

// A held-out case must not occur in any prompt: an example in a prompt that is also a test case inflates the score.
func TestHeldOutMemoryCasesAreInNoPrompt(t *testing.T) {
	cases := loadMemoryCases(t, "testdata/memory_cases.json")
	files, _ := os.ReadDir(".")
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(f.Name())
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ToLower(string(src))
		for _, c := range cases {
			if c.HeldOut && strings.Contains(text, strings.ToLower(c.Message)) {
				t.Errorf("held-out case %s occurs in %s", c.ID, f.Name())
			}
		}
	}
}
