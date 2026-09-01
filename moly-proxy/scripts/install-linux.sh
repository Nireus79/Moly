#!/bin/bash

# Moly CORS Proxy - Linux Installation Script
# Installs moly-proxy and configures systemd auto-start

set -e

echo "╔════════════════════════════════════════╗"
echo "║  Moly CORS Proxy - Linux Installation  ║"
echo "╚════════════════════════════════════════╝"

# Check if running as root for systemd service
if [[ $EUID -ne 0 ]]; then
  echo "⚠️  Note: Run with sudo for auto-start service configuration"
  SKIP_SERVICE=true
fi

# Check Node.js
if ! command -v node &> /dev/null; then
  echo "❌ Node.js not found. Please install Node.js 16+:"
  echo "   https://nodejs.org/"
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

# Configure systemd service if root
if [ -z "$SKIP_SERVICE" ]; then
  echo ""
  echo "Configuring systemd auto-start service..."

  # Create systemd service file
  PROXY_PATH=$(which moly-proxy)

  cat > /tmp/moly-proxy.service <<EOF
[Unit]
Description=Moly CORS Proxy for Ollama
After=network.target

[Service]
Type=simple
User=$SUDO_USER
ExecStart=$PROXY_PATH
Restart=on-failure
RestartSec=10
StandardOutPath=/var/log/moly-proxy.log
StandardErrorPath=/var/log/moly-proxy-error.log

[Install]
WantedBy=multi-user.target
EOF

  cp /tmp/moly-proxy.service /etc/systemd/system/
  systemctl daemon-reload
  systemctl enable moly-proxy

  echo "✓ Systemd service configured"

  # Offer to start service
  read -p "Start moly-proxy service now? (y/n) " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    systemctl start moly-proxy
    echo "✓ Service started"

    sleep 2
    if systemctl is-active --quiet moly-proxy; then
      echo "✓ Service is running"
    else
      echo "⚠️  Service failed to start. Check logs:"
      journalctl -u moly-proxy -n 20
    fi
  fi
else
  echo ""
  echo "To enable auto-start, run with sudo:"
  echo "  sudo bash scripts/install-linux.sh"
fi

echo ""
echo "╔════════════════════════════════════════╗"
echo "║       Installation Complete!           ║"
echo "╚════════════════════════════════════════╝"
echo ""
echo "Quick start:"
echo "  1. Make sure Ollama is running:"
echo "     ollama serve"
echo ""
echo "  2. In another terminal, start the proxy:"
echo "     moly-proxy"
echo ""
echo "  3. Install Moly extension"
echo "  4. Moly will auto-detect the proxy"
echo ""
echo "View logs:"
echo "  journalctl -u moly-proxy -f"
echo ""
echo "Stop service:"
echo "  sudo systemctl stop moly-proxy"
echo ""
