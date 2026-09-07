package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

// LLMClient - LLM wrapper supporting Claude and Ollama
type LLMClient struct {
	provider        string // "claude" or "ollama"
	apiKey          string
	model           string
	maxTokens       int
	temperature     float64
	timeout         time.Duration
	ollamaEndpoint  string // For Ollama provider
}

// LLMRequest - Request to Claude API
type LLMRequest struct {
	SystemPrompt      string
	UserPrompt        string
	Temperature       float64
	MaxTokens         int
	UseExtendedThinking bool
	Retries           int
}

// LLMResponse - Response from Claude API
type LLMResponse struct {
	Content           string
	StopReason        string
	TokensUsed        int
	CostUSD           float64
	ThinkingContent   string
	ProcessingTimeMs  int64
	Model             string
}

// NewLLMClient - Create new LLM client with environment configuration
func NewLLMClient() (*LLMClient, error) {
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		provider = "claude" // Default to Claude
	}

	model := os.Getenv("AGENT_MODEL")
	if model == "" {
		if provider == "ollama" {
			model = "mistral" // Default Ollama model
		} else {
			model = "claude-opus-5" // Default Claude model
		}
	}

	maxTokens := 2000
	if mt := os.Getenv("AGENT_MAX_TOKENS"); mt != "" {
		if parsed, err := strconv.Atoi(mt); err == nil {
			maxTokens = parsed
		}
	}

	temperature := 0.7
	if temp := os.Getenv("AGENT_TEMPERATURE"); temp != "" {
		if parsed, err := strconv.ParseFloat(temp, 64); err == nil {
			temperature = parsed
		}
	}

	timeout := 30 * time.Second
	if t := os.Getenv("AGENT_TIMEOUT_SECONDS"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil {
			timeout = time.Duration(parsed) * time.Second
		}
	}

	ollamaEndpoint := os.Getenv("OLLAMA_ENDPOINT")
	if ollamaEndpoint == "" {
		ollamaEndpoint = "http://127.0.0.1:11434"
	}

	apiKey := os.Getenv("CLAUDE_API_KEY")

	return &LLMClient{
		provider:       provider,
		apiKey:         apiKey,
		model:          model,
		maxTokens:      maxTokens,
		temperature:    temperature,
		timeout:        timeout,
		ollamaEndpoint: ollamaEndpoint,
	}, nil
}

// Call - Make API call to Claude with prompt
func (c *LLMClient) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	if req.Retries == 0 {
		req.Retries = 1
	}

	var lastErr error
	for attempt := 0; attempt < req.Retries; attempt++ {
		resp, err := c.callClaude(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		if attempt < req.Retries-1 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", req.Retries, lastErr)
}

// callClaude - Internal method to call LLM (Claude or Ollama) via HTTP
func (c *LLMClient) callClaude(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if req.UserPrompt == "" {
		return nil, errors.New("user prompt cannot be empty")
	}

	// Route to appropriate provider
	if c.provider == "ollama" {
		return c.callOllama(ctx, req)
	}
	return c.callClaudeAPI(ctx, req)
}

// callClaudeAPI - Call Anthropic Claude API
func (c *LLMClient) callClaudeAPI(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if req.SystemPrompt == "" {
		return nil, errors.New("system prompt cannot be empty")
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.maxTokens
	}

	temperature := req.Temperature
	if temperature == 0 && req.Temperature == 0 {
		temperature = c.temperature
	}

	// Build request body for Anthropic API
	body := map[string]interface{}{
		"model":       c.model,
		"max_tokens":  maxTokens,
		"temperature": temperature,
		"system":      req.SystemPrompt,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": req.UserPrompt,
			},
		},
	}

	bodyJSON, _ := json.Marshal(body)

	// Make HTTP request to Anthropic API
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.anthropic.com/v1/messages",
		bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: c.timeout}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	// Parse response
	type claudeResponse struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	var apiResp claudeResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, errors.New("empty response from API")
	}

	return &LLMResponse{
		Content:          apiResp.Content[0].Text,
		StopReason:       apiResp.StopReason,
		TokensUsed:       apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		Model:            c.model,
		ProcessingTimeMs: int64(time.Since(time.Now()).Milliseconds()),
	}, nil
}

// callOllama - Call Ollama API
func (c *LLMClient) callOllama(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	// Combine system prompt and user prompt for Ollama
	prompt := req.UserPrompt
	if req.SystemPrompt != "" {
		prompt = req.SystemPrompt + "\n\n" + req.UserPrompt
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.temperature
	}

	// Build Ollama request
	ollamaReq := map[string]interface{}{
		"model":       c.model,
		"prompt":      prompt,
		"stream":      false,
		"temperature": temperature,
	}

	bodyJSON, _ := json.Marshal(ollamaReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.ollamaEndpoint+"/api/generate",
		bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: c.timeout}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Ollama request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("Ollama error %d: %s", httpResp.StatusCode, string(body))
	}

	type ollamaResponse struct {
		Response string `json:"response"`
		Model    string `json:"model"`
	}

	var apiResp ollamaResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	return &LLMResponse{
		Content:          apiResp.Response,
		Model:            c.model,
		ProcessingTimeMs: int64(time.Since(time.Now()).Milliseconds()),
	}, nil
}

// GenerateSuggestions - Helper: generate multiple suggestions
func (c *LLMClient) GenerateSuggestions(ctx context.Context, prompt string, count int) ([]string, error) {
	if count <= 0 || count > 10 {
		count = 3
	}

	req := &LLMRequest{
		SystemPrompt:     "You are a helpful assistant generating communication suggestions.",
		UserPrompt:       fmt.Sprintf("%s Generate exactly %d suggestions.", prompt, count),
		MaxTokens:        1000,
		Temperature:      0.7,
		UseExtendedThinking: false,
		Retries:          2,
	}

	resp, err := c.Call(ctx, req)
	if err != nil {
		return nil, err
	}

	return []string{resp.Content}, nil
}

// AnalyzeRisk - Helper: analyze message for risk patterns
func (c *LLMClient) AnalyzeRisk(ctx context.Context, message string) (string, error) {
	req := &LLMRequest{
		SystemPrompt:     "You are a safety expert analyzing messages for concerning patterns.",
		UserPrompt:       fmt.Sprintf("Analyze for risk patterns: %s", message),
		MaxTokens:        500,
		Temperature:      0.3,
		UseExtendedThinking: true,
		Retries:          2,
	}

	resp, err := c.Call(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// ExtractContext - Helper: extract insights from message
func (c *LLMClient) ExtractContext(ctx context.Context, message string) (string, error) {
	req := &LLMRequest{
		SystemPrompt:     "You are an expert at extracting user insights from conversations.",
		UserPrompt:       fmt.Sprintf("Extract key insights from: %s", message),
		MaxTokens:        800,
		Temperature:      0.5,
		UseExtendedThinking: true,
		Retries:          2,
	}

	resp, err := c.Call(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// HealthCheck - Verify API connection and credentials
func (c *LLMClient) HealthCheck(ctx context.Context) error {
	req := &LLMRequest{
		SystemPrompt: "You are a helpful assistant.",
		UserPrompt:   "Respond with 'ok' if you can read this.",
		MaxTokens:    10,
		Temperature:  0.3,
		Retries:      1,
	}

	_, err := c.Call(ctx, req)
	return err
}
