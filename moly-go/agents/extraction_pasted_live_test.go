package agents

import (
	"context"
	"os"
	"testing"

	"moly/tools"
)

// A live check (off by default): text the user pastes that another person wrote describes that person, not the user. Found live,
// 2026-10-15: a pasted profile written in the first person came back as the user's own traits and the author's wish as the user's goal.
func TestLiveExtractionPastedTextIsNotTheUser(t *testing.T) {
	if os.Getenv("MOLY_LIVE_LLM") == "" {
		t.Skip()
	}
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Skip(err)
	}
	ce := NewContextExtractor(c)
	pasted := "Dana, 34\nAbout me: I am quiet at first and I love hiking and old films. I want a partner who is patient and honest. " +
		"I don't like drama or being rushed. Looking for something slow and real, close to home.\nInterests: hiking, cinema, baking"
	cases := []struct {
		name, msg, exch string
		wantUserTraits  bool
	}{
		{"pasted profile", pasted, "Moly: What did you notice on her profile?\n", false},
		{"pasted profile, no question before", "Here is her profile:\n" + pasted, "", false},
		{"pasted message", "She wrote me this: \"I can't stand being rushed, and I am happiest outdoors. I need someone who listens.\"", "", false},
		{"own traits", "I am shy and I love hiking.", "", true},
	}
	for _, k := range cases {
		ok := 0
		for i := 0; i < 3; i++ {
			ec, err := ce.ExtractWithExchange(context.Background(), k.msg, k.exch)
			if err != nil {
				t.Fatal(err)
			}
			got := len(ec.UserCharacteristics)+len(ec.UserInterests) > 0
			if got == k.wantUserTraits {
				ok++
			}
		}
		t.Logf("%d/3 user traits=%v for %s", ok, k.wantUserTraits, k.name)
		if ok < 2 {
			t.Errorf("%s: user traits expected=%v, got that %d of 3 times", k.name, k.wantUserTraits, ok)
		}
	}
}
