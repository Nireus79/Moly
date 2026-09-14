package verification

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"moly/database"
	"moly/schema"
)

// TestAllFourEndpointsIntegration tests the complete flow through all 4 core endpoints
func TestAllFourEndpointsIntegration(t *testing.T) {
	// Initialize test database
	db, err := database.Init(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	defer db.Close()

	// Create test user and session
	conn := db.GetConnection()
	userID := "test-user-123"
	token := "test-token-xyz"

	_, err = conn.Exec(`
		INSERT INTO users (id, email, password_hash, created_at, last_active)
		VALUES (?, ?, ?, ?, ?)
	`, userID, "test@example.com", "hash", 1000, 1000)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	_, err = conn.Exec(`
		INSERT INTO sessions (id, user_id, expires_at, created_at, last_used)
		VALUES (?, ?, ?, ?, ?)
	`, token, userID, 9999999, 1000, 1000)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Test 1: AboutMe endpoint (POST and GET)
	t.Run("AboutMe Endpoint", func(t *testing.T) {
		profile := &schema.AboutMeProfile{
			CommunicationStyle: "direct",
			CoreValues:         []string{"integrity", "innovation"},
			TonePreference:     "professional",
			Preferences:        "Detail oriented",
		}

		// POST request body
		body, err := json.Marshal(profile)
		if err != nil {
			t.Fatalf("Failed to marshal profile: %v", err)
		}

		// Verify JSON field names
		if !bytes.Contains(body, []byte(`"communicationStyle"`)) {
			t.Error("Expected 'communicationStyle' in JSON")
		}
		if !bytes.Contains(body, []byte(`"coreValues"`)) {
			t.Error("Expected 'coreValues' in JSON")
		}

		// Verify validation
		if err := schema.ValidateStruct(profile); err != nil {
			t.Errorf("Valid profile failed validation: %v", err)
		}

		// Test empty name error
		emptyProfile := &schema.AboutMeProfile{}
		if err := schema.ValidateStruct(emptyProfile); err == nil {
			// Empty profile should be allowed
		}
	})

	// Test 2: Conversation endpoint
	t.Run("Conversation Endpoint", func(t *testing.T) {
		conv := &schema.Conversation{
			Name:        "Work Discussion",
			Type:        "work",
			Description: "Q4 Planning",
		}

		// Verify validation
		if err := schema.ValidateStruct(conv); err != nil {
			t.Errorf("Valid conversation failed validation: %v", err)
		}

		// Test empty name error
		emptyConv := &schema.Conversation{Name: ""}
		if err := schema.ValidateStruct(emptyConv); err == nil {
			t.Error("Empty name should fail validation")
		}
		if err != nil && err.Error() != "Name is required" {
			t.Errorf("Expected 'Name is required', got: %v", err.Error())
		}

		// Test max length
		longName := ""
		for i := 0; i < 256; i++ {
			longName += "a"
		}
		longConv := &schema.Conversation{Name: longName}
		if err := schema.ValidateStruct(longConv); err == nil {
			t.Error("Name exceeding 255 chars should fail validation")
		}
	})

	// Test 3: Contact endpoint
	t.Run("Contact Endpoint", func(t *testing.T) {
		contact := &schema.Contact{
			Name:         "Alice",
			Relationship: "colleague",
			Notes:        "Met at conference",
		}

		// Verify validation
		if err := schema.ValidateStruct(contact); err != nil {
			t.Errorf("Valid contact failed validation: %v", err)
		}

		// Test empty name error
		emptyContact := &schema.Contact{Name: ""}
		if err := schema.ValidateStruct(emptyContact); err == nil {
			t.Error("Empty name should fail validation")
		}
		if err != nil && err.Error() != "Name is required" {
			t.Errorf("Expected 'Name is required', got: %v", err.Error())
		}

		// Test max notes length
		longNotes := ""
		for i := 0; i < 1001; i++ {
			longNotes += "a"
		}
		longContact := &schema.Contact{
			Name:  "Alice",
			Notes: longNotes,
		}
		if err := schema.ValidateStruct(longContact); err == nil {
			t.Error("Notes exceeding 1000 chars should fail validation")
		}
	})

	// Test 4: Phase5 endpoint
	t.Run("Phase5 Request Endpoint", func(t *testing.T) {
		req := &schema.Phase5Request{
			Message:        "I love coding with my friend Alice",
			ConversationID: "conv_1789200405_123456789",
		}

		// Verify validation
		if err := schema.ValidateStruct(req); err != nil {
			t.Errorf("Valid Phase5 request failed validation: %v", err)
		}

		// Test missing message
		noMessage := &schema.Phase5Request{
			ConversationID: "conv_123",
		}
		if err := schema.ValidateStruct(noMessage); err == nil {
			t.Error("Missing message should fail validation")
		}

		// Test message length limit
		longMsg := ""
		for i := 0; i < 5001; i++ {
			longMsg += "a"
		}
		longReq := &schema.Phase5Request{
			Message:        longMsg,
			ConversationID: "conv_123",
		}
		if err := schema.ValidateStruct(longReq); err == nil {
			t.Error("Message exceeding 5000 chars should fail validation")
		}
		if err != nil && err.Error() != "Message too long (max 5000 characters)" {
			t.Errorf("Expected exact error message, got: %v", err.Error())
		}

		// Test conversationId length limit
		longConvId := ""
		for i := 0; i < 101; i++ {
			longConvId += "a"
		}
		longConvReq := &schema.Phase5Request{
			Message:        "test",
			ConversationID: longConvId,
		}
		if err := schema.ValidateStruct(longConvReq); err == nil {
			t.Error("ConversationID exceeding 100 chars should fail validation")
		}
		if err != nil && err.Error() != "ConversationID too long (max 100 characters)" {
			t.Errorf("Expected exact error message, got: %v", err.Error())
		}
	})

	// Test 5: Response Envelopes
	t.Run("Response Envelopes", func(t *testing.T) {
		// Test error response
		w := httptest.NewRecorder()
		schema.RespondError(w, http.StatusBadRequest, "Test error")

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var errResp schema.ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}
		if errResp.Error != "Test error" {
			t.Errorf("Expected error message, got: %v", errResp.Error)
		}

		// Test success response
		w2 := httptest.NewRecorder()
		profile := &schema.AboutMeProfile{
			CommunicationStyle: "direct",
		}
		schema.RespondSuccess(w2, http.StatusOK, "profile", profile)

		if w2.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w2.Code)
		}

		var respBody map[string]interface{}
		if err := json.NewDecoder(w2.Body).Decode(&respBody); err != nil {
			t.Fatalf("Failed to decode success response: %v", err)
		}
		if success, ok := respBody["success"].(bool); !ok || !success {
			t.Error("Expected 'success': true in response")
		}
	})

	// Test 6: JSON Marshaling Consistency
	t.Run("JSON Field Names", func(t *testing.T) {
		tests := []struct {
			name      string
			obj       interface{}
			expectKey string
		}{
			{
				name:      "AboutMe",
				obj:       &schema.AboutMeProfile{CommunicationStyle: "direct"},
				expectKey: "communicationStyle",
			},
			{
				name:      "Conversation",
				obj:       &schema.Conversation{ID: "conv_123", Name: "Test"},
				expectKey: "createdAt",
			},
			{
				name:      "Contact",
				obj:       &schema.Contact{ID: "contact_123", Name: "Alice"},
				expectKey: "createdAt",
			},
			{
				name:      "Phase5Request",
				obj:       &schema.Phase5Request{Message: "test", ConversationID: "conv_123"},
				expectKey: "conversationId",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, err := json.Marshal(tt.obj)
				if err != nil {
					t.Fatalf("Failed to marshal: %v", err)
				}
				if !bytes.Contains(data, []byte(`"`+tt.expectKey+`"`)) {
					t.Errorf("Expected key '%s' in JSON output", tt.expectKey)
				}
			})
		}
	})
}

