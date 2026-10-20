package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := writeFrame(&buf, map[string]any{"action": "ping", "text": "ελληνικά ✓"}); err != nil {
		t.Fatal(err)
	}
	raw, err := readFrame(&buf)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	json.Unmarshal(raw, &got)
	if got["text"] != "ελληνικά ✓" {
		t.Fatalf("round trip lost the text: %v", got)
	}
}

func TestFrameLimits(t *testing.T) {
	huge := bytes.NewReader([]byte{0xff, 0xff, 0xff, 0x7f})
	if _, err := readFrame(huge); err == nil {
		t.Fatal("a frame over the limit must be refused before any allocation")
	}
	if err := writeFrame(io.Discard, strings.Repeat("x", maxFrame+10)); err == nil {
		t.Fatal("a reply over the 1 MB browser limit must be refused")
	}
	if _, err := readFrame(bytes.NewReader([]byte{5, 0, 0, 0, 'a'})); err == nil {
		t.Fatal("a truncated frame must be an error")
	}
}

type fakeBackend struct {
	srv        *httptest.Server
	statusCode atomic.Int32
	attached   atomic.Int32
	beats      atomic.Int32
	detached   atomic.Int32
}

func newFakeBackend(t *testing.T, token string) *fakeBackend {
	f := &fakeBackend{}
	f.statusCode.Store(200)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(f.statusCode.Load()))
		w.Write([]byte(`{"status":"ok","version":"2.1"}`))
	})
	guard := func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(tokenHeader) != token {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			h(w, r)
		}
	}
	mux.HandleFunc("/internal/attach", guard(func(w http.ResponseWriter, r *http.Request) {
		f.attached.Add(1)
		w.Write([]byte(`{"hostId":"h1"}`))
	}))
	mux.HandleFunc("/internal/heartbeat", guard(func(w http.ResponseWriter, r *http.Request) {
		f.beats.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.HandleFunc("/internal/detach", guard(func(w http.ResponseWriter, r *http.Request) {
		f.detached.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func hostWithBackend(t *testing.T) (*nativeHost, *fakeBackend, *bytes.Buffer) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	os.MkdirAll(filepath.Join(home, ".moly"), 0o700)
	os.WriteFile(filepath.Join(home, ".moly", tokenFile), []byte("tok123"), 0o600)
	fb := newFakeBackend(t, "tok123")
	var out bytes.Buffer
	h := newNativeHost(":0", "http://127.0.0.1:1", &out)
	h.base = fb.srv.URL
	h.spawn = func() error { t.Fatal("must not spawn when the backend answers"); return nil }
	return h, fb, &out
}

func lastReply(t *testing.T, out *bytes.Buffer) map[string]any {
	t.Helper()
	raw, err := readFrame(out)
	if err != nil {
		t.Fatalf("no reply frame: %v", err)
	}
	var m map[string]any
	json.Unmarshal(raw, &m)
	return m
}

func TestHostPingAndUnknownAction(t *testing.T) {
	h, _, out := hostWithBackend(t)
	h.handle([]byte(`{"action":"ping","id":7}`))
	r := lastReply(t, out)
	if r["pong"] != true || r["id"] != float64(7) {
		t.Fatalf("ping reply wrong: %v", r)
	}
	h.handle([]byte(`{"action":"format-disk"}`))
	if r := lastReply(t, out); r["success"] != false {
		t.Fatalf("unknown action must fail: %v", r)
	}
	h.handle([]byte(`not json`))
	if r := lastReply(t, out); r["success"] != false {
		t.Fatalf("bad json must fail: %v", r)
	}
}

func TestHostStartReturnsTokenAndAttachesOnce(t *testing.T) {
	h, fb, out := hostWithBackend(t)
	h.handle([]byte(`{"action":"start"}`))
	r := lastReply(t, out)
	if r["success"] != true || r["token"] != "tok123" || r["url"] != fb.srv.URL || r["version"] != "2.1" {
		t.Fatalf("start reply wrong: %v", r)
	}
	h.handle([]byte(`{"action":"start-backend"}`)) // the old action name still works
	if r := lastReply(t, out); r["success"] != true {
		t.Fatalf("start-backend alias failed: %v", r)
	}
	if fb.attached.Load() != 1 {
		t.Fatalf("host must attach once, attached %d times", fb.attached.Load())
	}
	h.detach()
	if fb.detached.Load() != 1 {
		t.Fatalf("host must detach when it ends, detached %d times", fb.detached.Load())
	}
}

func TestHostStartsBackendWhenNothingListens(t *testing.T) {
	h, fb, out := hostWithBackend(t)
	fb.statusCode.Store(200)
	down := true
	h.base = "http://127.0.0.1:1" // nothing there
	spawned := 0
	h.spawn = func() error {
		spawned++
		down = false
		h.base = fb.srv.URL // the backend "comes up"
		return nil
	}
	h.handle([]byte(`{"action":"start"}`))
	if r := lastReply(t, out); r["success"] != true || down || spawned != 1 {
		t.Fatalf("host must spawn exactly once and then succeed: spawned=%d reply=%v", spawned, r)
	}
}

func TestHostWaitsWhileBackendIsStarting(t *testing.T) {
	h, fb, out := hostWithBackend(t)
	fb.statusCode.Store(503)
	go func() { time.Sleep(700 * time.Millisecond); fb.statusCode.Store(200) }()
	h.handle([]byte(`{"action":"start"}`))
	if r := lastReply(t, out); r["success"] != true {
		t.Fatalf("host must wait for 503 to turn into 200 without spawning: %v", r)
	}
}

func TestHostStatus(t *testing.T) {
	h, fb, out := hostWithBackend(t)
	h.handle([]byte(`{"action":"status"}`))
	if r := lastReply(t, out); r["state"] != "running" || r["running"] != true {
		t.Fatalf("running: %v", r)
	}
	fb.statusCode.Store(503)
	h.handle([]byte(`{"action":"status"}`))
	if r := lastReply(t, out); r["state"] != "starting" {
		t.Fatalf("starting: %v", r)
	}
	h.base = "http://127.0.0.1:1"
	h.handle([]byte(`{"action":"status"}`))
	if r := lastReply(t, out); r["state"] != "stopped" || r["running"] != false {
		t.Fatalf("stopped: %v", r)
	}
}

func TestHostModelsWithoutOllamaIsNotAnError(t *testing.T) {
	h, _, out := hostWithBackend(t)
	h.handle([]byte(`{"action":"get-models"}`))
	r := lastReply(t, out)
	if r["success"] != true {
		t.Fatalf("no Ollama must give an empty list, not a failure: %v", r)
	}
}

func TestRunNativeHostEndsOnEOFAndRefusesForeignOrigin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("MOLY_EXTENSION_IDS", "abcdefghijklmnopabcdefghijklmnop")
	var frame bytes.Buffer
	writeFrame(&frame, map[string]string{"action": "ping"})
	var out bytes.Buffer
	if code := runNativeHost("chrome-extension://abcdefghijklmnopabcdefghijklmnop/", &frame, &out); code != 0 {
		t.Fatalf("exit code %d, want 0 at end of input", code)
	}
	deadline := time.Now().Add(2 * time.Second)
	for out.Len() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if out.Len() == 0 {
		t.Fatal("the ping must be answered before the host ends")
	}
	if code := runNativeHost("chrome-extension://ponmlkjihgfedcbaponmlkjihgfedcba/", bytes.NewReader(nil), io.Discard); code == 0 {
		t.Fatal("an extension that is not on the list must be refused")
	}
}

func TestHostManifest(t *testing.T) {
	if _, err := buildHostManifest("relative/moly", []string{"a"}); err == nil {
		t.Fatal("a relative path must be refused (macOS and Linux need an absolute path)")
	}
	if _, err := buildHostManifest("/opt/moly", nil); err == nil {
		t.Fatal("no extension IDs must be refused")
	}
	m, err := buildHostManifest("/opt/moly", []string{"aaa", "bbb"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(m)
	s := string(raw)
	for _, want := range []string{`"name":"com.moly.backend_host"`, `"type":"stdio"`, `"chrome-extension://aaa/"`, `"chrome-extension://bbb/"`, `"allowed_origins"`} {
		if !strings.Contains(s, want) {
			t.Errorf("manifest lacks %s: %s", want, s)
		}
	}
	if strings.Contains(s, "allowed_extensions") || strings.Contains(s, "*") {
		t.Errorf("no Firefox key and no wildcards: %s", s)
	}
}

func TestEnsureConfigDirWritesEmbeddedFilesWhenMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	wd, _ := os.Getwd()
	empty := t.TempDir()
	os.Chdir(empty)
	defer os.Chdir(wd)
	if err := ensureConfigDir(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join("config", "constitution.yaml")); err != nil {
		t.Fatalf("constitution.yaml must be readable after ensureConfigDir: %v", err)
	}
	cwd, _ := os.Getwd()
	if !strings.HasSuffix(cwd, filepath.Join(".moly", "runtime")) {
		t.Fatalf("must move to the runtime folder, is in %s", cwd)
	}
}

func TestEnsureConfigDirLeavesASourceCheckoutAlone(t *testing.T) {
	wd, _ := os.Getwd() // the test runs inside moly-go, which has config/constitution.yaml
	if err := ensureConfigDir(); err != nil {
		t.Fatal(err)
	}
	if now, _ := os.Getwd(); now != wd {
		t.Fatalf("working directory changed from %s to %s", wd, now)
	}
}

func TestAllowedIDsComeFromFileAndEnvironment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("MOLY_EXTENSION_IDS", "envid")
	os.MkdirAll(filepath.Join(home, ".moly"), 0o700)
	os.WriteFile(filepath.Join(home, ".moly", extensionIDsFile), []byte("chromeid\nedgeid\n\n"), 0o600)
	got := strings.Join(allowedExtensionIDs(), ",")
	if got != "envid,chromeid,edgeid" {
		t.Fatalf("ids = %q", got)
	}
}

func TestRegisterHostRecordsTheIDsForTheBackend(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("MOLY_EXTENSION_IDS", "")
	if code := runRegisterHost([]string{"--extension-ids", "chromeid,edgeid", "--path", "/opt/moly/moly"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if ids := strings.Join(allowedExtensionIDs(), ","); ids != "chromeid,edgeid" {
		t.Fatalf("backend would allow %q", ids)
	}
	m := filepath.Join(home, ".config", "google-chrome", "NativeMessagingHosts", hostName+".json")
	data, err := os.ReadFile(m)
	if err != nil || !strings.Contains(string(data), "chrome-extension://edgeid/") {
		t.Fatalf("manifest missing or wrong: %v %s", err, data)
	}
	if code := runUnregisterHost(); code != 0 {
		t.Fatal("unregister failed")
	}
	if _, err := os.Stat(m); err == nil {
		t.Fatal("manifest must be gone after unregister")
	}
	if len(allowedExtensionIDs()) != 0 {
		t.Fatal("ids file must be gone after unregister")
	}
}

func TestReRegisteringReplacesOldIDs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("MOLY_EXTENSION_IDS", "")
	runRegisterHost([]string{"--extension-ids", "olddev", "--path", "/opt/moly/moly"})
	runRegisterHost([]string{"--extension-ids", "newid,newid", "--path", "/opt/moly/moly"})
	if ids := strings.Join(allowedExtensionIDs(), ","); ids != "newid" {
		t.Fatalf("ids = %q, want only the latest registration", ids)
	}
}

func TestDoctorReportsAProblemWithoutExtensionIDsAndStaysFreeOfSecrets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("MOLY_EXTENSION_IDS", "")
	t.Setenv("MOLY_DB_KEY", strings.Repeat("ab", 32)) // the test machine may have no keychain
	t.Setenv("MOLY_PORT", ":1")                       // nothing listens there
	var out bytes.Buffer
	code := runDoctor(&out)
	text := out.String()
	if code != 1 || !strings.Contains(text, "problem  extension IDs") || !strings.Contains(text, "problem  browser host") {
		t.Fatalf("a bare installation must report problems (code %d):\n%s", code, text)
	}
	if strings.Contains(text, strings.Repeat("ab", 32)) {
		t.Fatalf("the doctor must never print the database key:\n%s", text)
	}

	runRegisterHost([]string{"--extension-ids", "abcdefghijklmnopabcdefghijklmnop", "--path", "/opt/moly/moly"})
	out.Reset()
	code = runDoctor(&out)
	if code != 0 || !strings.Contains(out.String(), "Chrome host") || !strings.Contains(out.String(), "Everything needed is in place") {
		t.Fatalf("after registering, no problem is expected (code %d):\n%s", code, out.String())
	}
}

func TestStopBackendAsksNicelyAndWaitsForTheEnd(t *testing.T) {
	quit := make(chan struct{}, 1)
	mux := http.NewServeMux()
	registerShutdown(mux, quit)
	policy, _ := newSecurityPolicy(":0", nil)
	var srv *httptest.Server
	stopped := make(chan struct{})
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) })
	srv = httptest.NewServer(policy.middleware(mux))
	policy.port = strings.TrimPrefix(srv.Listener.Addr().String()[strings.LastIndex(srv.Listener.Addr().String(), ":"):], ":")
	go func() { // plays the main loop: closes the server after the request
		<-quit
		time.Sleep(300 * time.Millisecond)
		srv.Close()
		close(stopped)
	}()

	if _, err := stopBackend(srv.URL, "wrong-token", time.Second); err == nil {
		t.Fatal("a wrong token must not stop the backend")
	}
	was, err := stopBackend(srv.URL, policy.token, 5*time.Second)
	if err != nil || !was {
		t.Fatalf("stop failed: was=%v err=%v", was, err)
	}
	<-stopped
	if was, err := stopBackend(srv.URL, policy.token, time.Second); was || err != nil {
		t.Fatalf("nothing is running any more: was=%v err=%v", was, err)
	}
}

func TestUnregisterRemovesFoldersItCreatedButNotOthers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("MOLY_EXTENSION_IDS", "")
	runRegisterHost([]string{"--extension-ids", "abc", "--path", "/opt/moly/moly"})
	keep := filepath.Join(home, ".config", "google-chrome", "NativeMessagingHosts", "other.host.json")
	os.WriteFile(keep, []byte("{}"), 0o644) // another program's host in the same folder
	runUnregisterHost()
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("another program's manifest must stay")
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "chromium")); err == nil {
		t.Fatal("the empty folders created for an uninstalled browser must be removed")
	}
}

func TestFirefoxHostStartAndManifest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("MOLY_EXTENSION_IDS", "moly@nireus99.test")
	if code := runNativeHost("other@example.org", bytes.NewReader(nil), io.Discard); code == 0 {
		t.Fatal("a Firefox add-on that is not on the list must be refused")
	}
	if code := runNativeHost("moly@nireus99.test", bytes.NewReader(nil), io.Discard); code != 0 {
		t.Fatalf("the listed add-on must be served, got exit code %d", code)
	}
	m, err := buildHostManifest("/opt/moly", []string{"aaa", "moly@nireus99.test"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(m)
	s := string(raw)
	for _, want := range []string{`"allowed_extensions":["moly@nireus99.test"]`, `"chrome-extension://aaa/"`} {
		if !strings.Contains(s, want) {
			t.Errorf("manifest lacks %s: %s", want, s)
		}
	}
}
