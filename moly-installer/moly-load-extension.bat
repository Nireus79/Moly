@echo off
REM Moly Extension Loader
REM Displays instructions for loading the Moly extension in Chrome/Brave

setlocal enabledelayedexpansion

set EXTENSION_PATH=%APPDATA%\Moly\extension

echo.
echo Moly Extension Loader
echo =====================
echo.
echo Extension location: %EXTENSION_PATH%
echo.
echo To load the extension:
echo 1. Open Chrome or Brave
echo 2. Go to: chrome://extensions/ (Chrome) or brave://extensions/ (Brave)
echo 3. Enable 'Developer mode' (toggle in top-right)
echo 4. Click 'Load unpacked'
echo 5. Select the folder: %EXTENSION_PATH%
echo.
echo After loading, Moly will auto-start when you click the extension icon!
echo.
pause
