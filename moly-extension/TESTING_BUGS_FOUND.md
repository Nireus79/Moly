# Testing Bugs Found - Phase 1 Manual Testing

**Date:** 2026-08-31
**Tester:** User Manual Testing Session
**Status:** IN PROGRESS - Issues documented, awaiting fixes

---

## Critical Issues (Blocking Functionality)

### 1. Settings Not Persisting ❌ CRITICAL
**Severity:** Critical
**Status:** Confirmed

**Description:**
- Ollama model is detected (shows "stable code")
- But when settings are saved, they don't persist
- Reloading page loses the model selection

**Steps to Reproduce:**
1. Open Settings
2. Select Ollama provider
3. Model detection shows available models
4. Click "Save & Validate"
5. Close settings
6. Reopen settings
7. Model selection is gone

**Expected:** Settings should persist in chrome.storage
**Actual:** Settings reset on reload

**Suspected Root Cause:** 
- Chrome storage API not being called correctly
- Settings store (Zustand) not syncing to chrome.storage
- Promise-based async operations not awaiting properly

**Files to Check:**
- `src/stores/settingsStore.ts` - updateProvider() method
- `src/settings/Settings.tsx` - handleSaveProvider() method
- Chrome storage permissions in manifest.json

---

### 2. Buttons Not Working ❌ CRITICAL
**Severity:** Critical
**Status:** Confirmed

**Description:**
- Various buttons in Settings/Popup don't respond to clicks
- No visual feedback when clicking
- May be related to React event handling

**Affected Buttons:**
- "Save & Validate" in Settings
- "Make Active" in Settings
- Contact selection in Popup
- Possibly others

**Steps to Reproduce:**
1. Open Settings
2. Try clicking any button
3. Button doesn't respond

**Expected:** Buttons should trigger their click handlers
**Actual:** No response/no visual feedback

**Suspected Root Cause:**
- React event handlers not properly bound
- onClick handlers not connected
- Event delegation issue
- JavaScript bundle not fully loaded when elements render

**Files to Check:**
- `src/settings/Settings.tsx` - button onClick handlers
- `src/popup/PopupEnhanced.tsx` - button onClick handlers
- `src/components/*.tsx` - all button components

---

### 3. Popup Window Too Small & Not Resizable ❌ CRITICAL
**Severity:** High
**Status:** Confirmed

**Description:**
- Popup appears very small (appears to be <400px width)
- Cannot resize window
- Content cramped and hard to read

**Current Size:** Approximately 300-350px width (estimated)
**Expected Size:** 380-500px minimum

**Expected:** Settings should be readable and adjustable
**Actual:** Window too small to use comfortably

**Suspected Root Cause:**
- CSS width not set correctly on popup-enhanced container
- Chrome popup default size too small
- CSS in popupEnhanced.css not being applied

**Files to Check:**
- `src/popup/popupEnhanced.css` - .popup-enhanced width
- `src/popup/PopupEnhanced.tsx` - inline styles
- `src/popup/popup.html` - viewport settings

**Fix:** Set width to 500px+ and ensure CSS loads

---

## Non-Critical Issues (Degraded Functionality)

### 4. Ollama Model Detection Works ✓ (Partially Working)
**Status:** Partially working
- Detection runs
- Shows "stable code" message
- But settings don't save

---

### 5. UI Responsiveness
**Status:** Needs testing
- Check if sidebar opens
- Check if chat interface works
- Check if message detection fires

---

## Test Summary

| Feature | Status | Notes |
|---------|--------|-------|
| Extension loads | ✓ | Window appears |
| Settings page opens | ✓ | Very small window |
| Ollama detection | ✓ | Works but doesn't save |
| Settings persistence | ✗ | Completely broken |
| Button clicks | ✗ | No response |
| Popup resizing | ✗ | Fixed size |
| Message detection | ? | Not tested |
| Sidebar chat | ? | Not tested |
| Contact management | ? | Not tested |

---

## Priority Fix Order

**Phase 1: Immediate (Must Fix)**
1. [ ] Fix button click handlers (React event binding)
2. [ ] Fix popup window size (CSS width)
3. [ ] Fix settings persistence (chrome.storage sync)

**Phase 2: Next (Should Fix)**
4. [ ] Test sidebar functionality
5. [ ] Test message detection
6. [ ] Test contact management
7. [ ] Test chat interface

**Phase 3: Polish (Nice to Have)**
8. [ ] Add error boundaries
9. [ ] Improve error messages
10. [ ] Add loading indicators

---

## Developer Notes

### Settings Persistence Flow
Current flow (should be):
1. User clicks "Save & Validate"
2. `handleSaveProvider()` called
3. `updateProvider()` called on store
4. Zustand store calls `chrome.storage.local.set()`
5. Data persists in chrome.storage

**Check:** Is the chrome.storage call actually happening?
- Add console.log in settingsStore before/after chrome.storage call
- Check DevTools → Storage tab to see if data is being saved

### Button Click Issue
Potential causes:
- Event handlers not attached due to async rendering
- React component not re-rendering after state change
- HTML script loading order issue (vendor chunks loading out of order)

**Check:** 
- DevTools → Elements to inspect button element
- DevTools → Console to see if click handler is firing
- Check if React is loaded before button.tsx runs

### Popup Size Issue
**Fix:** Set width in CSS
```css
.popup-enhanced {
  width: 500px;  /* Add this */
  max-width: 100%;
}
```

---

## Next Steps

1. **Read the TestingBugs file before proceeding to fixes**
2. **Fix issues in priority order** (buttons → size → persistence)
3. **Re-test after each fix**
4. **Update this file with fix status**
5. **Once all critical issues fixed, continue full feature testing**

---

**Total Critical Issues:** 3
**Total Non-Critical Issues:** 2
**Blocking Testing:** YES - Cannot proceed until button/settings issues fixed

