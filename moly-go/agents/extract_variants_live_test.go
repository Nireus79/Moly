package agents

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"moly/models"
	"moly/tools"
)

// A battery for the extraction prompt (MOLY_LIVE=1, e.g. LLM_PROVIDER=claude-cli): how often does a prompt fill the intention and
// the person where the message states them, and leave them empty where it does not. It only reports.
type extractCase struct {
	msg                        string
	exchange                   string
	wantIntention, wantContact bool
}

var extractBattery = []extractCase{
	{"I need to write to my manager.", "", true, true},
	{"Help me write a message to my boss about a problem.", "", true, true},
	{"Write a two-sentence thank-you note to my colleague Anna for covering my shift yesterday, warm and informal.", "", true, true},
	{"Maybe I should write to someone about it, I am not sure.", "", false, false},
	{"My boss is unfair to me, my sister is getting married next month, and I think my neighbour is spying on me. I do not know where to start.", "", true, true},
	{"I want to ask my sister Maria to lend me some money.", "", true, true},
	{"How do I convince my girlfriend to move in with me before I tell her family?", "", true, true},
	{"Yes", "Moly: Would you like me to draft a message to Dana asking her to share the tasks?\n", false, false},
	{"Her name is Dana.", "Moly: Can you give me a name for the manager you mentioned?\n", false, true},
	{"It is good but a bit too formal.", "Moly: Dear Dana, I would like to discuss my workload. Kind regards.\n", false, false},
}


func TestLiveExtractionPromptBattery(t *testing.T) {
	if os.Getenv("MOLY_LIVE") == "" {
		t.Skip()
	}
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Fatal(err)
	}
	ce := NewContextExtractor(c)
	runs := 3
	if v := os.Getenv("MOLY_RUNS"); v != "" {
		fmt.Sscanf(v, "%d", &runs)
	}
	type variant struct {
		name  string
		build func(c extractCase) string
	}
	variants := []variant{
		{"current", func(c extractCase) string {
			return ce.buildExtractionPrompt(c.msg, models.WithExchange(c.exchange))
		}},
		{"short", func(c extractCase) string { return shortExtractionPrompt(c.msg, models.WithExchange(c.exchange)) }},
	}
	for _, v := range variants {
		right, total := 0, 0
		var lines []string
		for _, cs := range extractBattery {
			ok := 0
			for i := 0; i < runs; i++ {
				resp, err := c.Call(context.Background(), &tools.LLMRequest{UserPrompt: v.build(cs), Temperature: 0.3, MaxTokens: 1200})
				total++
				if err != nil {
					continue
				}
				got, perr := ce.parseLLMExtraction(resp.Content)
				if perr != nil {
					continue
				}
				hasI := got.Intention != ""
				hasC := got.Contact != nil && (got.Contact.Name != "" || got.Contact.Label != "")
				if hasI == cs.wantIntention && hasC == cs.wantContact {
					ok++
					right++
				}
			}
			lines = append(lines, fmt.Sprintf("   %d/%d  %.60q", ok, runs, cs.msg))
		}
		t.Logf("prompt %-8s %d/%d right\n%s", v.name, right, total, strings.Join(lines, "\n"))
	}
}
