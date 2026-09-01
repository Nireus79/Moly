#!/bin/bash
# Build Moly Native Host for Linux

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "Building Moly Native Host for Linux..."

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
    moly-host.py

echo "Preparing installer structure..."
mkdir -p build/moly-installer

# Copy files
cp dist/moly-native-host build/moly-installer/
chmod +x build/moly-installer/moly-native-host

# Create installation script
cat > build/moly-installer/install.sh << 'EOF'
#!/bin/bash
# Install Moly Native Host on Linux

set -e

echo "Installing Moly Native Host..."

# Copy binary to system path
sudo cp moly-native-host /usr/local/bin/moly-native-host
sudo chmod +x /usr/local/bin/moly-native-host

# Create native messaging host config directory
mkdir -p ~/.config/google-chrome/NativeMessagingHosts
mkdir -p ~/.config/chromium/NativeMessagingHosts

# Create host configuration
# Note: Extension ID needs to be obtained from chrome://extensions
read -p "Enter your Moly extension ID (from chrome://extensions): " EXTENSION_ID

cat > ~/.config/google-chrome/NativeMessagingHosts/com.moly.installer.json << HOSTEOF
{
  "name": "com.moly.installer",
  "description": "Moly Installer Launcher",
  "path": "/usr/local/bin/moly-native-host",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://${EXTENSION_ID}/"
  ]
}
HOSTEOF

cp ~/.config/google-chrome/NativeMessagingHosts/com.moly.installer.json \
   ~/.config/chromium/NativeMessagingHosts/com.moly.installer.json

echo "Installation complete!"
echo "Please restart Chrome for changes to take effect."
EOF

chmod +x build/moly-installer/install.sh

echo "Creating tarball..."
tar -czf moly-installer-linux-x64.tar.gz -C build moly-installer/

echo "Cleaning up..."
rm -rf dist build *.spec

echo "Build complete! Installer: moly-installer-linux-x64.tar.gz"
echo ""
echo "To install:"
echo "  1. Extract: tar -xzf moly-installer-linux-x64.tar.gz"
echo "  2. Run: cd moly-installer && ./install.sh"
