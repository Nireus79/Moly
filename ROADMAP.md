# Moly v1.0 - Complete Roadmap

## Vision
**Unified AI Coaching Product:** Desktop app + browser extension that works seamlessly together. Install once, sidebar appears automatically on any webpage.

---

## Architecture Overview

```
USER WORKFLOW:
1. User downloads Moly installer
2. Runs installer → desktop app starts in background
3. Opens browser → extension auto-launches sidebar
4. Uses sidebar on any webpage (including blank pages)
5. Sidebar can expand to full browser width
6. On uninstall → clean removal with model choice
```

**Technical Stack:**
- **Desktop**: Electron app running HTTP server on :11436
- **Extension**: Chrome extension with content script that injects iframe
- **Native Host**: Python app for system integration (auto-launch, installation)
- **Sidebar**: Served from desktop app, embedded in browser via iframe
- **Communication**: HTTP requests between extension and desktop app

---

## Phase 1: Core Fixes (This Session)

### 1.1 Sandbox Issue - CRITICAL
**Problem:** Electron crashes with sandbox error  
**Status:** Workaround exists (`ELECTRON_DISABLE_SANDBOX=1`)  
**Fix:** One of three approaches:
- Option A: Disable sandbox for development (quick, temporary)
- Option B: Fix sandbox permissions (proper, needs root)
- Option C: Use `nodeIntegration: false` and proper IPC (safest)

**Tasks:**
- [ ] Choose sandbox approach
- [ ] Implement in main.js
- [ ] Test app launches without manual workaround
- [ ] Document in README

### 1.2 Hide Electron Window
**Problem:** App window is visible; should run in background  
**Status:** Not implemented  
**Solution:** Set window properties:
```javascript
const windowConfig = {
  show: false,  // Don't show on startup
  webPreferences: { nodeIntegration: false }
};
mainWindow.hide();  // Keep hidden
mainWindow.webContents.on('ready-to-show', () => {
  // Only show if user clicks app icon (future feature)
});
```

**Tasks:**
- [ ] Modify main.js to hide window
- [ ] Verify app still serves :11436
- [ ] Test sidebar loads without visible app window

### 1.3 Auto-Launch on Extension Load
**Problem:** Extension loads but app doesn't auto-start  
**Current:** User must manually start app  
**Solution:** Content script → native host → native host launches app

**Tasks:**
- [ ] Verify native messaging works (already tested)
- [ ] Test full flow: open webpage → app launches → sidebar appears
- [ ] Add timeout handling (retry if app takes time to start)

---

## Phase 2: Installation & Uninstallation

### 2.1 Installer Flow
**Scope:** Platform-specific installers (Windows MSI, macOS DMG, Linux AppImage)

**Installation Steps:**
1. Download installer
2. Run installer → shows welcome screen
3. Ask: "Install local model?" (optional)
   - If yes: show model selection (mistral, llama2, etc.)
   - If no: skip
4. Register native host for extension
5. Create desktop shortcuts
6. Launch app on first run

**Installer Features:**
- [ ] Welcome/license screen
- [ ] Model selection (download optional)
- [ ] System requirements check (Python, Node.js if needed)
- [ ] Native host registration
- [ ] Autostart configuration (systemd/LaunchAgent/Task Scheduler)
- [ ] Success screen with "Open Extension" button

### 2.2 Model Management UI
**In Desktop App Settings:**

```
Available Models:
  ✓ mistral (1.2GB) [Remove]
  ✓ llama2 (4GB) [Remove]
  
Install New Model:
  [Dropdown: llama2, mistral, neural-chat, ...]
  [Download] (shows progress)
```

**Tasks:**
- [ ] Build model management page in settings
- [ ] Add pull/remove model API calls
- [ ] Show download progress
- [ ] Handle errors gracefully

### 2.3 Uninstaller Flow
**Scope:** Clean removal of app, extension, native host

**Uninstall Steps:**
1. User runs uninstaller
2. Shows: "Keep local models?" (yes/no)
   - Yes: Keep ~/.local/share/moly/models
   - No: Delete models
3. Remove app from system
4. Remove extension manifest
5. Remove native host binary
6. Remove autostart config
7. Success message

**Tasks:**
- [ ] Create uninstaller script
- [ ] Ask user about models
- [ ] Clean all system locations
- [ ] Verify complete removal

---

## Phase 3: Sidebar UI & Functionality

### 3.1 Sidebar Layout (Expandable)
```
┌─────────────────────┐
│ Moly │ [^]  [✕]   │  <- Header (toggle expand, close)
├─────────────────────┤
│ [Chat message]      │
│ [User input]        │  <- Messages area
│ [AI suggestion]     │
├─────────────────────┤
│ [Model ▼] [Tone ▼] │  <- Quick settings
│ [Expand ↗] [Menu]  │  <- Actions
└─────────────────────┘

[Expanded to Full Browser Width:]
┌────────────────────────────────────────────────┐
│ [◄] Moly - Full Chat                   [✕]     │
├────────────────────────────────────────────────┤
│ Chat history                                    │
│ [User message]                                  │
│ [AI response]                                   │
│                                                 │
│ [Input field.................]  [Send]         │
│ [Model ▼] [Tone ▼] [Mode ▼]    [Settings]     │
└────────────────────────────────────────────────┘
```

