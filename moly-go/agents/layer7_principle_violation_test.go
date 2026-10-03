package agents

import (
	"context"
	"moly/tools"
	"testing"
)

func TestLayer7New(t *testing.T) {
	l7 := NewLayer7PrincipleViolationClarification(nil)
	if l7 == nil {
		t.Fatal("Failed to create Layer7")
	}
}

func TestLayer7Name(t *testing.T) {
	l7 := NewLayer7PrincipleViolationClarification(nil)
	if l7.Name() != "Layer7-PrincipleViolation" {
		t.Error("Name mismatch")
	}
}

func TestLayer7Priority(t *testing.T) {
	l7 := NewLayer7PrincipleViolationClarification(nil)
	if l7.Priority() != 75 {
		t.Errorf("Expected priority 75, got %d", l7.Priority())
	}
}

func TestLayer7CanSkip(t *testing.T) {
	l7 := NewLayer7PrincipleViolationClarification(nil)

	lc := &tools.LayerContext{
		Layer2: &tools.Layer2Result{
			IsObviousHarm: false,
		},
	}

	if !l7.CanSkip(lc) {
		t.Error("Should skip if Layer 2 found no harm")
	}
}

func TestLayer7Process(t *testing.T) {
	l7 := NewLayer7PrincipleViolationClarification(nil)

	lc := &tools.LayerContext{
		Layer2: &tools.Layer2Result{
			IsObviousHarm: true,
		},
	}

	result, err := l7.Process(context.Background(), lc)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if result.Layer7 == nil {
		t.Error("Layer7 result should not be nil")
	}

	if !result.Layer7.ViolationDetected {
		t.Error("Should detect violation")
	}
}
