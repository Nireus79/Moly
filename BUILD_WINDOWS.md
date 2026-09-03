# Build Moly Native Host on Windows

Run these commands in PowerShell on a Windows machine (x64):

```powershell
# Navigate to the project
cd C:\path\to\Moly\moly-installer\native-host

# Create virtual environment
python -m venv build-env

# Activate virtual environment
.\build-env\Scripts\Activate.ps1

# If you get execution policy error, run this first:
# Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

# Install PyInstaller
pip install pyinstaller

# Build the binary
python -m PyInstaller `
    --onefile `
    --console `
    --name moly-native-host-windows-x64 `
    --distpath ".\dist-windows" `
    moly-host.py

# Prepare release folder
if (!(Test-Path releases)) { mkdir releases }

# Copy binary
Copy-Item "dist-windows\moly-native-host-windows-x64.exe" "releases\"

# Create ZIP archive
Compress-Archive -Path "releases\moly-native-host-windows-x64.exe" `
    -DestinationPath "releases\moly-native-host-windows-x64.zip" -Force

# Cleanup
Remove-Item dist-windows -Recurse -Force
Remove-Item build -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item *.spec -Force -ErrorAction SilentlyContinue

# Deactivate virtual environment
deactivate

# Verify
Get-ChildItem releases\moly-native-host-windows-x64.zip | Format-List Length, Name
```

## What to expect:

- Output: `releases\moly-native-host-windows-x64.zip` (~10 MB)

## After building:

1. Copy the ZIP file to your Linux machine
2. Extract it in: `moly-installer/native-host/releases/`
3. Tell me when done

---

**Troubleshooting:**

If you get "python: command not found":
- Download Python from https://www.python.org/downloads/ (3.11+ recommended)
- Make sure to check "Add python.exe to PATH" during installation
- Close and reopen PowerShell after installing Python

If `Compress-Archive` doesn't work, install 7-Zip and use:
```powershell
& 'C:\Program Files\7-Zip\7z.exe' a "releases\moly-native-host-windows-x64.zip" "releases\moly-native-host-windows-x64.exe"
```

If execution policy blocks the script:
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```
