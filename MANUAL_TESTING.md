# Moly Manual Testing Guide

## Quick Start - Test Everything in 10 Minutes

### Step 1: Start the Desktop App
```bash
cd ~/vs_projects/Moly/Moly/moly-go
./moly-desktop
```

You should see:
```
[Moly] Desktop app initialized
[Moly] Sidebar server listening on 127.0.0.1:11436
```

### Step 2: Open Extension in Browser
1. Open Chrome/Chromium
2. Go to `chrome://extensions`
3. Enable "Developer mode" (top right)
4. Click "Moly" extension icon in top right
5. Click it to show the sidebar

### Step 3: Complete First-Run Setup
The sidebar should show a modal with 3 options:
- ✅ Local Models (Ollama)
- ✅ Claude (Anthropic)
- ✅ ChatGPT (OpenAI)

**Test 1: Try Local Models**
1. Click "Local Models (Ollama)"
2. Click "Continue"
3. Sidebar should close modal and show main UI

---

## Full Manual Test Plan

### Part 1: First-Run Setup (2 minutes)

#### Test 1.1: Local Provider Setup
1. Click extension → shows setup modal
2. Click "Local Models (Ollama)"
3. Click "Continue"
4. ✅ Modal disappears
5. ✅ Main UI appears
6. ✅ Model dropdown shows "mistral"
7. ✅ Mode dropdown shows "direct"

#### Test 1.2: Changing Provider (restart after each)
**To reset**: Delete `~/.config/moly/config.json` and restart app

1. Click extension → setup modal
2. Click "Claude (Anthropic)"
3. ✅ API key input appears with note
4. Enter fake key: `sk-test-123`
5. ✅ "Continue" button enables
6. Click "Continue"
7. ✅ Setup completes

Repeat for OpenAI with key `test-openai-key`

---

### Part 2: Settings Panel (3 minutes)

#### Test 2.1: Model Selection
1. Go to Settings Panel (top of sidebar)
2. Click Model dropdown
3. ✅ Shows 4 options: mistral, llama2, claude-3-sonnet, gpt-3.5-turbo
4. Select "llama2"
5. ✅ Dropdown updates
6. Close and reopen sidebar
7. ✅ Model is still "llama2" (persisted)

#### Test 2.2: Mode Selection
1. Click Mode dropdown
2. ✅ Shows 2 options: "Direct (Ready-to-use responses)" and "Socratic (Guiding questions)"
3. Select "Socratic"
4. ✅ Dropdown updates
5. Close and reopen sidebar
6. ✅ Mode is still "Socratic" (persisted)

---

### Part 3: Ollama Management (3 minutes)

**Prerequisites**: Have Ollama installed
```bash
# Install Ollama
curl https://ollama.ai/install.sh | sh

# Or on Mac
brew install ollama
```

#### Test 3.1: Ollama Status
1. Look at "Ollama" section in sidebar
2. Status should show:
   - "Checking..." initially
   - Then "Not Installed" or "Installed (Stopped)" or "Running"

#### Test 3.2: Start/Stop Ollama
1. Click "Start" button
2. ✅ Status changes to "Running" (within 3 seconds)
3. Click "Stop" button
4. ✅ Status changes to "Installed (Stopped)"
5. Click "Start" again
6. ✅ Status changes back to "Running"

#### Test 3.3: Model Installation
1. Make sure Ollama is running
2. Look at "Install Models" section
3. Click "Install" for "mistral"
4. ✅ Button changes to "Installing..."
5. Wait ~30 seconds (first download takes time)
6. ✅ Button changes back to "Install"
7. ✅ Model appears in "Available Models" section

#### Test 3.4: Model List
1. After installing mistral, look at "Available Models"
2. ✅ Shows "mistral" with a "Remove" button
3. Click "Remove"
4. ✅ Confirmation dialog appears
5. Click OK
6. ✅ Model disappears from list

---

### Part 4: Chat Functionality (2 minutes)

**Prerequisites**: Have Ollama running with at least one model installed

#### Test 4.1: Direct Mode Chat
1. Model dropdown: "mistral"
2. Mode dropdown: "direct"
3. Type message: "What is 2+2?"
4. Click "Send" (or press Enter)
5. ✅ Your message appears in blue on right
6. ✅ "Thinking..." appears
7. ✅ Response appears below (takes 5-10 seconds)
8. ✅ Response is direct answer

#### Test 4.2: Socratic Mode Chat
1. Mode dropdown: "socratic"
2. Type: "How do I learn Go?"
3. Click "Send"
4. ✅ Your message appears
5. ✅ Response appears with guiding questions
6. Response should ask things like "What interests you about Go?" not give direct answers

#### Test 4.3: Error Handling
1. Stop Ollama (if running)
2. Type: "Hello"
3. Click "Send"
4. ✅ Error message appears: "ollama not running"
5. Start Ollama
6. Click "Send" again
7. ✅ Message is sent and response appears

#### Test 4.4: Clear Messages
1. Have messages in chat
2. Click "Clear" button
3. ✅ All messages disappear
4. ✅ "Start a conversation..." message appears

