# Moly Installation Architecture

**Updated**: September 2, 2026  
**Status**: Orchestrated Setup via Native Host

---

## Overview

Moly orchestrates its own installation through native messaging. Users don't need terminal knowledge or manual installer apps.

---

## Installation Flow

### Phase 1: Detection (User Opens Moly)
```
Extension loads
  → Checks if native host installed (ping test)
  → If missing → Shows "Setup Required"
  → If available → Loads normally
```

### Phase 2: Download (User Clicks "Setup")
```
User sees "Setup Required" message
  → Clicks "Download Setup" button
  → Extension detects OS (Mac/Linux/Windows)
  → Downloads native host binary from GitHub
  → Browser shows download in notification
```

**What Gets Downloaded:**
- macOS Intel: `moly-installer-macos.tar.gz` (9 MB)
- macOS ARM64: `moly-installer-macos-arm64.tar.gz` (9 MB)
- Linux: `moly-installer-linux-x64.tar.gz` (9 MB)
- Windows: `moly-installer-windows-x64.exe` (9 MB)

### Phase 3: Install (User Runs Binary)
```
User downloads binary
  → OS file manager shows download
  → User double-clicks to execute
  → Binary runs and self-installs:
    - Copies itself to /usr/local/bin (Linux/macOS)
    - Copies itself to C:\Program Files\Moly (Windows)
    - Registers native messaging manifest
    - Configures auto-start services
    - Exits
```

### Phase 4: Verification (Extension Confirms)
```
Extension retries native host detection
  → Native host now available
  → Triggers setup completion via native messaging
  → Sets up auto-start for CORS proxy and Ollama
  → Shows "Setup Complete" message
```

---

## What Makes This "Moly-Controlled"

1. **Extension Orchestrates Everything**
   - Detects what's missing
   - Initiates download
   - Monitors installation progress
   - Triggers configuration steps

2. **Automation Within Native Host**
   - Binary self-installs (no separate installer app)
   - Auto-registers with Chrome
   - Configures auto-start automatically
   - Minimal user interaction

3. **Clear Messaging Throughout**
   - "Setup Required" → "Download Setup" → "Running Installer" → "Setup Complete"
   - User always knows what's happening
   - Error messages guide recovery

---

## Native Host Architecture

### Self-Installation Process

The native host binary (moly-native-host) has two modes:

**Startup Mode** (first run):
```
Binary starts
  → Checks if already installed
  → If not → Calls install_native_host()
  → If yes → Proceeds to native messaging loop
```

**Native Messaging Mode** (after installation):
```
Extension sends: {"action": "install", "extension_id": "xxx"}
  → Native host copies itself to system location
  → Writes native messaging manifest
  → Registers for auto-start
  → Returns success
```

### Native Messaging Manifest

**macOS** (`~/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.moly.native_host.json`):
```json
{
  "name": "com.moly.native_host",
  "path": "/usr/local/bin/moly-native-host",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://EXTENSION_ID/"
  ]
}
```

**Linux** (`~/.config/google-chrome/NativeMessagingHosts/com.moly.native_host.json`):
```json
{
  "name": "com.moly.native_host",
  "path": "/usr/local/bin/moly-native-host",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://EXTENSION_ID/"
  ]
}
```

**Windows** (`%LOCALAPPDATA%\Google\Chrome\User Data\NativeMessagingHosts\com.moly.native_host.json`):
```json
{
  "name": "com.moly.native_host",
  "path": "C:\\Program Files\\Moly\\moly-native-host.exe",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://EXTENSION_ID/"
  ]
}
```

---

## Native Messaging Actions

### After Installation

```
Extension → Native Host
{
  "action": "install",
  "extension_id": "chrome-extension-id-here"
}

Response:
{
  "success": true,
  "install_path": "/usr/local/bin/moly-native-host",
  "manifest_path": "~/.config/google-chrome/NativeMessagingHosts/com.moly.native_host.json"
}
```

### Auto-Start Configuration

```
Extension → Native Host
{
  "action": "setup-autostart"
}

Response:
{
  "success": true,
  "message": "Auto-start configured"
}
```

This creates:
- **Linux**: systemd user service `moly-proxy.service`
- **macOS**: LaunchAgent `com.moly.proxy.plist`
- **Windows**: Scheduled task "MolyProxy"

---

## User Experience Timeline

| Step | What User Sees | Duration | What's Happening |
|------|---|---|---|
| 1 | Moly loads | 1s | Extension checks for native host |
| 2 | "Setup Required" message | Instant | Native host not found |
| 3 | User clicks "Download Setup" | - | User initiates setup |
| 4 | Download notification | 1-2s | Browser downloads 9MB binary |
| 5 | User opens file manager | - | Browser shows "Downloaded" |
| 6 | User double-clicks binary | - | File manager opens |
| 7 | Binary runs | 2-3s | Self-install process runs |
| 8 | User sees completion | Instant | Binary exits cleanly |
| 9 | Extension confirms | 1-2s | Retries native host detection |
| 10 | "Setup Complete" | Instant | Ready to use Moly |

