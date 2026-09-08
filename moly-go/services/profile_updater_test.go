package services

import (
	"path/filepath"
	"testing"

	"moly/agents"
	"moly/database"
)

// setupUpdaterTest creates a test environment
func setupUpdaterTest(t *testing.T) *ProfileUpdater {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_profile_updater.db")

	systemKey := "test-profile-updater-key"
	db, err := database.Init(dbPath, systemKey)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	return NewProfileUpdater(db)
}

// TestProfileUpdaterBasicAboutMeUpdate tests updating AboutMe profile
func TestProfileUpdaterBasicAboutMeUpdate(t *testing.T) {
	updater := setupUpdaterTest(t)

	result := &agents.ExtractionResult{
		AboutMeUpdates: []agents.AboutMeUpdate{
			{
				Key:        "communication_style",
				Value:      "direct",
				Confidence: 0.85,
				Source:     "observed_behavior",
				Reasoning:  "User states they want to be direct",
			},
		},
		PatternDetections: []agents.PatternDetection{},
		ContactMentions:   []agents.ContactMention{},
		ConfidenceScore:   0.85,
	}

	stats, err := updater.UpdateProfile("test_user", result)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	if stats.AboutMeUpdated != 1 {
		t.Errorf("Expected 1 AboutMe update, got %d", stats.AboutMeUpdated)
	}

	t.Logf("✓ Basic AboutMe update: %+v", stats)
}

// TestProfileUpdaterLowConfidenceFiltering tests that low confidence items are skipped
func TestProfileUpdaterLowConfidenceFiltering(t *testing.T) {
	updater := setupUpdaterTest(t)
	updater.SetConfidenceThreshold(0.6)

	result := &agents.ExtractionResult{
		AboutMeUpdates: []agents.AboutMeUpdate{
			{Key: "style", Value: "direct", Confidence: 0.85},  // Above threshold
			{Key: "tone", Value: "warm", Confidence: 0.4},      // Below threshold
			{Key: "value", Value: "honesty", Confidence: 0.55}, // Below threshold
		},
		PatternDetections: []agents.PatternDetection{},
		ContactMentions:   []agents.ContactMention{},
	}

	stats, err := updater.UpdateProfile("test_user", result)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	// Only 1 should pass threshold
	if stats.AboutMeUpdated != 1 {
		t.Errorf("Expected 1 AboutMe update after filtering, got %d", stats.AboutMeUpdated)
	}

	t.Log("✓ Low-confidence filtering works")
}

// TestProfileUpdaterConfidenceThreshold tests threshold setting
func TestProfileUpdaterConfidenceThreshold(t *testing.T) {
	updater := setupUpdaterTest(t)

	// Test clamping
	updater.SetConfidenceThreshold(1.5)
	if updater.config.Threshold != 1 {
		t.Error("Threshold should be clamped to 1")
	}

	updater.SetConfidenceThreshold(-0.5)
	if updater.config.Threshold != 0 {
		t.Error("Threshold should be clamped to 0")
	}

	t.Log("✓ Confidence threshold clamping works")
}

// TestProfileUpdaterPatternDetection tests pattern updating
func TestProfileUpdaterPatternDetection(t *testing.T) {
	updater := setupUpdaterTest(t)

	result := &agents.ExtractionResult{
		AboutMeUpdates: []agents.AboutMeUpdate{},
		PatternDetections: []agents.PatternDetection{
			{
				Pattern:      "avoids_conflict_then_over_explains",
				Category:     "avoidance",
				Confidence:   0.8,
				Evidence:     "User mentioned being scared to speak up",
				IsGrowthArea: true,
			},
		},
		ContactMentions: []agents.ContactMention{},
	}

	stats, err := updater.UpdateProfile("test_user", result)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	if stats.PatternsAdded != 1 {
		t.Errorf("Expected 1 pattern added, got %d", stats.PatternsAdded)
	}

	t.Logf("✓ Pattern detection: %+v", stats)
}

// TestProfileUpdaterContactMention tests contact tracking
func TestProfileUpdaterContactMention(t *testing.T) {
	updater := setupUpdaterTest(t)

	result := &agents.ExtractionResult{
		AboutMeUpdates:    []agents.AboutMeUpdate{},
		PatternDetections: []agents.PatternDetection{},
		ContactMentions: []agents.ContactMention{
			{
				Name:             "Sarah",
				RelationshipType: "professional",
				ToneObserved:     "formal",
				Context:          "My boss",
				MainTopics:       []string{"feedback", "recognition"},
				Frequency:        "weekly",
				Confidence:       0.85,
			},
		},
	}

	stats, err := updater.UpdateProfile("test_user", result)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	if stats.ContactsAdded != 1 {
		t.Errorf("Expected 1 contact added, got %d", stats.ContactsAdded)
	}

	t.Logf("✓ Contact mention: %+v", stats)
}

