# Phase 1: Desktop App API Endpoints - COMPLETE

## What Was Built

### 1. Model Manager Service (`moly-desktop/src/services/modelManager.js`)
Created a new service that:
- Communicates with native host via native messaging protocol
- Manages config file (`~/.config/moly/config.json`)
- Provides methods for:
  - First-run detection (check if Ollama installed)
  - List installed models
  - Pull (install) new models
  - Remove models
  - Start/stop Ollama service

### 2. Desktop App HTTP Endpoints (`moly-desktop/src/main.js`)
Added 8 new API endpoints:

```javascript
GET  /api/status                  // App health check
GET  /api/first-run-check         // Detect Ollama, return setup status
GET  /api/models/list             // List installed models
POST /api/models/pull?name=NAME   // Install model
POST /api/models/remove?name=NAME // Delete model
POST /api/ollama/start            // Start Ollama service
POST /api/ollama/stop             // Stop Ollama service
GET  /api/settings                // Get saved config
POST /api/settings                // Save config
```

### 3. Native Host Communication
- Uses native messaging protocol (4-byte length prefix)
- Spawns moly-host.py subprocess with JSON commands
- Handles responses correctly
- Maps to native host actions:
  - check-ollama
  - get-models
  - pull-model
  - remove-model
  - start-ollama
  - stop-ollama

### 4. Config File System
- Location: `~/.config/moly/config.json`
- Schema includes:
  - provider (local, claude, openai)
  - model (active model)
  - ollama_installed / ollama_running
  - installed_models list
  - api_keys
  - first_run_complete flag

## Files Modified

1. **moly-desktop/src/main.js**
   - Added url module import
   - Added ModelManager import
   - Added URL parsing logic
   - Added 8 new API endpoints with async handlers
   - Added request body reader helper

2. **moly-desktop/src/services/modelManager.js** (NEW)
   - Config management
   - Native host communication
   - Model operations

## Testing

The endpoints are ready to test but require:
1. Desktop app running on 127.0.0.1:11436
2. Native host binary in ~/.local/bin/moly-native-host
3. Python 3 with Python modules (subprocess, json, etc.)

## What's Next

Phase 2: Wire sidebar to use these APIs
- Update sidebar HTML to load models from /api/models/list
- Add Ollama status to first-run check
- Load/save settings via /api/settings
- Add model management UI (install, remove, start, stop)

## Key Design Decisions

1. **Native Messaging Protocol**: Uses official browser native messaging format for compatibility
2. **Config File Storage**: Lightweight JSON file for persistence
3. **Async API Handlers**: All endpoints are async-ready for future streaming operations
4. **Error Handling**: All methods return errors in response JSON, no exceptions

## Architecture

```
Browser Extension
       ↓ (HTTP calls)
   /api/models/list ──→ Desktop App HTTP Server (Node.js)
   /api/ollama/start         ↓
   /api/settings        Model Manager Service
                             ↓ (native messaging protocol)
                       moly-host.py (Python subprocess)
                             ↓
                       Ollama API (localhost:11434)
                       System services (systemd, etc.)
```
