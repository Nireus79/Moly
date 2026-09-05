#!/bin/bash
# DEPRECATED - Use setup.sh instead
# This script is kept for backward compatibility only.

echo "This installer is deprecated."
echo ""
echo "Please use the new setup script instead:"
echo ""
echo "  cd $(dirname "$0")"
echo "  bash setup.sh"
echo ""
exit 0

# Ensure systemd user directory exists
mkdir -p ~/.config/systemd/user

# Create systemd service file
SERVICE_FILE="$HOME/.config/systemd/user/moly.service"
cat > "$SERVICE_FILE" << EOF
[Unit]
Description=Moly Desktop App
After=network.target

[Service]
Type=simple
ExecStart=$BINARY
Restart=always
RestartSec=10

[Install]
WantedBy=default.target
EOF

echo "✓ Created systemd service at $SERVICE_FILE"
echo ""

# Reload systemd
systemctl --user daemon-reload
echo "✓ Reloaded systemd"
echo ""

# Enable the service (auto-start on login)
systemctl --user enable moly.service
echo "✓ Enabled auto-start on login"
echo ""

# Start the service now
systemctl --user start moly.service
sleep 1

# Check if it's running
if systemctl --user is-active --quiet moly.service; then
    echo "✓ Started Moly Desktop App"
    echo ""
    echo "=== Installation Complete ==="
    echo ""
    echo "Moly is now running in the background."
    echo ""
    echo "Useful commands:"
    echo "  systemctl --user status moly      # Check status"
    echo "  systemctl --user stop moly        # Stop the app"
    echo "  systemctl --user restart moly     # Restart the app"
    echo "  journalctl --user -u moly -f      # View live logs"
    echo ""
    echo "The app will automatically start every time you log in."
    echo "Now install the Chrome extension and click the Moly icon!"
else
    echo "✗ Failed to start Moly"
    systemctl --user status moly
    exit 1
fi
