package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"moly/database"
	"moly/tools"
)

const fakeKey = "sk-ant-TEST-KEY-must-never-be-returned-0123456789"

func newTestLLMManager(t *testing.T) (*llmManager, *tools.SwappableLLM) {
	t.Helper()
	t.Setenv("LLM_PROVIDER", "")
	db, err := database.Init(testDBPath)
	if err != nil {
		t.Fatal(err)
	}
	store := db.GetAppSettings()
	clean := func() {
		for _, k := range []string{keyProvider, keyModel, keyEndpoint, keyAPIKey + "claude", keyAPIKey + "openai"} {
			store.Delete(k)
		}
	}
	clean()
	t.Cleanup(clean)
	swap := tools.NewSwappableLLM()
	return newLLMManager(store, swap), swap
}

func do(h http.HandlerFunc, method, target string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(method, target, &buf))
	return rec
}

func statusOf(t *testing.T, rec *httptest.ResponseRecorder) llmStatus {
	t.Helper()
	var st llmStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatalf("bad status JSON: %v %s", err, rec.Body.String())
	}
	return st
}

func TestLLMSettingsStorePlainKeyAndNeverReturnIt(t *testing.T) {
	m, swap := newTestLLMManager(t)
	rec := do(m.handlePut, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "claude-sonnet-5-5", "apiKey": fakeKey})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT: %d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), fakeKey) {
		t.Fatal("the PUT answer contains the API key")
	}
	st := statusOf(t, rec)
	if st.State != stateReady || st.Active != "claude" || !st.Keys["claude"].Stored || st.Keys["claude"].Sealed {
		t.Fatalf("unexpected status: %+v", st)
	}
	if swap.ProviderName() != "claude" {
		t.Fatalf("the live client was not switched: %s", swap.ProviderName())
	}
	if get := do(m.handleGet, "GET", "/api/settings/llm", nil); strings.Contains(get.Body.String(), fakeKey) {
		t.Fatal("the GET answer contains the API key")
	}
	raw, _, _ := m.store.Get(keyAPIKey + "claude")
	if raw != "plain:"+fakeKey {
		t.Fatalf("stored value = %q", raw)
	}
}

func TestLLMSettingsSealedKeyNeedsThePassphraseAfterRestart(t *testing.T) {
	m, _ := newTestLLMManager(t)
	rec := do(m.handlePut, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "apiKey": fakeKey, "passphrase": "correct horse battery"})
	if st := statusOf(t, rec); rec.Code != 200 || st.State != stateReady || !st.Keys["claude"].Sealed {
		t.Fatalf("sealing failed: %d %+v", rec.Code, st)
	}
	if raw, _, _ := m.store.Get(keyAPIKey + "claude"); strings.Contains(raw, fakeKey) || !strings.HasPrefix(raw, "sealed:v1:") {
		t.Fatalf("the saved value must be sealed, got %q", raw)
	}

	// A restart: a new manager over the same database knows nothing but the sealed value.
	swap2 := tools.NewSwappableLLM()
	m2 := newLLMManager(m.store, swap2)
	m2.load()
	if m2.State() != stateLocked || swap2.ProviderName() != "none" {
		t.Fatalf("after a restart the key must be locked, state %s provider %s", m2.State(), swap2.ProviderName())
	}
	if _, err := swap2.Call(context.Background(), &tools.LLMRequest{}); err != tools.ErrNoProvider {
		t.Fatalf("a locked backend must refuse to call the provider: %v", err)
	}

	if rec := do(m2.handleUnlock, "POST", "/api/settings/llm/unlock", map[string]string{"passphrase": "not the passphrase"}); rec.Code != http.StatusForbidden {
		t.Fatalf("wrong passphrase: %d", rec.Code)
	}
	if m2.State() != stateLocked {
		t.Fatal("a wrong passphrase must leave it locked")
	}
	rec = do(m2.handleUnlock, "POST", "/api/settings/llm/unlock", map[string]string{"passphrase": "correct horse battery"})
	if rec.Code != 200 || m2.State() != stateReady || swap2.ProviderName() != "claude" || strings.Contains(rec.Body.String(), fakeKey) {
		t.Fatalf("unlock failed or leaked: %d %s", rec.Code, rec.Body.String())
	}

	do(m2.handleLock, "POST", "/api/settings/llm/lock", nil)
	if m2.State() != stateLocked || swap2.ProviderName() != "none" {
		t.Fatal("lock must forget the key")
	}
}

