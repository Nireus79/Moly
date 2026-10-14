package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"moly/database"
	"moly/tools"
)

const (
	fakeKey  = "sk-ant-TEST-KEY-must-never-be-returned-0123456789"
	otherKey = "sk-ant-OTHER-USERS-KEY-must-stay-theirs-9876543210"
)

// testAuth says which user a request is from by a header, standing in for the session token.
func testAuth(r *http.Request) (string, error) {
	if u := r.Header.Get("X-Test-User"); u != "" {
		return u, nil
	}
	return "", fmt.Errorf("Invalid token")
}

// testUser makes a user in the shared test database; their settings go with them when the test ends.
func testUser(t *testing.T, db *database.Database, label string) string {
	t.Helper()
	id := fmt.Sprintf("user_%s_%d", label, time.Now().UnixNano())
	now := time.Now().Unix()
	if _, err := db.GetConnection().Exec(`INSERT INTO users (id, created_at, last_active) VALUES (?, ?, ?)`, id, now, now); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.GetConnection().Exec(`DELETE FROM users WHERE id = ?`, id) })
	return id
}

func newTestLLMManager(t *testing.T) (*llmManager, *database.Database, http.Handler) {
	t.Helper()
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	db, err := database.Init(testDBPath)
	if err != nil {
		t.Fatal(err)
	}
	m, err := newLLMManager(db, testAuth, "")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	m.register(mux)
	return m, db, mux
}

// do sends a request as a user ("" is no session at all).
func do(h http.Handler, user, method, target string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, target, &buf)
	if user != "" {
		req.Header.Set("X-Test-User", user)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
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

func callAs(m *llmManager, user string) error {
	_, err := m.For(user).Call(context.Background(), &tools.LLMRequest{SystemPrompt: "s", UserPrompt: "u"})
	return err
}

func TestLLMSettingsStorePlainKeyAndNeverReturnIt(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	u := testUser(t, db, "plain")
	rec := do(h, u, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "claude-sonnet-5-5", "apiKey": fakeKey})
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
	if get := do(h, u, "GET", "/api/settings/llm", nil); strings.Contains(get.Body.String(), fakeKey) {
		t.Fatal("the GET answer contains the API key")
	}
	raw, _, _ := m.store.Key(u, "claude")
	if raw != "plain:"+fakeKey {
		t.Fatalf("stored value = %q", raw)
	}
}

func TestLLMSettingsSealedKeyNeedsThePassphraseAfterRestart(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	u := testUser(t, db, "sealed")
	rec := do(h, u, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "claude-sonnet-5-5", "apiKey": fakeKey, "passphrase": "correct horse battery"})
	if st := statusOf(t, rec); rec.Code != 200 || st.State != stateReady || !st.Keys["claude"].Sealed {
		t.Fatalf("sealing failed: %d %+v", rec.Code, st)
	}
	if raw, _, _ := m.store.Key(u, "claude"); strings.Contains(raw, fakeKey) || !strings.HasPrefix(raw, "sealed:v1:") {
		t.Fatalf("the saved value must be sealed, got %q", raw)
	}

	// A restart: a new manager over the same database knows nothing but the sealed value.
	m2, err := newLLMManager(db, testAuth, "")
	if err != nil {
		t.Fatal(err)
	}
	mux2 := http.NewServeMux()
	m2.register(mux2)
	if st := statusOf(t, do(mux2, u, "GET", "/api/settings/llm", nil)); st.State != stateLocked || st.Active != "none" {
		t.Fatalf("after a restart the key must be locked: %+v", st)
	}
	if err := callAs(m2, u); err != tools.ErrNoProvider {
		t.Fatalf("a locked key must refuse to call the provider: %v", err)
	}

	if rec := do(mux2, u, "POST", "/api/settings/llm/unlock", map[string]string{"passphrase": "not the passphrase"}); rec.Code != http.StatusForbidden {
		t.Fatalf("wrong passphrase: %d", rec.Code)
	}
	if st := statusOf(t, do(mux2, u, "GET", "/api/settings/llm", nil)); st.State != stateLocked {
		t.Fatal("a wrong passphrase must leave it locked")
	}
	rec = do(mux2, u, "POST", "/api/settings/llm/unlock", map[string]string{"passphrase": "correct horse battery"})
	if st := statusOf(t, rec); rec.Code != 200 || st.State != stateReady || st.Active != "claude" || strings.Contains(rec.Body.String(), fakeKey) {
		t.Fatalf("unlock failed or leaked: %d %s", rec.Code, rec.Body.String())
	}

	do(mux2, u, "POST", "/api/settings/llm/lock", nil)
	if st := statusOf(t, do(mux2, u, "GET", "/api/settings/llm", nil)); st.State != stateLocked || st.Active != "none" {
		t.Fatal("lock must forget the key")
	}
}

