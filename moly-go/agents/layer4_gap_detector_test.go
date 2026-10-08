package agents

import (
	"context"
	"moly/models"
	"moly/tools"
	"testing"
)

func TestLayer4GapDetectorNew(t *testing.T) {
	l4 := NewLayer4GapDetector()
	if l4 == nil {
		t.Fatal("Failed to create Layer4")
	}
	if l4.Name() != "Layer4-GapDetection" {
		t.Errorf("Expected name 'Layer4-GapDetection', got '%s'", l4.Name())
	}
}

func TestLayer4Name(t *testing.T) {
	l4 := NewLayer4GapDetector()
	if l4.Name() != "Layer4-GapDetection" {
		t.Error("Name mismatch")
	}
}

func TestLayer4Priority(t *testing.T) {
	l4 := NewLayer4GapDetector()
	if l4.Priority() != 70 {
		t.Errorf("Expected priority 70, got %d", l4.Priority())
	}
}

func TestLayer4CanSkip(t *testing.T) {
	l4 := NewLayer4GapDetector()
	lc := &tools.LayerContext{}
	if l4.CanSkip(lc) {
		t.Error("Layer4 should never skip")
	}
}

func TestLayer4ProcessWithNilProfile(t *testing.T) {
	l4 := NewLayer4GapDetector()
	lc := &tools.LayerContext{
		Analysis: &models.AnalysisContext{
			CurrentMessage: "test",
		},
	}

	result, err := l4.Process(context.Background(), lc)
	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if result.Layer4 == nil {
		t.Error("Layer4 result should not be nil")
	}

	if result.Layer4.GapCount == 0 {
		t.Error("Should detect gaps with nil profile")
	}
}

func TestLayer4ProcessWithCompleteProfile(t *testing.T) {
	l4 := NewLayer4GapDetector()

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

func TestLayer4DetectsVagueNames(t *testing.T) {
	l4 := NewLayer4GapDetector()

	contact := models.Contact{
		Name:         "the girl",
		Relationship: "friend",
	}

	gaps := l4.gapAnalyzer.DetectGaps(
		nil,
		[]models.Contact{contact},
		&models.AnalysisContext{},
		0.5,
		0.5,
	)

	hasVagueNameGap := false
	for _, gap := range gaps {
		if gap.Type == "vague_contact_name" {
			hasVagueNameGap = true
			break
		}
	}

	if !hasVagueNameGap {
		t.Error("Should detect vague contact name")
	}
}

func TestLayer4DetectsLowConfidence(t *testing.T) {
	l4 := NewLayer4GapDetector()

	gaps := l4.gapAnalyzer.DetectGaps(
		&models.AboutMe{UserID: "user1"},
		[]models.Contact{{Name: "Alice", Relationship: "friend"}},
		&models.AnalysisContext{},
		0.3, // Low confidence
		0.5,
	)

	hasConfidenceGap := false
	for _, gap := range gaps {
		if gap.Type == "low_extraction_confidence" {
			hasConfidenceGap = true
			break
		}
	}

	if !hasConfidenceGap {
		t.Error("Should detect low extraction confidence")
	}
}

func TestLayer4DetectsImmatureContext(t *testing.T) {
	l4 := NewLayer4GapDetector()

	gaps := l4.gapAnalyzer.DetectGaps(
		&models.AboutMe{UserID: "user1"},
		[]models.Contact{{Name: "Alice", Relationship: "friend"}},
		&models.AnalysisContext{},
		0.8,
		0.2, // Immature
	)

	hasMaturityGap := false
	for _, gap := range gaps {
		if gap.Type == "immature_context" {
			hasMaturityGap = true
			break
		}
	}

	if !hasMaturityGap {
		t.Error("Should detect immature context")
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