#### Test 4.5: Enter Key
1. Type message
2. Press Enter (not Shift+Enter)
3. ✅ Message sends automatically
4. Type message
5. Press Shift+Enter
6. ✅ Newline added (doesn't send)

---

### Part 5: API Endpoint Testing (2 minutes)

Open browser console (F12 → Network tab) and test each endpoint:

#### Test 5.1: Status Endpoint
```bash
curl http://127.0.0.1:11436/api/status
```
Expected:
```json
{"status": "running"}
```

#### Test 5.2: First-Run Check
```bash
curl http://127.0.0.1:11436/api/first-run-check
```
Expected:
```json
{
  "first_run_complete": true,
  "ollama_installed": true,
  "ollama_running": true
}
```

#### Test 5.3: Settings Get
```bash
curl http://127.0.0.1:11436/api/settings
```
Expected: Full config JSON with provider, model, mode, etc.

#### Test 5.4: Settings Post
```bash
curl -X POST http://127.0.0.1:11436/api/settings \
  -H "Content-Type: application/json" \
  -d '{"model":"llama2","mode":"socratic"}'
```
Expected:
```json
{"success": true, "config": {...}}
```

#### Test 5.5: Models List
```bash
curl http://127.0.0.1:11436/api/models/list
```
Expected:
```json
{"models": ["mistral", "llama2"], "error": null}
```

#### Test 5.6: Chat
```bash
curl -X POST http://127.0.0.1:11436/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello","model":"mistral","mode":"direct"}'
```
Expected:
```json
{
  "success": true,
  "response": "Hello! How can I help you today?",
  "provider": "ollama"
}
```

---

## Test Scenarios

### Scenario 1: Fresh User (5 minutes)
1. Delete config: `rm ~/.config/moly/config.json`
2. Restart desktop app
3. Open sidebar
4. Complete first-run setup
5. See settings, Ollama, models sections
6. Send a test message

**Expected**: Everything works smoothly

### Scenario 2: Switch Providers (10 minutes)
1. Start with Local
2. Chat and verify works
3. Delete config
4. Restart and choose Claude
5. Enter API key
6. Try to chat (will fail if key invalid, but shouldn't crash)
7. Verify error message shown

**Expected**: Switching providers doesn't break the app

### Scenario 3: Offline Mode (5 minutes)
1. Make sure Ollama is running
2. Chat successfully
3. Stop Ollama
4. Try to chat
5. ✅ Error message appears
6. Start Ollama
7. Chat works again

**Expected**: App handles offline gracefully

### Scenario 4: Heavy Usage (5 minutes)
1. Send 10 messages rapidly
2. Switch modes between each
3. Change models
4. Start/stop Ollama
5. Install/remove models

**Expected**: No crashes, UI stays responsive

---

## Quick Testing Checklist

- [ ] Desktop app starts without errors
- [ ] Sidebar loads from extension click
- [ ] First-run modal appears on first click
- [ ] All three provider options clickable
- [ ] Setup completes and modal closes
- [ ] Settings panel shows dropdowns
- [ ] Model dropdown has 4 options
- [ ] Mode dropdown has 2 options
- [ ] Settings persist after reload
- [ ] Ollama status updates every 3 seconds
- [ ] Start/Stop buttons work
- [ ] Model install button works
- [ ] Model appears in list after install
- [ ] Model remove button works
- [ ] Chat message sends with Enter key
- [ ] Direct mode gives direct answers
- [ ] Socratic mode asks guiding questions
- [ ] Error message shown when Ollama stopped
- [ ] Clear button removes all messages
- [ ] All API endpoints return correct JSON
- [ ] CORS headers present in responses

---

## Debugging Tips

### App Not Starting
```bash
# Check if port is in use
lsof -i :11436

# Check systemd service
systemctl --user status moly
journalctl --user -u moly -f
```

### Sidebar Not Loading
```bash
# Check extension is loaded
chrome://extensions

# Check desktop app is running
curl http://127.0.0.1:11436/api/status

# Check browser console (F12)
# Look for network errors or JavaScript errors
```

### Chat Not Working
```bash
# Check config
cat ~/.config/moly/config.json

# Check Ollama is running
curl http://localhost:11434/api/tags

# Test chat endpoint directly
curl -X POST http://127.0.0.1:11436/api/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"test","model":"mistral","mode":"direct"}'
```

### Models Not Loading
```bash
# Check Ollama is running
ollama ps

# Check API endpoint
curl http://localhost:11434/api/tags

# Manually start Ollama
ollama serve
```

### Settings Not Persisting
```bash
# Check config directory exists
ls -la ~/.config/moly/

# Check permissions
chmod 755 ~/.config/moly
chmod 644 ~/.config/moly/config.json

# Check what's in config
cat ~/.config/moly/config.json | jq .
```

---

## Performance Benchmarks

**Expected Response Times:**
- Desktop app startup: 0.5-1 second
- Sidebar load: <1 second
- Settings update: <100ms
- Ollama status check: <500ms
- Chat response: 5-30 seconds (depends on model)
- Model install: 30 seconds - 10 minutes (depends on model size)

---

## Known Limitations (MVP)

- ❌ No way to change provider after setup (delete config to reset)
- ❌ No way to add API keys after setup
- ❌ Model dropdown is generic (doesn't match provider)
- ❌ No conversation history export
- ❌ No custom LLM prompt templates
- ❌ Works on Ollama port 11434 only (hardcoded)

These are intentional MVP limitations and can be added in v2.

---

## Success Criteria

✅ All features work as described above  
✅ No crashes or errors  
✅ Response times acceptable  
✅ Settings persist correctly  
✅ Error messages are clear  
✅ UI is responsive  

---

**Last Updated**: September 4, 2026  
**Status**: Ready for manual testing
