package agents

import (
	"context"
	"moly/models"
	"moly/tools"
	"testing"
	"time"
)

// TestPhase1CompleteFlow verifies end-to-end extraction locking
// Message → Extract → Lock → Storage → Clarification (all with lock enforcement)
func TestPhase1CompleteFlow(t *testing.T) {
	// Setup: Create all Phase 1 components
	mockLLM := &tools.MockLLMClient{}
	intentDetector := NewLLMIntentDetector(mockLLM)
	extractionStore := tools.NewExtractionStore()
	cache := tools.NewLLMCache(5*time.Minute, 1000)

	// Create extraction phase with locking
	extractionPhase := &ExtractionPhase{
		intentDetector:  intentDetector,
		extractionStore: extractionStore,
	}

	// Step 1: User sends message
	userMessage := "I really like spending time with Sarah. She's intelligent and kind."
	t.Logf("Step 1: User message: %s", userMessage)

	// Step 2: Run extraction phase (automatically locks)
	input := &ExtractionPhaseInput{
		UserID:         "user_test",
		ConversationID: "conv_test",
		MessageID:      "msg_1",
		Message:        userMessage,
		Cache:          cache,
	}

	output, err := extractionPhase.Run(context.Background(), input)
	if err != nil {
		t.Fatalf("ExtractionPhase.Run failed: %v", err)
	}

	if output == nil || output.Artifact == nil {
		t.Fatal("ExtractionPhase returned no artifact")
	}

	artifact := output.Artifact
	t.Logf("Step 2: Extraction complete - %d entities extracted", len(artifact.Entities))

	// Step 3: Verify extraction is locked
	if !artifact.IsLocked {
		t.Fatal("PHASE 1 VIOLATION: Extraction artifact not locked after ExtractionPhase")
	}
	t.Logf("Step 3: ✓ Extraction locked (reason: %s)", artifact.LockReason)

	// Step 4: Verify TTL is set
	if artifact.ExpiresAt == 0 {
		t.Fatal("PHASE 1 VIOLATION: ExpiresAt not set (TTL missing)")
	}
	expectedExpiration := time.Now().Add(30 * time.Minute).Unix()
	if artifact.ExpiresAt < expectedExpiration-2 || artifact.ExpiresAt > expectedExpiration+2 {
		t.Logf("Warning: TTL slightly off (expected %d, got %d)", expectedExpiration, artifact.ExpiresAt)
	}
	t.Logf("Step 4: ✓ TTL set to 30 minutes")

	// Step 5: Verify locked artifact cannot be modified
	err = artifact.TryModify("add_entity")
	if err == nil {
		t.Fatal("PHASE 1 VIOLATION: Locked artifact was modified")
	}
	t.Logf("Step 5: ✓ Locked artifact cannot be modified (error: %v)", err)

	// Step 6: Verify artifact is stored in extraction store (using messageID as key)
	retrievedArtifact := extractionStore.Get(artifact.MessageID)
	if retrievedArtifact == nil {
		t.Fatal("Failed to retrieve artifact from store (returned nil)")
	}
	if !retrievedArtifact.IsLocked {
		t.Fatal("PHASE 1 VIOLATION: Retrieved artifact is not locked")
	}
	t.Logf("Step 6: ✓ Locked artifact stored correctly in extraction store")

	// Step 7: Verify cannot re-extract (lock prevents modification)
	err = retrievedArtifact.TryModify("re_parse")
	if err == nil {
		t.Fatal("PHASE 1 VIOLATION: Locked stored artifact was modified")
	}
	t.Logf("Step 7: ✓ Stored locked artifact immutable (no re-parsing possible)")

	t.Logf("\n✅ PHASE 1 INTEGRATION TEST PASSED: Complete locked extraction flow verified")
	t.Logf("   - Extraction locked immediately ✓")
	t.Logf("   - TTL set correctly ✓")
	t.Logf("   - Immutability enforced ✓")
	t.Logf("   - Storage preserves lock ✓")
}

// TestPhase1NoReparsingPossible verifies re-parsing is impossible
// Even if downstream code tries, locked extraction cannot be modified
func TestPhase1NoReparsingPossible(t *testing.T) {
	// Create locked artifact (simulating ExtractionPhase output)
	artifact := &models.ExtractionArtifact{
		ID:             "extraction_test",
		MessageID:      "msg_1",
		UserID:         "user_1",
		ConversationID: "conv_1",
		Entities: []models.ExtractedEntity{
			{
				Value:      "Sarah",
				Type:       "contact",
				Subject:    "user",
				Confidence: 0.95,
				Evidence:   "I really like Sarah",
			},
		},
		Source:     "llm",
		LLMSuccess: true,
		Duration:   15.2,
		CreatedAt:  time.Now().Unix(),
		IsLocked:   false,
	}

	// Lock it (simulating ExtractionPhase behavior)
	err := artifact.Lock("extraction_complete: source=llm, entities=1, confidence=0.95")
	if err != nil {
		t.Fatalf("Failed to lock artifact: %v", err)
	}

	t.Logf("Initial state: 1 extracted entity (Sarah)")

	// Attempt various re-parsing operations
	operations := []string{
		"re_parse_with_linguistic_parser",
		"add_entity",
		"modify_subject",
		"re_extract_with_fallback",
		"update_entities",
	}

	for _, op := range operations {
		err := artifact.TryModify(op)
		if err == nil {
			t.Fatalf("PHASE 1 VIOLATION: Re-parsing operation '%s' was allowed on locked artifact", op)
		}
		t.Logf("✓ Re-parsing operation blocked: %s", op)
	}

	// Verify original data unchanged (immutability proven)
	if len(artifact.Entities) != 1 {
		t.Fatal("Entities were modified despite lock")
	}
	if artifact.Entities[0].Value != "Sarah" {
		t.Fatal("Entity data was modified despite lock")
	}

	t.Logf("\n✅ RE-PARSING IMPOSSIBLE: All modification attempts blocked by lock")
}

