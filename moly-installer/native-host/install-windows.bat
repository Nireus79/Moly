@echo off
REM Moly Native Host Installer for Windows
REM Self-managing installer: moves to Moly folder, runs, then self-deletes

setlocal enabledelayedexpansion

set MOLY_VERSION=v1.0.0
set GITHUB_REPO=https://github.com/Nireus79/Moly
set BINARY_URL=%GITHUB_REPO%/releases/download/%MOLY_VERSION%/moly-native-host-windows-x64.zip
set INSTALL_DIR=%ProgramFiles%\Moly
set MOLY_DATA_DIR=%APPDATA%\Moly
set SCRIPT_NAME=moly-install-windows.bat

REM Create Moly data directory
if not exist "%MOLY_DATA_DIR%" mkdir "%MOLY_DATA_DIR%"

REM Auto-elevate to administrator if needed
net session >nul 2>&1
if %errorlevel% neq 0 (
    powershell -Command "Start-Process cmd -ArgumentList '/c %~f0' -Verb RunAs" >nul 2>&1
    exit /b 0
)

echo.
echo ================================
echo Moly Native Host Installer
echo Version: %MOLY_VERSION%
echo ================================
echo.

REM Create directories
echo Setting up Moly directories...
if not exist "%MOLY_DATA_DIR%" mkdir "%MOLY_DATA_DIR%"
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
echo OK: Directories created

REM Download binary
echo.
echo Downloading native host binary...
set TEMP_FILE=%TEMP%\moly-native-host.zip
powershell -Command "(New-Object Net.ServicePointManager).SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12; (New-Object Net.WebClient).DownloadFile('%BINARY_URL%', '%TEMP_FILE%')"
if %errorlevel% neq 0 (
    echo Error: Failed to download binary
    pause
    exit /b 1
)
echo OK: Downloaded successfully

REM Extract binary
echo.
echo Extracting binary...
powershell -Command "Expand-Archive -Path '%TEMP_FILE%' -DestinationPath '%INSTALL_DIR%' -Force"
if %errorlevel% neq 0 (
    echo Error: Failed to extract binary
    pause
    exit /b 1
)
echo OK: Extracted to %INSTALL_DIR%

REM Verify extraction
if not exist "%INSTALL_DIR%\moly-native-host.exe" (
    echo Error: Binary not found after extraction
    pause
    exit /b 1
)
echo OK: Binary verified

REM Setup native messaging
echo.
echo Setting up native messaging host...

REM Create manifest file
(
echo {
echo   "name": "com.moly.native_host",
echo   "description": "Moly Native Host",
echo   "path": "%INSTALL_DIR%\moly-native-host.exe",
echo   "type": "stdio",
echo   "allowed_origins": [
echo     "chrome-extension:///",
echo     "chrome-extension:///popup.html",
echo     "chrome-extension:///sidebar.html"
echo   ]
echo }
) > "%INSTALL_DIR%\com.moly.native_host.json"

if not exist "%INSTALL_DIR%\com.moly.native_host.json" (
    echo Error: Failed to create native messaging config file
    pause
    exit /b 1
)

REM Create registry entry for native messaging
reg add "HKCU\Software\Google\Chrome\NativeMessagingHosts\com.moly.native_host" /ve /d "%INSTALL_DIR%\com.moly.native_host.json" /f >nul 2>&1

if %errorlevel% neq 0 (
    echo Error: Failed to set native messaging registry entry
    pause
    exit /b 1
)

echo OK: Native messaging configured

REM Setup auto-start
echo.
echo Setting up auto-start...

REM Create scheduled task for auto-start
schtasks /create /tn "Moly Native Host" /tr "%INSTALL_DIR%\moly-native-host.exe" /sc onstart /rl highest /f >nul 2>&1

if %errorlevel% equ 0 (
    echo OK: Auto-start configured via Task Scheduler
) else (
    echo Note: Auto-start configuration skipped (not critical)
)

REM Cleanup temporary files
echo.
echo Cleaning up temporary files...
del "%TEMP_FILE%" >nul 2>&1
echo OK: Temporary files cleaned

REM Self-delete the installer script from Downloads
echo Cleaning up installer...
set DOWNLOADS_PATH=%USERPROFILE%\Downloads\%SCRIPT_NAME%
if exist "%DOWNLOADS_PATH%" (
    del "%DOWNLOADS_PATH%" >nul 2>&1
    echo OK: Installer cleaned up
) else (
    echo OK: Installer cleanup complete
)

echo.
echo ================================
echo Installation Complete!
echo ================================
echo.
echo Next steps:
echo 1. Open Chrome
echo 2. Go to chrome://extensions/
echo 3. Enable "Developer mode"
echo 4. Load unpacked - select Moly extension folder
echo 5. Open Moly Settings - Set Up Local Model
echo 6. Click "Configure Setup" to download models
echo.
echo Moly is ready to use!
echo.
pause
