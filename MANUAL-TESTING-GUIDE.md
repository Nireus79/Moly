# Moly Manual Testing Guide

Step-by-step instructions for testing Moly locally before public release.

## Prerequisites

- Node.js 16+ installed
- Chrome/Chromium browser
- About 10GB free disk space
- Time: ~1-2 hours for complete testing

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
- No errors shown

## Part 3: Test with Claude API Key

### Step 3.1: Get Claude API Key

```
1. Visit: https://console.anthropic.com/keys
2. Create new API key (if needed)
3. Copy the key (starts with sk-ant-)
```

### Step 3.2: Configure Claude in Moly

```
1. Click Moly icon in toolbar
2. Click gear icon (settings)
3. Click "Claude" tab
4. Paste API key
5. Click "Save & Validate"
```

**Expected Result:**
- "Configuration saved successfully"
- Models dropdown shows Claude models
- Claude marked as "Active"

### Step 3.3: Test Suggestion Generation

```
1. Go back to Chat tab
2. Type test message: "Hey, how are you?"
3. Click send (or press Enter)
4. Wait 2-5 seconds

Expected output:
- Message appears in chat
- Moly generates 3-5 suggestions
- Status shows "Using: Claude (Cloud)"
- Each suggestion has Copy button
```

### Step 3.4: Test Modes and Contexts

```
Mode tests:
- Click "Socratic" - suggestions ask guiding questions
- Click "Direct" - suggestions are ready-to-use

Context tests:
- Formal: Suggestions are professional
- Friendly: Suggestions are casual  
- Dating: Suggestions are romantic

Example input: "I really like you"
- Formal: "I appreciate your qualities and would like to know you better"
- Friendly: "You seem really cool and I'd love to hang out"
- Dating: "There's something special about you that I find really attractive"
```

## Part 4: Test with Ollama (Local)

### Step 4.1: Install and Run Ollama

```bash
# Terminal 3: Install Ollama
# Download from: https://ollama.ai/

# Start Ollama
ollama serve

# Expected: "Listening on 127.0.0.1:11434"
```

### Step 4.2: Pull Mistral Model

```bash
# Terminal 4: Pull model
ollama pull mistral

# This downloads ~4GB - may take 5-15 minutes
# Progress shown in terminal
```

### Step 4.3: Auto-Detection Test

```
1. Reload Moly extension: chrome://extensions -> Moly -> reload
2. Open Moly sidebar
3. Should show: "Using: Ollama (Local)"

If not:
- Go to Settings
- Ollama tab should show "Configured"
- Click "Make Active" if needed
```

### Step 4.4: Test Ollama Suggestions

```
1. Type message: "Hi, I like you"
2. Click send
3. Wait 5-10 seconds (first response slower)
4. Suggestions appear
5. Status shows "Using: Ollama (Local)"

Expected:
- Ollama running locally
- No data sent to cloud
- Suggestions based on Mistral 7B model
```

### Step 4.5: Test CORS Proxy

```bash
# Verify proxy is still running
curl http://localhost:11435/api/tags

# Should show proxy response with CORS headers
curl -i http://localhost:11435/api/tags | grep -i "access-control"

# Expected: Access-Control-Allow-Origin: *
```

## Part 5: Test Fallback Scenarios

### Step 5.1: Stop Ollama, Test Claude Fallback

```bash
# Terminal 3: Stop Ollama
# Press Ctrl+C

# Back in Chrome:
1. Type another message
2. Click send
3. Wait a few seconds

Expected:
- Moly detects Ollama is down
- Falls back to Claude automatically
- Status shows "Using: Claude (Cloud - Fallback)"
- Suggestions still appear
```

### Step 5.2: Stop Claude, Test Error Handling

```
1. Remove Claude API key in Settings
2. Type message and send
3. Wait for response

Expected:
- Error message appears
- Clear error: "Please configure a provider"
- Settings link to configure
```

### Step 5.3: Multiple Fallback Test

```
With Ollama down and Claude key valid:
1. Send message
2. Should use Claude
3. Status shows "Claude (Cloud)"

With Ollama running but Claude key missing:
1. Send message
2. Should use Ollama
3. Status shows "Ollama (Local)"
```

## Part 6: Test LM Studio (if available)

### Step 6.1: Install LM Studio

```
1. Download from: https://lmstudio.ai/
2. Install application
3. Launch LM Studio
4. In app: Search for "Mistral 7B" and download model
5. Once loaded, it listens on localhost:8000
```

### Step 6.2: Configure Moly for LM Studio

```
1. Open Moly Settings
2. Click "Ollama (Local)" tab - it's the local models tab
3. Change URL to: http://localhost:8000
4. Click "Save & Validate"
5. Should find model from LM Studio
```

### Step 6.3: Test LM Studio

```
1. Type message in Moly
2. Send
3. Should use LM Studio
4. Status shows "Using: LM Studio (Local)"
```

## Part 7: Test Settings Panel

### Step 7.1: Provider Configuration

```
1. Settings → Claude tab
2. Paste valid API key
3. Models dropdown populates
4. Select model
5. Click "Make Active"
6. Verify Claude is active

Repeat for OpenAI if you have key
```

### Step 7.2: Preferences

```
1. Settings → (default view)
2. Test Chat Mode toggle:
   - Socratic mode
   - Direct mode
3. Test Communication Context:
   - Formal
   - Friendly
   - Dating
4. Each change saves automatically
```

### Step 7.3: Data Management

