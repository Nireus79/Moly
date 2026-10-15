package agents

import (
	"context"
	"os"
	"testing"

	"moly/tools"
)

// A live check (off by default): a nickname or an online username given by the user is a name (NameKnown), a role is not. Found live,
// 2026-10-15: a username was taken for a description, and Moly asked for a name for the person it had just named.
func TestLiveExtractionTakesAUsernameForAName(t *testing.T) {
	if os.Getenv("MOLY_LIVE_LLM") == "" {
		t.Skip()
	}
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Skip(err)
	}
	ce := NewContextExtractor(c)
	cases := []struct {
		msg, exch string
		wantKnown bool
	}{
		{"NightOwl88", "Moly: Can you give me a name for the woman you mentioned?\n", true},
		{"Her username is NightOwl88 and I want to write to her.", "", true},
		{"We still talk about NightOwl88", "Moly: Can you give me a name for the woman you mentioned?\n", true},
		{"I need to write to my manager.", "", false},
		{"Help me write a message to my boss about a problem.", "", false},
		{"Her name is Dana.", "Moly: Can you give me a name for the manager you mentioned?\n", true},
		{"I want to ask my sister to lend me some money.", "", false},
	}
	for _, k := range cases {
		ok := 0
		for i := 0; i < 3; i++ {
			ec, err := ce.ExtractWithExchange(context.Background(), k.msg, k.exch)
			if err != nil {
				t.Fatal(err)
			}
			if ec.Contact != nil && ec.Contact.NameKnown == k.wantKnown {
				ok++
			}
		}
		t.Logf("%d/3 NameKnown=%v for %q", ok, k.wantKnown, k.msg)
		if ok < 2 {
			t.Errorf("NameKnown=%v expected for %q, got it %d of 3 times", k.wantKnown, k.msg, ok)
		}
	}
}
