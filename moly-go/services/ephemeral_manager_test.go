package services

import (
	"path/filepath"
	"testing"
	"time"

	"moly/agents"
	"moly/database"
	"moly/tools"
)

// setupEphemeralTest creates test environment
func setupEphemeralTest(t *testing.T) *EphemeralConversationManager {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_ephemeral.db")

	systemKey := "test-ephemeral-key"
	db, err := database.Init(dbPath, systemKey)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	analyzer := agents.NewConversationAnalyzer(tools.NewMockLLMClient(), db)
	updater := NewProfileUpdater(db)

	return NewEphemeralConversationManager(db, analyzer, updater)
}

// TestEphemeralManagerSaveConversation tests saving a conversation
func TestEphemeralManagerSaveConversation(t *testing.T) {
	manager := setupEphemeralTest(t)

	messages := []agents.Message{
		{Role: "user", Content: "Hello", Timestamp: time.Now().Unix()},
		{Role: "assistant", Content: "Hi there!", Timestamp: time.Now().Unix()},
	}

	err := manager.SaveConversation(
		"test_user",
		"conv_123",
		messages,
		time.Now().Unix()-60,
		time.Now().Unix())

	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	t.Log("✓ Conversation saved successfully")
}

// TestEphemeralManagerSaveInvalidConversation tests validation
func TestEphemeralManagerSaveInvalidConversation(t *testing.T) {
	manager := setupEphemeralTest(t)

	// Test empty userID
	err := manager.SaveConversation("", "conv_123", []agents.Message{}, time.Now().Unix(), time.Now().Unix())
	if err == nil {
		t.Error("Should error on empty userID")
	}

	// Test empty conversationID
	messages := []agents.Message{{Role: "user", Content: "test", Timestamp: time.Now().Unix()}}
	err = manager.SaveConversation("user1", "", messages, time.Now().Unix(), time.Now().Unix())
	if err == nil {
		t.Error("Should error on empty conversationID")
	}

	// Test empty messages
	err = manager.SaveConversation("user1", "conv_1", []agents.Message{}, time.Now().Unix(), time.Now().Unix())
	if err == nil {
		t.Error("Should error on empty messages")
	}

	t.Log("✓ Validation working correctly")
}

// TestEphemeralManagerGetQueueStatus tests queue status query
func TestEphemeralManagerGetQueueStatus(t *testing.T) {
	manager := setupEphemeralTest(t)

	// Save a conversation to generate queue entry
	messages := []agents.Message{
		{Role: "user", Content: "test", Timestamp: time.Now().Unix()},
	}

	err := manager.SaveConversation("user1", "conv_1", messages, time.Now().Unix(), time.Now().Unix())
	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	// Check queue status
	status, err := manager.GetQueueStatus()
	if err != nil {
		t.Fatalf("GetQueueStatus failed: %v", err)
	}

	if status["pending"] != 1 {
		t.Errorf("Expected 1 pending item, got %d", status["pending"])
	}

	t.Logf("✓ Queue status: %+v", status)
}

// TestEphemeralManagerGetConversationCount tests counting active conversations
func TestEphemeralManagerGetConversationCount(t *testing.T) {
	manager := setupEphemeralTest(t)

	// Save multiple conversations
	for i := 0; i < 3; i++ {
		messages := []agents.Message{
			{Role: "user", Content: "test", Timestamp: time.Now().Unix()},
		}

		conversationID := string(rune('a' + i))
		err := manager.SaveConversation("user1", "conv_"+conversationID, messages,
			time.Now().Unix(), time.Now().Unix())
		if err != nil {
			t.Fatalf("SaveConversation failed: %v", err)
		}
	}

	count, err := manager.GetEphemeralConversationCount()
	if err != nil {
		t.Fatalf("GetEphemeralConversationCount failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 conversations, got %d", count)
	}

	t.Logf("✓ Active conversations: %d", count)
}

// TestEphemeralManagerCleanupExpired tests cleanup of old conversations
func TestEphemeralManagerCleanupExpired(t *testing.T) {
	manager := setupEphemeralTest(t)

	// The cleanup logic would need to manually insert old records
	// since SaveConversation sets expires_at to 24h in future
	// For now, just verify the function doesn't error

	stats, err := manager.CleanupExpired()
	if err != nil {
		t.Fatalf("CleanupExpired failed: %v", err)
	}

	t.Logf("✓ Cleanup executed: deleted %d conversations", stats.DeletedCount)
}

// TestEphemeralManagerRetryFailed tests retrying failed extractions
func TestEphemeralManagerRetryFailed(t *testing.T) {
	manager := setupEphemeralTest(t)

	count, err := manager.RetryFailedExtractions()
	if err != nil {
		t.Fatalf("RetryFailedExtractions failed: %v", err)
	}

	t.Logf("✓ Retry attempted: %d items", count)
}

// TestEphemeralManagerProcessQueue tests queue processing
func TestEphemeralManagerProcessQueue(t *testing.T) {
	manager := setupEphemeralTest(t)

	// Save a conversation
	messages := []agents.Message{
		{Role: "user", Content: "I need to tell my boss she's unfair", Timestamp: time.Now().Unix()},
		{Role: "assistant", Content: "That sounds challenging", Timestamp: time.Now().Unix()},
	}

	err := manager.SaveConversation("user1", "conv_process", messages,
		time.Now().Unix()-60, time.Now().Unix())
	if err != nil {
		t.Fatalf("SaveConversation failed: %v", err)
	}

	// Process queue
	stats, err := manager.ProcessQueue()
	if err != nil {
		t.Fatalf("ProcessQueue failed: %v", err)
	}

	t.Logf("✓ Queue processing: %d processed, %d failed", stats.Processed, stats.Failed)
}

// TestEphemeralManagerMultipleConversations tests handling multiple conversations
func TestEphemeralManagerMultipleConversations(t *testing.T) {
	manager := setupEphemeralTest(t)

	// Save multiple conversations from different users
	for userNum := 1; userNum <= 3; userNum++ {
		for convNum := 1; convNum <= 2; convNum++ {
			userID := string(rune('A' + userNum - 1))
			convID := string(rune('a' + convNum - 1))

			messages := []agents.Message{
				{Role: "user", Content: "test message", Timestamp: time.Now().Unix()},
			}

			err := manager.SaveConversation("user_"+userID, "conv_"+convID, messages,
				time.Now().Unix(), time.Now().Unix())
			if err != nil {
				t.Fatalf("SaveConversation failed: %v", err)
			}
		}
	}

	count, err := manager.GetEphemeralConversationCount()
	if err != nil {
		t.Fatalf("GetEphemeralConversationCount failed: %v", err)
	}

	if count != 6 {
		t.Errorf("Expected 6 conversations, got %d", count)
	}

	t.Logf("✓ Multiple conversations: %d total", count)
}

// TestEphemeralManagerGate - Complete manager verification
func TestEphemeralManagerGate(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"SaveConversation", TestEphemeralManagerSaveConversation},
		{"SaveInvalidConversation", TestEphemeralManagerSaveInvalidConversation},
		{"GetQueueStatus", TestEphemeralManagerGetQueueStatus},
		{"GetConversationCount", TestEphemeralManagerGetConversationCount},
		{"CleanupExpired", TestEphemeralManagerCleanupExpired},
		{"RetryFailed", TestEphemeralManagerRetryFailed},
		{"ProcessQueue", TestEphemeralManagerProcessQueue},
		{"MultipleConversations", TestEphemeralManagerMultipleConversations},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}

	t.Log("\n✓ EphemeralConversationManager gate PASSED - Ready for Phase 1.2")
}
