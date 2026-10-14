package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"moly/auth"
	"moly/database"
	"moly/privacy"
	"moly/secrets"
	"moly/tools"
)

// The AI provider, model and API key belong to a USER. They are saved in the encrypted database (tables user_ai_settings and
// user_ai_keys, package database), and this manager is the ONE place that reads them, unlocks them and turns them into a
// client. Every model call of the program gets its provider from here, through For(userID) (tools.LLMSource): a call made for
// one user can never use another user's key, and a call with no user fails.
//
// A key can be sealed with a passphrase (package secrets): then only the passphrase opens it, and it is held in memory (inside
// the user's client) only after that user unlocks it. A key is never returned by any endpoint, never logged and never exported.
//
// The only exception is the operator's override: when LLM_PROVIDER is set in the environment, the person who started the
// program has chosen the provider (a developer, or a test run) and that client serves every user of that run. Saved settings
// are then ignored and the settings endpoints answer 409.

const claimLegacyFor = auth.LocalProfileID // the one person of an installed Moly may take the settings that older versions saved for the installation

var (
	cloudProviders = map[string]bool{"claude": true, "openai": true}
	allProviders   = map[string]bool{"ollama": true, "claude": true, "openai": true}
	modelPattern   = regexp.MustCompile(`^[A-Za-z0-9._:/\-]{1,100}$`)
)

const (
	stateReady        = "ready"
	stateLocked       = "locked"       // a sealed key is saved and not yet unlocked
	stateUnconfigured = "unconfigured" // nothing usable yet
)

// userAI is what the manager knows about one user while the program runs.
type userAI struct {
	mu      sync.Mutex
	loaded  bool
	state   string
	lastErr string
	client  *tools.LLMClient
	failed  int // wrong passphrases in a row
}

type llmManager struct {
	mu        sync.Mutex // guards users
	users     map[string]*userAI
	store     *database.UserAI
	authUser  func(*http.Request) (string, error)
	envClient *tools.LLMClient // the operator's override; nil when LLM_PROVIDER is not set
	envFixed  bool
	// listModels asks a provider which models it offers (a field so that tests need no network)
	listModels func(context.Context, tools.LLMSettings) (*tools.ModelList, error)
}

// newLLMManager makes the manager. authUser says which user a request is from (a valid session), or an error. When the
// environment fixes the provider, its client is built now and an error is returned if it cannot be.
func newLLMManager(db *database.Database, authUser func(*http.Request) (string, error), operatorProvider string) (*llmManager, error) {
	m := &llmManager{users: map[string]*userAI{}, store: db.GetUserAI(), authUser: authUser, envFixed: strings.TrimSpace(operatorProvider) != "", listModels: tools.ListModels}
	if m.envFixed {
		c, err := tools.NewLLMClientFromEnv()
		if err != nil {
			return nil, err
		}
		m.envClient = c
	}
	return m, nil
}

// For returns the provider of one user (tools.LLMSource).
func (m *llmManager) For(userID string) tools.LLMProvider { return boundLLM{m: m, userID: userID} }

// Unbound returns a provider that fails every call (tools.LLMSource).
func (m *llmManager) Unbound() tools.LLMProvider { return tools.UnboundLLM() }

// Forget drops everything held in memory for a user, the unlocked key included. Called when the profile is deleted: a profile
// created later with the same id must not inherit it.
func (m *llmManager) Forget(userID string) {
	m.mu.Lock()
	delete(m.users, userID)
	m.mu.Unlock()
}

// boundLLM is the provider of one user. It resolves the user's current client at every call, so settings that change, a key
// that is locked or removed, take effect at once, and it keeps no key itself.
type boundLLM struct {
	m      *llmManager
	userID string
}

func (b boundLLM) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	c, err := b.m.clientFor(b.userID)
	if err != nil {
		return nil, err
	}
	return c.Call(ctx, req)
}

// clientFor returns the client a model call for this user must use.
func (m *llmManager) clientFor(userID string) (tools.LLMProvider, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, tools.ErrNoUser
	}
	if m.envClient != nil {
		return m.envClient, nil
	}
	u := m.user(userID)
	u.mu.Lock()
	defer u.mu.Unlock()
	m.loadLocked(userID, u)
	if u.client == nil {
		return nil, tools.ErrNoProvider
	}
	return u.client, nil
}

