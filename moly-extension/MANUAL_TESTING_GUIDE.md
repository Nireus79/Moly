# Moly Manual Testing Guide

## Pre-Testing Setup

### 1. Build the Extension
```bash
npm run build
```
**Expected**: `dist/` folder created with all compiled files
**Files to check**:
- dist/background.js
- dist/popup.js
- dist/sidebar.js
- dist/settings.js
- dist/content.js
- dist/manifest.json

### 2. Load in Chrome
1. Open `chrome://extensions/`
2. Enable "Developer mode" (top right)
3. Click "Load unpacked"
4. Select `/path/to/moly-extension/dist/` folder
5. Extension should appear with Moly icon

**Expected**: 
- Extension loads without errors
- Icon appears in Chrome toolbar
- No red warning badge

### 3. Open DevTools
- Right-click extension icon → "Inspect popup" (keep this open)
- Open any messaging website and open DevTools
- Go to "Console" tab to watch for errors

---

## Core Feature Tests

### Test 1: Message Detection

**Setup**:
1. Navigate to a dating/messaging app (Tinder, Bumble, Facebook, Discord, etc.)
2. Open DevTools Console (F12)
3. Look for log: "Moly content script loaded"
4. Look for log: "Starting message detection for platform: [platform-name]"

**Expected Console Output**:
```
Moly content script loaded
Initializing message detection for: tinder
Starting message detection for platform: tinder
Using 4 platform-specific message selectors
```

**Test Actions**:
1. Open a conversation with a message
2. Wait a few seconds
3. Look for console log: "Message detected: [sender-name]"
4. Check that the detected message appears in Sidebar

**Pass Criteria**:
- ✓ Platform correctly detected
- ✓ Message detected within 5 seconds
- ✓ Console shows no errors
- ✓ Sidebar shows detected message

**Fail Indicators**:
- ✗ "Unknown platform" message
- ✗ "Message detected" never appears in console
- ✗ Console shows errors
- ✗ Sidebar stays empty

---

### Test 2: Sidebar Message Display

**Setup**:
1. Open Sidebar (click Moly icon → "Open Chat" or Cmd+Shift+M)
2. Navigate to messaging app with open conversation

**Expected**:
- Sidebar opens on right side (should be resizable)
- Sidebar width is between 300px-500px
- No layout breaks

**Test Actions**:
1. Detect a message (from Test 1)
2. Check if message appears in Sidebar
3. Should see sender name and message preview

**Expected Display**:
```
Message from [Sender Name]
[Message text preview]
```

**Pass Criteria**:
- ✓ Sidebar opens properly
- ✓ Detected message appears
- ✓ Can resize sidebar by dragging edge
- ✓ No console errors

**Fail Indicators**:
- ✗ Sidebar doesn't open
- ✗ Sidebar fixed width (not resizable)
- ✗ Message doesn't appear
- ✗ Layout broken

---

### Test 3: Settings Configuration

**Setup**:
1. Click Moly icon → "Settings"
2. Settings page should open in new tab

**Expected Screen**:
- "Moly Settings" header
- Provider selection (Claude, OpenAI, Ollama tabs)
- API Key input field
- Model dropdown (empty initially)
- Save & Validate button

**Test Actions**:

#### 3a. Claude Provider
1. Click "Claude" tab
2. Enter valid Claude API key from console.anthropic.com
3. Click "Save & Validate"
4. Wait for validation to complete

**Expected**:
- Message: "Validated. Found X models" or "Provider configured successfully"
- Model dropdown should populate with Claude models
- Settings saved to chrome.storage

**Verification**:
- Open DevTools Console
- Run: `chrome.storage.local.get('settings', console.log)`
- Should see settings object with activeProvider: 'claude'

#### 3b. OpenAI Provider
1. Click "OpenAI" tab
2. Enter valid OpenAI API key
3. Click "Save & Validate"

**Expected**: Similar to Claude, finds OpenAI models

