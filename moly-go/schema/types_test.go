package schema

import (
	"strings"
	"testing"
)

// TestConversationNameValidation verifies exact error message for empty name.
func TestConversationNameValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     *Conversation
		wantError bool
		wantMsg   string
	}{
		{
			name: "empty name should error",
			input: &Conversation{
				ID:   "conv_123",
				Name: "",
			},
			wantError: true,
			wantMsg:   "Name is required",
		},
		{
			name: "valid name should pass",
			input: &Conversation{
				ID:   "conv_123",
				Name: "Test Conversation",
			},
			wantError: false,
		},
		{
			name: "name exceeding max length should error",
			input: &Conversation{
				ID:   "conv_123",
				Name: strings.Repeat("a", 256),
			},
			wantError: true,
			wantMsg:   "max 255",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateStruct() got error = %v, want error = %v", err, tt.wantError)
			}
			if tt.wantError && err != nil && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("ValidateStruct() error = %q, want to contain %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

// TestContactNameValidation verifies exact error message for empty contact name.
func TestContactNameValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     *Contact
		wantError bool
		wantMsg   string
	}{
		{
			name: "empty name should error",
			input: &Contact{
				ID:   "contact_123",
				Name: "",
			},
			wantError: true,
			wantMsg:   "Name is required",
		},
		{
			name: "valid name should pass",
			input: &Contact{
				ID:           "contact_123",
				Name:         "Alice",
				Relationship: "colleague",
			},
			wantError: false,
		},
		{
			name: "notes exceeding max length should error",
			input: &Contact{
				ID:    "contact_123",
				Name:  "Alice",
				Notes: strings.Repeat("a", 1001),
			},
			wantError: true,
			wantMsg:   "max 1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateStruct() got error = %v, want error = %v", err, tt.wantError)
			}
			if tt.wantError && err != nil && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("ValidateStruct() error = %q, want to contain %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

// TestPhase5RequestValidation verifies exact error messages for Phase5 constraints.
func TestPhase5RequestValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     *Phase5Request
		wantError bool
		wantMsg   string
	}{
		{
			name: "empty message should error",
			input: &Phase5Request{
				Message:        "",
				ConversationID: "conv_123",
			},
			wantError: true,
			wantMsg:   "Missing message or conversationId",
		},
		{
			name: "empty conversationId should error",
			input: &Phase5Request{
				Message:        "test message",
				ConversationID: "",
			},
			wantError: true,
			wantMsg:   "Missing message or conversationId",
		},
		{
			name: "message exceeding 5000 chars should error",
			input: &Phase5Request{
				Message:        strings.Repeat("a", 5001),
				ConversationID: "conv_123",
			},
			wantError: true,
			wantMsg:   "Message too long (max 5000 characters)",
		},
		{
			name: "conversationId exceeding 100 chars should error",
			input: &Phase5Request{
				Message:        "test message",
				ConversationID: strings.Repeat("a", 101),
			},
			wantError: true,
			wantMsg:   "ConversationID too long (max 100 characters)",
		},
		{
			name: "valid request should pass",
			input: &Phase5Request{
				Message:        "I love coding with my friend Alice",
				ConversationID: "conv_1789200405_123456789",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateStruct() got error = %v, want error = %v", err, tt.wantError)
			}
			if tt.wantError && err != nil && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("ValidateStruct() error = %q, want to contain %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

// TestAboutMeProfileValidation verifies field length constraints.
func TestAboutMeProfileValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     *AboutMeProfile
		wantError bool
		wantMsg   string
	}{
		{
			name: "empty profile should pass",
			input: &AboutMeProfile{
				CommunicationStyle: "",
				CoreValues:         []string{},
				TonePreference:     "",
				Preferences:        "",
			},
			wantError: false,
		},
		{
			name: "valid profile should pass",
			input: &AboutMeProfile{
				CommunicationStyle: "direct",
				CoreValues:         []string{"integrity", "innovation"},
				TonePreference:     "professional",
				Preferences:        "Detail oriented",
			},
			wantError: false,
		},
		{
			name: "communicationStyle exceeding 100 chars should error",
			input: &AboutMeProfile{
				CommunicationStyle: strings.Repeat("a", 101),
			},
			wantError: true,
			wantMsg:   "max 100",
		},
		{
			name: "tonePreference exceeding 100 chars should error",
			input: &AboutMeProfile{
				TonePreference: strings.Repeat("a", 101),
			},
			wantError: true,
			wantMsg:   "max 100",
		},
		{
			name: "preferences exceeding 1000 chars should error",
			input: &AboutMeProfile{
				Preferences: strings.Repeat("a", 1001),
			},
			wantError: true,
			wantMsg:   "max 1000",
		},
		{
			name: "coreValues exceeding 20 items should error",
			input: &AboutMeProfile{
				CoreValues: make([]string, 21),
			},
			wantError: true,
			wantMsg:   "max 20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateStruct() got error = %v, want error = %v", err, tt.wantError)
			}
			if tt.wantError && err != nil && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("ValidateStruct() error = %q, want to contain %q", err.Error(), tt.wantMsg)
			}
		})
	}
}
