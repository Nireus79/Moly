package agents

import (
	"context"
	"testing"

	"moly/config"
	"moly/models"
	"moly/tools"
)

type fixedLLM struct {
	content string
	err     error
}

func (l fixedLLM) Call(context.Context, *tools.LLMRequest) (*tools.LLMResponse, error) {
	return &tools.LLMResponse{Content: l.content}, l.err
}

var testPrinciples = []models.Principle{{ID: "user_autonomy", Description: "the user decides"}, {ID: "stakeholder_consideration", Description: "others are affected"}}

// deepeningFixture is a conversation where every deepening gate passes except the one a test changes.
func deepeningFixture(maturity float64) (*conversationAgent, *models.Context) {
	ca := &conversationAgent{
		socraticSelector: NewSocraticQuestionSelector(&models.QuestionLibrary{}, &models.Constitution{}),
		llmClient:        fixedLLM{content: `{"worth": true, "principles": ["stakeholder_consideration"]}`},
		constitution:     &models.Constitution{SupremePrinciples: testPrinciples},
	}
	ctx := &models.Context{
		ContextMaturity: maturity,
		ContextQuality:  config.ContextQualityComprehensive,
		ExtractedContext: &models.ExtractedContext{
			Intention: "write a first message",
			Goals:     []string{"write a first message"},
			Contact:   &models.ExtractedContact{Name: "Christine"},
		},
		ConversationHistory: []models.Message{{}, {}, {}, {}},
		Gaps:                []string{"a minor gap that is not asked"},
	}
	return ca, ctx
}

const longMessage = "She mentioned last week that she likes quiet evenings and old films, and I would like the message to feel honest and not pushy at all."

func TestDeepeningReadsTheGapMaturity(t *testing.T) {
	ca, ctx := deepeningFixture(0.5)
	if ok, _ := ca.deepeningAllowed(ctx, longMessage, IntentAnalysis{Confidence: 0.9}); !ok {
		t.Fatal("with an answered question and every other gate passing, deepening is allowed")
	}
	ca, ctx = deepeningFixture(0)
	if ok, _ := ca.deepeningAllowed(ctx, longMessage, IntentAnalysis{Confidence: 0.9}); ok {
		t.Fatal("gap maturity 0: nothing answered yet, deepening is not allowed")
	}
	ca, ctx = deepeningFixture(0.9)
	if ok, _ := ca.deepeningAllowed(ctx, longMessage, IntentAnalysis{Confidence: 0.9}); ok {
		t.Fatal("maturity 0.9: ready for help, deepening is not allowed")
	}
}

func TestDeepeningTakesThePrinciplesFromTheJudgement(t *testing.T) {
	ca, ctx := deepeningFixture(0.5)
	ok, principles := ca.deepeningAllowed(ctx, longMessage, IntentAnalysis{Confidence: 0.9})
	if !ok || len(principles) != 1 || principles[0] != "stakeholder_consideration" {
		t.Fatalf("the judged principle is used, got %v %v", ok, principles)
	}
}

func TestDeepeningNotWorthMeansNoQuestion(t *testing.T) {
	ca, ctx := deepeningFixture(0.5)
	ca.llmClient = fixedLLM{content: `{"worth": false, "principles": []}`}
	if ok, _ := ca.deepeningAllowed(ctx, longMessage, IntentAnalysis{Confidence: 0.9}); ok {
		t.Fatal("the model sees nothing to reflect on: help, no question")
	}
}

func TestJudgeDeepeningReadsTheModelAndFailsSafe(t *testing.T) {
	judge := func(l fixedLLM) DeepeningJudgment {
		return JudgeDeepening(context.Background(), l, "message", "goal", "Anna", nil, testPrinciples)
	}
	j := judge(fixedLLM{content: `{"worth": "true", "principles": ["user_autonomy", "made_up"]}`})
	if !j.Worth || len(j.Principles) != 1 || j.Principles[0] != "user_autonomy" {
		t.Fatalf("a string boolean is read and an unknown id dropped, got %+v", j)
	}
	if j := judge(fixedLLM{content: "not json"}); j.Worth {
		t.Fatal("unreadable: not worth")
	}
	if j := judge(fixedLLM{err: context.DeadlineExceeded}); j.Worth {
		t.Fatal("failed call: not worth")
	}
}
