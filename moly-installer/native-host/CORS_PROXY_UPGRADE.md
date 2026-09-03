# CORS Proxy Upgrade - Complete Integration

## Overview

The CORS proxy has been completely rewritten in Python and merged into the native host binary. This eliminates the npm/Node.js dependency and creates a single self-contained binary per platform.

## What Changed

### Before
- CORS proxy: Separate Node.js package (`moly-proxy`)
- Requires: npm installation, internet access, Node.js runtime
- Installation: `npm install -g moly-proxy`
- Management: Separate service/autostart configuration

### After
- CORS proxy: Built-in to native host (Python)
- Requires: Nothing (standalone binary)
- Installation: Binary included in native host download
- Management: Automatic via native host autostart

## How It Works

### Native Host Binary Architecture

```
moly-native-host binary
├── Native Messaging Handler (stdin/stdout)
│   └── Handles requests from Chrome extension
└── CORS Proxy Server (background thread)
    └── Runs on 127.0.0.1:11435 (automatic start)
```

### Two Operating Modes

1. **Native Messaging Mode** (default)
   - Extension communicates with binary via stdin/stdout
   - CORS proxy starts automatically in background
   - Binary stays running during extension session

2. **Proxy-Only Mode** (--proxy-mode flag)
   - For standalone/autostart scenarios
   - Just runs the CORS proxy on 127.0.0.1:11435
   - Useful for systemd/LaunchAgent/Task Scheduler

### Autostart Configuration

#### Linux (systemd)
```ini
[Unit]
Description=Moly Native Host
After=network.target

[Service]
ExecStart=/usr/local/bin/moly-native-host --proxy-mode
Restart=on-failure
RestartSec=5s
```

#### macOS (LaunchAgent)
```xml
<dict>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/moly-native-host</string>
        <string>--proxy-mode</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
```

#### Windows (Task Scheduler)
```xml
<Actions Context="Author">
    <Exec>
      <Command>C:\Program Files\Moly\moly-native-host.exe</Command>
      <Arguments>--proxy-mode</Arguments>
    </Exec>
</Actions>
```

## Technical Implementation

### CORS Proxy Code Location
- **File**: `moly-host.py` lines 30-118
- **Handler**: `CORSProxyHandler` class
- **Port**: 127.0.0.1:11435
- **Target**: 127.0.0.1:11434 (Ollama)

### Key Functions

```python
class CORSProxyHandler(http.server.BaseHTTPRequestHandler):
    """Strips security headers and forwards to Ollama"""
    # Removes: origin, sec-*, sec-ch-*, sec-gpc
    # Adds: Access-Control-Allow-* headers
    # Forwards: All HTTP methods

def start_cors_proxy_server():
    """Starts proxy in daemon thread"""
    # Runs in background while native host handles messages
    # Automatically starts when binary runs

def check_cors_proxy_running():
    """Verifies proxy responds"""
    # Tests if reachable on port 11435

def install_cors_proxy():
    """Ensures proxy is running"""
    # Called by extension during setup
    # Returns immediately if already running
```

### Integration Points

1. **main()** - Starts proxy automatically
2. **install_cors_proxy()** - Checks/starts proxy on demand
3. **setup_all()** - Orchestrates full setup including proxy
4. **setup_autostart()** - Configures autostart with --proxy-mode
5. **proxy_mode()** - Standalone proxy execution

## Advantages

✓ **No npm dependency** - Works on systems without Node.js  
✓ **Single binary** - One download = complete system  
✓ **Automatic startup** - Proxy runs with native host  
✓ **Smaller download** - No Node.js runtime overhead  
✓ **Professional** - Matches industry standard for system tools  
✓ **Simpler installation** - No package managers needed  
✓ **Better performance** - Python HTTP server is lightweight  
✓ **Zero external dependencies** - Works offline  

## Migration Guide

### For Users

If you have the old npm-based proxy installed:

```bash
# Optional: uninstall old proxy (not required)
npm uninstall -g moly-proxy

# Install new native host
# Follow the installer instructions
# Everything is built-in now!
```

### For Developers

If building from source:

```bash
# Old workflow
npm install -g moly-proxy
# Then install native host separately

# New workflow
# Just build and run the native host
bash build-linux.sh
./releases/moly-native-host-linux-x64 --proxy-mode
```

## Testing

### Test Proxy Alone
```bash
./releases/moly-native-host-linux-x64 --proxy-mode
# Proxy runs on 127.0.0.1:11435
```

### Test With Extension
```bash
# Just use Moly extension normally
# CORS proxy starts automatically
# No manual setup needed
```

### Verify It's Working
```bash
curl http://127.0.0.1:11435/api/tags
# If Ollama is running, you get response
# If Ollama is not running, you get "Bad Gateway" (expected)
```

## File Size Comparison

| Component | Node.js Proxy | Python Proxy |
|-----------|---------------|--------------|
| Proxy alone | ~5 MB (npm) | 0 KB (built-in) |
| Node.js runtime | ~40 MB | 0 MB (included in 9 MB) |
| Total | ~45 MB | ~9 MB |
| External deps | npm, Node.js | None |

## Troubleshooting

### Proxy not starting?
1. Check: `curl http://127.0.0.1:11435/api/tags`
2. Verify native host is running
3. Check Ollama is running on 11434
4. Restart extension or run installer

### Port 11435 already in use?
- Another instance of moly-native-host is running
- Or another application is using port 11435
- Kill: `pkill moly-native-host` (Linux/macOS)
- Or: Task Manager > End Task moly-native-host.exe (Windows)

### Getting "Bad Gateway"?
- Normal if Ollama is not running
- Start Ollama and try again
- CORS proxy correctly forwards errors

## Release Notes

**v1.0.0 - CORS Proxy Built-in**
- Merged CORS proxy into native host binary
- Removed npm dependency
- Single binary per platform
- Smaller download size
- Better performance
- Zero external dependencies

## Next Steps

1. Build native host for all platforms:
   - ✓ Linux x64 (9 MB)
   - macOS ARM64 (build with `bash build-macos.sh` on M1/M2)
   - macOS x64 (build on Intel Mac)
   - Windows x64 (build with `build-windows.bat` on Windows)

2. Create GitHub releases with binaries

3. Update installer scripts to download from releases

4. Test installation flow on all platforms

5. Update documentation

## References

- **Native Host Code**: `moly-installer/native-host/moly-host.py`
- **Build Scripts**: `moly-installer/native-host/build-*.sh/bat`
- **Build Instructions**: `moly-installer/native-host/BUILD_INSTRUCTIONS.md`
- **Original Proxy**: Old Node.js proxy in `moly-proxy/` (can be archived)
