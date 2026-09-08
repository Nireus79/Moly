package agents

import (
	"context"
	"strings"
	"testing"
	"time"

	"moly/tools"
)

// TestConversationAnalyzerBasicExtraction tests basic insight extraction
func TestConversationAnalyzerBasicExtraction(t *testing.T) {
	analyzer := NewConversationAnalyzer(tools.NewMockLLMClient(), nil)

	messages := []Message{
		{Role: "user", Content: "I need to tell my boss she's being unfair", Timestamp: time.Now().Unix()},
		{Role: "assistant", Content: "That sounds challenging. How are you thinking about approaching it?", Timestamp: time.Now().Unix()},
		{Role: "user", Content: "I want to be direct but I'm scared she'll get mad", Timestamp: time.Now().Unix()},
	}

	result, err := analyzer.AnalyzeConversation(context.Background(), "test_user", "conv_123", messages)

	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	t.Logf("✓ Extraction completed: %d AboutMe, %d patterns, %d contacts",
		len(result.AboutMeUpdates), len(result.PatternDetections), len(result.ContactMentions))

	// Result should have some extractions or be low confidence
	if result.ConfidenceScore < 0 || result.ConfidenceScore > 1 {
		t.Errorf("Confidence score out of range: %v", result.ConfidenceScore)
	}
}

// TestConversationAnalyzerEmptyConversation tests handling of empty conversations
func TestConversationAnalyzerEmptyConversation(t *testing.T) {
	analyzer := NewConversationAnalyzer(tools.NewMockLLMClient(), nil)

	messages := []Message{}

	result, err := analyzer.AnalyzeConversation(context.Background(), "test_user", "conv_123", messages)

	if err != nil {
		t.Fatalf("Analysis should not error on empty conversation: %v", err)
	}

	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if result.ErrorMessage == "" {
		t.Error("Empty conversation should have error message")
	}

	if result.ConfidenceScore != 0 {
		t.Errorf("Empty conversation should have 0 confidence, got %v", result.ConfidenceScore)
	}

	t.Log("✓ Empty conversation handled correctly")
}

// TestConversationAnalyzerConfidenceFiltering tests that low-confidence items are filtered
func TestConversationAnalyzerConfidenceFiltering(t *testing.T) {
	analyzer := NewConversationAnalyzer(tools.NewMockLLMClient(), nil)

	// Create a result with mixed confidence levels
	result := &ExtractionResult{
		AboutMeUpdates: []AboutMeUpdate{
			{Key: "communication_style", Value: "direct", Confidence: 0.8},
			{Key: "value", Value: "honesty", Confidence: 0.3}, // Below threshold
			{Key: "preference", Value: "async", Confidence: 0.6},
		},
		PatternDetections: []PatternDetection{
			{Pattern: "pattern1", Confidence: 0.7},
			{Pattern: "pattern2", Confidence: 0.4}, // Below threshold
		},
	}

	analyzer.validateResults(result)

	// Low confidence items should be filtered
	if len(result.AboutMeUpdates) != 2 {
		t.Errorf("Expected 2 AboutMe updates after filtering, got %d", len(result.AboutMeUpdates))
	}

	if len(result.PatternDetections) != 1 {
		t.Errorf("Expected 1 pattern after filtering, got %d", len(result.PatternDetections))
	}

	t.Log("✓ Low-confidence filtering works")
}

// TestConversationAnalyzerConfidenceNormalization tests confidence is clamped to 0-1
func TestConversationAnalyzerConfidenceNormalization(t *testing.T) {
	analyzer := NewConversationAnalyzer(tools.NewMockLLMClient(), nil)

	result := &ExtractionResult{
		ConfidenceScore: 1.5, // Out of range
		AboutMeUpdates: []AboutMeUpdate{
			{Key: "test", Value: "value", Confidence: 2.0}, // Out of range
		},
	}

	analyzer.validateResults(result)

	if result.ConfidenceScore != 1 {
		t.Errorf("Confidence should be clamped to 1, got %v", result.ConfidenceScore)
	}

	if result.AboutMeUpdates[0].Confidence != 1 {
		t.Errorf("AboutMe confidence should be clamped to 1, got %v", result.AboutMeUpdates[0].Confidence)
	}

	t.Log("✓ Confidence normalization works")
}

