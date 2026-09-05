@echo off
REM Moly Windows Installer - Batch Wrapper
REM This batch file makes it easier to run the PowerShell installer from Command Prompt

setlocal enabledelayedexpansion

REM Check if running with admin privileges
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: This installer requires administrator privileges
    echo.
    echo To run as administrator:
    echo 1. Press Windows + X
    echo 2. Select "Windows PowerShell (Admin)" or "Command Prompt (Admin)"
    echo 3. Run this batch file again
    echo.
    pause
    exit /b 1
)

REM Check if PowerShell is available
powershell -Command "exit" >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: PowerShell is not available on this system
    pause
    exit /b 1
)

REM Run the PowerShell installer
echo Launching Moly Windows Installer...
echo.

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0install.ps1"

if %errorlevel% equ 0 (
    echo.
    echo Installation script completed successfully.
) else (
    echo.
    echo Installation script encountered an error.
)

pause
exit /b %errorlevel%
