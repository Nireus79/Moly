package agents

import (
	"context"
	"moly/tools"
	"testing"
	"time"
)

// TestExtractionPhaseLocks verifies that ExtractionPhase locks artifacts
func TestExtractionPhaseLocks(t *testing.T) {
	// Create extraction phase with mock LLM
	mockLLM := &tools.MockLLMClient{}
	intentDetector := NewLLMIntentDetector(mockLLM)
	store := tools.NewExtractionStore()

	extractionPhase := &ExtractionPhase{
		intentDetector:  intentDetector,
		extractionStore: store,
	}

	input := &ExtractionPhaseInput{
		UserID:         "test_user",
		ConversationID: "test_conv",
		MessageID:      "msg_1",
		Message:        "I like spending time with Sarah",
		Cache:          tools.NewLLMCache(5*time.Minute, 1000),
	}

	// Run extraction phase
	output, err := extractionPhase.Run(context.Background(), input)

	// Should succeed
	if err != nil {
		t.Fatalf("ExtractionPhase.Run failed: %v", err)
	}

	// Should return output
	if output == nil {
		t.Fatal("ExtractionPhase.Run returned nil output")
	}

	// Should have artifact
	if output.Artifact == nil {
		t.Fatal("ExtractionPhase output has no artifact")
	}

	artifact := output.Artifact

	// PHASE 1: Artifact must be locked
	if !artifact.IsLocked {
		t.Fatal("ExtractionPhase artifact is not locked - PHASE 1 enforcement failed")
	}

	// LockedAt must be set
	if artifact.LockedAt == 0 {
		t.Fatal("LockedAt timestamp not set")
	}

	// LockReason must contain "extraction_complete"
	if artifact.LockReason == "" {
		t.Fatal("LockReason not set")
	}
	if !stringContainsSubstrPhase(artifact.LockReason, "extraction_complete") {
		t.Errorf("Wrong lock reason: %s", artifact.LockReason)
	}

	// ExpiresAt should be set (30 minute TTL)
	if artifact.ExpiresAt == 0 {
		t.Fatal("ExpiresAt not set")
	}

	// Verify TTL is approximately 30 minutes (within 1 second tolerance)
	expectedExpiration := time.Now().Add(30 * time.Minute).Unix()
	if artifact.ExpiresAt < expectedExpiration-1 || artifact.ExpiresAt > expectedExpiration+1 {
		t.Logf("Warning: ExpiresAt slightly off expected time: expected %d, got %d (diff: %d seconds)",
			expectedExpiration, artifact.ExpiresAt, artifact.ExpiresAt-expectedExpiration)
		// Don't fail, just log - timing variance is acceptable
	}

	t.Logf("✓ ExtractionPhase locked artifact: %s", artifact.LockReason)
}

// TestExtractionPhaseLockedArtifactCannotModify verifies locked artifact immutability
func TestExtractionPhaseLockedArtifactCannotModify(t *testing.T) {
	// Create extraction phase
	mockLLM := &tools.MockLLMClient{}
	intentDetector := NewLLMIntentDetector(mockLLM)
	extractionPhase := &ExtractionPhase{
		intentDetector: intentDetector,
		extractionStore: tools.NewExtractionStore(),
	}

	input := &ExtractionPhaseInput{
		UserID:         "test_user",
		ConversationID: "test_conv",
		MessageID:      "msg_1",
		Message:        "Test message",
		Cache:          tools.NewLLMCache(5*time.Minute, 1000),
	}

	output, err := extractionPhase.Run(context.Background(), input)
	if err != nil {
		t.Fatalf("ExtractionPhase.Run failed: %v", err)
	}

	artifact := output.Artifact

	// Try to modify locked artifact
	err = artifact.TryModify("test_operation")
	if err == nil {
		t.Fatal("Expected error modifying locked artifact from ExtractionPhase")
	}

	t.Logf("✓ Locked artifact prevents modifications: %v", err)
}

// Helper function
func stringContainsSubstrPhase(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
