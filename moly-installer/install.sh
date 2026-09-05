#!/bin/bash
# Moly Universal Installer for Linux and macOS
# Handles: Binary installation, configuration setup, native messaging registration
# Supports: Chrome, Brave, and Chromium-based browsers

set -e

MOLY_VERSION="1.0.0"
OS=$(uname -s)
ARCH=$(uname -m)
INSTALL_DIR=""
CONFIG_DIR=""
CHROME_NMH=""
BRAVE_NMH=""
EXTENSION_ID="${EXTENSION_ID:-jkvuyxvgeivlakjahixagdztxvrcpzbc}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS and set paths
case "$OS" in
    Linux*)
        echo -e "${GREEN}Detected Linux${NC}"
        INSTALL_DIR="${HOME}/.local/bin"
        CONFIG_DIR="${XDG_CONFIG_HOME:-${HOME}/.config}/moly"
        CHROME_NMH="${HOME}/.config/google-chrome/NativeMessagingHosts"
        BRAVE_NMH="${HOME}/.config/BraveSoftware/Brave-Browser/NativeMessagingHosts"
        ;;
    Darwin*)
        echo -e "${GREEN}Detected macOS${NC}"
        INSTALL_DIR="/usr/local/bin"
        CONFIG_DIR="${HOME}/Library/Application Support/Moly"
        CHROME_NMH="${HOME}/Library/Application Support/Google/Chrome/NativeMessagingHosts"
        BRAVE_NMH="${HOME}/Library/Application Support/BraveSoftware/Brave-Browser/NativeMessagingHosts"
        ;;
    *)
        echo -e "${RED}Error: Unsupported OS: $OS${NC}"
        echo "Supported: Linux, macOS"
        exit 1
        ;;
esac

echo "============================================"
echo "Moly Installer v$MOLY_VERSION"
echo "OS: $OS ($ARCH)"
echo "============================================"
echo ""

# Check if binary exists
if [ ! -f "./moly" ]; then
    echo -e "${RED}Error: moly binary not found in current directory${NC}"
    echo "Please run this installer from the directory containing the 'moly' binary"
    exit 1
fi

echo -e "${YELLOW}→ Creating directories...${NC}"
mkdir -p "$INSTALL_DIR" || {
    echo -e "${RED}Error: Failed to create $INSTALL_DIR${NC}"
    echo "Try: mkdir -p $INSTALL_DIR"
    exit 1
}

mkdir -p "$CONFIG_DIR" || {
    echo -e "${RED}Error: Failed to create $CONFIG_DIR${NC}"
    exit 1
}

echo -e "${YELLOW}→ Installing Moly binary...${NC}"
cp ./moly "$INSTALL_DIR/moly" || {
    echo -e "${RED}Error: Failed to copy binary to $INSTALL_DIR${NC}"
    exit 1
}
chmod +x "$INSTALL_DIR/moly"
echo -e "${GREEN}✓ Binary installed to $INSTALL_DIR/moly${NC}"

# Check if $INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Warning: $INSTALL_DIR is not in your PATH${NC}"
    echo "Add this line to your shell profile (~/.bashrc, ~/.zshrc, etc):"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

# Setup native messaging for Chrome and Brave
echo ""
echo -e "${YELLOW}→ Setting up native messaging...${NC}"

setup_native_messaging() {
    local nmh_dir="$1"
    local browser_name="$2"

    mkdir -p "$nmh_dir" || {
        echo -e "${YELLOW}Warning: Could not create $nmh_dir for $browser_name${NC}"
        return 1
    }

    local manifest_file="$nmh_dir/com.moly.backend_host.json"

    cat > "$manifest_file" << EOF
{
  "name": "com.moly.backend_host",
  "description": "Moly Backend Launcher",
  "path": "$INSTALL_DIR/moly",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://${EXTENSION_ID}/"
  ]
}
EOF

    chmod 644 "$manifest_file"
    echo -e "${GREEN}✓ Native messaging configured for $browser_name${NC}"
    echo "  Manifest: $manifest_file"
}

# Setup for Chrome
if [ -n "$CHROME_NMH" ]; then
    setup_native_messaging "$CHROME_NMH" "Chrome" || {
        echo -e "${YELLOW}Warning: Chrome native messaging not configured${NC}"
    }
fi

# Setup for Brave
if [ -n "$BRAVE_NMH" ]; then
    setup_native_messaging "$BRAVE_NMH" "Brave" || {
        echo -e "${YELLOW}Warning: Brave native messaging not configured${NC}"
    }
fi

# Verify installation
echo ""
echo -e "${YELLOW}→ Verifying installation...${NC}"

if [ ! -f "$INSTALL_DIR/moly" ]; then
    echo -e "${RED}Error: Binary verification failed${NC}"
    exit 1
fi

if [ ! -x "$INSTALL_DIR/moly" ]; then
    echo -e "${RED}Error: Binary is not executable${NC}"
    exit 1
fi

if [ ! -d "$CONFIG_DIR" ]; then
    echo -e "${RED}Error: Config directory not created${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Installation verification passed${NC}"

# Final summary
echo ""
echo "============================================"
echo -e "${GREEN}Installation Complete!${NC}"
echo "============================================"
echo ""
echo "Moly is now ready to use:"
echo ""
echo "1. Start Moly backend:"
echo "   moly"
echo ""
echo "2. Load extension in browser:"
if [ "$OS" = "Darwin" ]; then
    echo "   Chrome:  chrome://extensions"
    echo "   Brave:   brave://extensions"
else
    echo "   Chrome:  chrome://extensions"
    echo "   Brave:   brave://extensions"
fi
echo ""
echo "3. Install the extension:"
echo "   - Click 'Load unpacked'"
echo "   - Select: $HOME/path/to/moly-extension/dist"
echo ""
echo "Configuration:"
echo "  Location: $CONFIG_DIR"
echo "  Database: $CONFIG_DIR/moly.db"
echo ""
echo "To uninstall, run:"
echo "  rm $INSTALL_DIR/moly"
echo ""
