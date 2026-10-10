package tools

import (
	"moly/models"
	"testing"
)




// TestCannotLockTwice verifies that Lock() fails if already locked
func TestCannotLockTwice(t *testing.T) {
	artifact := &models.ExtractionArtifact{
		ID:       "extraction_test_4",
		IsLocked: false,
	}

	// Lock once
	err := artifact.Lock("first_lock")
	if err != nil {
		t.Fatalf("First lock failed: %v", err)
	}

	// Try to lock again - should fail
	err = artifact.Lock("second_lock")
	if err == nil {
		t.Fatal("Expected error locking twice")
	}
	if !stringContains(err.Error(), "already locked") {
		t.Errorf("Wrong error message: %v", err)
	}
}




// Helper function for string contains check
func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