func TestLLMSettingsSealAnExistingPlainKeyLater(t *testing.T) {
	_, db, h := newTestLLMManager(t)
	u := testUser(t, db, "later")
	do(h, u, "PUT", "/api/settings/llm", map[string]any{"provider": "openai", "model": "gpt-4o", "apiKey": fakeKey})
	rec := do(h, u, "PUT", "/api/settings/llm", map[string]any{"passphrase": "correct horse battery"})
	if st := statusOf(t, rec); rec.Code != 200 || !st.Keys["openai"].Sealed {
		t.Fatalf("sealing the stored key failed: %d %s", rec.Code, rec.Body.String())
	}
}

func TestLLMSettingsValidation(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	u := testUser(t, db, "valid")
	cases := []map[string]any{
		{"provider": "skynet"},
		{"provider": "claude", "apiKey": "has space"},
		{"provider": "claude", "model": "bad model!"},
		{"provider": "ollama", "ollamaEndpoint": "file:///etc/passwd"},
		{"provider": "ollama", "apiKey": "keys are for cloud providers"},
		{"provider": "claude", "apiKey": fakeKey, "passphrase": "short"},
	}
	for _, c := range cases {
		if rec := do(h, u, "PUT", "/api/settings/llm", c); rec.Code != http.StatusBadRequest {
			t.Errorf("%v: want 400, got %d %s", c, rec.Code, rec.Body.String())
		}
	}
	if s, _ := m.store.Settings(u); s.Provider != "" {
		t.Errorf("a rejected request must save nothing, provider is %q", s.Provider)
	}
}

func TestLLMSettingsCloudProviderWithoutKeyIsUnconfigured(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	u := testUser(t, db, "nokey")
	rec := do(h, u, "PUT", "/api/settings/llm", map[string]any{"provider": "claude"})
	if st := statusOf(t, rec); st.State != stateUnconfigured || st.Active != "none" {
		t.Fatalf("no key must mean unconfigured: %+v", st)
	}
	if err := callAs(m, u); err != tools.ErrNoProvider {
		t.Fatalf("no key, no call: %v", err)
	}
}

func TestLLMSettingsDeleteKey(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	u := testUser(t, db, "delete")
	do(h, u, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "claude-sonnet-5-5", "apiKey": fakeKey})
	rec := do(h, u, "DELETE", "/api/settings/llm/key?provider=claude", nil)
	if st := statusOf(t, rec); st.Keys["claude"].Stored || st.State != stateUnconfigured || st.Active != "none" {
		t.Fatalf("key not removed: %+v", st)
	}
	if _, ok, _ := m.store.Key(u, "claude"); ok {
		t.Fatal("the key is still in the database")
	}
}

// The operator's override (LLM_PROVIDER) serves every user and makes the settings read-only.
func TestLLMSettingsEnvironmentWins(t *testing.T) {
	_, db, _ := newTestLLMManager(t)
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"models":[{"name":"local-model"}]}`)) }))
	defer fake.Close()
	t.Setenv("LLM_PROVIDER", "ollama")
	t.Setenv("OLLAMA_ENDPOINT", fake.URL)
	m, err := newLLMManager(db, testAuth, "ollama")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	m.register(mux)
	u := testUser(t, db, "env")
	if rec := do(mux, u, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "claude-sonnet-5-5", "apiKey": fakeKey}); rec.Code != http.StatusConflict {
		t.Fatalf("with LLM_PROVIDER set the settings must be read-only, got %d", rec.Code)
	}
	if st := statusOf(t, do(mux, u, "GET", "/api/settings/llm", nil)); !st.FixedByEnv || st.Active != "ollama" {
		t.Fatalf("the operator's provider serves the user: %+v", st)
	}
}

func TestLLMSettingsTestEndpointReportsFailureWithoutSaving(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	u := testUser(t, db, "testep")
	rec := do(h, u, "POST", "/api/settings/llm/test", map[string]any{"provider": "ollama", "ollamaEndpoint": "http://127.0.0.1:1"})
	var out map[string]any
	json.Unmarshal(rec.Body.Bytes(), &out)
	if out["ok"] != false || out["error"] == "" {
		t.Fatalf("an unreachable Ollama must be reported: %s", rec.Body.String())
	}
	if s, _ := m.store.Settings(u); s.Provider != "" {
		t.Fatal("the test endpoint must save nothing")
	}
}

func TestScrubRemovesAKeyFromAMessage(t *testing.T) {
	if got := scrub("401 invalid x-api-key "+fakeKey, fakeKey); strings.Contains(got, fakeKey) {
		t.Fatalf("scrub left the key: %s", got)
	}
}

func TestOllamaModelsListsTheModelsOfAServer(t *testing.T) {
	_, db, h := newTestLLMManager(t)
	u := testUser(t, db, "models")
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"models":[{"name":"mistral:latest"},{"name":"llama3:8b"}]}`))
	}))
	defer fake.Close()
	rec := do(h, u, "GET", "/api/settings/llm/ollama-models?endpoint="+fake.URL, nil)
	var out struct {
		Models []string `json:"models"`
		Error  string   `json:"error"`
	}
	json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Models) != 2 || out.Models[0] != "mistral:latest" || out.Error != "" {
		t.Fatalf("models = %+v", out)
	}
	rec = do(h, u, "GET", "/api/settings/llm/ollama-models?endpoint=http://127.0.0.1:1", nil)
	json.Unmarshal(rec.Body.Bytes(), &out)
	if len(out.Models) != 0 || out.Error == "" {
		t.Fatalf("an unreachable server must be reported: %s", rec.Body.String())
	}
	if rec := do(h, u, "GET", "/api/settings/llm/ollama-models?endpoint=file:///etc", nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("a non-http endpoint must be refused, got %d", rec.Code)
	}
}

