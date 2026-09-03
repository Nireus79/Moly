# Moly CORS Proxy

CORS proxy that enables the Moly Chrome extension to communicate with local Ollama instances.

## Problem

Chrome extensions cannot directly make requests to local applications due to CORS (Cross-Origin Resource Sharing) and browser security policies. Ollama's CORS configuration blocks requests with `chrome-extension://` origins.

## Solution

This proxy:
1. Listens on `http://127.0.0.1:11435`
2. Forwards requests to Ollama at `http://127.0.0.1:11434`
3. Strips browser security headers that Ollama rejects
4. Adds CORS headers to allow browser communication

## Installation

### Global (Recommended)

```bash
npm install -g moly-proxy
```

Then start manually:
```bash
moly-proxy
```

Or enable auto-start via the Moly extension settings.

### Local Development

```bash
git clone https://github.com/Nireus79/Moly.git
cd Moly/moly-proxy
npm install
npm start
```

## Auto-Start

The Moly extension can automatically install and configure auto-start:

**Linux (systemd)**
```bash
sudo systemctl enable moly-proxy
sudo systemctl start moly-proxy
```

**macOS (LaunchAgent)**
Configured via: `~/Library/LaunchAgents/com.moly.proxy.plist`

**Windows (Scheduled Task)**
Configured via: Task Scheduler → Moly CORS Proxy

## Logs

**Linux:**
```bash
journalctl -u moly-proxy -f
```

**macOS:**
```bash
tail -f ~/.moly/proxy.log
```

**Windows:**
Event Viewer → Applications and Services Logs

## Environment Variables

- `OLLAMA_HOST` - Ollama server address (default: `127.0.0.1:11434`)
- `PROXY_PORT` - Proxy port (default: `11435`)

## Architecture

```
Chrome Extension (1142x)
    ↓ fetch() with CORS headers
Browser (strips some, adds others)
    ↓
Moly CORS Proxy (127.0.0.1:11435)
    ↓ strips chrome-extension origin header
Local Ollama (127.0.0.1:11434)
```

## What Headers Get Stripped

- `origin` (chrome-extension://)
- `sec-fetch-*` (browser security headers)
- `sec-ch-ua*` (user agent hints)
- `sec-gpc` (privacy preferences)

## Troubleshooting

**"Address already in use" error**
- Another instance is running: `lsof -i :11435` (Linux/macOS) or `netstat -ano | findstr :11435` (Windows)
- Kill existing process and retry

**"Connection refused to Ollama"**
- Ensure Ollama is running: `ollama serve`
- Check Ollama is on port 11434: `curl http://127.0.0.1:11434/api/tags`

**Proxy not found after install**
- Verify installation: `which moly-proxy` (Linux/macOS) or `where moly-proxy` (Windows)
- Check npm global path: `npm config get prefix`

## License

MIT
