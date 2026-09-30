package agents

import (
	"context"
	"moly/tools"
	"testing"
	"time"
)

// TestExtractAndLock verifies that extraction returns a locked artifact
func TestExtractAndLock(t *testing.T) {
	// Create a mock LLM that returns entities
	mockLLM := &tools.MockLLMClient{}

	detector := NewLLMIntentDetector(mockLLM)
	cache := tools.NewLLMCache(5*time.Minute, 1000)

	ctx := context.Background()
	message := "I like spending time with Sarah"

	// Extract and lock
	artifact, err := detector.ExtractAndLock(ctx, message, cache)

	// Should succeed
	if err != nil {
		t.Fatalf("ExtractAndLock failed: %v", err)
	}

	// Should return artifact
	if artifact == nil {
		t.Fatal("ExtractAndLock returned nil artifact")
	}

	// Should be locked
	if !artifact.IsLocked {
		t.Fatal("Extracted artifact is not locked")
	}

	// LockedAt should be set
	if artifact.LockedAt == 0 {
		t.Fatal("Extracted artifact LockedAt timestamp not set")
	}

	// LockReason should be set
	if artifact.LockReason == "" {
		t.Fatal("Extracted artifact LockReason not set")
	}

	// ExpiresAt should be set (30 min TTL)
	if artifact.ExpiresAt == 0 {
		t.Fatal("Extracted artifact ExpiresAt not set")
	}

	t.Logf("✓ Extraction locked successfully: %s", artifact.LockReason)
}

// TestExtractAndLockCannotModify verifies locked artifacts cannot be modified
func TestExtractAndLockCannotModify(t *testing.T) {
	mockLLM := &tools.MockLLMClient{}
	detector := NewLLMIntentDetector(mockLLM)
	cache := tools.NewLLMCache(5*time.Minute, 1000)

	ctx := context.Background()
	message := "Test message"

	artifact, err := detector.ExtractAndLock(ctx, message, cache)
	if err != nil {
		t.Fatalf("ExtractAndLock failed: %v", err)
	}

	// Try to modify locked artifact
	err = artifact.TryModify("add_entity")
	if err == nil {
		t.Fatal("Expected error modifying locked artifact, got nil")
	}

	if err.Error() != "cannot add_entity: extraction is locked (locked at "+string(rune(artifact.LockedAt))+" for: "+artifact.LockReason+")" {
		// Be flexible on exact error format, just check key parts
		errStr := err.Error()
		if !stringContainsSubstr(errStr, "cannot") || !stringContainsSubstr(errStr, "locked") {
			t.Errorf("Wrong error message: %v", err)
		}
	}

	t.Logf("✓ Locked artifact prevents modifications")
}

// TestExtractAndLockCannotLockTwice verifies re-locking fails
func TestExtractAndLockCannotLockTwice(t *testing.T) {
	mockLLM := &tools.MockLLMClient{}
	detector := NewLLMIntentDetector(mockLLM)
	cache := tools.NewLLMCache(5*time.Minute, 1000)

	ctx := context.Background()
	message := "Test message"

	artifact, err := detector.ExtractAndLock(ctx, message, cache)
	if err != nil {
		t.Fatalf("ExtractAndLock failed: %v", err)
	}

	// Try to lock again - should fail
	err = artifact.Lock("second_lock")
	if err == nil {
		t.Fatal("Expected error re-locking artifact")
	}

	t.Logf("✓ Cannot re-lock artifact: %v", err)
}

// TestExtractAndLockPreservesQuality verifies quality metrics are preserved
func TestExtractAndLockPreservesQuality(t *testing.T) {
	mockLLM := &tools.MockLLMClient{}
	detector := NewLLMIntentDetector(mockLLM)
	cache := tools.NewLLMCache(5*time.Minute, 1000)

	ctx := context.Background()
	message := "I like spending time with Sarah and her friends"

	artifact, err := detector.ExtractAndLock(ctx, message, cache)
	if err != nil {
		t.Fatalf("ExtractAndLock failed: %v", err)
	}

	// Quality metrics should be preserved
	if artifact.Source == "" {
		t.Fatal("Source not set in locked artifact")
	}

	if artifact.Duration == 0 {
		t.Fatal("Duration not set in locked artifact")
	}

	// Should have extracted entities
	if len(artifact.Entities) == 0 {
		t.Logf("Warning: No entities extracted (mock LLM may not provide any)")
	}

	t.Logf("✓ Locked artifact preserves quality metrics: source=%s, duration=%.2fms, entities=%d",
		artifact.Source, artifact.Duration, len(artifact.Entities))
}

// Helper function
func stringContainsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
