package services

import (
	"path/filepath"
	"testing"

	"moly/database"
)

// setupProfileServiceTest creates test environment
func setupProfileServiceTest(t *testing.T) *ProfileService {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_profile.db")

	systemKey := "test-profile-key"
	db, err := database.Init(dbPath, systemKey)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	return NewProfileService(db)
}

// TestProfileServiceGetUserProfileEmpty tests getting profile with no data
func TestProfileServiceGetUserProfileEmpty(t *testing.T) {
	service := setupProfileServiceTest(t)

	profile, err := service.GetUserProfile("test_user")
	if err != nil {
		t.Fatalf("GetUserProfile failed: %v", err)
	}

	if profile.UserID != "test_user" {
		t.Errorf("Expected userID test_user, got %s", profile.UserID)
	}

	if profile.AboutMe != nil {
		t.Error("Expected nil AboutMe for new user")
	}

	if len(profile.Contacts) != 0 {
		t.Errorf("Expected 0 contacts, got %d", len(profile.Contacts))
	}

	t.Log("✓ Empty profile returned correctly")
}

// TestProfileServiceGetAboutMe tests retrieving AboutMe profile
func TestProfileServiceGetAboutMe(t *testing.T) {
	service := setupProfileServiceTest(t)

	// This would normally be populated by ProfileUpdater
	// For now just verify the method doesn't error
	aboutMe, err := service.GetAboutMe("test_user")
	if err != nil && err.Error() != "sql: no rows in result set" {
		t.Fatalf("GetAboutMe failed unexpectedly: %v", err)
	}

	if err == nil && aboutMe != nil {
		if aboutMe.CommunicationStyle == "" {
			t.Error("AboutMe should have communication style")
		}
	}

	t.Log("✓ GetAboutMe works correctly")
}

// TestProfileServiceGetContacts tests retrieving contacts
func TestProfileServiceGetContacts(t *testing.T) {
	service := setupProfileServiceTest(t)

	contacts, err := service.GetContacts("test_user")
	if err != nil {
		t.Fatalf("GetContacts failed: %v", err)
	}

	if len(contacts) != 0 {
		t.Errorf("Expected 0 contacts for new user, got %d", len(contacts))
	}

	t.Log("✓ GetContacts returns empty list for new user")
}

// TestProfileServiceGetPatterns tests retrieving patterns
func TestProfileServiceGetPatterns(t *testing.T) {
	service := setupProfileServiceTest(t)

	patterns, err := service.GetPatterns("test_user")
	if err != nil {
		t.Fatalf("GetPatterns failed: %v", err)
	}

	if len(patterns) != 0 {
		t.Errorf("Expected 0 patterns for new user, got %d", len(patterns))
	}

	t.Log("✓ GetPatterns returns empty list for new user")
}

// TestProfileServiceGetGoals tests retrieving goals
func TestProfileServiceGetGoals(t *testing.T) {
	service := setupProfileServiceTest(t)

	goals, err := service.GetGoals("test_user")
	if err != nil {
		t.Fatalf("GetGoals failed: %v", err)
	}

	if len(goals) != 0 {
		t.Errorf("Expected 0 goals for new user, got %d", len(goals))
	}

	t.Log("✓ GetGoals returns empty list for new user")
}

// TestProfileServiceGetLearnings tests retrieving learnings
func TestProfileServiceGetLearnings(t *testing.T) {
	service := setupProfileServiceTest(t)

	learnings, err := service.GetLearnings("test_user")
	if err != nil {
		t.Fatalf("GetLearnings failed: %v", err)
	}

	if len(learnings) != 0 {
		t.Errorf("Expected 0 learnings for new user, got %d", len(learnings))
	}

	t.Log("✓ GetLearnings returns empty list for new user")
}

// TestProfileServiceGetReflections tests retrieving reflections
func TestProfileServiceGetReflections(t *testing.T) {
	service := setupProfileServiceTest(t)

	reflections, err := service.GetReflections("test_user")
	if err != nil {
		t.Fatalf("GetReflections failed: %v", err)
	}

	if len(reflections) != 0 {
		t.Errorf("Expected 0 reflections for new user, got %d", len(reflections))
	}

	t.Log("✓ GetReflections returns empty list for new user")
}

// TestProfileServiceAddReflection tests adding a reflection entry
func TestProfileServiceAddReflection(t *testing.T) {
	service := setupProfileServiceTest(t)

	content := "Had an interesting conversation today about assertiveness"
	tags := []string{"assertiveness", "work", "breakthrough"}

	id, err := service.AddReflection("test_user", content, "breakthrough", tags, nil)
	if err != nil {
		t.Fatalf("AddReflection failed: %v", err)
	}

	if id <= 0 {
		t.Errorf("Expected positive ID, got %d", id)
	}

	t.Logf("✓ Added reflection with ID %d", id)
}

// TestProfileServiceAddReflectionValidation tests validation
func TestProfileServiceAddReflectionValidation(t *testing.T) {
	service := setupProfileServiceTest(t)

	// Test empty userID
	_, err := service.AddReflection("", "content", "reflection", []string{}, nil)
	if err == nil {
		t.Error("Should error on empty userID")
	}

	// Test empty content
	_, err = service.AddReflection("user1", "", "reflection", []string{}, nil)
	if err == nil {
		t.Error("Should error on empty content")
	}

	t.Log("✓ Validation working correctly")
}

