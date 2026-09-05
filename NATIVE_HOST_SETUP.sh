#!/bin/bash

# Moly Native Host Setup
# Allows extension to auto-start Go backend

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
GO_BINARY="$SCRIPT_DIR/moly-go/moly"
HOST_MANIFEST="$SCRIPT_DIR/moly-go/com.moly.backend_host.json"

# Validate Go binary exists
if [ ! -f "$GO_BINARY" ]; then
    echo "Error: Go binary not found at $GO_BINARY"
    echo "Build it first: cd moly-go && go build -o moly ."
    exit 1
fi

# Detect platform
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    # Linux
    MANIFEST_DIR="$HOME/.config/google-chrome/NativeMessagingHosts"
    mkdir -p "$MANIFEST_DIR"
    
    # Create manifest with correct path
    sed "s|MOLY_BACKEND_PATH|$GO_BINARY|g" "$HOST_MANIFEST" > "$MANIFEST_DIR/com.moly.backend_host.json"
    
    echo "✓ Native host manifest installed to $MANIFEST_DIR"
    echo "✓ Extension can now auto-start backend on icon click"
    
elif [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    MANIFEST_DIR="$HOME/Library/Application Support/Google/Chrome/NativeMessagingHosts"
    mkdir -p "$MANIFEST_DIR"
    
    # Create manifest with correct path
    sed "s|MOLY_BACKEND_PATH|$GO_BINARY|g" "$HOST_MANIFEST" > "$MANIFEST_DIR/com.moly.backend_host.json"
    
    echo "✓ Native host manifest installed to $MANIFEST_DIR"
    echo "✓ Extension can now auto-start backend on icon click"
    
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
    # Windows
    echo "Windows setup (run as Administrator):"
    echo "1. Edit registry: HKEY_LOCAL_MACHINE\SOFTWARE\Google\Chrome\NativeMessagingHosts\com.moly.backend_host"
    echo "2. Create string value pointing to: $GO_BINARY"
    echo "Or use PowerShell script to automate this"
else
    echo "Unsupported OS: $OSTYPE"
    exit 1
fi
