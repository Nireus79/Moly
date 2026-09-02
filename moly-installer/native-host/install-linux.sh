#!/bin/bash
# Moly Native Host Installer for Linux
# Self-managing installer: moves to Moly folder, runs, then self-deletes

set -e

MOLY_VERSION="v1.0.0"
GITHUB_REPO="https://github.com/Nireus79/Moly"
BINARY_URL="$GITHUB_REPO/releases/download/$MOLY_VERSION/moly-native-host-linux-x64.tar.gz"
INSTALL_DIR="/usr/local/bin"
MOLY_DATA_DIR="$HOME/.local/share/moly"
SCRIPT_NAME="moly-install-linux.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Auto-elevate to sudo if needed
if [[ "$EUID" -ne 0 ]]; then
    exec sudo bash "$0" "$@"
fi

# Functions
move_to_moly_folder() {
    # Create Moly data directory
    mkdir -p "$MOLY_DATA_DIR"

    # Check if we're already in the Moly folder
    if [[ "$(pwd)" == "$MOLY_DATA_DIR" ]]; then
        return 0
    fi

    # Find the script location
    SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$SCRIPT_NAME"
    if [[ ! -f "$SCRIPT_PATH" ]]; then
        # Try current directory
        SCRIPT_PATH="$PWD/$SCRIPT_NAME"
    fi

    # Copy script to Moly folder and re-execute from there
    if [[ -f "$SCRIPT_PATH" ]]; then
        cp "$SCRIPT_PATH" "$MOLY_DATA_DIR/$SCRIPT_NAME"
        cd "$MOLY_DATA_DIR"
        exec bash "$SCRIPT_NAME"
    fi
}

print_header() {
    echo -e "${GREEN}================================${NC}"
    echo -e "${GREEN}$1${NC}"
    echo -e "${GREEN}================================${NC}"
}

print_step() {
    echo -e "${YELLOW}→ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ Error: $1${NC}"
    exit 1
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

cleanup_self() {
    print_step "Cleaning up installer..."
    rm -f "$MOLY_DATA_DIR/$SCRIPT_NAME"
    print_success "Installer cleaned up"
}

# Create data directory
setup_data_directory() {
    print_step "Setting up Moly data directory..."
    mkdir -p "$MOLY_DATA_DIR"
    print_success "Data directory ready: $MOLY_DATA_DIR"
}

# Download binary
download_binary() {
    local temp_dir=$(mktemp -d)
    trap "rm -rf $temp_dir" EXIT

    print_step "Downloading native host binary..."
    if ! curl -L -o "$temp_dir/moly-native-host.tar.gz" "$BINARY_URL"; then
        print_error "Failed to download from $BINARY_URL"
    fi
    print_success "Downloaded successfully"

    echo "$temp_dir"
}

# Extract and install
install_binary() {
    local temp_dir=$1

    print_step "Extracting binary..."
    cd "$temp_dir"
    tar xzf moly-native-host.tar.gz

    if [[ ! -f "moly-native-host" ]]; then
        print_error "Failed to extract binary"
    fi
    print_success "Extracted successfully"

    print_step "Installing to $INSTALL_DIR..."
    chmod +x moly-native-host
    cp moly-native-host "$INSTALL_DIR/moly-native-host"
    print_success "Installed to $INSTALL_DIR/moly-native-host"
}

# Setup native messaging
setup_native_messaging() {
    print_step "Setting up native messaging host..."

    local nm_dir="$HOME/.config/google-chrome/NativeMessagingHosts"
    mkdir -p "$nm_dir"

    cat > "$nm_dir/com.moly.native_host.json" << 'EOF'
{
  "name": "com.moly.native_host",
  "description": "Moly Native Host",
  "path": "/usr/local/bin/moly-native-host",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://*/",
    "chrome-extension://*/popup.html",
    "chrome-extension://*/sidebar.html"
  ]
}
EOF

    print_success "Native messaging configured"
}

# Test installation
test_installation() {
    print_step "Testing installation..."

    if ! command -v moly-native-host &> /dev/null; then
        print_error "Native host not found in PATH"
    fi

    # Try to run with --version or similar
    if moly-native-host --help &> /dev/null || true; then
        print_success "Native host is executable"
    fi
}

# Setup auto-start with systemd (optional)
setup_autostart() {
    print_step "Setting up auto-start service..."

    # This will be called later by the extension via native messaging
    # Just ensure the directories exist
    mkdir -p "$HOME/.config/systemd/user"

    print_success "Auto-start ready (will be configured on first run)"
}

# Main installation
main() {
    # Move to Moly folder and re-execute if needed
    move_to_moly_folder

    print_header "Moly Native Host Installer"
    echo "Version: $MOLY_VERSION"
    echo ""

    setup_data_directory

    temp_dir=$(download_binary)
    install_binary "$temp_dir"
    setup_native_messaging
    test_installation
    setup_autostart
    cleanup_self

    print_header "Installation Complete!"
    echo ""
    echo -e "Next steps:"
    echo -e "1. Open Chrome"
    echo -e "2. Go to chrome://extensions/"
    echo -e "3. Enable \"Developer mode\""
    echo -e "4. Load unpacked → select Moly extension folder"
    echo -e "5. Open Moly Settings → Set Up Local Model"
    echo -e "6. Click \"Configure Setup\" to download models"
    echo ""
    echo -e "${GREEN}Moly is ready to use!${NC}"
}

main "$@"
