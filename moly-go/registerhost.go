package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// moly --register-host [--extension-ids a,b] writes the native messaging host manifest for Chrome, Chromium, Edge,
// Brave and Firefox, so the installers only have to call it. moly --unregister-host removes them again.

type browserTarget struct {
	name  string
	dir   string // manifest folder (Linux, macOS)
	reg   string // registry key under HKCU (Windows)
	gecko bool   // Firefox family: reads "allowed_extensions" and needs a Firefox add-on ID
}

func browserTargets(home string) []browserTarget {
	switch runtime.GOOS {
	case "windows":
		return []browserTarget{
			{"Chrome", "", `Software\Google\Chrome\NativeMessagingHosts`, false},
			{"Chromium", "", `Software\Chromium\NativeMessagingHosts`, false},
			{"Edge", "", `Software\Microsoft\Edge\NativeMessagingHosts`, false},
			{"Brave", "", `Software\BraveSoftware\Brave-Browser\NativeMessagingHosts`, false},
			{"Firefox", "", `Software\Mozilla\NativeMessagingHosts`, true},
		}
	case "darwin":
		base := filepath.Join(home, "Library", "Application Support")
		return []browserTarget{
			{"Chrome", filepath.Join(base, "Google", "Chrome", "NativeMessagingHosts"), "", false},
			{"Chromium", filepath.Join(base, "Chromium", "NativeMessagingHosts"), "", false},
			{"Edge", filepath.Join(base, "Microsoft Edge", "NativeMessagingHosts"), "", false},
			{"Brave", filepath.Join(base, "BraveSoftware", "Brave-Browser", "NativeMessagingHosts"), "", false},
			{"Firefox", filepath.Join(base, "Mozilla", "NativeMessagingHosts"), "", true},
		}
	default:
		cfg := os.Getenv("XDG_CONFIG_HOME")
		if cfg == "" {
			cfg = filepath.Join(home, ".config")
		}
		return []browserTarget{
			{"Chrome", filepath.Join(cfg, "google-chrome", "NativeMessagingHosts"), "", false},
			{"Chromium", filepath.Join(cfg, "chromium", "NativeMessagingHosts"), "", false},
			{"Edge", filepath.Join(cfg, "microsoft-edge", "NativeMessagingHosts"), "", false},
			{"Brave", filepath.Join(cfg, "BraveSoftware", "Brave-Browser", "NativeMessagingHosts"), "", false},
			{"Firefox", filepath.Join(home, ".mozilla", "native-messaging-hosts"), "", true},
		}
	}
}

// systemTargets are the folders every user of the computer shares, for the .deb and .pkg installers (they run as root and cannot write into
// anyone's home). MOLY_SYSROOT puts a prefix before each folder, so a package can be staged and tested without root.
func systemTargets() []browserTarget {
	root := os.Getenv("MOLY_SYSROOT")
	at := func(p string) string { return filepath.Join(root, p) }
	switch runtime.GOOS {
	case "linux":
		return []browserTarget{
			{"Chrome", at("/etc/opt/chrome/native-messaging-hosts"), "", false},
			{"Chromium", at("/etc/chromium/native-messaging-hosts"), "", false},
			{"Edge", at("/etc/opt/edge/native-messaging-hosts"), "", false},
			{"Firefox", at("/usr/lib/mozilla/native-messaging-hosts"), "", true},
		}
	case "darwin":
		return []browserTarget{
			{"Chrome", at("/Library/Google/Chrome/NativeMessagingHosts"), "", false},
			{"Chromium", at("/Library/Application Support/Chromium/NativeMessagingHosts"), "", false},
			{"Edge", at("/Library/Microsoft/Edge/NativeMessagingHosts"), "", false},
			{"Brave", at("/Library/Application Support/BraveSoftware/Brave-Browser/NativeMessagingHosts"), "", false},
			{"Firefox", at("/Library/Application Support/Mozilla/NativeMessagingHosts"), "", true},
		}
	}
	return nil
}

