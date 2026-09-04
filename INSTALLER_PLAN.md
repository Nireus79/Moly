# Moly Installer/Uninstaller Strategy

## Installation Flow

### 1. Installer Package Contents
```
moly-installer/
├── moly-desktop (Electron app)
├── moly-extension (Chrome/Edge extension)
├── native-host (binary for system integration)
├── setup-wizard (interactive setup)
└── resources/
    └── ollama-detection-script
```

### 2. Installation Steps

**Step 1: Extract & Install**
- Copy desktop app to `~/.local/bin/moly` (executable)
- Register native host manifest
- Copy browser extension to Chrome extensions dir
- Set up autostart (systemd/LaunchAgent/Task Scheduler)

**Step 2: First Run Detection**
- Detect if Ollama is installed: check `which ollama` or registry
- Detect if local models exist: query `ollama list` or check `~/.ollama/models`
- Store detection result in `~/.config/moly/config.json`

**Step 3: First Run Prompt**
- Show UI: "Local models detected! Use local or cloud?"
- Options:
  - ✓ Use Local Models (if Ollama found)
  - ✓ Use Cloud Models (Claude/OpenAI)
  - ✓ Install Local Models Now

### 3. Model Installation UI

**In Sidebar Settings:**
```
Models Panel:
├── Detected Models (if local)
│   ├── mistral (2GB) [Start] [Stop] [Remove]
│   └── llama2 (4GB) [Start] [Stop] [Remove]
│
├── Available Models
│   ├── neural-chat [Install]
│   ├── orca-mini [Install]
│   └── Search: [______] [Search]
│
└── Installation Progress
    └── Downloading model-name... 45% [████░░░░░]
```

**Model Management:**
- List available models from Ollama registry
- Show install progress with download speed
- Start/stop models via Ollama API
- Remove models (delete from disk)

### 4. Model Detection & Startup

**On App Start:**
1. Check if Ollama is running: `curl http://127.0.0.1:11434/api/tags`
2. If not running but installed: offer to start it
3. List installed models from Ollama response
4. Save to config: which provider (local/cloud), which model

**Provider Selection:**
- User selects in settings
- Sidebar UI sends requests to desktop app
- Desktop app routes to correct provider (local Ollama or cloud API)

## Uninstallation Flow

### 1. Uninstaller Prompt

```
Uninstall Moly?

This will remove:
- Desktop application
- Browser extension
- System integration files

Local models (if installed) are stored separately.
Keep local models? [Yes] [No] [Cancel]
```

### 2. Uninstall Actions

**If User Keeps Models:**
- Keep: `~/.ollama/models/` directory
- Keep: Ollama binary (if installed via Moly)
- Remove: Moly app, extension, native host
- Remove: `~/.config/moly/` config (except model list)

**If User Removes Models:**
- Remove: Everything including `~/.ollama/`
- Remove: All Moly infrastructure
- Clean: Browser extension data

### 3. Cleanup

**Remove Files:**
- `~/.local/bin/moly` (app binary)
- `~/.local/bin/moly-native-host` (native host)
- `~/.config/chrome/NativeMessagingHosts/com.moly.native_host.json`
- `~/.config/moly/` (config directory)
- systemd/LaunchAgent/Task Scheduler entries

**Keep (if user chooses):**
- `~/.ollama/` (Ollama directory with models)
- Ollama binary (if separately installed)

## Implementation Details

### A. Detect Ollama Installation

```bash
# Linux/macOS
which ollama

# Windows
where ollama

# Check if running
curl http://127.0.0.1:11434/api/tags

# List models
ollama list
```

### B. Model Management API

**List Models:**
```
GET http://127.0.0.1:11434/api/tags
Response: { "models": [...] }
```

**Install Model:**
```bash
ollama pull model-name
# Or via API (streaming)
POST http://127.0.0.1:11434/api/pull
{"name": "model-name"}
```

**Remove Model:**
```bash
ollama rm model-name
```

**Start/Stop Model:**
- Ollama auto-starts models on first request
- Stop by killing the process (graceful)

### C. Sidebar Integration

**New Settings Tab: "Models"**
- List installed models
- Show model size and status
- Install new models UI
- Start/stop buttons
- Delete model option

**Provider Selection:**
- Global setting: Local vs Cloud
- If Local selected: model dropdown shows installed models
- If Cloud selected: API key input for Claude/OpenAI

### D. Config Storage

```json
~/.config/moly/config.json
{
  "provider": "local",
  "model": "mistral",
  "ollama_installed": true,
  "ollama_running": true,
  "models": [
    {"name": "mistral", "size": "2.3GB"},
    {"name": "llama2", "size": "4.0GB"}
  ],
  "cloud_provider": "claude",
  "api_keys": {}
}
```

## Feasibility Assessment

| Feature | Feasible? | Notes |
|---------|-----------|-------|
| Detect Ollama | ✓ Yes | Via `which` or registry queries |
| List models | ✓ Yes | Ollama API or `ollama list` |
| Install models | ✓ Yes | `ollama pull` command |
| Start/stop models | ✓ Yes | Auto-start on request, kill on stop |
| Provider switching | ✓ Yes | Desktop app routes to correct provider |
| Plugin & play | ✓ Yes | Detect + use existing models |
| Clean uninstall | ✓ Yes | Selective file removal |

## Timeline

- **Phase 1**: Installer script (Windows/macOS/Linux) - 4-6 hours
- **Phase 2**: First-run detection & setup wizard - 3-4 hours
- **Phase 3**: Model management UI in sidebar - 4-5 hours
- **Phase 4**: Testing on all platforms - 3-4 hours

**Total: ~14-19 hours**

## Next Steps

1. ✓ Core product complete (desktop + extension + auto-launch)
2. → Build cross-platform installers
3. → Implement first-run setup
4. → Add model management UI
5. → Test uninstaller flows
