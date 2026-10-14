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
