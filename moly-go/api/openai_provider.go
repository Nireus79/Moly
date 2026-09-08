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

// OpenAIProvider implements LLMProvider for OpenAI's GPT models
type OpenAIProvider struct {
	apiKey string
	model  string
	client *http.Client
	config LLMConfig
}

// OpenAIRequest is the request structure for OpenAI API
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float32         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

// OpenAIMessage is a message in the OpenAI request
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse is the response structure for OpenAI API
type OpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config LLMConfig) (LLMProvider, error) {
	if config.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	model := config.DefaultModel
	if model == "" {
		model = "gpt-4-turbo" // Latest stable model
	}

	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	provider := &OpenAIProvider{
		apiKey: config.OpenAIAPIKey,
		model:  model,
		client: client,
		config: config,
	}

	log.Printf("[OpenAIProvider] Initialized with model: %s\n", model)
	return provider, nil
}

// GenerateCompletion generates a completion using OpenAI API
func (op *OpenAIProvider) GenerateCompletion(
	ctx context.Context,
	prompt string,
	systemPrompt string,
) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Add timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(op.config.Timeout)*time.Second)
	defer cancel()

	// Build messages
	messages := []OpenAIMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Prepare request
	reqBody := OpenAIRequest{
		Model:       op.model,
		Messages:    messages,
		Temperature: op.config.Temperature,
		MaxTokens:   op.config.MaxTokens,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://api.openai.com/v1/chat/completions",
		bytes.NewReader(data),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", op.apiKey))

	resp, err := op.client.Do(req)
	if err != nil {
		log.Printf("[OpenAIProvider] API error: %v\n", err)
		return "", fmt.Errorf("openai API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai error: status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openaiResp OpenAIResponse
	err = json.NewDecoder(resp.Body).Decode(&openaiResp)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("empty response from openai")
	}

	return openaiResp.Choices[0].Message.Content, nil
}

// GetModel returns the model name
func (op *OpenAIProvider) GetModel() string {
	return op.model
}

// GetProvider returns the provider name
func (op *OpenAIProvider) GetProvider() string {
	return "openai"
}

// IsHealthy checks if the provider is healthy by making a test request
func (op *OpenAIProvider) IsHealthy(ctx context.Context) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Quick health check
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	messages := []OpenAIMessage{
		{
			Role:    "user",
			Content: "Say 'ok'",
		},
	}

	reqBody := OpenAIRequest{
		Model:     op.model,
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
		"https://api.openai.com/v1/chat/completions",
		bytes.NewReader(data),
	)
	if err != nil {
		return false, fmt.Errorf("failed to create health check request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", op.apiKey))

	resp, err := op.client.Do(req)
	if err != nil {
		log.Printf("[OpenAIProvider] Health check failed: %v\n", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Println("[OpenAIProvider] Health check passed")
		return true, nil
	}

	return false, fmt.Errorf("openai health check failed with status %d", resp.StatusCode)
}
