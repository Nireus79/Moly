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

	"moly/keystore"
)

// moly --doctor prints what an installation needs and what is missing, in plain lines ("ok", "warning", "problem"). It
// is for the installer and for support: it contains no message text, no keys and no personal data.

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

	dbPath := filepath.Join(dir, "moly.db")
	switch ks := keystore.Inspect(dir, dbPath, keystore.SystemKeyring{}).Status(); ks.State {
	case keystore.StateReady:
		line("ok", "database key", "kept by: "+string(ks.Backend))
	case keystore.StateLocked:
		line("ok", "database key", "sealed with your passphrase: Moly asks for it when the browser starts it")
	case keystore.StateNeedsSetup:
		if ks.KeychainAvailable {
			line("ok", "database key", "will be kept in the system keychain on first start")
		} else {
			line("warning", "database key", "no system keychain here: Moly will ask you on first start to protect the key with a passphrase (recommended) or a key file. "+ks.Reason)
		}
	default:
		line("problem", "database key", ks.Reason)
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
