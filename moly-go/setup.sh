#!/bin/bash
# Moly Backend - Setup Script
# Installs the Go backend for use with the Chrome/Brave extension

set -e

echo "=== Moly Backend Setup ==="
echo ""

# Try to auto-detect extension ID from browser profile
detect_extension_id() {
  local browser="$1"
  local profile_path="$2"

  if [ ! -d "$profile_path" ]; then
    return 1
  fi

  # Search for Moly extension in all installed extensions
  local moly_dir=$(find "$profile_path" -type d -name "*" 2>/dev/null | while read dir; do
    if [ -f "$dir/manifest.json" ] && grep -q '"name": "Moly' "$dir/manifest.json" 2>/dev/null; then
      echo "$dir"
      break
    fi
  done)

  if [ -n "$moly_dir" ]; then
    # Extract extension ID from directory path
    # Path format: ~/.config/google-chrome/Default/Extensions/EXTENSION_ID/VERSION/
    basename "$(dirname "$moly_dir")"
    return 0
  fi

  return 1
}

# Try to find extension ID
EXTENSION_ID=""

# Try Brave first
if EXTENSION_ID=$(detect_extension_id "Brave" "$HOME/.config/BraveSoftware/Brave-Browser/Default/Extensions" 2>/dev/null); then
  echo "✓ Found Moly extension in Brave: $EXTENSION_ID"
elif EXTENSION_ID=$(detect_extension_id "Chrome" "$HOME/.config/google-chrome/Default/Extensions" 2>/dev/null); then
  echo "✓ Found Moly extension in Chrome: $EXTENSION_ID"
fi

# If not found, ask user
if [ -z "$EXTENSION_ID" ]; then
  echo "Could not auto-detect Moly extension."
  echo ""
  echo "To find your extension ID:"
  echo "  1. Open brave://extensions (or chrome://extensions)"
  echo "  2. Look for 'Moly - Messaging Coach'"
  echo "  3. Copy the ID (e.g., 'abcdefghijklmnopqrstuvwxyz')"
  echo ""

  if [ -n "$1" ]; then
    EXTENSION_ID="$1"
    echo "Using provided ID: $EXTENSION_ID"
  else
    read -p "Enter your extension ID: " EXTENSION_ID
  fi
fi

if [ -z "$EXTENSION_ID" ]; then
  echo "Error: No extension ID provided"
  exit 1
fi

echo ""

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BINARY="$SCRIPT_DIR/moly"
INSTALL_DIR="$HOME/.local/bin"
NATIVE_HOST_DIR="$HOME/.config/google-chrome/NativeMessagingHosts"

# Check if binary exists
if [ ! -f "$BINARY" ]; then
    echo "Error: moly binary not found at $BINARY"
    echo "Please run: cd $SCRIPT_DIR && go build -o moly ."
    exit 1
fi

# Check if binary is executable
if [ ! -x "$BINARY" ]; then
    echo "Error: moly is not executable"
    chmod +x "$BINARY"
    echo "✓ Made binary executable"
fi

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

# Copy binary to standard location
echo "Installing binary to $INSTALL_DIR/moly..."
cp "$BINARY" "$INSTALL_DIR/moly"
chmod +x "$INSTALL_DIR/moly"
echo "✓ Binary installed"
echo ""

# Create native messaging host directories (support both Chrome and Brave, both system-wide and profile-specific)
CHROME_SYSTEM_DIR="$HOME/.config/google-chrome/NativeMessagingHosts"
CHROME_PROFILE_DIR="$HOME/.config/google-chrome/Default/NativeMessagingHosts"
BRAVE_SYSTEM_DIR="$HOME/.config/BraveSoftware/Brave-Browser/NativeMessagingHosts"
BRAVE_PROFILE_DIR="$HOME/.config/BraveSoftware/Brave-Browser/Default/NativeMessagingHosts"

mkdir -p "$CHROME_SYSTEM_DIR" "$CHROME_PROFILE_DIR" "$BRAVE_SYSTEM_DIR" "$BRAVE_PROFILE_DIR"

# Create native messaging host manifest (MUST match what extension calls: com.moly.backend_host)
MANIFEST_TEMPLATE=$(cat << EOF
{
  "name": "com.moly.backend_host",
  "description": "Moly Native Host - Launches Moly backend",
  "path": "/home/REPLACE_USERNAME/.local/bin/moly-native-host",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://$EXTENSION_ID/"
  ]
}
EOF
)