func (m *llmManager) user(userID string) *userAI {
	m.mu.Lock()
	defer m.mu.Unlock()
	u := m.users[userID]
	if u == nil {
		u = &userAI{}
		m.users[userID] = u
	}
	return u
}

// loadLocked reads the user's saved settings the first time they are needed (u.mu held).
func (m *llmManager) loadLocked(userID string, u *userAI) {
	if u.loaded {
		return
	}
	u.loaded = true
	if userID == claimLegacyFor {
		if moved, err := m.store.ClaimLegacy(userID); err != nil {
			u.state, u.lastErr = stateUnconfigured, "The settings saved by an older version could not be moved: "+err.Error()
			return
		} else if moved {
			log.Printf("[Moly] AI settings saved for the whole installation were moved to the local profile")
		}
	}
	m.applyLocked(userID, u, "")
}

// applyLocked builds the user's client from the user's saved settings (u.mu held). unlockedKey, when set, is the plain key of
// the user's provider.
func (m *llmManager) applyLocked(userID string, u *userAI, unlockedKey string) {
	u.client = nil
	settings, err := m.store.Settings(userID)
	if err != nil {
		u.state, u.lastErr = stateUnconfigured, "The AI settings could not be read."
		return
	}
	s := tools.LLMSettings{Provider: settings.Provider, Model: settings.Model, OllamaEndpoint: settings.OllamaEndpoint}
	if cloudProviders[s.Provider] {
		stored, has, err := m.store.Key(userID, s.Provider)
		switch {
		case err != nil:
			u.state, u.lastErr = stateUnconfigured, "The saved API key could not be read."
			return
		case !has:
			u.state, u.lastErr = stateUnconfigured, "No API key is saved for "+s.Provider+"."
			return
		case unlockedKey != "":
			s.APIKey = unlockedKey
		case secrets.IsSealed(stored):
			u.state, u.lastErr = stateLocked, "The API key is protected by a passphrase: unlock it."
			return
		default:
			plain, ok := secrets.FromPlain(stored)
			if !ok {
				u.state, u.lastErr = stateUnconfigured, "The saved API key is unreadable; enter it again."
				return
			}
			s.APIKey = plain
		}
	}
	if s.APIKey != "" {
		privacy.Register(s.APIKey) // masked in every log line from now on
	}
	client, err := tools.NewLLMClientWith(s)
	if err != nil {
		u.state, u.lastErr = stateUnconfigured, scrub(err.Error(), s.APIKey)
		return
	}
	u.state, u.lastErr, u.client = stateReady, "", client
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

// statusLocked describes one user's settings (u.mu held). It never holds a key.
func (m *llmManager) statusLocked(userID string, u *userAI) llmStatus {
	m.loadLocked(userID, u)
	st := llmStatus{State: u.state, Error: u.lastErr, Active: "none", FixedByEnv: m.envFixed, Keys: map[string]keyInfo{}}
	if m.envClient != nil {
		st.Provider, st.Model, st.State, st.Error, st.Active = m.envClient.Provider, m.envClient.Model, stateReady, "", m.envClient.Provider
		return st
	}
	if s, err := m.store.Settings(userID); err == nil {
		st.Provider, st.Model, st.OllamaEndpoint = s.Provider, s.Model, s.OllamaEndpoint
	}
	if u.client != nil {
		st.Active = u.client.Provider
	}
	for p := range cloudProviders {
		v, has, _ := m.store.Key(userID, p)
		st.Keys[p] = keyInfo{Stored: has, Sealed: has && secrets.IsSealed(v)}
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

// forUser runs a handler for the user the request is from, and for nobody else: no valid session, no answer.
func (m *llmManager) forUser(h func(w http.ResponseWriter, r *http.Request, userID string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := m.authUser(r)
		if err != nil || strings.TrimSpace(userID) == "" {
			log.Printf("[Settings] %s %s refused: the request carries no valid user session", r.Method, r.URL.Path)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
			return
		}
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		h(rec, r, userID)
		if r.Method != http.MethodGet || rec.code >= 400 { // the extension polls GET every 15 seconds: only changes and failures are logged
			log.Printf("[Settings] %s %s -> %d", r.Method, r.URL.Path, rec.code)
		}
	}
}

// statusRecorder remembers the status a handler wrote, so that it can be logged (never the body: it may hold a key).
type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}

func (m *llmManager) fixedByEnv(w http.ResponseWriter) bool {
	if m.envFixed {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "the provider is fixed by the LLM_PROVIDER environment variable"})
		return true
	}
	return false
}

