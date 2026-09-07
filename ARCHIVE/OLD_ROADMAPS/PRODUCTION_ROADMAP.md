# Moly Production Readiness Roadmap

**Current Status**: 2/10 - Alpha/Development  
**Target Status**: 8/10 - Ready for Commercial Release  
**Estimated Timeline**: 10-14 weeks  
**Start Date**: Now  

---

## PHASE 1: CRITICAL FIXES (Weeks 1-2)
*These BLOCK production release. Must complete before anything else.*

### 1.1 Cross-Platform Database Paths [2 days]
**Files to modify**: `moly-go/database.go`

**Current code (line 74)**:
```go
dbPath := filepath.Join(os.Getenv("HOME"), ".config", "moly", "moly.db")
```

**Change to**:
```go
dbPath := getConfigDir("moly.db")

func getConfigDir(filename string) string {
    var configDir string
    
    switch runtime.GOOS {
    case "windows":
        // Windows: %APPDATA%\Moly
        appData := os.Getenv("APPDATA")
        if appData == "" {
            appData = os.ExpandEnv("$USERPROFILE\\AppData\\Roaming")
        }
        configDir = filepath.Join(appData, "Moly")
        
    case "darwin":
        // macOS: ~/Library/Application Support/Moly
        configDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "Moly")
        
    default:
        // Linux: $XDG_CONFIG_HOME/moly or ~/.config/moly
        if xdgHome := os.Getenv("XDG_CONFIG_HOME"); xdgHome != "" {
            configDir = filepath.Join(xdgHome, "moly")
        } else {
            configDir = filepath.Join(os.Getenv("HOME"), ".config", "moly")
        }
    }
    
    // Create directory if it doesn't exist
    os.MkdirAll(configDir, 0700)
    return filepath.Join(configDir, filename)
}
```

**Testing**:
- [ ] Build on Windows
- [ ] Build on macOS
- [ ] Build on Linux
- [ ] Verify database created in correct location on each OS

---

### 1.2 Configuration Management [3 days]
**Create new file**: `moly-go/config.go`

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
	DatabasePath   string `json:"database_path"`
	LogLevel       string `json:"log_level"`
	CORSProxyPort  string `json:"cors_proxy_port"`
	OllamaEndpoint string `json:"ollama_endpoint"`
}

var DefaultConfig = Config{
	Port:           ":11436",
	Host:           "127.0.0.1",
	LogLevel:       "info",
	CORSProxyPort:  ":11435",
	OllamaEndpoint: "http://127.0.0.1:11434",
}

