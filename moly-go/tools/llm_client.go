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
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
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

	// Reply serves the calls that ask for RoleReply, when the reply model is another model than this one. Nil: this client serves them.
	// It has none of its own (a client in this field has no Reply).
	Reply *LLMClient
	// Roles are the models chosen for the two roles when Moly chose them (Claude and OpenAI); empty when the person or the
	// operator named a model, and for Ollama.
	Roles RoleModels

	noTemperature     atomic.Bool // set when the Claude or OpenAI API refused a temperature for this model
	noThinkOff        atomic.Bool // set when the Claude API refused thinking "disabled" for this model
	noReasoningEffort atomic.Bool // set when the OpenAI API refused a reasoning effort for this model
}

var claudeMessagesURL = "https://api.anthropic.com/v1/messages"

// claude5Model matches the Claude 5 family (claude-opus-5-5, claude-sonnet-5-5, claude-haiku-5-5, ...). The API refuses a
// "temperature" for these models ("`temperature` is deprecated for this model"), so none is sent.
var claude5Model = regexp.MustCompile(`^claude-[a-z]+-([5-9]|[1-9][0-9])(-|$)`)

func claudeTakesTemperature(model string) bool { return !claude5Model.MatchString(model) }

var openAIChatURL = "https://api.openai.com/v1/chat/completions"

// The vendors' addresses can be moved to a stand-in server for the browser tests (a container has no network and no key):
// MOLY_ANTHROPIC_BASE_URL and MOLY_OPENAI_BASE_URL, for example http://127.0.0.1:11999. Unset, the real addresses are used.
func init() {
	if base := strings.TrimRight(os.Getenv("MOLY_ANTHROPIC_BASE_URL"), "/"); base != "" {
		claudeModelsURL, claudeMessagesURL = base+"/v1/models", base+"/v1/messages"
	}
	if base := strings.TrimRight(os.Getenv("MOLY_OPENAI_BASE_URL"), "/"); base != "" {
		openAIModelsURL, openAIChatURL = base+"/v1/models", base+"/v1/chat/completions"
	}
}

// openAIReasoningModel matches the OpenAI models that think before they answer (o1, o3, o4-mini, gpt-5 and later). They fix
// the temperature at its default and count the thinking inside the output limit.
var openAIReasoningModel = regexp.MustCompile(`^(o[0-9]|gpt-([5-9]|[1-9][0-9]))(-|\.|$)`)

// reasoningHeadroom is added to the output limit of a reasoning model, so the thinking cannot use up the whole limit and
// leave the answer empty. The thinking counts toward the limit; the more effort, the more room (OpenAI advises reserving a
// large number of tokens for reasoning).
func reasoningHeadroom(effort string) int {
	switch effort {
	case "high":
		return 24000
	case "medium":
		return 12000
	}
	return 4000
}

// LLMRequest - Request to Claude API
type LLMRequest struct {
	SystemPrompt        string
	UserPrompt          string
	Temperature         float64
	MaxTokens           int
	UseExtendedThinking bool
	Retries             int
	Timeout             time.Duration // Optional: override default timeout for this request
	JSONMode            bool          // Ask the provider for syntactically valid JSON where it supports that (Ollama, OpenAI)
	// Effort asks a Claude 5 model to think before it answers, at this level ("low", "medium", "high"). Empty: no thinking, for
	// the short structured answers of judges and extraction. The reply that is read by the user, and the questions Moly asks,
	// need the thinking: Anthropic's guidance is that with thinking off or at low effort the model is more likely to write
	// reasoning into the reply, and that strict instruction following needs "high". Ignored by other providers and models.
	Effort string
	// Role says what the call is for: RoleFast (the default) for the small checks (judges, extraction, safety, summaries), RoleReply
	// for the text the user reads and the questions Moly writes, which need judgement (no added facts, one question). The client
	// of the user sends a RoleReply call to its reply model, if it has one (LLMClient.Reply); otherwise to itself.
	Role string
}

// The roles of a model call (LLMRequest.Role).
const (
	RoleFast  = "fast"
	RoleReply = "reply"
)

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

