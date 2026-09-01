# Moly Build Guide

Instructions for building production binaries and packages.

---

## Prerequisites

### All Platforms
```bash
git clone https://github.com/Nireus79/Moly.git
cd Moly
```

### Python 3.10+ (Required for native host)
```bash
# Check version
python3 --version
# Should be 3.10 or higher
```

### PyInstaller (Convert Python → Binary)
```bash
pip install pyinstaller
```

### Platform-Specific Tools

**macOS**:
- Xcode Command Line Tools: `xcode-select --install`
- Create DMG: Built-in tools (hdiutil)

**Linux**:
- AppImage tool: `sudo apt install appimagetool` (Ubuntu)
- OR: Download from https://appimage.org/

**Windows**:
- 7-Zip or WinRAR (for packaging)
- NSIS (Nullsoft Installer) - optional for EXE installer

---

## Step 1: Build Native Host Binaries

### macOS (Intel x64)

```bash
cd moly-installer/native-host

# Using build script
./build-macos.sh

# Output: moly-installer-macos.dmg
# Verify: file moly-installer-macos.dmg
```

**Manual build** (if script fails):
```bash
python3 -m PyInstaller \
  --onefile \
  --console \
  --name moly-native-host \
  moly-host.py

# Copy to system path
sudo cp dist/moly-native-host /usr/local/bin/moly-native-host
sudo chmod +x /usr/local/bin/moly-native-host
```

### macOS (Apple Silicon)

```bash
# Same as Intel build - PyInstaller auto-detects ARM64
./build-macos.sh

# Creates: moly-installer-macos-arm64.dmg
# Verify both architectures:
file dist/moly-native-host
# Output: Mach-O 64-bit executable arm64
```

### Linux x64

```bash
cd moly-installer/native-host

# Using build script
./build-linux.sh

# Output: moly-installer-linux-x64.tar.gz
# Verify:
tar -tzf moly-installer-linux-x64.tar.gz | head
```

**Manual build**:
```bash
python3 -m PyInstaller \
  --onefile \
  --console \
  --name moly-native-host \
  moly-host.py

# Create tarball
tar -czf moly-installer-linux-x64.tar.gz dist/moly-native-host

# Install to system
sudo cp dist/moly-native-host /usr/local/bin/moly-native-host
sudo chmod +x /usr/local/bin/moly-native-host
```

### Windows x64

```bash
cd moly-installer/native-host

# Using build script (PowerShell as Administrator)
.\build-windows.bat

# Output: moly-installer-windows-x64.exe
```

**Manual build** (Command Prompt as Administrator):
```cmd
python -m PyInstaller ^
  --onefile ^
  --console ^
  --name moly-native-host ^
  moly-host.py

mkdir "C:\Program Files\Moly"
copy dist\moly-native-host.exe "C:\Program Files\Moly\moly-native-host.exe"
```

---

## Step 2: Verify Binaries

### Test Each Binary

**macOS**:
```bash
# Direct test
/usr/local/bin/moly-native-host

# Or from DMG
hdiutil mount moly-installer-macos.dmg
cd /Volumes/Moly\ Installer
./Moly\ Installer.app/Contents/MacOS/moly-installer ping
```

**Linux**:
```bash
# Test after extraction
tar -xzf moly-installer-linux-x64.tar.gz
./moly-installer-linux-x64/moly-native-host ping

# Test system install
/usr/local/bin/moly-native-host --help
```

**Windows**:
```cmd
"C:\Program Files\Moly\moly-native-host.exe" --help
```

All should respond to "ping" action with `{"pong": true}`

---

## Step 3: Create GitHub Release

### Create Git Tag
```bash
git tag -a v1.0.0 -m "Release v1.0.0 - Complete one-click installer"
git push origin v1.0.0
```

### Upload to GitHub

Go to https://github.com/Nireus79/Moly/releases/new

1. Click "Create a new release"
2. Select tag: `v1.0.0`
3. Title: "Moly v1.0.0 - One-Click Installer"
4. Description:
```markdown
Complete one-click cross-platform installer for Moly.

## What's New
- Automatic Ollama installation and setup
- CORS proxy auto-installation
- Native service control via Moly UI
- Cross-platform auto-start (macOS/Linux/Windows)
- One-click model management

## Files
- moly-installer-macos.dmg - macOS Intel
- moly-installer-macos-arm64.dmg - macOS Apple Silicon
- moly-installer-linux-x64.tar.gz - Linux x64
- moly-installer-windows-x64.exe - Windows x64

## Installation
See INSTALLER-README.md for detailed instructions.

## Requirements
- macOS 10.13+, Ubuntu 20.04+, Windows 10+
- 4GB RAM minimum
- 10GB free disk space
```

