# Moly Windows Installation Guide

## Prerequisites

- **Windows 7+** or **Windows Server 2008 R2+**
- **Administrator privileges** (required for registry setup)
- **.NET Framework 3.5+** (usually pre-installed)
- **PowerShell 3.0+** (built-in since Windows 7 SP1)
- One of: Google Chrome, Brave, or Chromium-based browser

## Installation Steps

### Step 1: Prepare the Binary

Build the Moly backend for Windows:

```cmd
cd moly-go
go build -o moly.exe
cd ..\moly-installer
```

**Verify the binary exists:**
```cmd
dir moly.exe
```

### Step 2: Run the Installer

#### Option A: Using install.bat (Recommended for most users)

1. Open Command Prompt **as Administrator**
   - Press `Windows Key + X`
   - Select "Command Prompt (Admin)" or "PowerShell (Admin)"

2. Navigate to the installer directory:
   ```cmd
   cd path\to\moly\moly-installer
   ```

3. Run the installer:
   ```cmd
   install.bat
   ```

4. Wait for the installation to complete. The installer will:
   - Create installation directory
   - Copy the Moly binary
   - Setup native messaging for Chrome and Brave
   - Verify the installation
   - Display completion message

#### Option B: Using PowerShell

1. Open PowerShell **as Administrator**

2. Navigate to the installer directory:
   ```powershell
   cd path\to\moly\moly-installer
   ```

3. Run the installer:
   ```powershell
   powershell -ExecutionPolicy Bypass -File install.ps1
   ```

### Step 3: Verify Installation

After installation completes successfully, verify by checking:

**Registry entries (for native messaging):**
```cmd
reg query "HKCU\Software\Google\Chrome\NativeMessagingHosts\com.moly.backend_host"
reg query "HKCU\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\com.moly.backend_host"
```

**Binary location:**
```cmd
dir "%APPDATA%\Moly\moly.exe"
```

## Configuration Locations

| Item | Path |
|------|------|
| **Installation** | `%APPDATA%\Moly\moly.exe` |
| **Config Directory** | `%APPDATA%\Moly\` |
| **Database** | `%APPDATA%\Moly\moly.db` |
| **Config File** | `%APPDATA%\Moly\moly.config.json` |

**Actual paths:**
```
C:\Users\YourUsername\AppData\Roaming\Moly\moly.exe
C:\Users\YourUsername\AppData\Roaming\Moly\moly.db
```

## Running Moly

### From Command Prompt

```cmd
cd %APPDATA%\Moly
moly.exe
```

Or directly:
```cmd
moly
```

### From PowerShell

```powershell
& "$env:APPDATA\Moly\moly.exe"
```

### As a Windows Service (Advanced)

To run Moly as a service, use [NSSM](https://nssm.cc/) or [sc.exe](https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/sc-create):

```cmd
nssm install Moly "%APPDATA%\Moly\moly.exe"
nssm start Moly
```

## Environment Variables

You can customize Moly behavior using environment variables:

```cmd
set MOLY_PORT=:9999
set MOLY_HOST=0.0.0.0
set MOLY_LOG_LEVEL=debug
set MOLY_CORS_PROXY_PORT=:8888
moly
```

Or in PowerShell:
```powershell
$env:MOLY_PORT = ":9999"
$env:MOLY_HOST = "0.0.0.0"
& "$env:APPDATA\Moly\moly.exe"
```

## Configuration File

Create `%APPDATA%\Moly\moly.config.json`:

```json
{
  "port": ":11436",
  "host": "127.0.0.1",
  "log_level": "info",
  "cors_proxy_port": ":11435",
  "ollama_endpoint": "http://127.0.0.1:11434"
}
```

Environment variables take precedence over config file settings.

## Troubleshooting

### "Administrator Privileges Required"

The installer needs admin access to write to the registry.

**Fix:**
1. Right-click `install.bat` or PowerShell window
2. Select "Run as administrator"
3. Re-run the installer

### "moly command not found"

The binary directory is not in your PATH, or you need to open a new terminal.

**Fix:**
1. Close and reopen Command Prompt/PowerShell
2. Or run with full path:
   ```cmd
   "%APPDATA%\Moly\moly.exe"
   ```

### Native Messaging Not Working

The registry entries may not be created correctly.

**Verify:**
```cmd
reg query "HKCU\Software\Google\Chrome\NativeMessagingHosts\com.moly.backend_host"
```

**Check the value contains correct path:**
Should show path to: `C:\Users\YourUsername\AppData\Roaming\Moly\moly.exe`

**Fix:** Re-run the installer as administrator.

### "PowerShell Execution Policy" Error

PowerShell blocks unsigned scripts by default.

**Fix (Temporary - this session only):**
```powershell
powershell -ExecutionPolicy Bypass -File install.ps1
```

**Fix (Permanent - all sessions):**
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### Binary Path Issues

If the binary is installed but shows wrong path in registry:

**Manual Fix:**
```cmd
reg add "HKCU\Software\Google\Chrome\NativeMessagingHosts\com.moly.backend_host" /ve /d "{\"name\":\"com.moly.backend_host\",\"path\":\"%APPDATA%\Moly\moly.exe\",\"type\":\"stdio\",\"allowed_origins\":[\"chrome-extension://jkvuyxvgeivlakjahixagdztxvrcpzbc/\"]}" /f
```

### "Access Denied" During Installation

The installer directory may be protected (e.g., in Program Files).

**Fix:**
1. Save moly.exe to your Downloads or Desktop
2. Run installer from there
3. Or use `%TEMP%` directory:
   ```cmd
   mkdir %TEMP%\moly-install
   cd %TEMP%\moly-install
   ```

### Firewall/Antivirus Warnings

Your antivirus or firewall may block Moly.

**Fix:**
1. Add `%APPDATA%\Moly\moly.exe` to your antivirus whitelist
2. Check Windows Defender Smart Screen settings
3. Ensure firewall allows localhost connections

### Browser Extension Can't Connect

Native messaging may not be properly registered.

**Verify in browser:**

**Chrome/Brave DevTools:**
1. Open chrome://extensions (or brave://extensions)
2. Find Moly extension
3. Click "Inspect" on the extension service worker
4. Check console for native messaging errors
5. Try sending a test message

**Common Issues:**
- Registry path wrong (should be `Software\Google\Chrome\...`)
- Binary path in registry contains forward slashes (should use backslashes)
- Extension ID mismatch (verify with `chrome-extension://xxxxx/`)

