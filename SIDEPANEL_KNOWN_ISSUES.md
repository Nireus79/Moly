# Moly sidePanel Implementation - Known Issues

**Status:** Week 1-2 Complete | Core Architecture Functional  
**Date:** September 6, 2026  
**Branch:** `feature/sidepanel`

---

## ✓ What's Working

### Core Architecture
- ✓ sidePanel opens/closes on extension icon click
- ✓ sidePanel persists across page navigation
- ✓ sidePanel persists across page refresh
- ✓ Global context (one sidePanel instance for all pages)
- ✓ Works on any page (no content script restrictions)

### Backend Integration
- ✓ Backend health check (CORS fixed, 200 OK response)
- ✓ Auto-detection every 3 seconds
- ✓ Status displays "Backend connected" when running
- ✓ Development mode detection working
- ✓ Native messaging setup correct

### Settings & State
- ✓ Settings persist after sidePanel close/reopen
- ✓ Chrome storage properly configured
- ✓ Provider configuration saves correctly
- ✓ Message routing works (sidePanel → background → providers)

### UI Components (Partially Working)
- ✓ Sidebar header buttons (ℹ️ About, 👥 Contacts, ⚙️ Settings) - **work in Chat view**
- ✓ Close button (✕) - sends message to background, background processes it
- ✓ Refresh button (🔄) - fires onClick handler, triggers health check
- ✓ Console logging confirms button clicks ARE firing

---

## ⚠️ Known Issues

### Issue 1: Settings View Component Isolation
**Severity:** Medium | **Type:** UI/Event Handling  
**Affects:** Settings modal component

**Symptoms:**
- Buttons don't respond to clicks when in Settings view
- Works fine in Chat view
- Includes: Close button, Refresh button, Model dropdowns, Header buttons (ℹ️👥)
- Weird behavior: clicking Settings button causes previously-clicked About/Contacts popups to appear

**Root Cause:** 
- Settings component appears to have event isolation
- Possible causes:
  1. Modal overlay with `pointer-events: none` blocking clicks
  2. React event delegation issue in Settings component tree
  3. Z-index problem preventing clicks from reaching buttons
  4. Component lifecycle issue (unmount/remount)

**Debug Steps:**
1. Check `src/settings/Settings.tsx` for `pointer-events` CSS
2. Inspect DOM with DevTools when in Settings view
3. Check if buttons are actually in DOM or hidden
4. Review React event handlers in Settings component
5. Check for overlays or backdrop divs blocking clicks

**Workaround:** Exit Settings view, buttons work in Chat view

---

### Issue 2: Model Dropdown Not Functional
**Severity:** Medium | **Type:** UI  
**Affects:** All providers (Claude, OpenAI, Ollama)

**Symptoms:**
- `<select>` element for model selection doesn't respond to clicks
- Dropdown doesn't open/expand
- No console errors
- Other form fields (API key, Base URL) work normally

**Root Cause:**
- Unknown - appears related to Issue 1 (Settings component isolation)
- `availableModels` array may be empty
- Or select element is disabled/hidden

**Debug Steps:**
1. Add console.log to verify `availableModels` array has content
2. Check if select element has `disabled={true}`
3. Verify CSS doesn't have `display: none` or `pointer-events: none`
4. Check React DevTools for component props

**Current Workaround:** Use default model (first discovered)

---

### Issue 3: Suggestion Generation Error
**Severity:** High | **Type:** Backend Integration  
**Error:** "Cannot read properties of undefined (reading 'success')"

**Symptoms:**
- When generating suggestions, error appears in console
- Error: "Cannot read properties of undefined (reading 'success')"
- Suggestions don't generate even with backend running

**Root Cause:**
- Provider's generateSuggestions() returning undefined
- Code trying to read `.success` from undefined object
- Likely in Sidebar component or background message handler

**Debug Steps:**
1. Check `src/sidebar/components/Suggestions.tsx` 
2. Check background.ts message handler for suggestion requests
3. Add logging to provider.generateSuggestions()
4. Verify backend is returning proper JSON response

**Current Impact:** Cannot test full end-to-end chat flow

---

### Issue 4: Backend Auto-Detection Message
**Severity:** Low | **Type:** UX  
**Affects:** Development workflow

