package agents

import (
	"context"
	"moly/tools"
	"testing"
)

func TestLayer5New(t *testing.T) {
	l5 := NewLayer5UnifiedConflictDetection(nil, nil)
	if l5 == nil {
		t.Fatal("Failed to create Layer5")
	}
	if l5.Name() != "Layer5-UnifiedConflictDetection" {
		t.Error("Name mismatch")
	}
}

func TestLayer5Name(t *testing.T) {
	l5 := NewLayer5UnifiedConflictDetection(nil, nil)
	if l5.Name() != "Layer5-UnifiedConflictDetection" {
		t.Error("Name mismatch")
	}
}

func TestLayer5Priority(t *testing.T) {
	l5 := NewLayer5UnifiedConflictDetection(nil, nil)
	if l5.Priority() != 60 {
		t.Errorf("Expected priority 60, got %d", l5.Priority())
	}
}

func TestLayer5SkipWithNoGaps(t *testing.T) {
	l5 := NewLayer5UnifiedConflictDetection(nil, nil)

	lc := &tools.LayerContext{
		Layer3: &tools.Layer3Result{
			MaturityScore: 0.5,
		},
		Layer4: &tools.Layer4Result{
			GapCount: 0,
		},
		Layer6: &tools.Layer6Result{
			IsAmbiguous: false,
		},
	}

	if !l5.CanSkip(lc) {
		t.Error("Should skip with no gaps and not ambiguous")
	}
}

func TestLayer5SkipWithImmatureContext(t *testing.T) {
	l5 := NewLayer5UnifiedConflictDetection(nil, nil)

	lc := &tools.LayerContext{
		Layer3: &tools.Layer3Result{
			MaturityScore: 0.1, // Immature
		},
	}

	if !l5.CanSkip(lc) {
		t.Error("Should skip with immature context")
	}
}

func TestLayer5ProcessWithNoConflicts(t *testing.T) {
	l5 := NewLayer5UnifiedConflictDetection(nil, nil)

	lc := &tools.LayerContext{
		Layer3: &tools.Layer3Result{
			MaturityScore: 0.5,
		},
	}

	result, err := l5.Process(context.Background(), lc)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if result.Layer5 == nil {
		t.Error("Layer5 result should not be nil")
	}

	if result.Layer5.ConflictCount != 0 {
		t.Errorf("Expected 0 conflicts, got %d", result.Layer5.ConflictCount)
	}
}

func TestLayer5CriticalConflictFiltering(t *testing.T) {
	conflicts := []tools.Conflict{
		{Type: "critical", Severity: "critical", Confidence: 0.9},
		{Type: "high", Severity: "high", Confidence: 0.9},
		{Type: "high_low", Severity: "high", Confidence: 0.5},
		{Type: "medium", Severity: "medium", Confidence: 0.9},
	}

	critical := filterCriticalConflicts(conflicts)

	if len(critical) != 2 {
		t.Errorf("Expected 2 critical conflicts, got %d", len(critical))
	}
}
