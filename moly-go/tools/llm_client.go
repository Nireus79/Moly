package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// LLMClient - Claude API wrapper for agent reasoning
type LLMClient struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	timeout     time.Duration
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
	apiKey := os.Getenv("CLAUDE_API_KEY")
	if apiKey == "" {
		return nil, errors.New("CLAUDE_API_KEY environment variable not set")
	}

	model := os.Getenv("AGENT_MODEL")
	if model == "" {
		model = "claude-opus-5"
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

	return &LLMClient{
		apiKey:      apiKey,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		timeout:     timeout,
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

// callClaude - Internal method to call Claude API
func (c *LLMClient) callClaude(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.maxTokens
	}

	temperature := req.Temperature
	if temperature == 0 && req.Temperature != 0 {
		temperature = temperature
	}

	resp := &LLMResponse{
		Model: c.model,
	}

	// TODO: Implement actual Anthropic SDK call
	// This is a stub that demonstrates the expected behavior
	// Production implementation would use anthropic-sdk-go

	if req.SystemPrompt == "" {
		return nil, errors.New("system prompt cannot be empty")
	}

	if req.UserPrompt == "" {
		return nil, errors.New("user prompt cannot be empty")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return resp, nil
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