func TestLLMSettingsSealAnExistingPlainKeyLater(t *testing.T) {
	m, _ := newTestLLMManager(t)
	do(m.handlePut, "PUT", "/api/settings/llm", map[string]any{"provider": "openai", "apiKey": fakeKey})
	rec := do(m.handlePut, "PUT", "/api/settings/llm", map[string]any{"passphrase": "correct horse battery"})
	if st := statusOf(t, rec); rec.Code != 200 || !st.Keys["openai"].Sealed {
		t.Fatalf("sealing the stored key failed: %d %s", rec.Code, rec.Body.String())
	}
}

func TestLLMSettingsValidation(t *testing.T) {
	m, _ := newTestLLMManager(t)
	cases := []map[string]any{
		{"provider": "skynet"},
		{"provider": "claude", "apiKey": "has space"},
		{"provider": "claude", "model": "bad model!"},
		{"provider": "ollama", "ollamaEndpoint": "file:///etc/passwd"},
		{"provider": "ollama", "apiKey": "keys are for cloud providers"},
		{"provider": "claude", "apiKey": fakeKey, "passphrase": "short"},
	}
	for _, c := range cases {
		if rec := do(m.handlePut, "PUT", "/api/settings/llm", c); rec.Code != http.StatusBadRequest {
			t.Errorf("%v: want 400, got %d %s", c, rec.Code, rec.Body.String())
		}
	}
	if raw, ok, _ := m.store.Get(keyProvider); ok {
		t.Errorf("a rejected request must save nothing, provider is %q", raw)
	}
}

func TestLLMSettingsCloudProviderWithoutKeyIsUnconfigured(t *testing.T) {
	m, swap := newTestLLMManager(t)
	rec := do(m.handlePut, "PUT", "/api/settings/llm", map[string]any{"provider": "claude"})
	if st := statusOf(t, rec); st.State != stateUnconfigured || swap.ProviderName() != "none" {
		t.Fatalf("no key must mean unconfigured: %+v", st)
	}
}

func TestLLMSettingsDeleteKey(t *testing.T) {
	m, swap := newTestLLMManager(t)
	do(m.handlePut, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "apiKey": fakeKey})
	rec := do(m.handleDeleteKey, "DELETE", "/api/settings/llm/key?provider=claude", nil)
	if st := statusOf(t, rec); st.Keys["claude"].Stored || st.State != stateUnconfigured || swap.ProviderName() != "none" {
		t.Fatalf("key not removed: %+v", st)
	}
	if _, ok, _ := m.store.Get(keyAPIKey + "claude"); ok {
		t.Fatal("the key is still in the database")
	}
}

func TestLLMSettingsEnvironmentWins(t *testing.T) {
	m, _ := newTestLLMManager(t)
	t.Setenv("LLM_PROVIDER", "ollama")
	m = newLLMManager(m.store, tools.NewSwappableLLM())
	if rec := do(m.handlePut, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "apiKey": fakeKey}); rec.Code != http.StatusConflict {
		t.Fatalf("with LLM_PROVIDER set the settings must be read-only, got %d", rec.Code)
	}
}

func TestLLMSettingsTestEndpointReportsFailureWithoutSaving(t *testing.T) {
	m, _ := newTestLLMManager(t)
	rec := do(m.handleTest, "POST", "/api/settings/llm/test", map[string]any{"provider": "ollama", "ollamaEndpoint": "http://127.0.0.1:1"})
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	if out["ok"] != false || out["error"] == "" {
		t.Fatalf("an unreachable Ollama must be reported: %s", rec.Body.String())
	}
	if _, ok, _ := m.store.Get(keyProvider); ok {
		t.Fatal("the test endpoint must save nothing")
	}
}

func TestScrubRemovesAKeyFromAMessage(t *testing.T) {
	if got := scrub("401 invalid x-api-key "+fakeKey, fakeKey); strings.Contains(got, fakeKey) {
		t.Fatalf("scrub left the key: %s", got)
	}
}

func TestOllamaModelsListsTheModelsOfAServer(t *testing.T) {
	m, _ := newTestLLMManager(t)
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"models":[{"name":"mistral:latest"},{"name":"llama3:8b"}]}`))
	}))
	defer fake.Close()
	rec := do(m.handleOllamaModels, "GET", "/api/settings/llm/ollama-models?endpoint="+fake.URL, nil)
	var out struct {
		Models []string `json:"models"`
		Error  string   `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Models) != 2 || out.Models[0] != "mistral:latest" || out.Error != "" {
		t.Fatalf("models = %+v", out)
	}
	rec = do(m.handleOllamaModels, "GET", "/api/settings/llm/ollama-models?endpoint=http://127.0.0.1:1", nil)
	json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Models) != 0 || out.Error == "" {
		t.Fatalf("an unreachable server must be reported: %s", rec.Body.String())
	}
	if rec := do(m.handleOllamaModels, "GET", "/api/settings/llm/ollama-models?endpoint=file:///etc", nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("a non-http endpoint must be refused, got %d", rec.Code)
	}
}