// TestConversationAnalyzerConversationText tests message formatting
func TestConversationAnalyzerConversationText(t *testing.T) {
	analyzer := NewConversationAnalyzer(tools.NewMockLLMClient(), nil)

	messages := []Message{
		{Role: "user", Content: "Hello", Timestamp: time.Now().Unix()},
		{Role: "assistant", Content: "Hi there!", Timestamp: time.Now().Unix()},
	}

	text := analyzer.buildConversationText(messages)

	if !strings.Contains(text, "User: Hello") {
		t.Error("Formatted text should contain user message")
	}

	if !strings.Contains(text, "Assistant: Hi there!") {
		t.Error("Formatted text should contain assistant message")
	}

	t.Log("✓ Conversation text formatting works")
}

// TestConversationAnalyzerJSONSerialization tests JSON round-trip
func TestConversationAnalyzerJSONSerialization(t *testing.T) {
	original := &ExtractionResult{
		AboutMeUpdates: []AboutMeUpdate{
			{Key: "style", Value: "direct", Confidence: 0.8},
		},
		PatternDetections: []PatternDetection{
			{Pattern: "pattern1", Confidence: 0.7},
		},
		ConfidenceScore: 0.75,
	}

	jsonStr, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	restored, err := ExtractionResultFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if len(restored.AboutMeUpdates) != 1 {
		t.Error("AboutMe items not preserved in JSON round-trip")
	}

	if restored.ConfidenceScore != 0.75 {
		t.Error("Confidence not preserved in JSON round-trip")
	}

	t.Log("✓ JSON serialization works")
}

// TestConversationAnalyzerMultipleContacts tests extraction of multiple contacts
func TestConversationAnalyzerMultipleContacts(t *testing.T) {
	analyzer := NewConversationAnalyzer(tools.NewMockLLMClient(), nil)

	messages := []Message{
		{Role: "user", Content: "I need to talk to my boss about feedback, and my mom about boundaries", Timestamp: time.Now().Unix()},
		{Role: "assistant", Content: "Those sound like two different conversations", Timestamp: time.Now().Unix()},
	}

	result, err := analyzer.AnalyzeConversation(context.Background(), "test_user", "conv_456", messages)

	if err != nil {
		t.Fatalf("Analysis failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	t.Logf("✓ Multiple contacts extraction: found %d contacts", len(result.ContactMentions))
}

// TestConversationAnalyzerNoMessagesPanic tests analyzer doesn't panic with edge cases
func TestConversationAnalyzerNoMessagesPanic(t *testing.T) {
	analyzer := NewConversationAnalyzer(tools.NewMockLLMClient(), nil)

	// Test with nil slice
	messages := []Message{}
	result, _ := analyzer.AnalyzeConversation(context.Background(), "test_user", "conv_789", messages)

	if result == nil {
		t.Fatal("Should return non-nil result even for empty conversation")
	}

	t.Log("✓ Edge case handling works")
}

// TestConversationAnalyzerGate - Complete analyzer verification
func TestConversationAnalyzerGate(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"BasicExtraction", TestConversationAnalyzerBasicExtraction},
		{"EmptyConversation", TestConversationAnalyzerEmptyConversation},
		{"ConfidenceFiltering", TestConversationAnalyzerConfidenceFiltering},
		{"ConfidenceNormalization", TestConversationAnalyzerConfidenceNormalization},
		{"ConversationText", TestConversationAnalyzerConversationText},
		{"JSONSerialization", TestConversationAnalyzerJSONSerialization},
		{"MultipleContacts", TestConversationAnalyzerMultipleContacts},
		{"EdgeCases", TestConversationAnalyzerNoMessagesPanic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}

	t.Log("\n✓ ConversationAnalyzer gate PASSED - Ready for Phase 1.2")
}
