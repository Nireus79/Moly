# Moly Quick Start Guide

Get Moly up and running in 5 minutes.

## Option 1: Local AI (Recommended for Privacy)

### For Non-Programmers (Easiest)

```bash
npm install -g moly-installer
moly-installer
```

Follow the interactive wizard:
1. Choose provider (Ollama or LM Studio)
2. Select model (Mistral 7B recommended)
3. Enable auto-start
4. Let it download and install

**That's it!** Moly will auto-detect on first use.

### For Developers

#### Using Ollama

```bash
# 1. Install Ollama
# Download from https://ollama.ai

# 2. Start Ollama
ollama serve

# 3. Pull a model (in another terminal)
ollama pull mistral

# 4. Install CORS proxy
npm install -g moly-proxy

# 5. Start proxy (in another terminal)
moly-proxy

# 6. Install Moly extension (see below)
```

#### Using LM Studio

```bash
# 1. Download LM Studio
# From https://lmstudio.ai/

# 2. Launch LM Studio app

# 3. Search for and download model (e.g., "Mistral 7B")

# 4. Install Moly extension (see below)
```

## Option 2: Cloud AI (Less Private, No Setup)

### Using Claude

```bash
# 1. Get API key
# Visit: https://console.anthropic.com/keys

# 2. Install Moly extension

# 3. In Moly settings, enter your Claude API key

# Done! Use cloud AI without any local setup
```

### Using OpenAI

```bash
# 1. Get API key
# Visit: https://platform.openai.com/api-keys

# 2. Install Moly extension

# 3. In Moly settings, enter your OpenAI API key

# Done! Use cloud AI
```

## Install Moly Extension

### Chrome/Chromium

**Option A: From Chrome Web Store** (Coming soon)
- Visit: https://chromewebstore.google.com/...

**Option B: Load Manually**
```
1. Open chrome://extensions/
2. Enable "Developer mode" (top right)
3. Click "Load unpacked"
4. Select: /path/to/Moly/moly-extension/dist/
5. Moly appears in your toolbar
```

## First Use

1. **Click Moly icon** in toolbar (appears in top right)
2. **Sidebar opens** on the right side
3. **Type a message** or paste one from a chat app
4. **Select mode:**
   - Socratic: Guiding questions to refine your response
   - Direct: Ready-to-use suggestions
5. **Choose context:**
   - Formal: Professional tone
   - Friendly: Casual tone
   - Dating: Romantic tone
6. **Get suggestions** and copy to use

## Troubleshooting First Use

### "No provider found"
- Make sure you have Ollama/LM Studio running OR Claude/OpenAI API key configured
- Check Moly Settings panel (gear icon)

### "Auto-detection didn't work"
- Manually configure in Settings
- For Ollama: Check it's running at localhost:11435 (proxy) or 11434 (direct)
- For LM Studio: Check it's running at localhost:8000
- For Claude/OpenAI: Paste your API key

### "Suggestions are slow"
- Local models may take 5-10 seconds first time
- Subsequent requests should be faster
- Reduce message length for faster responses

### "CORS error"
- Make sure CORS proxy is running: `moly-proxy`
- Restart browser
- Check proxy is at port 11435

## Common Tasks

### Change AI Provider

1. Click Moly icon
2. Click gear (⚙️) icon
3. Select provider tab (Claude, OpenAI, Ollama)
4. Enter configuration
5. Click "Make Active"

### Switch Modes

Bottom of Moly sidebar shows:
- **Mode selector**: Toggle Socratic ↔ Direct
- **Context selector**: Toggle Formal ↔ Friendly ↔ Dating

### View Chat History

1. Click Moly icon
2. Scroll up in the chat area to see previous messages
3. Click message to see full conversation

### Export Conversation

1. Use browser developer tools (F12)
2. Check Chrome Storage for conversation data
3. Or manually copy/paste from Moly sidebar

## Performance Tips

1. **For speed:** Use Ollama with Mistral 7B on 8GB+ RAM
2. **For quality:** Use Claude or OpenAI APIs
3. **For privacy:** Use local Ollama/LM Studio (no data sent)
4. **For lower load:** Reduce context (shorter previous messages)

## System Requirements

**For Local Models:**
- OS: Windows 10+, macOS 10.15+, Linux
- RAM: 4GB minimum (8GB recommended)
- Disk: 10GB free for model download
- CPU: 2+ cores

**For Cloud Only:**
- OS: Any modern browser OS
- RAM: 1GB
- Disk: 100MB
- Internet: Required

## What's Next?

- Explore different models (Llama 2, Neural Chat, etc.)
- Try both Socratic and Direct modes
- Experiment with different contexts
- Switch between local and cloud providers

## Getting Help

- **Documentation:** https://github.com/Nireus79/Moly
- **Issues:** https://github.com/Nireus79/Moly/issues
- **Troubleshooting:** See TROUBLESHOOTING.md in this repo

## Pro Tips

1. **Multiple instances:** Keep Ollama and proxy running in background
2. **Keyboard shortcuts:** Use Ctrl+A to select all in input box
3. **API keys:** Store safely, never share or commit to git
4. **Models:** Try different models to find best for your needs
5. **Offline mode:** Local Ollama works completely offline after setup

---

**Happy messaging! 🚀**

For detailed setup, see README.md and TROUBLESHOOTING.md
