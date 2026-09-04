#!/bin/bash
set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
HOME_DIR="$HOME"
CONFIG_DIR="$HOME_DIR/.config/moly"
INSTALL_DIR="$HOME_DIR/.local/bin"
NATIVE_HOST="$INSTALL_DIR/moly-native-host"
PYTHON_HOST="$SCRIPT_DIR/native-host/moly-host.py"

echo "Moly Uninstaller"
echo "================"
echo ""
echo "This will remove:"
echo "  - Desktop application"
echo "  - Browser extension configuration"
echo "  - System integration files"
echo ""
echo "Local models (if installed) are stored separately in ~/.ollama"
echo ""

read -p "Keep local models? (y/n) " -n 1 -r
echo
KEEP_MODELS="true"
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    KEEP_MODELS="false"
fi

echo "Uninstalling Moly..."
echo ""

python3 << PYTHON_SCRIPT
import json
import sys

try:
    with open('$NATIVE_HOST', 'r') as f:
        # Check if native host exists and is executable
        pass
except:
    pass

PYTHON_SCRIPT

if [ -f "$NATIVE_HOST" ]; then
    echo "Calling cleanup from native host..."
    python3 "$PYTHON_HOST" << EOF
{
  "action": "cleanup",
  "keep_models": $KEEP_MODELS
}
EOF
fi

echo "Removing installation files..."

rm -f "$NATIVE_HOST" 2>/dev/null || true
rm -f "$NATIVE_HOST.exe" 2>/dev/null || true

if [ "$KEEP_MODELS" = "false" ]; then
    echo "Removing local models..."
    rm -rf "$HOME_DIR/.ollama/models" 2>/dev/null || true
fi

rm -rf "$CONFIG_DIR" 2>/dev/null || true

if [ -d "$HOME_DIR/.config/chrome/NativeMessagingHosts" ]; then
    rm -f "$HOME_DIR/.config/chrome/NativeMessagingHosts/com.moly.native_host.json" 2>/dev/null || true
fi

rm -f "$HOME_DIR/.config/autostart/moly.desktop" 2>/dev/null || true
rm -f "$HOME_DIR/Library/LaunchAgents/com.moly.app.plist" 2>/dev/null || true

echo ""
echo "Uninstall complete!"
if [ "$KEEP_MODELS" = "true" ]; then
    echo "Local models have been kept in ~/.ollama"
else
    echo "Local models have been removed"
fi
echo ""
echo "To remove the browser extension:"
echo "  1. Open Chrome/Edge"
echo "  2. Go to chrome://extensions"
echo "  3. Find 'Moly' and click Remove"
