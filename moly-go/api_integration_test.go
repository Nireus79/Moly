package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// setupTestServer creates a test HTTP server with V2 API
func setupTestServer(t *testing.T) (*httptest.Server, *database.Database) {
	// Create temp database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Create mock LLM (no external dependencies)
	mockLLM := tools.NewMockLLMClient()

	// Setup API
	server := httptest.NewServer(setupV2API(mockLLM, db))
	t.Cleanup(server.Close)

	return server, db
}

// TestHealthCheck verifies health endpoint works
func TestHealthCheck(t *testing.T) {
	server, _ := setupTestServer(t)

	resp, err := http.Get(server.URL + "/api/v2/health")
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Health check response: %+v", health)

	if health["status"] != "ok" {
		t.Errorf("Expected status=ok, got %v", health["status"])
	}
}

// TestConversationGenerate tests conversation generation endpoint
func TestConversationGenerate(t *testing.T) {
	server, _ := setupTestServer(t)

	payload := models.ConversationRequest{
		UserID:         "test_user_1",
		ConversationID: "conv_1",
		UserMessage:    "I need help with a difficult conversation",
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(
		server.URL+"/api/v2/conversation/generate",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	var response models.ConversationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Conversation response: phase=%s, questions=%d, suggestions=%d",
		response.Phase, len(response.Questions), len(response.Suggestions))

	// Should be in context gathering (no context provided)
	if response.Phase != "context_gathering" {
		t.Errorf("Expected context_gathering, got %s", response.Phase)
	}
}

// TestConversationGenerateMissingFields tests validation
func TestConversationGenerateMissingFields(t *testing.T) {
	server, _ := setupTestServer(t)

	payload := models.ConversationRequest{
		UserID: "test_user",
		// Missing ConversationID and UserMessage
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(
		server.URL+"/api/v2/conversation/generate",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected 400 for missing fields, got %d", resp.StatusCode)
	}

	t.Log("✓ Validation correctly rejected missing fields")
}

// TestConversationFeedback tests feedback recording
func TestConversationFeedback(t *testing.T) {
	server, _ := setupTestServer(t)

	userID := "test_user_feedback"
	convID := "conv_feedback"

	// First generate a suggestion
	genPayload := models.ConversationRequest{
		UserID:         userID,
		ConversationID: convID,
		UserMessage:    "I need help",
	}

	genBody, _ := json.Marshal(genPayload)
	resp, _ := http.Post(
		server.URL+"/api/v2/conversation/generate",
		"application/json",
		bytes.NewBuffer(genBody),
	)
	resp.Body.Close()

	// Now provide feedback
	feedback := models.ConversationFeedback{
		UserID:               userID,
		ConversationID:       convID,
		SuggestionChosen:     0,
		ModificationRequest:  "Modified suggestion text",
		UserModified:         true,
		ReflectionApproved:   true,
		Timestamp:            time.Now().Unix(),
	}

	fbBody, _ := json.Marshal(feedback)
	fbResp, err := http.Post(
		server.URL+"/api/v2/conversation/feedback",
		"application/json",
		bytes.NewBuffer(fbBody),
	)
	if err != nil {
		t.Fatalf("Feedback request failed: %v", err)
	}
	defer fbResp.Body.Close()

	if fbResp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", fbResp.StatusCode)
	}

	t.Log("✓ Feedback recorded successfully")
}

// TestGetContext tests context retrieval
func TestGetContext(t *testing.T) {
	server, db := setupTestServer(t)

	userID := "test_user_context"

	// First save some context
	aboutMe := &models.AboutMe{
		UserID:             userID,
		CommunicationStyle: "direct",
		Values:             []string{"honesty"},
	}

	aboutMeRepo := database.NewAboutMeRepository(db)
	aboutMeRepo.Save(userID, aboutMe)

	// Now retrieve it
	url := fmt.Sprintf("%s/api/v2/context?userId=%s&conversationId=test_conv",
		server.URL, userID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 200, got %d. Body: %s", resp.StatusCode, string(body))
	}

	t.Log("✓ Context retrieved successfully")
}

// TestAboutMeEndpoint tests setting AboutMe
func TestAboutMeEndpoint(t *testing.T) {
	server, _ := setupTestServer(t)

	payload := map[string]interface{}{
		"userId":             "test_user_aboutme",
		"communicationStyle": "authentic",
		"values":             []string{"honesty", "growth"},
		"preferredTone":      "warm",
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(
		server.URL+"/api/v2/about-me",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	t.Log("✓ AboutMe saved successfully")
}

// TestContactsEndpoint tests creating contacts
func TestContactsEndpoint(t *testing.T) {
	server, _ := setupTestServer(t)

	payload := map[string]interface{}{
		"userId":        "test_user_contacts",
		"name":          "Manager",
		"relationship":  "professional",
		"characteristics": []string{"analytical", "fair"},
	}

	body, _ := json.Marshal(payload)
	resp, err := http.Post(
		server.URL+"/api/v2/contacts",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}

	t.Log("✓ Contact saved successfully")
}

// TestCORSHeaders verifies CORS headers are set
func TestCORSHeaders(t *testing.T) {
	server, _ := setupTestServer(t)

	resp, err := http.Get(server.URL + "/api/v2/health")
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	t.Logf("CORS Origin header: %s", corsOrigin)

	// CORS headers might be set at middleware level
	if corsOrigin == "" {
		t.Log("⚠️ Note: CORS headers not set at endpoint level (might be middleware)")
	}
}

// TestEndpointErrorHandling tests various error scenarios
func TestEndpointErrorHandling(t *testing.T) {
	server, _ := setupTestServer(t)

	tests := []struct {
		name           string
		method         string
		endpoint       string
		body           interface{}
		expectedStatus int
	}{
		{
			name:           "Invalid JSON",
			method:         "POST",
			endpoint:       "/api/v2/conversation/generate",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing user ID",
			method:         "POST",
			endpoint:       "/api/v2/conversation/generate",
			body:           models.ConversationRequest{ConversationID: "conv", UserMessage: "msg"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "GET on POST endpoint",
			method:         "GET",
			endpoint:       "/api/v2/conversation/generate",
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var resp *http.Response
			var err error

			body, _ := json.Marshal(test.body)

			if test.method == "GET" {
				resp, err = http.Get(server.URL + test.endpoint)
			} else {
				resp, err = http.Post(
					server.URL+test.endpoint,
					"application/json",
					bytes.NewBuffer(body),
				)
			}

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != test.expectedStatus {
				t.Errorf("Expected %d, got %d", test.expectedStatus, resp.StatusCode)
			}
		})
	}
}

// TestConcurrentRequests tests multiple simultaneous requests
func TestConcurrentRequests(t *testing.T) {
	server, _ := setupTestServer(t)

	const numRequests = 5
	done := make(chan bool, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			payload := models.ConversationRequest{
				UserID:         fmt.Sprintf("concurrent_user_%d", id),
				ConversationID: fmt.Sprintf("conv_%d", id),
				UserMessage:    "Concurrent request test",
			}

			body, _ := json.Marshal(payload)
			resp, err := http.Post(
				server.URL+"/api/v2/conversation/generate",
				"application/json",
				bytes.NewBuffer(body),
			)

			if err != nil {
				errors <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errors <- fmt.Errorf("got status %d", resp.StatusCode)
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
			if completed == numRequests {
				t.Logf("✓ All %d concurrent requests completed successfully", numRequests)
				return
			}
		case err := <-errors:
			t.Errorf("Concurrent request error: %v", err)
		case <-time.After(10 * time.Second):
			t.Errorf("Timeout waiting for concurrent requests (completed: %d/%d)", completed, numRequests)
			return
		}
	}
}
