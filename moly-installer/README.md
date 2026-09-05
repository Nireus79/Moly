# Moly Installer

This directory contains cross-platform installation scripts for Moly.

## Quick Start

### Linux & macOS

```bash
# 1. Build the Moly backend
cd ../moly-go
go build -o moly

# 2. Run the installer
cd ../moly-installer
chmod +x install.sh
./install.sh
```

### Windows

**Method 1: Using Batch Wrapper (Easiest)**

```cmd
# 1. Build the Moly backend
cd ..\moly-go
go build -o moly.exe

# 2. Run the installer (Command Prompt as Administrator)
cd ..\moly-installer
install.bat
```

**Method 2: Direct PowerShell**

```powershell
# 1. Build the Moly backend
cd ..\moly-go
go build -o moly.exe

# 2. Run the installer (PowerShell as Administrator)
cd ..\moly-installer
powershell -ExecutionPolicy Bypass -File install.ps1
```

**Important:** Administrator privileges are required for registry setup

## Installation Details

### Linux

**Install Directory:** `~/.local/bin/moly`  
**Config Directory:** `~/.config/moly/` (or `$XDG_CONFIG_HOME/moly`)  
**Database:** `~/.config/moly/moly.db`

**Native Messaging:**
- Chrome: `~/.config/google-chrome/NativeMessagingHosts/com.moly.backend_host.json`
- Brave: `~/.config/BraveSoftware/Brave-Browser/NativeMessagingHosts/com.moly.backend_host.json`

### macOS

**Install Directory:** `/usr/local/bin/moly`  
**Config Directory:** `~/Library/Application Support/Moly/`  
**Database:** `~/Library/Application Support/Moly/moly.db`

**Native Messaging:**
- Chrome: `~/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.moly.backend_host.json`
- Brave: `~/Library/Application Support/BraveSoftware/Brave-Browser/NativeMessagingHosts/com.moly.backend_host.json`

### Windows

**Install Directory:** `%APPDATA%\Moly\moly.exe`  
**Config Directory:** `%APPDATA%\Moly\`  
**Database:** `%APPDATA%\Moly\moly.db`

**Native Messaging:**
- Registry paths configured by installer
- Chrome: `HKCU:\Software\Google\Chrome\NativeMessagingHosts\com.moly.backend_host`
- Brave: `HKCU:\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\com.moly.backend_host`

## Installer Features

### install.sh (Linux/macOS)

- **OS Detection:** Automatically detects Linux vs macOS
- **Architecture Support:** Works on x86_64, ARM, etc.
- **Directory Creation:** Creates install and config directories with proper permissions
- **Native Messaging:** Sets up Chrome and Brave extension communication
- **Verification:** Checks that installation was successful
- **User-Friendly:** Color-coded output and clear instructions
- **PATH Warning:** Alerts if binary directory not in PATH

### install.ps1 (Windows - Coming Soon)

- **Registry Setup:** Configures Windows Registry for native messaging
- **PATH Configuration:** Adds to PATH if needed
- **Error Handling:** Comprehensive error messages
- **Rollback:** Can uninstall cleanly

## Configuration

### Environment Variables

The installer respects these environment variables:

**Linux/macOS:**
- `XDG_CONFIG_HOME` - Override config directory (Linux only)
- `HOME` - User home directory (automatic)
- `EXTENSION_ID` - Moly extension ID (default: `jkvuyxvgeivlakjahixagdztxvrcpzbc`)

**Windows:**
- `APPDATA` - User app data directory (automatic)
- `EXTENSION_ID` - Moly extension ID (optional)

### Moly Backend Configuration

After installation, you can configure Moly using environment variables:

```bash
# Override server port
export MOLY_PORT=:9999

# Override bind address
export MOLY_HOST=0.0.0.0

# Override log level
export MOLY_LOG_LEVEL=debug

# Override CORS proxy port
export MOLY_CORS_PROXY_PORT=:8888

# Start Moly with custom config
moly
```

Or create a config file at:
- Linux: `~/.config/moly/moly.config.json`
- macOS: `~/Library/Application Support/Moly/moly.config.json`
- Windows: `%APPDATA%\Moly\moly.config.json`

Example `moly.config.json`:
```json
{
  "port": ":11436",
  "host": "127.0.0.1",
  "log_level": "info",
  "cors_proxy_port": ":11435",
  "ollama_endpoint": "http://127.0.0.1:11434"
}
```

## Troubleshooting

### "moly binary not found"

Make sure you're running the installer from the directory containing the compiled `moly` binary:

```bash
cd moly-go
go build -o moly
cd ../moly-installer
./install.sh
```

### "command not found: moly"

The installer installed to `~/.local/bin`, which may not be in your PATH.

Add to your shell profile (`~/.bashrc`, `~/.zshrc`, etc):
```bash
export PATH="$HOME/.local/bin:$PATH"
```

Then reload: `source ~/.bashrc` or open a new terminal

### Native messaging not working

Check that the manifest file was created:

**Linux:**
```bash
cat ~/.config/google-chrome/NativeMessagingHosts/com.moly.backend_host.json
```

**macOS:**
```bash
cat ~/Library/Application\ Support/Google/Chrome/NativeMessagingHosts/com.moly.backend_host.json
```

Verify the path in the manifest points to your installed binary:
```bash
which moly
```

### Cannot create directories

The installer needs write permissions to:
- Your home directory
- `.local/bin` (Linux) or `/usr/local/bin` (macOS)

You may need to create these manually:
```bash
# Linux
mkdir -p ~/.local/bin
mkdir -p ~/.config/moly

# macOS
mkdir -p ~/Library/Application\ Support/Moly
```

## Uninstallation

### Linux/macOS

```bash
# Remove binary
rm ~/.local/bin/moly          # Linux
rm /usr/local/bin/moly        # macOS

# Remove config (optional)
rm -rf ~/.config/moly         # Linux
rm -rf ~/Library/Application\ Support/Moly  # macOS

# Remove native messaging manifests (optional)
rm -rf ~/.config/google-chrome/NativeMessagingHosts/com.moly.backend_host.json
rm -rf ~/.config/BraveSoftware/Brave-Browser/NativeMessagingHosts/com.moly.backend_host.json
```

### Windows

The installer creates an uninstall script or you can manually remove:
- `%APPDATA%\Moly\` directory
- Registry entries under `HKCU:\Software\Google\Chrome\NativeMessagingHosts\com.moly.backend_host`

## Development

### Testing the Installer

```bash
# Build the binary
cd moly-go
go build -o moly

# Test the installer (dry run)
cd ../moly-installer
bash -x install.sh  # Run with debug output
```

### Customizing Installation Paths

Edit the installer script to change default directories:

```bash
# In install.sh, modify the case statement:
case "$OS" in
    Linux*)
        INSTALL_DIR="$HOME/.local/bin"  # Change this line
        ...
```

### Platform Support

Current status:
- ✅ Linux (x86_64, ARM64)
- ✅ macOS (Intel, Apple Silicon)
- ✅ Windows (x86_64, ARM64)

## Support

For issues with the installer, check:
1. Binary was built: `ls -la ./moly`
2. Installer is executable: `chmod +x install.sh`
3. You're in the right directory: `pwd`
4. Sufficient disk space: `df -h $HOME`
5. Directory permissions: `ls -la $HOME/.local`

For Moly backend issues after installation:
```bash
# Check if Moly is running
moly --version  # or similar flag (TBD)

# Check logs
tail ~/.config/moly/moly.log  # Linux
tail ~/Library/Application\ Support/Moly/moly.log  # macOS
```
