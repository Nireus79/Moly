package agents

import (
	"context"
	"moly/models"
	"moly/tools"
	"testing"
)

func TestLayer4GapDetectorNew(t *testing.T) {
	mockLLM := &tools.MockLLMClient{}
	l4 := NewLayer4GapDetector(mockLLM)
	if l4 == nil {
		t.Fatal("Failed to create Layer4")
	}
	if l4.Name() != "Layer4-GapDetection" {
		t.Errorf("Expected name 'Layer4-GapDetection', got '%s'", l4.Name())
	}
}

func TestLayer4Name(t *testing.T) {
	mockLLM := &tools.MockLLMClient{}
	l4 := NewLayer4GapDetector(mockLLM)
	if l4.Name() != "Layer4-GapDetection" {
		t.Error("Name mismatch")
	}
}

func TestLayer4Priority(t *testing.T) {
	mockLLM := &tools.MockLLMClient{}
	l4 := NewLayer4GapDetector(mockLLM)
	if l4.Priority() != 70 {
		t.Errorf("Expected priority 70, got %d", l4.Priority())
	}
}

func TestLayer4CanSkip(t *testing.T) {
	mockLLM := &tools.MockLLMClient{}
	l4 := NewLayer4GapDetector(mockLLM)
	lc := &tools.LayerContext{}
	if l4.CanSkip(lc) {
		t.Error("Layer4 should never skip")
	}
}

func TestLayer4NoGenericProfileGaps(t *testing.T) {
	// FIX #75: generic profile gaps were removed; gaps come from the LLM, not fixed rules.
	l4 := NewLayer4GapDetector(&tools.MockLLMClient{})
	gaps := l4.gapAnalyzer.DetectGaps(nil, nil, &models.AnalysisContext{}, 0.9, 0.9, "", nil, nil)
	if len(gaps) != 0 {
		t.Fatalf("generic profile gaps reappeared: %d gaps", len(gaps))
	}
}

func TestLayer4ProcessWithCompleteProfile(t *testing.T) {
	mockLLM := &tools.MockLLMClient{}
	l4 := NewLayer4GapDetector(mockLLM)

	profile := &models.AboutMe{
		UserID:             "user1",
		CommunicationStyle: "direct",
		Values:             []string{"authenticity", "growth"},
	}

	contact := models.Contact{
		Name:         "Alice",
		Relationship: "friend",
	}

	lc := &tools.LayerContext{
		Analysis: &models.AnalysisContext{
			CurrentMessage:   "test",
			UserProfile:      profile,
			RelevantContacts: []models.Contact{contact},
		},
	}

	result, err := l4.Process(context.Background(), lc)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if result.Layer4.GapCount > 2 {
		t.Errorf("Should detect few gaps with complete profile, got %d", result.Layer4.GapCount)
	}
}

func TestLayer4NoVagueNameRule(t *testing.T) {
	// FIX #75: contact names are context, not fixed gap triggers.
	l4 := NewLayer4GapDetector(&tools.MockLLMClient{})
	gaps := l4.gapAnalyzer.DetectGaps(nil, []models.Contact{{Name: "her"}}, &models.AnalysisContext{}, 0.9, 0.9, "", nil, nil)
	for _, g := range gaps {
		if g.Type == "vague_contact_name" {
			t.Fatal("vague-name rule reappeared")
		}
	}
}

func TestLayer4NoConfidenceRule(t *testing.T) {
	// FIX #75: low extraction confidence does not trigger a fixed gap.
	l4 := NewLayer4GapDetector(&tools.MockLLMClient{})
	gaps := l4.gapAnalyzer.DetectGaps(nil, nil, &models.AnalysisContext{}, 0.1, 0.9, "", nil, nil)
	if len(gaps) != 0 {
		t.Fatalf("confidence rule reappeared: %d gaps", len(gaps))
	}
}

func TestLayer4NoMaturityRule(t *testing.T) {
	// FIX #75: low maturity does not trigger a fixed gap.
	l4 := NewLayer4GapDetector(&tools.MockLLMClient{})
	gaps := l4.gapAnalyzer.DetectGaps(nil, nil, &models.AnalysisContext{}, 0.9, 0.1, "", nil, nil)
	if len(gaps) != 0 {
		t.Fatalf("maturity rule reappeared: %d gaps", len(gaps))
	}
}

func TestLayer4CriticalGapFiltering(t *testing.T) {
	gaps := []tools.Gap{
		{Type: "high_severity", Severity: "high", Confidence: 0.9},
		{Type: "medium_high_conf", Severity: "medium", Confidence: 0.9},
		{Type: "medium_low_conf", Severity: "medium", Confidence: 0.5},
		{Type: "low_severity", Severity: "low", Confidence: 0.9},
	}

	critical := filterCriticalGaps(gaps)

	// only what the model rated "high" is critical; a confidence number does not make a "medium" gap critical
	if len(critical) != 1 || critical[0].Type != "high_severity" {
		t.Errorf("Expected only the high gap to be critical, got %+v", critical)
	}
}

// Of two gaps of one type the more severe is kept, in the place of the first (the first used to be kept, whatever it was rated).
func TestDeduplicationKeepsTheMoreSevereGapOfAType(t *testing.T) {
	l4 := NewLayer4GapDetector(&tools.MockLLMClient{})
	got := l4.deduplicateGaps([]tools.Gap{
		{Type: "t", Description: "lesser", Severity: "low"},
		{Type: "t", Description: "blocking", Severity: "high"},
		{Type: "u", Description: "other", Severity: "medium"},
	})
	if len(got) != 2 || got[0].Description != "blocking" || got[1].Description != "other" {
		t.Fatalf("got %+v", got)
	}
}