// ---- The key belongs to the user ----

// Another user cannot read, use, replace, unlock or delete a user's key.
func TestAUsersKeyIsNotAnotherUsersKey(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	a, b := testUser(t, db, "a"), testUser(t, db, "b")
	do(h, a, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "claude-sonnet-5-5", "apiKey": fakeKey, "passphrase": "correct horse battery"})
	do(h, b, "PUT", "/api/settings/llm", map[string]any{"provider": "openai", "model": "gpt-4o", "apiKey": otherKey})

	// B sees only B's settings
	stB := statusOf(t, do(h, b, "GET", "/api/settings/llm", nil))
	if stB.Provider != "openai" || stB.Keys["claude"].Stored || !stB.Keys["openai"].Stored {
		t.Fatalf("B sees something that is not B's: %+v", stB)
	}
	// B cannot unlock A's sealed key with A's passphrase: B has no sealed key
	if rec := do(h, b, "POST", "/api/settings/llm/unlock", map[string]string{"passphrase": "correct horse battery"}); rec.Code != http.StatusBadRequest {
		t.Fatalf("B must have nothing to unlock, got %d", rec.Code)
	}
	// B deleting "claude" deletes nothing of A's
	do(h, b, "DELETE", "/api/settings/llm/key?provider=claude", nil)
	if _, ok, _ := m.store.Key(a, "claude"); !ok {
		t.Fatal("B removed A's key")
	}
	// B's calls use B's provider, A's use A's; each client holds only its own key
	m.mu.Lock()
	ca, cb := m.users[a].client, m.users[b].client
	m.mu.Unlock()
	if ca == nil || ca.ApiKey != fakeKey || ca.Provider != "claude" {
		t.Fatalf("A's client: %+v", ca)
	}
	if cb == nil || cb.ApiKey != otherKey || cb.Provider != "openai" {
		t.Fatalf("B's client: %+v", cb)
	}
	// one user changing their settings changes nothing for the other
	do(h, b, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "claude-sonnet-5-5", "apiKey": "sk-ant-B-NEW-KEY"})
	if k, _, _ := m.store.Key(a, "claude"); !strings.HasPrefix(k, "sealed:") {
		t.Fatalf("A's key was replaced by B: %q", k)
	}
}

// A user who has no key gets no key: the key of another user is never a fallback.
func TestAUserWithoutAKeyNeverUsesAnotherUsersKey(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	a, b := testUser(t, db, "owner"), testUser(t, db, "nokey")
	do(h, a, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "claude-sonnet-5-5", "apiKey": fakeKey})
	if rec := do(h, b, "PUT", "/api/settings/llm", map[string]any{"provider": "claude"}); statusOf(t, rec).State != stateUnconfigured {
		t.Fatalf("B has no key: %s", rec.Body.String())
	}
	if err := callAs(m, b); err != tools.ErrNoProvider {
		t.Fatalf("B must not be served with A's key: %v", err)
	}
}

// No user, no provider.
func TestACallWithoutAUserIsRefused(t *testing.T) {
	m, _, _ := newTestLLMManager(t)
	if err := callAs(m, ""); err != tools.ErrNoUser {
		t.Fatalf("a call with no user: %v", err)
	}
	if _, err := m.Unbound().Call(context.Background(), &tools.LLMRequest{}); err != tools.ErrNoUser {
		t.Fatalf("the provider of nobody: %v", err)
	}
}

