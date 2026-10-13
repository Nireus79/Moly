package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// stopBackend asks a running backend to shut down and waits until its port is free. It returns false when nothing was running.
func stopBackend(base, token string, wait time.Duration) (wasRunning bool, err error) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, _ := http.NewRequest(http.MethodPost, base+"/internal/shutdown", nil)
	req.Header.Set(tokenHeader, token)
	resp, err := client.Do(req)
	if err != nil {
		return false, nil // nothing listens: already stopped
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return true, fmt.Errorf("the backend refused to stop (status %d)", resp.StatusCode)
	}
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		r, err := client.Get(base + "/api/status")
		if err != nil {
			return true, nil
		}
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		time.Sleep(200 * time.Millisecond)
	}
	return true, fmt.Errorf("the backend did not stop within %v", wait)
}

// runStop is "moly --stop": it stops this user's running backend, if any.
func runStop(w io.Writer) int {
	cfg := LoadConfig()
	data, err := os.ReadFile(filepath.Join(molyDataDir(), tokenFile))
	if err != nil {
		fmt.Fprintln(w, "Moly is not running.")
		return 0
	}
	was, err := stopBackend("http://127.0.0.1"+cfg.Port, strings.TrimSpace(string(data)), 15*time.Second)
	switch {
	case err != nil:
		fmt.Fprintln(w, "Moly:", err)
		return 1
	case was:
		fmt.Fprintln(w, "Moly was stopped.")
	default:
		fmt.Fprintln(w, "Moly is not running.")
	}
	return 0
}
