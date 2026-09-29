package agents

import (
	"testing"
	"time"

	"moly/models"
)

// TestLayer4ClarificationTrackingIntention tests if Layer 4 recognizes stated intention
func TestLayer4ClarificationTrackingIntention(t *testing.T) {
	agent := &conversationAgent{}

	history := []models.Message{
		{Role: "user", Content: "Hello Moly", Timestamp: time.Now().Unix()},
		{Role: "assistant", Content: "Hi there!", Timestamp: time.Now().Unix()},
		{Role: "user", Content: "I want to write a message to someone but I'm not sure what to say", Timestamp: time.Now().Unix()},
	}

	extractedCtx := &models.ExtractedContext{
		Intention: "get help with first message",
	}

	hasIntention := agent.hasClarificationBeenAddressed("userIntention", history, extractedCtx)
	if !hasIntention {
		t.Error("Expected to find user intention 'I want to write' but didn't")
	}

	t.Logf("✓ Correctly identified user intention from conversation")
}

// TestLayer4ClarificationTrackingInterests tests if Layer 4 recognizes stated interests
func TestLayer4ClarificationTrackingInterests(t *testing.T) {
	agent := &conversationAgent{}

	history := []models.Message{
		{Role: "user", Content: "I'm interested in someone, we have common interests in BDSM", Timestamp: time.Now().Unix()},
		{Role: "assistant", Content: "That's great to know", Timestamp: time.Now().Unix()},
		{Role: "user", Content: "We both like the same things and I want to connect with them", Timestamp: time.Now().Unix()},
	}

	extractedCtx := &models.ExtractedContext{
		Goals: []string{"write a message", "connect"},
	}

	hasInterests := agent.hasClarificationBeenAddressed("userInterests", history, extractedCtx)
	if !hasInterests {
		t.Error("Expected to find user interests but didn't")
	}

	t.Logf("✓ Correctly identified user interests from conversation")
}

// TestLayer4RequiresClarificationWhenMissing tests that Layer 4 still asks when info is missing
func TestLayer4RequiresClarificationWhenMissing(t *testing.T) {
	agent := &conversationAgent{}

	// Minimal message that doesn't express intention or interests clearly
	history := []models.Message{
		{Role: "user", Content: "Alice", Timestamp: time.Now().Unix()},
	}

	contact := &models.ExtractedContact{
		Name:         "Alice",
		Relationship: "romantic",
		Confidence:   0.9,
	}

	// No extracted context with goals - completely missing both
	extractedCtx := &models.ExtractedContext{
		Intention: "", // Empty intention
		Goals:     []string{}, // Empty goals
	}

	needsClarity := agent.shouldRequireClarificationForContact(contact, history, extractedCtx, nil)
	if !needsClarity {
		t.Error("Expected Layer 4 to require clarification for missing intention and interests, but it didn't")
	}

	t.Logf("✓ Correctly required clarification when intention and interests missing")
}

// TestLayer4AllowsProceedingWithBothIntentionAndInterests tests that Layer 4 proceeds when both are present
func TestLayer4AllowsProceedingWithBothIntentionAndInterests(t *testing.T) {
	agent := &conversationAgent{}

	history := []models.Message{
		{Role: "user", Content: "I want to write something playful to someone", Timestamp: time.Now().Unix()},
		{Role: "assistant", Content: "Tell me more", Timestamp: time.Now().Unix()},
		{Role: "user", Content: "We have common interests and I think we'd be great together", Timestamp: time.Now().Unix()},
	}

	contact := &models.ExtractedContact{
		Name:         "Lace",
		Relationship: "romantic",
		Confidence:   0.95,
	}

	extractedCtx := &models.ExtractedContext{
		Intention: "write a playful, smart message",
		Goals:     []string{"connect with Lace", "express interest"},
	}

	needsClarity := agent.shouldRequireClarificationForContact(contact, history, extractedCtx, nil)
	if needsClarity {
		t.Error("Expected Layer 4 to allow proceeding with both intention and interests, but it asked for clarification")
	}

	t.Logf("✓ Correctly allowed proceeding when both intention and interests present")
}

// TestLayer4RecognizesConfirmedPreferences tests that confirmed preferences bypass requirement
func TestLayer4RecognizesConfirmedPreferences(t *testing.T) {
	agent := &conversationAgent{}

	history := []models.Message{} // Empty history

	contact := &models.ExtractedContact{
		Name:         "Bob",
		Relationship: "professional",
		Confidence:   0.9,
	}

	confirmedPrefs := map[string]interface{}{
		"intention":                "send work email",
		"userInterestAlignment":    "professional development",
		"userIntentionWithContact": true,
	}

	needsClarity := agent.shouldRequireClarificationForContact(contact, history, nil, confirmedPrefs)
	if needsClarity {
		t.Error("Expected Layer 4 to skip clarification when preferences are confirmed, but it asked")
	}

	t.Logf("✓ Correctly recognized confirmed preferences")
}

// TestLayer4MultiMessageConversationIsSufficient tests that multi-message context is enough
func TestLayer4MultiMessageConversationIsSufficient(t *testing.T) {
	agent := &conversationAgent{}

	history := []models.Message{
		{Role: "user", Content: "Message 1: I want to write to someone", Timestamp: time.Now().Unix()},
		{Role: "assistant", Content: "Response 1", Timestamp: time.Now().Unix()},
		{Role: "user", Content: "Message 2: We have things in common", Timestamp: time.Now().Unix()},
		{Role: "assistant", Content: "Response 2", Timestamp: time.Now().Unix()},
		{Role: "user", Content: "Message 3: I think we'd get along", Timestamp: time.Now().Unix()},
	}

	contact := &models.ExtractedContact{
		Name:         "Sam",
		Relationship: "romantic",
	}

	extractedCtx := &models.ExtractedContext{
		Intention: "write message",
		Goals:     []string{"connect"},
	}

	needsClarity := agent.shouldRequireClarificationForContact(contact, history, extractedCtx, nil)
	if needsClarity {
		t.Error("Expected Layer 4 to accept 3+ message conversation, but it asked for more clarification")
	}

	t.Logf("✓ Correctly accepted multi-message conversation as sufficient")
}
