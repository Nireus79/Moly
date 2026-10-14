package agents

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"moly/tools"
)

// The prompts carry examples. A prompt example must never turn up in what the model returns for a message that has nothing to do
// with it, and a message that mentions nobody must not get a person. Run against a real model (off by default):
//
//	MOLY_LIVE_LLM=1 LLM_PROVIDER=claude-cli AGENT_MODEL=haiku go test ./agents/ -run TestLiveExtractionDoesNotEchoPromptExamples -v
//
// MOLY_LIVE_REPLAYS sets the replays per message (default 3).
func TestLiveExtractionDoesNotEchoPromptExamples(t *testing.T) {
	if os.Getenv("MOLY_LIVE_LLM") != "1" {
		t.Skip("set MOLY_LIVE_LLM=1 to run against a real model")
	}
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Fatal(err)
	}
	replays := 3
	if v := os.Getenv("MOLY_LIVE_REPLAYS"); v != "" {
		fmt.Sscan(v, &replays)
	}
	// words that the extraction prompts use in their examples; none of them is in any message below
	echoes := []string{"sister", "party", "dana", "anna", "christine", "mark", "transparency"}
	type tc struct {
		msg      string
		nobody   bool // the message mentions no other person: a contact is invented
		personIn string
	}
	cases := []tc{
		{"Help me write a cover letter.", true, ""},
		{"I can't decide whether to switch careers.", true, ""},
		{"I feel exhausted after work every day.", true, ""},
		{"I am nervous about a presentation tomorrow.", true, ""},
		{"What can you do?", true, ""},
		{"Thank you, that helps.", true, ""},
		{"I want to understand why I always put off hard conversations.", true, ""},
		{"Πώς μπορώ να γράψω ένα καλό βιογραφικό;", true, ""},
		{"My landlord keeps ignoring my emails about the heating.", false, "landlord"},
		{"My neighbour plays loud music at night and I want to ask him to stop.", false, "neighbour"},
		{"I need to say no to an invitation without hurting anyone.", true, ""},
		{"I want to apologise to a friend I have not spoken to in years.", false, "friend"},
		{"Should I tell my team lead that I am looking for another job?", false, "team lead"},
		{"My grandfather is ill and I do not know what to say when I visit.", false, "grandfather"},
	}
	type res struct {
		tc
		got []string // "ok", "echo:<word>" or "phantom:<label>"
	}
	var all []*res
	for _, k := range cases {
		all = append(all, &res{tc: k, got: make([]string, replays)})
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for _, r := range all {
		for i := 0; i < replays; i++ {
			wg.Add(1)
			go func(r *res, i int) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				ec, err := NewContextExtractor(c).ExtractWithExchange(context.Background(), r.msg, "")
				if err != nil || ec == nil {
					r.got[i] = "error"
					return
				}
				var seen []string
				seen = append(seen, ec.Intention)
				for _, p := range ec.People() {
					seen = append(seen, p.Name, p.Label, p.Role, p.Evidence)
				}
				joined := strings.ToLower(strings.Join(seen, " | "))
				msg := strings.ToLower(r.msg)
				for _, e := range echoes {
					if strings.Contains(joined, e) && !strings.Contains(msg, e) {
						r.got[i] = "echo:" + e
						return
					}
				}
				if r.nobody && ec.Contact != nil {
					r.got[i] = "phantom:" + ec.Contact.Label
					return
				}
				r.got[i] = "ok"
			}(r, i)
		}
	}
	wg.Wait()
	bad, total := 0, 0
	for _, r := range all {
		for _, g := range r.got {
			total++
			if g != "ok" {
				bad++
				t.Logf("  %q -> %s", r.msg, g)
			}
		}
	}
	t.Logf("extraction leaks: %d of %d replays", bad, total)
	if bad > 0 {
		t.Fail()
	}
}
