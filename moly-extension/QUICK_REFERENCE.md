# Moly Quick Reference - Testing Cheat Sheet

## 1-Minute Setup

```bash
npm run build
# Load dist/ folder in chrome://extensions (Developer mode)
# Keep DevTools Console open (F12)
```

---

## Console Commands Reference

### Check Extension Status
```javascript
// Is message detection running?
chrome.runtime.sendMessage({type: 'GET_DETECTION_STATUS'}, console.log)
// Output: {isRunning: true, processedCount: 5}
```

### View All Settings
```javascript
chrome.storage.local.get('settings', console.log)
// Output: {settings: {activeProvider: 'claude', ...}}
```

### View Last Detected Message
```javascript
chrome.storage.local.get('lastDetectedMessage', console.log)
// Output: {lastDetectedMessage: {sender: 'Alice', text: '...'}}
```

### View All Contacts
```javascript
chrome.storage.local.get('contacts', console.log)
// Output: {contacts: [{id: '...', name: 'Alice', ...}]}
```

### View Everything in Storage
```javascript
chrome.storage.local.get(null, (items) => console.table(items))
// Shows all stored data in table format
```

### Clear All Data (Start Fresh)
```javascript
chrome.storage.local.clear(() => console.log('Done'))
```

### Clear Just Settings
```javascript
chrome.storage.local.remove('settings', () => console.log('Settings cleared'))
```

---

## What Console Should Show

### When You Open a Messaging Site
```
Moly content script loaded
Initializing message detection for: [platform]
Starting message detection for platform: [platform]
Using X platform-specific message selectors
```

### When a Message Arrives
```
Message detected: [Sender Name] - [preview]
Sidebar received detected message: [Sender Name]
```

### When Validation Runs
```
Validated and discovering models...
Found 3 Claude models
```

---

## Quick Tests (Copy/Paste)

### Test 1: Is Content Script Loaded?
**Console**:
```javascript
console.log('Test: Page should load content script within 1-2 seconds')
```
**Expected**: "Moly content script loaded" appears in console

**If not**: Extension not loaded, refresh page, check if installed

---

### Test 2: Is Platform Detected?
**Console**:
```javascript
chrome.runtime.sendMessage({type: 'GET_PLATFORM'}, console.log)
```
**Expected**: `{platform: 'tinder'}` or `{platform: 'facebook'}` etc.

**If showing 'unknown'**: This platform may not have selectors yet

---

### Test 3: Can Provider Validate?
**Console**:
```javascript
chrome.storage.local.get('settings', (result) => {
  const settings = result.settings;
  if (settings?.providers?.claude?.apiKey) {
    console.log('Claude API key found:', settings.providers.claude.apiKey.slice(0,5) + '...')
  } else {
    console.log('No Claude API key configured')
  }
})
```
**Expected**: Shows API key prefix if configured

---

### Test 4: Message Detection Working?
**How to Test**:
1. Open DevTools Console
2. Send yourself a test message
3. Wait 2 seconds
4. Look for "Message detected:" in console

**Expected**: See "Message detected: [Your Name] - [Message]"

**If not appearing**: Check if message is "incoming" (from other person, not you)

---

## Debugging Checklist

When something doesn't work:

1. **Open DevTools Console** (F12)
   - Look for red errors
   - Look for warning messages
   - Note exact error text

2. **Check Extension Status**
   ```javascript
   chrome.runtime.sendMessage({type: 'GET_DETECTION_STATUS'}, console.log)
   ```
   - Should show `{isRunning: true}`

3. **Check Settings Saved**
   ```javascript
   chrome.storage.local.get('settings', console.log)
   ```
   - Should show your configuration
   - If empty: settings not saved

4. **Check If Message Detected**
   ```javascript
   chrome.storage.local.get('lastDetectedMessage', console.log)
   ```
   - Should show most recent message
   - If null: message never detected

5. **Check Platform Recognition**
   - Navigate to known platform
   - Console should show platform name
   - If "unknown": needs platform support

---

## Most Common Issues & Fixes

| Problem | Check | Fix |
|---------|-------|-----|
| Extension doesn't load | chrome://extensions for errors | Rebuild: `npm run build` |
| No "content script loaded" message | Console when page loads | Extension not active, refresh |
| Message never detected | "Message detected:" in console | Try sending message to yourself |
| Message doesn't appear in Sidebar | Check lastDetectedMessage in storage | Close and reopen Sidebar |
| Settings won't save | Check console for validation error | Verify API key is correct |
| Model dropdown empty | Check validation message | Try revalidating provider |
| Sidebar won't open | Try Settings first | Update Chrome to 114+ |
| Suggestions not working | Check activeProvider in settings | Configure provider in Settings |

---

## Platform Selector Reference

If a platform isn't detecting, these are the selectors being used:

### Tinder
```
Message: [data-qa="messageItem"], .Bubble__bubble, [role="article"]
Sender: [data-qa="messageSender"], .Bubble__senderName
```

