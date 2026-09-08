# Phase 1.2 Implementation Checklist

**Status**: Starting (Backend complete, Extension in progress)  
**Timeline**: ~40 hours estimated  
**Complexity**: Medium (UI + Extension communication)

---

## Overview

Phase 1.2 builds the browser extension UI that allows users to interact with the Moly backend. Users will:
1. Select text they want help with
2. Get AI suggestions instantly
3. Save context (AboutMe, Contact profiles)
4. See suggestions refined over time

---

## Architecture

```
User Types Message
    ↓
Content Script Detects Context
    ↓
Popup Sends to Background
    ↓
Background Worker Calls Backend
    ↓
Backend Processes (LLM + DB)
    ↓
Response Shows Suggestions
    ↓
User Chooses + Modifies
    ↓
Feedback Recorded to Backend
```

---

## Files Created So Far (Phase 1.2)

✅ `moly-extension/manifest.json` - Extension metadata  
✅ `moly-extension/background.js` - Service worker (600 lines)

---

## Files Still Needed (Phase 1.2)

### 1. Content Script (Medium Priority)
**File**: `moly-extension/content.js`  
**Purpose**: Injects UI into web pages, detects textareas/forms  
**Responsibilities**:
- Listen for user typing in textareas/contenteditable
- Add "Get Moly Help" button near text areas
- Capture selected text
- Send to background worker
- Display suggestions in popup

**Estimated size**: 300-400 lines  
**Dependencies**: background.js  

### 2. Popup HTML (High Priority)
**File**: `moly-extension/popup.html`  
**Purpose**: Main UI when user clicks extension icon  
**Sections**:
- Header with logo
- Status (connected to backend / status)
- Quick setup (if first time)
- About Me / Contact forms
- Conversation history
- Settings button

**Estimated size**: 150 lines  

### 3. Popup CSS (High Priority)
**File**: `moly-extension/popup.css`  
**Purpose**: Style the popup UI  
**Requirements**:
- Dark/light theme support
- Responsive to different window sizes
- Accessible colors and fonts
- Smooth animations

**Estimated size**: 400-500 lines  

### 4. Popup JavaScript (High Priority)
**File**: `moly-extension/popup.js`  
**Purpose**: Handle popup interactions  
**Responsibilities**:
- Display backend status
- Show conversation suggestions
- Form handling (AboutMe, Contact)
- Send/receive messages with background worker
- Update UI with responses

**Estimated size**: 400-500 lines  

### 5. Options Page HTML (Medium Priority)
**File**: `moly-extension/options.html`  
**Purpose**: Configuration page  
**Sections**:
- Backend URL setting
- LLM provider selection
- Log level setting
- Data export/import
- About Moly

**Estimated size**: 100 lines  

### 6. Options CSS (Medium Priority)
**File**: `moly-extension/options.css`  
**Purpose**: Style options page  
**Estimated size**: 200 lines  

### 7. Options JavaScript (Medium Priority)
**File**: `moly-extension/options.js`  
**Purpose**: Handle settings  
**Responsibilities**:
- Load settings from chrome.storage
- Save settings
- Validate backend URL
- Test connection

**Estimated size**: 200-300 lines  

### 8. Icon Assets (Low Priority)
**Files**: 
- `moly-extension/images/icon-16.png` (16x16)
- `moly-extension/images/icon-48.png` (48x48)
- `moly-extension/images/icon-128.png` (128x128)

**Purpose**: Extension icon in browser toolbar  
**Note**: Can use placeholder SVG for now  

---

## Testing Checklist

### Unit Tests
- [ ] background.js message handling
- [ ] content.js textarea detection
- [ ] popup.js UI updates

### Integration Tests
- [ ] Extension connects to backend
- [ ] Health check works
- [ ] Suggestion generation end-to-end
- [ ] Feedback recording end-to-end

### Manual Tests
- [ ] Load extension in Chrome
- [ ] Click icon, see popup
- [ ] Open options page
- [ ] Type in textarea, get suggestions
- [ ] Modify suggestion
- [ ] Record feedback
- [ ] See updated profile

### Compatibility Tests
- [ ] Chrome 120+
- [ ] Edge 120+
- [ ] Firefox (if manifests v3 support)

---

## Implementation Order

