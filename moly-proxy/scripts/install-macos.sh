#!/bin/bash

# Moly CORS Proxy - macOS Installation Script
# Installs moly-proxy and configures LaunchAgent auto-start

echo "╔═════════════════════════════════════════╗"
echo "║  Moly CORS Proxy - macOS Installation   ║"
echo "╚═════════════════════════════════════════╝"

# Check Node.js
if ! command -v node &> /dev/null; then
  echo "❌ Node.js not found. Please install Node.js 16+:"
  echo "   https://nodejs.org/"
  echo ""
  echo "   Or via Homebrew:"
  echo "   brew install node"
  exit 1
fi

NODE_VERSION=$(node -v)
echo "✓ Node.js found: $NODE_VERSION"

# Check npm
if ! command -v npm &> /dev/null; then
  echo "❌ npm not found. Please install npm."
  exit 1
fi

NPM_VERSION=$(npm -v)
echo "✓ npm found: $NPM_VERSION"

# Install globally
echo ""
echo "Installing moly-proxy globally..."
npm install -g moly-proxy

echo "✓ moly-proxy installed successfully"

# Configure LaunchAgent
echo ""
echo "Configuring LaunchAgent auto-start..."

PROXY_PATH=$(which moly-proxy)
LAUNCH_AGENTS="$HOME/Library/LaunchAgents"
PLIST_FILE="$LAUNCH_AGENTS/com.moly.proxy.plist"

mkdir -p "$LAUNCH_AGENTS"

cat > "$PLIST_FILE" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.moly.proxy</string>
    <key>ProgramArguments</key>
    <array>
        <string>$PROXY_PATH</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>/var/log/moly-proxy.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/moly-proxy-error.log</string>
</dict>
</plist>
EOF

echo "✓ LaunchAgent plist created at: $PLIST_FILE"

# Load the service
launchctl load "$PLIST_FILE"
echo "✓ Service loaded"

# Check if it's running
sleep 2
if launchctl list | grep -q com.moly.proxy; then
  echo "✓ Service is running"
else
  echo "⚠️  Service may not be running. Check logs:"
  echo "  tail -f /var/log/moly-proxy.log"
fi

echo ""
echo "╔═════════════════════════════════════════╗"
echo "║       Installation Complete!            ║"
echo "╚═════════════════════════════════════════╝"
echo ""
echo "Quick start:"
echo "  1. Make sure Ollama is running:"
echo "     ollama serve"
echo ""
echo "  2. The proxy starts automatically at login"
echo "  3. Install Moly extension"
echo "  4. Moly will auto-detect the proxy"
echo ""
echo "Check status:"
echo "  launchctl list | grep com.moly.proxy"
echo ""
echo "View logs:"
echo "  tail -f /var/log/moly-proxy.log"
echo ""
echo "Unload service:"
echo "  launchctl unload ~/Library/LaunchAgents/com.moly.proxy.plist"
echo ""
