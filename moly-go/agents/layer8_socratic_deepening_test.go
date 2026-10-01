package agents

import (
	"context"
	"moly/tools"
	"testing"
)

func TestLayer8New(t *testing.T) {
	l8 := NewLayer8SocraticDeepening()
	if l8 == nil {
		t.Fatal("Failed to create Layer8")
	}
}

func TestLayer8Name(t *testing.T) {
	l8 := NewLayer8SocraticDeepening()
	if l8.Name() != "Layer8-SocraticDeepening" {
		t.Error("Name mismatch")
	}
}

func TestLayer8Priority(t *testing.T) {
	l8 := NewLayer8SocraticDeepening()
	if l8.Priority() != 50 {
		t.Errorf("Expected priority 50, got %d", l8.Priority())
	}
}

func TestLayer8SkipLowMaturity(t *testing.T) {
	l8 := NewLayer8SocraticDeepening()
	lc := &tools.LayerContext{
		Layer3: &tools.Layer3Result{
			MaturityScore: 0.3, // Below threshold
		},
	}

	if !l8.CanSkip(lc) {
		t.Error("Should skip with low maturity")
	}
}

func TestLayer8ProcessMaturesContext(t *testing.T) {
	l8 := NewLayer8SocraticDeepening()
	lc := &tools.LayerContext{
		Layer3: &tools.Layer3Result{
			MaturityScore: 0.8,
		},
		Layer4: &tools.Layer4Result{
			CriticalGaps: []tools.Gap{},
		},
		Layer6: &tools.Layer6Result{
			IsAmbiguous: false,
		},
	}

	result, err := l8.Process(context.Background(), lc)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if result.Layer8 == nil {
		t.Error("Layer8 result should not be nil")
	}

	if len(result.Layer8.SocraticQuestions) == 0 {
		t.Error("Should generate questions for mature context")
	}
}