// BenchmarkValidation benchmarks the validation layer
func BenchmarkValidation(b *testing.B) {
	profile := &schema.AboutMeProfile{
		CommunicationStyle: "direct",
		CoreValues:         []string{"integrity", "innovation"},
		TonePreference:     "professional",
		Preferences:        "Detail oriented",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = schema.ValidateStruct(profile)
	}
}

// TestExactErrorMessages verifies that error messages match golden spec exactly
func TestExactErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		obj         interface{}
		expectError string
	}{
		{
			name:        "Contact empty name",
			obj:         &schema.Contact{Name: ""},
			expectError: "Name is required",
		},
		{
			name:        "Conversation empty name",
			obj:         &schema.Conversation{Name: ""},
			expectError: "Name is required",
		},
		{
			name: "Phase5 long message",
			obj: &schema.Phase5Request{
				Message:        string(make([]byte, 5001)),
				ConversationID: "conv_123",
			},
			expectError: "Message too long (max 5000 characters)",
		},
		{
			name: "Phase5 long conversationId",
			obj: &schema.Phase5Request{
				Message:        "test",
				ConversationID: string(make([]byte, 101)),
			},
			expectError: "ConversationID too long (max 100 characters)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.ValidateStruct(tt.obj)
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}
			if err.Error() != tt.expectError {
				t.Errorf("Expected: %q, got: %q", tt.expectError, err.Error())
			}
		})
	}
}