**Priority 1** (Core functionality):
1. Popup HTML/CSS
2. Popup JavaScript
3. Content Script
4. Test end-to-end

**Priority 2** (Configuration):
5. Options page (HTML/CSS/JS)
6. Settings integration

**Priority 3** (Polish):
7. Icon assets
8. Error handling improvements
9. Loading states

---

## Known Challenges

### CORS Issues
- Backend at `localhost:8080`
- Extension has `<all_urls>` permission
- Should work without proxy in development
- May need reverse proxy for production

### Message Passing
- Content script → Background worker → API
- Must handle async/await properly
- Must close connections gracefully

### UI Constraints
- Small popup window (320x600)
- Must handle long suggestions
- Must show real-time feedback

### Security
- Don't store auth tokens in extension
- Sanitize all user input before display
- Validate backend responses

---

## Estimated Implementation Timeline

| Component | Effort | Prereq | Est. Time |
|-----------|--------|--------|-----------|
| Popup HTML/CSS | Medium | None | 4 hours |
| Popup JS | Medium | Popup HTML | 6 hours |
| Content Script | Medium | Popup JS | 6 hours |
| Options Page | Easy | Popup | 3 hours |
| Integration Testing | Medium | All UI | 6 hours |
| Bug fixes + Polish | Medium | Tests | 6 hours |

**Total Phase 1.2**: ~31 hours  
**With buffer**: ~40 hours

---

## Success Criteria

Phase 1.2 is complete when:

- [ ] Extension loads in Chrome DevTools
- [ ] Popup shows backend health status
- [ ] User can type a message in any textarea
- [ ] User gets 2+ AI suggestions
- [ ] User can modify suggestion
- [ ] Modified text is recorded as feedback
- [ ] Extension shows it learned (simple UI indicator)
- [ ] All integration tests pass
- [ ] No console errors
- [ ] Works with Ollama + heuristic fallback

---

## Files Manifest

```
moly-extension/
  ├── manifest.json              ✅ Done
  ├── background.js              ✅ Done
  ├── content.js                 ⏳ TODO
  ├── popup.html                 ⏳ TODO
  ├── popup.css                  ⏳ TODO
  ├── popup.js                   ⏳ TODO
  ├── options.html               ⏳ TODO
  ├── options.css                ⏳ TODO
  ├── options.js                 ⏳ TODO
  ├── images/
  │   ├── icon-16.png            ⏳ TODO (SVG placeholder OK)
  │   ├── icon-48.png            ⏳ TODO
  │   └── icon-128.png           ⏳ TODO
  └── README.md                  ⏳ TODO
```

---

## Next Steps

1. Create `content.js` (textarea detection + UI injection)
2. Create `popup.html` + `popup.css` + `popup.js` (main UI)
3. Create `options.html` + `options.css` + `options.js` (settings)
4. Test in Chrome DevTools
5. Debug API communication
6. Polish UI and add error handling

---

## Backend API Reference (for Frontend Dev)

All endpoints return JSON and handle CORS:

### POST /api/v2/conversation/generate
**Request**:
```json
{
  "userId": "string",
  "conversationId": "string",
  "userMessage": "string"
}
```

**Response**:
```json
{
  "phase": "context_gathering|suggestions_ready",
  "questions": ["string"],
  "suggestions": [
    {
      "text": "string",
      "confidence": 0.0-1.0,
      "reasoning": "string"
    }
  ]
}
```

### POST /api/v2/conversation/feedback
**Request**:
```json
{
  "userId": "string",
  "conversationId": "string",
  "suggestionChosen": 0,
  "modificationRequest": "string",
  "userModified": true,
  "reflectionApproved": true
}
```

### GET /api/v2/health
**Response**:
```json
{
  "status": "ok",
  "database": "ok",
  "llm": "ok|none",
  "time": "2026-09-08T12:00:00Z"
}
```

---

## Current Limitations (Phase 1.2)

- Extension only works on pages with textareas/contenteditable
- No support for code editors yet
- Limited to English
- No conversation history UI yet
- No user authentication (uses local ID)

---

## Future Enhancements (Phase 2+)

- Cloud sync of user profiles
- Multi-device support
- Advanced context awareness
- Learning dashboard
- Privacy controls
- Offline mode
- Community suggestions

---

**Phase 1.2 is the final piece to make Moly usable end-to-end.**
