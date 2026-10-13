package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"moly/database"
	"moly/privacy"
	"moly/secrets"
	"moly/tools"
)

// The AI provider, model and API key are saved in the encrypted database (table app_settings). An API key can also be
// sealed with a passphrase (package secrets): then only the passphrase opens it, and it is held in memory only after
// the user unlocks it. The key is never returned by any endpoint and never logged.

const (
	keyProvider = "llm.provider"
	keyModel    = "llm.model"
	keyEndpoint = "llm.ollama_endpoint"
	keyAPIKey   = "llm.api_key." // + provider
)

var (
	cloudProviders = map[string]bool{"claude": true, "openai": true}
	allProviders   = map[string]bool{"ollama": true, "claude": true, "openai": true}
	modelPattern   = regexp.MustCompile(`^[A-Za-z0-9._:/\-]{1,100}$`)
)

type llmManager struct {
	mu       sync.Mutex
	store    *database.AppSettings
	swap     *tools.SwappableLLM
	envFixed bool // LLM_PROVIDER is set: the environment decides, saved settings are ignored
	state    string
	lastErr  string
	failed   int
}

const (
	stateReady        = "ready"
	stateLocked       = "locked"       // a sealed key is saved and not yet unlocked
	stateUnconfigured = "unconfigured" // nothing usable yet
)

func newLLMManager(store *database.AppSettings, swap *tools.SwappableLLM) *llmManager {
	return &llmManager{store: store, swap: swap, envFixed: strings.TrimSpace(os.Getenv("LLM_PROVIDER")) != ""}
}

func (m *llmManager) State() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *llmManager) LastError() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastErr
}

func (m *llmManager) get(key string) string {
	v, _, _ := m.store.Get(key)
	return v
}

// load applies the saved settings. Call once at startup.
func (m *llmManager) load() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applyLocked("")
}

// applyLocked builds the client from the saved settings. unlockedKey, when set, is the plain key of the provider.
func (m *llmManager) applyLocked(unlockedKey string) {
	s := tools.LLMSettings{}
	if !m.envFixed {
		s.Provider = m.get(keyProvider)
		s.Model = m.get(keyModel)
		s.OllamaEndpoint = m.get(keyEndpoint)
		if cloudProviders[s.Provider] {
			stored := m.get(keyAPIKey + s.Provider)
			switch {
			case stored == "":
				m.setState(stateUnconfigured, "No API key is saved for "+s.Provider+".", nil, "")
				return
			case unlockedKey != "":
				s.APIKey = unlockedKey
			case secrets.IsSealed(stored):
				m.setState(stateLocked, "The API key is protected by a passphrase: unlock it.", nil, "")
				return
			default:
				plain, ok := secrets.FromPlain(stored)
				if !ok {
					m.setState(stateUnconfigured, "The saved API key is unreadable; enter it again.", nil, "")
					return
				}
				s.APIKey = plain
			}
		}
	}
	if s.APIKey != "" {
		privacy.Register(s.APIKey) // masked in every log line from now on
	}
	client, err := tools.NewLLMClientWith(s)
	if err != nil {
		m.setState(stateUnconfigured, scrub(err.Error(), s.APIKey), nil, "")
		return
	}
	m.setState(stateReady, "", client, client.Provider)
}

func (m *llmManager) setState(state, msg string, client tools.LLMProvider, name string) {
	m.state, m.lastErr = state, msg
	if client == nil {
		m.swap.Set(nil, "")
		return
	}
	m.swap.Set(client, name)
}

// scrub removes a key from a message, in case a provider echoes it.
func scrub(msg, key string) string {
	if key != "" {
		msg = strings.ReplaceAll(msg, key, "[key]")
	}
	return msg
}

type keyInfo struct {
	Stored bool `json:"stored"`
	Sealed bool `json:"sealed"`
}

type llmStatus struct {
	Provider       string             `json:"provider"`
	Model          string             `json:"model"`
	OllamaEndpoint string             `json:"ollamaEndpoint"`
	State          string             `json:"state"`
	Error          string             `json:"error,omitempty"`
	Active         string             `json:"active"`
	FixedByEnv     bool               `json:"fixedByEnvironment"`
	Keys           map[string]keyInfo `json:"keys"`
}

