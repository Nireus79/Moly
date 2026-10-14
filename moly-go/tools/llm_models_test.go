package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func fakeVendor(t *testing.T, target *string, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	old := *target
	*target = srv.URL
	t.Cleanup(func() { *target = old })
}

// The models come from the provider, with the user's key; nothing is assumed. Claude: active models only, newest first, the
// suggestion is the newest of the balanced ("sonnet") line.
func TestListClaudeModelsAsksTheProviderWithTheUsersKey(t *testing.T) {
	var gotKey, gotVersion, gotQuery string
	fakeVendor(t, &claudeModelsURL, func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotVersion, gotQuery = r.Header.Get("x-api-key"), r.Header.Get("anthropic-version"), r.URL.RawQuery
		if r.URL.Query().Get("after_id") == "" {
			w.Write([]byte(`{"data":[{"id":"claude-opus-9","display_name":"Claude Opus 9","line":"opus","lifecycle":"active"},{"id":"claude-sonnet-9","display_name":"Claude Sonnet 9","line":"sonnet","lifecycle":"active"}],"has_more":true,"last_id":"claude-sonnet-9"}`))
			return
		}
		w.Write([]byte(`{"data":[{"id":"claude-haiku-9","display_name":"Claude Haiku 9","line":"haiku","lifecycle":"active"},{"id":"claude-old-1","display_name":"Old","lifecycle":"retired"}],"has_more":false}`))
	})
	list, err := ListModels(context.Background(), LLMSettings{Provider: "claude", APIKey: "sk-ant-USER-KEY"})
	if err != nil {
		t.Fatal(err)
	}
	if gotKey != "sk-ant-USER-KEY" || gotVersion == "" || !strings.Contains(gotQuery, "lifecycle=active") {
		t.Errorf("key %q version %q query %q", gotKey, gotVersion, gotQuery)
	}
	var ids []string
	for _, m := range list.Models {
		ids = append(ids, m.ID)
	}
	if strings.Join(ids, ",") != "claude-opus-9,claude-sonnet-9,claude-haiku-9" {
		t.Errorf("both pages, retired model left out, got %v", ids)
	}
	if list.Suggested != "claude-sonnet-9" || list.Models[0].Label != "Claude Opus 9" {
		t.Errorf("suggested %q, first label %q", list.Suggested, list.Models[0].Label)
	}
}

func TestListClaudeModelsWithoutAKeyOrWithABadOneSaysSo(t *testing.T) {
	if _, err := ListModels(context.Background(), LLMSettings{Provider: "claude"}); err == nil || !strings.Contains(err.Error(), "API key") {
		t.Fatalf("no key: %v", err)
	}
	fakeVendor(t, &claudeModelsURL, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`))
	})
	if _, err := ListModels(context.Background(), LLMSettings{Provider: "claude", APIKey: "bad"}); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("a rejected key must be reported with its status: %v", err)
	}
}

// OpenAI's list has no field for what a model is for: the families that are not chat are left out, the newest comes first.
func TestListOpenAIModelsLeavesOutWhatIsNotChatAndSortsNewestFirst(t *testing.T) {
	var gotAuth string
	fakeVendor(t, &openAIModelsURL, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"data":[
			{"id":"text-embedding-9","created":300},{"id":"gpt-old","created":100},{"id":"whisper-2","created":250},
			{"id":"gpt-new","created":400},{"id":"gpt-image-9","created":390},{"id":"o9-mini","created":350},{"id":"tts-9","created":340}]}`))
	})
	list, err := ListModels(context.Background(), LLMSettings{Provider: "openai", APIKey: "sk-USER"})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range list.Models {
		ids = append(ids, m.ID)
	}
	if gotAuth != "Bearer sk-USER" || strings.Join(ids, ",") != "gpt-new,o9-mini,gpt-old" || list.Suggested != "gpt-new" {
		t.Errorf("auth %q models %v suggested %q", gotAuth, ids, list.Suggested)
	}
}

func TestListOllamaModelsAsksTheServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"models":[{"name":"llama3:8b"},{"name":"mistral:latest"}]}`))
	}))
	defer srv.Close()
	list, err := ListModels(context.Background(), LLMSettings{Provider: "ollama", OllamaEndpoint: srv.URL})
	if err != nil || len(list.Models) != 2 || list.Suggested != "llama3:8b" {
		t.Fatalf("%+v %v", list, err)
	}
	if _, err := ListModels(context.Background(), LLMSettings{Provider: "ollama", OllamaEndpoint: "http://127.0.0.1:1"}); err == nil {
		t.Fatal("an unreachable server must be reported")
	}
}

// No model is assumed: when none is chosen, the provider is asked, and nothing is made up if it cannot answer.
func TestAClientWithoutAModelTakesTheProvidersSuggestion(t *testing.T) {
	fakeVendor(t, &claudeModelsURL, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":[{"id":"claude-sonnet-9","display_name":"S","line":"sonnet","lifecycle":"active"}],"has_more":false}`))
	})
	c, err := NewLLMClientWith(LLMSettings{Provider: "claude", APIKey: "k"})
	if err != nil || c.Model != "claude-sonnet-9" {
		t.Fatalf("model %v err %v", c, err)
	}
	fakeVendor(t, &claudeModelsURL, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) })
	if _, err := NewLLMClientWith(LLMSettings{Provider: "claude", APIKey: "k"}); err == nil || !strings.Contains(err.Error(), "no model is chosen") {
		t.Fatalf("with no model and no list there is no client: %v", err)
	}
	// a chosen model needs no question
	if c, err := NewLLMClientWith(LLMSettings{Provider: "claude", APIKey: "k", Model: "claude-whatever"}); err != nil || c.Model != "claude-whatever" {
		t.Fatalf("a chosen model is used as it is: %v %v", c, err)
	}
}
