@echo off
REM Moly One-Click Startup (Windows)
REM Now starts Go Backend + CORS Proxy automatically

setlocal enabledelayedexpansion

set SCRIPT_DIR=%~dp0
set GO_DIR=%SCRIPT_DIR%moly-go

echo.
echo ========================================
echo  MOLY STARTUP - One-Click Launch
echo ========================================
echo.

REM Check if Go binary is built
if not exist "%GO_DIR%\moly.exe" (
    echo ^ Go backend not built. Building...
    cd /d "%GO_DIR%"
    go build -o moly.exe .
    if errorlevel 1 (
        echo X Build failed
        exit /b 1
    )
    echo + Go backend built
)

REM Start Go Backend (which auto-starts CORS Proxy)
echo ^ Starting Go Backend on 11436...
cd /d "%GO_DIR%"
start "Moly Go Backend" moly.exe
timeout /t 2 >nul

echo + Go Backend started
echo + CORS Proxy auto-started on 11435
echo.

echo ========================================
echo  MOLY READY FOR TESTING
echo ========================================
echo.
echo Services running:
echo   + Go Backend      http://127.0.0.1:11436
echo   + CORS Proxy     http://127.0.0.1:11435
echo   ^ Ollama (11434)  - start manually if needed
echo.
echo Next steps:
echo   1. Open chrome://extensions
echo   2. Load unpacked → moly-extension/dist/
echo   3. Click Moly icon to test
echo   4. Type a message
echo   5. See SafetyAlert + suggestions
echo.
pause
