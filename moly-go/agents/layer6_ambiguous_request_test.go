package agents

import (
	"context"
	"moly/models"
	"moly/tools"
	"testing"
)

func TestLayer6New(t *testing.T) {
	l6 := NewLayer6AmbiguousRequestHandler(nil)
	if l6 == nil {
		t.Fatal("Failed to create Layer6")
	}
	if l6.Name() != "Layer6-AmbiguousRequest" {
		t.Error("Name mismatch")
	}
}

func TestLayer6Name(t *testing.T) {
	l6 := NewLayer6AmbiguousRequestHandler(nil)
	if l6.Name() != "Layer6-AmbiguousRequest" {
		t.Error("Name mismatch")
	}
}

func TestLayer6Priority(t *testing.T) {
	l6 := NewLayer6AmbiguousRequestHandler(nil)
	if l6.Priority() != 65 {
		t.Errorf("Expected priority 65, got %d", l6.Priority())
	}
}

func TestLayer6SkipIfMatureContext(t *testing.T) {
	l6 := NewLayer6AmbiguousRequestHandler(nil)

	lc := &tools.LayerContext{
		Layer3: &tools.Layer3Result{
			MaturityScore: 0.8, // Mature
		},
	}

	if !l6.CanSkip(lc) {
		t.Error("Should skip Layer6 with mature context")
	}
}

func TestLayer6SkipIfNoGaps(t *testing.T) {
	l6 := NewLayer6AmbiguousRequestHandler(nil)

	lc := &tools.LayerContext{
		Layer4: &tools.Layer4Result{
			GapCount: 0,
		},
	}

	if !l6.CanSkip(lc) {
		t.Error("Should skip Layer6 with no gaps")
	}
}

func TestLayer6ProcessClear(t *testing.T) {
	l6 := NewLayer6AmbiguousRequestHandler(nil)

	lc := &tools.LayerContext{
		Layer4: &tools.Layer4Result{
			GapCount: 0,
		},
		Analysis: &models.AnalysisContext{
			ExtractedConfidence: 0.95,
		},
	}

	result, err := l6.Process(context.Background(), lc)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if result.Layer6.IsAmbiguous {
		t.Error("Should not be ambiguous")
	}

	if !result.Layer6.ShouldProceedToResponse {
		t.Error("Should proceed to response")
	}
}

func TestLayer6ProcessAmbiguous(t *testing.T) {
	l6 := NewLayer6AmbiguousRequestHandler(nil)

	lc := &tools.LayerContext{
		Layer4: &tools.Layer4Result{
			GapCount: 2,
			DetectedGaps: []tools.Gap{
				{Type: "missing_values", Description: "Missing values"},
			},
		},
		Analysis: &models.AnalysisContext{
			ExtractedConfidence: 0.5,
		},
	}

	result, err := l6.Process(context.Background(), lc)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if !result.Layer6.IsAmbiguous {
		t.Error("Should be ambiguous")
	}

	if len(result.Layer6.ClarificationQuestions) == 0 {
		t.Error("Should generate clarification questions")
	}
}

func TestLayer6LowConfidenceIsAmbiguous(t *testing.T) {
	l6 := NewLayer6AmbiguousRequestHandler(nil)

	lc := &tools.LayerContext{
		Layer4: &tools.Layer4Result{
			GapCount: 0,
		},
		Analysis: &models.AnalysisContext{
			ExtractedConfidence: 0.4, // Low confidence
		},
	}

	if !l6.clarifier.IsAmbiguous(lc) {
		t.Error("Should be ambiguous with low extraction confidence")
	}
}

func TestLayer6GenerateQuestions(t *testing.T) {
	l6 := NewLayer6AmbiguousRequestHandler(nil)

	lc := &tools.LayerContext{
		Layer4: &tools.Layer4Result{
			DetectedGaps: []tools.Gap{
				{Type: "missing_communication_style"},
				{Type: "missing_values"},
			},
		},
	}

	questions := l6.GenerateClarificationQuestions(lc)

	if len(questions) == 0 {
		t.Error("Should generate questions for gaps")
	}

	if len(questions) > 3 {
		t.Errorf("Should limit to 3 questions, got %d", len(questions))
	}
}

func TestLayer6AmbiguousElements(t *testing.T) {
	lc := &tools.LayerContext{
		Analysis: &models.AnalysisContext{
			ExtractedEntities: []models.ExtractedEntity{
				{Value: "Alice", IsAmbiguous: false},
				{Value: "the girl", IsAmbiguous: true},
			},
		},
	}

	elements := detectAmbiguousElements(lc)

	if len(elements) != 1 {
		t.Errorf("Expected 1 ambiguous element, got %d", len(elements))
	}

	if elements[0] != "the girl" {
		t.Errorf("Expected 'the girl', got '%s'", elements[0])
	}
}
