#!/bin/bash
# Moly Unified Installer for Linux and macOS
# Handles: Backend binary, extension, configuration, native messaging
# Supports: Chrome, Brave, and Chromium-based browsers

set -e

MOLY_VERSION="1.0.0"
OS=$(uname -s)
ARCH=$(uname -m)
INSTALL_DIR=""
CONFIG_DIR=""
EXTENSION_DIR=""
CHROME_NMH=""
BRAVE_NMH=""
EXTENSION_ID="${EXTENSION_ID:-jkvuyxvgeivlakjahixagdztxvrcpzbc}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

case "$OS" in
    Linux*)
        echo -e "${GREEN}Detected Linux${NC}"
        INSTALL_DIR="${HOME}/.local/bin"
        CONFIG_DIR="${XDG_CONFIG_HOME:-${HOME}/.config}/moly"
        EXTENSION_DIR="${HOME}/.moly/extension"
        CHROME_NMH="${HOME}/.config/google-chrome/NativeMessagingHosts"
        BRAVE_NMH="${HOME}/.config/BraveSoftware/Brave-Browser/NativeMessagingHosts"
        ;;
    Darwin*)
        echo -e "${GREEN}Detected macOS${NC}"
        INSTALL_DIR="${HOME}/.local/bin"
        CONFIG_DIR="${HOME}/Library/Application Support/Moly"
        EXTENSION_DIR="${HOME}/Library/Application Support/Moly/extension"
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
echo "Moly Unified Installer v$MOLY_VERSION"
echo "OS: $OS ($ARCH)"
echo "============================================"
echo ""
echo -e "${GREEN}User-level installation (no sudo needed)${NC}"
echo ""

# Check if we can build from source
if [ -d "$PROJECT_ROOT/moly-go" ] && [ -d "$PROJECT_ROOT/moly-extension" ]; then
    echo -e "${BLUE}Building from source...${NC}"
    BUILD_FROM_SOURCE=true
elif [ -f "./moly" ]; then
    echo -e "${BLUE}Using pre-built binary...${NC}"
    BUILD_FROM_SOURCE=false
else
    echo -e "${RED}Error: Cannot find binary or source to build${NC}"
    echo "This script should be run from the moly-installer directory"
    exit 1
fi

echo -e "${YELLOW}→ Creating directories...${NC}"
for dir in "$INSTALL_DIR" "$CONFIG_DIR" "$EXTENSION_DIR"; do
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir" || {
            echo -e "${RED}Error: Failed to create directory: $dir${NC}"
            exit 1
        }
    fi
done
echo -e "${GREEN}✓ Directories created${NC}"

# Build backend if needed
if [ "$BUILD_FROM_SOURCE" = true ]; then
    echo ""
    echo -e "${YELLOW}→ Building Moly backend...${NC}"
    cd "$PROJECT_ROOT/moly-go"
    go build -o moly || {
        echo -e "${RED}Error: Failed to build backend${NC}"
        exit 1
    }
    cd "$SCRIPT_DIR"
    BINARY="$PROJECT_ROOT/moly-go/moly"
else
    BINARY="./moly"
fi

echo -e "${YELLOW}→ Installing Moly backend binary...${NC}"
cp "$BINARY" "$INSTALL_DIR/moly" || {
    echo -e "${RED}Error: Failed to copy binary to $INSTALL_DIR${NC}"
    exit 1
}
chmod +x "$INSTALL_DIR/moly"
echo -e "${GREEN}✓ Backend installed to $INSTALL_DIR/moly${NC}"

