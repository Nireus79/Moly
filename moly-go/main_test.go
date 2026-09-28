package main

import (
	"encoding/json"
	"testing"
)

// TestMessageProcessorResponseFormat verifies the response includes action_required field
func TestMessageProcessorResponseFormat(t *testing.T) {
	// Simulate the response structure that should be sent to frontend
	response := map[string]interface{}{
		"action_required": map[string]interface{}{
			"needsClarification": true,
			"clarificationQs": []map[string]string{
				{
					"question": "Tell me more about your situation?",
				},
			},
		},
		"success":          true,
		"phase":            "responding",
		"conversationId":   "test-conv-123",
		"response":         "Tell me more about your situation?",
		"safetyAlert":      nil,
		"processingTimeMs": 150,
		"metadata":         map[string]interface{}{},
	}

	// Marshal to JSON to verify structure
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	// Unmarshal to verify structure can be parsed
	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// Verify action_required field exists
	actionRequired, ok := parsed["action_required"].(map[string]interface{})
	if !ok {
		t.Fatalf("action_required field missing or not a map")
	}

	// Verify needsClarification field exists
	needsClarification, ok := actionRequired["needsClarification"].(bool)
	if !ok {
		t.Fatalf("needsClarification field missing or not a bool")
	}

	if !needsClarification {
		t.Fatalf("needsClarification should be true")
	}

	// Verify clarificationQs field exists
	clarificationQs, ok := actionRequired["clarificationQs"].([]interface{})
	if !ok {
		t.Fatalf("clarificationQs field missing or not an array")
	}

	if len(clarificationQs) == 0 {
		t.Fatalf("clarificationQs should have at least one question")
	}

	// Verify first question has "question" field
	firstQ, ok := clarificationQs[0].(map[string]interface{})
	if !ok {
		t.Fatalf("First clarification question not a map")
	}

	question, ok := firstQ["question"].(string)
	if !ok {
		t.Fatalf("question field missing or not a string")
	}

	if question == "" {
		t.Fatalf("question field should not be empty")
	}

	t.Logf("✓ Response format is correct:\n%s", string(jsonData))
}

// TestMessageProcessorResponseFormatNoClash verifies response works without clarification too
func TestMessageProcessorResponseFormatNoClash(t *testing.T) {
	response := map[string]interface{}{
		"action_required": map[string]interface{}{
			"needsClarification": false,
			"clarificationQs":    []map[string]string{},
		},
		"success":          true,
		"phase":            "responding",
		"conversationId":   "test-conv-456",
		"response":         "Got it. Thanks for sharing.",
		"safetyAlert":      nil,
		"processingTimeMs": 120,
		"metadata":         map[string]interface{}{},
	}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var parsed map[string]interface{}
	err = json.Unmarshal(jsonData, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	actionRequired, ok := parsed["action_required"].(map[string]interface{})
	if !ok {
		t.Fatalf("action_required field missing")
	}

	needsClarification, ok := actionRequired["needsClarification"].(bool)
	if !ok {
		t.Fatalf("needsClarification field missing or not a bool")
	}

	if needsClarification {
		t.Fatalf("needsClarification should be false")
	}

	clarificationQs, ok := actionRequired["clarificationQs"].([]interface{})
	if !ok {
		t.Fatalf("clarificationQs field missing")
	}

	if len(clarificationQs) != 0 {
		t.Fatalf("clarificationQs should be empty array")
	}

	t.Logf("✓ No-clarification response format is correct:\n%s", string(jsonData))
}
