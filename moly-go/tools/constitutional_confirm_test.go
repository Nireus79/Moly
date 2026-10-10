package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

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
