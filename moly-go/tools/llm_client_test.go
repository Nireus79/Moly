package tools

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewLLMClient(t *testing.T) {
	tests := []struct {
		name      string
		setupEnv  func()
		cleanupEnv func()
		wantErr   bool
	}{
		{
			name: "with valid API key",
			setupEnv: func() {
				os.Setenv("CLAUDE_API_KEY", "test-key-123")
			},
			cleanupEnv: func() {
				os.Unsetenv("CLAUDE_API_KEY")
			},
			wantErr: false,
		},
		{
			name: "without API key",
			setupEnv: func() {
				os.Unsetenv("CLAUDE_API_KEY")
			},
			cleanupEnv: func() {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			_, err := NewLLMClient()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLLMClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLLMClientConfiguration(t *testing.T) {
	os.Setenv("CLAUDE_API_KEY", "test-key")
	os.Setenv("AGENT_MODEL", "claude-sonnet-5")
	os.Setenv("AGENT_MAX_TOKENS", "5000")
	os.Setenv("AGENT_TEMPERATURE", "0.5")
	defer func() {
		os.Unsetenv("CLAUDE_API_KEY")
		os.Unsetenv("AGENT_MODEL")
		os.Unsetenv("AGENT_MAX_TOKENS")
		os.Unsetenv("AGENT_TEMPERATURE")
	}()

	client, err := NewLLMClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.model != "claude-sonnet-5" {
		t.Errorf("model = %s, want claude-sonnet-5", client.model)
	}

	if client.maxTokens != 5000 {
		t.Errorf("maxTokens = %d, want 5000", client.maxTokens)
	}

	if client.temperature != 0.5 {
		t.Errorf("temperature = %f, want 0.5", client.temperature)
	}
}

func TestLLMClientCall(t *testing.T) {
	os.Setenv("CLAUDE_API_KEY", "test-key")
	defer os.Unsetenv("CLAUDE_API_KEY")

	client, err := NewLLMClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name    string
		request *LLMRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &LLMRequest{
				SystemPrompt: "You are helpful",
				UserPrompt:   "Hello",
				MaxTokens:    100,
				Retries:      1,
			},
			wantErr: false,
		},
		{
			name: "missing system prompt",
			request: &LLMRequest{
				UserPrompt: "Hello",
				Retries:    1,
			},
			wantErr: true,
		},
		{
			name: "missing user prompt",
			request: &LLMRequest{
				SystemPrompt: "You are helpful",
				Retries:      1,
			},
			wantErr: true,
		},
		{
			name:    "nil request",
			request: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err := client.Call(ctx, tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("Call() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLLMClientGenerateSuggestions(t *testing.T) {
	os.Setenv("CLAUDE_API_KEY", "test-key")
	defer os.Unsetenv("CLAUDE_API_KEY")

	client, err := NewLLMClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	suggestions, err := client.GenerateSuggestions(ctx, "Hello, how are you?", 3)
	if err != nil {
		t.Fatalf("GenerateSuggestions() error = %v", err)
	}

	if suggestions == nil {
		t.Error("Expected suggestions, got nil")
	}
}

func TestLLMClientContextTimeout(t *testing.T) {
	os.Setenv("CLAUDE_API_KEY", "test-key")
	defer os.Unsetenv("CLAUDE_API_KEY")

	client, err := NewLLMClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond)

	_, err = client.Call(ctx, &LLMRequest{
		SystemPrompt: "Test",
		UserPrompt:   "Test",
		Retries:      1,
	})

	if err == nil {
		t.Error("Expected error for deadline exceeded")
	}

	errStr := err.Error()
	hasDeadlineExceeded := errStr == "context deadline exceeded" || strings.Contains(errStr, "deadline exceeded")
	if err != context.DeadlineExceeded && !hasDeadlineExceeded {
		t.Errorf("Expected deadline exceeded error, got %v", err)
	}
}