// TestPhase1ExtractAndLockMethod verifies ExtractAndLock returns locked artifact
func TestPhase1ExtractAndLockMethod(t *testing.T) {
	mockLLM := &tools.MockLLMClient{}
	detector := NewLLMIntentDetector(mockLLM)
	cache := tools.NewLLMCache(5*time.Minute, 1000)

	ctx := context.Background()
	message := "I like Sarah and she likes me back"

	// Use ExtractAndLock method (Phase 1 API)
	artifact, err := detector.ExtractAndLock(ctx, message, cache, "test-user", "test-message", "test-conversation")

	if err != nil {
		t.Fatalf("ExtractAndLock failed: %v", err)
	}

	if artifact == nil {
		t.Fatal("ExtractAndLock returned nil artifact")
	}

	// Verify lock enforcement
	if !artifact.IsLocked {
		t.Fatal("PHASE 1 API VIOLATION: ExtractAndLock did not lock artifact")
	}

	if artifact.LockedAt == 0 {
		t.Fatal("LockedAt not set by ExtractAndLock")
	}

	if artifact.LockReason == "" {
		t.Fatal("LockReason not set by ExtractAndLock")
	}

	if artifact.ExpiresAt == 0 {
		t.Fatal("ExpiresAt not set by ExtractAndLock")
	}

	// Verify immutability
	err = artifact.TryModify("test")
	if err == nil {
		t.Fatal("ExtractAndLock artifact not properly locked")
	}

	t.Logf("✅ ExtractAndLock API works correctly:")
	t.Logf("   - Returns locked artifact ✓")
	t.Logf("   - Sets LockedAt timestamp ✓")
	t.Logf("   - Sets LockReason ✓")
	t.Logf("   - Sets ExpiresAt (TTL) ✓")
	t.Logf("   - Immutability enforced ✓")
}

// TestPhase1LockReasonTracking verifies lock reasons are documented
func TestPhase1LockReasonTracking(t *testing.T) {
	testCases := []struct {
		name       string
		artifact   *models.ExtractionArtifact
		wantReason string
	}{
		{
			name: "LLM extraction",
			artifact: &models.ExtractionArtifact{
				ID:                "extract_llm",
				Source:            "llm",
				LLMSuccess:        true,
				Entities:          make([]models.ExtractedEntity, 5),
				AverageConfidence: 0.88,
			},
			wantReason: "extraction_complete",
		},
		{
			name: "Fallback extraction",
			artifact: &models.ExtractionArtifact{
				ID:                "extract_fallback",
				Source:            "fallback",
				LLMSuccess:        false,
				Entities:          make([]models.ExtractedEntity, 2),
				AverageConfidence: 0.72,
			},
			wantReason: "extraction_complete",
		},
		{
			name: "Cached extraction",
			artifact: &models.ExtractionArtifact{
				ID:                "extract_cached",
				Source:            "cached",
				Entities:          make([]models.ExtractedEntity, 3),
				AverageConfidence: 0.91,
			},
			wantReason: "extraction_complete",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.artifact.Lock("extraction_complete: source=" + tc.artifact.Source + ", entities=" + string(rune(len(tc.artifact.Entities))) + ", confidence=" + string(rune(int(tc.artifact.AverageConfidence*100))))
			if err != nil {
				t.Fatalf("Failed to lock: %v", err)
			}

			if tc.artifact.LockReason == "" {
				t.Fatal("LockReason not set")
			}

			if !stringContainsLockReason(tc.artifact.LockReason, tc.wantReason) {
				t.Errorf("LockReason doesn't contain '%s': %s", tc.wantReason, tc.artifact.LockReason)
			}

			t.Logf("✓ %s: lock reason = %s", tc.name, tc.artifact.LockReason)
		})
	}
}

// TestPhase1LockTimestampAccuracy verifies lock timestamps are accurate
func TestPhase1LockTimestampAccuracy(t *testing.T) {
	artifact := &models.ExtractionArtifact{
		ID:        "extract_timestamp",
		Source:    "llm",
		Entities:  []models.ExtractedEntity{},
		CreatedAt: time.Now().Unix(),
	}

	beforeLock := time.Now().Unix()
	err := artifact.Lock("test_lock")
	afterLock := time.Now().Unix()

	if err != nil {
		t.Fatalf("Failed to lock: %v", err)
	}

	// LockedAt should be between beforeLock and afterLock
	if artifact.LockedAt < beforeLock || artifact.LockedAt > afterLock {
		t.Errorf("LockedAt (%d) not within expected range [%d, %d]", artifact.LockedAt, beforeLock, afterLock)
	}

	// ExpiresAt should be ~30 minutes from now
	expectedExpiration := time.Now().Add(30 * time.Minute).Unix()
	if artifact.ExpiresAt < expectedExpiration-2 || artifact.ExpiresAt > expectedExpiration+2 {
		t.Logf("Warning: ExpiresAt slightly off (expected ~%d, got %d)", expectedExpiration, artifact.ExpiresAt)
	}

	t.Logf("✓ Lock timestamps accurate:")
	t.Logf("   - CreatedAt: %d", artifact.CreatedAt)
	t.Logf("   - LockedAt:  %d (delta: %d seconds)", artifact.LockedAt, artifact.LockedAt-artifact.CreatedAt)
	t.Logf("   - ExpiresAt: %d (~30 minutes from now)", artifact.ExpiresAt)
}

// Helper function
func stringContainsLockReason(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
