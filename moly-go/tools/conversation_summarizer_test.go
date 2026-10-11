package tools

import (
	"context"
	"strings"
	"testing"
)

type captureLLM struct {
	out    string
	prompt *string
}

func (c captureLLM) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	*c.prompt = req.UserPrompt
	return &LLMResponse{Content: c.out}, nil
}

func TestSummarizeTurnGivesThePreviousSummaryTheFixedFactsAndBothSidesOfTheTurn(t *testing.T) {
	var prompt string
	cs := NewConversationSummarizer(captureLLM{out: "Memory: The user asks Dana to share the tasks. Mark covers for her.", prompt: &prompt})
	got, err := cs.SummarizeTurn(context.Background(), "OLD-MEMORY", SummaryAnchors{Goal: "Ask Dana to share tasks", People: []string{"Dana (the user's manager)"}},
		"Yes, but keep him out of it.", "Here is the draft to Dana.")
	if err != nil {
		t.Fatal(err)
	}
	if got != "The user asks Dana to share the tasks. Mark covers for her." {
		t.Fatalf("the label is removed, got %q", got)
	}
	for _, want := range []string{"OLD-MEMORY", "Ask Dana to share tasks", "Dana (the user's manager)", "User: Yes, but keep him out of it.", "Moly: Here is the draft to Dana."} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("the writer must be given %q:\n%s", want, prompt)
		}
	}
}

func TestCleanSummary(t *testing.T) {
	for in, want := range map[string]string{
		"```\nThe user wants X.\n```": "The user wants X.",
		"Summary: The user wants X.":  "The user wants X.",
		"\"The user wants X.\"":       "The user wants X.",
		`{"arc": "x"}`:                "",
		"[1, 2]":                      "",
		"   ":                         "",
	} {
		if got := cleanSummary(in); got != want {
			t.Errorf("cleanSummary(%q) = %q, want %q", in, got, want)
		}
	}
	long := strings.Repeat("A sentence that goes on. ", 100)
	if got := cleanSummary(long); len([]rune(got)) > maxSummaryRunes || !strings.HasSuffix(got, ".") {
		t.Fatalf("a long summary is cut at the end of a sentence within %d characters, got %d", maxSummaryRunes, len([]rune(got)))
	}
}

func TestSummarizeTurnFailsSafe(t *testing.T) {
	var prompt string
	cs := NewConversationSummarizer(captureLLM{out: `{"not": "prose"}`, prompt: &prompt})
	if _, err := cs.SummarizeTurn(context.Background(), "", SummaryAnchors{}, "hello there", ""); err == nil {
		t.Fatal("an unusable answer is an error, so the caller keeps the previous summary")
	}
	if _, err := NewConversationSummarizer(nil).SummarizeTurn(context.Background(), "", SummaryAnchors{}, "x", ""); err == nil {
		t.Fatal("no model, no summary")
	}
	if _, err := cs.SummarizeTurn(context.Background(), "", SummaryAnchors{}, "  ", ""); err == nil {
		t.Fatal("nothing to summarize")
	}
}
