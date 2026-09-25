package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// LLMClient - LLM wrapper supporting Claude, OpenAI, and Ollama
type LLMClient struct {
	Provider       string // "claude", "openai", or "ollama"
	ApiKey         string
	Model          string
	MaxTokens      int
	Temperature    float64
	Timeout        time.Duration
	OllamaEndpoint string // For Ollama provider
}

// LLMRequest - Request to Claude API
type LLMRequest struct {
	SystemPrompt        string
	UserPrompt          string
	Temperature         float64
	MaxTokens           int
	UseExtendedThinking bool
	Retries             int
}

// LLMResponse - Response from Claude API
type LLMResponse struct {
	Content          string
	StopReason       string
	TokensUsed       int
	CostUSD          float64
	ThinkingContent  string
	ProcessingTimeMs int64
	Model            string
}

// LLMProvider - Interface for LLM clients (both real and mock)
type LLMProvider interface {
	Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
}

// NewLLMClient - Create new LLM client with LOCAL-FIRST provider priority
// Privacy first: local models > cloud (optional via settings)
func NewLLMClient() (*LLMClient, error) {
	provider := os.Getenv("LLM_PROVIDER")
	var model string
	var apiKey string
	var ollamaEndpoint string

	// PRIVACY FIRST: Check for local Ollama
	ollamaEndpoint = os.Getenv("OLLAMA_ENDPOINT")
	if ollamaEndpoint == "" {
		ollamaEndpoint = "http://127.0.0.1:11434"
	}

	// Try to detect local Ollama first
	if isOllamaAvailable(ollamaEndpoint) {
		provider = "ollama"
		model = os.Getenv("AGENT_MODEL")
		if model == "" {
			model = "mistral" // Default Ollama model
		}
		log.Printf("[LLMClient] Local Ollama detected at %s, using model: %s", ollamaEndpoint, model)
	} else if provider == "" {
		// If no explicit provider set and Ollama not available, check for cloud API key
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("CLAUDE_API_KEY")
		}

		if apiKey != "" {
			provider = "claude"
			model = os.Getenv("AGENT_MODEL")
			if model == "" {
				model = "claude-opus-5"
			}
			log.Printf("[LLMClient] Using Claude (API key provided)")
		} else {
			// No local model, no API key -> FAIL
			provider = "none"
		}
	}

	// Apply environment overrides
	if explicitProvider := os.Getenv("LLM_PROVIDER"); explicitProvider != "" {
		provider = explicitProvider
	}

	// Get model if not already set
	if model == "" {
		model = os.Getenv("AGENT_MODEL")
		if model == "" {
			switch provider {
			case "ollama":
				model = "mistral"
			case "openai":
				model = "gpt-4-turbo"
			case "claude":
				model = "claude-opus-5"
			}
		}
	}

	// Get API key if not already set
	if apiKey == "" && (provider == "claude" || provider == "openai") {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("CLAUDE_API_KEY")
		}
		if apiKey == "" && provider == "openai" {
			apiKey = os.Getenv("OPENAI_API_KEY")
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

	// Default 900 seconds (15 minutes) for slow systems
	// CRITICAL: Must be VERY HIGH to support resource-constrained systems
	// Better to wait long than to break on slow LLM responses
	// Can be overridden with AGENT_TIMEOUT_SECONDS env var
	timeout := 900 * time.Second // 15 minutes minimum
	if t := os.Getenv("AGENT_TIMEOUT_SECONDS"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil {
			timeout = time.Duration(parsed) * time.Second
			// Enforce minimum 10 minute timeout for safety
			if timeout < 600*time.Second {
				log.Printf("[LLMClient] WARNING: Configured timeout %v is below 10-minute minimum, enforcing minimum", timeout)
				timeout = 600 * time.Second
			}
		}
	}

	// Fail fast if no LLM provider is available
	if provider == "none" {
		return nil, fmt.Errorf(
			"[LLMClient] FATAL: No LLM provider configured.\n\n" +
			"Moly requires an LLM provider to function. Please configure one of:\n\n" +
			"  1. LOCAL (Recommended - Privacy First):\n" +
			"     Install Ollama from https://ollama.ai\n" +
			"     Run: ollama pull mistral (or your preferred model)\n" +
			"     Moly will auto-detect it at http://127.0.0.1:11434\n\n" +
			"  2. CLAUDE (Anthropic):\n" +
			"     Set environment variable: ANTHROPIC_API_KEY=your-key\n" +
			"     or: CLAUDE_API_KEY=your-key\n\n" +
			"  3. OPENAI (OpenAI):\n" +
			"     Set environment variable: OPENAI_API_KEY=your-key\n\n" +
			"Moly cannot run with degraded functionality. It must either work with a provider or fail cleanly.",
		)
	}

	return &LLMClient{
		Provider:       provider,
		ApiKey:         apiKey,
		Model:          model,
		MaxTokens:      maxTokens,
		Temperature:    temperature,
		Timeout:        timeout,
		OllamaEndpoint: ollamaEndpoint,
	}, nil
}

