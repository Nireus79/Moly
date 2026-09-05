# Ollama Setup for Moly Extension

## Quick Start

```bash
# 1. Start Ollama
ollama serve

# 2. In another terminal, pull a model
ollama pull mistral:latest

# 3. Start Go backend (Moly)
cd moly-go
./moly

# 4. Load extension in Chrome
# chrome://extensions → Load unpacked → dist/
```

## Common Issues

### 403 Forbidden from Browser

**Cause**: Browser CORS policy blocks direct requests to Ollama on 11434

**Solution A: Use reverse proxy (recommended)**

```bash
# Start CORS proxy that forwards to Ollama
cd moly-go
# Add this to your systemd service or run manually:
# A simple Node proxy or Caddy reverse proxy works
```

**Solution B: Configure Ollama CORS headers**

Ollama doesn't natively support CORS from browser requests. The extension expects:
- Port 11434 (direct Ollama) OR
- Port 11435 (CORS proxy that forwards to Ollama)

**Solution C: Use cloud LLM instead**

If Ollama setup is complex:
1. Settings → Configure Claude or OpenAI API key
2. Extension will use cloud LLM instead of local Ollama
3. All features work the same

### 11435 Port Connection Refused

**Cause**: CORS proxy not running

**Check if it's running:**
```bash
curl http://localhost:11435/api/tags
# Should return JSON with models, not connection refused
```

**Start the proxy:**
```bash
# Option 1: Simple Node proxy
npx http-proxy-middleware-server \
  --source http://localhost:11434 \
  --target-host localhost \
  --target-port 11435 \
  --allow-origin "*"

# Option 2: Caddy reverse proxy
# Create Caddyfile:
# localhost:11435 {
#   reverse_proxy localhost:11434
# }
caddy run
```

### Ollama Endpoint Not Found

**Cause**: Ollama not running or models not installed

**Check:**
```bash
# Is Ollama running?
curl http://localhost:11434/api/tags

# If error, start Ollama:
ollama serve

# In another terminal, list models:
ollama list

# If no models, install one:
ollama pull mistral:latest
```

### Extension Still Won't Use Ollama

**Debug:**
1. Open extension console: `chrome://extensions` → Moly → Inspect views
2. Look for `[Moly]` logs
3. Check what error message appears
4. Common: "Ollama API error: 403 Forbidden"

**If 403 persists:**
- Ollama is running but rejecting browser requests
- Switch to Claude/OpenAI API in Settings
- Or set up proper CORS proxy

## Architecture

```
Extension (11435 or 11434)
    ↓
CORS Proxy (11435) [optional]
    ↓
Ollama API (11434)
    ↓
Mistral/Llama2/Neural Chat models
```

**With proxy (recommended):**
- Browser can make requests to 11435
- Proxy forwards to Ollama on 11434
- CORS headers properly set

**Direct (if Ollama allows):**
- Browser makes requests directly to 11434
- Requires Ollama to allow cross-origin requests
- May fail with 403

## Setup Options

### Option 1: Simple Proxy (Recommended)

```bash
# Using Caddy (simplest)
brew install caddy  # macOS
# Or download from https://caddyserver.com/download

# Create Caddyfile:
cat > Caddyfile << 'EOF'
localhost:11435 {
    reverse_proxy localhost:11434 {
        header_up X-Forwarded-Host {host}
    }
}
EOF

# Run:
caddy run
```

### Option 2: Use Node Proxy

```bash
# Create proxy.js:
const http = require('http');
const httpProxy = require('http-proxy');

const proxy = httpProxy.createProxyServer({
  target: 'http://localhost:11434',
  changeOrigin: true,
  ws: true,
});

http.createServer((req, res) => {
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
  
  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
  } else {
    proxy.web(req, res);
  }
}).listen(11435);

console.log('CORS proxy listening on 11435, forwarding to 11434');

// Run:
node proxy.js
```

### Option 3: Use Extension with Cloud LLM

Simplest for testing:
1. Get API key: Claude (console.anthropic.com) or OpenAI (platform.openai.com)
2. Settings → Add API key
3. Extension uses cloud LLM, all features work
4. No CORS proxy needed

## Testing Ollama Setup

```bash
# 1. Check Ollama is running on 11434
curl -X GET http://localhost:11434/api/tags
# Should return: {"models": [...]}

# 2. Check CORS proxy is running on 11435
curl -X GET http://localhost:11435/api/tags
# Should also return: {"models": [...]}

# 3. Test from browser console
fetch('http://localhost:11435/api/tags')
  .then(r => r.json())
  .then(d => console.log(d))
  .catch(e => console.error('CORS error:', e))
```

## Extension Fallback Behavior

1. **Try Ollama on 11435** (with CORS proxy)
   - Recommended setup
   - Most reliable

2. **Fallback to Ollama on 11434** (direct)
   - Works if Ollama allows CORS
   - May fail with 403

3. **Fallback to Claude/OpenAI**
   - If Ollama unavailable
   - Uses API keys from Settings
   - Always works if keys configured

## Production Setup

For always-working extension:

1. **Linux**: Systemd service for Ollama + proxy
2. **macOS**: LaunchAgent for Ollama + Caddy
3. **Windows**: Task Scheduler + proxy EXE

Or simplest: Use Claude/OpenAI API (no local setup needed)

## Troubleshooting Checklist

- [ ] Ollama running: `ps aux | grep ollama`
- [ ] Ollama responsive: `curl http://localhost:11434/api/tags`
- [ ] Model installed: `ollama list` (shows at least one)
- [ ] Proxy running: `curl http://localhost:11435/api/tags`
- [ ] Extension built: `cd moly-extension && npm run build`
- [ ] Extension loaded: `chrome://extensions` shows Moly
- [ ] Go backend running: `cd moly-go && ./moly` (optional, for safety/ethics)
- [ ] Console logs: Check chrome extension inspector for errors

## Support

If still getting 403:
1. Switch to Claude/OpenAI in Settings (easiest)
2. Or set up Caddy proxy as shown above
3. Or ask for help with Ollama CORS configuration
