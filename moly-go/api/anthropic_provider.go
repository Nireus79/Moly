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

// AnthropicProvider implements LLMProvider for Anthropic's Claude
type AnthropicProvider struct {
	apiKey string
	model  string
	client *http.Client
	config LLMConfig
}

// AnthropicRequest is the request structure for Anthropic API
type AnthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []AnthropicMessage `json:"messages"`
	System      string             `json:"system"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float32            `json:"temperature"`
}

// AnthropicMessage is a message in the Anthropic request
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse is the response structure for Anthropic API
type AnthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string  `json:"model"`
	StopReason   string  `json:"stop_reason"`
	StopSequence *string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(config LLMConfig) (LLMProvider, error) {
	if config.AnthropicAPIKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	model := config.DefaultModel
	if model == "" {
		model = "claude-3-5-sonnet-20241022" // Latest model
	}

	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	provider := &AnthropicProvider{
		apiKey: config.AnthropicAPIKey,
		model:  model,
		client: client,
		config: config,
	}

	log.Printf("[AnthropicProvider] Initialized with model: %s\n", model)
	return provider, nil
}

// GenerateCompletion generates a completion using Claude API
func (ap *AnthropicProvider) GenerateCompletion(
	ctx context.Context,
	prompt string,
	systemPrompt string,
) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Add timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(ap.config.Timeout)*time.Second)
	defer cancel()

	// Build messages
	messages := []AnthropicMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Prepare request
	reqBody := AnthropicRequest{
		Model:       ap.model,
		Messages:    messages,
		System:      systemPrompt,
		MaxTokens:   ap.config.MaxTokens,
		Temperature: ap.config.Temperature,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://api.anthropic.com/v1/messages",
		bytes.NewReader(data),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", ap.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := ap.client.Do(req)
	if err != nil {
		log.Printf("[AnthropicProvider] API error: %v\n", err)
		return "", fmt.Errorf("anthropic API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("anthropic error: status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var anthropicResp AnthropicResponse
	err = json.NewDecoder(resp.Body).Decode(&anthropicResp)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("empty response from anthropic")
	}

	return anthropicResp.Content[0].Text, nil
}

// GetModel returns the model name
func (ap *AnthropicProvider) GetModel() string {
	return ap.model
}

// GetProvider returns the provider name
func (ap *AnthropicProvider) GetProvider() string {
	return "anthropic"
}

// IsHealthy checks if the provider is healthy by making a test request
func (ap *AnthropicProvider) IsHealthy(ctx context.Context) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Quick health check
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	messages := []AnthropicMessage{
		{
			Role:    "user",
			Content: "Say 'ok'",
		},
	}

	reqBody := AnthropicRequest{
		Model:     ap.model,
		Messages:  messages,
		MaxTokens: 100,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal health check: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://api.anthropic.com/v1/messages",
		bytes.NewReader(data),
	)
	if err != nil {
		return false, fmt.Errorf("failed to create health check request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", ap.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := ap.client.Do(req)
	if err != nil {
		log.Printf("[AnthropicProvider] Health check failed: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Println("[AnthropicProvider] Health check passed")
		return true, nil
	}

	return false, fmt.Errorf("anthropic health check failed with status %d", resp.StatusCode)
}
