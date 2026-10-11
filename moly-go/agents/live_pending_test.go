package agents

import (
	"context"
	"os"
	"testing"

	"moly/config"
	"moly/models"
	"moly/tools"
)

type recLLM struct {
	inner tools.LLMProvider
	t     *testing.T
}

func (r recLLM) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	resp, err := r.inner.Call(ctx, req)
	if err == nil {
		r.t.Logf("RAW %q", resp.Content)
	}
	return resp, err
}

// TestLivePending replays the pending-question judge against the real model (MOLY_LIVE=1, Ollama running). It only reports; it never fails.
func TestLivePending(t *testing.T) {
	if os.Getenv("MOLY_LIVE") == "" {
		t.Skip()
	}
	os.Setenv("LLM_PROVIDER", "ollama")
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Fatal(err)
	}
	llm := recLLM{c, t}
	cases := []struct {
		name string
		msg  string
		q    []string
		want []bool
	}{
		{"role answered (description)", "She is my manager. I want to ask her to share the extra tasks more fairly.", []string{"The role of Dana in the user's professional life"}, []bool{true}},
		{"role answered (question)", "She is my manager. I want to ask her to share the extra tasks more fairly.", []string{"What's Dana's professional role in your work?"}, []bool{true}},
		{"not answered", "Make it a bit shorter too.", []string{"The role of Dana in the user's professional life"}, []bool{false}},
		{"two items, one answered", "She is my manager.", []string{"The role of Dana in the user's professional life", "How formal the message should be"}, []bool{true, false}},
	}
	for _, c := range cases {
		ok := 0
		for i := 0; i < 5; i++ {
			got := JudgePendingAnswered(context.Background(), llm, c.msg, c.q)
			same := true
			for j := range got {
				if got[j] != c.want[j] {
					same = false
				}
			}
			if same {
				ok++
			}
		}
		t.Logf("%-30s expected result in %d/5 runs", c.name, ok)
	}
}

// TestLiveEvaluatorYes replays the safety evaluator on a bare "Yes" after Moly's question (MOLY_LIVE=1). It only reports.
func TestLiveEvaluatorYes(t *testing.T) {
	if os.Getenv("MOLY_LIVE") == "" {
		t.Skip()
	}
	os.Setenv("LLM_PROVIDER", "ollama")
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Fatal(err)
	}
	cons, err := config.LoadConstitution("../config/constitution.yaml")
	if err != nil {
		t.Fatal(err)
	}
	ev := tools.NewConstitutionalEvaluator(recLLM{c, t}, cons)
	flagged := 0
	for i := 0; i < 6; i++ {
		a := &models.AnalysisContext{
			CurrentMessage: "Yes",
			RecentMessages: []models.Message{
				{Role: "user", Content: "She is my manager. I want to ask her to share the extra tasks more fairly."},
				{Role: "assistant", Content: "Would you like to focus on discussing a fairer distribution of tasks with your manager?"},
				{Role: "user", Content: "Yes"},
			},
		}
		v, err := ev.EvaluateWithAnalysisContextAndMaturity(context.Background(), a, 0, "critical")
		if err == nil && len(v.MatchedPrinciples) > 0 {
			flagged++
		}
	}
	t.Logf("a bare Yes was flagged in %d of 6 runs", flagged)
}