```
1. Settings → Advanced
2. Click "Clear All Settings"
3. Confirm
4. Verify settings cleared
5. Chrome storage reset
```

## Part 8: Test Edge Cases

### Test 8.1: Very Long Message

```
1. Paste a 2000+ character message
2. Send
3. Should handle without errors
4. Suggestions generated
```

### Test 8.2: Special Characters

```
Messages to test:
- "Hello! 你好 مرحبا"
- "Test with emojis: 😀 ❤️ 🎉"
- "Quotes: \"test\" 'test'"
- "Code: <html></html>"

Expected: All handled correctly
```

### Test 8.3: Rapid Requests

```
1. Send 3 messages in quick succession
2. Each should generate suggestions
3. No crashes or errors
4. Requests queue properly
```

### Test 8.4: Port Conflicts

```bash
# Test with proxy on different port
moly-proxy --port 11440

# Moly should still work if configured for 11440
# Or fallback to direct Ollama at 11434
```

## Part 9: Test Browser Compatibility

### Test 9.1: Chrome Latest

```
- Chrome version 120+ 
- All features work ✓
- Performance good ✓
```

### Test 9.2: Edge Browser

```
- Edge 120+
- Extension loads ✓
- Features work ✓
```

### Test 9.3: Chromium

```
- Chromium latest
- Extension loads ✓
- Features work ✓
```

## Part 10: Performance Testing

### Test 10.1: Suggestion Generation Time

```
Time measurements (from send to suggestions appear):

Ollama (local):
- First request: 8-15 seconds (model loads)
- Subsequent: 3-8 seconds

Claude (cloud):
- 2-5 seconds

OpenAI:
- 1-3 seconds

Record times for baseline
```

### Test 10.2: Memory Usage

```
1. Open DevTools (F12)
2. Go to Memory tab
3. Take heap snapshot before using Moly
4. Use Moly for 10+ messages
5. Take another snapshot
6. Compare memory growth

Expected: < 100MB growth for normal usage
```

### Test 10.3: Storage Usage

```
In DevTools → Application → Local Storage:
1. Check chrome-extension://... storage
2. Should contain: settings, conversations
3. Typical size: < 1MB with 10+ messages
```

## Part 11: Offline Testing (Local Only)

### Test 11.1: Disconnect Internet

```
1. Disconnect from internet
2. Ollama/LM Studio running locally
3. Try generating suggestions
4. Should work completely offline
5. No cloud API calls attempted
```

### Test 11.2: Reconnect Internet

```
1. Reconnect to internet
2. Try Claude/OpenAI suggestion
3. Should work again
```

## Part 12: Error Scenarios

### Test 12.1: Invalid API Keys

```
1. Enter invalid Claude key: "test123"
2. Click Save
3. Error message appears
4. Clear error with X button
5. Try again with valid key
```

### Test 12.2: Network Errors

```
1. Turn off internet
2. Try to use Claude
3. Clear error message appears
4. Message like "Failed to reach Claude API"
5. Fallback works if Ollama available
```

### Test 12.3: Model Not Found

```
1. Configure Ollama with model that's not pulled
2. Try to generate suggestions
3. Clear error message
4. Manual instruction to pull model
```

## Testing Checklist

### Core Functionality
- [ ] Extension loads in Chrome
- [ ] Moly icon appears in toolbar
- [ ] Sidebar opens when clicked
- [ ] Messages input works
- [ ] Suggestions generate correctly

### Providers
- [ ] Claude works with valid key
- [ ] OpenAI works with valid key
- [ ] Ollama works when running locally
- [ ] LM Studio works when running
- [ ] Auto-detection works
- [ ] Manual configuration works

### Features
- [ ] Chat history displays
- [ ] Mode switching (Socratic/Direct)
- [ ] Context switching (Formal/Friendly/Dating)
- [ ] Copy suggestions to clipboard
- [ ] Settings save correctly
- [ ] Clear all settings works

### Fallback
- [ ] Falls back when primary fails
- [ ] Shows which provider is active
- [ ] Error messages clear and helpful
- [ ] Multiple fallbacks work (local → cloud)

### Performance
- [ ] Suggestions generate in reasonable time
- [ ] No freezing or lag
- [ ] Memory usage reasonable
- [ ] Storage usage reasonable

### Edge Cases
- [ ] Long messages handled
- [ ] Special characters handled
- [ ] Rapid requests handled
- [ ] Port conflicts handled
- [ ] Offline mode works
- [ ] Network errors handled

### Documentation
- [ ] QUICKSTART is clear
- [ ] TROUBLESHOOTING is helpful
- [ ] Settings are self-explanatory
- [ ] Error messages guide users

## Reporting Issues

Format for bug reports:

```
Title: [Brief description]

Environment:
- OS: Windows/macOS/Linux version
- Browser: Chrome/Edge version
- Node.js: version
- Ollama: version (if used)
- Moly: commit hash

Steps to Reproduce:
1. ...
2. ...
3. ...

Expected Result:
- ...

Actual Result:
- ...

Screenshots/Logs:
- [Attach browser console output]
- [Attach any error messages]
```

## Sign-Off

Once all tests pass:

```
Manual Testing Complete
Date: YYYY-MM-DD
Tester: Name

Tests Passed: All
Known Issues: [List any]
Recommendations: [List any]

Ready for: [ ] Beta [ ] Public Release
```

---

**Total Testing Time:** 1-2 hours
**Browser:** Chrome recommended
**Support:** GitHub Issues for any findings
