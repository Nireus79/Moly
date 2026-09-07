# Phase 1: Critical Fixes - Quick Start Guide

## Overview
**Duration**: 2 weeks  
**Developer-Weeks**: 8  
**Outcome**: Moly works on Windows, macOS, and Linux

---

## Task 1.1: Cross-Platform Database Paths

### What to do
Replace hardcoded Linux path with platform detection

### File: `moly-go/database.go`

**Step 1**: Add imports
```go
import (
    "runtime"
)
```

**Step 2**: Find line 74 (current code):
```go
dbPath := filepath.Join(os.Getenv("HOME"), ".config", "moly", "moly.db")
```

**Step 3**: Replace with:
```go
dbPath := getConfigDir("moly.db")
```

**Step 4**: Add new function before `initDatabase()`:
```go
func getConfigDir(filename string) string {
    var configDir string
    
    switch runtime.GOOS {
    case "windows":
        appData := os.Getenv("APPDATA")
        if appData == "" {
            appData = os.ExpandEnv("$USERPROFILE\\AppData\\Roaming")
        }
        configDir = filepath.Join(appData, "Moly")
        
    case "darwin":
        configDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Moly")
        
    default:
        if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
            configDir = filepath.Join(xdgHome, "moly")
        } else {
            configDir = filepath.Join(os.Getenv("HOME"), ".config", "moly")
        }
    }
    
    os.MkdirAll(configDir, 0700)
    return filepath.Join(configDir, filename)
}
```

### Testing
```bash
# Build on Linux
cd moly-go
go build -o moly
ls ~/.config/moly/moly.db  # Should exist after first run

# Build on macOS
go build -o moly
ls ~/Library/Application\ Support/Moly/moly.db  # Should exist

# Build on Windows (PowerShell)
go build -o moly.exe
ls "$env:APPDATA\Moly\moly.db"  # Should exist
```

---

## Task 1.2: Configuration Management

### What to do
Add support for environment variables and config files

### Create new file: `moly-go/config.go`

```go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	Port           string `json:"port"`
	Host           string `json:"host"`
	LogLevel       string `json:"log_level"`
	CORSProxyPort  string `json:"cors_proxy_port"`
}

var DefaultConfig = Config{
	Port:           ":11436",
	Host:           "127.0.0.1",
	LogLevel:       "info",
	CORSProxyPort:  ":11435",
}

func LoadConfig() Config {
	config := DefaultConfig
	
	// Environment variables override defaults
	if port := os.Getenv("MOLY_PORT"); port != "" {
		config.Port = port
	}
	if host := os.Getenv("MOLY_HOST"); host != "" {
		config.Host = host
	}
	if level := os.Getenv("MOLY_LOG_LEVEL"); level != "" {
		config.LogLevel = level
	}
	
	// Try to load from config file (optional)
	configFile := getConfigFilePath()
	if data, err := os.ReadFile(configFile); err == nil {
		json.Unmarshal(data, &config)
	}
	
	return config
}

func getConfigFilePath() string {
	var configDir string
	
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = os.ExpandEnv("$USERPROFILE\\AppData\\Roaming")
		}
		configDir = filepath.Join(appData, "Moly")
	case "darwin":
		configDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Moly")
	default:
		if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
			configDir = filepath.Join(xdgHome, "moly")
		} else {
			configDir = filepath.Join(os.Getenv("HOME"), ".config", "moly")
		}
	}
	
	return filepath.Join(configDir, "moly.config.json")
}
```

### Update `main.go`

Find the `func main()` section and replace hardcoded values:

**Before**:
```go
func main() {
	log.SetFlags(log.Lshortfile)
	...
	addr := Host + Port  // Hardcoded constants
	log.Printf("[Moly] Listening on %s", addr)
}
```

**After**:
```go
func main() {
	log.SetFlags(log.Lshortfile)
	
	config := LoadConfig()  // Load config
	
	// Use config values instead of constants
	addr := config.Host + config.Port
	log.Printf("[Moly] Listening on %s", addr)
	log.Printf("[Moly] Log level: %s", config.LogLevel)
}
```

### Testing
```bash
# Default behavior (no env vars)
go run . 
# Should start on 127.0.0.1:11436

# Override with env var
MOLY_PORT=:9999 go run .
# Should start on 127.0.0.1:9999

# Create config file
cat > ~/.config/moly/moly.config.json << 'CONF'
{
  "port": ":8888",
  "log_level": "debug"
}
CONF
go run .
# Should start on 127.0.0.1:8888 with debug logging
```

---

## Task 1.3: CORS Proxy Path Robustness

