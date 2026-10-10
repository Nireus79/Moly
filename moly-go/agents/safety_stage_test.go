package agents

import (
	"context"
	"errors"
	"testing"

	"moly/models"
	"moly/tools"
)

type fakeJudge struct {
	verdict *tools.ConstitutionalVerdict
	err     error
	gate    string
	calls   int
}

func (f *fakeJudge) EvaluateWithAnalysisContextAndMaturity(_ context.Context, _ *models.AnalysisContext, _ float64, gate string) (*tools.ConstitutionalVerdict, error) {
	f.calls++
	f.gate = gate
	return f.verdict, f.err
}

func TestSeverityGateBands(t *testing.T) {
	cases := []struct {
		maturity float64
		want     string
	}{{0, "critical"}, {0.29, "critical"}, {0.3, "high"}, {0.49, "high"}, {0.5, "medium"}, {0.69, "medium"}, {0.7, "low"}, {1, "low"}}
	for _, c := range cases {
		if got := SeverityGate(c.maturity); got != c.want {
			t.Errorf("SeverityGate(%.2f) = %s, want %s", c.maturity, got, c.want)
		}
	}
}

func TestEvaluateSafetyFailsClosed(t *testing.T) {
	a := &models.AnalysisContext{CurrentMessage: "hi"}
	if _, err := EvaluateSafety(context.Background(), &fakeJudge{err: errors.New("down")}, a, 0); !errors.Is(err, ErrSafetyUnavailable) {
		t.Fatalf("an evaluator error must deny, got %v", err)
	}
	if _, err := EvaluateSafety(context.Background(), &fakeJudge{}, a, 0); !errors.Is(err, ErrSafetyUnavailable) {
		t.Fatalf("a missing verdict must deny, got %v", err)
	}
	if _, err := EvaluateSafety(context.Background(), nil, a, 0); !errors.Is(err, ErrSafetyUnavailable) {
		t.Fatalf("no evaluator must deny, got %v", err)
	}
	if _, err := EvaluateSafety(context.Background(), &fakeJudge{verdict: &tools.ConstitutionalVerdict{Allowed: true}}, nil, 0); !errors.Is(err, ErrSafetyUnavailable) {
		t.Fatalf("no context must deny, got %v", err)
	}
}

func TestEvaluateSafetyClearVerdictIsNotBlocked(t *testing.T) {
	j := &fakeJudge{verdict: &tools.ConstitutionalVerdict{Allowed: true, OverallSeverity: "clear"}}
	res, err := EvaluateSafety(context.Background(), j, &models.AnalysisContext{CurrentMessage: "hi"}, 0.6)
	if err != nil || res.Blocked || res.Alert != nil {
		t.Fatalf("a clear verdict must pass: %+v %v", res, err)
	}
	if j.calls != 1 || j.gate != "medium" {
		t.Fatalf("one call, gate from maturity: calls=%d gate=%s", j.calls, j.gate)
	}
}

type countingLLM struct {
	calls int
	err   error
}

func (c *countingLLM) Call(context.Context, *tools.LLMRequest) (*tools.LLMResponse, error) {
	c.calls++
	return &tools.LLMResponse{Content: `{"violations": []}`}, c.err
}

func layer2With(llm *countingLLM) *Layer2PrincipleCheckAdapter {
	return NewLayer2PrincipleCheckAdapter(tools.NewConstitutionalEvaluator(llm, &models.Constitution{}))
}

func layerCtxFor(msg string, verdict interface{}) *tools.LayerContext {
	a := &models.AnalysisContext{CurrentMessage: msg, SafetyVerdict: verdict}
	return tools.NewLayerContext(a, "u", "m", "c", nil)
}

func TestLayer2UsesThePrecomputedVerdictWithoutAskingAgain(t *testing.T) {
	llm := &countingLLM{}
	lc := layerCtxFor("hello", &tools.ConstitutionalVerdict{Allowed: true, OverallSeverity: "clear"})
	if _, err := layer2With(llm).Process(context.Background(), lc); err != nil {
		t.Fatal(err)
	}
	if llm.calls != 0 || lc.Layer2 == nil || lc.Layer2.Verdict == nil {
		t.Fatalf("the verdict must be reused: calls=%d layer2=%+v", llm.calls, lc.Layer2)
	}
}

func TestLayer2FailureWithoutAVerdictStopsInsteadOfPassing(t *testing.T) {
	llm := &countingLLM{err: errors.New("down")}
	lc := layerCtxFor("hello", nil)
	_, err := layer2With(llm).Process(context.Background(), lc)
	if !errors.Is(err, ErrSafetyUnavailable) {
		t.Fatalf("a failing evaluation must return ErrSafetyUnavailable, got %v", err)
	}
}
