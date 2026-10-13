package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
)

// moly --doctor prints what an installation needs and what is missing, in plain lines ("ok", "warning", "problem"). It
// is for the installer and for support: it contains no message text, no keys and no personal data.

func keychainCheck() (ok bool, detail string) {
	if os.Getenv(dbKeyEnvName) != "" {
		return true, "not needed: the database key comes from " + dbKeyEnvName
	}
	const svc, user = "moly", "doctor-test"
	if err := keyring.Set(svc, user, "x"); err != nil {
		return false, "the operating system keychain is not available (" + err.Error() + ")"
	}
	defer keyring.Delete(svc, user)
	if v, err := keyring.Get(svc, user); err != nil || v != "x" {
		return false, "the keychain accepted a value but did not return it"
	}
	return true, "works"
}

// dbKeyEnvName mirrors the variable the database package reads.
const dbKeyEnvName = "MOLY_DB_KEY"

func runDoctor(w io.Writer) int {
	problems := 0
	line := func(level, name, detail string) {
		if level == "problem" {
			problems++
		}
		fmt.Fprintf(w, "%-8s %-22s %s\n", level, name, detail)
	}

	exe, _ := os.Executable()
	line("ok", "Moly", fmt.Sprintf("%s (%s/%s) at %s", version, runtime.GOOS, runtime.GOARCH, exe))

	dir := molyDataDir()
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		line("ok", "data folder", dir)
	} else {
		line("ok", "data folder", dir+" (created on first start)")
	}

	if ok, detail := keychainCheck(); ok {
		line("ok", "keychain", detail)
	} else {
		line("problem", "keychain", detail+". Moly keeps its database key there. On Linux install and unlock GNOME Keyring or KWallet (any Secret Service provider).")
	}

	cfg := LoadConfig()
	client := &http.Client{Timeout: 2 * time.Second}
	if resp, err := client.Get(strings.TrimRight(cfg.OllamaEndpoint, "/") + "/api/tags"); err == nil {
		resp.Body.Close()
		line("ok", "Ollama", "answers at "+cfg.OllamaEndpoint)
	} else {
		line("warning", "Ollama", "not running at "+cfg.OllamaEndpoint+" (fine if you use Claude or OpenAI with your own key)")
	}

	ids := allowedExtensionIDs()
	if len(ids) == 0 {
		line("problem", "extension IDs", "none registered: run moly --register-host --extension-ids <id>")
	} else {
		line("ok", "extension IDs", strings.Join(ids, ", "))
	}

	home, _ := os.UserHomeDir()
	registered := 0
	for _, t := range browserTargets(home) {
		if t.reg != "" {
			continue // the Windows registry is checked by the Windows installer
		}
		path := filepath.Join(t.dir, hostName+".json")
		if _, err := os.Stat(path); err == nil {
			registered++
			line("ok", t.name+" host", path)
		}
	}
	if runtime.GOOS != "windows" && registered == 0 {
		line("problem", "browser host", "not registered for any browser: run moly --register-host --extension-ids <id>")
	}

	if resp, err := client.Get("http://127.0.0.1" + cfg.Port + "/api/status"); err == nil {
		resp.Body.Close()
		line("ok", "backend", fmt.Sprintf("answering on port %s (status %d)", strings.TrimPrefix(cfg.Port, ":"), resp.StatusCode))
	} else {
		line("ok", "backend", "not running (it starts when the extension asks for it)")
	}

	if problems > 0 {
		fmt.Fprintf(w, "\n%d problem(s) found.\n", problems)
		return 1
	}
	fmt.Fprintln(w, "\nEverything needed is in place.")
	return 0
}