// Every settings endpoint needs a valid session.
func TestTheSettingsEndpointsNeedASession(t *testing.T) {
	_, _, h := newTestLLMManager(t)
	for _, c := range []struct{ method, path string }{
		{"GET", "/api/settings/llm"}, {"PUT", "/api/settings/llm"}, {"POST", "/api/settings/llm/unlock"}, {"POST", "/api/settings/llm/lock"},
		{"POST", "/api/settings/llm/test"}, {"POST", "/api/settings/llm/models"}, {"DELETE", "/api/settings/llm/key?provider=claude"}, {"GET", "/api/settings/llm/ollama-models"},
	} {
		if rec := do(h, "", c.method, c.path, map[string]any{}); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a session: %d", c.method, c.path, rec.Code)
		}
	}
}

// The machine's environment cannot replace a user's key, nor give a user one they did not save.
func TestTheEnvironmentNeverReplacesOrSuppliesAUsersKey(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-FROM-THE-ENVIRONMENT")
	t.Setenv("AGENT_MODEL", "some-environment-model")
	a, b := testUser(t, db, "envkey"), testUser(t, db, "envnokey")
	do(h, a, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "claude-sonnet-5-5", "apiKey": fakeKey})
	m.mu.Lock()
	ca := m.users[a].client
	m.mu.Unlock()
	if ca == nil || ca.ApiKey != fakeKey || ca.Model != "claude-sonnet-5-5" {
		t.Fatalf("the environment replaced the user's key or model: %+v", ca)
	}
	do(h, b, "PUT", "/api/settings/llm", map[string]any{"provider": "claude"})
	if err := callAs(m, b); err != tools.ErrNoProvider {
		t.Fatalf("the environment's key was handed to a user who saved none: %v", err)
	}
}

// Settings saved for the installation by older versions go to the local profile once, and to nobody else.
func TestLegacyInstallationSettingsGoToTheLocalProfileOnly(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	local := claimLegacyFor
	now := time.Now().Unix()
	conn := db.GetConnection()
	conn.Exec(`INSERT OR IGNORE INTO users (id, created_at, last_active) VALUES (?, ?, ?)`, local, now, now)
	conn.Exec(`DELETE FROM user_ai_settings WHERE user_id = ?`, local)
	conn.Exec(`DELETE FROM user_ai_keys WHERE user_id = ?`, local)
	t.Cleanup(func() {
		conn.Exec(`DELETE FROM user_ai_settings WHERE user_id = ?`, local)
		conn.Exec(`DELETE FROM user_ai_keys WHERE user_id = ?`, local)
		conn.Exec(`DELETE FROM app_settings WHERE key LIKE 'llm.%'`)
	})
	legacy := db.GetAppSettings()
	legacy.Set("llm.provider", "claude")
	legacy.Set("llm.model", "claude-sonnet-5-5")
	legacy.Set("llm.api_key.claude", "plain:"+fakeKey)

	other := testUser(t, db, "notlocal")
	if st := statusOf(t, do(h, other, "GET", "/api/settings/llm", nil)); st.Provider != "" || st.Keys["claude"].Stored {
		t.Fatalf("another user was given the installation's settings: %+v", st)
	}
	if _, ok, _ := legacy.Get("llm.api_key.claude"); !ok {
		t.Fatal("the installation's key must stay until the local profile claims it")
	}
	st := statusOf(t, do(h, local, "GET", "/api/settings/llm", nil))
	if st.Provider != "claude" || !st.Keys["claude"].Stored || st.State != stateReady {
		t.Fatalf("the local profile did not get the settings of the installation: %+v", st)
	}
	if _, ok, _ := legacy.Get("llm.api_key.claude"); ok {
		t.Fatal("a claimed key must leave app_settings")
	}
	m.Forget(local) // a restart: the settings are the local profile's now, and nobody can claim them again
	if st := statusOf(t, do(h, local, "GET", "/api/settings/llm", nil)); st.Provider != "claude" {
		t.Fatalf("the claimed settings were lost: %+v", st)
	}
}

// A deleted profile's unlocked key leaves memory, so a profile created later with the same id does not inherit it.
func TestForgetDropsAnUnlockedKey(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	u := testUser(t, db, "forget")
	do(h, u, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "claude-sonnet-5-5", "apiKey": fakeKey, "passphrase": "correct horse battery"})
	if _, err := m.clientFor(u); err != nil { // resolves the client without calling the provider
		t.Fatalf("setup: the key should be unlocked, got %v", err)
	}
	m.Forget(u)
	if err := callAs(m, u); err != tools.ErrNoProvider {
		t.Fatalf("after Forget the sealed key is locked again: %v", err)
	}
}