### What to do
Make proxy discovery more robust

### File: `moly-go/main.go`

Find `func startCORSProxy()` (around line 26)

**Replace**:
```go
func startCORSProxy() error {
	proxyScript, err := findCORSProxyScript()
	if err != nil {
		log.Printf("[Moly] WARNING: %v", err)
		log.Printf("[Moly] Continuing without CORS proxy")
		return nil
	}

	proxyCmd = exec.Command("node", proxyScript)
	...rest stays same...
}

func findCORSProxyScript() (string, error) {
	// 1. Check environment variable
	if proxyPath := os.Getenv("MOLY_PROXY_PATH"); proxyPath != "" {
		if _, err := os.Stat(proxyPath); err == nil {
			return proxyPath, nil
		}
	}
	
	// 2. Check binary directory
	exePath, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "moly-proxy.js")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	
	// 3. Check source directory (dev mode)
	candidate := "../moly-proxy/bin/moly-proxy.js"
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	
	return "", fmt.Errorf("CORS proxy not found. Set MOLY_PROXY_PATH if using custom location")
}
```

### Testing
```bash
# With MOLY_PROXY_PATH set
export MOLY_PROXY_PATH="/path/to/moly-proxy.js"
go run .
# Should find proxy

# Without env var (should find in dev directory)
go run .
# Should also work
```

---

## Task 1.4: Remove Unused Endpoints

### What to do
Clean up backend by removing 20+ unused endpoints

### Identify unused endpoints
```bash
cd moly-go
# Show all endpoints
grep "http.HandleFunc" main.go | grep -oE '"/api/[^"]*'

# Check each one against extension
for endpoint in $(grep "http.HandleFunc" main.go | grep -oE '"/api/[^"]*'); do
  grep -r "$endpoint" ../moly-extension/src || echo "UNUSED: $endpoint"
done
```

### Remove endpoints from `main.go`

Delete these lines:
- `http.HandleFunc("/api/analytics/contacts", ...)`
- `http.HandleFunc("/api/analytics/patterns", ...)`
- `http.HandleFunc("/api/analytics/summary", ...)`
- `http.HandleFunc("/api/analytics/tone", ...)`
- `http.HandleFunc("/api/analytics/topics", ...)`
- `http.HandleFunc("/api/analyze-context", ...)`
- `http.HandleFunc("/api/analyze-mode-shift", ...)`
- `http.HandleFunc("/api/api-key", ...)`
- `http.HandleFunc("/api/chat", ...)`
- And others from the unused list

### Delete handler functions
Remove the corresponding handler functions:
- `func handleAnalyticsContacts(w http.ResponseWriter, r *http.Request) {}`
- `func handleChat(w http.ResponseWriter, r *http.Request) {}`
- etc.

### Create documentation
Create `DEPRECATED_ENDPOINTS.md`:
```markdown
# Removed Endpoints

These endpoints were defined in the backend but never used by the frontend.

## Removed in Phase 1:
- /api/analytics/* - Analytics not implemented in frontend
- /api/chat - Legacy chat endpoint
- /api/draft-message - Draft functionality incomplete
- [Full list...]

## Reason:
Dead code creates maintenance burden and confusion.
```

---

## Verification Checklist

After completing all 4 tasks:

- [ ] `getConfigDir()` function added to `database.go`
- [ ] `config.go` file created
- [ ] `main.go` uses `LoadConfig()`
- [ ] `findCORSProxyScript()` function added
- [ ] Unused endpoints removed
- [ ] Code compiles: `go build ./moly-go`
- [ ] Test on Linux: `./moly` creates DB in ~/.config/moly/
- [ ] Test on macOS: `./moly` creates DB in ~/Library/Application\ Support/Moly/
- [ ] Test on Windows: `./moly.exe` creates DB in %APPDATA%\Moly\
- [ ] Environment variables work: `MOLY_PORT=:9999 ./moly`
- [ ] Config file works: Create moly.config.json and verify it loads

---

## Time Estimate
- Task 1.1: 2 hours
- Task 1.2: 3 hours  
- Task 1.3: 2 hours
- Task 1.4: 1 hour
- Testing: 2 hours
- **Total: 10 hours (1-2 days for 1 developer)**

---

## Commit

After all tasks completed:
```bash
git add moly-go/
git add DEPRECATED_ENDPOINTS.md
git commit -m "fix: Phase 1 - Cross-platform support and cleanup

- Add platform-specific database paths (Windows/macOS/Linux)
- Add configuration management (env vars + config files)
- Make CORS proxy discovery more robust
- Remove 20+ unused backend endpoints
- All platforms now supported in setup"
```

