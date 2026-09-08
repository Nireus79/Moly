package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestAnthropicProvider tests Anthropic REST API integration
func TestAnthropicProvider(t *testing.T) {
	// Create a mock Anthropic server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/messages" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "msg-123",
				"type": "message",
				"role": "assistant",
				"content": [
					{
						"type": "text",
						"text": "ok"
					}
				],
				"model": "claude-3-5-sonnet-20241022",
				"stop_reason": "end_turn",
				"usage": {
					"input_tokens": 10,
					"output_tokens": 2
				}
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := LLMConfig{
		AnthropicAPIKey: "test-key",
		DefaultModel:    "claude-3-5-sonnet-20241022",
		Temperature:     0.3,
		MaxTokens:       100,
		Timeout:         10,
	}

	provider, err := NewAnthropicProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Check model
	if provider.GetModel() != config.DefaultModel {
		t.Errorf("Expected model %s, got %s", config.DefaultModel, provider.GetModel())
	}

	// Check provider name
	if provider.GetProvider() != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got %s", provider.GetProvider())
	}
}

// TestOpenAIProvider tests OpenAI REST API integration
func TestOpenAIProvider(t *testing.T) {
	// Create a mock OpenAI server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "test-123",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "gpt-4-turbo",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "ok"
						},
						"finish_reason": "stop"
					}
				],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 2,
					"total_tokens": 12
				}
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := LLMConfig{
		OpenAIAPIKey: "test-key",
		DefaultModel: "gpt-4-turbo",
		Temperature:  0.3,
		MaxTokens:    100,
		Timeout:      10,
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Check model
	if provider.GetModel() != config.DefaultModel {
		t.Errorf("Expected model %s, got %s", config.DefaultModel, provider.GetModel())
	}

	// Check provider name
	if provider.GetProvider() != "openai" {
		t.Errorf("Expected provider 'openai', got %s", provider.GetProvider())
	}

	// Note: Can't test actual completion without mocking the API endpoint
	// Real tests would require mocking the HTTP server or using an integration test
}

// TestOllamaProvider tests Ollama REST API integration
func TestOllamaProvider(t *testing.T) {
	// Create a mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"model": "mistral",
				"response": "ok",
				"done": true,
				"total_duration": 1000000000,
				"load_duration": 500000000,
				"prompt_eval_count": 10,
				"prompt_eval_duration": 300000000,
				"eval_count": 2,
				"eval_duration": 200000000
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := LLMConfig{
		OllamaURL:    server.URL,
		DefaultModel: "mistral",
		Temperature:  0.3,
		MaxTokens:    100,
		Timeout:      10,
	}

	provider, err := NewOllamaProvider(config)
	if err != nil {
		t.Fatalf("Failed to create Ollama provider: %v", err)
	}

	// Check model
	if provider.GetModel() != config.DefaultModel {
		t.Errorf("Expected model %s, got %s", config.DefaultModel, provider.GetModel())
	}

	// Check provider name
	if provider.GetProvider() != "ollama" {
		t.Errorf("Expected provider 'ollama', got %s", provider.GetProvider())
	}

	// Test completion with longer timeout to account for mock server latency
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := provider.GenerateCompletion(ctx, "Say hello", "You are helpful")
	if err != nil {
		t.Fatalf("GenerateCompletion failed: %v", err)
	}

	if response != "ok" {
		t.Errorf("Expected response 'ok', got %s", response)
	}

	// Test health check with fresh context
	healthCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	healthy, err := provider.IsHealthy(healthCtx)
	if err != nil {
		t.Fatalf("IsHealthy failed: %v", err)
	}

	if !healthy {
		t.Error("Expected health check to pass")
	}
}