func (m *llmManager) status() llmStatus {
	st := llmStatus{
		Provider: m.get(keyProvider), Model: m.get(keyModel), OllamaEndpoint: m.get(keyEndpoint),
		State: m.state, Error: m.lastErr, Active: m.swap.ProviderName(), FixedByEnv: m.envFixed,
		Keys: map[string]keyInfo{},
	}
	for p := range cloudProviders {
		v := m.get(keyAPIKey + p)
		st.Keys[p] = keyInfo{Stored: v != "", Sealed: secrets.IsSealed(v)}
	}
	return st
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func readBody(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return false
	}
	return true
}

func (m *llmManager) fixedByEnv(w http.ResponseWriter) bool {
	if m.envFixed {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "the provider is fixed by the LLM_PROVIDER environment variable"})
		return true
	}
	return false
}

// GET /api/settings/llm
func (m *llmManager) handleGet(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	writeJSON(w, http.StatusOK, m.status())
}

type llmPut struct {
	Provider       *string `json:"provider"`
	Model          *string `json:"model"`
	OllamaEndpoint *string `json:"ollamaEndpoint"`
	APIKey         string  `json:"apiKey"`
	Passphrase     string  `json:"passphrase"`
}

func validCandidate(provider, model, endpoint, apiKey string) string {
	switch {
	case !allProviders[provider]:
		return "provider must be ollama, claude or openai"
	case model != "" && !modelPattern.MatchString(model):
		return "invalid model name"
	case len(apiKey) > 512 || strings.ContainsAny(apiKey, " \t\r\n"):
		return "invalid API key"
	}
	if endpoint != "" {
		if u, err := url.Parse(endpoint); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return "the Ollama address must be an http or https URL"
		}
	}
	return ""
}

// PUT /api/settings/llm
func (m *llmManager) handlePut(w http.ResponseWriter, r *http.Request) {
	if m.fixedByEnv(w) {
		return
	}
	var req llmPut
	if !readBody(w, r, &req) {
		return
	}
	req.APIKey = strings.TrimSpace(req.APIKey)

	m.mu.Lock()
	defer m.mu.Unlock()
	provider := m.get(keyProvider)
	if req.Provider != nil {
		provider = strings.ToLower(strings.TrimSpace(*req.Provider))
	}
	model, endpoint := m.get(keyModel), m.get(keyEndpoint)
	if req.Model != nil {
		model = strings.TrimSpace(*req.Model)
	}
	if req.OllamaEndpoint != nil {
		endpoint = strings.TrimSpace(*req.OllamaEndpoint)
	}
	if msg := validCandidate(provider, model, endpoint, req.APIKey); msg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if req.APIKey != "" && !cloudProviders[provider] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "only Claude and OpenAI use an API key"})
		return
	}

	// What to store for the key: a new key (plain or sealed), or an existing plain key now sealed.
	var toStore, plainForApply string
	switch {
	case req.APIKey != "" && req.Passphrase != "":
		sealed, err := secrets.Seal(req.Passphrase, req.APIKey)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		toStore, plainForApply = sealed, req.APIKey
	case req.APIKey != "":
		toStore, plainForApply = secrets.Plain(req.APIKey), req.APIKey
	case req.Passphrase != "" && cloudProviders[provider]:
		plain, ok := secrets.FromPlain(m.get(keyAPIKey + provider))
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "there is no plain key to protect; send the key with the passphrase"})
			return
		}
		sealed, err := secrets.Seal(req.Passphrase, plain)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		toStore, plainForApply = sealed, plain
	}

	m.store.Set(keyProvider, provider)
	m.store.Set(keyModel, model)
	m.store.Set(keyEndpoint, endpoint)
	if toStore != "" {
		m.store.Set(keyAPIKey+provider, toStore)
	}
	m.applyLocked(plainForApply)
	writeJSON(w, http.StatusOK, m.status())
}