#### 3c. Ollama (Local)
1. Click "Ollama" tab
2. Enter base URL (usually http://localhost:11434)
3. Ollama must be running locally
4. Click "Save & Validate"

**Expected**: Validates against local Ollama instance

**Pass Criteria**:
- ✓ Provider switches work
- ✓ Validation runs and provides feedback
- ✓ Settings persist (page refresh → data still there)
- ✓ Model dropdown populates
- ✓ No console errors

**Fail Indicators**:
- ✗ "Error: Provider validation failed"
- ✗ Model dropdown stays empty
- ✗ Settings don't persist after refresh
- ✗ Console errors on validation

---

### Test 4: Settings Persistence

**Setup**: After configuring provider (Test 3)

**Test Actions**:
1. Set provider to "Claude"
2. Enter API key
3. Select chat mode: "Socratic"
4. Select communication context: "Dating"
5. Click "Save & Validate"
6. Close the Settings tab
7. Reopen Settings (click Moly icon → "Settings")

**Expected**:
- Provider still set to Claude
- API key still masked (showing "sk-...xyz")
- Chat mode: "Socratic"
- Communication context: "Dating"

**Pass Criteria**:
- ✓ All settings persist after page reload
- ✓ Can update model without re-entering API key
- ✓ Settings show in both tabs

**Fail Indicators**:
- ✗ Settings reset to defaults
- ✗ Can't update without re-entering API key
- ✗ Inconsistent state between tabs

---

### Test 5: Chat Mode Switching

**Setup**:
1. Detect a message (Test 1)
2. Open Sidebar
3. Sidebar header should show "Socratic" and "Direct" buttons

**Test Actions**:
1. Click "Socratic" button
2. Send a test message or request suggestion
3. Should get guiding questions
4. Click "Direct" button
5. Send test message
6. Should get ready-to-use suggestions

**Expected Output for Socratic**:
```
Q1: What are you trying to achieve?
Q2: How do you want them to feel?
...
```

**Expected Output for Direct**:
```
Option 1: [Message suggestion]
Option 2: [Message suggestion]
Option 3: [Message suggestion]
```

**Pass Criteria**:
- ✓ Mode switching works
- ✓ Correct output format for each mode
- ✓ Can switch between modes
- ✓ No API errors

**Fail Indicators**:
- ✗ Mode buttons don't respond
- ✗ Wrong suggestions appear
- ✗ Console shows API errors

---

### Test 6: Communication Context

**Setup**: Sidebar open with detected message

**Test Actions**:
1. Look for context selector (Formal / Friendly / Dating)
2. Click each one
3. Send a test message in each context
4. Observe tone changes in suggestions

**Expected**: Suggestions should change tone based on context
- Formal: Professional, respectful
- Friendly: Warm, approachable
- Dating: Genuine, interested

**Pass Criteria**:
- ✓ Context selector visible
- ✓ Can click between contexts
- ✓ Suggestions change tone appropriately
- ✓ Changes apply immediately

**Fail Indicators**:
- ✗ Context selector missing
- ✗ Clicking does nothing
- ✗ Suggestions identical across contexts

---

### Test 7: Copy Suggestions

**Setup**: Generate suggestions (Test 5)

**Test Actions**:
1. Click "Copy" button on any suggestion
2. Paste in text editor or chat box
3. Verify exact text copied

**Expected**: Full suggestion text copied to clipboard

**Pass Criteria**:
- ✓ Copy button works
- ✓ Correct text pasted
- ✓ No extra characters

**Fail Indicators**:
- ✗ Nothing copied
- ✗ Wrong text copied
- ✗ Extra/missing characters

---

### Test 8: Error Handling

**Test Actions**:

#### 8a. Invalid API Key
1. Settings → Claude
2. Enter invalid API key (e.g., "test")
3. Click Save & Validate
4. Expected: Error message: "Error: Provider validation failed"

#### 8b. Missing Configuration
1. Clear all settings
2. Go to Sidebar
3. Try to get suggestions without configuring provider
4. Expected: Error message explaining need to configure

#### 8c. Button Failures
1. Try to open Settings when service fails
2. Expected: Alert showing error message

**Pass Criteria**:
- ✓ Errors show user-friendly messages
- ✓ No silent failures
- ✓ Console shows detailed errors for debugging
- ✓ App recovers after error

**Fail Indicators**:
- ✗ Silent failures (nothing happens)
- ✗ Cryptic error messages
- ✗ App becomes unresponsive

---

## Platform-Specific Tests

### Tinder
**Platform detection**: Should see "Starting message detection for platform: tinder"
**Message detection**: Look in chat list
**Expected selectors**: `[data-qa="messageItem"]`, `.Bubble__bubble`, `[role="article"]`

### Bumble
**Platform detection**: "bumble"
**Message detection**: Open conversation
**Expected selectors**: `[data-testid="message-item"]`, `.message-container`

### Facebook Messenger
**Platform detection**: "facebook"
**Message detection**: Open a conversation
**Expected selectors**: `[data-qa="message_container"]`, `.msg`

### Discord
**Platform detection**: "discord"
**Message detection**: Open a DM or channel
**Expected selectors**: `[data-testid="message"]`, `.messageContent-2qWWxC`

### LinkedIn
**Platform detection**: "linkedin"
**Message detection**: Open messaging
**Expected selectors**: `[data-qa="message-item"]`, `.msg-s-message-list__item`

### WhatsApp Web
**Platform detection**: "whatsapp"
**Message detection**: Open a chat
**Expected selectors**: `[data-testid="message"]`, `.message-in`

**For each platform**:
1. Open DevTools Console
2. Send yourself a test message
3. Look for console log: "Message detected: [sender]"
4. Verify sidebar shows the message

---

## Edge Cases to Test

### 1. Multiple Messages
**Test**: Send 3-5 messages rapidly
**Expected**: All detected, sidebar updates for each

### 2. Very Long Messages
**Test**: Send message with 500+ characters
**Expected**: Truncated properly, no UI break

### 3. Messages with Emojis
**Test**: Send message with emojis
**Expected**: Displayed correctly, no encoding issues

### 4. Special Characters
**Test**: Send message with quotes, apostrophes, etc.
**Expected**: Displayed correctly, no escaping issues

### 5. Refresh Page
**Test**: Detect message → Refresh page → Check sidebar
**Expected**: Message still shows (loaded from storage)

### 6. Close and Reopen Sidebar
**Test**: Detect message → Close sidebar → Reopen
**Expected**: Message still there

### 7. Switch Tabs
**Test**: Detect message → Switch to different tab → Back
**Expected**: Message persists or reloads properly

### 8. Fast API Key Changes
**Test**: Change provider API keys rapidly
**Expected**: Latest key used, no race conditions

---

## Console Output Reference

### What You Should See (Good Signs)
```
Moly content script loaded
Initializing message detection for: tinder
Starting message detection for platform: tinder
Using 4 platform-specific message selectors
Message detected: [Sender Name] - [Message preview]
Sidebar received detected message: [Sender Name]
```

### What Indicates Problems (Bad Signs)
```
ERROR: Failed to configure provider
Uncaught TypeError: Cannot read property 'querySelector'
Provider validation failed: 401 Unauthorized
No settings found. Please configure a provider
Message detection: No messages found (repeated)
```

---

## Debugging Commands

Run these in DevTools Console:

### Check if extension loaded
```javascript
chrome.runtime.sendMessage({type: 'GET_DETECTION_STATUS'}, console.log)
```

### View stored settings
```javascript
chrome.storage.local.get('settings', console.log)
```

### View stored messages
```javascript
chrome.storage.local.get('lastDetectedMessage', console.log)
```

### View all storage
```javascript
chrome.storage.local.get(null, (items) => console.table(items))
```

### Check detected contacts
```javascript
chrome.storage.local.get('contacts', console.log)
```

### Clear all data (Start fresh)
```javascript
chrome.storage.local.clear(() => console.log('Storage cleared'))
```

---

## What Not to Expect (Yet)

These are Phase 2 features - they don't exist yet:
- ✗ Encrypted storage for API keys
- ✗ Tone detection for incoming messages
- ✗ Message templates
- ✗ Message scheduling
- ✗ Analytics dashboard
- ✗ Premium subscription features
- ✗ Support for Gemini, Mistral, LLaMA

---

## Testing Order (Recommended)

1. **Setup** (5 min)
   - Build extension
   - Load in Chrome
   - Open DevTools

2. **Core Detection** (15 min)
   - Test on 1-2 platforms
   - Verify message detection works
   - Check console for expected logs

3. **Sidebar** (10 min)
   - Verify sidebar opens
   - Check detected messages appear
   - Test resizing

4. **Settings** (20 min)
   - Configure at least one provider
   - Validate settings persist
   - Test model discovery

5. **Features** (20 min)
   - Test both chat modes
   - Test all contexts
   - Test copy functionality

6. **Error Handling** (10 min)
   - Try invalid API key
   - Try to use without configuration
   - Check error messages

7. **Edge Cases** (15 min)
   - Multiple messages
   - Long messages
   - Page refresh
   - Sidebar open/close

**Total Time**: ~90 minutes for thorough testing

---

## Success Criteria

### Must Pass (Blocking Issues)
- [ ] Extension loads in Chrome without errors
- [ ] Message detected on at least one platform
- [ ] Detected message appears in Sidebar
- [ ] Settings save and persist
- [ ] Provider validation works
- [ ] Both chat modes work
- [ ] No console errors during normal use

### Should Pass (Important)
- [ ] All platforms tested
- [ ] Settings available on page reload
- [ ] Can switch between providers
- [ ] Can copy suggestions
- [ ] Error messages are user-friendly
- [ ] Sidebar resizable

### Nice to Pass (Polish)
- [ ] All 8 edge cases work
- [ ] Animations smooth
- [ ] No performance issues
- [ ] Professional appearance

---

## Known Limitations

These are expected and by design:
- Settings store is large (1.2MB uncompressed) - Phase 2 will optimize
- Message detection uses generic + platform-specific selectors - may miss some UI variations
- Model discovery rate-limited by API - may take 5-10 seconds
- No encryption yet - API keys stored in plaintext (Phase 2)
- No analytics - coming in Phase 2

---

## Before You Start

**Checklist**:
- [ ] Read this entire guide
- [ ] Have at least one API key ready (Claude recommended)
- [ ] Have access to a messaging platform for testing
- [ ] Understand expected behavior
- [ ] Know how to open DevTools (F12)
- [ ] Know how to check chrome.storage
- [ ] Understand what to look for in console

**Questions to answer before testing**:
1. Which provider will you test with? (Claude/OpenAI/Ollama)
2. Which platforms will you test? (Tinder/Facebook/Discord/etc.)
3. How much time do you have? (30 min quick / 90 min thorough)
4. Are you checking for bugs or validating features?

---

**Ready to test? Start with the Pre-Testing Setup section above.**

**Found an issue? Note:**
1. What you did
2. What you expected
3. What actually happened
4. Console error message (if any)
5. Which platform/provider
