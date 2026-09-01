# Moly Testing Guide

Complete testing procedures for Moly installer, services, and functionality across all platforms.

---

## Prerequisites

- Node.js 16+ installed
- Chrome/Chromium browser
- About 10GB free disk space
- Time: ~1-2 hours for complete testing

---

## Part 1: Setup Local Environment

### Step 1.1: Verify Project Build

```bash
cd /path/to/Moly/Moly/moly-extension

# Build the extension
npm run build

# Verify build succeeded
ls -la dist/
# Should show: manifest.json, sidebar.html, settings.html, etc.
```

**Expected Result:** No errors, build completes in ~5 seconds

### Step 1.2: Start CORS Proxy

```bash
# Terminal 1: Install and start proxy
npm install -g moly-proxy
moly-proxy

# Expected output:
# [Moly Proxy] Started on http://127.0.0.1:11435
# [Moly Proxy] Proxying to http://localhost:11434
```

### Step 1.3: Verify Proxy is Running

```bash
# Terminal 2: Test proxy
curl http://localhost:11435/api/tags

# Expected: Will fail with "Ollama unreachable" (Ollama not running yet)
# That's OK - proxy is working
```

---

## Part 2: Load Extension in Chrome

### Step 2.1: Open Extension Management

```
1. Open Chrome
2. Type: chrome://extensions/
3. Press Enter
```

### Step 2.2: Enable Developer Mode

```
1. Look for "Developer mode" toggle (top right)
2. Click to enable it
3. Page refreshes, new buttons appear
```

### Step 2.3: Load Extension

```
1. Click "Load unpacked" button
2. Navigate to: /path/to/Moly/Moly/moly-extension/dist/
3. Select the dist folder
4. Click "Open"
```

**Expected Result:** 
- Moly extension appears in list
- Blue "M" icon appears in Chrome toolbar

---

## Part 3: Test Installer Scenarios

### Scenario 1: No Local Model Installed

**Expected Behavior**: Show setup dialog on "Start Setup" button

**Test Steps:**
1. Open Moly Settings
2. Go to "Local Models Status" section
3. If no local model detected, click "Start Setup"
4. InstallerDialog should appear with platform-specific instructions
5. Verify:
   - [ ] Platform name is correct (macOS/Linux/Windows)
   - [ ] Setup instructions are platform-appropriate
   - [ ] All buttons are visible and clickable

**Platform Verification:**

**macOS:**
- [ ] Instructions mention .dmg file
- [ ] Instructions include "Drag to Applications"
- [ ] Release page opens in new tab
- [ ] Download starts moly-installer-macos.dmg

**Linux:**
- [ ] Instructions mention chmod +x
- [ ] Instructions show terminal commands
- [ ] Release page opens correctly
- [ ] Download starts moly-installer-linux-x64

**Windows:**
- [ ] Instructions mention .exe file
- [ ] Instructions mention User Account Control prompt
- [ ] Release page opens correctly
- [ ] Download starts moly-installer-windows-x64.exe

### Scenario 2: Native Host Installed

**Expected Behavior**: Service control should work directly

**Test Steps:**
1. Install native host for your platform (see INSTALLATION_ARCHITECTURE.md)
2. Restart Chrome
3. Open Moly Settings → "Local Models Status"
4. Click service control buttons
5. Verify:
   - [ ] Start/Stop buttons appear
   - [ ] Services respond correctly
   - [ ] Status updates reflect actual state

**Platform Verification:**

**macOS:**
- [ ] LaunchAgent configured correctly
- [ ] Service starts/stops via Moly UI
- [ ] Auto-start works after reboot

**Linux:**
- [ ] Systemd service configured
- [ ] Service control works via native messaging
- [ ] Auto-start works after reboot

**Windows:**
- [ ] Task Scheduler entry created
- [ ] Service control works via native messaging
- [ ] Auto-start works after reboot

### Scenario 3: Model Management

**Test Steps:**
1. Open Moly Settings
2. If Ollama running with models:
   - [ ] Models appear in list
   - [ ] Can select different model
   - [ ] "Add Another Model" button visible
3. Click "Add Another Model"
   - [ ] Dropdown shows model options
   - [ ] Clicking "Copy Command" works
   - [ ] Or automated pull attempts (if native host available)

