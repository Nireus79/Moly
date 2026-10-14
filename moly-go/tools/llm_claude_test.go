package tools

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestClaudeTakesTemperature(t *testing.T) {
	for model, want := range map[string]bool{
		"claude-sonnet-5-5": false, "claude-opus-5-5": false, "claude-haiku-5-5": false, "claude-fable-5-1": false, "claude-opus-5": false,
		"claude-3-5-sonnet-20241022": true, "claude-sonnet-4-5": true, "claude-opus-4-1": true,
	} {
		if got := claudeTakesTemperature(model); got != want {
			t.Errorf("%s: takes temperature = %v, want %v", model, got, want)
		}
	}
}

// The API answered 400 "`temperature` is deprecated for this model" to every request that carried a temperature.
func claudeStub(t *testing.T, seen *[]bool) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		_, has := body["temperature"]
		*seen = append(*seen, has)
		if has {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"` + "`temperature` is deprecated for this model." + `"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"OK"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	t.Cleanup(srv.Close)
	old := claudeMessagesURL
	claudeMessagesURL = srv.URL
	t.Cleanup(func() { claudeMessagesURL = old })
}

func TestClaude5SendsNoTemperature(t *testing.T) {
	var seen []bool
	claudeStub(t, &seen)
	c := &LLMClient{Provider: "claude", ApiKey: "k", Model: "claude-sonnet-5-5", MaxTokens: 8, Temperature: 0.7, Timeout: time.Minute}
	if _, err := c.Call(context.Background(), &LLMRequest{SystemPrompt: "s", UserPrompt: "u", Temperature: 0.2}); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0] {
		t.Fatalf("requests with a temperature: %v, want one request without", seen)
	}
}

func TestClaudeUnknownModelRetriesWithoutTemperatureOnce(t *testing.T) {
	var seen []bool
	claudeStub(t, &seen)
	c := &LLMClient{Provider: "claude", ApiKey: "k", Model: "claude-future", MaxTokens: 8, Temperature: 0.7, Timeout: time.Minute}
	for i := 0; i < 2; i++ {
		if _, err := c.Call(context.Background(), &LLMRequest{SystemPrompt: "s", UserPrompt: "u", Temperature: 0.2}); err != nil {
			t.Fatal(err)
		}
	}
	if len(seen) != 3 || !seen[0] || seen[1] || seen[2] {
		t.Fatalf("temperature per request: %v, want [true false false]", seen)
	}
}

func TestOpenAIReasoningModelPattern(t *testing.T) {
	for model, want := range map[string]bool{
		"gpt-5.6-sol": true, "gpt-5": true, "gpt-5-mini": true, "gpt-6-luna": true, "gpt-6.1-sol": true, "o3": true, "o4-mini": true, "o1-preview": true,
		"gpt-4-turbo": false, "gpt-4o": false, "gpt-4.1": false, "gpt-4o-mini": false,
	} {
		if got := openAIReasoningModel.MatchString(model); got != want {
			t.Errorf("%s: reasoning = %v, want %v", model, got, want)
		}
	}
}

