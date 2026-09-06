# Quick Start: Testing Moly sidePanel

**Status:** ✅ Ready to test - All Priority issues fixed  
**Build:** Complete and tested  
**Time to first test:** 2 minutes

---

## 1-2-3 Setup

### Step 1: Start Backend (30 seconds)
```bash
# In terminal 1
MOLY_PROXY_PATH=~/vs_projects/Moly/Moly/moly-proxy/bin/moly-proxy.js ~/.local/bin/moly &
# Wait for: "Backend listening on http://127.0.0.1:7357"
```

### Step 2: Load Extension (30 seconds)
```bash
# Extension already built in dist/
# In Chrome:
1. Go to chrome://extensions/
2. Enable "Developer mode" (top right)
3. Click "Load unpacked"
4. Select moly-extension/dist/
5. Extension appears with Moly icon
```

### Step 3: Test Open/Close (30 seconds)
```
1. Click Moly extension icon
   → sidePanel opens on right side
2. Click ✕ close button
   → sidePanel closes
3. Click icon again
   → sidePanel reopens
✓ PASS if sidebar persists across navigation
```

---

## 5-Minute Smoke Test

```
1. Click extension icon → sidePanel opens
   EXPECT: Right sidebar appears

2. Wait 3 seconds → Backend status shows
   EXPECT: Green ✓ "Backend connected"
   
3. Click ⚙️ Settings
   EXPECT: Settings page loads, not blank

4. Click Provider Tab (Claude/OpenAI/Ollama)
   EXPECT: Tab highlights, config changes
   
5. Click Model dropdown
   EXPECT: Dropdown responds to clicks
   
6. Click ← Back button
   EXPECT: Returns to chat view

✓ ALL PASS = Extension is working!
```

---

## Testing Each Provider

### Claude (Cloud)
```
1. Go to Settings → Claude tab
2. Get API key from: https://console.anthropic.com/keys
3. Paste into "Claude API Key" field
4. Click "Save & Validate"
   EXPECT: Green message "Provider configured and activated"
5. Type test message: "Hello"
6. Click send
   EXPECT: Suggestions appear after ~2-5 seconds
```

### OpenAI (Cloud)
```
1. Go to Settings → OpenAI tab
2. Get API key from: https://platform.openai.com/keys
3. Paste into "OpenAI API Key" field
4. Click "Save & Validate"
   EXPECT: Green message "Provider configured and activated"
5. Type test message: "Hello"
6. Click send
   EXPECT: Suggestions appear after ~1-3 seconds
```

### Ollama (Local)
```
Prerequisites:
1. Install Ollama from https://ollama.ai
2. Run: ollama serve
3. Pull model: ollama pull llama2
4. Check: curl http://127.0.0.1:11434/api/tags

Then:
1. Go to Settings → Ollama tab
2. Enter Base URL: http://127.0.0.1:11434
3. Wait for "Found X model(s)" message
4. Select model from dropdown
5. Click "Save & Validate"
6. Type test message: "Hello"
7. Click send
   EXPECT: Suggestions appear (may take 5-30 seconds depending on model)
```

---

## What Should Work

| Feature | Status |
|---------|--------|
| Open/Close sidePanel | ✅ Working |
| Settings configuration | ✅ Working |
| Model dropdown | ✅ Working |
| Suggestion generation | ✅ Working |
| Error messages | ✅ Working |
| Backend detection | ✅ Working |
| Provider switching | ✅ Working |
| Message persistence | ✅ Working |

---

## If Something Breaks

### No suggestions appearing?
```
1. Open DevTools (F12)
2. Check Console tab for errors
3. Look for [Sidebar] or [Background] logs
4. Check if provider is configured (Settings view)
5. Check backend status (should be green ✓)
```

### Settings buttons don't respond?
```
This was Priority 1 fix - should be working now!
If not:
1. Reload extension: Go to chrome://extensions/ and click refresh
2. Clear settings: DevTools Console: chrome.storage.local.clear()
3. Reload tab
```

### Suggestion error "Cannot read properties"?
```
This was Priority 2 fix - should be working now!
If error appears:
1. Check API key is valid
2. Check provider is configured
3. Check backend status
4. Look for specific error message in Settings
```

### Backend says "not running"?
```
1. Make sure backend process is running:
   MOLY_PROXY_PATH=... ~/.local/bin/moly &
2. Check it's on http://127.0.0.1:7357
3. Test with: curl http://127.0.0.1:7357/api/status
4. Should see: {"status":"running"}
```

---

## Issue Reporting

If you find issues, note:
1. **What happened** (screenshot if possible)
2. **Expected behavior**
3. **Steps to reproduce**
4. **Console errors** (F12 → Console tab)
5. **Chrome version** (chrome://version)
6. **Which provider** was being used

Then create issue on GitHub or email details.

---

## Files to Know

| File | Purpose |
|------|---------|
| `TESTING_CHECKLIST.md` | Full 50+ test cases |
| `SIDEPANEL_KNOWN_ISSUES.md` | Issue tracking |
| `SESSION_SUMMARY.md` | Detailed session notes |
| `moly-extension/dist/` | Built extension |
| `moly-extension/src/background.ts` | Message handler |
| `moly-extension/src/sidebar/Sidebar.tsx` | Chat interface |
| `moly-extension/src/settings/Settings.tsx` | Config UI |

---

## Quick Links

- **PR:** https://github.com/Nireus79/Moly/pull/1
- **Branch:** `feature/sidepanel`
- **Claude API:** https://console.anthropic.com/keys
- **OpenAI API:** https://platform.openai.com/keys
- **Ollama:** https://ollama.ai

---

## Next Steps

1. **Run this quick test** (5 minutes)
2. **Run full TESTING_CHECKLIST.md** (30-60 minutes)
3. **Report any issues** found
4. **Merge to master** when ready
5. **Prepare for Chrome Web Store**

---

**Last Updated:** September 6, 2026  
**Status:** ✅ Ready for QA testing
