package agents

import (
	"context"
	"os"
	"strings"
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

// TestLiveExtractorYes extracts from a bare "Yes", alone and with the exchange (MOLY_LIVE=1). It only reports.
func TestLiveExtractorYes(t *testing.T) {
	if os.Getenv("MOLY_LIVE") == "" {
		t.Skip()
	}
	os.Setenv("LLM_PROVIDER", "ollama")
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Fatal(err)
	}
	ce := NewContextExtractor(c)
	exchange := "User: She is my manager. I want to ask her to share the extra tasks more fairly.\nMoly: Would you like to focus on discussing a fairer distribution of tasks with your manager?\n"
	for _, withExchange := range []bool{false, true} {
		ex := ""
		if withExchange {
			ex = exchange
		}
		for i := 0; i < 3; i++ {
			got, err := ce.ExtractWithExchange(context.Background(), "Yes", ex)
			if err != nil || got == nil {
				t.Logf("exchange=%v: error %v", withExchange, err)
				continue
			}
			contact := "none"
			if got.Contact != nil {
				contact = got.Contact.Name + "/" + got.Contact.Label
			}
			t.Logf("exchange=%-5v intention=%q contact=%s", withExchange, got.Intention, contact)
		}
	}
}

// TestLiveReplyWithDialogue asks the reply model to react to "too formal" with and without the dialogue (MOLY_LIVE=1).
func TestLiveReplyWithDialogue(t *testing.T) {
	if os.Getenv("MOLY_LIVE") == "" {
		t.Skip()
	}
	os.Setenv("LLM_PROVIDER", "ollama")
	c, err := tools.NewLLMClient()
	if err != nil {
		t.Fatal(err)
	}
	ca := &conversationAgent{}
	system := ca.buildAdaptiveSystemPrompt("warm", false, false, ResponseType(""), false)
	packet := &models.DialoguePacket{
		Goal:     "Ask my manager Dana to share the extra tasks more fairly",
		People:   []string{"Dana (the user's manager)"},
		Summary:  "The user is overloaded because her manager Dana keeps assigning extra tasks; she wants Dana to share them more fairly, warmly but directly.",
		MolyLast: "Dear Dana, I would like to request a formal review of the distribution of additional tasks within our team, as my current workload has become unsustainable. Kind regards.",
	}
	message := "They just said: \"It is good but a bit too formal.\"\n\nRespond directly to what they said. Address their specific concern, not just be generally friendly."
	for _, with := range []bool{false, true} {
		user := message
		if with {
			user = packet.Text() + message + "\nRead what they said as the next turn of the dialogue above: a comment such as \"too formal\" or a short reply such as \"yes\" is about your last message, not a new subject."
		}
		resp, err := c.Call(context.Background(), &tools.LLMRequest{SystemPrompt: system, UserPrompt: user, Temperature: 0.7, MaxTokens: 400})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("dialogue=%-5v -> %q", with, strings.ReplaceAll(resp.Content, "\n", " "))
	}
}
