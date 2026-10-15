package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"moly/config"
	"moly/models"
)

type confirmLLM struct {
	out string
	err error
}

func (c confirmLLM) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	return &LLMResponse{Content: c.out}, c.err
}

func evaluatorWith(llm LLMProvider) *ConstitutionalEvaluator {
	return NewConstitutionalEvaluator(llm, &models.Constitution{SupremePrinciples: []models.Principle{
		{ID: "harm_prevention", Name: "Harm Prevention", Severity: "critical"},
	}})
}

const harmJSON = `{"violations": [{"principle_id": "harm_prevention", "evidence": "pressure her", "reasoning": "asks to act against her wishes", "confidence": 0.9, "is_direct_harm": false, "harm_kind": "none"}]}`

func TestConfirmationDecidesNotWords(t *testing.T) {
	msg := "I don't want to pressure her"
	// The evidence contains a "harm word" and used to be accepted by word match. The model says it is not a violation.
	v, err := evaluatorWith(confirmLLM{out: `{"confirmed": false, "reason": "negated"}`}).validateAndParse(harmJSON, msg)
	if err != nil || len(v.MatchedPrinciples) != 0 {
		t.Fatalf("an unconfirmed violation must be dropped: %+v %v", v, err)
	}
	// Single-word and platform evidence used to be rejected by word lists. The model confirms: it stands.
	one := `{"violations": [{"principle_id": "harm_prevention", "evidence": "fetlife", "reasoning": "x", "confidence": 0.9, "is_direct_harm": false}]}`
	v, _ = evaluatorWith(confirmLLM{out: `{"confirmed": true}`}).validateAndParse(one, "I saw her on fetlife")
	if len(v.MatchedPrinciples) != 1 {
		t.Fatalf("a confirmed violation must stand whatever its words: %+v", v)
	}
}

func TestConfirmationFailureKeepsTheFirstJudgement(t *testing.T) {
	for _, llm := range []confirmLLM{{err: errors.New("down")}, {out: "not json"}, {out: `{}`}} {
		v, _ := evaluatorWith(llm).validateAndParse(harmJSON, "I want to pressure her")
		if len(v.MatchedPrinciples) != 1 {
			t.Fatalf("an unanswered confirmation must keep the violation: %+v", v)
		}
	}
}

func TestSafetyAlertNamesWhoToAsk(t *testing.T) {
	mk := func(kind string) *ConstitutionalVerdict {
		return &ConstitutionalVerdict{Allowed: false, OverallSeverity: "critical",
			MatchedPrinciples: []PrincipleMatch{{PrincipleID: "harm_prevention", HarmKind: kind}}}
	}
	self := mk("self_harm").ToSafetyAlert()
	if self.AlertType != "crisis" || !strings.Contains(self.Message, "specialist") {
		t.Fatalf("self-harm: %+v", self)
	}
	law := mk("illegal").ToSafetyAlert()
	if law.AlertType != "illegal" || !strings.Contains(law.Message, "law expert") {
		t.Fatalf("illegal: %+v", law)
	}
	for _, a := range []*models.SafetyAlert{self, law} {
		if strings.Contains(strings.ToLower(a.Message), "helpline") || strings.Contains(a.Message, "?") {
			t.Fatalf("no help lines and no questions: %q", a.Message)
		}
	}
}

type promptLLM struct{ seen *string }

func (p promptLLM) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	*p.seen = req.UserPrompt
	return &LLMResponse{Content: `{"confirmed": false}`}, nil
}

// Found live: the confirmation got only the text "Yes" and the first judgement's reasoning, and agreed. It now sees the same
// exchange as the first judgement, and not the first judgement's reasoning.
func TestConfirmationSeesTheExchangeNotTheFirstReasoning(t *testing.T) {
	var seen string
	ev := evaluatorWith(promptLLM{&seen})
	exchange := "User: I want to ask my manager to share the tasks.\nMoly: Would you like to focus on a fairer distribution?\nUser: Yes\n"
	if _, err := ev.validateAndParseWithExchange(
		`{"violations": [{"principle_id": "harm_prevention", "evidence": "Yes", "reasoning": "FIRST-REASONING-MARKER", "confidence": 0.9, "is_direct_harm": false, "harm_kind": "none"}]}`,
		"Yes", exchange); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(seen, "Would you like to focus on a fairer distribution?") {
		t.Fatalf("the confirmation sees the exchange, got %q", seen)
	}
	if strings.Contains(seen, "FIRST-REASONING-MARKER") {
		t.Fatalf("the confirmation does not see the first reasoning, got %q", seen)
	}
}

