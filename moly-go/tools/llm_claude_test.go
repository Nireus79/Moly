package tools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