5. Upload files:
   - `moly-installer-macos.dmg`
   - `moly-installer-macos-arm64.dmg`
   - `moly-installer-linux-x64.tar.gz`
   - `moly-installer-windows-x64.exe`

6. Click "Publish release"

---

## Step 4: Update Download URLs

After GitHub release is created, update download URLs:

**File**: `moly-installer/src/downloadManager.js`

The URLs should already be correct (they point to v1.0.0 release), but verify:

```javascript
// macOS Intel
url = 'https://github.com/Nireus79/Moly/releases/download/v1.0.0/moly-installer-macos.dmg';

// macOS Apple Silicon
url = 'https://github.com/Nireus79/Moly/releases/download/v1.0.0/moly-installer-macos-arm64.dmg';

// Linux
url = 'https://github.com/Nireus79/Moly/releases/download/v1.0.0/moly-installer-linux-x64.tar.gz';

// Windows
url = 'https://github.com/Nireus79/Moly/releases/download/v1.0.0/moly-installer-windows-x64.exe';
```

---

## Step 5: Test Installer on Real Systems

### macOS Testing

```bash
# Mount DMG
hdiutil mount moly-installer-macos.dmg

# Run installer
cd /Volumes/Moly\ Installer
./Moly\ Installer

# Follow prompts:
# 1. Accept system requirements
# 2. Choose provider (Ollama recommended)
# 3. Select model (Mistral 7B)
# 4. Enable auto-start

# Verify installation
ls -la /Applications/Ollama.app
ls -la /usr/local/bin/moly-proxy
ls -la /usr/local/bin/moly-native-host

# Check auto-start
cat ~/Library/LaunchAgents/com.ollama.plist | head
```

### Linux Testing

```bash
# Extract installer
tar -xzf moly-installer-linux-x64.tar.gz
cd moly-installer-linux-x64

# Run installer
./install.sh

# Or use as one-file installer
chmod +x moly-installer-linux-x64
./moly-installer-linux-x64

# Verify installation
which ollama
which moly-proxy
which moly-native-host

# Check auto-start
systemctl status ollama
systemctl status moly-proxy
```

### Windows Testing

```cmd
# Run installer as Administrator
moly-installer-windows-x64.exe

# Follow wizard prompts

# Verify installation
dir "C:\Program Files\Moly"
dir "%LOCALAPPDATA%\Programs\Ollama"

# Check auto-start in Task Scheduler
taskmgr.exe
# Look for: \Microsoft\Windows\Ollama\Start Ollama
# Look for: \Microsoft\Windows\Moly\Start Proxy
```

---

## Troubleshooting Build

### PyInstaller Issues

**Problem**: "ModuleNotFoundError: No module named '_bz2'"
```bash
# Solution: Install missing dependencies
python3 -m pip install --upgrade pip
python3 -m pip install pyinstaller
```

**Problem**: "Cannot find 'codesign'"  (macOS)
```bash
# Solution: Install Xcode Command Line Tools
xcode-select --install
```

**Problem**: "Permission denied" (Linux/macOS)
```bash
# Solution: Make script executable
chmod +x build-linux.sh
chmod +x build-macos.sh
```

### Binary Issues

**Problem**: Binary won't run
```bash
# Check if executable
file moly-native-host

# Try running directly
./moly-native-host ping

# Check Python runtime
ldd ./moly-native-host  # Linux
otool -L ./moly-native-host  # macOS
```

**Problem**: "Command not found" after installation
```bash
# Check path
which moly-native-host
echo $PATH

# Add to PATH if needed
export PATH="/usr/local/bin:$PATH"
```

---

## Build Checklist

- [ ] All prerequisites installed
- [ ] Native host builds for macOS Intel
- [ ] Native host builds for macOS ARM64
- [ ] Native host builds for Linux
- [ ] Native host builds for Windows
- [ ] All binaries tested locally
- [ ] GitHub release created
- [ ] All files uploaded to release
- [ ] Download URLs verified
- [ ] Tested installer on macOS Intel
- [ ] Tested installer on macOS ARM64
- [ ] Tested installer on Linux
- [ ] Tested installer on Windows
- [ ] All tests passed without errors
- [ ] Ready for public release

---

## Next Steps

After successful build and testing:

1. Submit extension to Chrome Web Store
2. Create user documentation
3. Announce release
4. Monitor for issues
5. Plan v1.1 improvements

---

**Time estimate**: 2-3 weeks for build + testing
**Owner**: Nireus79
**Repository**: https://github.com/Nireus79/Moly