### 3.2 Sidebar Features
**Chat:**
- [ ] Message history (current conversation)
- [ ] User input textarea
- [ ] Send button (Enter or click)
- [ ] Clear conversation
- [ ] Copy messages to clipboard

**Settings Panel (In Sidebar):**
- [ ] Model selection dropdown
- [ ] Tone/context selection (Friendly/Formal/Playful)
- [ ] Communication mode (Socratic/Direct)
- [ ] Open desktop app button
- [ ] Settings link
- [ ] Help link

**Expand Feature:**
- [ ] Toggle button to expand sidebar to full browser width
- [ ] Full-screen sidebar with all features
- [ ] Collapse back to sidebar
- [ ] Persist user preference

### 3.3 Sidebar Styling
- [ ] Matches desktop app theme
- [ ] Light/dark mode support
- [ ] Responsive (works in narrow sidebar and full browser)
- [ ] Smooth animations
- [ ] Accessible (keyboard navigation, high contrast)

**Tasks:**
- [ ] Build HTML/CSS for sidebar
- [ ] Add React components if needed
- [ ] Implement expand/collapse
- [ ] Test on various page widths
- [ ] Ensure works on blank pages

---

## Phase 4: Cross-Platform Support

### 4.1 Windows
- [ ] MSI installer
- [ ] Task Scheduler for autostart
- [ ] Registry for native host
- [ ] Electron sandbox configuration

### 4.2 macOS
- [ ] DMG installer
- [ ] LaunchAgent for autostart
- [ ] Finder integration
- [ ] Code signing (future: App Store)

### 4.3 Linux
- [ ] AppImage + deb package
- [ ] systemd user service for autostart
- [ ] Native host registration in ~/.config
- [ ] FUSE dependency resolution

---

## Phase 5: Documentation

### 5.1 User Documentation
- [ ] Installation guide (per platform)
- [ ] Quick start guide
- [ ] Troubleshooting guide
- [ ] FAQ

### 5.2 Developer Documentation
- [ ] Architecture overview
- [ ] API documentation
- [ ] Extension development guide
- [ ] Contributing guide

### 5.3 Configuration
- [ ] .env variables
- [ ] systemd service files
- [ ] Native host manifest format
- [ ] Extension permissions explained

---

## Current Status

### ✓ Completed
- [x] Desktop app serves sidebar on :11436
- [x] Extension injects iframe from :11436
- [x] Content script skips restricted pages
- [x] Native messaging works (can launch app)
- [x] Basic chat UI in sidebar

### 🚧 In Progress
- [ ] Sandbox fix (Electron)
- [ ] Hide app window
- [ ] Auto-launch flow

### 📋 TODO (Priority Order)
1. Fix Electron sandbox
2. Hide app window
3. Build sidebar UI (expand/collapse)
4. Implement model management
5. Create installer/uninstaller
6. Write documentation
7. Test on all platforms

---

## Key Technical Decisions

| Decision | Rationale | Status |
|----------|-----------|--------|
| **Sidebar = iframe from :11436** | Decouples extension from app, easier deployment | ✓ Working |
| **Native messaging for launch** | Secure, OS-standard way to run executables | ✓ Working |
| **Auto-hide app window** | Users only interact with sidebar, not app window | 🚧 Pending |
| **Expandable sidebar** | Full browser width when needed, compact normally | 📋 Planned |
| **systemd/LaunchAgent/Task Scheduler** | Standard OS autostart mechanisms | 📋 Planned |

---

## Success Criteria

- ✓ Install → everything ready
- ✓ Open any webpage → sidebar appears
- ✓ Sidebar fully functional (chat, settings, expand)
- ✓ Uninstall → complete removal + model choice
- ✓ Works on Windows, macOS, Linux
- ✓ Zero manual setup needed
- ✓ No visible app window by default
- ✓ Clear documentation

---

## Timeline Estimate

| Phase | Duration | Status |
|-------|----------|--------|
| Phase 1: Core Fixes | 2-3 hours | In Progress |
| Phase 2: Installation | 4-6 hours | Planned |
| Phase 3: Sidebar UI | 3-4 hours | Planned |
| Phase 4: Cross-Platform | 4-6 hours | Planned |
| Phase 5: Documentation | 2-3 hours | Planned |
| **Total** | **15-22 hours** | |

---

## Questions to Answer

1. Should app have visible UI at all? (Settings screen via browser?)
2. Should users be able to pause/stop sidebar?
3. Should sidebar remember conversation between sessions?
4. Should there be premium/free tiers?
5. What LLM models to pre-install?

---

*Last Updated: 2026-09-04*  
*Status: Active Development*