// NewLLMClientFromEnv creates the client that the OPERATOR chose with environment variables (LLM_PROVIDER, AGENT_MODEL,
// OLLAMA_ENDPOINT, ANTHROPIC_API_KEY, OPENAI_API_KEY): a developer override or a test run. It belongs to whoever started the
// program and is used for every user of that run. Nothing else reads those variables.
func NewLLMClientFromEnv() (*LLMClient, error) { return newLLMClient(LLMSettings{}, true) }

// NewLLMClient is NewLLMClientFromEnv (live tests and scripts).
func NewLLMClient() (*LLMClient, error) { return NewLLMClientFromEnv() }

// LLMSettings are the choices of ONE USER, saved in the encrypted database.
type LLMSettings struct {
	Provider       string // "ollama", "claude" or "openai"
	Model          string
	APIKey         string // the key of Provider
	OllamaEndpoint string
}

// NewLLMClientWith creates the client of a user from that user's settings and nothing else. The environment cannot change
// the provider, the model or the key here: an ANTHROPIC_API_KEY of the machine must never replace the key a user saved (or
// supply one the user did not). Only the tuning variables (AGENT_MAX_TOKENS, AGENT_TEMPERATURE, AGENT_TIMEOUT_SECONDS) apply.
func NewLLMClientWith(s LLMSettings) (*LLMClient, error) { return newLLMClient(s, false) }

// newLLMClient builds a client. fromEnv selects whose choices count: the operator's environment, or only s.
func newLLMClient(s LLMSettings, fromEnv bool) (*LLMClient, error) {
	env := func(name string) string {
		if fromEnv {
			return os.Getenv(name)
		}
		return ""
	}
	provider := strings.ToLower(strings.TrimSpace(env("LLM_PROVIDER")))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(s.Provider))
	}
	model := env("AGENT_MODEL")
	if model == "" {
		model = s.Model
	}
	ollamaEndpoint := env("OLLAMA_ENDPOINT")
	if ollamaEndpoint == "" {
		ollamaEndpoint = s.OllamaEndpoint
	}
	if ollamaEndpoint == "" {
		ollamaEndpoint = "http://127.0.0.1:11434"
	}
	var apiKey string
	var roles RoleModels // set when Moly chooses the models of the two roles

	switch provider {
	case "ollama":
		if !isOllamaAvailable(ollamaEndpoint) {
			return nil, fmt.Errorf("no Ollama server answered at %s", ollamaEndpoint)
		}
		if model == "" {
			var err error
			if model, err = detectModel("ollama", "", ollamaEndpoint); err != nil {
				return nil, err
			}
		}
	case "claude":
		apiKey = env("ANTHROPIC_API_KEY")
		if apiKey == "" {
			apiKey = s.APIKey
		}
		if apiKey == "" {
			return nil, fmt.Errorf("the Claude provider needs an API key")
		}
		if model == "" {
			var err error
			if roles, err = chooseRoleModels("claude", apiKey, fromEnv); err != nil {
				return nil, err
			}
			model = roles.Fast
		}
	case "claude-cli":
		// For testing with the developer's own Claude login (see llm_claude_cli.go). Prompts leave the machine. Only the operator
		// can choose it (environment); a user's saved settings cannot.
		if !fromEnv {
			return nil, fmt.Errorf("unknown provider %q; use one of: ollama, claude, openai", provider)
		}
		if _, err := exec.LookPath("claude"); err != nil {
			return nil, fmt.Errorf("LLM_PROVIDER=claude-cli requires the claude command (Claude Code) in PATH")
		}
		if model == "" {
			model = "haiku"
		}
		// A measurement run can give the two roles different models (MOLY_MODEL_FAST, MOLY_MODEL_REPLY: "haiku", "sonnet"), to see
		// how a candidate for the reply role behaves before it is approved (scripts/approve_model.sh).
		if fast, reply := os.Getenv("MOLY_MODEL_FAST"), os.Getenv("MOLY_MODEL_REPLY"); fast != "" || reply != "" {
			if fast == "" {
				fast = model
			}
			if reply == "" {
				reply = fast
			}
			roles = RoleModels{Fast: fast, Reply: reply}
			model = fast
		}
		log.Printf("[LLMClient] WARNING: provider claude-cli sends prompts to Anthropic through the local Claude Code login; use throwaway test users only")
	case "openai":
		apiKey = env("OPENAI_API_KEY")
		if apiKey == "" {
			apiKey = s.APIKey
		}
		if apiKey == "" {
			return nil, fmt.Errorf("the OpenAI provider needs an API key")
		}
		if model == "" {
			var err error
			if roles, err = chooseRoleModels("openai", apiKey, fromEnv); err != nil {
				return nil, err
			}
			model = roles.Fast
		}
	case "":
		if !isOllamaAvailable(ollamaEndpoint) {
			return nil, fmt.Errorf("no AI provider is chosen and no Ollama server answered at %s; choose a provider in Moly's settings", ollamaEndpoint)
		}
		provider = "ollama"
		if model == "" {
			var err error
			if model, err = detectModel("ollama", "", ollamaEndpoint); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("unknown AI provider %q; use one of: ollama, claude, openai", provider)
	}
	log.Printf("[LLMClient] Provider=%s Model=%s", provider, model)

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

	client := &LLMClient{
		Provider:       provider,
		ApiKey:         apiKey,
		Model:          model,
		MaxTokens:      maxTokens,
		Temperature:    temperature,
		Timeout:        timeout,
		OllamaEndpoint: ollamaEndpoint,
		Roles:          roles,
	}
	if roles.Reply != "" && roles.Reply != roles.Fast {
		// the same provider, key and tuning, the reply model (it keeps its own record of what that model refuses)
		client.Reply = &LLMClient{
			Provider: provider, ApiKey: apiKey, Model: roles.Reply, MaxTokens: maxTokens, Temperature: temperature,
			Timeout: timeout, OllamaEndpoint: ollamaEndpoint,
		}
	}
	log.Printf("[LLMClient] Models: quick checks=%s replies=%s (tested: %v)", client.Model, client.ReplyModel(), roles.Tested || roles.Fast == "")
	return client, nil
}