// ---- Which models are available is asked of the provider, with the user's own key ----

func stubModels(m *llmManager, asked *[]tools.LLMSettings) {
	m.listModels = func(ctx context.Context, s tools.LLMSettings) (*tools.ModelList, error) {
		*asked = append(*asked, s)
		return &tools.ModelList{Models: []tools.ModelInfo{{ID: "model-1", Label: "Model 1"}, {ID: "model-2", Label: "Model 2"}}, Suggested: "model-2"}, nil
	}
}

func TestModelsAreListedWithTheKeyOfTheRequestOrTheSavedKey(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	var asked []tools.LLMSettings
	stubModels(m, &asked)
	u := testUser(t, db, "models1")

	// a key typed in the form, not saved yet
	rec := do(h, u, "POST", "/api/settings/llm/models", map[string]string{"provider": "claude", "apiKey": otherKey})
	var got tools.ModelList
	json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != 200 || len(got.Models) != 2 || got.Suggested != "model-2" || len(asked) != 1 || asked[0].APIKey != otherKey {
		t.Fatalf("typed key: %d %s asked=%+v", rec.Code, rec.Body.String(), asked)
	}
	if strings.Contains(rec.Body.String(), otherKey) {
		t.Fatal("the key is in the answer")
	}
	if _, has, _ := m.store.Key(u, "claude"); has {
		t.Fatal("listing the models must not save the typed key")
	}

	// no key typed: the saved one is used
	do(h, u, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "model-1", "apiKey": fakeKey})
	asked = nil
	do(h, u, "POST", "/api/settings/llm/models", map[string]string{"provider": "claude"})
	if len(asked) != 1 || asked[0].APIKey != fakeKey {
		t.Fatalf("the saved key should be used, asked=%+v", asked)
	}
}

func TestModelsOfASealedKeyNeedItUnlocked(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	var asked []tools.LLMSettings
	stubModels(m, &asked)
	u := testUser(t, db, "models2")
	do(h, u, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "model-1", "apiKey": fakeKey, "passphrase": "correct horse battery"})

	// unlocked right after saving: usable
	if rec := do(h, u, "POST", "/api/settings/llm/models", map[string]string{"provider": "claude"}); rec.Code != 200 || len(asked) != 1 || asked[0].APIKey != fakeKey {
		t.Fatalf("unlocked: %d %s", rec.Code, rec.Body.String())
	}
	// after a restart it is locked: the answer says so and nothing is asked of the provider
	m2, err := newLLMManager(db, testAuth, "")
	if err != nil {
		t.Fatal(err)
	}
	var asked2 []tools.LLMSettings
	stubModels(m2, &asked2)
	mux2 := http.NewServeMux()
	m2.register(mux2)
	rec := do(mux2, u, "POST", "/api/settings/llm/models", map[string]string{"provider": "claude"})
	if rec.Code != http.StatusConflict || len(asked2) != 0 || strings.Contains(rec.Body.String(), fakeKey) {
		t.Fatalf("locked: %d %s asked=%+v", rec.Code, rec.Body.String(), asked2)
	}
}

// Another user's saved key is never used to list models for me.
func TestModelsNeverUseAnotherUsersKey(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	var asked []tools.LLMSettings
	stubModels(m, &asked)
	a, b := testUser(t, db, "mowner"), testUser(t, db, "mother")
	do(h, a, "PUT", "/api/settings/llm", map[string]any{"provider": "claude", "model": "model-1", "apiKey": fakeKey})
	rec := do(h, b, "POST", "/api/settings/llm/models", map[string]string{"provider": "claude"})
	if rec.Code != http.StatusBadRequest || len(asked) != 0 {
		t.Fatalf("B has no key and must get none of A's: %d %s asked=%+v", rec.Code, rec.Body.String(), asked)
	}
}

func TestModelsListFailureIsReportedWithoutTheKey(t *testing.T) {
	m, db, h := newTestLLMManager(t)
	m.listModels = func(ctx context.Context, s tools.LLMSettings) (*tools.ModelList, error) {
		return nil, fmt.Errorf("the provider answered 401: invalid x-api-key %s", s.APIKey)
	}
	u := testUser(t, db, "models3")
	rec := do(h, u, "POST", "/api/settings/llm/models", map[string]string{"provider": "openai", "apiKey": fakeKey})
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "401") || strings.Contains(rec.Body.String(), fakeKey) {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
