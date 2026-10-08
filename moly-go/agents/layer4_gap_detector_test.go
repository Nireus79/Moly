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

	if len(critical) != 2 {
		t.Errorf("Expected 2 critical gaps, got %d", len(critical))
	}
}

func TestLayer4VagueNamePatterns(t *testing.T) {
	vagueNames := []string{"the girl", "the guy", "my ex", "someone"}
	concreteNames := []string{"Alice", "Bob", "Sarah"}

	for _, name := range vagueNames {
		if !isVagueContactName(name) {
			t.Errorf("Should detect '%s' as vague", name)
		}
	}

	for _, name := range concreteNames {
		if isVagueContactName(name) {
			t.Errorf("Should not detect '%s' as vague", name)
		}
	}
}