func TestStakeholderPrincipleDoesNotListOmissionsAsViolations(t *testing.T) {
	cons, err := config.LoadConstitution("../config/constitution.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range cons.SupremePrinciples {
		if p.ID != "stakeholder_consideration" {
			continue
		}
		for _, v := range p.Violations {
			if strings.Contains(v, "Only considering") || strings.HasPrefix(v, "Ignoring") || strings.Contains(v, "without considering") {
				t.Fatalf("an omission is not a violation of what a user says: %q", v)
			}
		}
	}
}

type sequenceLLM struct {
	answers []string
	calls   *int
}

func (s sequenceLLM) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	i := *s.calls
	*s.calls++
	if i >= len(s.answers) {
		i = len(s.answers) - 1
	}
	return &LLMResponse{Content: s.answers[i]}, nil
}

// A single sampled "no" must not remove a flagged violation (a dynamic run on Claude lost an explicit self-harm statement that
// way); two "no" in a row do.
func TestOneNoDoesNotDropAViolationButTwoDo(t *testing.T) {
	no, yes := `{"confirmed": false}`, `{"confirmed": true}`
	for _, c := range []struct {
		answers []string
		kept    bool
		calls   int
	}{
		{[]string{yes}, true, 1},
		{[]string{no, yes}, true, 2},
		{[]string{no, no}, false, 2},
	} {
		calls := 0
		v, err := evaluatorWith(sequenceLLM{c.answers, &calls}).validateAndParse(harmJSON, "I want to pressure her")
		if err != nil {
			t.Fatal(err)
		}
		if kept := len(v.MatchedPrinciples) == 1; kept != c.kept || calls != c.calls {
			t.Fatalf("answers %v: kept=%v (want %v) after %d calls (want %d)", c.answers, kept, c.kept, calls, c.calls)
		}
	}
}

// A finding the model itself is not sure of (below the floor) is dropped unless it says the harm is direct (found live, 2026-10-15:
// a finding at confidence 0.50 counted as "high" because severity comes from the principle, and started a chain of questions).
func TestAFindingBelowTheConfidenceFloorIsDropped(t *testing.T) {
	unsure := `{"violations": [{"principle_id": "harm_prevention", "evidence": "pressure her", "reasoning": "x", "confidence": 0.5, "is_direct_harm": false, "harm_kind": "none"}]}`
	v, _ := evaluatorWith(confirmLLM{out: `{"confirmed": true}`}).validateAndParse(unsure, "I want to pressure her")
	if len(v.MatchedPrinciples) != 0 || v.OverallSeverity != "clear" && v.OverallSeverity != "low" {
		t.Fatalf("an unsure finding must not count, got %d findings, severity %s", len(v.MatchedPrinciples), v.OverallSeverity)
	}
	sure := `{"violations": [{"principle_id": "harm_prevention", "evidence": "pressure her", "reasoning": "x", "confidence": 0.7, "is_direct_harm": false, "harm_kind": "none"}]}`
	if v, _ := evaluatorWith(confirmLLM{out: `{"confirmed": true}`}).validateAndParse(sure, "I want to pressure her"); len(v.MatchedPrinciples) != 1 {
		t.Fatalf("a finding at 0.7 counts, got %d", len(v.MatchedPrinciples))
	}
	direct := `{"violations": [{"principle_id": "harm_prevention", "evidence": "pressure her", "reasoning": "x", "confidence": 0.5, "is_direct_harm": true, "harm_kind": "illegal"}]}`
	if v, _ := evaluatorWith(confirmLLM{out: `{"confirmed": true}`}).validateAndParse(direct, "I want to pressure her"); len(v.MatchedPrinciples) != 1 {
		t.Fatalf("a finding the model calls direct harm is never dropped by the floor, got %d", len(v.MatchedPrinciples))
	}
}