# Create manifest in all locations
for DIR in "$CHROME_SYSTEM_DIR" "$CHROME_PROFILE_DIR" "$BRAVE_SYSTEM_DIR" "$BRAVE_PROFILE_DIR"; do
  echo "$MANIFEST_TEMPLATE" > "$DIR/com.moly.backend_host.json"
  sed -i "s|REPLACE_USERNAME|$USER|g" "$DIR/com.moly.backend_host.json"
done

echo "✓ Created native messaging host manifests in all locations"
echo "  - Chrome system-wide: $CHROME_SYSTEM_DIR/com.moly.backend_host.json"
echo "  - Chrome Default profile: $CHROME_PROFILE_DIR/com.moly.backend_host.json"
echo "  - Brave system-wide: $BRAVE_SYSTEM_DIR/com.moly.backend_host.json"
echo "  - Brave Default profile: $BRAVE_PROFILE_DIR/com.moly.backend_host.json"
echo ""

# Create native messaging host launcher script
NATIVE_HOST_SCRIPT="$INSTALL_DIR/moly-native-host"
cat > "$NATIVE_HOST_SCRIPT" << 'EOF'
#!/usr/bin/env python3
import sys
import json
import struct
import subprocess
import socket
import time
from pathlib import Path

APP_BINARY = str(Path.home() / ".local" / "bin" / "moly")

def can_connect(host, port, timeout=1):
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(timeout)
        result = sock.connect_ex((host, port))
        sock.close()
        return result == 0
    except:
        return False

def send_response(success, message='', error=''):
    response = {"success": success}
    if message:
        response["message"] = message
    if error:
        response["error"] = error
    resp_json = json.dumps(response)
    resp_bytes = resp_json.encode('utf-8')
    sys.stdout.buffer.write(struct.pack('<I', len(resp_bytes)))
    sys.stdout.buffer.write(resp_bytes)
    sys.stdout.buffer.flush()

try:
    # Read 4-byte length
    length_bytes = sys.stdin.buffer.read(4)
    if len(length_bytes) != 4:
        sys.exit(1)

    length = struct.unpack('<I', length_bytes)[0]
    data = sys.stdin.buffer.read(length)
    request = json.loads(data.decode('utf-8'))

    # Handle start-backend action (sent by extension)
    if request.get('action') in ('start-backend', 'launch-app'):
        try:
            # Check if app is already running
            if can_connect('127.0.0.1', 11436):
                send_response(True, "Moly app already running")
                sys.exit(0)

            # Launch Go binary in background
            subprocess.Popen(
                [APP_BINARY],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                start_new_session=True
            )

            # Wait for port 11436 (timeout after 20 seconds)
            for i in range(40):
                if can_connect('127.0.0.1', 11436):
                    send_response(True, "Moly app launched")
                    sys.exit(0)
                time.sleep(0.5)

            # Timeout
            send_response(False, error="App launched but port never became available")
        except Exception as e:
            send_response(False, error=str(e))
    else:
        send_response(False, error="Unknown action")

except Exception as e:
    try:
        send_response(False, error=f"Native host error: {str(e)}")
    except:
        pass
    sys.exit(1)
EOF

chmod +x "$NATIVE_HOST_SCRIPT"
echo "✓ Created native messaging host launcher"
echo ""

# Verify setup
echo "=== Verification ==="
echo ""

if [ -f "$INSTALL_DIR/moly" ] && [ -x "$INSTALL_DIR/moly" ]; then
    echo "✓ Moly binary installed at $INSTALL_DIR/moly"
else
    echo "✗ Binary installation failed"
    exit 1
fi

if [ -f "$NATIVE_HOST_SCRIPT" ] && [ -x "$NATIVE_HOST_SCRIPT" ]; then
    echo "✓ Native host launcher installed at $NATIVE_HOST_SCRIPT"
else
    echo "✗ Native host installation failed"
    exit 1
fi

if [ -f "$BRAVE_PROFILE_DIR/com.moly.backend_host.json" ]; then
    echo "✓ Native messaging manifests created in all locations"
    echo "  - Extension: $EXTENSION_ID"
else
    echo "✗ Native messaging manifest creation failed"
    exit 1
fi

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Moly backend is ready!"
echo ""
echo "Next steps:"
echo "1. Click the Moly extension icon"
echo "2. Backend will auto-start automatically"
echo ""
echo ""
echo "The binary is installed at: $INSTALL_DIR/moly"
echo "The native host is at: $NATIVE_HOST_SCRIPT"
echo ""
echo "To run backend manually:"
echo "  $INSTALL_DIR/moly"
echo ""
