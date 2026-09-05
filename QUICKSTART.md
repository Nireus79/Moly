# Moly Extension - Quick Start Guide

## ONE COMMAND TO LAUNCH

```bash
# Linux/macOS
./START_MOLY.sh

# Windows
START_MOLY.bat
```

This single command starts:
- Go Backend (11436) ✓
- CORS Proxy (11435) ✓ (auto-started by Go backend)
- Extension ready to load

That's it. No additional services to start.

## Load Extension in Chrome

```
1. Open chrome://extensions
2. Enable "Developer mode" (top-right toggle)
3. Click "Load unpacked"
4. Select: Moly/moly-extension/dist/
5. Click Moly icon in Chrome toolbar
```

## Test the Extension

1. **Icon Click**: Should show BackendStatus indicator (✓ connected)
2. **Type Message**: "Hey, how are you?"
3. **See Results**: 
   - SafetyAlert appears if crisis detected
   - Ethics check shows violations
   - Suggestions appear below
   - Copy button for each suggestion
4. **Optional**: Test without local backend by stopping services (Ctrl+C) - extension falls back to Claude/OpenAI if configured

## Services Auto-Started

| Service | Port | What It Does | Auto-Start |
|---------|------|-------------|-----------|
| Go Backend | 11436 | Safety/ethics analysis | ✓ When you run START_MOLY.sh |
| CORS Proxy | 11435 | Allows browser to talk to Ollama | ✓ Started by Go Backend |
| Ollama | 11434 | Local LLM models | Manual (optional: `ollama serve`) |

## How It Works

```
You run START_MOLY.sh
  ↓
Go Backend starts (11436)
  ↓
Go Backend auto-spawns CORS Proxy (11435)
  ↓
You load extension in Chrome
  ↓
You click Moly icon
  ↓
Extension checks if backend running (11436) → ✓ Connected
  ↓
You type message
  ↓
Extension sends to Go Backend:
  • Safety check (crisis detection)
  • Ethics evaluation
  • Question generation
  ↓
Extension requests LLM via CORS Proxy (11435)
  ↓
CORS Proxy forwards to Ollama (11434)
  or Claude/OpenAI if configured
  ↓
You see suggestions with Copy buttons
```

## Graceful Degradation

Even if something is down, Moly keeps working:

- **No Ollama**: Uses Claude/OpenAI if API key configured
- **No CORS Proxy**: Uses fallback direct Ollama (may have CORS issues)
- **No Go Backend**: No safety/ethics analysis, but LLM suggestions still work
- **No internet/LLM**: Shows error message but extension doesn't crash

The extension is designed to never break, only to degrade gracefully.

## Troubleshooting

### "Backend unavailable" message in sidebar
- Make sure START_MOLY.sh is still running
- Check: `curl http://localhost:11436/api/status`
- Restart: Ctrl+C then run START_MOLY.sh again

### No suggestions appearing
- Check if Ollama is running: `ollama serve` (separate terminal)
- Or configure Claude/OpenAI API key in Settings (⚙️)
- Check F12 → Extensions → Inspect → Console for errors

### CORS Proxy not starting
- Check Node.js installed: `node --version`
- Check moly-proxy/bin/moly-proxy.js exists
- Restart Go backend: kill process and run START_MOLY.sh

### Port already in use
```bash
# Find what's using port 11436
lsof -i :11436

# Kill it
pkill -f moly-go

# Restart
./START_MOLY.sh
```

## Settings

Click ⚙️ in extension to configure:
- **LLM Provider**: Ollama (default) / Claude / OpenAI
- **Ollama Model**: mistral, llama2, neural-chat, etc.
- **Claude API Key**: From console.anthropic.com
- **OpenAI API Key**: From platform.openai.com

## One-Minute Setup

```bash
# Step 1: Run startup
./START_MOLY.sh

# Step 2: Open Chrome and go to extensions
chrome://extensions

# Step 3: Load unpacked extension
# Select: Moly/moly-extension/dist/

# Step 4: Click Moly icon and start typing

# Done! Extension is ready to use
```

---

**Status**: Fully automated, ready to test  
**Time to working**: ~2 minutes
