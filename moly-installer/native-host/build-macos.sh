#!/bin/bash
# Build Moly Native Host for macOS

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "Building Moly Native Host for macOS..."

# Check if Python 3 is available
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is required but not installed"
    exit 1
fi

# Create the executable using PyInstaller
echo "Installing PyInstaller..."
pip3 install --user pyinstaller

echo "Building executable..."
python3 -m PyInstaller \
    --onefile \
    --console \
    --name moly-native-host \
    --add-data "moly-host.py:." \
    moly-host.py

echo "Creating app bundle..."
mkdir -p "Moly Installer.app/Contents/MacOS"
mkdir -p "Moly Installer.app/Contents/Resources"

# Create Info.plist
cat > "Moly Installer.app/Contents/Info.plist" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>moly-installer</string>
    <key>CFBundleName</key>
    <string>Moly Installer</string>
    <key>CFBundleIdentifier</key>
    <string>com.moly.installer</string>
    <key>CFBundleVersion</key>
    <string>1.0.0</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
</dict>
</plist>
EOF

# Copy executable
cp dist/moly-native-host "Moly Installer.app/Contents/MacOS/moly-installer"
chmod +x "Moly Installer.app/Contents/MacOS/moly-installer"

echo "Creating installer disk image..."
hdiutil create -volname "Moly Installer" -srcfolder . -ov -format UDZO moly-installer-macos.dmg

echo "Cleaning up..."
rm -rf dist build *.spec

echo "Build complete! Installer: moly-installer-macos.dmg"
