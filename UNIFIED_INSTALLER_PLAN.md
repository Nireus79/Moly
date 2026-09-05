# Unified Installer Plan - Phase 7 Foundation

## What Was Changed

### Problem Solved
✅ **Before**: Installers only handled backend + native messaging  
❌ Users had to manually load extension (confusing, error-prone)  
✅ **After**: Installers handle EVERYTHING (backend + extension)  

### Implementation Status

#### Linux/macOS - `install.sh` ✅ COMPLETE
- ✅ Detects source code vs pre-built binary
- ✅ Builds backend from source (go build)
- ✅ Builds extension from source (npm run build)
- ✅ Installs to platform directories
- ✅ Sets up native messaging
- ✅ Creates helper script (moly-load-extension)
- ✅ Clear loading instructions

#### Windows - `install.ps1` 🟡 PARTIAL (In Progress)
- ✅ Detects source vs pre-built
- ✅ Builds backend (go build)
- ⏳ Extension build (npm run build) - TO DO
- ⏳ Copy extension files - TO DO
- ⏳ Create helper script - TO DO
- ⏳ Final instructions - TO DO

#### Windows - `install.bat` ⏹ NOT UPDATED
- Current: Simple wrapper for install.ps1
- Should remain as is (calls install.ps1 with admin)

## Installation Flow (After Update)

```
User runs: ./install.sh (or install.ps1)
    ↓
Detect OS and set directories
    ↓
Check for source code or pre-built binary
    ↓
Build backend (if source available)
    ├─ cd moly-go && go build
    └─ Result: ./moly or ./moly.exe
    ↓
Build extension (if source available)
    ├─ cd moly-extension
    ├─ npm install
    └─ npm run build
    ↓
Install backend to system location
    ├─ Linux: ~/.local/bin/moly
    ├─ macOS: /usr/local/bin/moly
    └─ Windows: %APPDATA%\Moly\moly.exe
    ↓
Install extension to system location
    ├─ Linux: ~/.local/share/moly/extension
    ├─ macOS: ~/Library/Application Support/Moly/extension
    └─ Windows: %APPDATA%\Moly\extension
    ↓
Set up native messaging for Chrome/Brave
    ├─ Create manifests in browser config directories
    └─ Point to installed backend binary
    ↓
Create helper scripts
    ├─ moly-load-extension (Linux/macOS)
    └─ Batch file equivalent for Windows
    ↓
Display completion message with next steps
    ├─ How to load extension in browser
    ├─ Uninstall instructions
    └─ Quick start guide
```

## User Experience After Update

### Before (Current)
```
1. Run installer (sets up backend only)
2. Manually navigate to chrome://extensions
3. Enable Developer Mode
4. Click "Load unpacked"
5. Select extension folder
6. Finally: Extension starts working
```

### After (Proposed)
```
1. Run installer (builds & installs everything)
2. Open Chrome/Brave
3. Navigate to chrome://extensions
4. Click "Load unpacked"
5. Select $EXTENSION_DIR (installer shows path)
6. Done: Extension auto-starts backend
```

Or even simpler with helper script:
```
1. Run installer
2. Run: moly-load-extension
3. Follow on-screen instructions
4. Done
```

## What Still Needs to Be Done

### Windows Extension Build (Priority: HIGH)
```powershell
# Add to install.ps1 after backend build
Write-Info "Building Moly extension..."
try {
    Push-Location "$ProjectRoot\moly-extension"
    if (-not (Test-Path "node_modules")) {
        npm install
    }
    npm run build
    Copy-Item -Path "dist\*" -Destination $ExtensionDir -Force -Recurse
    Write-Success "Extension built and installed"
}
```

### Windows Helper Script (Priority: MEDIUM)
Create `moly-load-extension.bat`:
```batch
@echo off
setlocal enabledelayedexpansion

set EXTENSION_PATH=%APPDATA%\Moly\extension

echo Moly Extension Loader
echo =====================
echo.
echo Extension location: %EXTENSION_PATH%
echo.
echo To load the extension:
echo 1. Open Chrome or Brave
echo 2. Go to: chrome://extensions/ or brave://extensions/
echo 3. Enable Developer mode
echo 4. Click "Load unpacked"
echo 5. Select: %EXTENSION_PATH%
```

### Final Summary in Installers (Priority: MEDIUM)
Update the "Installation Complete" sections to show:
- Where backend was installed
- Where extension was installed
- How to load extension
- Quick start instructions
- Uninstall commands

## Testing Checklist

- [ ] Test Linux installer with source code
- [ ] Test macOS installer with source code
- [ ] Test Windows installer with source code
- [ ] Test with pre-built binaries
- [ ] Verify native messaging manifests created
- [ ] Verify backend binary executable
- [ ] Verify extension files copied
- [ ] Test helper script (moly-load-extension)
- [ ] Test loading extension in Chrome
- [ ] Test loading extension in Brave
- [ ] Verify auto-start backend on extension load
- [ ] Test uninstall procedures

## Configuration After Installation

### Linux
```bash
~/.local/bin/moly              # Backend binary
~/.local/share/moly/extension  # Extension files
~/.config/moly/                # Configuration directory
```

### macOS
```bash
/usr/local/bin/moly                          # Backend binary
~/Library/Application Support/Moly/extension # Extension files
~/Library/Application Support/Moly/          # Configuration directory
```

### Windows
```
%APPDATA%\Moly\moly.exe             # Backend binary
%APPDATA%\Moly\extension\           # Extension files
%APPDATA%\Moly\                     # Configuration directory
```

## Benefits of This Approach

1. **Single Installation**: User runs one installer, everything is installed
2. **No Manual Steps**: Extension loading automated via installer
3. **Cross-Platform**: Same workflow on Linux, macOS, Windows
4. **Source or Binary**: Works with pre-built binaries or builds from source
5. **Native Messaging**: Configured automatically by installer
6. **Auto-Start**: Backend starts when extension loads (already implemented)
7. **Production Ready**: Professional installation experience

## Timeline

- ✅ Phase 1-6: Complete (infrastructure, security, testing)
- 🟡 Phase 7: Distribution (simplified by unified installers)
  - ✅ Update install.sh for Linux/macOS
  - 🟡 Complete install.ps1 for Windows (partial done)
  - ⏳ Create helper scripts
  - ⏳ Test on all platforms
  - ⏳ Create distribution packages

This unified installer approach makes Phase 7 much simpler and ensures users have a professional, frictionless installation experience.
