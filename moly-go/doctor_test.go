package main

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A package installed for every user (.deb, .pkg) registers the browser link in the shared folders; the check must see it.
func TestDoctorSeesTheSystemWideRegistration(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("system folders exist on Linux and macOS")
	}
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("MOLY_SYSROOT", root)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("MOLY_EXTENSION_IDS", "chromeid")
	if code := runRegisterHost([]string{"--system", "--path", "/opt/moly/moly"}); code != 0 {
		t.Fatalf("register: %d", code)
	}
	var out bytes.Buffer
	runDoctor(&out)
	if strings.Contains(out.String(), "not registered for any browser") {
		t.Fatalf("the system-wide registration must count:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Chrome host") {
		t.Fatalf("the Chrome manifest must be listed:\n%s", out.String())
	}
}
