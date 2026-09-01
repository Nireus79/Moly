# Moly Installer - Complete Solution

One-click installation of Ollama + Moly extension for Mac, Linux, and Windows.

## What This Does

The Moly Installer provides an automatic, platform-aware installation experience:

1. **Detects your OS** (Mac/Linux/Windows)
2. **Downloads Ollama** if needed (open-source local AI)
3. **Selects a model** (Mistral 7B recommended, 4GB)
4. **Enables auto-start** (runs on system startup)
5. **Installs CORS proxy** (allows extension ↔ Ollama communication)
6. **Sets up native bridge** (one-click "Launch Installer" button in Moly)

After setup:
- Ollama runs automatically
- Moly extension can use local AI
- Works completely offline
- No API keys needed

## Installation

### Method 1: From Chrome Extension (Easiest)

1. Install [Moly extension](https://chrome.google.com/webstore)
2. Open Extension → Settings
3. Go to "Local Models Status" section
4. Click "Start Setup" button
5. Choose your platform (auto-detected)
6. Click "Launch Installer" or "Download Installer"
7. Follow the wizard

### Method 2: Direct Download

Download for your platform:
- **macOS**: [moly-installer-macos.dmg](https://github.com/user/moly-installer/releases)
- **Linux**: [moly-installer-linux-x64](https://github.com/user/moly-installer/releases)
- **Windows**: [moly-installer-windows-x64.exe](https://github.com/user/moly-installer/releases)

Then follow platform-specific instructions below.

## Platform-Specific Installation

### macOS Installation

**Requirements:**
- macOS 10.13+ (High Sierra or newer)
- 4GB RAM minimum (8GB recommended)
- 10GB free disk space
- Intel or Apple Silicon Mac

**Steps:**

1. Download `moly-installer-macos.dmg`
2. Double-click the .dmg file
3. Drag "Moly Installer" to Applications folder
4. Open Applications → Moly Installer
5. Follow the wizard:
   - System check (may require password for auto-start)
   - Choose Ollama model (Mistral 7B recommended)
   - Configure CORS proxy
   - Install and start services

**After Installation:**
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Manually start if needed
/Applications/Ollama.app/Contents/MacOS/ollama serve
```

**Auto-Start Configuration:**
- LaunchAgent installed at: `~/Library/LaunchAgents/com.ollama.plist`
- Runs automatically on login

### Linux Installation

**Requirements:**
- Ubuntu 20.04+ (or equivalent)
- 4GB RAM minimum (8GB recommended)
- 10GB free disk space
- sudo access for system services

**Steps:**

1. Download `moly-installer-linux-x64`
2. Open terminal in Downloads folder
3. Run setup:
   ```bash
   chmod +x moly-installer-linux-x64
   ./moly-installer-linux-x64
   ```
4. Follow the wizard:
   - System check
   - Choose Ollama model
   - Configure CORS proxy
   - Setup systemd service

**After Installation:**
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Manually start if needed
systemctl start ollama

# Check status
systemctl status ollama

# View logs
sudo journalctl -u ollama -f
```

**Auto-Start Configuration:**
- Systemd service: `/etc/systemd/system/ollama.service`
- Runs automatically on boot
- Enable with: `systemctl enable ollama`

### Windows Installation

**Requirements:**
- Windows 10 (Build 1909+) or Windows 11
- 4GB RAM minimum (8GB recommended)
- 10GB free disk space
- Administrator rights (for Task Scheduler)

**Steps:**

1. Download `moly-installer-windows-x64.exe`
2. Double-click to run
3. If User Account Control prompt appears, click "Yes"
4. Follow the wizard:
   - System check
   - Choose installation directory
   - Select Ollama model
   - Configure CORS proxy
   - Setup auto-start

**After Installation:**
```powershell
# Check if Ollama is running
Invoke-WebRequest -Uri "http://localhost:11434/api/tags"

# Manually start Ollama
& "C:\Users\[YourUsername]\AppData\Local\Programs\Ollama\ollama.exe" serve

# Check status in Services app
services.msc
```

**Auto-Start Configuration:**
- Task Scheduler: `\Microsoft\Windows\Ollama\Start Ollama`
- Runs automatically on login

## Usage After Installation

### In Moly Extension

1. Open Moly sidebar in any dating/messaging app
2. Type your message or context
3. Click settings (bottom right)
4. Under "Local Models Status" → you should see:
   ```
   ✓ Ollama Running (1 model)
   ✓ Mistral 7B available
   ```
5. Select model in dropdown
6. Moly uses your local AI automatically

### Manual Ollama Commands

```bash
# List installed models
ollama list

# Run a model interactively
ollama run mistral

# Download another model
ollama pull llama2

# Start service
ollama serve          # (macOS/Linux)
# or Windows: Run Ollama from Start Menu

# Stop service
# macOS: System Preferences → General → Login Items → Remove Ollama
# Linux: systemctl stop ollama
# Windows: Task Scheduler → Disable Ollama startup
```

## System Architecture

```
Browser (Chrome/Edge)
    ↓
Moly Extension (sidebar)
    ↓
CORS Proxy (localhost:11435)
    ↓
Ollama (localhost:11434)
    ↓
Local AI Model (Mistral 7B, etc)
```

The CORS proxy is necessary because:
- Browsers restrict cross-origin requests
- Moly needs to communicate with Ollama
- Proxy adds CORS headers to allow communication
- Runs locally, never touches your data

## Troubleshooting

### "Ollama not detected"

**macOS:**
```bash
# Check if installed
/Applications/Ollama.app/Contents/MacOS/ollama --version

# Start manually
/Applications/Ollama.app/Contents/MacOS/ollama serve
```

**Linux:**
```bash
# Check if installed
which ollama

# Start service
sudo systemctl start ollama
```

**Windows:**
```powershell
# Check installation directory
dir "C:\Users\[YourUsername]\AppData\Local\Programs\Ollama"

# Start from Command Prompt
"C:\Users\[YourUsername]\AppData\Local\Programs\Ollama\ollama.exe" serve
```

### "Model download failed"

**Solution:**
```bash
# Download manually
ollama pull mistral

# Check available models
ollama list

# Switch model in Moly Settings
```

### "Cannot connect to Ollama"

**Check if running:**
```bash
# macOS/Linux
curl http://localhost:11434/api/tags

# Windows PowerShell
Invoke-WebRequest -Uri "http://localhost:11434/api/tags"
```

**If not running, start manually:**
```bash
# macOS
/Applications/Ollama.app/Contents/MacOS/ollama serve

# Linux
ollama serve

# Windows
# Open Start Menu → Ollama → Run
```

### "Moly extension not detecting Ollama"

1. Make sure Ollama is running
2. Open Moly Settings
3. Click "Refresh Models" button
4. Wait 5-10 seconds for detection
5. If still not working:
   - Restart Chrome
   - Check browser console (F12) for errors
   - Verify Ollama is on localhost:11434

### "CORS Proxy error"

**If seeing network errors about localhost:11435:**

The extension tries 2 ways to connect:
1. Via CORS proxy (localhost:11435) - requires `moly-proxy` running
2. Direct to Ollama (localhost:11434) - fallback, always works

If proxy isn't running, direct connection is used automatically.

To start the proxy manually:
```bash
npm install -g moly-proxy
moly-proxy
```

### "Model very slow / hanging"

This usually means:
- Model is too large for your hardware
- System is out of memory
- Disk is full

**Solution:**
1. Try a smaller model:
   ```bash
   ollama pull tinyllama  # 1.1GB, faster
   ```

2. Close other applications
3. Wait for response (first query takes longer)

### "Auto-start not working"

**macOS:**
```bash
# Check LaunchAgent
cat ~/Library/LaunchAgents/com.ollama.plist

# Restart login agent
launchctl unload ~/Library/LaunchAgents/com.ollama.plist
launchctl load ~/Library/LaunchAgents/com.ollama.plist
```

**Linux:**
```bash
# Check service
systemctl status ollama

# Enable auto-start
sudo systemctl enable ollama

# Restart
sudo systemctl restart ollama
```

**Windows:**
```powershell
# Open Task Scheduler
taskmgr.exe

# Check: \Microsoft\Windows\Ollama\Start Ollama
# Should be set to "Enabled"
```

## Uninstalling

### macOS
```bash
# Stop service
launchctl unload ~/Library/LaunchAgents/com.ollama.plist

# Delete
rm -rf /Applications/Ollama.app
rm -rf ~/.ollama
```

### Linux
```bash
# Stop service
sudo systemctl stop ollama
sudo systemctl disable ollama

# Remove
sudo apt remove ollama  # if installed via apt
# or
sudo rm /usr/local/bin/ollama
rm -rf ~/.ollama
```

### Windows
```powershell
# Control Panel → Programs → Programs and Features
# Find "Ollama" and click Uninstall
# or manually:
Remove-Item -Path "C:\Users\[YourUsername]\AppData\Local\Programs\Ollama" -Recurse
```

## System Requirements

### Minimum (for basic operation)
- 4GB RAM
- 10GB free disk space
- Recent OS (macOS 10.13+, Ubuntu 20.04+, Windows 10)

### Recommended (for smooth experience)
- 8GB RAM
- 20GB free disk space
- SSD storage
- i5/Ryzen 5 or better CPU

### Models by System

| Model | Size | RAM Needed | Speed |
|-------|------|-----------|-------|
| TinyLlama | 1.1GB | 2GB | Very Fast |
| Mistral 7B | 4GB | 4GB | Fast |
| Llama 2 7B | 3.8GB | 4GB | Fast |
| Neural Chat | 4.1GB | 4GB | Fast |
| Dolphin Mixtral | 26GB | 16GB | Slow |

## Performance Tips

### Faster inference:
- Use smaller models (TinyLlama, Phi)
- More RAM = faster (up to 64GB)
- SSD > HDD
- Close other applications
- Disable background browser tabs

### Lower resource usage:
- Use tiny models (1-2GB)
- Set swap space on Linux
- Monitor with `top` (Mac/Linux) or Task Manager (Windows)

## Security & Privacy

✓ **All local** - No data sent to internet
✓ **No API keys** - No credentials stored
✓ **Offline capable** - Works completely offline
✓ **Open source** - Ollama is fully open source
✓ **Fully encrypted** - Chat stays on your machine

The installer and Ollama:
- Never collect telemetry
- Never phone home
- Never track usage
- Never sell data

## Getting Help

### Check Logs

**macOS:**
```bash
log stream --predicate 'process contains "ollama"'
```

**Linux:**
```bash
sudo journalctl -u ollama -f
```

**Windows:**
```powershell
Get-EventLog -LogName Application -Source ollama -Newest 20
```

### Get Debug Info
```bash
# Show system info
ollama info

# Show version
ollama --version

# Full diagnostic
ollama diagnostics
```

### Online Resources
- [Ollama GitHub](https://github.com/ollama/ollama)
- [Ollama Models](https://ollama.ai/models)
- [Moly Issues](https://github.com/user/moly)
- [Troubleshooting Guide](./TROUBLESHOOTING.md)

## Building from Source

See [BUILD.md](./BUILD.md) for instructions to build the installer yourself.

## License

Moly: MIT License  
Ollama: MIT License  
Dependencies: See respective licenses

## Contributing

Issues and PRs welcome! See [CONTRIBUTING.md](../CONTRIBUTING.md)

---

**Last Updated**: 2026-09-01  
**Version**: 1.0.0  
**Status**: Production Ready