// POST /api/settings/llm/unlock
func (m *llmManager) handleUnlock(w http.ResponseWriter, r *http.Request) {
	if m.fixedByEnv(w) {
		return
	}
	var req struct {
		Passphrase string `json:"passphrase"`
	}
	if !readBody(w, r, &req) {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	provider := m.get(keyProvider)
	stored := m.get(keyAPIKey + provider)
	if !cloudProviders[provider] || !secrets.IsSealed(stored) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "there is no passphrase-protected key to unlock"})
		return
	}
	key, err := secrets.Open(req.Passphrase, stored)
	if err != nil {
		m.failed++
		delay := m.failed
		if delay > 5 {
			delay = 5
		}
		time.Sleep(time.Duration(delay) * time.Second) // slows down guessing
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "wrong passphrase"})
		return
	}
	m.failed = 0
	m.applyLocked(key)
	writeJSON(w, http.StatusOK, m.status())
}

// POST /api/settings/llm/lock forgets an unlocked key (it stays saved, sealed).
func (m *llmManager) handleLock(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()
	provider := m.get(keyProvider)
	if cloudProviders[provider] && secrets.IsSealed(m.get(keyAPIKey+provider)) {
		m.applyLocked("")
	}
	writeJSON(w, http.StatusOK, m.status())
}

// DELETE /api/settings/llm/key?provider=claude
func (m *llmManager) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	if m.fixedByEnv(w) {
		return
	}
	provider := strings.ToLower(r.URL.Query().Get("provider"))
	if !cloudProviders[provider] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider must be claude or openai"})
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store.Delete(keyAPIKey + provider)
	m.applyLocked("")
	writeJSON(w, http.StatusOK, m.status())
}

// POST /api/settings/llm/test tries a provider with a tiny request and saves nothing.
func (m *llmManager) handleTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider       string `json:"provider"`
		Model          string `json:"model"`
		OllamaEndpoint string `json:"ollamaEndpoint"`
		APIKey         string `json:"apiKey"`
	}
	if !readBody(w, r, &req) {
		return
	}
	req.Provider, req.APIKey = strings.ToLower(strings.TrimSpace(req.Provider)), strings.TrimSpace(req.APIKey)
	if msg := validCandidate(req.Provider, req.Model, req.OllamaEndpoint, req.APIKey); msg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"ok": "false", "error": msg})
		return
	}
	if req.APIKey != "" {
		privacy.Register(req.APIKey)
	}
	client, err := tools.NewLLMClientWith(tools.LLMSettings{Provider: req.Provider, Model: req.Model, APIKey: req.APIKey, OllamaEndpoint: req.OllamaEndpoint})
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": scrub(err.Error(), req.APIKey)})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	_, err = client.Call(ctx, &tools.LLMRequest{SystemPrompt: "Reply with the single word OK.", UserPrompt: "Test.", MaxTokens: 8, Temperature: 0, Retries: 1})
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": scrub(err.Error(), req.APIKey)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// GET /api/settings/llm/ollama-models?endpoint=http://127.0.0.1:11434 lists the models of an Ollama server. The backend asks
// (the extension cannot: Ollama refuses browser extension origins).
func (m *llmManager) handleOllamaModels(w http.ResponseWriter, r *http.Request) {
	endpoint := strings.TrimSpace(r.URL.Query().Get("endpoint"))
	if endpoint == "" {
		endpoint = m.get(keyEndpoint)
	}
	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434"
	}
	if msg := validCandidate("ollama", "", endpoint, ""); msg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(endpoint, "/")+"/api/tags", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"models": []string{}, "error": "Ollama did not answer at " + endpoint})
		return
	}
	defer resp.Body.Close()
	var body struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(http.MaxBytesReader(w, resp.Body, 1<<20)).Decode(&body) != nil {
		writeJSON(w, http.StatusOK, map[string]any{"models": []string{}, "error": "Ollama gave an unexpected answer"})
		return
	}
	names := []string{}
	for _, mod := range body.Models {
		names = append(names, mod.Name)
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": names})
}

func (m *llmManager) register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/settings/llm/ollama-models", m.handleOllamaModels)
	mux.HandleFunc("GET /api/settings/llm", m.handleGet)
	mux.HandleFunc("PUT /api/settings/llm", m.handlePut)
	mux.HandleFunc("POST /api/settings/llm/test", m.handleTest)
	mux.HandleFunc("POST /api/settings/llm/unlock", m.handleUnlock)
	mux.HandleFunc("POST /api/settings/llm/lock", m.handleLock)
	mux.HandleFunc("DELETE /api/settings/llm/key", m.handleDeleteKey)
}