func LoadConfig() Config {
	config := DefaultConfig
	
	// 1. Environment variables (highest priority)
	if port := os.Getenv("MOLY_PORT"); port != "" {
		config.Port = port
	}
	if host := os.Getenv("MOLY_HOST"); host != "" {
		config.Host = host
	}
	if level := os.Getenv("MOLY_LOG_LEVEL"); level != "" {
		config.LogLevel = level
	}
	
	// 2. Config file if exists
	configFile := getConfigFilePath()
	if data, err := os.ReadFile(configFile); err == nil {
		json.Unmarshal(data, &config)
	}
	
	// 3. Database path (always use platform-specific)
	config.DatabasePath = getConfigDir("moly.db")
	
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

**Update `main.go`**:
```go
func main() {
	log.SetFlags(log.Lshortfile)
	
	// Load configuration (replaces hardcoded values)
	config := LoadConfig()
	
	// Use config.Port, config.Host, etc.
	addr := config.Host + config.Port
	log.Printf("[Moly] Starting on %s", addr)
	...
}
```

**Testing**:
- [ ] Test with environment variables
- [ ] Test with config file
- [ ] Test precedence (env > config file > defaults)
- [ ] Verify all platforms read correct defaults

---

### 1.3 CORS Proxy Path Robustness [2 days]
**File**: `moly-go/main.go` (lines 26-86)

**Current problem**: Looks for proxy relative to binary location

**Solution**:
```go
func findCORSProxyScript() (string, error) {
	// Strategy 1: Check if MOLY_PROXY_PATH is set
	if proxyPath := os.Getenv("MOLY_PROXY_PATH"); proxyPath != "" {
		if _, err := os.Stat(proxyPath); err == nil {
			return proxyPath, nil
		}
	}
	
	// Strategy 2: Look in same directory as binary
	exePath, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exePath), "moly-proxy.js")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	
	// Strategy 3: Look in installation directory (Windows Program Files, macOS /Applications)
	candidates := []string{
		"/opt/moly/moly-proxy.js",           // Linux
		"C:\\Program Files\\Moly\\moly-proxy.js", // Windows
		"/Applications/Moly/moly-proxy.js",  // macOS
	}
	
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	
	return "", fmt.Errorf("CORS proxy script not found. Set MOLY_PROXY_PATH environment variable")
}
```

**Update startCORSProxy()**:
```go
func startCORSProxy(config Config) error {
	proxyScript, err := findCORSProxyScript()
	if err != nil {
		return fmt.Errorf("CORS proxy: %v", err)
	}
	...
}
```

---

### 1.4 Remove Unused Backend Endpoints [1 day]
**Audit**: All unused `/api/*` endpoints

**Create**: `DEPRECATED_ENDPOINTS.md` documenting what was removed and why

**Files to modify**:
- `moly-go/main.go`: Remove all `http.HandleFunc()` calls for unused endpoints
- Delete handler functions: `handleChat`, `handleAnalytics*`, `handleDraftMessage`, etc.

**How to identify unused**:
```bash
grep "http.HandleFunc" moly-go/main.go | grep -oE '"/api/[^"]*' | while read endpoint; do
  grep -q "$endpoint" ../moly-extension/src || echo "UNUSED: $endpoint"
done
```

**Result**: Reduce backend code by ~30%, improve maintainability

---

## PHASE 2: CROSS-PLATFORM SUPPORT (Weeks 2-4)
*Platform-specific installers and setup automation.*

### 2.1 Unified Setup Script [3 days]
**Create**: `moly-installer/install.sh` (replaces current setup.sh)

```bash
#!/bin/bash
# Universal Moly installer for Linux/macOS

set -e

MOLY_VERSION="1.0.0"
OS=$(uname -s)
ARCH=$(uname -m)
INSTALL_DIR=""
CONFIG_DIR=""

# Detect OS and set paths
case "$OS" in
    Linux*)
        INSTALL_DIR="$HOME/.local/bin"
        CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/moly"
        ;;
    Darwin*)
        INSTALL_DIR="/usr/local/bin"
        CONFIG_DIR="$HOME/Library/Application Support/Moly"
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "Installing Moly v$MOLY_VERSION on $OS..."

# Create directories
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"

# Copy binary
cp ./moly "$INSTALL_DIR/moly"
chmod +x "$INSTALL_DIR/moly"

# Setup native messaging (platform-specific)
setup_native_messaging

echo "Installation complete!"
echo "Next: Run 'moly' from terminal or load extension from brave://extensions"
```

**Create**: `moly-installer/install.ps1` (Windows PowerShell)

```powershell
# Universal Moly installer for Windows

$MOLY_VERSION = "1.0.0"
$INSTALL_DIR = "$env:APPDATA\Moly"
$CONFIG_DIR = "$env:APPDATA\Moly\Config"

Write-Host "Installing Moly v$MOLY_VERSION on Windows..."

# Create directories
New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
New-Item -ItemType Directory -Path $CONFIG_DIR -Force | Out-Null

# Copy binary
Copy-Item -Path ".\moly.exe" -Destination "$INSTALL_DIR\moly.exe" -Force

# Setup native messaging
$registryPath = "HKCU:\Software\Google\Chrome\NativeMessagingHosts\com.moly.backend_host"
New-Item -Path $registryPath -Force | Out-Null
Set-ItemProperty -Path $registryPath -Name "(Default)" -Value "$INSTALL_DIR\moly-native-host.json"

Write-Host "Installation complete!"
Write-Host "Next: Run 'moly' from PowerShell or load extension from chrome://extensions"
```

---

### 2.2 Windows Native Messaging [2 days]
**Create**: `moly-go/setup_windows.go`

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func setupNativeMessagingWindows(extensionID string) error {
	// Read/write to Windows Registry
	// HKEY_CURRENT_USER\Software\Google\Chrome\NativeMessagingHosts\com.moly.backend_host
	// HKEY_CURRENT_USER\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\com.moly.backend_host
	
	manifest := map[string]interface{}{
		"name":             "com.moly.backend_host",
		"description":      "Moly Backend Launcher",
		"path":             filepath.Join(os.Getenv("APPDATA"), "Moly", "moly-native-host.exe"),
		"type":             "stdio",
		"allowed_origins":  []string{fmt.Sprintf("chrome-extension://%s/", extensionID)},
	}
	
	data, _ := json.MarshalIndent(manifest, "", "  ")
	
	// Write manifest to both Chrome and Brave registry keys
	browsers := []string{
		"Google\\Chrome\\NativeMessagingHosts",
		"BraveSoftware\\Brave-Browser\\NativeMessagingHosts",
	}
	
	for _, browser := range browsers {
		registryKey := fmt.Sprintf("Software\\%s\\com.moly.backend_host", browser)
		// Write to registry using syscall/registry package
		// Set default value = manifest content
	}
	
	return nil
}
```

---

### 2.3 macOS Native Messaging [2 days]
**Create**: `moly-installer/setup-macos.sh`

```bash
#!/bin/bash
# macOS-specific native messaging setup

EXTENSION_ID="${1:-jkvuyxvgeivlakjahixagdztxvrcpzbc}"
CHROME_NMH="$HOME/Library/Application Support/Google/Chrome/NativeMessagingHosts"
BRAVE_NMH="$HOME/Library/Application Support/BraveSoftware/Brave-Browser/NativeMessagingHosts"

mkdir -p "$CHROME_NMH"
mkdir -p "$BRAVE_NMH"

create_manifest() {
    local path=$1
    cat > "$path/com.moly.backend_host.json" << EOF
{
  "name": "com.moly.backend_host",
  "description": "Moly Backend Launcher",
  "path": "/usr/local/bin/moly-native-host",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://$EXTENSION_ID/"
  ]
}
