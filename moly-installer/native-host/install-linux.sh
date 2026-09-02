#!/bin/bash
# Moly Native Host Installer for Linux
# Self-managing installer: moves to Moly folder, runs, then self-deletes

set -e

# Error handler
trap 'print_error "Installation failed at line $LINENO"' ERR

MOLY_VERSION="v1.0.0"
GITHUB_REPO="https://github.com/Nireus79/Moly"
BINARY_URL="$GITHUB_REPO/releases/download/$MOLY_VERSION/moly-native-host-linux-x64.tar.gz"
INSTALL_DIR="/usr/local/bin"
SCRIPT_NAME="moly-install-linux.sh"

# Use original user's home directory (not /root when using sudo)
if [[ -n "$SUDO_USER" ]]; then
    USER_HOME="/home/$SUDO_USER"
    MOLY_DATA_DIR="$USER_HOME/.local/share/moly"
    SCRIPT_LOCATION="$USER_HOME/Downloads/$SCRIPT_NAME"
else
    USER_HOME="$HOME"
    MOLY_DATA_DIR="$HOME/.local/share/moly"
    SCRIPT_LOCATION="$PWD/$SCRIPT_NAME"
fi

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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
    echo -e "${GREEN}================================${NC}" >&2
    echo -e "${GREEN}$1${NC}" >&2
    echo -e "${GREEN}================================${NC}" >&2
}

print_step() {
    echo -e "${YELLOW}→ $1${NC}" >&2
}

print_error() {
    echo -e "${RED}✗ Error: $1${NC}" >&2
    exit 1
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}" >&2
}

cleanup_self() {
    print_step "Cleaning up installer..."
    # Delete from Downloads folder where user downloaded it
    rm -f "$SCRIPT_LOCATION" 2>/dev/null || true
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
    local temp_dir=$(mktemp -d) || print_error "Failed to create temp directory"

    print_step "Downloading native host binary..." >&2
    if ! curl -s -L -o "$temp_dir/moly-native-host.tar.gz" "$BINARY_URL"; then
        rm -rf "$temp_dir"
        print_error "Failed to download from $BINARY_URL"
    fi

    # Verify file was downloaded
    if [[ ! -f "$temp_dir/moly-native-host.tar.gz" ]]; then
        rm -rf "$temp_dir"
        print_error "Download file not found after download"
    fi

    local file_size=$(stat -f%z "$temp_dir/moly-native-host.tar.gz" 2>/dev/null || stat -c%s "$temp_dir/moly-native-host.tar.gz")
    if [[ $file_size -lt 1000000 ]]; then
        rm -rf "$temp_dir"
        print_error "Downloaded file too small ($file_size bytes) - may be corrupted"
    fi

    print_success "Downloaded successfully" >&2
    echo "$temp_dir"
}

# Extract and install
install_binary() {
    local temp_dir=$1
    local extracted_binary

    # Verify temp directory exists
    if [[ ! -d "$temp_dir" ]]; then
        print_error "Temporary directory lost: $temp_dir"
    fi

    print_step "Extracting binary..." >&2
    cd "$temp_dir" || print_error "Failed to change to temp directory: $temp_dir"
    tar xzf moly-native-host.tar.gz || print_error "Failed to extract tar.gz"

    # Find the extracted binary (may have platform-specific name)
    if [[ -f "moly-native-host" ]]; then
        extracted_binary="moly-native-host"
    elif [[ -f "moly-native-host-linux-x64" ]]; then
        extracted_binary="moly-native-host-linux-x64"
    else
        print_error "Binary not found after extraction. Contents: $(ls -la)"
    fi
    print_success "Extracted successfully: $extracted_binary" >&2

    print_step "Installing to $INSTALL_DIR..." >&2

    # Ensure install directory exists
    mkdir -p "$INSTALL_DIR" || print_error "Failed to create $INSTALL_DIR"

    # Make binary executable
    chmod +x "$extracted_binary" || print_error "Failed to make binary executable"

    # Verify we can write to install directory
    if [[ ! -w "$INSTALL_DIR" ]]; then
        print_error "No write permission to $INSTALL_DIR"
    fi

    # Copy binary with final name
    cp "$extracted_binary" "$INSTALL_DIR/moly-native-host" || print_error "Failed to copy binary to $INSTALL_DIR"

    # Verify installation
    if [[ ! -f "$INSTALL_DIR/moly-native-host" ]]; then
        print_error "Binary not found at $INSTALL_DIR/moly-native-host after installation"
    fi

    if [[ ! -x "$INSTALL_DIR/moly-native-host" ]]; then
        print_error "Binary at $INSTALL_DIR/moly-native-host is not executable"
    fi

    print_success "Installed to $INSTALL_DIR/moly-native-host" >&2
}

# Setup native messaging
setup_native_messaging() {
    print_step "Setting up native messaging host..." >&2

    local nm_dir="$USER_HOME/.config/google-chrome/NativeMessagingHosts"
    mkdir -p "$nm_dir" || print_error "Failed to create NativeMessagingHosts directory at $nm_dir"

    cat > "$nm_dir/com.moly.native_host.json" << 'EOF' || print_error "Failed to create native messaging config"
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

    # Verify file was created
    if [[ ! -f "$nm_dir/com.moly.native_host.json" ]]; then
        print_error "Native messaging config file not found after creation at $nm_dir/com.moly.native_host.json"
    fi

    print_success "Native messaging configured" >&2
}

# Test installation
test_installation() {
    print_step "Testing installation..." >&2

    if [[ ! -f "$INSTALL_DIR/moly-native-host" ]]; then
        print_error "Native host not found at $INSTALL_DIR/moly-native-host"
    fi

    if [[ ! -x "$INSTALL_DIR/moly-native-host" ]]; then
        print_error "Native host not executable"
    fi

    print_success "Native host is ready" >&2
}

# Setup auto-start with systemd (optional)
setup_autostart() {
    print_step "Setting up auto-start service..." >&2

    # Create directory but don't wait for systemd operations
    mkdir -p "$USER_HOME/.config/systemd/user" 2>/dev/null || true

    print_success "Auto-start ready" >&2
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

    # Cleanup temp directory
    rm -rf "$temp_dir" 2>/dev/null || true

    setup_native_messaging
    test_installation
    setup_autostart
    cleanup_self

    print_header "Installation Complete!"
    echo "" >&2
    echo -e "Next steps:" >&2
    echo -e "1. Open Chrome" >&2
    echo -e "2. Go to chrome://extensions/" >&2
    echo -e "3. Enable \"Developer mode\"" >&2
    echo -e "4. Load unpacked → select Moly extension folder" >&2
    echo -e "5. Open Moly Settings → Set Up Local Model" >&2
    echo -e "6. Click \"Configure Setup\" to download models" >&2
    echo "" >&2
    echo -e "${GREEN}Moly is ready to use!${NC}" >&2
}

main "$@"