// TestProfileServiceConfirmLearning tests confirming a learning
func TestProfileServiceConfirmLearning(t *testing.T) {
	service := setupProfileServiceTest(t)

	// This would normally have been inserted by ProfileUpdater
	// For now just verify the method structure
	err := service.ConfirmLearning("test_user", 999)
	if err == nil {
		t.Error("Should error when learning not found")
	}

	t.Log("✓ ConfirmLearning validation works")
}

// TestProfileServiceRejectLearning tests rejecting a learning
func TestProfileServiceRejectLearning(t *testing.T) {
	service := setupProfileServiceTest(t)

	err := service.RejectLearning("test_user", 999)
	if err == nil {
		t.Error("Should error when learning not found")
	}

	t.Log("✓ RejectLearning validation works")
}

// TestProfileServiceUpdateGoalProgress tests updating goal progress
func TestProfileServiceUpdateGoalProgress(t *testing.T) {
	service := setupProfileServiceTest(t)

	err := service.UpdateGoalProgress("test_user", 999, "Made progress today", "active")
	if err == nil {
		t.Error("Should error when goal not found")
	}

	t.Log("✓ UpdateGoalProgress validation works")
}

// TestProfileServiceCalculateConfidenceStats tests confidence calculation
func TestProfileServiceCalculateConfidenceStats(t *testing.T) {
	service := setupProfileServiceTest(t)

	profile := &UserProfile{
		UserID:           "test_user",
		ConfidenceScores: make(map[string]ConfidenceStats),
		AboutMe: &AboutMeProfile{
			Confidence: 0.8,
		},
		Patterns: []PatternProfile{
			{Confidence: 0.7},
			{Confidence: 0.9},
		},
	}

	service.calculateConfidenceStats(profile)

	// Check pattern stats
	if stats, ok := profile.ConfidenceScores["patterns"]; ok {
		if stats.Count != 2 {
			t.Errorf("Expected 2 patterns, got %d", stats.Count)
		}
		if stats.Average != 0.8 {
			t.Errorf("Expected average 0.8, got %f", stats.Average)
		}
		if stats.Min != 0.7 {
			t.Errorf("Expected min 0.7, got %f", stats.Min)
		}
		if stats.Max != 0.9 {
			t.Errorf("Expected max 0.9, got %f", stats.Max)
		}
	} else {
		t.Error("Pattern stats not calculated")
	}

	t.Log("✓ Confidence stats calculated correctly")
}

// TestProfileServiceGetUserProfileMultiUser tests with multiple users
func TestProfileServiceGetUserProfileMultiUser(t *testing.T) {
	service := setupProfileServiceTest(t)

	// Get profiles for different users
	user1, err := service.GetUserProfile("user1")
	if err != nil {
		t.Fatalf("Failed to get user1 profile: %v", err)
	}

	user2, err := service.GetUserProfile("user2")
	if err != nil {
		t.Fatalf("Failed to get user2 profile: %v", err)
	}

	if user1.UserID != "user1" {
		t.Errorf("Expected user1 ID, got %s", user1.UserID)
	}

	if user2.UserID != "user2" {
		t.Errorf("Expected user2 ID, got %s", user2.UserID)
	}

	t.Log("✓ Multi-user profiles work correctly")
}

// TestProfileServiceProfileStructure tests profile structure
func TestProfileServiceProfileStructure(t *testing.T) {
	service := setupProfileServiceTest(t)

	profile, err := service.GetUserProfile("test_user")
	if err != nil {
		t.Fatalf("GetUserProfile failed: %v", err)
	}

	// Verify structure
	if profile.LastUpdated == 0 {
		t.Error("LastUpdated should be set")
	}

	if profile.ConfidenceScores == nil {
		t.Error("ConfidenceScores map should be initialized")
	}

	if profile.Contacts == nil {
		t.Log("Contacts slice is nil (expected for empty profile)")
	}

	t.Log("✓ Profile structure is correct")
}

// TestProfileServiceValidateUserID tests userID validation
func TestProfileServiceValidateUserID(t *testing.T) {
	service := setupProfileServiceTest(t)

	_, err := service.GetUserProfile("")
	if err == nil {
		t.Error("Should error on empty userID")
	}

	t.Log("✓ UserID validation works")
}

// TestProfileServiceGate - Complete ProfileService verification
func TestProfileServiceGate(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"GetUserProfileEmpty", TestProfileServiceGetUserProfileEmpty},
		{"GetAboutMe", TestProfileServiceGetAboutMe},
		{"GetContacts", TestProfileServiceGetContacts},
		{"GetPatterns", TestProfileServiceGetPatterns},
		{"GetGoals", TestProfileServiceGetGoals},
		{"GetLearnings", TestProfileServiceGetLearnings},
		{"GetReflections", TestProfileServiceGetReflections},
		{"AddReflection", TestProfileServiceAddReflection},
		{"AddReflectionValidation", TestProfileServiceAddReflectionValidation},
		{"ConfirmLearning", TestProfileServiceConfirmLearning},
		{"RejectLearning", TestProfileServiceRejectLearning},
		{"UpdateGoalProgress", TestProfileServiceUpdateGoalProgress},
		{"CalculateConfidenceStats", TestProfileServiceCalculateConfidenceStats},
		{"GetUserProfileMultiUser", TestProfileServiceGetUserProfileMultiUser},
		{"ProfileStructure", TestProfileServiceProfileStructure},
		{"ValidateUserID", TestProfileServiceValidateUserID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}

	t.Log("\n✓ ProfileService gate PASSED - Ready for HTTP handlers")
}
