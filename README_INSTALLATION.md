# Moly - Installation & Usage Guide

## What is Moly?

Moly is a Chrome extension that helps you craft better messages through intelligent dialogue and safety analysis.

**Features:**
- Chat-based context building with Moly
- Safety checks (crisis/illegal language detection)
- Ethics evaluation against 10 principles
- AI-generated message suggestions
- Works offline with local LLM or online with Claude/OpenAI

## Requirements

- **Browser**: Chrome, Chromium, or Edge (Chromium-based)
- **Operating System**: Linux, macOS, or Windows
- **Go**: For building the backend
- **Node.js**: For CORS proxy
- **Optional**: Ollama or Claude/OpenAI API key

## One-Line Installation

```bash
./INSTALL_AND_LOAD.sh
```

This script will:
1. Check for Chrome/Chromium
2. Build Go backend if needed
3. Set up native host integration
4. Open Chrome extensions page
5. Guide you to load the extension

That's it. No complicated setup.

## Step-by-Step (Manual)

### Step 1: Set Up Native Host (One-Time)

```bash
./NATIVE_HOST_SETUP.sh
```

This creates a system configuration that allows the extension to auto-start the backend.

### Step 2: Load Extension in Chrome

```
1. Open: chrome://extensions
2. Enable "Developer mode" (top-right toggle)
3. Click "Load unpacked"
4. Select: Moly/moly-extension/dist/
5. Done!
```

### Step 3: Use It

```
1. Click Moly icon in toolbar
2. Backend auto-starts automatically
3. Type a message
4. Get safety analysis + suggestions
```

## How It Works

```
Click Icon
  ↓
Extension checks if backend running (11436)
  ↓
Backend not running? Auto-start via native host
  ↓
Go backend starts + spawns CORS Proxy (11435)
  ↓
BackendStatus shows: ✓ Connected
  ↓
Type message → Safety check + Ethics eval + LLM suggestions
  ↓
Copy suggestion to clipboard
```

## Services (Auto-Started)

| Service | Port | Purpose |
|---------|------|---------|
| Go Backend | 11436 | Safety/ethics analysis |
| CORS Proxy | 11435 | Browser → Ollama forwarding |
| Ollama | 11434 | Local LLM (optional) |

**Note**: The extension auto-starts the Go backend and CORS Proxy. Ollama is optional.

## Settings

Click ⚙️ in the extension to configure:

**LLM Provider**:
- Ollama (default) - local, free
- Claude - requires API key
- OpenAI - requires API key

**Ollama Settings**:
- Model: mistral, llama2, neural-chat, etc.
- Requires: `ollama serve` running separately

**Claude**:
- API Key from: console.anthropic.com
- No setup needed otherwise

**OpenAI**:
- API Key from: platform.openai.com
- No setup needed otherwise

## Troubleshooting

### "Backend unavailable" message

The backend didn't auto-start. Try:

1. Click Moly icon again (triggers auto-start)
2. Wait 2 seconds
3. Should show ✓ Connected

If still failing:
```bash
# Check if backend is running
curl http://localhost:11436/api/status

# If not, manually start
./START_MOLY.sh
```

### Extension doesn't load in Chrome

1. Built extension? `ls moly-extension/dist/manifest.json`
2. Try rebuild: `cd moly-extension && npm run build`
3. Refresh extensions page (F5)

### No suggestions appearing

Check:
1. LLM configured in Settings ⚙️
2. If Ollama: running `ollama serve`?
3. If Claude/OpenAI: API key valid?
4. Console for errors: F12 → Extensions → Inspect

### CORS Proxy errors

The CORS Proxy is auto-started by the Go backend. If you see CORS errors:

1. Check proxy running: `curl http://localhost:11435/api/tags`
2. If not, restart backend: Kill and run `./START_MOLY.sh`

### Native host not working

If auto-start doesn't work:

1. Re-run setup: `./NATIVE_HOST_SETUP.sh`
2. Restart Chrome
3. Try again

If still failing, use manual startup:
```bash
./START_MOLY.sh
# Then click extension - backend already running
```

## Advanced: Manual Service Startup

If native host doesn't work, you can manually start services:

```bash
# Terminal 1: Start Go backend + CORS Proxy
./START_MOLY.sh

# Terminal 2 (optional): Start Ollama
ollama serve

# Then: Load extension in Chrome and click icon
```

The backend is already running, so extension will detect it immediately.

## File Structure

```
Moly/
├── INSTALL_AND_LOAD.sh       ← Run this (does everything)
├── NATIVE_HOST_SETUP.sh      ← Manual native host setup
├── START_MOLY.sh             ← Manual service startup
├── INSTALL.md                ← Detailed installation guide
├── README_INSTALLATION.md    ← This file
│
├── moly-extension/
│   └── dist/                 ← Load this in Chrome
│       └── manifest.json
│
├── moly-go/                  ← Backend source
│   ├── main.go
│   ├── moly                  ← Built binary (auto-runs CORS Proxy)
│   └── com.moly.backend_host.json
│
└── moly-proxy/               ← CORS Proxy source
    └── bin/moly-proxy.js     ← Auto-spawned by Go backend
```

## System Requirements

### Minimum
- 2GB RAM
- 100MB disk
- Modern Chrome/Chromium

### Recommended
- 4GB RAM
- 500MB disk
- Chrome 90+

### Build Requirements
- Go 1.16+
- Node.js 14+
- npm or yarn

## Uninstall

1. Go to `chrome://extensions`
2. Click "Remove" on Moly
3. Done

Optional: Clean up native host
```bash
# Linux
rm ~/.config/google-chrome/NativeMessagingHosts/com.moly.backend_host.json

# macOS
rm ~/Library/Application\ Support/Google/Chrome/NativeMessagingHosts/com.moly.backend_host.json
```

## FAQ

**Q: Does Moly read my messages?**  
A: No. You paste them manually. Moly never reads your browser.

**Q: Does Moly send data to servers?**  
A: Only to LLM providers (Claude/OpenAI) if you configure them. Local Ollama stays local.

**Q: Will I get banned from platforms?**  
A: No. Moly is a passive suggestion tool. You copy-paste manually - no automated actions.

**Q: Can I use it offline?**  
A: Yes, if you have Ollama running locally. No internet needed.

**Q: Does it work on Firefox/Safari?**  
A: No. Moly is a Chrome extension only. Those browsers don't support the native host API.

**Q: Can I run multiple profiles?**  
A: Yes. Each Chrome profile loads the extension independently. Conversation history is per-profile.

## Support

- Check INSTALL.md for detailed setup
- See TEST_RESULTS.md for verification
- Read ARCHITECTURE_COMPLETE.md for technical details

## Next Steps

1. Run: `./INSTALL_AND_LOAD.sh`
2. Click "Load unpacked" in Chrome
3. Click Moly icon
4. Start typing

---

**Status**: Ready to install  
**Time to working**: ~3 minutes  
**One-time setup**: Yes
