# Moly Native Host Installers

These platform-specific installers automate the complete setup of Moly's native host component.

## Overview

Each installer handles:
- ✓ Downloading the native host binary
- ✓ Extracting to proper system location
- ✓ Setting up native messaging bridge
- ✓ Configuring auto-start service
- ✓ Creating Moly data folders
- ✓ Cleaning up temporary files

**Result**: Users download one file, run it once, and Moly is fully installed.

---

## Installation Files

### Linux: `install-linux.sh`

**Requirements:**
- Linux (any distribution)
- Bash shell
- curl or wget
- sudo access

**Installation:**
```bash
# Download
curl -L -o ~/Downloads/moly-install.sh \
  https://github.com/Nireus79/Moly/releases/download/v1.0.0/install-linux.sh

# Make executable
chmod +x ~/Downloads/moly-install.sh

# Run (will prompt for password)
~/Downloads/moly-install.sh
```

**What it does:**
- Downloads binary from GitHub
- Installs to `/usr/local/bin/moly-native-host`
- Creates `~/.local/share/moly/` data directory
- Sets up native messaging at `~/.config/google-chrome/NativeMessagingHosts/`
- Configures systemd auto-start (optional)

**Data locations:**
- Binary: `/usr/local/bin/moly-native-host`
- Config: `~/.config/google-chrome/NativeMessagingHosts/com.moly.native_host.json`
- Data: `~/.local/share/moly/`

---

### macOS: `install-macos.sh`

**Requirements:**
- macOS 10.12 or later
- Bash shell
- curl
- sudo access

**Installation:**
```bash
# Download
curl -L -o ~/Downloads/moly-install.sh \
  https://github.com/Nireus79/Moly/releases/download/v1.0.0/install-macos.sh

# Make executable
chmod +x ~/Downloads/moly-install.sh

# Run (will prompt for password)
~/Downloads/moly-install.sh
```

**What it does:**
- Detects architecture (Intel x64 or Apple Silicon ARM64)
- Downloads correct binary for your Mac
- Installs to `/usr/local/bin/moly-native-host`
- Creates `~/Library/Application Support/Moly/` data directory
- Sets up native messaging
- Configures LaunchAgent for auto-start

**Data locations:**
- Binary: `/usr/local/bin/moly-native-host`
- Config: `~/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.moly.native_host.json`
- LaunchAgent: `~/Library/LaunchAgents/com.moly.native-host.plist`
- Data: `~/Library/Application Support/Moly/`

**Note on Security:**
- LaunchAgent allows Moly to auto-start when you log in
- You can remove it anytime: `rm ~/Library/LaunchAgents/com.moly.native-host.plist`

---

### Windows: `install-windows.bat`

**Requirements:**
- Windows 10 or later
- Administrator access

**Installation:**

1. Download: Right-click and save as:
   ```
   https://github.com/Nireus79/Moly/releases/download/v1.0.0/install-windows.bat
   ```

2. Right-click `install-windows.bat` → "Run as administrator"

3. Click "Yes" if prompted by User Account Control

**What it does:**
- Downloads binary from GitHub
- Installs to `C:\Program Files\Moly\`
- Creates `%APPDATA%\Moly\` data directory
- Sets up native messaging in Windows registry
- Configures Task Scheduler for auto-start
- Cleans up temporary files

**Data locations:**
- Binary: `C:\Program Files\Moly\moly-native-host.exe`
- Config: `C:\Program Files\Moly\com.moly.native_host.json`
- Task: Task Scheduler → "Moly Native Host"
- Data: `%APPDATA%\Moly\` (usually `C:\Users\YourUsername\AppData\Roaming\Moly\`)

**Note on Permissions:**
- Task Scheduler runs with highest privileges
- This is needed for Ollama service management
- You can disable it anytime in Task Scheduler

---

## Uninstallation

### Linux
```bash
# Remove binary
sudo rm /usr/local/bin/moly-native-host

# Remove native messaging config
rm ~/.config/google-chrome/NativeMessagingHosts/com.moly.native_host.json

# Remove data directory (optional)
rm -rf ~/.local/share/moly/
```

### macOS
```bash
# Remove binary
sudo rm /usr/local/bin/moly-native-host

# Remove native messaging config
rm ~/Library/Application\ Support/Google/Chrome/NativeMessagingHosts/com.moly.native_host.json

# Remove LaunchAgent (auto-start)
rm ~/Library/LaunchAgents/com.moly.native-host.plist

# Remove data directory (optional)
rm -rf ~/Library/Application\ Support/Moly/
```

### Windows
1. Open Task Scheduler
2. Find "Moly Native Host" and delete it
3. Delete folder: `C:\Program Files\Moly\`
4. Delete folder: `C:\Users\YourUsername\AppData\Roaming\Moly\` (optional)

Or via command line (as administrator):
```batch
schtasks /delete /tn "Moly Native Host" /f
rmdir /s /q "C:\Program Files\Moly\"
```

---

## Troubleshooting

### "Permission denied" on Linux/macOS
Make sure the script is executable:
```bash
chmod +x install-linux.sh
chmod +x install-macos.sh
```

### "Not found in PATH" error
The installer didn't complete successfully. Try again and look for error messages:
```bash
# For detailed output, run with bash -x
bash -x ~/Downloads/install-linux.sh
```

### Chrome doesn't detect native host
1. Restart Chrome completely
2. Check native messaging config file exists
3. On Linux/macOS: `cat ~/.config/google-chrome/NativeMessagingHosts/com.moly.native_host.json`
4. On Windows: Check Registry: `HKCU\Software\Google\Chrome\NativeMessagingHosts`

### Extension says "Native host not available"
1. Run the installer again
2. Restart Chrome
3. Check logs: `journalctl -u moly-native-host` (Linux)

---

## Building Installers from Source

### Linux
```bash
# Already a shell script, just make it executable
chmod +x moly-installer/native-host/install-linux.sh
```

### macOS
```bash
# Already a shell script, just make it executable
chmod +x moly-installer/native-host/install-macos.sh
```

### Windows
```batch
REM Already a batch file, ready to use
REM Just download and run: install-windows.bat
```

---

## Distribution

For GitHub releases, upload as:
- `moly-install-linux.sh` (Linux)
- `moly-install-macos.sh` (macOS)
- `moly-install-windows.bat` (Windows)

Users download and run for their platform. No technical knowledge required.

---

## Security Notes

- Installers are plain text (shell/batch scripts)
- You can review them before running
- Only install from official GitHub releases
- Installers require sudo/admin only for system directories
- No telemetry or home-phone-home

---

## Support

If installation fails:
1. Check error messages carefully
2. Report issues at: https://github.com/Nireus79/Moly/issues
3. Include platform, OS version, and error message

---

*Last Updated: September 2026*
*Version: v1.0.0*
