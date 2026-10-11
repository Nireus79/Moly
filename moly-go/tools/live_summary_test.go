package tools

import (
	"context"
	"os"
	"testing"
)

// TestLiveSummaryAcrossTurns rewrites the summary turn by turn with the real model (MOLY_LIVE=1). It only reports.
func TestLiveSummaryAcrossTurns(t *testing.T) {
	if os.Getenv("MOLY_LIVE") == "" {
		t.Skip()
	}
	os.Setenv("LLM_PROVIDER", "ollama")
	llm, err := NewLLMClient()
	if err != nil {
		t.Fatal(err)
	}
	cs := NewConversationSummarizer(llm)
	anchors := SummaryAnchors{Goal: "Ask Dana to share the extra tasks more fairly", People: []string{"Dana (the user's manager)", "Mark (a colleague)"}}
	turns := [][2]string{
		{"My manager Dana keeps giving me extra tasks and I can't keep up.", "What would you like Dana to do about it?"},
		{"Mark, a colleague, has been covering for me and he is getting tired of it. I want Dana to share the tasks without making Mark look like he complained.", "Would you like me to draft a message to Dana?"},
		{"Yes, but keep him out of it, please.", "Hi Dana, I have been carrying many extra tasks lately and would like us to share them more evenly. Could we talk this week?"},
		{"It is good but a bit too formal.", "Hi Dana, I'm swamped with all the extra tasks lately - can we split them more evenly? Free to chat this week."},
	}
	memory := ""
	for i, turn := range turns {
		out, err := cs.SummarizeTurn(context.Background(), memory, anchors, turn[0], turn[1])
		if err != nil {
			t.Fatalf("turn %d: %v", i+1, err)
		}
		memory = out
		t.Logf("after turn %d (%d chars): %s", i+1, len([]rune(out)), out)
	}
}