// GET /api/settings/llm
func (m *llmManager) handleGet(w http.ResponseWriter, r *http.Request, userID string) {
	u := m.user(userID)
	u.mu.Lock()
	defer u.mu.Unlock()
	writeJSON(w, http.StatusOK, m.statusLocked(userID, u))
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
func (m *llmManager) handlePut(w http.ResponseWriter, r *http.Request, userID string) {
	if m.fixedByEnv(w) {
		return
	}
	var req llmPut
	if !readBody(w, r, &req) {
		return
	}
	req.APIKey = strings.TrimSpace(req.APIKey)

	u := m.user(userID)
	u.mu.Lock()
	defer u.mu.Unlock()
	m.loadLocked(userID, u)
	cur, err := m.store.Settings(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "the settings could not be read"})
		return
	}
	provider, model, endpoint := cur.Provider, cur.Model, cur.OllamaEndpoint
	if req.Provider != nil {
		provider = strings.ToLower(strings.TrimSpace(*req.Provider))
	}
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
		stored, _, _ := m.store.Key(userID, provider)
		plain, ok := secrets.FromPlain(stored)
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

	if err := m.store.SaveSettings(userID, database.UserAISettings{Provider: provider, Model: model, OllamaEndpoint: endpoint}); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "the settings could not be saved"})
		return
	}
	if toStore != "" {
		if err := m.store.SaveKey(userID, provider, toStore); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "the key could not be saved"})
			return
		}
	}
	m.applyLocked(userID, u, plainForApply)
	writeJSON(w, http.StatusOK, m.statusLocked(userID, u))
}

// POST /api/settings/llm/unlock
func (m *llmManager) handleUnlock(w http.ResponseWriter, r *http.Request, userID string) {
	if m.fixedByEnv(w) {
		return
	}
	var req struct {
		Passphrase string `json:"passphrase"`
	}
	if !readBody(w, r, &req) {
		return
	}
	u := m.user(userID)
	u.mu.Lock()
	defer u.mu.Unlock()
	m.loadLocked(userID, u)
	cur, _ := m.store.Settings(userID)
	stored, _, _ := m.store.Key(userID, cur.Provider)
	if !cloudProviders[cur.Provider] || !secrets.IsSealed(stored) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "there is no passphrase-protected key to unlock"})
		return
	}
	key, err := secrets.Open(req.Passphrase, stored)
	if err != nil {
		u.failed++
		delay := u.failed
		if delay > 5 {
			delay = 5
		}
		time.Sleep(time.Duration(delay) * time.Second) // slows down guessing
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "wrong passphrase"})
		return
	}
	u.failed = 0
	m.applyLocked(userID, u, key)
	writeJSON(w, http.StatusOK, m.statusLocked(userID, u))
}

// POST /api/settings/llm/lock forgets the user's unlocked key (it stays saved, sealed).
func (m *llmManager) handleLock(w http.ResponseWriter, r *http.Request, userID string) {
	u := m.user(userID)
	u.mu.Lock()
	defer u.mu.Unlock()
	m.loadLocked(userID, u)
	if !m.envFixed {
		cur, _ := m.store.Settings(userID)
		if stored, _, _ := m.store.Key(userID, cur.Provider); cloudProviders[cur.Provider] && secrets.IsSealed(stored) {
			m.applyLocked(userID, u, "")
		}
	}
	writeJSON(w, http.StatusOK, m.statusLocked(userID, u))
}

