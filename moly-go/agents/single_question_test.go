package agents

import (
	"context"
	"errors"
	"strings"
	"testing"

	"moly/tools"
)

type rewriteLLM struct {
	out   string
	err   error
	calls int
}

func (r *rewriteLLM) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	r.calls++
	return &tools.LLMResponse{Content: r.out}, r.err
}

func TestMeetsQuestionContract(t *testing.T) {
	long := strings.Repeat("word ", 46) + "?"
	cases := map[string]bool{
		"How formal should the first message be?": true,
		"":                                      false,
		"What do you want? How would she feel?": false,
		"This has no question.":                 false,
		long:                                    false,
	}
	for text, want := range cases {
		if got := meetsQuestionContract(text); got != want {
			t.Errorf("meetsQuestionContract(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestFirstQuestion(t *testing.T) {
	got := firstQuestion("I understand you still want to proceed. But consider: how would she feel? What is the worst outcome?")
	if got != "But consider: how would she feel?" {
		t.Fatalf("got %q", got)
	}
	if firstQuestion("No question here.") != "" {
		t.Fatal("no question, no fallback")
	}
}

func TestSingleQuestionKeepsAGoodQuestionWithoutAModelCall(t *testing.T) {
	llm := &rewriteLLM{out: "never used?"}
	ca := &conversationAgent{llmClient: llm}
	q := "How formal should the first message be?"
	if got := ca.singleQuestion("x", q); got != q || llm.calls != 0 {
		t.Fatalf("got %q after %d calls", got, llm.calls)
	}
}

func TestSingleQuestionRewritesACompoundQuestion(t *testing.T) {
	llm := &rewriteLLM{out: "\"What do you hope Christine feels when she reads it?\""}
	ca := &conversationAgent{llmClient: llm}
	got := ca.singleQuestion("x", "How would she feel? What is the worst outcome? What else could you do?")
	if got != "What do you hope Christine feels when she reads it?" || llm.calls != 1 {
		t.Fatalf("got %q after %d calls", got, llm.calls)
	}
}

func TestSingleQuestionFallsBackWhenTheRewriteBreaksTheContract(t *testing.T) {
	compound := "Intro sentence. How would she feel? What is the worst outcome?"
	for _, llm := range []*rewriteLLM{{out: "One? Two?"}, {err: errors.New("down")}} {
		ca := &conversationAgent{llmClient: llm}
		if got := ca.singleQuestion("x", compound); got != "How would she feel?" {
			t.Fatalf("got %q", got)
		}
	}
	if got := (&conversationAgent{}).singleQuestion("x", compound); got != "How would she feel?" {
		t.Fatalf("no model: got %q", got)
	}
}
