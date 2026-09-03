@echo off
REM Build Moly Native Host for Windows (x64)
REM Produces: moly-native-host-windows-x64.exe (standalone binary)

setlocal enabledelayedexpansion

echo === Building Moly Native Host for Windows (x64) ===
echo.

REM Check if Python 3 is available
python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: Python 3 is required but not installed
    echo Visit: https://www.python.org/downloads/
    exit /b 1
)

REM Install PyInstaller
echo Installing PyInstaller...
pip install --quiet pyinstaller
if %errorlevel% neq 0 (
    echo Error: Failed to install PyInstaller
    exit /b 1
)

REM Create releases directory
if not exist releases mkdir releases

REM Build executable
echo Building standalone binary...
python -m PyInstaller ^
    --onefile ^
    --console ^
    --name moly-native-host-windows-x64 ^
    moly-host.py

if %errorlevel% neq 0 (
    echo Error: Build failed
    exit /b 1
)

REM Copy to releases
echo Preparing release...
copy dist\moly-native-host-windows-x64.exe releases\
if %errorlevel% neq 0 (
    echo Error: Failed to copy binary
    exit /b 1
)

REM Create ZIP archive for distribution
echo Creating ZIP archive...
cd releases
powershell -Command "Compress-Archive -Path moly-native-host-windows-x64.exe -DestinationPath moly-native-host-windows-x64.zip -Force"
cd ..

REM Clean up build artifacts
echo Cleaning up...
if exist dist rmdir /s /q dist
if exist build rmdir /s /q build
del *.spec 2>nul

echo.
echo ✓ Build complete!
echo.
echo  Binary: releases\moly-native-host-windows-x64.exe
echo  Archive: releases\moly-native-host-windows-x64.zip
echo.
echo To test:
echo  releases\moly-native-host-windows-x64.exe --proxy-mode
echo.
pause