**Total Time**: ~5-10 minutes (mostly user thinking time)  
**Terminal Required**: NONE ✓

---

## Error Handling

### "Download Failed"
- Check internet connection
- Retry download
- Link to manual download on GitHub

### "Installation Failed"
- User must run binary manually
- Check file permissions
- Restart browser

### "Service Control Not Working"
- Native host installed but not responding
- Restart browser
- Check system service status

---

## Platform-Specific Details

### macOS

**Intel (x64)**:
- Download: moly-installer-macos.tar.gz
- Extract: Binary goes to `/usr/local/bin/`
- Permissions: chmod +x automatically

**Apple Silicon (ARM64)**:
- Download: moly-installer-macos-arm64.tar.gz
- Extract: Binary goes to `/usr/local/bin/`
- Architecture: Native ARM64 binary

**Auto-Start**:
- LaunchAgent created at `~/Library/LaunchAgents/com.moly.proxy.plist`
- Loads on login automatically
- Can manage in System Settings → General → Login Items

### Linux

**x64 Only**:
- Download: moly-installer-linux-x64.tar.gz
- Extract: Binary goes to `/usr/local/bin/`
- Permissions: chmod +x automatically

**Auto-Start**:
- Systemd user service: `~/.config/systemd/user/moly-proxy.service`
- Enable: `systemctl --user enable moly-proxy`
- Start: `systemctl --user start moly-proxy`
- Can also use system-wide service if needed

### Windows

**x64 Only**:
- Download: moly-installer-windows-x64.exe
- Extract: Binary goes to `C:\Program Files\Moly\`
- Permissions: Set automatically via installer

**Auto-Start**:
- Scheduled task: "MolyProxy" in Task Scheduler
- Runs on boot with user account
- Can disable in Task Scheduler if needed

---

## Testing the Setup Flow

### Test 1: Detection Works
1. Start Moly (no native host installed)
2. Should show "Setup Required"
3. Native messaging test should timeout (2s)

### Test 2: Download Works
1. Click "Download Setup"
2. Browser should download 9MB file
3. File should be executable

### Test 3: Installation Works
1. Run downloaded binary
2. Binary should exit cleanly
3. No terminal output (silent install)

### Test 4: Verification Works
1. After binary runs, click "Verify Setup"
2. Extension should find native host
3. Should show "Setup Complete"

### Test 5: Services Work
1. Click "Start Ollama" in settings
2. Should start Ollama service
3. Detection should show "Ollama Running"

---

## Security Notes

### What the Binary Does
- ✓ Copies itself to system location
- ✓ Creates configuration files
- ✓ No elevated privileges required (non-admin)
- ✓ No registry modifications (Windows)
- ✓ No system-wide services

### What Stays Local
- ✓ Chat history (local storage)
- ✓ Settings (local storage)
- ✓ API keys (encrypted storage)
- ✓ Conversation context (extension memory)

### What's Encrypted
- API keys (Zustand encrypted store)
- Sensitive settings (browser's storage.sync)
- Chat history (optional encryption)

---

## Troubleshooting

### Native Host Not Found After Installation

**On Linux**:
```bash
# Check if binary exists
ls -la /usr/local/bin/moly-native-host

# Check permissions
file /usr/local/bin/moly-native-host

# Check manifest
ls -la ~/.config/google-chrome/NativeMessagingHosts/
```

**On macOS**:
```bash
# Check if binary exists
ls -la /usr/local/bin/moly-native-host

# Check manifest
ls -la ~/Library/Application\ Support/Google/Chrome/NativeMessagingHosts/
```

### Binary Won't Execute

**Linux/macOS**:
```bash
# Make executable
chmod +x /usr/local/bin/moly-native-host

# Verify
file /usr/local/bin/moly-native-host
```

### Manifest Not Recognized

**Linux/macOS**:
```bash
# Check manifest is valid JSON
cat ~/.config/google-chrome/NativeMessagingHosts/com.moly.native_host.json

# Ensure extension_id matches
# (Get from chrome://extensions/ page)
```

---

## Future Enhancements

### v1.1
- Automatic native host update check
- Graceful downgrade if incompatible
- Better error recovery

### v2.0
- Pre-built installers for air-gapped networks
- USB portable version
- Docker container support

---

## Summary

**Before (Separate Installers)**:
- Users download installer app
- Run separate installer
- Configure everything manually
- 20+ minute process
- High friction

**Now (Moly-Orchestrated)**:
- Extension detects what's needed
- User downloads binary once
- Extension guides through it
- Automatic configuration
- 5-10 minute process
- Low friction

This is "Moly-controlled installation" - the extension orchestrates everything, users just follow simple prompts.

---

*This architecture makes Moly truly one-click and eliminates the need for separate installer apps.*