// ApprovedReplies is implemented by a provider that knows whether the models it uses were approved by Moly (see approved_models.json).
// Other providers (a test double, Ollama) are not asked and count as not approved.
type ApprovedReplies interface {
	RepliesAreApproved() bool
}

// RepliesAreApproved says that both models of this client were chosen from the approved table and were measured on Moly's prompts.
func (c *LLMClient) RepliesAreApproved() bool { return c.Roles.Tested }

// ReplyModel is the model that serves the calls that ask for RoleReply.
func (c *LLMClient) ReplyModel() string {
	if c.Reply != nil {
		return c.Reply.Model
	}
	return c.Model
}

// chooseRoleModels asks the provider what the key can use and picks the model of each role from the approved table. The operator
// can name them (MOLY_MODEL_FAST, MOLY_MODEL_REPLY) for a test run; such a choice is never the person's.
func chooseRoleModels(provider, apiKey string, fromEnv bool) (RoleModels, error) {
	if fromEnv {
		fast, reply := os.Getenv("MOLY_MODEL_FAST"), os.Getenv("MOLY_MODEL_REPLY")
		if fast != "" || reply != "" {
			if fast == "" {
				fast = reply
			}
			if reply == "" {
				reply = fast
			}
			log.Printf("[LLMClient] WARNING: the operator named the models (MOLY_MODEL_FAST / MOLY_MODEL_REPLY); they are not the tested ones")
			return RoleModels{Fast: fast, Reply: reply}, nil
		}
	}
	list, err := modelLister(context.Background(), LLMSettings{Provider: provider, APIKey: apiKey})
	if err != nil {
		return RoleModels{}, fmt.Errorf("the available models could not be listed: %w", err)
	}
	return ResolveRoleModels(provider, list, ApprovedModels())
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

// calculateAdaptiveTimeout calculates a reasonable timeout based on prompt length
// Longer prompts need more time for the LLM to process
func (c *LLMClient) calculateAdaptiveTimeout(req *LLMRequest) time.Duration {
	// If request specifies a timeout, use that
	if req.Timeout > 0 {
		return req.Timeout
	}

	// Calculate based on prompt length
	totalPromptLen := len(req.SystemPrompt) + len(req.UserPrompt)

	// Minimum 30 seconds, plus ~1 second per 500 characters of prompt
	// This accounts for LLM processing time increasing with input size
	baseTimeout := 30 * time.Second
	extraSeconds := (totalPromptLen / 500) * 5 // 5 seconds per 500 chars
	adaptiveTimeout := baseTimeout + time.Duration(extraSeconds)*time.Second

	// Cap at 30 minutes for testing phase (increased from 3 min production safety limit)
	if adaptiveTimeout > 1800*time.Second {
		adaptiveTimeout = 1800 * time.Second
	}

	// If c.Timeout is set and longer, use that as baseline
	if c.Timeout > 0 && c.Timeout > baseTimeout {
		adaptiveTimeout = c.Timeout + time.Duration(extraSeconds)*time.Second
	}

	log.Printf("[LLMClient] Adaptive timeout: prompt_len=%d bytes -> timeout=%v", totalPromptLen, adaptiveTimeout)
	return adaptiveTimeout
}

// Call - Make API call to LLM with prompt
func (c *LLMClient) Call(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}
	if req.Role == RoleReply && c.Reply != nil {
		return c.Reply.Call(ctx, req)
	}

	if req.Retries == 0 {
		req.Retries = 1
	}

	// Calculate adaptive timeout based on prompt length
	adaptiveTimeout := c.calculateAdaptiveTimeout(req)

	// If no timeout in context, create one with the adaptive timeout
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, adaptiveTimeout)
		defer cancel()
	}

	log.Printf("[LLMClient] Calling %s with prompt length=%d, retries=%d, timeout=%v", c.Provider, len(req.UserPrompt), req.Retries, adaptiveTimeout)

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
		case "claude":
			resp, err = c.callClaude(ctx, req)
		case "claude-cli":
			resp, err = c.callClaudeCLI(ctx, req)
		default:
			err = fmt.Errorf("unknown provider: %s", c.Provider)
		}

		if err == nil {
			log.Printf("[LLMClient] Success with %s (tokens=%d, time=%dms)", c.Provider, resp.TokensUsed, resp.ProcessingTimeMs)
			if resp.StopReason == "max_tokens" || resp.StopReason == "length" {
				// an answer cut off at its limit is usually unreadable JSON: say so, so that the cause is not a mystery
				log.Printf("[LLMClient] WARNING: the answer was cut off at the token limit (%s); the limit of this call is too small", resp.StopReason)
			}
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

// claudeThinkingHeadroom is added to the output limit when a model may think: its thinking counts toward max_tokens.
// The higher the effort, the more it may need.
func claudeThinkingHeadroom(effort string) int {
	if effort == "high" {
		return 6000
	}
	return 2000
}

// callClaudeAPI - Call Anthropic Claude API
//
// Claude 5 models think adaptively and their thinking counts toward max_tokens, so a short structured answer (a judge, a JSON
// form) can be cut off or come back with no text. These calls need no thinking: it is switched off ("thinking": disabled).
// A model that refuses that (HTTP 400 naming thinking) is asked for low effort with extra room instead, and the client remembers.
// The system prompt is optional: an empty one is left out of the request.
func (c *LLMClient) callClaudeAPI(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.MaxTokens
	}

	temperature := req.Temperature
	if temperature == 0 && req.Temperature == 0 {
		temperature = c.Temperature
	}

	thinks := !claudeTakesTemperature(c.Model) // the Claude 5 family: no temperature, adaptive thinking
	opts := claudeOptions{temperature: claudeTakesTemperature(c.Model) && !c.noTemperature.Load()}
	switch {
	case thinks && (req.Effort == "low" || req.Effort == "medium" || req.Effort == "high"):
		opts.effort = req.Effort // adaptive thinking (the default when "thinking" is left out), steered by effort
	case thinks && c.noThinkOff.Load():
		opts.effort = "low" // a model that refuses "disabled": the lowest effort is the nearest thing
	case thinks:
		opts.thinkingOff = true
	}

	log.Printf("[Claude] Calling with model=%s, tokens=%d, temp=%.1f", c.Model, maxTokens, temperature)

	var httpResp *http.Response
	for attempt := 0; ; attempt++ {
		limit := maxTokens
		if opts.effort != "" {
			limit += claudeThinkingHeadroom(opts.effort)
		}
		resp, err := c.sendClaude(ctx, req, limit, temperature, opts)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusOK {
			httpResp = resp
			break
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		text := string(body)
		// A model the pattern does not know yet may refuse a parameter: send the request again without it and remember it.
		switch {
		case resp.StatusCode == http.StatusBadRequest && attempt < 3 && opts.temperature && strings.Contains(text, "temperature") && strings.Contains(text, "deprecated"):
			c.noTemperature.Store(true)
			opts.temperature = false
			continue
		case resp.StatusCode == http.StatusBadRequest && attempt < 3 && opts.thinkingOff && strings.Contains(text, "thinking"):
			c.noThinkOff.Store(true)
			opts.thinkingOff, opts.effort = false, "low"
			continue
		case resp.StatusCode == http.StatusBadRequest && attempt < 3 && opts.effort != "" && (strings.Contains(text, "effort") || strings.Contains(text, "output_config")):
			opts.effort = "" // the model takes no effort setting: its own default
			continue
		}
		log.Printf("[Claude] API error %d: %s", resp.StatusCode, text)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, text)
	}
	defer httpResp.Body.Close()

	log.Printf("[Claude] API call successful, parsing response...")

	// Parse response. The answer is the text blocks; a thinking block may come first and is not part of it.
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

	var text strings.Builder
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return nil, fmt.Errorf("the model returned no text (stop reason: %s, %d output tokens)", apiResp.StopReason, apiResp.Usage.OutputTokens)
	}

	return &LLMResponse{
		Content:          text.String(),
		StopReason:       apiResp.StopReason,
		TokensUsed:       apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		Model:            c.Model,
		ProcessingTimeMs: int64(time.Since(time.Now()).Milliseconds()),
	}, nil
}

