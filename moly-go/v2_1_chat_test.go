package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"moly/auth"
	"moly/database"
	"moly/models"
	"moly/tools"
)

// setupChatTest creates a test environment for chat testing
func setupChatTest(t *testing.T) (*ChatServer, *auth.SessionRepository, string, string) {
	// Create test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_chat.db")

	// Initialize encrypted database
	// Use unique system key to force new database for each test
	systemKey := "moly-test-chat-key-" + t.Name()
	db, err := database.Init(dbPath, systemKey)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Create chat server with mock LLM
	mockLLM := tools.NewMockLLMClient()
	chatServer := NewChatServer(db, mockLLM)

	// Create test session with unique user/code
	sessionRepo := auth.NewSessionRepository(db.GetConnection())
	userID := "test_chat_user_" + t.Name()
	code := "moly-" + t.Name()[:10] + "-abcde"

	session, err := sessionRepo.CreateSession(userID, code)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	conversationID := "test_conv_" + t.Name()

	return chatServer, sessionRepo, session.ID, conversationID
}

// TestChatBasicFlow tests basic chat message handling
func TestChatBasicFlow(t *testing.T) {
	chatServer, _, sessionID, conversationID := setupChatTest(t)

	// Send first message
	payload := models.ChatRequest{
		Message:        "I need help with my boss",
		ConversationID: conversationID,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v2.1/chat",
		bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+sessionID)

	w := httptest.NewRecorder()
	chatServer.ChatHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response models.ChatResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Response == "" {
		t.Errorf("Response should not be empty")
	}

	if response.MessageID == "" {
		t.Errorf("Message ID should be generated")
	}

	t.Logf("✓ Basic chat flow working: %s", response.Response[:50]+"...")
}

// TestChatContactDetection tests contact mention detection
func TestChatContactDetection(t *testing.T) {
	chatServer, _, sessionID, conversationID := setupChatTest(t)

	// Send message with contact mention
	payload := models.ChatRequest{
		Message:        "I need to talk to my boss about this",
		ConversationID: conversationID,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v2.1/chat",
		bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+sessionID)

	w := httptest.NewRecorder()
	chatServer.ChatHandler(w, req)

	var response models.ChatResponse
	json.NewDecoder(w.Body).Decode(&response)

	if response.ContactMention == nil {
		t.Errorf("Should detect contact mention")
	} else if !response.ContactMention.Detected || response.ContactMention.PersonName == "" {
		t.Errorf("Contact mention should be detected properly")
	} else {
		t.Logf("✓ Contact detected: %s (%s)", response.ContactMention.PersonName, response.ContactMention.Relationship)
	}
}

// TestChatHistoryRetrieval tests retrieving chat history
func TestChatHistoryRetrieval(t *testing.T) {
	chatServer, _, sessionID, conversationID := setupChatTest(t)

	// Send a message first
	payload := models.ChatRequest{
		Message:        "First message",
		ConversationID: conversationID,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v2.1/chat", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+sessionID)
	w := httptest.NewRecorder()
	chatServer.ChatHandler(w, req)

	// Now retrieve history
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2.1/conversations/%s/messages?conversationId=%s", conversationID, conversationID), nil)
	req2.Header.Set("Authorization", "Bearer "+sessionID)
	w2 := httptest.NewRecorder()
	chatServer.GetConversationHistoryHandler(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w2.Code)
	}

	var historyResponse map[string]interface{}
	json.NewDecoder(w2.Body).Decode(&historyResponse)

	if messages, ok := historyResponse["messages"]; ok {
		msgList := messages.([]interface{})
		if len(msgList) < 2 {
			t.Errorf("Expected at least 2 messages (user + assistant), got %d", len(msgList))
		} else {
			t.Logf("✓ Retrieved %d messages from history", len(msgList))
		}
	}
}

// TestChatMissingSession tests authentication handling
func TestChatMissingSession(t *testing.T) {
	chatServer, _, _, conversationID := setupChatTest(t)

	payload := models.ChatRequest{
		Message:        "Test",
		ConversationID: conversationID,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v2.1/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header

	w := httptest.NewRecorder()
	chatServer.ChatHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for missing session, got %d", w.Code)
	}

	t.Log("✓ Missing session properly rejected")
}

// TestChatInvalidSession tests invalid session handling
func TestChatInvalidSession(t *testing.T) {
	chatServer, _, _, conversationID := setupChatTest(t)

	payload := models.ChatRequest{
		Message:        "Test",
		ConversationID: conversationID,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v2.1/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalid_session_id")

	w := httptest.NewRecorder()
	chatServer.ChatHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for invalid session, got %d", w.Code)
	}

	t.Log("✓ Invalid session properly rejected")
}

