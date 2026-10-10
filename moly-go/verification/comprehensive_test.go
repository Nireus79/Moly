package verification

import (
	"testing"

	"moly/schema"
)


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
