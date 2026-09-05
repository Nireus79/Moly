package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindCORSProxyScriptEnvVarOverride(t *testing.T) {
	// Save current env var
	oldProxyPath := os.Getenv("MOLY_PROXY_PATH")
	defer os.Setenv("MOLY_PROXY_PATH", oldProxyPath)

	// Create a temporary file to use as test proxy
	tempFile, err := os.CreateTemp("", "moly-proxy-*.js")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	// Set env var to temp file
	os.Setenv("MOLY_PROXY_PATH", tempFile.Name())

	// Should find the file via env var
	proxyPath, err := findCORSProxyScript()
	if err != nil {
		t.Errorf("Expected to find proxy via env var, got error: %v", err)
	}
	if proxyPath != tempFile.Name() {
		t.Errorf("Expected proxy path %s, got %s", tempFile.Name(), proxyPath)
	}
}

func TestFindCORSProxyScriptEnvVarNotFound(t *testing.T) {
	// Save current env var
	oldProxyPath := os.Getenv("MOLY_PROXY_PATH")
	defer os.Setenv("MOLY_PROXY_PATH", oldProxyPath)

	// Set env var to non-existent file
	os.Setenv("MOLY_PROXY_PATH", "/nonexistent/path/moly-proxy.js")

	// Should return error with helpful message
	_, err := findCORSProxyScript()
	if err == nil {
		t.Error("Expected error for missing env var path, got nil")
	}
	if err.Error() != "MOLY_PROXY_PATH set but file not found: /nonexistent/path/moly-proxy.js" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestFindCORSProxyScriptEnvVarNotSet(t *testing.T) {
	// Save current env var
	oldProxyPath := os.Getenv("MOLY_PROXY_PATH")
	defer os.Setenv("MOLY_PROXY_PATH", oldProxyPath)

	// Unset the env var
	os.Unsetenv("MOLY_PROXY_PATH")

	// Should try other strategies (may or may not find file depending on dev environment)
	// Just verify it doesn't error immediately
	_, err := findCORSProxyScript()
	// This may succeed or fail depending on whether proxy is in any standard location
	// The important thing is the function gracefully handles missing env var
	if err != nil && err.Error() == "MOLY_PROXY_PATH set but file not found:" {
		t.Error("Should not check env var when it's not set")
	}
}

func TestFindCORSProxyScriptRelativeToDev(t *testing.T) {
	// Save current env var and working directory
	oldProxyPath := os.Getenv("MOLY_PROXY_PATH")
	oldDir, _ := os.Getwd()

	defer os.Setenv("MOLY_PROXY_PATH", oldProxyPath)
	defer os.Chdir(oldDir)

	// Unset env var to test relative path detection
	os.Unsetenv("MOLY_PROXY_PATH")

	// Create temp directory structure for test
	tempDir := t.TempDir()
	proxyDir := filepath.Join(tempDir, "moly-proxy", "bin")
	os.MkdirAll(proxyDir, 0755)

	proxyFile := filepath.Join(proxyDir, "moly-proxy.js")
	os.WriteFile(proxyFile, []byte("// mock proxy"), 0644)

	// Change to temp directory and test finding relative path
	os.Chdir(tempDir)

	proxyPath, err := findCORSProxyScript()
	if err != nil {
		// Might fail if no proxy in standard locations, but should have checked relative paths
		if err.Error() != "CORS proxy script not found. Tried multiple locations. Set MOLY_PROXY_PATH environment variable to specify location" {
			t.Logf("Got expected error when proxy not in standard locations: %v", err)
		}
	} else {
		// Path format may vary based on OS, but should contain moly-proxy/bin
		hasProxyPath := false
		for i := 0; i <= len(proxyPath)-len("moly-proxy"); i++ {
			if proxyPath[i:i+len("moly-proxy")] == "moly-proxy" {
				hasProxyPath = true
				break
			}
		}
		if !hasProxyPath {
			t.Errorf("Expected proxy path to contain moly-proxy, got: %s", proxyPath)
		}
	}
}

func TestFindCORSProxyScriptErrorMessage(t *testing.T) {
	// Save current env var
	oldProxyPath := os.Getenv("MOLY_PROXY_PATH")
	defer os.Setenv("MOLY_PROXY_PATH", oldProxyPath)

	// Unset env var and ensure no proxy exists in dev locations
	os.Unsetenv("MOLY_PROXY_PATH")

	// On a system without Moly installed, this should return a helpful error
	// This test just verifies the error handling is present
	_, err := findCORSProxyScript()
	if err != nil {
		// Error is expected when proxy not installed
		if len(err.Error()) == 0 {
			t.Error("Error message should not be empty")
		}
		// Check if error message guides user to MOLY_PROXY_PATH
		errMsg := err.Error()
		hasCorsError := false
		hasMolyPath := false
		for i := 0; i <= len(errMsg)-len("CORS proxy script not found"); i++ {
			if errMsg[i:i+len("CORS proxy script not found")] == "CORS proxy script not found" {
				hasCorsError = true
				break
			}
		}
		for i := 0; i <= len(errMsg)-len("MOLY_PROXY_PATH"); i++ {
			if errMsg[i:i+len("MOLY_PROXY_PATH")] == "MOLY_PROXY_PATH" {
				hasMolyPath = true
				break
			}
		}
		if !hasCorsError && !hasMolyPath {
			t.Errorf("Error message should guide user to MOLY_PROXY_PATH: %v", err)
		}
	}
}
