package agents

import (
	"context"
	"os"
	"testing"

	"moly/tools"
)

// A live check against a real model (off by default): messages that state no goal and name no person must not produce an
// invented goal or person. Run with MOLY_LIVE_LLM=1 and LLM_PROVIDER set (for example claude-cli).
func TestLiveExtractionInventsNothingFromAMessageWithNoGoal(t *testing.T) {
	if os.Getenv("MOLY_LIVE_LLM") != "1" {
		t.Skip("set MOLY_LIVE_LLM=1 to run against a real model")
	}
	client, err := tools.NewLLMClient()
	if err != nil {
		t.Fatal(err)
	}
	ce := NewContextExtractor(client)
	for _, msg := range []string{"Thanks", "ok", "Hi Moly", "Yes", "I don't know", "That makes sense.", "Hmm, let me think about it.", "What can you do?"} {
		for run := 0; run < 3; run++ {
			ec, err := ce.Extract(context.Background(), msg)
			if err != nil {
				t.Fatal(err)
			}
			if ec.Intention != "" || ec.Contact != nil || len(ec.OtherContacts) > 0 {
				t.Errorf("%q run %d: invented intention=%q contact=%v others=%d", msg, run, ec.Intention, ec.Contact, len(ec.OtherContacts))
			}
		}
	}
}
