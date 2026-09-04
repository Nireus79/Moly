# Moly Testing Guide - Phase 6

## Overview

Complete testing checklist for Moly installer/uninstaller with full model management.

---

## Pre-Testing Setup

### 1. Verify Files
```bash
# Check all required files exist
ls moly-desktop/src/services/modelManager.js
ls moly-desktop/src/main.js
ls moly-installer/native-host/moly-host.py
ls moly-installer/moly-uninstall.sh
ls moly-installer/moly-uninstall.bat
ls moly-extension/src/content.js
```

### 2. Install Dependencies
```bash
cd moly-desktop
npm install
cd ../moly-extension
npm install
```

### 3. Build Extension
```bash
cd moly-extension
npm run build
# Check that dist/ folder is created with manifest.json
```

---

## Phase 1: API Endpoints Testing

### Test 1.1: App Startup
```bash
cd moly-desktop
npm start
# Verify: App starts without errors
# Verify: No window visible (headless mode)
# Check console: "[Moly] Sidebar server listening on 127.0.0.1:11436"
```

### Test 1.2: API Status Endpoint
```bash
# In another terminal
curl http://127.0.0.1:11436/api/status
# Expected: {"status":"running"}
```

### Test 1.3: First-Run Check Endpoint
```bash
curl http://127.0.0.1:11436/api/first-run-check
# Expected: {"ollama_installed": false, "ollama_running": false, "first_run_complete": false}
# Or with Ollama: {"ollama_installed": true, "ollama_running": false, "first_run_complete": false}
```

### Test 1.4: Settings Endpoints
```bash
# GET settings (should be default)
curl http://127.0.0.1:11436/api/settings

# POST settings
curl -X POST http://127.0.0.1:11436/api/settings \
  -H "Content-Type: application/json" \
  -d '{"provider":"local","model":"mistral"}'

# GET settings (should show updated values)
curl http://127.0.0.1:11436/api/settings
```

### Test 1.5: Sidebar HTML Endpoint
```bash
curl http://127.0.0.1:11436/sidebar.html | head -20
# Expected: HTML with <html>, <head>, <body>, etc.
```

---

## Phase 2: Sidebar UI Testing

### Test 2.1: Open Sidebar in Browser
```bash
1. Open a webpage (e.g., https://www.google.com)
2. Open Chrome DevTools (F12)
3. Go to chrome://extensions/
4. Enable Developer Mode (top right)
5. Click "Load unpacked"
6. Select moly-extension/dist/ folder
7. Find "Moly" extension in list
```

### Test 2.2: Click Moly Icon
```bash
1. Go back to google.com
2. Look for Moly icon in extension bar
3. Click it
4. Verify: Sidebar appears on right side of page
5. Verify: Sidebar shows "Moly" header
```

### Test 2.3: Sidebar Elements
```bash
Sidebar should show:
✓ Header with "Moly" title
✓ Settings button (⚙️)
✓ Expand button (↗️)
✓ Messages area (empty, shows "Start a conversation...")
✓ Settings panel (collapsed) with dropdowns:
  - Model (should load from API or show "Loading...")
  - Tone (Friendly/Formal/Playful)
  - Mode (Direct/Socratic)
✓ Ollama section showing status
✓ Input area with textarea and buttons (Send, Clear)
✓ Model installation section (if Ollama is local provider)
```

### Test 2.4: Settings Panel Toggle
```bash
1. Click ⚙️ button
2. Verify: Settings panel expands
3. Click ⚙️ again
4. Verify: Settings panel collapses
```

### Test 2.5: Sidebar Expand
```bash
1. Click ↗️ button
2. Verify: Sidebar expands to 100vw (full browser width)
3. Message area should still be readable
4. Click ↗️ again
5. Verify: Sidebar collapses back to 400px
```

---

## Phase 3: First-Run Setup Testing