# Build and install extension if source available
if [ "$BUILD_FROM_SOURCE" = true ]; then
    echo ""
    echo -e "${YELLOW}→ Building Moly extension...${NC}"
    cd "$PROJECT_ROOT/moly-extension"

    if [ ! -d "node_modules" ]; then
        echo "Installing dependencies..."
        npm install || {
            echo -e "${RED}Error: Failed to install extension dependencies${NC}"
            exit 1
        }
    fi

    npm run build || {
        echo -e "${RED}Error: Failed to build extension${NC}"
        exit 1
    }

    cd "$SCRIPT_DIR"

    echo -e "${YELLOW}→ Installing extension files...${NC}"
    cp -r "$PROJECT_ROOT/moly-extension/dist"/* "$EXTENSION_DIR/" || {
        echo -e "${RED}Error: Failed to copy extension files${NC}"
        exit 1
    }
    echo -e "${GREEN}✓ Extension installed to $EXTENSION_DIR${NC}"
fi

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
        echo -e "${YELLOW}Warning: Failed to create $browser_name native messaging directory${NC}"
        return 1
    }

    local manifest_file="$nmh_dir/com.moly.backend_host.json"
    local temp_manifest="/tmp/moly_manifest_$$.json"

    cat > "$temp_manifest" << EOF
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

    cp "$temp_manifest" "$manifest_file" || {
        echo -e "${YELLOW}Warning: Failed to install native messaging manifest for $browser_name${NC}"
        rm "$temp_manifest"
        return 1
    }
    chmod 644 "$manifest_file"
    rm "$temp_manifest"
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

# Create helper script for loading extension
if [ "$BUILD_FROM_SOURCE" = true ]; then
    echo ""
    echo -e "${YELLOW}→ Creating extension loader script...${NC}"

    LOAD_SCRIPT="$INSTALL_DIR/moly-load-extension"
    TEMP_LOAD_SCRIPT="/tmp/moly_load_extension_$$.sh"
    cat > "$TEMP_LOAD_SCRIPT" << 'LOAD_EOF'
#!/bin/bash
# Moly Extension Loader
# Opens Chrome/Brave with instructions to load Moly extension

EXTENSION_PATH="$HOME/.moly/extension"
[ "$(uname -s)" = "Darwin" ] && EXTENSION_PATH="$HOME/Library/Application Support/Moly/extension"

echo "Moly Extension Loader"
echo "====================="
echo ""
echo "Extension location: $EXTENSION_PATH"
echo ""
echo "To load the extension:"
echo "1. Open Chrome or Brave"
echo "2. Go to: chrome://extensions/ (Chrome) or brave://extensions/ (Brave)"
echo "3. Enable 'Developer mode' (toggle in top-right)"
echo "4. Click 'Load unpacked'"
echo "5. Select the folder: $EXTENSION_PATH"
echo ""
echo "After loading, Moly will auto-start when you click the extension icon!"
LOAD_EOF

    cp "$TEMP_LOAD_SCRIPT" "$LOAD_SCRIPT" || {
        echo -e "${YELLOW}Warning: Failed to create extension loader script${NC}"
        rm "$TEMP_LOAD_SCRIPT"
    }
    chmod +x "$LOAD_SCRIPT"
    rm "$TEMP_LOAD_SCRIPT"
    echo -e "${GREEN}✓ Extension loader created${NC}"
fi

# Final summary
echo ""
echo "============================================"
echo -e "${GREEN}Installation Complete!${NC}"
echo "============================================"
echo ""
echo "Moly is now installed with:"
echo "  ✓ Backend binary: $INSTALL_DIR/moly"
if [ "$BUILD_FROM_SOURCE" = true ]; then
    echo "  ✓ Extension: $EXTENSION_DIR"
fi
echo "  ✓ Configuration: $CONFIG_DIR"
echo ""
echo -e "${BLUE}Quick Start:${NC}"
echo ""

if [ "$BUILD_FROM_SOURCE" = true ]; then
    echo "1. Open Chrome or Brave and navigate to:"
    if [ "$OS" = "Darwin" ]; then
        echo "   Chrome:  chrome://extensions"
        echo "   Brave:   brave://extensions"
    else
        echo "   Chrome:  chrome://extensions"
        echo "   Brave:   brave://extensions"
    fi
    echo ""
    echo "2. Enable 'Developer mode' (toggle in top-right)"
    echo ""
    echo "3. Click 'Load unpacked' and select:"
    echo "   $EXTENSION_DIR"
    echo ""
    echo "4. Click the Moly extension icon - backend starts automatically!"
    echo ""
    echo -e "${YELLOW}Or run:${NC}"
    echo "   moly-load-extension"
else
    echo "1. Start the backend:"
    echo "   moly"
    echo ""
    echo "2. Load extension in browser:"
    echo "   Chrome:  chrome://extensions"
    echo "   Brave:   brave://extensions"
    echo ""
    echo "3. Click 'Load unpacked' and select your extension directory"
fi
echo ""
echo -e "${BLUE}Uninstall:${NC}"
echo "  rm $INSTALL_DIR/moly"
if [ "$BUILD_FROM_SOURCE" = true ]; then
    echo "  rm -rf $EXTENSION_DIR"
fi
echo "  rm -rf $CONFIG_DIR"
echo ""
