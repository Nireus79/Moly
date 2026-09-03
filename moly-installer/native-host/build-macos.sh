#!/bin/bash
# Build Moly Native Host for macOS
# Produces: moly-native-host-macos-arm64 or moly-native-host-macos-x64
# Requires: Python 3 and PyInstaller

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "=== Building Moly Native Host for macOS ==="

# Check if Python 3 is available
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is required but not installed"
    echo "Install via: brew install python3"
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)
if [[ "$ARCH" == "arm64" ]]; then
    BINARY_NAME="moly-native-host-macos-arm64"
    echo "Building for Apple Silicon (ARM64)..."
elif [[ "$ARCH" == "x86_64" ]]; then
    BINARY_NAME="moly-native-host-macos-x64"
    echo "Building for Intel (x86_64)..."
else
    echo "Error: Unsupported architecture: $ARCH"
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
    --name "$BINARY_NAME" \
    --add-data ".:." \
    --distpath "./dist-macos" \
    moly-host.py

# Copy to releases directory
echo "Preparing release..."
mkdir -p releases
cp "dist-macos/$BINARY_NAME" "releases/$BINARY_NAME"

# Make executable
chmod +x "releases/$BINARY_NAME"

# Create tarball for distribution
echo "Creating tarball..."
tar -czf "releases/$BINARY_NAME.tar.gz" -C releases "$BINARY_NAME"

# Cleanup build artifacts
echo "Cleaning up..."
rm -rf dist-macos build *.spec __pycache__

echo ""
echo "✓ Build complete!"
echo "  Binary: releases/$BINARY_NAME"
echo "  Tarball: releases/$BINARY_NAME.tar.gz"
echo ""
echo "To test:"
echo "  ./releases/$BINARY_NAME --proxy-mode"
