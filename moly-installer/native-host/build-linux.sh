#!/bin/bash
# Build Moly Native Host for Linux (x64)
# Produces: moly-native-host-linux-x64 (standalone binary)

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "=== Building Moly Native Host for Linux (x64) ==="

# Check if Python 3 is available
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is required but not installed"
    exit 1
fi

# Install PyInstaller
echo "Installing PyInstaller..."
pip3 install --user pyinstaller

# Create the executable using PyInstaller
echo "Building standalone binary..."
python3 -m PyInstaller \
    --onefile \
    --console \
    --name moly-native-host \
    --add-data ".:." \
    --distpath "./dist-linux" \
    moly-host.py

# Copy to releases directory
echo "Preparing release..."
mkdir -p releases
cp dist-linux/moly-native-host releases/moly-native-host-linux-x64

# Make executable
chmod +x releases/moly-native-host-linux-x64

# Create tarball for distribution
echo "Creating tarball..."
tar -czf releases/moly-native-host-linux-x64.tar.gz -C releases moly-native-host-linux-x64

# Cleanup build artifacts
echo "Cleaning up..."
rm -rf dist-linux build *.spec __pycache__

echo ""
echo "✓ Build complete!"
echo "  Binary: releases/moly-native-host-linux-x64"
echo "  Tarball: releases/moly-native-host-linux-x64.tar.gz"
echo ""
echo "To test:"
echo "  ./releases/moly-native-host-linux-x64 --proxy-mode"
