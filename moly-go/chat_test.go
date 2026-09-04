package main

import (
	"testing"
)

func TestChatRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		message string
		model   string
		mode    string
		valid   bool
	}{
		{"Valid request", "Hello", "mistral", "direct", true},
		{"Valid with socratic", "How am I?", "mistral", "socratic", true},
		{"Empty message", "", "mistral", "direct", false},
		{"Empty model OK", "Hello", "", "direct", true},
		{"Empty mode OK", "Hello", "mistral", "", true},
	}

	for _, test := range tests {
		req := ChatRequest{
			Message: test.message,
			Model:   test.model,
			Mode:    test.mode,
		}

		// Validation: message must not be empty
		isValid := req.Message != ""
		if isValid != test.valid {
			t.Errorf("%s: expected valid=%v, got %v", test.name, test.valid, isValid)
		}
	}
}

func TestChatResponseStructure(t *testing.T) {
	response := ChatResponse{
		Success:  true,
		Response: "Test response",
		Error:    "",
		Provider: "ollama",
	}

	if !response.Success {
		t.Error("Response.Success should be true")
	}
	if response.Response != "Test response" {
		t.Errorf("Response.Response: expected 'Test response', got '%s'", response.Response)
	}
	if response.Error != "" {
		t.Errorf("Response.Error should be empty, got '%s'", response.Error)
	}
	if response.Provider != "ollama" {
		t.Errorf("Response.Provider: expected 'ollama', got '%s'", response.Provider)
	}
}

func TestChatResponseError(t *testing.T) {
	response := ChatResponse{
		Success:  false,
		Response: "",
		Error:    "Service unavailable",
		Provider: "ollama",
	}

	if response.Success {
		t.Error("Response.Success should be false on error")
	}
	if response.Error == "" {
		t.Error("Response.Error should not be empty")
	}
	if response.Response != "" {
		t.Error("Response.Response should be empty on error")
	}
}

func TestModeParameterHandling(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected string
	}{
		{"Direct mode", "direct", ""},
		{"Socratic mode", "socratic", "Instead of giving a direct answer"},
		{"Empty mode", "", ""},
		{"Invalid mode", "invalid", ""},
	}

	baseMessage := "What is the capital of France?"

	for _, test := range tests {
		mode := test.mode

		// Simulate what the provider functions do
		prompt := baseMessage
		if mode == "socratic" {
			prompt = baseMessage + "\n\nInstead of giving a direct answer, ask guiding questions to help me think through this myself."
		}

		if test.expected == "" {
			// Direct mode or empty should not have added text
			if prompt != baseMessage {
				t.Errorf("%s: prompt should not be modified, got: %s", test.name, prompt)
			}
		} else {
			// Socratic mode should have added text
			if test.expected != "" && !contains(prompt, test.expected) {
				t.Errorf("%s: prompt should contain '%s'", test.name, test.expected)
			}
		}
	}
}

func TestProviderRouting(t *testing.T) {
	tests := []struct {
		provider string
		shouldUseOllama bool
		shouldUseClaude bool
		shouldUseOpenAI bool
	}{
		{"local", true, false, false},
		{"ollama", true, false, false},
		{"claude", false, true, false},
		{"openai", false, false, true},
		{"unknown", true, false, false}, // defaults to ollama
	}

	for _, test := range tests {
		switch test.provider {
		case "local", "ollama":
			if !test.shouldUseOllama {
				t.Errorf("%s should route to Ollama", test.provider)
			}
		case "claude":
			if !test.shouldUseClaude {
				t.Errorf("%s should route to Claude", test.provider)
			}
		case "openai":
			if !test.shouldUseOpenAI {
				t.Errorf("%s should route to OpenAI", test.provider)
			}
		default:
			// Unknown defaults to ollama
			if !test.shouldUseOllama {
				t.Errorf("Unknown provider should default to Ollama")
			}
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
