package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// OllamaProvider implements LLMProvider for local Ollama
type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
	config  LLMConfig
}

// OllamaRequest is the request structure for Ollama API
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse is the response structure for Ollama API
type OllamaResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	TotalDuration      int64  `json:"total_duration"`
	LoadDuration       int64  `json:"load_duration"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	PromptEvalDuration int64  `json:"prompt_eval_duration"`
	EvalCount          int    `json:"eval_count"`
	EvalDuration       int64  `json:"eval_duration"`
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(config LLMConfig) (LLMProvider, error) {
	if config.OllamaURL == "" {
		return nil, fmt.Errorf("OLLAMA_URL not set")
	}

	model := config.DefaultModel
	if model == "" {
		model = "mistral" // Good default for local
	}

	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	provider := &OllamaProvider{
		baseURL: config.OllamaURL,
		model:   model,
		client:  client,
		config:  config,
	}

	log.Printf("[OllamaProvider] Initialized with model: %s at %s\n", model, config.OllamaURL)
	return provider, nil
}

// GenerateCompletion generates a completion using Ollama
func (op *OllamaProvider) GenerateCompletion(
	ctx context.Context,
	prompt string,
	systemPrompt string,
) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Combine system and user prompts
	fullPrompt := fmt.Sprintf("%s\n\n%s", systemPrompt, prompt)

	// Prepare request
	reqBody := OllamaRequest{
		Model:  op.model,
		Prompt: fullPrompt,
		Stream: false,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/api/generate", op.baseURL),
		bytes.NewReader(data),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := op.client.Do(req)
	if err != nil {
		log.Printf("[OllamaProvider] API error: %v\n", err)
		return "", fmt.Errorf("ollama API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama error: %s", string(body))
	}

	// Parse response
	var ollamaResp OllamaResponse
	err = json.NewDecoder(resp.Body).Decode(&ollamaResp)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return ollamaResp.Response, nil
}

// GetModel returns the model name
func (op *OllamaProvider) GetModel() string {
	return op.model
}

// GetProvider returns the provider name
func (op *OllamaProvider) GetProvider() string {
	return "ollama"
}

// IsHealthy checks if Ollama is running and the model is available
func (op *OllamaProvider) IsHealthy(ctx context.Context) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Quick health check
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	reqBody := OllamaRequest{
		Model:  op.model,
		Prompt: "Say ok",
		Stream: false,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal health check request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/api/generate", op.baseURL),
		bytes.NewReader(data),
	)
	if err != nil {
		return false, fmt.Errorf("failed to create health check request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := op.client.Do(req)
	if err != nil {
		log.Printf("[OllamaProvider] Health check failed (connection): %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Println("[OllamaProvider] Health check passed")
		return true, nil
	}

	return false, fmt.Errorf("ollama health check failed with status %d", resp.StatusCode)
}
