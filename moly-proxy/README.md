# Moly CORS Proxy

A lightweight CORS proxy for local Ollama instances. Solves browser CORS restrictions so Moly extension can communicate with Ollama running on your machine.

## Why This Proxy?

Ollama doesn't send CORS headers by default, blocking browser requests. This proxy adds those headers, allowing Moly to work seamlessly.

## Installation

### From NPM (Recommended)
```bash
npm install -g moly-proxy
```

### Local Development
```bash
cd moly-proxy
npm install
npm start
```

## Usage

### Start the Proxy
```bash
moly-proxy
```

This starts the proxy at `http://127.0.0.1:11435` and proxies to `http://localhost:11434`.

### Custom Configuration
```bash
moly-proxy --ollama-url http://localhost:11434 --port 11435
```

### Environment Variables
```bash
OLLAMA_URL=http://localhost:11434
MOLY_PROXY_PORT=11435
moly-proxy
```

## Auto-Start Services

The proxy can be configured to auto-start on system boot.

### Linux (systemd)
```bash
sudo tee /etc/systemd/system/moly-proxy.service > /dev/null <<EOF
[Unit]
Description=Moly CORS Proxy for Ollama
After=network.target

[Service]
Type=simple
User=$USER
ExecStart=$(which moly-proxy)
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable moly-proxy
sudo systemctl start moly-proxy
```

### macOS (LaunchAgent)
```bash
mkdir -p ~/Library/LaunchAgents
cat > ~/Library/LaunchAgents/com.moly.proxy.plist <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.moly.proxy</string>
    <key>ProgramArguments</key>
    <array>
        <string>$(which moly-proxy)</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/moly-proxy.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/moly-proxy-error.log</string>
</dict>
</plist>
EOF

launchctl load ~/Library/LaunchAgents/com.moly.proxy.plist
```

### Windows (Task Scheduler)
See `scripts/install-windows.ps1` for PowerShell installation script.

## Testing

```bash
npm test
```

This starts the proxy and tests CORS header configuration.

## Troubleshooting

### Port Already in Use
```bash
# Find what's using port 11435
lsof -i :11435

# Use a different port
moly-proxy --port 11436
```

### Ollama Not Responding
Make sure Ollama is running:
```bash
ollama serve
```

### CORS Still Not Working
- Verify proxy is running: `curl http://127.0.0.1:11435/api/tags`
- Check Ollama is at localhost:11434: `curl http://localhost:11434/api/tags`
- Look for errors in proxy logs

## Architecture

```
Browser (Moly Extension)
    ↓
http://127.0.0.1:11435 (Proxy with CORS headers)
    ↓
http://localhost:11434 (Ollama Server)
```

## Features

- Lightweight Express.js proxy
- Automatic CORS header injection
- Support for all Ollama endpoints
- Streams large responses
- Error handling and diagnostics
- Cross-platform (Linux, macOS, Windows)

## Security Notes

- Proxy only listens on `127.0.0.1` (localhost) - not exposed to network
- All CORS origin checks are permissive (local use only)
- No authentication required (assumes local network trust)
- Suitable for personal/development use

## Environment

- Node.js 16+
- Express.js 4.18+
- Works on Linux, macOS, Windows

## License

MIT

## Support

Issues: https://github.com/Nireus79/Moly/issues