// DELETE /api/settings/llm/key?provider=claude
func (m *llmManager) handleDeleteKey(w http.ResponseWriter, r *http.Request, userID string) {
	if m.fixedByEnv(w) {
		return
	}
	provider := strings.ToLower(r.URL.Query().Get("provider"))
	if !cloudProviders[provider] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider must be claude or openai"})
		return
	}
	u := m.user(userID)
	u.mu.Lock()
	defer u.mu.Unlock()
	m.loadLocked(userID, u)
	if err := m.store.DeleteKey(userID, provider); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "the key could not be removed"})
		return
	}
	m.applyLocked(userID, u, "")
	writeJSON(w, http.StatusOK, m.statusLocked(userID, u))
}

// POST /api/settings/llm/test tries a provider with a tiny request and saves nothing. The key is the one in the request.
func (m *llmManager) handleTest(w http.ResponseWriter, r *http.Request, userID string) {
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
func (m *llmManager) handleOllamaModels(w http.ResponseWriter, r *http.Request, userID string) {
	endpoint := strings.TrimSpace(r.URL.Query().Get("endpoint"))
	if endpoint == "" {
		if s, err := m.store.Settings(userID); err == nil {
			endpoint = s.OllamaEndpoint
		}
	}
	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434"
	}
	if msg := validCandidate("ollama", "", endpoint, ""); msg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	list, err := m.listModels(r.Context(), tools.LLMSettings{Provider: "ollama", OllamaEndpoint: endpoint})
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"models": []string{}, "error": err.Error()})
		return
	}
	names := []string{}
	for _, mod := range list.Models {
		names = append(names, mod.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": names})
}

// POST /api/settings/llm/models lists the models a cloud provider offers to the user's key: the key in the request when the
// form has a new one, else the key the user saved (it must be unlocked). The key is used for this one question and is not
// stored, returned or logged. The answer carries the models and the one to preselect.
func (m *llmManager) handleModels(w http.ResponseWriter, r *http.Request, userID string) {
	var req struct {
		Provider string `json:"provider"`
		APIKey   string `json:"apiKey"`
	}
	if !readBody(w, r, &req) {
		return
	}
	provider, key := strings.ToLower(strings.TrimSpace(req.Provider)), strings.TrimSpace(req.APIKey)
	if !cloudProviders[provider] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider must be claude or openai"})
		return
	}
	if msg := validCandidate(provider, "", "", key); msg != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
		return
	}
	if key == "" {
		u := m.user(userID)
		u.mu.Lock()
		stored, has, _ := m.store.Key(userID, provider)
		switch {
		case !has:
			u.mu.Unlock()
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "enter your API key to see the models"})
			return
		case secrets.IsSealed(stored):
			if u.client != nil && u.client.Provider == provider {
				key = u.client.ApiKey // unlocked
			}
		default:
			key, _ = secrets.FromPlain(stored)
		}
		u.mu.Unlock()
		if key == "" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "the saved key is locked: unlock it to see the models"})
			return
		}
	}
	privacy.Register(key)
	list, err := m.listModels(r.Context(), tools.LLMSettings{Provider: provider, APIKey: key})
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"models": []tools.ModelInfo{}, "error": scrub(err.Error(), key)})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// register adds the settings endpoints. Every one of them acts on the user the request is from, and on nobody else.
func (m *llmManager) register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/settings/llm/ollama-models", m.forUser(m.handleOllamaModels))
	mux.HandleFunc("GET /api/settings/llm", m.forUser(m.handleGet))
	mux.HandleFunc("PUT /api/settings/llm", m.forUser(m.handlePut))
	mux.HandleFunc("POST /api/settings/llm/test", m.forUser(m.handleTest))
	mux.HandleFunc("POST /api/settings/llm/models", m.forUser(m.handleModels))
	mux.HandleFunc("POST /api/settings/llm/unlock", m.forUser(m.handleUnlock))
	mux.HandleFunc("POST /api/settings/llm/lock", m.forUser(m.handleLock))
	mux.HandleFunc("DELETE /api/settings/llm/key", m.forUser(m.handleDeleteKey))
}

