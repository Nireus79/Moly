# Implementation Audit: Installer/Uninstaller with Model Management

## Summary
**Status**: MOSTLY BUILT - 60-70% complete  
**Ready to Build**: Last 30-40% (desktop app API endpoints, first-run wizard integration, uninstaller)

---

## What's Already Done

### 1. Native Host (`moly-installer/native-host/moly-host.py`) ✓

All core functions already implemented:

```python
✓ check_ollama()              # Detect if Ollama installed
✓ start_ollama()              # Start Ollama service
✓ stop_ollama()               # Stop Ollama service
✓ get_installed_models()      # List installed models
✓ pull_model(model_name)      # Download model via ollama pull
✓ remove_model(model_name)    # Delete model
✓ cleanup_moly(keep_models)   # Uninstall with model choice
✓ get_system_info()           # Platform detection
```

These functions do everything needed for model detection and management.

### 2. Desktop App Sidebar (`moly-desktop/src/main.js`) ✓

- Serves HTTP server on :11436
- Delivers /sidebar.html with complete embedded UI
- Has /api/status endpoint
- Sidebar includes:
  - Chat interface
  - Settings panel (Model, Tone, Mode dropdowns)
  - localStorage persistence
  - Expand to full-width feature

### 3. UI Components (`moly-extension/src/settings/components/`) ✓

Already built:
- **SetupWizard.tsx** - Full setup flow
- **ModelSelectionDialog.tsx** - Choose local vs cloud
- **ModelManagement.tsx** - Manage installed models
- **LocalModelSetup.tsx** - Install new models
- **ServiceManager.tsx** - Start/stop Ollama
- **InstallerDialog.tsx** - Installation UI
- **LocalModelStatus.tsx** - Show model status

All UI components for model management already exist.

### 4. Settings Store (`moly-extension/src/stores/settingsStore.ts`) ✓

Exists and handles:
- Provider selection (local, claude, openai)
- Model selection
- API keys
- Persistence

---

## What Needs to be Built

### 1. Desktop App API Endpoints (HIGH PRIORITY)

Desktop app only has:
```javascript
GET /sidebar.html       ✓ Exists
GET /api/status         ✓ Exists
POST /api/generate      ? (calls Ollama directly from sidebar)
```

**NEEDED** - Add these endpoints:

```javascript
// Model Management
GET  /api/models/list             # List installed models
POST /api/models/pull/:name       # Download model
POST /api/models/remove/:name     # Delete model
POST /api/ollama/start            # Start Ollama service
POST /api/ollama/stop             # Stop Ollama service
GET  /api/ollama/status           # Check if running

// Settings
GET  /api/settings                # Get saved config (provider, model, etc)
POST /api/settings                # Save config

// First Run
GET  /api/first-run-check         # Detect Ollama, list models
```

### 2. Desktop App Routes These to Native Host

Desktop app (Node.js) needs to:
1. Receive HTTP request
2. Call native host via native messaging
3. Return response as JSON

Currently native host is only launched for app startup, NOT for model management.

### 3. First-Run Setup Flow

**What needs to happen on first run:**

1. Extension loads → calls desktop app
2. Desktop app detects Ollama
3. If Ollama found:
   - List installed models
   - Ask user: "Use local models or cloud?"
4. If no Ollama:
   - Ask user: "Use cloud models or install local?"
5. Save choice to config
6. Show sidebar

**Currently:** Setup wizard components exist but aren't wired up to first-run flow.

### 4. Uninstaller Script

**What needs to be built:**

Platform-specific uninstallers:
- `moly-uninstall.sh` (Linux/macOS)
- `moly-uninstall.bat` (Windows)

Must:
1. Show dialog: "Keep local models?"
2. Remove app, extension, native host files
3. Optionally remove ~/.ollama/ if user chooses
4. Clean config files

**Currently:** `cleanup_moly()` function exists in native host, but there's no standalone uninstaller script.

### 5. Config Management

**Need to add:**

```json
~/.config/moly/config.json
{
  "version": "1.0",
  "provider": "local|claude|openai",
  "model": "mistral|gpt-4|claude-3-sonnet",
  "ollama_installed": true,
  "ollama_running": false,
  "installed_models": [
    {"name": "mistral", "size": "2.3GB", "date_installed": "2026-09-04"},
    {"name": "llama2", "size": "4.0GB", "date_installed": "2026-09-03"}
  ],
  "api_keys": {
    "claude": "sk-...",
    "openai": "sk-..."
  },
  "first_run_complete": true
}
```

**Desktop app needs:**
- Load config on startup
- Save config on changes
- Initialize on first run

