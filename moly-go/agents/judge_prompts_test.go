package agents

import (
	"context"
	"strings"
	"testing"

	"moly/tools"
)

type recordingLLM struct {
	out    string
	prompt string
}

func (r *recordingLLM) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	r.prompt = req.UserPrompt
	return &tools.LLMResponse{Content: r.out}, nil
}

// Measured on Claude haiku (live run 2026-10-11): a prompt that only offers "yes / no / unclear" without saying what each
// means made the model answer "unclear" to a plain "Yes, switch." (3 of 8 replays) and "none" ("not a person") to
// "girlfriend". The prompts name what each answer means; these tests keep them from being shortened back.
func TestGoalSwitchJudgeNamesWhatEachAnswerMeans(t *testing.T) {
	llm := &recordingLLM{out: `{"answer": "yes"}`}
	if got := JudgeSwitchAnswer(context.Background(), llm, "Switch to X?", "Yes, switch.", "X"); got != SwitchYes {
		t.Fatalf("got %v", got)
	}
	for _, want := range []string{`"yes": the answer agrees`, `"no": the answer declines`, `"unclear": only when`} {
		if !strings.Contains(llm.prompt, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, llm.prompt)
		}
	}
}

func TestPersonReferenceJudgeNamesWhatEachAnswerMeans(t *testing.T) {
	llm := &recordingLLM{out: `{"refersTo": "new", "name": ""}`}
	JudgePersonReference(context.Background(), llm, "girlfriend", "my girlfriend is moving", "", nil)
	for _, want := range []string{"girlfriend, boss, sister, neighbour", "a group (family, team, everyone", "a pronoun with nobody to point to"} {
		if !strings.Contains(llm.prompt, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, llm.prompt)
		}
	}
}
