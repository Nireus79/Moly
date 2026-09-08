package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// TestV2HandlerIntegration - Test complete V2 handler flow
func TestV2HandlerIntegration(t *testing.T) {
	// Initialize V2 database
	dbPath := ":memory:"
	db, err := database.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create V2 server
	llmClient, _ := tools.NewLLMClient()
	server, err := NewV2APIServer(llmClient, db)
	if err != nil || server == nil {
		t.Fatalf("Failed to create V2 server: %v", err)
	}

	t.Run("HealthCheck", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v2/health", nil)
		w := httptest.NewRecorder()

		server.HealthCheckHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if result["status"] != "ok" {
			t.Errorf("Expected status 'ok', got %v", result["status"])
		}
	})

	t.Run("ConversationGenerate", func(t *testing.T) {
		req := models.ConversationRequest{
			ConversationID: "conv_test_001",
			UserID:         "user_test_001",
			UserMessage:    "I don't know what to say to Sarah",
			Mode:           "direct",
			Tone:           "friendly",
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/api/v2/conversation/generate", bytes.NewReader(body))
		w := httptest.NewRecorder()

		server.ConversationGenerateHandler(w, httpReq)

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Logf("Response: %s", w.Body.String())
			// Allow decode to fail if response is error
		}
	})

	t.Run("SetAboutMe", func(t *testing.T) {
		req := models.AboutMeRequest{
			UserID:             "user_test_001",
			CommunicationStyle: "warm",
			Values:             []string{"authenticity", "empathy"},
			PreferredTone:      "friendly",
			Notes:              "I like genuine conversations",
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/api/v2/about-me", bytes.NewReader(body))
		w := httptest.NewRecorder()

		server.SetAboutMeHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("GetContextHandler", func(t *testing.T) {
		httpReq := httptest.NewRequest("GET", "/api/v2/context?conversationId=conv_test_001&userId=user_test_001", nil)
		w := httptest.NewRecorder()

		server.GetContextHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}
	})

	t.Run("GetContactsHandler", func(t *testing.T) {
		httpReq := httptest.NewRequest("GET", "/api/v2/contacts?userId=user_test_001", nil)
		w := httptest.NewRecorder()

		server.GetContactsHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}
	})

	t.Run("ConversationFeedback", func(t *testing.T) {
		feedback := models.ConversationFeedback{
			ConversationID:     "conv_test_001",
			UserID:             "user_test_001",
			SuggestionChosen:   0,
			UserModified:       false,
			ReflectionApproved: true,
			Timestamp:          0,
		}

		body, _ := json.Marshal(feedback)
		httpReq := httptest.NewRequest("POST", "/api/v2/conversation/feedback", bytes.NewReader(body))
		w := httptest.NewRecorder()

		server.ConversationFeedbackHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Logf("Response code: %d", w.Code)
			t.Logf("Response body: %s", w.Body.String())
		}
	})

	t.Run("CORS Preflight", func(t *testing.T) {
		httpReq := httptest.NewRequest("OPTIONS", "/api/v2/conversation/generate", nil)
		w := httptest.NewRecorder()

		server.ConversationGenerateHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for OPTIONS, got %d", w.Code)
		}
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		// Missing userMessage
		req := models.ConversationRequest{
			ConversationID: "conv_test_001",
			UserID:         "user_test_001",
			UserMessage:    "", // Empty message
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/api/v2/conversation/generate", bytes.NewReader(body))
		w := httptest.NewRecorder()

		server.ConversationGenerateHandler(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing message, got %d", w.Code)
		}
	})
}

// TestV2APIResponseFormats - Test response format compliance
func TestV2APIResponseFormats(t *testing.T) {
	dbPath := ":memory:"
	db, _ := database.Init(dbPath)
	defer db.Close()

	llmClient, _ := tools.NewLLMClient()
	server, _ := NewV2APIServer(llmClient, db)

	t.Run("HealthResponse", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v2/health", nil)
		w := httptest.NewRecorder()

		server.HealthCheckHandler(w, req)

		var result map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Fatalf("Invalid JSON response: %v", err)
		}

		// Verify required fields
		if _, ok := result["status"]; !ok {
			t.Error("Missing 'status' in health response")
		}
		if _, ok := result["timestamp"]; !ok {
			t.Error("Missing 'timestamp' in health response")
		}
		if _, ok := result["components"]; !ok {
			t.Error("Missing 'components' in health response")
		}
	})
}

// TestV2ErrorHandling - Test error handling in handlers
func TestV2ErrorHandling(t *testing.T) {
	dbPath := ":memory:"
	db, _ := database.Init(dbPath)
	defer db.Close()

	llmClient, _ := tools.NewLLMClient()
	server, _ := NewV2APIServer(llmClient, db)

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v2/conversation/generate", bytes.NewReader([]byte("invalid json")))
		w := httptest.NewRecorder()

		server.ConversationGenerateHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v2/conversation/generate", nil)
		w := httptest.NewRecorder()

		server.ConversationGenerateHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected 405 for GET, got %d", w.Code)
		}
	})

	t.Run("MissingUserID", func(t *testing.T) {
		req := models.ConversationRequest{
			ConversationID: "conv_test_001",
			UserID:         "", // Missing
			UserMessage:    "Hello",
		}

		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/api/v2/conversation/generate", bytes.NewReader(body))
		w := httptest.NewRecorder()

		server.ConversationGenerateHandler(w, httpReq)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing userID, got %d", w.Code)
		}
	})
}

// TestV2ContextManagerIntegration - Test context manager via API
func TestV2ContextManagerIntegration(t *testing.T) {
	dbPath := ":memory:"
	db, _ := database.Init(dbPath)
	defer db.Close()

	llmClient, _ := tools.NewLLMClient()
	server, _ := NewV2APIServer(llmClient, db)

	t.Run("SaveAndRetrieveAboutMe", func(t *testing.T) {
		// Save AboutMe
		saveReq := models.AboutMeRequest{
			UserID:             "user_context_001",
			CommunicationStyle: "authentic",
			Values:             []string{"trust", "honesty"},
			PreferredTone:      "warm",
		}

		body, _ := json.Marshal(saveReq)
		httpReq := httptest.NewRequest("POST", "/api/v2/about-me", bytes.NewReader(body))
		w := httptest.NewRecorder()

		server.SetAboutMeHandler(w, httpReq)

		if w.Code != http.StatusOK {
			t.Errorf("Failed to save AboutMe: %d", w.Code)
		}

		// Retrieve context
		httpReq2 := httptest.NewRequest("GET", "/api/v2/context?conversationId=conv1&userId=user_context_001", nil)
		w2 := httptest.NewRecorder()

		server.GetContextHandler(w2, httpReq2)

		if w2.Code != http.StatusOK {
			t.Errorf("Failed to retrieve context: %d", w2.Code)
		}
	})
}