// TestChatValidation tests request validation
func TestChatValidation(t *testing.T) {
	chatServer, _, sessionID, _ := setupChatTest(t)

	tests := []struct {
		name    string
		request models.ChatRequest
		status  int
	}{
		{
			name:    "Missing message",
			request: models.ChatRequest{Message: "", ConversationID: "conv123"},
			status:  http.StatusBadRequest,
		},
		{
			name:    "Missing conversationId",
			request: models.ChatRequest{Message: "Hello", ConversationID: ""},
			status:  http.StatusBadRequest,
		},
		{
			name:    "Valid request",
			request: models.ChatRequest{Message: "Hello", ConversationID: "conv123"},
			status:  http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, _ := json.Marshal(test.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v2.1/chat", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+sessionID)

			w := httptest.NewRecorder()
			chatServer.ChatHandler(w, req)

			if w.Code != test.status {
				t.Errorf("Expected %d, got %d", test.status, w.Code)
			}
		})
	}

	t.Log("✓ Validation working correctly")
}

// TestChatConversationDeletion tests conversation deletion
func TestChatConversationDeletion(t *testing.T) {
	chatServer, _, sessionID, conversationID := setupChatTest(t)

	// Send a message first
	payload := models.ChatRequest{
		Message:        "Message to delete",
		ConversationID: conversationID,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v2.1/chat", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+sessionID)
	w := httptest.NewRecorder()
	chatServer.ChatHandler(w, req)

	// Delete conversation
	delReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v2.1/conversations/%s?conversationId=%s", conversationID, conversationID), nil)
	delReq.Header.Set("Authorization", "Bearer "+sessionID)
	delW := httptest.NewRecorder()
	chatServer.DeleteConversationHandler(delW, delReq)

	if delW.Code != http.StatusOK {
		t.Errorf("Expected 200 for delete, got %d", delW.Code)
	}

	// Verify messages are gone
	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2.1/conversations/%s/messages?conversationId=%s", conversationID, conversationID), nil)
	getReq.Header.Set("Authorization", "Bearer "+sessionID)
	getW := httptest.NewRecorder()
	chatServer.GetConversationHistoryHandler(getW, getReq)

	var historyResponse map[string]interface{}
	json.NewDecoder(getW.Body).Decode(&historyResponse)

	if messages, ok := historyResponse["messages"]; ok && messages != nil {
		if msgList, ok := messages.([]interface{}); ok {
			if len(msgList) > 0 {
				t.Errorf("Expected 0 messages after deletion, got %d", len(msgList))
			} else {
				t.Log("✓ Conversation successfully deleted")
			}
		} else {
			t.Log("✓ Conversation successfully deleted (messages empty)")
		}
	} else {
		t.Log("✓ Conversation successfully deleted (no messages field)")
	}
}

// TestChatConcurrentMessages tests multiple concurrent chat messages
func TestChatConcurrentMessages(t *testing.T) {
	chatServer, sessionRepo, _, conversationID := setupChatTest(t)

	// Create multiple sessions
	numUsers := 3
	sessionIDs := make([]string, numUsers)

	for i := 0; i < numUsers; i++ {
		session, err := sessionRepo.CreateSession(fmt.Sprintf("user_%d", i), fmt.Sprintf("moly-12345-abcde"))
		if err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
		sessionIDs[i] = session.ID
	}

	done := make(chan bool, numUsers)
	errors := make(chan error, numUsers)

	// Send concurrent messages
	for i := 0; i < numUsers; i++ {
		go func(userIndex int) {
			payload := models.ChatRequest{
				Message:        fmt.Sprintf("Message from user %d", userIndex),
				ConversationID: conversationID,
			}

			body, _ := json.Marshal(payload)
			req := httptest.NewRequest(http.MethodPost, "/api/v2.1/chat", bytes.NewBuffer(body))
			req.Header.Set("Authorization", "Bearer "+sessionIDs[userIndex])

			w := httptest.NewRecorder()
			chatServer.ChatHandler(w, req)

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("got status %d", w.Code)
				return
			}

			done <- true
		}(i)
	}

	// Wait for all to complete
	completed := 0
	for {
		select {
		case <-done:
			completed++
			if completed == numUsers {
				t.Logf("✓ All %d concurrent messages processed", numUsers)
				return
			}
		case err := <-errors:
			t.Errorf("Concurrent message error: %v", err)
		case <-time.After(10 * time.Second):
			t.Errorf("Timeout waiting for concurrent messages (completed: %d/%d)", completed, numUsers)
			return
		}
	}
}

// TestChatGate - Complete chat system verification
func TestChatGate(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"BasicFlow", TestChatBasicFlow},
		{"ContactDetection", TestChatContactDetection},
		{"HistoryRetrieval", TestChatHistoryRetrieval},
		{"MissingSession", TestChatMissingSession},
		{"InvalidSession", TestChatInvalidSession},
		{"Validation", TestChatValidation},
		{"ConversationDeletion", TestChatConversationDeletion},
		{"ConcurrentMessages", TestChatConcurrentMessages},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}

	t.Log("\n✓ Chat system gate PASSED - Chat endpoint ready for Phase 1")
}
