@echo off
REM Build Moly Native Host for Windows

setlocal enabledelayedexpansion

echo Building Moly Native Host for Windows...

REM Check if Python 3 is available
python --version >nul 2>&1
if %errorlevel% neq 0 (
    echo Error: Python 3 is required but not installed
    exit /b 1
)

REM Install PyInstaller
echo Installing PyInstaller...
pip install pyinstaller

REM Build executable
echo Building executable...
python -m PyInstaller ^
    --onefile ^
    --console ^
    --name moly-native-host ^
    moly-host.py

REM Create installer structure
echo Preparing installer structure...
if not exist build\moly-installer mkdir build\moly-installer

REM Copy files
copy dist\moly-native-host.exe build\moly-installer\
copy moly-host.py build\moly-installer\

REM Create installation script
echo Creating installation script...
(
echo @echo off
echo setlocal enabledelayedexpansion
echo.
echo echo Installing Moly Native Host...
echo.
echo REM Create installation directory
echo if not exist "C:\Program Files\Moly" mkdir "C:\Program Files\Moly"
echo.
echo REM Copy files
echo copy moly-native-host.exe "C:\Program Files\Moly\"
echo copy moly-host.py "C:\Program Files\Moly\"
echo.
echo REM Create registry entry
echo echo Creating registry entry...
echo powershell -Command "^
echo   $regPath = 'HKLM:\SOFTWARE\Google\Chrome\NativeMessagingHosts\com.moly.installer'; ^
echo   if -not ^(Test-Path $regPath^) { New-Item $regPath -Force ^| Out-Null }; ^
echo   Set-ItemProperty $regPath -Name '(Default)' -Value 'C:\Program Files\Moly\com.moly.installer.json' -Force; ^
echo " >nul
echo.
echo REM Create host configuration file
echo echo Creating host configuration...
echo (
echo   echo {
echo   echo   "name": "com.moly.installer",
echo   echo   "description": "Moly Installer Launcher",
echo   echo   "path": "C:\\Program Files\\Moly\\moly-native-host.exe",
echo   echo   "type": "stdio",
echo   echo   "allowed_origins": [
echo   echo     "chrome-extension://[EXTENSION_ID]/"
echo   echo   ]
echo   echo }
echo ) > "C:\Program Files\Moly\com.moly.installer.json"
echo.
echo echo Installation complete!
echo echo Please restart Chrome for changes to take effect.
echo.
echo pause
) > build\moly-installer\install.bat

REM Create 7-Zip archive
echo Creating installer archive...
if exist moly-installer-windows-x64.7z del moly-installer-windows-x64.7z

REM Check if 7-Zip is installed
where 7z >nul 2>&1
if %errorlevel% equ 0 (
    7z a -r moly-installer-windows-x64.7z build\moly-installer\*
) else (
    echo Warning: 7-Zip not found, creating ZIP instead...
    powershell -Command "Compress-Archive -Path build\moly-installer -DestinationPath moly-installer-windows-x64.zip"
)

REM Clean up
echo Cleaning up...
rmdir /s /q dist
rmdir /s /q build
del *.spec 2>nul

echo.
echo Build complete!
if exist moly-installer-windows-x64.7z (
    echo Installer: moly-installer-windows-x64.7z
) else (
    echo Installer: moly-installer-windows-x64.zip
)
echo.
echo To install:
echo   1. Extract the archive
echo   2. Right-click install.bat and select "Run as administrator"
echo   3. Follow the prompts
echo.
pause