## Uninstallation

### Remove All Files and Registry Entries

```cmd
REM Remove binary and config
rmdir /s /q "%APPDATA%\Moly"

REM Remove native messaging registry entries
reg delete "HKCU\Software\Google\Chrome\NativeMessagingHosts\com.moly.backend_host" /f
reg delete "HKCU\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts\com.moly.backend_host" /f
```

Or in PowerShell:
```powershell
Remove-Item -Path "$env:APPDATA\Moly" -Recurse -Force -ErrorAction SilentlyContinue
Remove-ItemProperty -Path "HKCU:\Software\Google\Chrome\NativeMessagingHosts" -Name "com.moly.backend_host" -ErrorAction SilentlyContinue
Remove-ItemProperty -Path "HKCU:\Software\BraveSoftware\Brave-Browser\NativeMessagingHosts" -Name "com.moly.backend_host" -ErrorAction SilentlyContinue
```

## Getting Help

If installation fails:

1. **Check the installer output** for specific error messages
2. **Run as Administrator** if you get permission errors
3. **Verify prerequisites** (PowerShell 3.0, .NET Framework)
4. **Check Windows Event Viewer** for system errors
5. **See the main README.md** for more help

## What Gets Installed

**Registry Entries:**
- Chrome native messaging host registration
- Brave native messaging host registration

**Files:**
- `%APPDATA%\Moly\moly.exe` - Main binary
- `%APPDATA%\Moly\moly.db` - SQLite database (created on first run)
- `%APPDATA%\Moly\moly.config.json` - Configuration file (optional)

**No files added to:**
- System directories (System32, Program Files)
- Windows Registry (except native messaging host)
- Start Menu or Desktop shortcuts

All files are isolated in your user's AppData folder.

## Performance Notes

- First startup may be slow as Moly initializes the database
- Native messaging should respond within milliseconds
- CORS proxy adds ~10-20ms latency for cross-origin requests
- Database operations are synchronous (single-threaded)

## Security Notes

- Moly only accepts connections on `127.0.0.1` by default
- Registry entries are user-scoped (not system-wide)
- No data leaves your computer by default
- Extension ID limits which extension can communicate with Moly
- All configuration is local to user AppData

## Next Steps

After installation:

1. Load the Moly extension in your browser
2. Configure your LLM provider (Claude, OpenAI, Ollama)
3. Start using Moly!

See the main README.md for browser extension setup instructions.
