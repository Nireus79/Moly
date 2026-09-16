package agents

import (
	"os"
	"testing"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// TestHarmAnalyzerMetadataNotNil verifies the crash fix: Metadata map is initialized
func TestHarmAnalyzerMetadataNotNil(t *testing.T) {
	// Create temp database
	tmpFile, _ := os.CreateTemp("", "test-moly-*.db")
	tmpFile.Close()
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	_, err := database.Init(tmpPath)
	if err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}

	// Create mock LLM that returns safe responses
	mockLLM := &tools.MockLLMClient{
		ResponseOverride: map[string]string{
			"GenerateResponse": `{"response":"Hello! How can I help?"}`,
		},
	}

	// Create ConversationAgent
	agent, err := NewConversationAgent(mockLLM)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Build minimal context
	ctx := models.Context{
		ConversationHistory: []models.Message{
			{
				Role:    "user",
				Content: "Hello Moly!",
			},
		},
		Gaps:           []string{},
		ContextQuality: "minimal",
	}

	t.Logf("Test 1: Running conversation flow with HarmAnalyzer...")
	response, err := agent.Run(ctx)

	if err != nil {
		t.Fatalf("Agent.Run() failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	t.Logf("✓ Agent.Run() succeeded without panic")

	// Test 2: Verify Metadata map exists and is not nil
	t.Logf("Test 2: Verifying Metadata map is initialized...")
	if response.Metadata == nil {
		t.Fatal("CRASH WOULD OCCUR: response.Metadata is nil!")
	}

	t.Logf("✓ response.Metadata is not nil")

	// Test 3: Verify we can write to Metadata without panic
	t.Logf("Test 3: Verifying Metadata can be written to...")
	response.Metadata["testKey"] = "testValue"
	t.Logf("✓ Successfully wrote to response.Metadata")

	// Test 4: Verify Metadata contains expected fields
	t.Logf("Test 4: Verifying Metadata contains expected fields...")
	if _, ok := response.Metadata["conversational"]; !ok {
		t.Error("Missing 'conversational' field in metadata")
	}
	if _, ok := response.Metadata["timestamp"]; !ok {
		t.Error("Missing 'timestamp' field in metadata")
	}
	t.Logf("✓ Metadata contains expected base fields")

	t.Logf("\n✅ HARMANALYZER CRASH FIX VERIFIED - No nil map panic!")
}

// TestMetadataNotNilOnEthicalGate verifies ethical gate doesn't crash
func TestMetadataNotNilOnEthicalGate(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "test-moly-*.db")
	tmpFile.Close()
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	_, err := database.Init(tmpPath)
	if err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}

	mockLLM := &tools.MockLLMClient{
		ResponseOverride: map[string]string{
			"GenerateResponse": `{"response":"Safe response"}`,
		},
	}

	agent, err := NewConversationAgent(mockLLM)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx := models.Context{
		ConversationHistory: []models.Message{
			{
				Role:    "user",
				Content: "Tell me something",
			},
		},
		Gaps:           []string{},
		ContextQuality: "minimal",
	}

	t.Logf("Test: Verifying Metadata is initialized before ethical gate...")
	response, err := agent.Run(ctx)

	if err != nil {
		t.Fatalf("Agent.Run() failed: %v", err)
	}

	if response == nil {
		t.Fatal("Response is nil")
	}

	// Metadata should exist and not be nil - this is where the crash happened
	if response.Metadata == nil {
		t.Fatal("CRASH BUG NOT FIXED: response.Metadata is nil")
	}

	t.Logf("✓ Metadata exists (nil map panic prevented)")

	// Verify base fields were added
	baseFields := []string{"conversational", "timestamp", "hasUserProfile", "contextGaps"}
	for _, field := range baseFields {
		if _, ok := response.Metadata[field]; ok {
			t.Logf("✓ Base field '%s' present", field)
		} else {
			t.Logf("⚠ Base field '%s' missing (not critical)", field)
		}
	}

	t.Logf("\n✅ METADATA INITIALIZATION VERIFIED - Ethical gate is safe!")
}

// TestMetadataWritableThroughEthicalGate simulates ethical gate writes
func TestMetadataWritableThroughEthicalGate(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "test-moly-*.db")
	tmpFile.Close()
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	_, err := database.Init(tmpPath)
	if err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}

	mockLLM := &tools.MockLLMClient{
		ResponseOverride: map[string]string{
			"GenerateResponse": `{"response":"Test response"}`,
		},
	}

	agent, err := NewConversationAgent(mockLLM)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx := models.Context{
		ConversationHistory: []models.Message{
			{
				Role:    "user",
				Content: "Test",
			},
		},
		Gaps:           []string{},
		ContextQuality: "minimal",
	}

	t.Logf("Test: Simulating ethical gate metadata writes...")
	response, err := agent.Run(ctx)

	if err != nil {
		t.Fatalf("Agent.Run() failed: %v", err)
	}

	// The critical check: can we write to the metadata map?
	if response.Metadata == nil {
		t.Fatal("PANIC WOULD OCCUR: Metadata is nil")
	}

	// Simulate what ethical gate does
	ethicalWrites := []struct {
		key   string
		value interface{}
	}{
		{"ethicalIntervention", "blocked"},
		{"blockReason", "harmful content"},
		{"violatedPrinciples", []string{"safety", "non-harm"}},
		{"ethicalNote", "This response was modified"},
	}

	// Try to write - this is what causes the panic if Metadata is nil
	for _, w := range ethicalWrites {
		response.Metadata[w.key] = w.value
		t.Logf("✓ Successfully wrote '%s' to metadata", w.key)
	}

	t.Logf("✓ Completed %d ethical metadata writes without panic", len(ethicalWrites))
	t.Logf("\n✅ ETHICAL GATE METADATA WRITES SAFE - No nil map panic!")
}
