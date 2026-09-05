#!/bin/bash
# Moly Backend - Setup Script
# Installs the Go backend for use with the Chrome extension

set -e

echo "=== Moly Backend Setup ==="
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

# Create native messaging host directory
mkdir -p "$NATIVE_HOST_DIR"

# Create native messaging host manifest
NATIVE_HOST_FILE="$NATIVE_HOST_DIR/com.moly.native_host.json"
cat > "$NATIVE_HOST_FILE" << 'EOF'
{
  "name": "com.moly.native_host",
  "description": "Moly Native Host - Launches Moly backend",
  "path": "/home/REPLACE_USERNAME/.local/bin/moly-native-host",
  "type": "stdio"
}
EOF

# Replace username placeholder
sed -i "s|REPLACE_USERNAME|$USER|g" "$NATIVE_HOST_FILE"
echo "✓ Created native messaging host manifest"
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

    # Handle launch-app action
    if request.get('action') == 'launch-app':
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

if [ -f "$NATIVE_HOST_FILE" ]; then
    echo "✓ Native messaging manifest created at $NATIVE_HOST_FILE"
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
echo "1. Reload the Chrome extension (chrome://extensions)"
echo "2. Click the Moly icon"
echo "3. Backend will auto-start automatically"
echo ""
echo "The binary is installed at: $INSTALL_DIR/moly"
echo "The native host is at: $NATIVE_HOST_SCRIPT"
echo ""
echo "To run backend manually:"
echo "  $INSTALL_DIR/moly"
echo ""