### Scenario 4: Suggestion Generation

**Test Steps:**
1. Open Moly sidebar on any website
2. Type or paste a message
3. Click "Generate Suggestions"
4. Verify:
   - [ ] Suggestions appear within 5 seconds
   - [ ] 3-5 suggestions shown
   - [ ] "Copy" button works
   - [ ] Mode (Socratic/Direct) affects suggestions
   - [ ] Context (Formal/Friendly/Dating) affects tone

---

## Part 4: Cross-Platform Testing

### macOS Testing

**Requirements:**
- macOS 10.13+
- Intel or Apple Silicon Mac

**Test Checklist:**
- [ ] Extension loads without errors
- [ ] Sidebar opens on messaging apps
- [ ] Ollama detection works
- [ ] Service control works
- [ ] Auto-start configured correctly
- [ ] All buttons functional
- [ ] No permission errors

### Linux Testing

**Requirements:**
- Ubuntu 20.04+ or similar
- x64 processor

**Test Checklist:**
- [ ] Extension loads in Chrome
- [ ] Ollama detection works
- [ ] CORS proxy starts
- [ ] Native host binary runs
- [ ] Systemd service configured
- [ ] Service control functional
- [ ] No permission errors

### Windows Testing

**Requirements:**
- Windows 10 or 11
- x64 processor

**Test Checklist:**
- [ ] Extension loads in Chrome
- [ ] Ollama detection works
- [ ] CORS proxy starts
- [ ] Native host binary runs
- [ ] Task Scheduler entry created
- [ ] Service control functional
- [ ] UAC prompts handled correctly

---

## Testing Checklist - Full Workflow

### Setup Phase
- [ ] Extension installs from unpacked folder
- [ ] No console errors on startup
- [ ] Settings page loads
- [ ] All UI components render

### Detection Phase
- [ ] Local model detection works
- [ ] Cloud provider detection works
- [ ] Status shows correctly
- [ ] Detection cache works (30-second window)

### Service Control Phase
- [ ] Start button works
- [ ] Stop button works
- [ ] Status updates reflect actual state
- [ ] Error messages are clear

### Model Management Phase
- [ ] Model list displays
- [ ] Can select different model
- [ ] Add model attempts automation
- [ ] Fallback to manual works

### Suggestion Phase
- [ ] Sidebar opens
- [ ] Can input text
- [ ] Suggestions generate
- [ ] Can copy suggestions
- [ ] Mode selection works
- [ ] Context selection works

---

## Error Scenarios to Test

1. **Ollama Crashes**: Kill Ollama mid-operation
   - [ ] Extension detects status change
   - [ ] Error message clear
   - [ ] Can restart via UI

2. **Network Disconnection**: Disable network
   - [ ] Timeout occurs (not infinite hang)
   - [ ] Error message shown
   - [ ] Connection recovery works

3. **Invalid API Keys**: Set bad cloud API key
   - [ ] Error message on generation
   - [ ] Can correct in settings
   - [ ] Can switch providers

4. **Out of Disk Space**: Simulate full disk
   - [ ] Model pull fails gracefully
   - [ ] Error message helpful
   - [ ] No data corruption

5. **Permissions Issues**: Run without proper permissions
   - [ ] Clear error messages
   - [ ] Guidance on fixing
   - [ ] Doesn't crash

---

## Performance Testing

- [ ] Extension loads in < 2 seconds
- [ ] Settings page responds quickly
- [ ] Suggestion generation < 10 seconds (Ollama)
- [ ] No memory leaks over 1 hour
- [ ] UI responsive during operations

---

## Regression Testing

Test that previously working features still work:
- [ ] Conversation history saves
- [ ] Settings persist across sessions
- [ ] Multiple models can be managed
- [ ] Provider switching works
- [ ] Chat modes function correctly

---

## Final Validation Before Release

- [ ] All tests pass on all supported platforms
- [ ] No console errors or warnings
- [ ] Performance acceptable
- [ ] Error messages are helpful
- [ ] Documentation is accurate
- [ ] User can complete full workflow without terminal
- [ ] Auto-start works after system reboot

---

*Use this guide for thorough testing before each release.*
