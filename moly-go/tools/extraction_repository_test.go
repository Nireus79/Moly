package tools

import (
	"moly/models"
	"strconv"
	"testing"
	"time"
)

// TestSaveLockedEnforcement verifies that only locked extractions can be saved
func TestSaveLockedEnforcement(t *testing.T) {
	repo := NewExtractionRepository()

	// Create unlocked extraction
	unlockedArtifact := &models.ExtractionArtifact{
		ID:             "extraction_test_1",
		MessageID:      "msg_1",
		UserID:         "user_1",
		ConversationID: "conv_1",
		IsLocked:       false, // NOT locked
		Entities:       []models.ExtractedEntity{},
	}

	// Try to save unlocked extraction - should fail
	err := repo.SaveLocked(unlockedArtifact)
	if err == nil {
		t.Fatal("Expected error saving unlocked extraction, got nil")
	}
	if err.Error() != "cannot save unlocked extraction (ID: extraction_test_1) - must lock before persisting" {
		t.Errorf("Wrong error message: %v", err)
	}

	// Now lock it
	err = unlockedArtifact.Lock("test_lock")
	if err != nil {
		t.Fatalf("Failed to lock artifact: %v", err)
	}

	// Try to save locked extraction - should succeed
	err = repo.SaveLocked(unlockedArtifact)
	if err != nil {
		t.Fatalf("Failed to save locked extraction: %v", err)
	}

	// Verify it was saved
	if repo.Size() != 1 {
		t.Errorf("Expected 1 extraction in repo, got %d", repo.Size())
	}
}

// TestGetLockedEnforcement verifies that GetLocked returns error for unlocked extractions
func TestGetLockedEnforcement(t *testing.T) {
	repo := NewExtractionRepository()

	// Save a locked extraction
	artifact := &models.ExtractionArtifact{
		ID:             "extraction_test_2",
		MessageID:      "msg_2",
		UserID:         "user_1",
		ConversationID: "conv_1",
		IsLocked:       false,
		Entities:       []models.ExtractedEntity{},
	}
	artifact.Lock("test")
	repo.SaveLocked(artifact)

	// GetLocked should succeed
	retrieved, err := repo.GetLocked("extraction_test_2")
	if err != nil {
		t.Fatalf("Failed to get locked extraction: %v", err)
	}
	if retrieved.ID != "extraction_test_2" {
		t.Errorf("Wrong extraction returned")
	}

	// GetLocked with non-existent ID should fail
	_, err = repo.GetLocked("non_existent")
	if err == nil {
		t.Fatal("Expected error for non-existent extraction")
	}
}

// TestTryModifyEnforcement verifies that modifications are blocked for locked extractions
func TestTryModifyEnforcement(t *testing.T) {
	repo := NewExtractionRepository()

	artifact := &models.ExtractionArtifact{
		ID:             "extraction_test_3",
		MessageID:      "msg_3",
		UserID:         "user_1",
		ConversationID: "conv_1",
		IsLocked:       false,
		Entities:       []models.ExtractedEntity{},
	}
	artifact.Lock("test")
	repo.SaveLocked(artifact)

	// Try to modify locked extraction - should fail
	err := repo.TryModify("extraction_test_3", "add_entity")
	if err == nil {
		t.Fatal("Expected error modifying locked extraction")
	}

	// Error should mention the operation
	lockedAtStr := strconv.FormatInt(artifact.LockedAt, 10)
	expectedError := "cannot add_entity on locked extraction (ID: extraction_test_3, locked_at: " + lockedAtStr + " reason: test)"
	if err.Error() != expectedError {
		// We're flexible on exact format, just check key parts
		if !stringContains(err.Error(), "cannot") || !stringContains(err.Error(), "add_entity") || !stringContains(err.Error(), "locked") {
			t.Errorf("Wrong error message: %v", err)
		}
	}
}

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