type hostManifest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`    // Chrome family: chrome-extension://<id>/
	AllowedExts    []string `json:"allowed_extensions,omitempty"` // Firefox: the add-on ID
}

// buildHostManifest checks the inputs: the path must be absolute and there must be at least one extension ID.
func buildHostManifest(exe string, ids []string) (hostManifest, error) {
	if !filepath.IsAbs(exe) {
		return hostManifest{}, errors.New("the host path must be absolute")
	}
	if len(ids) == 0 {
		return hostManifest{}, errors.New("no extension IDs: pass --extension-ids or set MOLY_EXTENSION_IDS")
	}
	m := hostManifest{Name: hostName, Description: "Moly backend host", Path: exe, Type: "stdio"}
	for _, id := range ids {
		if isGeckoID(id) {
			m.AllowedExts = append(m.AllowedExts, id)
		} else {
			m.AllowedOrigins = append(m.AllowedOrigins, "chrome-extension://"+id+"/")
		}
	}
	return m, nil
}

func runRegisterHost(args []string) int {
	fs := flag.NewFlagSet("register-host", flag.ContinueOnError)
	idsFlag := fs.String("extension-ids", "", "comma separated extension IDs (Chrome and Edge store IDs, Firefox add-on ID)")
	exeFlag := fs.String("path", "", "path of the Moly binary (default: this binary)")
	system := fs.Bool("system", false, "register for every user of this computer (a package installer running as root); the extension IDs are read from extension-ids.txt next to the program")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	// A new registration replaces the old list (stale development IDs must not stay allowed).
	ids := splitIDs(append(configuredExtensionIDs(), *idsFlag), ",")
	if *system {
		ids = splitIDs(append(ids, idsNextToProgram()), ",")
	}
	exe := *exeFlag
	if exe == "" {
		var err error
		if exe, err = os.Executable(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	m, err := buildHostManifest(exe, ids)
	if err != nil {
		fmt.Fprintln(os.Stderr, "register-host:", err)
		return 2
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	home, _ := os.UserHomeDir()
	targets := browserTargets(home)
	if *system {
		targets = systemTargets() // the IDs stay in the file next to the program; no user's data folder is touched
	} else if err := os.MkdirAll(molyDataDir(), 0o700); err == nil {
		os.WriteFile(filepath.Join(molyDataDir(), extensionIDsFile), []byte(strings.Join(ids, "\n")+"\n"), 0o600)
	}
	for _, t := range targets {
		// Each browser family gets the host only when an extension for it is listed.
		if (t.gecko && len(m.AllowedExts) == 0) || (!t.gecko && len(m.AllowedOrigins) == 0) {
			continue
		}
		var path string
		if runtime.GOOS == "windows" {
			path = filepath.Join(molyDataDir(), hostName+".json") // one file, every browser's registry key points at it
		} else {
			path = filepath.Join(t.dir, hostName+".json")
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", t.name, err)
			continue
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", t.name, err)
			continue
		}
		if t.reg != "" {
			key := `HKCU\` + t.reg + `\` + hostName
			if out, err := exec.Command("reg", "add", key, "/ve", "/t", "REG_SZ", "/d", path, "/f").CombinedOutput(); err != nil {
				fmt.Fprintf(os.Stderr, "%s: reg add failed: %v %s\n", t.name, err, out)
				continue
			}
		}
		fmt.Printf("%s: registered (%s)\n", t.name, path)
	}
	return 0
}

func runUnregisterHost(args []string) int {
	system := len(args) > 0 && args[0] == "--system"
	home, _ := os.UserHomeDir()
	targets := browserTargets(home)
	if system {
		targets = systemTargets()
	} else {
		os.Remove(filepath.Join(molyDataDir(), extensionIDsFile))
	}
	for _, t := range targets {
		if t.reg != "" {
			exec.Command("reg", "delete", `HKCU\`+t.reg+`\`+hostName, "/f").Run()
			continue
		}
		if err := os.Remove(filepath.Join(t.dir, hostName+".json")); err == nil {
			fmt.Printf("%s: removed\n", t.name)
		}
		// Folders we may have created for a browser that is not installed: remove them only when empty.
		if !system {
			removeEmptyParents(t.dir, home)
		}
	}
	if runtime.GOOS == "windows" {
		os.Remove(filepath.Join(molyDataDir(), hostName+".json"))
	}
	return 0
}

// removeEmptyParents removes dir and then each parent that is empty, up to (not including) the shared folders every
// program uses (the home folder, the config folder, Application Support). A folder with anything in it stays.
func removeEmptyParents(dir, home string) {
	shared := map[string]bool{
		home:                           true,
		filepath.Join(home, ".config"): true,
		os.Getenv("XDG_CONFIG_HOME"):   true,
		filepath.Join(home, "Library"): true,
		filepath.Join(home, "Library", "Application Support"): true,
	}
	for d := dir; d != "" && !shared[d] && d != filepath.Dir(d); d = filepath.Dir(d) {
		if os.Remove(d) != nil {
			return
		}
	}
}