// isOllamaAvailable - Check if local Ollama is running
func isOllamaAvailable(endpoint string) bool {
	req, err := http.NewRequest("GET", endpoint+"/api/tags", nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", "Moly/1.0")
	client := &http.Client{Timeout: 5 * time.Minute} // Increased for resource-constrained systems
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// Call - Make API call to LLM with prompt
func (c *LLMClient) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	if req.Retries == 0 {
		req.Retries = 1
	}

	log.Printf("[LLMClient] Calling %s with prompt length=%d, retries=%d", c.Provider, len(req.UserPrompt), req.Retries)

	// If no provider available, return error
	if c.Provider == "none" {
		return nil, errors.New("no LLM provider configured - no local Ollama and no cloud API key")
	}

	var lastErr error
	for attempt := 0; attempt < req.Retries; attempt++ {
		log.Printf("[LLMClient] Attempt %d/%d", attempt+1, req.Retries)

		var resp *LLMResponse
		var err error

		// Route to appropriate provider
		switch c.Provider {
		case "ollama":
			resp, err = c.callOllama(ctx, req)
		case "openai":
			resp, err = c.callOpenAI(ctx, req)
		case "claude", "":
			resp, err = c.callClaude(ctx, req)
		default:
			err = fmt.Errorf("unknown provider: %s", c.Provider)
		}

		if err == nil {
			log.Printf("[LLMClient] Success with %s (tokens=%d, time=%dms)", c.Provider, resp.TokensUsed, resp.ProcessingTimeMs)
			return resp, nil
		}

		lastErr = err
		log.Printf("[LLMClient] Attempt %d failed: %v", attempt+1, err)

		if attempt < req.Retries-1 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			log.Printf("[LLMClient] Waiting %v before retry...", backoff)
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				log.Printf("[LLMClient] Context cancelled")
				return nil, ctx.Err()
			}
		}
	}

	log.Printf("[LLMClient] Failed after %d attempts: %v", req.Retries, lastErr)
	return nil, fmt.Errorf("failed after %d attempts: %w", req.Retries, lastErr)
}

// callClaude - Internal method to call LLM (Claude, OpenAI, or Ollama) via HTTP
func (c *LLMClient) callClaude(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	if req.UserPrompt == "" {
		return nil, errors.New("user prompt cannot be empty")
	}

	// Route to appropriate provider
	switch c.Provider {
	case "ollama":
		return c.callOllama(ctx, req)
	case "openai":
		return c.callOpenAI(ctx, req)
	default:
		return c.callClaudeAPI(ctx, req)
	}
}

