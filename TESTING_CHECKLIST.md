# Moly sidePanel - Testing Checklist

**Last Updated:** September 6, 2026  
**Status:** Ready for full end-to-end testing  
**Build:** npm run build complete, extension ready to load

## Prerequisites

1. Backend running: `MOLY_PROXY_PATH=~/vs_projects/Moly/Moly/moly-proxy/bin/moly-proxy.js ~/.local/bin/moly &`
2. Extension built: `npm run build` (done ✓)
3. Extension loaded in Chrome: 
   - Go to `chrome://extensions/`
   - Enable Developer mode
   - Click "Load unpacked"
   - Select `moly-extension/dist/`

## Core Functionality Tests

### 1. Sidebar Open/Close
- [ ] Click extension icon → sidePanel opens
- [ ] Click ✕ close button → sidePanel closes
- [ ] Click extension icon again → sidePanel opens
- [ ] sidePanel persists across page navigation
- [ ] sidePanel persists across page refresh

### 2. Backend Detection
- [ ] Backend healthy status shows (green ✓)
- [ ] "Backend connected" message appears
- [ ] Status updates every 3 seconds
- [ ] Refresh button updates status
- [ ] If backend terminates: shows development mode message with copy button

### 3. Settings Configuration (FIXED)
- [ ] Click ⚙️ Settings button
- [ ] Settings view loads completely
- [ ] Provider tabs (Claude, OpenAI, Ollama) are clickable
- [ ] Switching providers works smoothly
- [ ] Model dropdown is responsive
- [ ] Can select models from dropdown
- [ ] API key field accepts input
- [ ] Base URL field accepts input (for Ollama)
- [ ] Save & Validate button works
- [ ] ← Back button returns to chat view
- [ ] Settings changes persist after close/reopen

### 4. Provider Configuration

#### Claude
- [ ] Enter valid Claude API key
- [ ] Model dropdown populates with models
- [ ] Can select a model
- [ ] Click "Save & Validate"
- [ ] Message shows "Provider configured and activated"

#### OpenAI
- [ ] Enter valid OpenAI API key
- [ ] Model dropdown populates with models
- [ ] Can select a model
- [ ] Click "Save & Validate"
- [ ] Message shows "Provider configured and activated"

#### Ollama
- [ ] Ollama running locally (check with: curl http://127.0.0.1:11434/api/tags)
- [ ] Enter Ollama base URL
- [ ] Wait for model discovery
- [ ] Message shows "Found X model(s)"
- [ ] Can select a model
- [ ] Click "Save & Validate"
- [ ] Message shows "Provider configured and activated"

### 5. Suggestion Generation (FIXED)
- [ ] Enter a message in chat input
- [ ] Click send
- [ ] Sidebar shows "⏳ Generating suggestions..."
- [ ] Suggestions appear after LLM responds
- [ ] Each suggestion has text and copy button
- [ ] No error "Cannot read properties of undefined"
- [ ] Error handling works: bad API key shows clear error message
- [ ] Error handling works: no provider configured shows clear error message
- [ ] Fallback to OpenAI works if Claude fails

### 6. Chat Workflow
- [ ] Type message in input field
- [ ] Press Enter or click send
- [ ] Message appears in chat history
- [ ] Moly responds
- [ ] Suggestions appear
- [ ] Can copy suggestion to clipboard
- [ ] Multiple messages in conversation work
- [ ] Chat history persists after close/reopen

### 7. Error Handling
- [ ] Invalid API key → clear error message
- [ ] No provider configured → helpful error
- [ ] Backend connection refused → shows development mode help
- [ ] Network error → appropriate error message
- [ ] Empty message → no crash, appropriate feedback

### 8. UI Polish (if testing)
- [ ] No console errors in DevTools
- [ ] No memory leaks (check task manager)
- [ ] Smooth animations
- [ ] Loading states display correctly
- [ ] Buttons have hover feedback
- [ ] Color scheme is consistent
- [ ] Text is readable
- [ ] No overlapping UI elements

## Quick Start Testing (5 minutes)
1. Load extension
2. Click icon → sidePanel opens ✓
3. Wait for backend status ✓
4. Click ⚙️ → Settings opens ✓
5. Configure Claude with API key
6. Type message and send
7. Verify suggestions appear

## Provider Verification

### Working Providers
- [x] Claude 3.5 Sonnet (requires API key)
- [x] OpenAI GPT-4/3.5 (requires API key)
- [x] Ollama (requires local installation)

### Fallback Logic
When primary provider fails, system should try:
1. Claude (if not primary) → Check if configured
2. OpenAI (if not Claude) → Check if configured
3. Show error if all fail

## Known Constraints
- sidePanel only works in Chrome 114+
- Requires 300px minimum width (adjustable)
- Suggestions take time with local models (expected)
- Backend must be running for offline/local features

## Debugging Help

### Check Console
Open DevTools (F12) while sidePanel is active:
- Check for errors in Console tab
- Look for "[Background]" or "[Sidebar]" logs
- Search for specific errors

### Check Background
Chrome DevTools → Extensions → Details → "Service Worker" link:
- View background worker logs
- Check for message handling errors

### Reset Settings
In DevTools Console:
```javascript
chrome.storage.local.clear()
location.reload()
```

## Test Report Template

**Tester:** [Your Name]  
**Date:** [Date]  
**Chrome Version:** [version]  
**Build Version:** [commit hash]  

### Issues Found
- [ ] Issue 1: [Description]
  - Steps to reproduce: 
  - Expected: 
  - Actual: 
  - Severity: [Critical/High/Medium/Low]

### Features Working
- [ ] Feature 1: [Status]
- [ ] Feature 2: [Status]

### Overall Status: [✓ Pass / ⚠ Needs Work / ✗ Fail]

---

## Post-Testing Tasks

After testing, check:
1. [ ] No new console errors found
2. [ ] All core features working
3. [ ] Error handling is helpful
4. [ ] Performance is acceptable
5. [ ] Ready for PR review

## Related Files
- SIDEPANEL_KNOWN_ISSUES.md - Known issues and fixes
- src/sidebar/Sidebar.tsx - Main chat interface
- src/settings/Settings.tsx - Settings configuration
- src/background.ts - Message handling & suggestion generation
- dist/ - Built extension ready for testing