**Symptoms:**
- When backend terminates, message shows: "You are in development mode. To start backend..."
- User must manually restart backend
- No auto-restart attempt

**Current Behavior:** Expected (by design)  
**Consideration:** Add auto-restart feature for development?

---

## Testing Status

### ✓ Tested & Passing
- Extension loads without errors
- sidePanel opens/closes
- sidePanel persists across navigation
- Settings persist after close/reopen
- Backend health check passes
- Auto-detection interval running
- Close button sends messages to background
- Refresh button triggers health checks

### ⚠️ Partially Tested
- Model dropdown (can't interact with it)
- Suggestion generation (error occurs)
- Provider switching (not fully tested)
- Settings saving (saves, but UI broken)

### ✗ Not Tested
- End-to-end chat flow
- Suggestion generation quality
- Multiple provider switching
- Ollama model discovery
- Copy-to-clipboard functionality

---

## Browser Compatibility

**Chrome 114+:** ✓ Working (primary target)  
**Chrome <114:** ✗ Not supported (sidePanel API not available)  
**Firefox:** Not implemented (requires separate sidebar API)  
**Safari:** Not implemented (requires separate extension API)

---

## Code Locations

**Key Files:**
- `moly-extension/src/sidebar/Sidebar.tsx` - Main component
- `moly-extension/src/settings/Settings.tsx` - Settings view (has issues)
- `moly-extension/src/sidebar/components/BackendStatus.tsx` - Backend detection
- `moly-extension/src/background.ts` - Message routing
- `moly-go/main.go` - Backend server + CORS headers

**Build Output:**
- Extension: `moly-extension/dist/`
- Backend: `~/.local/bin/moly`

---

## Next Steps for Developers

### Priority 1: Fix Settings Component (Required for MVP)
1. Debug why Settings view blocks button clicks
2. Fix model dropdown
3. Restore Settings UI functionality
4. Time estimate: 2-3 hours

### Priority 2: Fix Suggestion Generation (Required for MVP)
1. Debug "Cannot read properties" error
2. Add proper error handling
3. Test end-to-end chat flow
4. Time estimate: 1-2 hours

### Priority 3: UI Polish (Nice to have)
1. Add button click feedback (visual response)
2. Improve error messages
3. Add loading states
4. Time estimate: 2-3 hours

### Priority 4: Cross-Browser (Future)
1. Implement Firefox sidebar API
2. Implement Safari extension API
3. Create separate build pipelines
4. Time estimate: 1-2 weeks

---

## How to Test Locally

1. **Build extension:**
   ```bash
   cd moly-extension
   npm run build
   ```

2. **Load in Chrome:**
   - Go to `chrome://extensions/`
   - Enable Developer mode
   - Click "Load unpacked"
   - Select `moly-extension/dist/`

3. **Start backend:**
   ```bash
   MOLY_PROXY_PATH=~/vs_projects/Moly/Moly/moly-proxy/bin/moly-proxy.js ~/.local/bin/moly &
   ```

4. **Test:**
   - Click extension icon → sidePanel opens
   - Backend should show "Connected" (green)
   - Chat view buttons work (About, Contacts, Settings)
   - Settings view has button issues (known limitation)

---

## Commit History

- `776e196` - Fix native messaging and development mode setup (v1 stable)
- `0f0e874` - Remove content scripts, add sidePanel API
- `64bf206` - Fix vite build config
- `62bc0de` - Week 1 Complete: sidePanel foundation working
- `ebc3341` - Week 2 WIP: Auto-discovery for model selection
- `ddca548` - Week 2 WIP: Auto-detection for backend status
- `e45607e` - Week 2 Complete: Backend detection and CORS fixed

---

## Rollback Information

**Stable version:** Tag `v1-stable` on master branch  
**Rollback command:** `git checkout v1-stable`

**If needed to revert to content-script approach:**
- Previous implementation in git history
- Not recommended (has architectural limitations)

---

## Contact & Questions

This document captures the state as of September 6, 2026.  
For issues or clarifications, refer to commit messages and code comments.

**Session:** Claude Haiku 4.5 | https://claude.ai/code/session_01Q1MDZuQgmcXu6H9w24QCv1