### Test 3.1: First Run Detection
```bash
1. Delete ~/.config/moly/config.json (or it doesn't exist)
2. Open sidebar
3. Verify: Setup wizard modal appears
4. Modal should show: "Welcome to Moly"
5. Three options visible:
   - Local Models (Ollama)
   - Claude (Anthropic)
   - ChatGPT (OpenAI)
```

### Test 3.2: Select Local Models (Ollama)
```bash
1. Click "Local Models (Ollama)"
2. Verify: Option gets selected (highlighted)
3. Verify: API key input disappears
4. Click "Continue"
5. Verify: Modal closes
6. Verify: /api/settings saved with provider: "local"
7. Verify: Settings file created at ~/.config/moly/config.json
8. Open sidebar again - should NOT show setup wizard
```

### Test 3.3: Select Claude
```bash
1. Delete ~/.config/moly/config.json again
2. Open sidebar
3. Click "Claude (Anthropic)"
4. Verify: Option gets selected (highlighted)
5. Verify: API key input appears
6. Enter a test API key (can be fake for this test)
7. Click "Continue"
8. Verify: Modal closes
9. Verify: /api/settings has provider: "claude"
10. Verify: api_keys.claude is saved
```

### Test 3.4: API Key Validation
```bash
1. Delete ~/.config/moly/config.json
2. Open sidebar
3. Click "Claude (Anthropic)"
4. Leave API key empty
5. Click "Continue"
6. Verify: Alert says "Please enter your API key"
7. Enter API key and continue
8. Verify: Works normally
```

### Test 3.5: Ollama Not Installed Case
```bash
# Requires system without Ollama
1. Uninstall Ollama if installed
2. Delete ~/.config/moly/config.json
3. Open sidebar
4. Verify: Setup wizard appears
5. Verify: "Local Models (Ollama)" is disabled/grayed out
6. Message says "Ollama not installed on this system"
7. Can only select Claude or OpenAI
```

---

## Phase 4: Model Management Testing

### Test 4.1: Model List Population (Ollama)
```bash
# If Ollama is running with models installed:
1. Set provider to "local"
2. Open sidebar
3. Verify: Model dropdown shows installed models
4. Each model should be selectable
5. Try selecting different models
6. Verify: Selection persists when modal closes
```

### Test 4.2: Model List Empty (Ollama not running)
```bash
1. Stop Ollama service
2. Delete ~/.config/moly/config.json
3. Do first-run setup, select "Local"
4. Open sidebar
5. Verify: Model dropdown shows "No local models found"
6. Verify: Dropdown is disabled
```

### Test 4.3: Ollama Status Indicator
```bash
1. Start Ollama manually
2. Open sidebar
3. Verify: "Ollama: Running" appears in green
4. Stop Ollama
5. Verify: Status updates to "Ollama: Installed but not running" in orange
```

### Test 4.4: Start Ollama Button
```bash
1. Stop Ollama
2. Open sidebar
3. Click "Start" button in Ollama section
4. Wait 2-3 seconds
5. Verify: Status changes to green "Ollama: Running"
6. Verify: Model dropdown populates with installed models
```

### Test 4.5: Stop Ollama Button
```bash
1. Ollama should be running
2. Open sidebar
3. Click "Stop" button
4. Wait 1-2 seconds
5. Verify: Status changes to orange "Ollama: Installed but not running"
6. Verify: Model dropdown becomes disabled
```

---

## Phase 5: Model Installation Testing

### Test 5.1: Model Installation Section Visibility
```bash
# Ollama/Local provider:
1. Set provider to "local"
2. Open sidebar
3. Verify: "Install Models" section visible
4. Shows list of available models:
   - mistral (4.1GB)
   - llama2 (3.8GB)
   - neural-chat (4.0GB)
   - orca-mini (1.3GB)
   - dolphin-mixtral (26.0GB)
5. Each model has "Install" button

# Cloud provider:
1. Set provider to "claude" or "openai"
2. Open sidebar
3. Verify: "Install Models" section hidden
4. Message appears: "Model installation only available with Ollama"
```