// openAIStub records the body of each request. It answers 400 like the API does when the request carries max_tokens, or
// (when strictTemp) a temperature.
func openAIStub(t *testing.T, strictTemp bool, seen *[]map[string]any) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		*seen = append(*seen, body)
		_, hasMax := body["max_tokens"]
		_, hasTemp := body["temperature"]
		if hasMax {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"Unsupported parameter: 'max_tokens' is not supported with this model. Use 'max_completion_tokens' instead."}}`))
			return
		}
		if hasTemp && strictTemp {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"Unsupported value: 'temperature' does not support 0.2 with this model. Only the default (1) value is supported."}}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"OK"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	t.Cleanup(srv.Close)
	old := openAIChatURL
	openAIChatURL = srv.URL
	t.Cleanup(func() { openAIChatURL = old })
}

func TestOpenAIReasoningModelSendsCompletionLimitAndNoTemperature(t *testing.T) {
	var seen []map[string]any
	openAIStub(t, true, &seen)
	c := &LLMClient{Provider: "openai", ApiKey: "k", Model: "gpt-5.6-sol", MaxTokens: 8, Temperature: 0.7, Timeout: time.Minute}
	if _, err := c.Call(context.Background(), &LLMRequest{SystemPrompt: "s", UserPrompt: "u", Temperature: 0.2, MaxTokens: 100}); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 {
		t.Fatalf("requests: %d, want 1", len(seen))
	}
	if _, has := seen[0]["temperature"]; has {
		t.Error("a temperature was sent to a reasoning model")
	}
	if got := seen[0]["max_completion_tokens"]; got != float64(100+reasoningHeadroom) {
		t.Errorf("max_completion_tokens = %v, want %d", got, 100+reasoningHeadroom)
	}
}

func TestOpenAIOlderModelKeepsTemperatureAndGetsCompletionLimit(t *testing.T) {
	var seen []map[string]any
	openAIStub(t, false, &seen)
	c := &LLMClient{Provider: "openai", ApiKey: "k", Model: "gpt-4o", MaxTokens: 8, Temperature: 0.7, Timeout: time.Minute}
	if _, err := c.Call(context.Background(), &LLMRequest{SystemPrompt: "s", UserPrompt: "u", Temperature: 0.2, MaxTokens: 100}); err != nil {
		t.Fatal(err)
	}
	if seen[0]["temperature"] != 0.2 || seen[0]["max_completion_tokens"] != float64(100) {
		t.Errorf("body = %v", seen[0])
	}
}

func TestOpenAIUnknownModelRetriesWithoutTemperatureOnce(t *testing.T) {
	var seen []map[string]any
	openAIStub(t, true, &seen)
	c := &LLMClient{Provider: "openai", ApiKey: "k", Model: "future-chat", MaxTokens: 8, Temperature: 0.7, Timeout: time.Minute}
	for i := 0; i < 2; i++ {
		if _, err := c.Call(context.Background(), &LLMRequest{SystemPrompt: "s", UserPrompt: "u", Temperature: 0.2}); err != nil {
			t.Fatal(err)
		}
	}
	has := func(i int) bool { _, ok := seen[i]["temperature"]; return ok }
	if len(seen) != 3 || !has(0) || has(1) || has(2) {
		t.Fatalf("temperature per request: %d requests, want [true false false]", len(seen))
	}
}

// ---- Claude 5: adaptive thinking, optional system prompt ----

// claudeBodies runs a fake Messages API that answers with reply(request body, n-th request) and records every request body.
func claudeBodies(t *testing.T, reply func(body map[string]any, n int) (int, string)) *[]map[string]any {
	t.Helper()
	var seen []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		seen = append(seen, body)
		code, out := reply(body, len(seen))
		w.WriteHeader(code)
		_, _ = w.Write([]byte(out))
	}))
	t.Cleanup(srv.Close)
	old := claudeMessagesURL
	claudeMessagesURL = srv.URL
	t.Cleanup(func() { claudeMessagesURL = old })
	return &seen
}

const textAnswer = `{"content":[{"type":"text","text":"OK"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`

func claudeClient(model string) *LLMClient {
	return &LLMClient{Provider: "claude", ApiKey: "k", Model: model, MaxTokens: 100, Temperature: 0.7, Timeout: time.Minute}
}

// Found live (2026-10-14): the extraction call has no system prompt and the client refused it ("system prompt cannot be empty"),
// so with Claude nothing was ever extracted from a message.
func TestClaudeAcceptsARequestWithoutASystemPrompt(t *testing.T) {
	seen := claudeBodies(t, func(map[string]any, int) (int, string) { return 200, textAnswer })
	resp, err := claudeClient("claude-haiku-5-5").Call(context.Background(), &LLMRequest{UserPrompt: "u"})
	if err != nil || resp.Content != "OK" {
		t.Fatalf("a request with no system prompt must work: %v %+v", err, resp)
	}
	if _, has := (*seen)[0]["system"]; has {
		t.Error("an empty system prompt must be left out of the request")
	}
}

// Found live: a Claude 5 answer can start with a thinking block; the client took the first block and returned "" (0 chars).
func TestClaudeTakesTheTextBlocksNotTheFirstBlock(t *testing.T) {
	claudeBodies(t, func(map[string]any, int) (int, string) {
		return 200, `{"content":[{"type":"thinking","thinking":"hmm","signature":"x"},{"type":"text","text":"{\"a\":"},{"type":"text","text":"1}"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":9}}`
	})
	resp, err := claudeClient("claude-haiku-5-5").Call(context.Background(), &LLMRequest{SystemPrompt: "s", UserPrompt: "u"})
	if err != nil || resp.Content != `{"a":1}` {
		t.Fatalf("the answer is the text blocks, got %q (%v)", resp.Content, err)
	}
}

// An answer with no text is an error, not an empty success: the caller retries or reports it.
func TestClaudeAnAnswerWithoutTextIsAnError(t *testing.T) {
	claudeBodies(t, func(map[string]any, int) (int, string) {
		return 200, `{"content":[{"type":"thinking","thinking":"..."}],"stop_reason":"max_tokens","usage":{"input_tokens":1,"output_tokens":100}}`
	})
	_, err := claudeClient("claude-haiku-5-5").Call(context.Background(), &LLMRequest{SystemPrompt: "s", UserPrompt: "u", Retries: 1})
	if err == nil || !strings.Contains(err.Error(), "no text") || !strings.Contains(err.Error(), "max_tokens") {
		t.Fatalf("want an error naming the stop reason, got %v", err)
	}
}

func TestClaude5IsAskedNotToThinkAndOlderModelsAreNot(t *testing.T) {
	seen := claudeBodies(t, func(map[string]any, int) (int, string) { return 200, textAnswer })
	claudeClient("claude-sonnet-5-5").Call(context.Background(), &LLMRequest{SystemPrompt: "s", UserPrompt: "u"})
	claudeClient("claude-3-5-sonnet-20241022").Call(context.Background(), &LLMRequest{SystemPrompt: "s", UserPrompt: "u"})
	if th, _ := (*seen)[0]["thinking"].(map[string]any); th["type"] != "disabled" {
		t.Errorf("a Claude 5 model is asked for no thinking, got %v", (*seen)[0]["thinking"])
	}
	if _, has := (*seen)[1]["thinking"]; has {
		t.Error("an older model is not sent a thinking setting")
	}
}

// A model that refuses "thinking: disabled" (HTTP 400) gets low effort and more room, once, and the client remembers.
func TestClaudeModelThatRefusesDisabledThinkingGetsLowEffortAndRoom(t *testing.T) {
	seen := claudeBodies(t, func(body map[string]any, n int) (int, string) {
		if th, ok := body["thinking"].(map[string]any); ok && th["type"] == "disabled" {
			return 400, `{"type":"error","error":{"type":"invalid_request_error","message":"thinking.type: disabled is not supported for this model"}}`
		}
		return 200, textAnswer
	})
	c := claudeClient("claude-opus-5-5")
	for i := 0; i < 2; i++ {
		if _, err := c.Call(context.Background(), &LLMRequest{SystemPrompt: "s", UserPrompt: "u", MaxTokens: 50}); err != nil {
			t.Fatal(err)
		}
	}
	if len(*seen) != 3 {
		t.Fatalf("one refused request, then two accepted (the client remembers), got %d requests", len(*seen))
	}
	last := (*seen)[2]
	if oc, _ := last["output_config"].(map[string]any); oc["effort"] != "low" {
		t.Errorf("low effort expected, got %v", last["output_config"])
	}
	if _, has := last["thinking"]; has {
		t.Error("the refused setting is no longer sent")
	}
	if last["max_tokens"] != float64(50+claudeThinkingHeadroom) {
		t.Errorf("room for the thinking expected, max_tokens = %v", last["max_tokens"])
	}
}

// ---- OpenAI: optional system prompt, an answer with no text is an error ----

func openAIBodies(t *testing.T, reply string) *[]map[string]any {
	t.Helper()
	var seen []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		seen = append(seen, body)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(srv.Close)
	old := openAIChatURL
	openAIChatURL = srv.URL
	t.Cleanup(func() { openAIChatURL = old })
	return &seen
}

func TestOpenAIAcceptsARequestWithoutASystemPrompt(t *testing.T) {
	seen := openAIBodies(t, `{"choices":[{"message":{"content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
	c := &LLMClient{Provider: "openai", ApiKey: "k", Model: "gpt-4o", MaxTokens: 100, Temperature: 0.7, Timeout: time.Minute}
	if _, err := c.Call(context.Background(), &LLMRequest{UserPrompt: "u"}); err != nil {
		t.Fatal(err)
	}
	msgs, _ := (*seen)[0]["messages"].([]any)
	if len(msgs) != 1 || msgs[0].(map[string]any)["role"] != "user" {
		t.Fatalf("only the user message is sent, got %v", msgs)
	}
}

func TestOpenAIAnAnswerWithoutTextIsAnError(t *testing.T) {
	openAIBodies(t, `{"choices":[{"message":{"content":""},"finish_reason":"length"}],"usage":{"prompt_tokens":1,"completion_tokens":100}}`)
	c := &LLMClient{Provider: "openai", ApiKey: "k", Model: "gpt-5.6-sol", MaxTokens: 100, Timeout: time.Minute}
	_, err := c.Call(context.Background(), &LLMRequest{SystemPrompt: "s", UserPrompt: "u", Retries: 1})
	if err == nil || !strings.Contains(err.Error(), "no text") || !strings.Contains(err.Error(), "length") {
		t.Fatalf("want an error naming the finish reason, got %v", err)
	}
}

// An answer cut off at its limit is the likely cause of unreadable JSON: the client says so in the log.
func TestACutOffAnswerIsLogged(t *testing.T) {
	claudeBodies(t, func(map[string]any, int) (int, string) {
		return 200, `{"content":[{"type":"text","text":"{\"a\": [{\"b\": 1}"}],"stop_reason":"max_tokens","usage":{"input_tokens":1,"output_tokens":100}}`
	})
	var buf strings.Builder
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	if _, err := claudeClient("claude-haiku-5-5").Call(context.Background(), &LLMRequest{SystemPrompt: "s", UserPrompt: "u"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "cut off at the token limit") {
		t.Fatalf("the log must say the answer was cut off, got %q", buf.String())
	}
}