### Bumble
```
Message: [data-testid="message-item"], .message-container
Sender: [data-testid="message-sender"], .message-sender-name
```

### Facebook
```
Message: [data-qa="message_container"], .msg, [role="article"]
Sender: [data-qa="message_sender"], .message__author
```

### Discord
```
Message: [data-testid="message"], .messageContent-2qWWxC
Sender: [data-testid="message-author-username"], .username-2d0OO5
```

### LinkedIn
```
Message: [data-qa="message-item"], .msg-s-message-list__item
Sender: [data-qa="message-sender"], .msg-s-message-list__item-text__sender
```

### WhatsApp
```
Message: [data-testid="message"], .message-in
Sender: [data-testid="message-sender"], .message-meta
```

### Telegram
```
Message: [data-mid], .message
Sender: .from_name, .message-author
```

---

## Expected Timings

| Action | Time |
|--------|------|
| Content script load | 0.5-2 sec after page load |
| Message detection | 0.2-2 sec after message arrives |
| Sidebar message receive | 0.1-1 sec after detection |
| Settings validation | 5-15 sec (depends on API) |
| Model discovery | 5-20 sec (depends on API) |
| Suggestion generation | 3-10 sec (depends on model) |

---

## API Key Testing Values

To test without real API keys:

### Claude
- Valid format: `sk-ant-...` (long string)
- Invalid format: `test` (won't validate)
- Test with: Your actual API key from console.anthropic.com

### OpenAI
- Valid format: `sk-...` (starts with sk-)
- Invalid format: `openai-test`
- Test with: Your actual API key from platform.openai.com

### Ollama
- Valid: `http://localhost:11434` (Ollama must be running)
- Invalid: `http://localhost:9999` (port doesn't exist)
- Test: Must have Ollama installed locally

---

## Expected File Sizes (After Build)

```
background.js        3.70 kB ✓ (small)
popup.js            8.53 kB ✓ (small)
content.js         14.93 kB ✓ (ok)
sidebar.js         16.04 kB ✓ (ok)
settings.js        19.13 kB ✓ (ok)
providerManager    19.90 kB ✓ (ok)
settingsStore    1,253 kB   ⚠️ (large - needs Phase 2 optimization)
```

If settingsStore is much larger, there might be unused dependencies.

---

## Testing Matrix

Quick way to track what you've tested:

```
Platform Tests:
- [ ] Tinder
- [ ] Bumble
- [ ] Hinge
- [ ] Match
- [ ] OkCupid
- [ ] FetLife
- [ ] Facebook
- [ ] LinkedIn
- [ ] Discord
- [ ] Slack
- [ ] Twitter/X
- [ ] WhatsApp
- [ ] Telegram

Feature Tests:
- [ ] Message detection works
- [ ] Settings persist
- [ ] Validation works
- [ ] Provider switching works
- [ ] Model discovery works
- [ ] Socratic mode works
- [ ] Direct mode works
- [ ] All contexts work
- [ ] Copy works
- [ ] Error messages work

Edge Cases:
- [ ] Multiple messages
- [ ] Long messages
- [ ] Special characters
- [ ] Page refresh
- [ ] Sidebar open/close
- [ ] Provider switching
- [ ] Tab switching
- [ ] Fast API key changes
```

---

## When to Call It Done

### Minimum (MVP Testing)
✓ Extension loads
✓ Message detected on 1 platform
✓ Settings save and persist
✓ No critical console errors

### Standard (Confident)
✓ All MVP checks pass
✓ Message detected on 3+ platforms
✓ Both chat modes work
✓ All contexts work
✓ Error messages clear

### Thorough (Production Ready)
✓ All Standard checks pass
✓ All edge cases tested
✓ Performance acceptable
✓ No warnings in console
✓ All 13 platforms tested

---

## If You Get Stuck

### Step 1: Clear Data
```javascript
chrome.storage.local.clear()
```
Then refresh page and start fresh.

### Step 2: Check Logs
- Look at DevTools Console for red errors
- Copy exact error message
- Search error in code

### Step 3: Verify Setup
- Chrome version 114+? (check chrome://version)
- Extension actually installed? (check chrome://extensions)
- API key actually valid? (test in provider's web interface)
- Messaging site actually has messages? (send yourself one)

### Step 4: Try Different Platform
- If one platform fails, try another
- All platforms may not work perfectly
- That's okay for Phase 1

### Step 5: Check Known Issues
- See TESTING_EXPECTATIONS.md for common issues
- See MANUAL_TESTING_GUIDE.md for detailed troubleshooting

---

## Quick Restart Process

If extension gets stuck:

1. Go to `chrome://extensions`
2. Turn off Moly extension
3. Refresh page where you're testing
4. Turn on Moly extension
5. Refresh page again
6. Should work

Or:

1. `chrome.storage.local.clear()`
2. Refresh page
3. Reconfigure in Settings

---

**Need more detail? See MANUAL_TESTING_GUIDE.md**

**Want to know what changed? See TESTING_EXPECTATIONS.md**