---

## What's Partially Done

### 1. Sidebar Model Selection

**Exists:** Settings panel with Model dropdown  
**Problem:** Dropdown is hardcoded:
```javascript
<option value="mistral">Mistral</option>
<option value="llama2">Llama2</option>
<option value="neural-chat">Neural Chat</option>
```

**Needs:** Populate dropdown from `/api/models/list` response

### 2. Native Host Binary Compilation

**Exists:** Python source code  
**Needs:** Compiled binary for distribution
- moly-host (Linux)
- moly-host.exe (Windows)
- moly-host (macOS)

(Currently using Python source, need PyInstaller binary for deployment)

---

## Implementation Priority (Next Steps)

### Phase 1: Desktop App API Layer (4-6 hours)
1. Add HTTP endpoints to main.js
2. Wire desktop app → native host messaging
3. Return JSON responses
4. Add config file management

### Phase 2: First-Run Setup (3-4 hours)
1. Call /api/first-run-check on app start
2. Show setup wizard if first run
3. Wire wizard buttons to API endpoints
4. Save settings to config

### Phase 3: Sidebar Integration (3-4 hours)
1. Populate model dropdown from API
2. Add start/stop buttons for Ollama
3. Show model management UI
4. Wire to API endpoints

### Phase 4: Uninstaller (2-3 hours)
1. Create uninstall script
2. Add dialog for model cleanup choice
3. Call cleanup_moly() from native host
4. Test complete removal

### Phase 5: Testing & Polish (3-4 hours)
1. Test first-run flow
2. Test model installation
3. Test start/stop
4. Test uninstall
5. Test cross-platform

**Total: 15-21 hours**

---

## File Changes Needed

### Desktop App (moly-desktop/src/main.js)
- Add 7 new endpoints (model list, install, remove, start, stop)
- Add config loading/saving
- Add first-run detection
- Add native host communication for model ops

### Native Host (moly-installer/native-host/moly-host.py)
- No changes needed (all functions exist)
- Just needs to be called via native messaging

### Extension (moly-extension)
- Use existing SetupWizard.tsx
- Wire sidebar to call desktop app endpoints
- Update model dropdown to use API

### Config System
- Create `~/.config/moly/config.json` handler
- First-run check logic
- Provider/model persistence

### Uninstaller
- Create `moly-uninstall.sh`
- Create `moly-uninstall.bat`
- Call native host cleanup

---

## Key Design Decisions

| Item | Decision | Reasoning |
|------|----------|-----------|
| **Model List Source** | Query /api/models/list (desktop → native host) | Single source of truth, keeps logic centralized |
| **Config Storage** | `~/.config/moly/config.json` | Standard Linux/macOS location, Windows handled via APPDATA |
| **First Run** | Check desktop app on sidebar load | Extension can't run sync code, must query app |
| **Model Install Progress** | Stream events via POST (chunked response) | Shows download progress to user |
| **Ollama Start/Stop** | Via native host subprocess calls | Need OS-level process management |

---

## Success Criteria

- [ ] Detect Ollama on first run
- [ ] Ask user: local or cloud models?
- [ ] Populate model dropdown from installed models
- [ ] User can install new models through UI
- [ ] User can start/stop Ollama from UI
- [ ] Settings persist across sessions
- [ ] Uninstaller asks about keeping models
- [ ] Complete removal works
- [ ] Cross-platform tested

---

## Current Files Status

```
✓ DONE
- moly-desktop/src/main.js (sidebar server exists)
- moly-installer/native-host/moly-host.py (all model functions exist)
- moly-extension/src/settings/components/ (UI components exist)
- moly-extension/src/stores/settingsStore.ts (config store exists)

🚧 IN PROGRESS / PARTIAL
- moly-desktop/src/main.js (needs API endpoints)
- moly-desktop/src/main.js (needs config file handling)
- moly-extension/src/sidebar/ (needs to call APIs)

📋 TODO
- moly-desktop/src/main.js (native messaging integration for models)
- config.json system
- First-run setup orchestration
- moly-uninstall.sh / moly-uninstall.bat
- Binary compilation (PyInstaller)
```

---

## Next Immediate Steps

1. **Add API endpoints to desktop app** - Model list, install, delete, start, stop
2. **Add config file management** - Load/save ~/.config/moly/config.json
3. **Add first-run check endpoint** - /api/first-run-check
4. **Wire sidebar to APIs** - Update model dropdown, add control buttons
5. **Create uninstall scripts** - Platform-specific uninstallers
6. **Test full flow** - Install → setup → use → uninstall

Ready to start with Phase 1?