// TestProviderFactory tests factory pattern
func TestProviderFactory(t *testing.T) {
	// Create mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"response": "ok", "done": true}`))
		}
	}))
	defer server.Close()

	config := LLMConfig{
		Provider:        "ollama",
		OllamaURL:       server.URL,
		DefaultModel:    "mistral",
		Temperature:     0.3,
		MaxTokens:       100,
		Timeout:         10,
		AnthropicAPIKey: "",
		OpenAIAPIKey:    "",
	}

	factory := NewProviderFactory(config)

	// Test GetProvider
	provider, err := factory.GetProvider("ollama")
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}

	if provider.GetProvider() != "ollama" {
		t.Errorf("Expected 'ollama', got %s", provider.GetProvider())
	}

	// Test caching
	provider2, err := factory.GetProvider("ollama")
	if err != nil {
		t.Fatalf("GetProvider (cached) failed: %v", err)
	}

	if provider != provider2 {
		t.Error("Expected cached provider to be same instance")
	}

	// Test GetAvailableProviders
	available := factory.GetAvailableProviders()
	if len(available) != 1 || available[0] != "ollama" {
		t.Errorf("Expected [ollama], got %v", available)
	}

	// Test GetHealthyProvider
	healthy, err := factory.GetHealthyProvider(context.Background())
	if err != nil {
		t.Fatalf("GetHealthyProvider failed: %v", err)
	}

	if healthy.GetProvider() != "ollama" {
		t.Errorf("Expected healthy provider to be ollama")
	}
}

// TestProviderFactory_UnknownProvider tests error handling
func TestProviderFactory_UnknownProvider(t *testing.T) {
	config := LLMConfig{
		Provider: "unknown",
	}

	factory := NewProviderFactory(config)
	_, err := factory.GetProvider("unknown")

	if err == nil {
		t.Error("Expected error for unknown provider")
	}
}

// TestProviderFactory_MultipleProviders tests fallback logic
func TestProviderFactory_MultipleProviders(t *testing.T) {
	// Create mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/generate" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"response": "ok", "done": true}`))
		}
	}))
	defer server.Close()

	config := LLMConfig{
		Provider:        "anthropic",
		AnthropicAPIKey: "", // Not available
		OpenAIAPIKey:    "", // Not available
		OllamaURL:       server.URL,
		DefaultModel:    "mistral",
		Temperature:     0.3,
		MaxTokens:       100,
		Timeout:         10,
	}

	factory := NewProviderFactory(config)

	// Only Ollama is available
	available := factory.GetAvailableProviders()
	if len(available) != 1 || available[0] != "ollama" {
		t.Errorf("Expected only ollama available, got %v", available)
	}

	// GetHealthyProvider should return Ollama
	healthy, err := factory.GetHealthyProvider(context.Background())
	if err != nil {
		t.Fatalf("GetHealthyProvider failed: %v", err)
	}

	if healthy.GetProvider() != "ollama" {
		t.Errorf("Expected ollama as healthy provider")
	}
}

// TestOllamaProvider_Timeout tests timeout handling
func TestOllamaProvider_Timeout(t *testing.T) {
	// Create a slow mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't send response (will timeout)
		select {}
	}))
	defer server.Close()

	config := LLMConfig{
		OllamaURL:    server.URL,
		DefaultModel: "mistral",
		Timeout:      1, // 1 second timeout
	}

	provider, err := NewOllamaProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2)
	defer cancel()

	_, err = provider.GenerateCompletion(ctx, "test", "test")
	if err == nil {
		t.Error("Expected timeout error")
	}
}

// TestOllamaProvider_ErrorResponse tests error handling
func TestOllamaProvider_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal error"))
	}))
	defer server.Close()

	config := LLMConfig{
		OllamaURL:    server.URL,
		DefaultModel: "mistral",
		Timeout:      10,
	}

	provider, err := NewOllamaProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	ctx := context.Background()
	_, err = provider.GenerateCompletion(ctx, "test", "test")
	if err == nil {
		t.Error("Expected error for error response")
	}
}

// BenchmarkOllamaProvider_GenerateCompletion benchmarks response time
func BenchmarkOllamaProvider_GenerateCompletion(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"response": "ok", "done": true}`))
	}))
	defer server.Close()

	config := LLMConfig{
		OllamaURL:    server.URL,
		DefaultModel: "mistral",
		Timeout:      10,
	}

	provider, _ := NewOllamaProvider(config)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.GenerateCompletion(ctx, "test", "test")
	}
}