// TestProfileUpdaterGoalProgress tests goal progress tracking
func TestProfileUpdaterGoalProgress(t *testing.T) {
	updater := setupUpdaterTest(t)

	result := &agents.ExtractionResult{
		AboutMeUpdates:    []agents.AboutMeUpdate{},
		PatternDetections: []agents.PatternDetection{},
		ContactMentions:   []agents.ContactMention{},
		GoalProgressUpdates: []agents.GoalProgressUpdate{
			{
				GoalDescription: "say no without apologizing",
				Progress:        "in_progress",
				Evidence:        "User mentioned trying to be more assertive",
				Confidence:      0.7,
			},
		},
	}

	stats, err := updater.UpdateProfile("test_user", result)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	if stats.GoalsUpdated != 1 {
		t.Errorf("Expected 1 goal updated, got %d", stats.GoalsUpdated)
	}

	t.Logf("✓ Goal progress: %+v", stats)
}

// TestProfileUpdaterEmptyResult tests handling of empty results
func TestProfileUpdaterEmptyResult(t *testing.T) {
	updater := setupUpdaterTest(t)

	result := &agents.ExtractionResult{
		AboutMeUpdates:      []agents.AboutMeUpdate{},
		PatternDetections:   []agents.PatternDetection{},
		ContactMentions:     []agents.ContactMention{},
		GoalProgressUpdates: []agents.GoalProgressUpdate{},
		ConfidenceScore:     0,
	}

	stats, err := updater.UpdateProfile("test_user", result)
	if err != nil {
		t.Fatalf("UpdateProfile should not error on empty result: %v", err)
	}

	if stats.TotalUpdates != 0 {
		t.Errorf("Empty result should have 0 updates, got %d", stats.TotalUpdates)
	}

	t.Log("✓ Empty result handling works")
}

// TestProfileUpdaterNilResult tests handling of nil result
func TestProfileUpdaterNilResult(t *testing.T) {
	updater := setupUpdaterTest(t)

	_, err := updater.UpdateProfile("test_user", nil)
	if err == nil {
		t.Error("UpdateProfile should error on nil result")
	}

	t.Log("✓ Nil result error handling works")
}

// TestProfileUpdaterEmptyUserID tests handling of empty userID
func TestProfileUpdaterEmptyUserID(t *testing.T) {
	updater := setupUpdaterTest(t)

	result := &agents.ExtractionResult{}

	_, err := updater.UpdateProfile("", result)
	if err == nil {
		t.Error("UpdateProfile should error on empty userID")
	}

	t.Log("✓ Empty userID error handling works")
}

// TestProfileUpdaterImplicitLearning tests adding implicit learnings
func TestProfileUpdaterImplicitLearning(t *testing.T) {
	updater := setupUpdaterTest(t)

	err := updater.AddImplicitLearning(
		"test_user",
		"prefers_direct_language",
		"true",
		"communication_style",
		0.75,
		"observed_behavior")

	if err != nil {
		t.Fatalf("AddImplicitLearning failed: %v", err)
	}

	t.Log("✓ Implicit learning added")
}

// TestProfileUpdaterComplexExtraction tests a complex realistic extraction
func TestProfileUpdaterComplexExtraction(t *testing.T) {
	updater := setupUpdaterTest(t)

	result := &agents.ExtractionResult{
		AboutMeUpdates: []agents.AboutMeUpdate{
			{Key: "communication_style", Value: "direct", Confidence: 0.85},
			{Key: "value", Value: "honesty", Confidence: 0.8},
			{Key: "tone_preference", Value: "warm", Confidence: 0.75},
		},
		PatternDetections: []agents.PatternDetection{
			{Pattern: "avoids_conflict", Category: "avoidance", Confidence: 0.8, IsGrowthArea: true},
			{Pattern: "over_explains_when_defensive", Category: "clarity", Confidence: 0.75, IsGrowthArea: true},
		},
		ContactMentions: []agents.ContactMention{
			{Name: "boss", RelationshipType: "professional", Frequency: "weekly", ToneObserved: "formal", Confidence: 0.9},
			{Name: "mom", RelationshipType: "family", Frequency: "monthly", ToneObserved: "warm", Confidence: 0.85},
		},
		GoalProgressUpdates: []agents.GoalProgressUpdate{
			{GoalDescription: "be more assertive with authority", Progress: "in_progress", Confidence: 0.8},
		},
		ConfidenceScore: 0.82,
	}

	stats, err := updater.UpdateProfile("complex_user", result)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	if stats.TotalUpdates == 0 {
		t.Error("Complex extraction should produce updates")
	}

	t.Logf("✓ Complex extraction: %+v", stats)
}

// TestProfileUpdaterGate - Complete updater verification
func TestProfileUpdaterGate(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"BasicAboutMeUpdate", TestProfileUpdaterBasicAboutMeUpdate},
		{"LowConfidenceFiltering", TestProfileUpdaterLowConfidenceFiltering},
		{"ConfidenceThreshold", TestProfileUpdaterConfidenceThreshold},
		{"PatternDetection", TestProfileUpdaterPatternDetection},
		{"ContactMention", TestProfileUpdaterContactMention},
		{"GoalProgress", TestProfileUpdaterGoalProgress},
		{"EmptyResult", TestProfileUpdaterEmptyResult},
		{"NilResult", TestProfileUpdaterNilResult},
		{"EmptyUserID", TestProfileUpdaterEmptyUserID},
		{"ImplicitLearning", TestProfileUpdaterImplicitLearning},
		{"ComplexExtraction", TestProfileUpdaterComplexExtraction},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}

	t.Log("\n✓ ProfileUpdater gate PASSED - Ready for Phase 1.2")
}