// claudeOptions are the optional parts of a request to the Claude API.
type claudeOptions struct {
	temperature bool   // send a temperature
	thinkingOff bool   // send "thinking": disabled
	effort      string // send output_config.effort (adaptive thinking at that level); empty: the model's own default
}

func (c *LLMClient) sendClaude(ctx context.Context, req *LLMRequest, maxTokens int, temperature float64, o claudeOptions) (*http.Response, error) {
	body := map[string]interface{}{
		"model":      c.Model,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": req.UserPrompt},
		},
	}
	if req.SystemPrompt != "" {
		body["system"] = req.SystemPrompt
	}
	if o.temperature {
		body["temperature"] = temperature
	}
	if o.thinkingOff {
		body["thinking"] = map[string]string{"type": "disabled"}
	}
	if o.effort != "" {
		body["output_config"] = map[string]string{"effort": o.effort}
	}
	bodyJSON, _ := json.Marshal(body)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", claudeMessagesURL, bytes.NewReader(bodyJSON))
	if err != nil {
		log.Printf("[Claude] Request creation failed: %v", err)
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.ApiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// HTTP timeout should be longer than context timeout to avoid premature cancellation
	client := &http.Client{Timeout: c.Timeout + (5 * time.Minute)}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("[Claude] API request failed: %v", err)
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	return resp, nil
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
		"model":   c.Model,
		"prompt":  prompt,
		"stream":  false,
		"options": map[string]interface{}{"temperature": temperature}, // Ollama ignores a top-level temperature
	}
	if req.JSONMode {
		ollamaReq["format"] = "json"
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
// openAIMessages is the conversation of a request: the system prompt first when there is one (it is optional), then the user's.
func openAIMessages(req *LLMRequest) []map[string]string {
	var msgs []map[string]string
	if req.SystemPrompt != "" {
		msgs = append(msgs, map[string]string{"role": "system", "content": req.SystemPrompt})
	}
	return append(msgs, map[string]string{"role": "user", "content": req.UserPrompt})
}

func (c *LLMClient) callOpenAI(ctx context.Context, req *LLMRequest) (*LLMResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.MaxTokens
	}

	temperature := req.Temperature
	if temperature == 0 && req.Temperature == 0 {
		temperature = c.Temperature
	}

	log.Printf("[OpenAI] Calling with model=%s, tokens=%d, temp=%.1f", c.Model, maxTokens, temperature)

	reasoning := openAIReasoningModel.MatchString(c.Model)
	// A reasoning model thinks at the effort the call asks for; a call that asks for none (a judge, an extraction) gets "low".
	effort := ""
	if reasoning && !c.noReasoningEffort.Load() {
		effort = "low"
		if req.Effort == "low" || req.Effort == "medium" || req.Effort == "high" {
			effort = req.Effort
		}
	}
	if reasoning {
		maxTokens += reasoningHeadroom(req.Effort)
	}

	send := func(withTemperature bool) (*http.Response, error) {
		// Build request body for OpenAI API. max_tokens is replaced by max_completion_tokens, which every chat model accepts
		// and the newer ones require.
		body := map[string]interface{}{
			"model":                 c.Model,
			"max_completion_tokens": maxTokens,
			"messages":              openAIMessages(req),
		}
		if withTemperature {
			body["temperature"] = temperature
		}
		if effort != "" {
			body["reasoning_effort"] = effort
		}
		if req.JSONMode {
			body["response_format"] = map[string]string{"type": "json_object"}
		}
		bodyJSON, _ := json.Marshal(body)

		httpReq, err := http.NewRequestWithContext(ctx, "POST", openAIChatURL, bytes.NewReader(bodyJSON))
		if err != nil {
			log.Printf("[OpenAI] Request creation failed: %v", err)
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.ApiKey))

		// HTTP timeout should be longer than context timeout
		client := &http.Client{Timeout: c.Timeout + (5 * time.Minute)}
		httpResp, err := client.Do(httpReq)
		if err != nil {
			log.Printf("[OpenAI] Request failed: %v", err)
			return nil, fmt.Errorf("OpenAI request failed: %w", err)
		}
		return httpResp, nil
	}

	withTemperature := !reasoning && !c.noTemperature.Load()
	httpResp, err := send(withTemperature)
	if err != nil {
		return nil, err
	}
	defer func() { httpResp.Body.Close() }()

	// A model the pattern does not know yet may refuse the temperature ("Unsupported value: 'temperature' does not support
	// ...") or an effort level (the levels differ from model to model): send the request again without it, and remember it for
	// this client.
	for attempt := 0; httpResp.StatusCode == http.StatusBadRequest && attempt < 3; attempt++ {
		body, _ := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		text := string(body)
		switch {
		case withTemperature && (strings.Contains(text, "'temperature'") || strings.Contains(text, "`temperature`")):
			c.noTemperature.Store(true)
			withTemperature = false
		case effort != "" && strings.Contains(text, "reasoning_effort"):
			c.noReasoningEffort.Store(true)
			effort = ""
		default:
			log.Printf("[OpenAI] API error %d: %s", httpResp.StatusCode, text)
			return nil, fmt.Errorf("OpenAI error %d: %s", httpResp.StatusCode, text)
		}
		if httpResp, err = send(withTemperature); err != nil {
			return nil, err
		}
	}

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
			FinishReason string `json:"finish_reason"`
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
	if strings.TrimSpace(apiResp.Choices[0].Message.Content) == "" {
		// a model that thinks first can spend the whole limit on it and answer nothing
		return nil, fmt.Errorf("the model returned no text (finish reason: %s)", apiResp.Choices[0].FinishReason)
	}

	return &LLMResponse{
		Content:          apiResp.Choices[0].Message.Content,
		StopReason:       apiResp.Choices[0].FinishReason,
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
