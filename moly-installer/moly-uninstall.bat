@echo off
setlocal enabledelayedexpansion

echo Moly Uninstaller
echo ================
echo.
echo This will remove:
echo   - Desktop application
echo   - Browser extension configuration
echo   - System integration files
echo.
echo Local models (if installed) are stored separately in %%APPDATA%%\.ollama
echo.

set /p KEEP_MODELS="Keep local models? (y/n): "

if /i "%KEEP_MODELS%"=="y" (
    set KEEP_MODELS=true
) else (
    set KEEP_MODELS=false
)

echo Uninstalling Moly...
echo.

set INSTALL_DIR=%USERPROFILE%\.local\bin
set CONFIG_DIR=%APPDATA%\moly
set NATIVE_HOST=%INSTALL_DIR%\moly-native-host.exe
set PYTHON_HOST=%~dp0native-host\moly-host.py
set USERNAME_SAFE=%USERNAME%

if exist "%NATIVE_HOST%" (
    echo Calling cleanup from native host...
    python "%PYTHON_HOST%" << EOF
{
  "action": "cleanup",
  "keep_models": !KEEP_MODELS!
}
EOF
)

echo Removing installation files...

if exist "%NATIVE_HOST%" (
    del /q "%NATIVE_HOST%" 2>nul
)

if "%KEEP_MODELS%"=="false" (
    echo Removing local models...
    if exist "%APPDATA%\.ollama\models" (
        rmdir /s /q "%APPDATA%\.ollama\models" 2>nul
    )
)

if exist "%CONFIG_DIR%" (
    rmdir /s /q "%CONFIG_DIR%" 2>nul
)

set NATIVE_MESSAGING_DIR=%APPDATA%\Google\Chrome\User Data\Default\Extensions
for /d %%D in (%NATIVE_MESSAGING_DIR%\*) do (
    if exist "%%D\com.moly.native_host.json" (
        del /q "%%D\com.moly.native_host.json" 2>nul
    )
)

echo.
echo Uninstall complete!

if "%KEEP_MODELS%"=="true" (
    echo Local models have been kept in %%APPDATA%%\.ollama
) else (
    echo Local models have been removed
)

echo.
echo To remove the browser extension:
echo   1. Open Chrome/Edge
echo   2. Go to chrome://extensions
echo   3. Find 'Moly' and click Remove
echo.

pause
