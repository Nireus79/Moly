package main

import (
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"moly/keystore"
)

type downKeyring struct{}

func (downKeyring) Get(string, string) (string, error) { return "", errors.New("no secret service") }
func (downKeyring) Set(string, string, string) error   { return errors.New("no secret service") }
func (downKeyring) Delete(string, string) error        { return nil }

func gateServer(t *testing.T) (*httptest.Server, *keystore.Store) {
	t.Helper()
	dir := t.TempDir()
	store := keystore.Open(dir, filepath.Join(dir, "moly.db"), downKeyring{})
	mux := http.NewServeMux()
	registerKeyRoutes(mux, store, "test")
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, store
}

func gateDo(t *testing.T, method, url, body string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	buf := new(strings.Builder)
	b := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(b)
		buf.Write(b[:n])
		if err != nil {
			break
		}
	}
	return resp.StatusCode, buf.String()
}

func TestGateStatusIsLockedUntilTheKeyIsChosen(t *testing.T) {
	srv, store := gateServer(t)
	if code, body := gateDo(t, "GET", srv.URL+"/api/status", ""); code != http.StatusLocked || !strings.Contains(body, `"locked"`) {
		t.Fatalf("status = %d %s", code, body)
	}
	code, body := gateDo(t, "GET", srv.URL+"/api/keystore/status", "")
	if code != 200 || !strings.Contains(body, `"needs_setup"`) || !strings.Contains(body, `"passphrase"`) {
		t.Fatalf("keystore status = %d %s", code, body)
	}
	if code, _ := gateDo(t, "GET", srv.URL+"/api/user/export", ""); code != http.StatusLocked {
		t.Fatalf("other routes while locked = %d", code)
	}
	// A short passphrase is refused, the file mode needs a confirmation, and nothing is ready yet.
	if code, _ := gateDo(t, "POST", srv.URL+"/api/keystore/setup", `{"mode":"passphrase","passphrase":"short"}`); code != 400 {
		t.Fatalf("short passphrase: %d", code)
	}
	if code, _ := gateDo(t, "POST", srv.URL+"/api/keystore/setup", `{"mode":"file"}`); code != 400 {
		t.Fatalf("file without confirmation: %d", code)
	}
	if _, ok := store.Key(); ok {
		t.Fatal("key before a valid choice")
	}
	code, body = gateDo(t, "POST", srv.URL+"/api/keystore/setup", `{"mode":"passphrase","passphrase":"a long enough passphrase"}`)
	if code != 200 || !strings.Contains(body, `"ready"`) {
		t.Fatalf("setup = %d %s", code, body)
	}
	k, ok := store.Key()
	if !ok {
		t.Fatal("not ready after setup")
	}
	select {
	case <-store.Ready():
	case <-time.After(time.Second):
		t.Fatal("Ready not closed")
	}
	if strings.Contains(body, hex.EncodeToString(k)) {
		t.Fatal("the key is in the answer")
	}
	// Once ready the gate says "starting" (503), and setup is no longer allowed.
	if code, _ := gateDo(t, "GET", srv.URL+"/api/status", ""); code != http.StatusServiceUnavailable {
		t.Fatalf("status after ready = %d", code)
	}
	if code, _ := gateDo(t, "POST", srv.URL+"/api/keystore/setup", `{"mode":"file","confirmInsecure":true}`); code != 400 {
		t.Fatalf("second setup: %d", code)
	}
}

func TestGateUnlockWithWrongAndRightPassphrase(t *testing.T) {
	dir := t.TempDir()
	db := filepath.Join(dir, "moly.db")
	first := keystore.Open(dir, db, downKeyring{})
	if err := first.Setup("passphrase", "a long enough passphrase", false); err != nil {
		t.Fatal(err)
	}
	store := keystore.Open(dir, db, downKeyring{}) // the next start: locked
	mux := http.NewServeMux()
	registerKeyRoutes(mux, store, "test")
	srv := httptest.NewServer(mux)
	defer srv.Close()
	if _, body := gateDo(t, "GET", srv.URL+"/api/keystore/status", ""); !strings.Contains(body, `"locked"`) {
		t.Fatalf("body = %s", body)
	}
	if code, body := gateDo(t, "POST", srv.URL+"/api/keystore/unlock", `{"passphrase":"wrong wrong wrong"}`); code != http.StatusForbidden || strings.Contains(body, "enough") {
		t.Fatalf("wrong = %d %s", code, body)
	}
	if code, _ := gateDo(t, "POST", srv.URL+"/api/keystore/unlock", `{"passphrase":"a long enough passphrase"}`); code != 200 {
		t.Fatalf("right = %d", code)
	}
	if _, ok := store.Key(); !ok {
		t.Fatal("not unlocked")
	}
}
