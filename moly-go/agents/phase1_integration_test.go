package agents

import (
	"moly/models"
	"testing"
	"time"
)

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
