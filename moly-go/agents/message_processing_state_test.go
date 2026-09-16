package agents

import (
	"os"
	"testing"

	"moly/database"
)

// getTestDB creates a test SQLite database in memory
func getTestDB(t *testing.T) *database.Database {
	// Create a temp file for SQLite
	tmpFile, err := os.CreateTemp("", "test-moly-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp database file: %v", err)
	}
	tmpFile.Close()
	tmpPath := tmpFile.Name()

	// Clean up after test
	t.Cleanup(func() {
		os.Remove(tmpPath)
	})

	// Initialize database
	db, err := database.Init(tmpPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	return db
}

func TestMessageProcessingStateDeduplication(t *testing.T) {
	db := getTestDB(t)
	manager := NewMessageProcessingStateManager(db)

	userID := "test_user_123"
	conversationID := "conv_456"
	messageID := "msg_789"

	// Test 1: Create initial state (all stages incomplete)
	t.Logf("Test 1: Creating initial message processing state...")
	state, err := manager.GetOrCreateState(userID, conversationID, messageID)
	if err != nil {
		t.Fatalf("Failed to create state: %v", err)
	}
	if state == nil {
		t.Fatal("State is nil")
	}
	if state.Version != 1 {
		t.Errorf("Expected version 1, got %d", state.Version)
	}

	// Verify all stages are incomplete
	for _, stage := range []string{
		StageContextExtraction,
		StageRiskAssessment,
		StageSafetyCheck,
		StageResponseGeneration,
	} {
		if manager.IsStageComplete(state, stage) {
			t.Errorf("Stage %s should be incomplete initially", stage)
		}
	}
	t.Logf("✓ Initial state created with all stages incomplete")

	// Test 2: Mark context extraction as complete
	t.Logf("Test 2: Marking context extraction as complete...")
	mockResult := map[string]interface{}{
		"name":         "Alice",
		"relationship": "friend",
		"confidence":   0.95,
	}

	err = manager.MarkStageComplete(state, StageContextExtraction, mockResult)
	if err != nil {
		t.Fatalf("Failed to mark stage complete: %v", err)
	}

	if !manager.IsStageComplete(state, StageContextExtraction) {
		t.Errorf("Stage %s should be marked complete", StageContextExtraction)
	}
	t.Logf("✓ Context extraction marked complete")

	// Test 3: Reload state from database (simulating retry)
	t.Logf("Test 3: Reloading state from database (simulating retry)...")
	reloadedState, err := manager.GetOrCreateState(userID, conversationID, messageID)
	if err != nil {
		t.Fatalf("Failed to reload state: %v", err)
	}

	// Verify context extraction still marked complete
	if !manager.IsStageComplete(reloadedState, StageContextExtraction) {
		t.Error("Context extraction should still be marked complete after reload")
	}
	t.Logf("✓ State reloaded, context extraction still complete (DEDUP SUCCESS)")

	// Test 4: Verify other stages are still incomplete
	if manager.IsStageComplete(reloadedState, StageRiskAssessment) {
		t.Error("Risk assessment should still be incomplete")
	}
	t.Logf("✓ Other stages correctly still incomplete")

	// Test 5: Get cached result
	t.Logf("Test 5: Retrieving cached result...")
	cachedResult := manager.GetStageResult(reloadedState, StageContextExtraction)
	if cachedResult == nil {
		t.Fatal("Cached result is nil")
	}

	// Verify we can access cached data
	resultMap, ok := cachedResult.(map[string]interface{})
	if !ok {
		t.Fatalf("Cached result is wrong type: %T", cachedResult)
	}

	if name, ok := resultMap["name"].(string); ok && name == "Alice" {
		t.Logf("✓ Cached result retrieved correctly: name=%s", name)
	} else {
		t.Error("Failed to retrieve cached name from result")
	}

	// Test 6: Get remaining stages
	t.Logf("Test 6: Checking remaining stages...")
	remaining := manager.GetRemainingStages(reloadedState)
	if len(remaining) == 0 {
		t.Fatal("Should have remaining stages")
	}

	hasContextExtraction := false
	for _, stage := range remaining {
		if stage == StageContextExtraction {
			hasContextExtraction = true
			break
		}
	}

	if hasContextExtraction {
		t.Error("Context extraction should not be in remaining stages")
	}
	t.Logf("✓ Remaining stages correct: %d stages pending", len(remaining))

	// Test 7: Mark risk assessment complete
	t.Logf("Test 7: Marking risk assessment complete...")
	mockRiskResult := map[string]interface{}{
		"level":    "clear",
		"severity": 0,
	}

	err = manager.MarkStageComplete(reloadedState, StageRiskAssessment, mockRiskResult)
	if err != nil {
		t.Fatalf("Failed to mark risk assessment complete: %v", err)
	}

	if !manager.IsStageComplete(reloadedState, StageRiskAssessment) {
		t.Error("Risk assessment should be marked complete")
	}
	t.Logf("✓ Risk assessment marked complete")

	// Test 8: Cleanup
	t.Logf("Test 8: Deleting state (cleanup)...")
	err = manager.DeleteState(userID, conversationID, messageID)
	if err != nil {
		t.Fatalf("Failed to delete state: %v", err)
	}

	// New state should have version 1 (fresh)
	freshState, _ := manager.GetOrCreateState(userID, conversationID, messageID)
	if freshState.Version != 1 {
		t.Errorf("Fresh state after cleanup should have version 1, got %d", freshState.Version)
	}
	t.Logf("✓ State deleted and fresh state created")

	t.Logf("\n✅ ALL DEDUPLICATION TESTS PASSED")
}