// callClaudeAPI - Call Anthropic Claude API
func (c *LLMClient) callClaudeAPI(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if req.SystemPrompt == "" {
		return nil, errors.New("system prompt cannot be empty")
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.MaxTokens
	}

	temperature := req.Temperature
	if temperature == 0 && req.Temperature == 0 {
		temperature = c.Temperature
	}

	log.Printf("[Claude] Calling with model=%s, tokens=%d, temp=%.1f", c.Model, maxTokens, temperature)

	// Build request body for Anthropic API
	body := map[string]interface{}{
		"model":       c.Model,
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
		log.Printf("[Claude] Request creation failed: %v", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.ApiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// HTTP timeout should be longer than context timeout to avoid premature cancellation
	httpTimeout := c.Timeout + (5 * time.Minute)
	client := &http.Client{Timeout: httpTimeout}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("[Claude] API request failed: %v", err)
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		log.Printf("[Claude] API error %d: %s", httpResp.StatusCode, string(body))
		return nil, fmt.Errorf("API error %d: %s", httpResp.StatusCode, string(body))
	}

	log.Printf("[Claude] API call successful, parsing response...")

	// Parse response
	type claudeResponse struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
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
		Model:            c.Model,
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
		temperature = c.Temperature
	}

	log.Printf("[Ollama] Calling %s at %s with temp=%.1f", c.Model, c.OllamaEndpoint, temperature)

	// Build Ollama request
	ollamaReq := map[string]interface{}{
		"model":       c.Model,
		"prompt":      prompt,
		"stream":      false,
		"temperature": temperature,
	}

	bodyJSON, _ := json.Marshal(ollamaReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.OllamaEndpoint+"/api/generate",
		bytes.NewReader(bodyJSON))
	if err != nil {
		log.Printf("[Ollama] Request creation failed: %v", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// HTTP timeout should be longer than context timeout
	httpTimeout := c.Timeout + (5 * time.Minute)
	client := &http.Client{Timeout: httpTimeout}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("[Ollama] Request failed: %v", err)
		return nil, fmt.Errorf("Ollama request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		log.Printf("[Ollama] API error %d: %s", httpResp.StatusCode, string(body))
		return nil, fmt.Errorf("Ollama error %d: %s", httpResp.StatusCode, string(body))
	}

	log.Printf("[Ollama] API call successful, parsing response...")

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
		Model:            c.Model,
		ProcessingTimeMs: int64(time.Since(time.Now()).Milliseconds()),
	}, nil
}

// callOpenAI - Call OpenAI API
func (c *LLMClient) callOpenAI(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if req.SystemPrompt == "" {
		return nil, errors.New("system prompt cannot be empty")
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.MaxTokens
	}

	temperature := req.Temperature
	if temperature == 0 && req.Temperature == 0 {
		temperature = c.Temperature
	}

	log.Printf("[OpenAI] Calling with model=%s, tokens=%d, temp=%.1f", c.Model, maxTokens, temperature)

	// Build request body for OpenAI API
	body := map[string]interface{}{
		"model":       c.Model,
		"max_tokens":  maxTokens,
		"temperature": temperature,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": req.SystemPrompt,
			},
			{
				"role":    "user",
				"content": req.UserPrompt,
			},
		},
	}

	bodyJSON, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.openai.com/v1/chat/completions",
		bytes.NewReader(bodyJSON))
	if err != nil {
		log.Printf("[OpenAI] Request creation failed: %v", err)
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.ApiKey))

	// HTTP timeout should be longer than context timeout
	httpTimeout := c.Timeout + (5 * time.Minute)
	client := &http.Client{Timeout: httpTimeout}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("[OpenAI] Request failed: %v", err)
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		log.Printf("[OpenAI] API error %d: %s", httpResp.StatusCode, string(body))
		return nil, fmt.Errorf("OpenAI error %d: %s", httpResp.StatusCode, string(body))
	}

	log.Printf("[OpenAI] API call successful, parsing response...")

	// Parse response
	type openaiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}

	var apiResp openaiResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, errors.New("empty response from OpenAI")
	}

	return &LLMResponse{
		Content:          apiResp.Choices[0].Message.Content,
		TokensUsed:       apiResp.Usage.PromptTokens + apiResp.Usage.CompletionTokens,
		Model:            c.Model,
		ProcessingTimeMs: int64(time.Since(time.Now()).Milliseconds()),
	}, nil
}

// GenerateSuggestions - Helper: generate multiple suggestions
func (c *LLMClient) GenerateSuggestions(ctx context.Context, prompt string, count int) ([]string, error) {
	if count <= 0 || count > 10 {
		count = 3
	}

	req := &LLMRequest{
		SystemPrompt:        "You are a helpful assistant generating communication suggestions.",
		UserPrompt:          fmt.Sprintf("%s Generate exactly %d suggestions.", prompt, count),
		MaxTokens:           1000,
		Temperature:         0.7,
		UseExtendedThinking: false,
		Retries:             2,
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
		SystemPrompt:        "You are a safety expert analyzing messages for concerning patterns.",
		UserPrompt:          fmt.Sprintf("Analyze for risk patterns: %s", message),
		MaxTokens:           500,
		Temperature:         0.3,
		UseExtendedThinking: true,
		Retries:             2,
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
		SystemPrompt:        "You are an expert at extracting user insights from conversations.",
		UserPrompt:          fmt.Sprintf("Extract key insights from: %s", message),
		MaxTokens:           800,
		Temperature:         0.5,
		UseExtendedThinking: true,
		Retries:             2,
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
