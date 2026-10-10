package agents

import (
	"context"
	"errors"
	"testing"

	"moly/models"
)

func TestJudgeSwitchAnswerOnlyAClearYesCounts(t *testing.T) {
	cases := map[string]SwitchAnswer{
		`{"answer": "yes"}`:     SwitchYes,
		`{"answer": "No"}`:      SwitchNo,
		`{"answer": "unclear"}`: SwitchUnclear,
		`{"answer": "maybe"}`:   SwitchUnclear,
		`not json`:              SwitchUnclear,
	}
	for out, want := range cases {
		if got := JudgeSwitchAnswer(context.Background(), &rewriteLLM{out: out}, "q", "a", "g"); got != want {
			t.Errorf("%s: got %s, want %s", out, got, want)
		}
	}
	if JudgeSwitchAnswer(context.Background(), &rewriteLLM{err: errors.New("down")}, "q", "a", "g") != SwitchUnclear {
		t.Fatal("a failed judgement must not change the lock")
	}
	if JudgeSwitchAnswer(context.Background(), &rewriteLLM{out: `{"answer":"yes"}`}, "q", "", "g") != SwitchUnclear {
		t.Fatal("an empty answer is not a yes")
	}
}

func TestLastAssistantTextSkipsTheCurrentMessage(t *testing.T) {
	history := []models.Message{
		{Role: "user", Content: "current"},
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "first"},
		{Role: "user", Content: "mid"},
		{Role: "assistant", Content: "latest"},
	}
	if got := LastAssistantText(history); got != "latest" {
		t.Fatalf("got %q", got)
	}
	if LastAssistantText(history[:2]) != "" {
		t.Fatal("no assistant message, no text")
	}
}