// TestCleanupExpired removes expired locked extractions
func TestCleanupExpired(t *testing.T) {
	repo := NewExtractionRepository()

	now := time.Now().Unix()

	// Create extraction that expired
	expiredArtifact := &models.ExtractionArtifact{
		ID:        "extraction_expired",
		MessageID: "msg_exp",
		UserID:    "user_1",
		CreatedAt: now - 3600, // 1 hour ago
		ExpiresAt: now - 100,  // Already expired
		IsLocked:  false,
	}
	expiredArtifact.Lock("test")
	repo.SaveLocked(expiredArtifact)

	// Create extraction that hasn't expired
	activeArtifact := &models.ExtractionArtifact{
		ID:        "extraction_active",
		MessageID: "msg_act",
		UserID:    "user_1",
		CreatedAt: now - 100,  // 100 seconds ago
		ExpiresAt: now + 3600, // Expires 1 hour from now
		IsLocked:  false,
	}
	activeArtifact.Lock("test")
	repo.SaveLocked(activeArtifact)

	if repo.Size() != 2 {
		t.Errorf("Expected 2 extractions before cleanup, got %d", repo.Size())
	}

	// Cleanup
	removed := repo.CleanupExpired()

	if removed != 1 {
		t.Errorf("Expected to remove 1 extraction, removed %d", removed)
	}

	if repo.Size() != 1 {
		t.Errorf("Expected 1 extraction after cleanup, got %d", repo.Size())
	}

	// Verify the right one was kept
	remaining, err := repo.GetLocked("extraction_active")
	if err != nil {
		t.Fatalf("Failed to get remaining extraction: %v", err)
	}
	if remaining.ID != "extraction_active" {
		t.Errorf("Wrong extraction kept after cleanup")
	}
}

// TestDeleteIfUnlocked allows deletion of unlocked extractions
func TestDeleteIfUnlocked(t *testing.T) {
	repo := NewExtractionRepository()

	// Create unlocked extraction (failed processing, e.g.)
	unlockedArtifact := &models.ExtractionArtifact{
		ID:        "extraction_unlocked",
		MessageID: "msg_u",
		UserID:    "user_1",
		IsLocked:  false,
	}
	// Save directly to repo (bypassing SaveLocked)
	repo.mu.Lock()
	repo.extractions[unlockedArtifact.ID] = unlockedArtifact
	repo.mu.Unlock()

	// Delete unlocked - should succeed
	err := repo.DeleteIfUnlocked("extraction_unlocked")
	if err != nil {
		t.Fatalf("Failed to delete unlocked extraction: %v", err)
	}

	if repo.Size() != 0 {
		t.Errorf("Expected 0 extractions after delete, got %d", repo.Size())
	}

	// Try to delete locked extraction - should fail
	lockedArtifact := &models.ExtractionArtifact{
		ID:       "extraction_locked",
		IsLocked: false,
	}
	lockedArtifact.Lock("test")
	repo.SaveLocked(lockedArtifact)

	err = repo.DeleteIfUnlocked("extraction_locked")
	if err == nil {
		t.Fatal("Expected error deleting locked extraction")
	}
	if !stringContains(err.Error(), "cannot delete locked") {
		t.Errorf("Wrong error message: %v", err)
	}
}

// TestStats returns accurate statistics
func TestStats(t *testing.T) {
	repo := NewExtractionRepository()

	// Add 3 locked, 1 unlocked
	for i := 1; i <= 3; i++ {
		artifact := &models.ExtractionArtifact{
			ID:        "extraction_locked_" + string(rune(i)),
			IsLocked:  false,
			CreatedAt: time.Now().Unix() - int64(i*100),
		}
		artifact.Lock("test")
		repo.SaveLocked(artifact)
	}

	// Add unlocked (saved manually for testing)
	unlocked := &models.ExtractionArtifact{
		ID:       "extraction_unlocked",
		IsLocked: false,
	}
	repo.mu.Lock()
	repo.extractions[unlocked.ID] = unlocked
	repo.mu.Unlock()

	stats := repo.Stats()

	if stats.Total != 4 {
		t.Errorf("Expected 4 total extractions, got %d", stats.Total)
	}
	if stats.Locked != 3 {
		t.Errorf("Expected 3 locked extractions, got %d", stats.Locked)
	}
	if stats.Unlocked != 1 {
		t.Errorf("Expected 1 unlocked extraction, got %d", stats.Unlocked)
	}
	if stats.AverageAge <= 0 {
		t.Errorf("Expected positive average age, got %f", stats.AverageAge)
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
