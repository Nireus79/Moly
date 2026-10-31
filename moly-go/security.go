package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Production mode (MOLY_PRODUCTION=1, set by the native host or the installer) makes the local API safe to leave running:
// it listens on the loopback address only, answers only the Moly extension's origin, and needs a per-run token.
// Development mode (the default) is unchanged: all interfaces, CORS *, no token.

const (
	tokenHeader = "X-Moly-Token"
	tokenFile   = "session.token"
)

// knownExtensionIDs are the store IDs of the published extension (Chrome Web Store, Edge Add-ons).
// MOLY_EXTENSION_IDS (comma separated) adds more, for unpacked builds during testing.
var knownExtensionIDs = []string{}

func productionMode() bool { return os.Getenv("MOLY_PRODUCTION") == "1" }

// molyDataDir is the folder of the database and the token file: .moly in the user's home folder
// ($HOME on Linux and macOS, the profile folder on Windows, where $HOME is normally not set).
func molyDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.ExpandEnv("$HOME")
	}
	return filepath.Join(home, ".moly")
}

// extensionIDsFile lists the IDs --register-host was given, one per line. A backend started by a browser has no
// environment of ours, so this file is how it learns which extensions may talk to it.
const extensionIDsFile = "extension-ids.txt"

// configuredExtensionIDs are the IDs given by the build and the environment (not the file).
func configuredExtensionIDs() []string {
	return splitIDs(append(append([]string{}, knownExtensionIDs...), os.Getenv("MOLY_EXTENSION_IDS")), ",")
}

func splitIDs(lists []string, sep string) []string {
	var ids []string
	seen := map[string]bool{}
	for _, list := range lists {
		for _, id := range strings.Split(list, sep) {
			if id = strings.TrimSpace(id); id != "" && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// idsNextToProgram reads extension-ids.txt in the folder of the program: where a package installer (.deb, .pkg) puts the IDs, because it has
// no user's data folder to write to.
func idsNextToProgram() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return idsInDir(filepath.Dir(exe))
}

func idsInDir(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, extensionIDsFile))
	if err != nil {
		return ""
	}
	var ids []string
	for _, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			ids = append(ids, line)
		}
	}
	return strings.Join(ids, ",")
}

func allowedExtensionIDs() []string {
	lists := append(configuredExtensionIDs(), idsNextToProgram())
	if data, err := os.ReadFile(filepath.Join(molyDataDir(), extensionIDsFile)); err == nil {
		lists = append(lists, strings.ReplaceAll(string(data), "\n", ","))
	}
	return splitIDs(lists, ",")
}

type securityPolicy struct {
	origins map[string]bool // "chrome-extension://<id>"
	firefox bool            // a Firefox add-on is registered: any "moz-extension://" origin may pass (see originAllowed)
	token   string
	port    string // "11436"
}

func newSecurityPolicy(port string, extensionIDs []string) (*securityPolicy, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	p := &securityPolicy{origins: map[string]bool{}, token: hex.EncodeToString(raw), port: strings.TrimPrefix(port, ":")}
	for _, id := range extensionIDs {
		if isGeckoID(id) {
			p.firefox = true
			continue
		}
		p.origins["chrome-extension://"+id] = true
	}
	return p, nil
}

// isGeckoID tells a Firefox add-on ID ("name@domain" or "{uuid}") from a Chrome / Edge extension ID (32 letters).
func isGeckoID(id string) bool {
	return strings.Contains(id, "@") || strings.HasPrefix(id, "{")
}

// originAllowed: Firefox gives every installation a random "moz-extension://<uuid>" origin, so it cannot be listed in
// advance. It is accepted only when a Firefox add-on is registered; the session token (handed out by the native host, which
// Firefox lets only that add-on start) is what keeps other add-ons out.
func (p *securityPolicy) originAllowed(origin string) bool {
	if p.origins[origin] {
		return true
	}
	rest, ok := strings.CutPrefix(origin, "moz-extension://")
	return p.firefox && ok && rest != "" && !strings.ContainsAny(rest, "/?# ")
}

// writeToken stores the token where the native host (same user) can read it, readable by the owner only.
func (p *securityPolicy) writeToken(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, tokenFile), []byte(p.token), 0o600)
}

func (p *securityPolicy) hostAllowed(hostport string) bool {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return false
	}
	if port != p.port {
		return false
	}
	switch host {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	return false
}

func (p *securityPolicy) needsToken(path string) bool {
	if path == "/api/status" {
		return false
	}
	return strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/internal/")
}

func denyJSON(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// middleware enforces, in order: loopback Host (stops DNS rebinding), allowed Origin, token. It also answers preflight
// requests and recovers from handler panics.
func (p *securityPolicy) middleware(next http.Handler) http.Handler {
	inner := recoverMiddleware(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !p.hostAllowed(r.Host) {
			denyJSON(w, http.StatusForbidden, "host not allowed")
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			if !p.originAllowed(origin) {
				denyJSON(w, http.StatusForbidden, "origin not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID, "+tokenHeader)
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if p.needsToken(r.URL.Path) && subtle.ConstantTimeCompare([]byte(r.Header.Get(tokenHeader)), []byte(p.token)) != 1 {
			denyJSON(w, http.StatusUnauthorized, "missing or wrong token")
			return
		}
		inner.ServeHTTP(w, r)
	})
}

// listenLoopback binds the loopback address before the slow startup work, so a second Moly stops at once
// instead of opening the same database. The error says whether the port is taken.
func listenLoopback(port string) (net.Listener, error) {
	lis, err := net.Listen("tcp", "127.0.0.1"+port)
	if err != nil {
		return nil, errors.New("port " + strings.TrimPrefix(port, ":") + " is not free (is Moly already running?): " + err.Error())
	}
	return lis, nil
}