### Test 5.2: Install Small Model
```bash
1. Provider set to "local"
2. Ollama running
3. Click "Install" for orca-mini
4. Verify: Progress bar appears
5. Shows: "Installing orca-mini..."
6. Progress bar fills over time
7. Verify: Model actually downloads (~1-5 minutes for orca-mini)
8. On success: "Successfully installed orca-mini!"
9. Progress bar hides automatically
10. Model dropdown refreshes to show new model
```

### Test 5.3: Install Large Model (Optional)
```bash
1. Click "Install" for dolphin-mixtral (26GB)
2. Verify: Progress appears
3. Let it download for at least 30 seconds
4. Verify: Progress updates
5. Can cancel by navigating away (progress hides)
```

### Test 5.4: Installation Error Handling
```bash
1. Try to install a model with invalid name (if supported)
2. Verify: Error message appears
3. Shows: "Installation failed: [error details]"
4. Progress bar clears
5. Can retry by clicking Install again
```

### Test 5.5: Refresh Models Button
```bash
1. After installing models
2. Click refresh button (🔄) next to Model dropdown
3. Verify: Model list refreshes
4. New model appears in dropdown
```

---

## Phase 6: Chat Functionality Testing

### Test 6.1: Local Model Chat
```bash
1. Provider: "local"
2. Model: "mistral" or similar
3. Ollama: running
4. Type message: "Hello"
5. Click "Send"
6. Verify: Message appears in chat
7. Verify: Loading indicator shows
8. Verify: Response appears from Ollama
9. Verify: Model, tone, mode affect response
```

### Test 6.2: Cloud Provider Chat (Claude)
```bash
1. Provider: "claude"
2. API Key: valid Claude API key in ~/.config/moly/config.json
3. Type message: "Hello"
4. Click "Send"
# This will fail if API not configured but shows proper error handling
```

### Test 6.3: Message History
```bash
1. Send multiple messages
2. Verify: All messages appear in chat
3. User messages: blue/purple, right-aligned
4. Assistant responses: gray, left-aligned
5. Scroll through message history
```

### Test 6.4: Clear Chat
```bash
1. Send several messages
2. Click "Clear" button
3. Verify: Chat clears
4. Shows "Start a conversation..."
5. Message history cleared (but config persists)
```

---

## Phase 7: Uninstaller Testing

### Test 7.1: Linux/macOS Uninstaller
```bash
# Make sure ~/.config/moly/config.json exists first
cd moly-installer
bash moly-uninstall.sh

# Prompts: "Keep local models? (y/n)"
# Type: y
# Verify:
✓ Messages about removing files
✓ ~/.config/moly removed
✓ ~/.local/bin/moly-native-host removed
✓ Ollama models preserved in ~/.ollama/models

# Second test with keep_models=false:
bash moly-uninstall.sh
# Type: n
# Verify:
✓ ~/.ollama/models removed too
✓ Config directory removed
✓ Only native host files cleaned
```

### Test 7.2: Windows Uninstaller
```bash
# Run as Administrator (or will fail on some operations)
cd moly-installer
moly-uninstall.bat

# Prompts: "Keep local models? (y/n):"
# Type: y
# Verify:
✓ Messages appear
✓ %APPDATA%\moly removed
✓ Native host exe removed
✓ Ollama models preserved in %APPDATA%\.ollama\models
```

### Test 7.3: Post-Uninstall Check
```bash
# After uninstalling:
1. Check that app no longer runs on :11436
2. curl http://127.0.0.1:11436/api/status
   # Expected: Connection refused
3. Extension still shows in chrome://extensions/
4. Instructions tell user to manually remove extension
5. Click extension icon - should show error or nothing
```

---

## Phase 8: Cross-Platform Testing

### Platform Matrix

| Platform | OS Version | Ollama | Status |
|----------|-----------|--------|--------|
| Linux | Ubuntu 22.04 | Yes | To Test |
| Linux | Ubuntu 20.04 | No | To Test |
| macOS | 13.x (Intel) | Yes | To Test |
| macOS | 14.x (ARM) | Yes | To Test |
| Windows | 10 | No | To Test |
| Windows | 11 | Yes | To Test |

