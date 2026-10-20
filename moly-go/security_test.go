package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const testExtOrigin = "chrome-extension://abcdefghijklmnopabcdefghijklmnop"

func testPolicy(t *testing.T) (*securityPolicy, http.Handler) {
	t.Helper()
	p, err := newSecurityPolicy(":11436", []string{"abcdefghijklmnopabcdefghijklmnop"})
	if err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"ok":true}`)) })
	return p, p.middleware(next)
}

func call(h http.Handler, method, path, host string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://"+host+path, nil)
	req.Host = host
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestProductionRejectsForeignHost(t *testing.T) {
	_, h := testPolicy(t)
	for _, host := range []string{"evil.example:11436", "127.0.0.1:9999", "192.168.1.5:11436", "127.0.0.1"} {
		if rec := call(h, "GET", "/api/status", host, nil); rec.Code != http.StatusForbidden {
			t.Errorf("host %q: want 403, got %d", host, rec.Code)
		}
	}
	for _, host := range []string{"127.0.0.1:11436", "localhost:11436", "[::1]:11436"} {
		if rec := call(h, "GET", "/api/status", host, nil); rec.Code != http.StatusOK {
			t.Errorf("host %q: want 200, got %d", host, rec.Code)
		}
	}
}

func TestProductionRejectsForeignOrigin(t *testing.T) {
	p, h := testPolicy(t)
	hdr := map[string]string{"Origin": "https://evil.example", tokenHeader: p.token}
	if rec := call(h, "POST", "/api/message-processor", "127.0.0.1:11436", hdr); rec.Code != http.StatusForbidden {
		t.Fatalf("foreign origin with a valid token: want 403, got %d", rec.Code)
	}
	hdr["Origin"] = "chrome-extension://ponmlkjihgfedcbaponmlkjihgfedcba"
	if rec := call(h, "OPTIONS", "/api/message-processor", "127.0.0.1:11436", hdr); rec.Code != http.StatusForbidden {
		t.Fatalf("another extension's preflight: want 403, got %d", rec.Code)
	}
}

func TestProductionAnswersOnlyTheExtensionOrigin(t *testing.T) {
	_, h := testPolicy(t)
	rec := call(h, "OPTIONS", "/api/message-processor", "127.0.0.1:11436", map[string]string{"Origin": testExtOrigin})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight: want 204, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testExtOrigin {
		t.Fatalf("allow-origin = %q, want the extension origin (never *)", got)
	}
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), tokenHeader) {
		t.Fatalf("the token header is not allowed in preflight")
	}
}

func TestProductionNeedsTokenOnApiAndInternal(t *testing.T) {
	p, h := testPolicy(t)
	for _, path := range []string{"/api/message-processor", "/api/conversations", "/internal/attach"} {
		if rec := call(h, "POST", path, "127.0.0.1:11436", nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s without token: want 401, got %d", path, rec.Code)
		}
		if rec := call(h, "POST", path, "127.0.0.1:11436", map[string]string{tokenHeader: "wrong"}); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s with a wrong token: want 401, got %d", path, rec.Code)
		}
		if rec := call(h, "POST", path, "127.0.0.1:11436", map[string]string{tokenHeader: p.token}); rec.Code != http.StatusOK {
			t.Errorf("%s with the token: want 200, got %d", path, rec.Code)
		}
	}
}

func TestProductionStatusNeedsNoToken(t *testing.T) {
	_, h := testPolicy(t)
	if rec := call(h, "GET", "/api/status", "127.0.0.1:11436", nil); rec.Code != http.StatusOK {
		t.Fatalf("status without token: want 200, got %d", rec.Code)
	}
}

func TestProductionRecoversPanic(t *testing.T) {
	p, _ := newSecurityPolicy(":11436", nil)
	h := p.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("boom") }))
	rec := call(h, "POST", "/api/message-processor", "127.0.0.1:11436", map[string]string{tokenHeader: p.token})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500 after a panic, got %d", rec.Code)
	}
}

func TestDevelopmentModeStaysOpen(t *testing.T) {
	h := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) }))
	rec := call(h, "POST", "/api/message-processor", "192.168.1.5:11436", map[string]string{"Origin": "https://anything.example"})
	if rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("development mode must stay as it was (CORS *, no token): code %d, origin %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestTokenFileIsPrivateAndRandom(t *testing.T) {
	p1, _ := newSecurityPolicy(":11436", nil)
	p2, _ := newSecurityPolicy(":11436", nil)
	if len(p1.token) != 64 || p1.token == p2.token {
		t.Fatalf("token must be 64 hex characters and differ per run")
	}
	dir := filepath.Join(t.TempDir(), "moly")
	if err := p1.writeToken(dir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, tokenFile))
	if err != nil || string(data) != p1.token {
		t.Fatalf("token file content wrong: %v", err)
	}
	if runtime.GOOS != "windows" {
		st, _ := os.Stat(filepath.Join(dir, tokenFile))
		if st.Mode().Perm() != 0o600 {
			t.Fatalf("token file mode = %v, want 0600", st.Mode().Perm())
		}
	}
}

func TestSecondInstanceCannotTakeThePort(t *testing.T) {
	lis, err := listenLoopback(":0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()
	port := lis.Addr().String()[strings.LastIndex(lis.Addr().String(), ":"):]
	if _, err := listenLoopback(port); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("a second listen on %s must fail with a clear message, got %v", port, err)
	}
	if !strings.HasPrefix(lis.Addr().String(), "127.0.0.1:") {
		t.Fatalf("must listen on loopback only, got %s", lis.Addr())
	}
}

func TestStartupSwitchAnswers503ThenDelegates(t *testing.T) {
	s := &startupSwitch{}
	rec := call(s, "GET", "/api/status", "127.0.0.1:11436", nil)
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "starting") {
		t.Fatalf("before ready: want 503 starting, got %d %q", rec.Code, rec.Body.String())
	}
	s.Set(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ready")) }))
	if rec := call(s, "GET", "/api/status", "127.0.0.1:11436", nil); rec.Body.String() != "ready" {
		t.Fatalf("after Set: want the real handler, got %q", rec.Body.String())
	}
}

func TestHostRegistryIdleLogic(t *testing.T) {
	now := time.Now()
	r := newHostRegistry()
	r.now = func() time.Time { return now }
	r.since = now

	if r.Idle() {
		t.Fatal("must not be idle right after start")
	}
	now = now.Add(idleGrace + time.Second)
	if !r.Idle() {
		t.Fatal("no host ever attached: idle after the grace period")
	}

	id := r.Attach()
	if r.Idle() {
		t.Fatal("a live host means not idle")
	}
	now = now.Add(20 * time.Second)
	if !r.Heartbeat(id) {
		t.Fatal("heartbeat of a known host must succeed")
	}
	now = now.Add(20 * time.Second) // 20 s since the heartbeat, still alive
	if r.Idle() {
		t.Fatal("host with a recent heartbeat must keep Moly alive")
	}
	now = now.Add(hostTimeout + time.Second) // heartbeats stopped (browser crashed): host expires
	if !r.Idle() {
		t.Fatal("no heartbeat for longer than the grace period: idle")
	}
	if r.Heartbeat("nope") {
		t.Fatal("unknown host id must be refused")
	}
}

func TestCleanDetachGivesAFullGracePeriod(t *testing.T) {
	now := time.Now()
	r := newHostRegistry()
	r.now = func() time.Time { return now }
	id := r.Attach()
	now = now.Add(5 * time.Minute)
	r.Heartbeat(id)
	r.Detach(id)
	now = now.Add(idleGrace - time.Second)
	if r.Idle() {
		t.Fatal("a browser that just detached may reattach (page reload): not idle yet")
	}
	now = now.Add(2 * time.Second)
	if !r.Idle() {
		t.Fatal("grace over: idle")
	}
}

func TestSeveralHostsShareOneBackend(t *testing.T) {
	now := time.Now()
	r := newHostRegistry()
	r.now = func() time.Time { return now }
	a, b := r.Attach(), r.Attach()
	r.Detach(a)
	now = now.Add(idleGrace + time.Second)
	if r.Heartbeat(b); r.Idle() {
		t.Fatal("one browser still attached: not idle")
	}
	r.Detach(b)
	now = now.Add(idleGrace + time.Second)
	if !r.Idle() {
		t.Fatal("all browsers gone: idle")
	}
}

func TestFirefoxOriginOnlyWhenAddonRegistered(t *testing.T) {
	const moz = "moz-extension://0f3c2a9e-6d1b-4c1f-8a55-3b0e5d2c7a11"
	chromeOnly, _ := newSecurityPolicy(":11436", []string{"abcdefghijklmnopabcdefghijklmnop"})
	if chromeOnly.originAllowed(moz) {
		t.Fatal("a moz-extension origin must be refused when no Firefox add-on is registered")
	}
	p, _ := newSecurityPolicy(":11436", []string{"abcdefghijklmnopabcdefghijklmnop", "moly@nireus79.github.io"})
	if !p.originAllowed(moz) || !p.originAllowed(testExtOrigin) {
		t.Fatal("both the Firefox and the Chrome origin must pass")
	}
	for _, bad := range []string{"moz-extension://", "moz-extension://a/b", "moz-extension://a b", "https://moz-extension://x", "chrome-extension://ponmlkjihgfedcbaponmlkjihgfedcba"} {
		if p.originAllowed(bad) {
			t.Errorf("%q must be refused", bad)
		}
	}
	h := p.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) }))
	if rec := call(h, "GET", "/api/status", "127.0.0.1:11436", map[string]string{"Origin": moz}); rec.Code != http.StatusOK {
		t.Fatalf("status %d for the Firefox origin", rec.Code)
	}
}

func TestFirefoxSlowAnswerKeepsConnectionAlive(t *testing.T) {
	old := keepAliveEvery
	keepAliveEvery = 20 * time.Millisecond
	defer func() { keepAliveEvery = old }()
	const moz = "moz-extension://0f3c2a9e-6d1b-4c1f-8a55-3b0e5d2c7a11"
	p, _ := newSecurityPolicy(":11436", []string{"abcdefghijklmnopabcdefghijklmnop", "moly@nireus79.github.io"})
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		w.Write([]byte(`{"ok":true}`))
	})
	h := p.middleware(slow)
	req := httptest.NewRequest("POST", "http://127.0.0.1:11436/api/message-processor", nil)
	req.Host = "127.0.0.1:11436"
	req.Header.Set("Origin", moz)
	req.Header.Set(tokenHeader, p.token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.HasPrefix(body, " ") || strings.TrimSpace(body) != `{"ok":true}` {
		t.Fatalf("status %d body %q: want leading spaces and the answer", rec.Code, body)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != moz || rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("headers lost: %v", rec.Header())
	}
	// A fast answer, and any answer for Chrome, is untouched.
	req.Header.Set("Origin", testExtOrigin)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Body.String() != `{"ok":true}` {
		t.Fatalf("Chrome body %q", rec.Body.String())
	}
}