### Platform-Specific Checks

**Linux:**
- [ ] Native host path: ~/.local/bin/moly-native-host
- [ ] Config path: ~/.config/moly/config.json
- [ ] Systemd integration working
- [ ] Uninstaller removes all files

**macOS:**
- [ ] Native host path: ~/.local/bin/moly-native-host
- [ ] Config path: ~/.config/moly/config.json
- [ ] LaunchAgent created at ~/Library/LaunchAgents/com.moly.app.plist
- [ ] Uninstaller removes all files

**Windows:**
- [ ] Native host path: %USERPROFILE%\.local\bin\moly-native-host.exe
- [ ] Config path: %APPDATA%\moly\config.json
- [ ] Task Scheduler entry created
- [ ] Uninstaller via .bat file works

---

## Error Scenarios Testing

### Test E1: No Internet Connection
```bash
1. Disable internet
2. Try to use cloud provider
3. Verify: Error message shows
4. Verify: Can still use local Ollama (if running)
```

### Test E2: Corrupted Config File
```bash
1. Edit ~/.config/moly/config.json with bad JSON
2. Open sidebar
3. Verify: Falls back to defaults
4. Verify: App still works
5. Verify: Config resets on save
```

### Test E3: Ollama Crash
```bash
1. Ollama running normally
2. Kill Ollama process: pkill ollama
3. Try to chat
4. Verify: Error message
5. Status updates to "not running"
6. Can click Start to restart
```

### Test E4: Invalid API Key
```bash
1. Provider: "claude"
2. Invalid API key in config
3. Try to send message
4. Verify: Error from Claude API
5. Shows: "Error: Unauthorized" or similar
```

---

## Checklist Summary

### Core Functionality
- [ ] App starts without errors
- [ ] Desktop server listens on :11436
- [ ] All 8 API endpoints respond correctly
- [ ] Config file created and persisted
- [ ] First-run setup works
- [ ] Sidebar loads correctly
- [ ] All UI elements visible

### First-Run Setup
- [ ] Modal appears on first run
- [ ] Three provider options shown
- [ ] Selection works
- [ ] API key input shows/hides
- [ ] Config saved correctly
- [ ] Modal doesn't appear on subsequent runs

### Model Management
- [ ] Models list loads from API
- [ ] Ollama detection works
- [ ] Status indicator shows correct state
- [ ] Start/Stop buttons work
- [ ] Model installation works
- [ ] Progress indicator shows
- [ ] Model list refreshes after install

### Settings & Persistence
- [ ] Settings panel toggle works
- [ ] Selections persist on reload
- [ ] Model dropdown respects settings
- [ ] Provider affects available features
- [ ] Config file correctly stores all data

### Chat
- [ ] Messages send and display
- [ ] Model/tone/mode dropdowns work
- [ ] Clear button works
- [ ] Response appears in chat

### Uninstall
- [ ] Uninstaller script runs
- [ ] Asks about model cleanup
- [ ] Files removed correctly
- [ ] Models preserved if chosen
- [ ] Post-uninstall instructions shown

### Cross-Platform
- [ ] Works on Windows
- [ ] Works on macOS
- [ ] Works on Linux
- [ ] Paths correct for each OS
- [ ] Config persists on each OS

---

## Final Sign-Off

When all tests pass:

```bash
git tag -a v1.0-beta -m "Phases 1-6 complete - ready for testing"
git push --tags
```

---

## Notes

- Test with both Ollama installed and not installed
- Test with valid and invalid API keys
- Test with internet on and off
- Test config file corruption and recovery
- Test rapid clicking and concurrent operations
- Monitor console for errors (F12 in browser)
- Check system logs for permission issues

---

**Total Testing Time Estimate**: 4-6 hours  
**Expected Issues**: Minor UI glitches, edge cases  
**Success Criteria**: All functionality works on at least one platform
