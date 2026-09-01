# Moly Workflow Design - All User Scenarios

Complete design covering every possible user journey. Must be "stupid proof."

## User Scenarios

### Scenario 1: User has Ollama installed with models already running
```
User state: Ollama installed + models pulled + Ollama running
User action: Installs Moly extension
Expected: Works immediately without any setup

Flow:
1. Moly extension loads
2. Auto-detection checks localhost:11434 (direct Ollama)
3. Discovers models (e.g., stable-code, mistral)
4. Shows "Using: Ollama (Local)" - DONE
5. Can generate suggestions immediately
```

**Current Status:** ✓ WORKS (direct fallback enabled)

---

### Scenario 2: User has Ollama installed but it's NOT running
```
User state: Ollama installed + models exist + Ollama stopped
User action: Installs Moly extension
Expected: Prompts to start Ollama

Flow:
1. Moly loads, checks localhost:11434
2. Ollama not responding
3. Shows: "Ollama installed but not running"
4. [Start Ollama] button → executes 'ollama serve'
   OR guides to manual start: "Run: ollama serve"
5. After starting, auto-refresh detects models
6. Uses Ollama automatically
```

**Current Status:** ✗ NEEDS IMPLEMENTATION
- Extension can't execute `ollama serve`
- Need: Clear instruction banner in Settings
- Need: Manual trigger to re-detect

---

### Scenario 3: User has Ollama with Model A, wants to add Model B via Moly
```
User state: Ollama running with mistral
User action: Clicks "Add Another Model" in Moly Settings
Expected: Adds model without manual CLI

Flow:
1. Moly Settings → Ollama tab → [Add Model]
2. Prompts: "Which model?"
3. Shows available models from ollama.ai
4. User selects stable-code
5. Moly runs: `ollama pull stable-code` (via installer)
6. Model auto-discovered after pull
7. Model available in dropdown
```

**Current Status:** ✗ NEEDS IMPLEMENTATION
- Need: Model selection UI
- Need: Ability to launch pull command
- Need: Real-time progress display

---

### Scenario 4: User has nothing, wants local model
```
User state: No Ollama, no LM Studio, no models
User action: Opens Moly, goes to Settings
Expected: Clear path to full setup

Flow:
1. User opens Settings → Local Models tab
2. Shows: "No local model detected"
3. Displays three options:
   a) [One-Click Setup] → Downloads + installs Ollama + pulls Mistral
   b) [Use LM Studio] → Links to lmstudio.ai
   c) [Cloud Only] → Use Claude/OpenAI instead
4. User clicks [One-Click Setup]
5. Installer launches (or downloads moly-installer)
6. Follows wizard (system check → download → model pull → auto-start)
7. Returns to Moly
8. Auto-detects running local model
9. DONE
```

**Current Status:** ✗ NEEDS IMPLEMENTATION
- Need: Local Model section in Settings
- Need: Status display (what's installed/running)
- Need: Launch installer from extension
- Need: Return from installer to extension

---

### Scenario 5: User installs Ollama/LM Studio manually, then opens Moly
```
User state: User went to ollama.ai, installed manually, pulled model
User action: Opens Moly extension
Expected: Auto-detects and works

Flow:
1. Ollama running with model
2. Moly loads
3. Auto-detection finds it
4. Models discovered
5. Works immediately
```

**Current Status:** ✓ WORKS (direct fallback enabled)

---

### Scenario 6: User has Claude API key but also wants local model
```
User state: Claude API key configured + Ollama running
User action: Just uses Moly
Expected: Prefers local, falls back to Claude if needed

Flow:
1. Moly tries Ollama first
2. If works: Uses local (no API costs)
3. If fails: Falls back to Claude
4. Shows which one was used
```

**Current Status:** ✓ WORKS (fallback system)

---

### Scenario 7: Multiple models, user wants to switch
```
User state: Ollama has mistral + stable-code + neural-chat
User action: Wants to use stable-code instead of current
Expected: Easy model switching

Flow:
1. Moly Settings → Ollama tab
2. Model dropdown shows: [mistral, stable-code, neural-chat]
3. User selects stable-code
4. [Save] button
5. Next suggestion uses stable-code
```

**Current Status:** ✓ WORKS (model dropdown exists)

---

## Design Requirements

### Settings Panel Redesign

**NEW: "Local Models" Section**
```
┌─ LOCAL MODELS ─────────────────────┐
│                                    │
│ Status: ○ Not Detected             │
│                                    │
│ Available Options:                 │
│ [✓] Ollama (CLI)                   │
│ [ ] LM Studio (GUI)                │
│                                    │
│ ┌─ Setup Guide ──────────────────┐│
│ │ No local model detected yet    ││
│ │                                ││
│ │ [One-Click Setup] → Installs   ││
│ │ [Manual Setup] → Guides        ││
│ │ [Cloud Only] → Use APIs        ││
│ └────────────────────────────────┘│
│                                    │
└────────────────────────────────────┘
```

### Smart Detection Logic

**On Extension Load:**
```javascript
// Check for running local models
1. Try localhost:11434 (Ollama direct)
   ├─ If responds → "Ollama Running"
   └─ If fails → Try next

2. Try localhost:8000 (LM Studio)
   ├─ If responds → "LM Studio Running"
   └─ If fails → Try next

3. Check if executables exist locally
   ├─ Ollama binary at ~/.ollama/? → "Ollama Installed (Not Running)"
   ├─ LM Studio at ~/.config/LM Studio? → "LM Studio Installed (Not Running)"
   └─ If none → "Not Installed"

4. Cloud APIs
   ├─ Claude API key configured? → "Cloud Ready"
   └─ OpenAI API key configured? → "Cloud Ready"

// Display status
Show most useful option first
```

---

## Implementation Plan

### Phase A: Detection & Status Display (MUST HAVE)

1. **Enhance Auto-Detection**
   - Detect if Ollama/LM Studio installed but not running
   - Detect if executables exist
   - Cache results for performance

2. **Status Banner in Settings**
   ```
   ✓ Ollama Detected (Running, stable-code model)
   ○ LM Studio (Not Running)
   ○ Claude (Configured)
   ```

3. **Action Buttons**
   ```
   [Start Ollama] → Shows instructions or attempts start
   [Configure] → Goes to relevant setup section
   [Add Model] → Shows model selection
   ```

### Phase B: Setup Guidance (CRITICAL)

1. **Local Model Section in Settings**
   - Clear status display
   - Three setup options (One-Click / Manual / Cloud)
   - Links to resources

2. **Installer Integration**
   - Detect if installer available
   - Launch installer from Settings
   - Return to Moly after setup

3. **Setup Instructions**
   - If Ollama not running: "Run: ollama serve"
   - If no model: "Run: ollama pull mistral"
   - If neither: "Use One-Click Setup"

### Phase C: Model Management (NICE TO HAVE)

1. **Add Model Button**
   - Show available models from ollama.ai
   - Pull selected model
   - Auto-refresh after pull

2. **Model Switching**
   - Easy dropdown to switch between loaded models
   - Shows which model is active

---

## Stupid-Proof Checklist

- [ ] User opens extension → immediately see what's available
- [ ] User can't be confused about what's running vs installed
- [ ] Clear action items if something's missing
- [ ] Ability to start stopped services if possible
- [ ] Installer accessible from within Settings
- [ ] Auto-detection runs continuously
- [ ] Status refreshes when needed
- [ ] Works offline after first setup
- [ ] No jargon - clear language for non-programmers
- [ ] Fallback to cloud works seamlessly if local fails
- [ ] Multiple models supported without confusion
- [ ] Can add/remove models without leaving extension

---

## User Messages (Copy)

**When Ollama installed but not running:**
```
"Ollama is installed but not running. 

To use it:
1. Open terminal
2. Run: ollama serve
3. Come back here and click 'Refresh'

Or use Claude for cloud-based responses."
```

**When nothing installed:**
```
"Ready to set up local AI?

• One-Click Setup (recommended)
  → Downloads Ollama + Mistral model
  → Runs automatically
  → Works offline after setup

• Manual Setup
  → I'll guide you through each step

• Cloud Only
  → Use Claude or OpenAI instead
  → No local installation needed"
```

**When multiple models available:**
```
"You have 3 models loaded:
✓ mistral (4.4 GB) - Currently using
• stable-code (3.9 GB) - [Use This]
• neural-chat (4.1 GB) - [Use This]

[Add Another Model]"
```

---

## Priority

**MUST HAVE (MVP):**
1. ✓ Auto-detect running Ollama (DONE)
2. ✓ Fallback system (DONE)
3. ✗ Status display in Settings (NOT DONE)
4. ✗ Setup guidance section (NOT DONE)

**SHOULD HAVE (v1.1):**
1. ✗ Installer integration
2. ✗ Detect installed but stopped services
3. ✗ Auto-refresh detection

**NICE TO HAVE (v2.0):**
1. ✗ Add model from Settings
2. ✗ Model management UI
3. ✗ Service control (Start/Stop)

---

## Testing Scenarios

After implementation, verify all 7 scenarios work:

- [ ] Scenario 1: User with running Ollama → Works immediately
- [ ] Scenario 2: User with installed but stopped Ollama → Gets guidance
- [ ] Scenario 3: User wants to add model → Can do it via Settings
- [ ] Scenario 4: User with nothing → Clear setup path
- [ ] Scenario 5: User installs manually → Auto-detects
- [ ] Scenario 6: User has both local + cloud → Works
- [ ] Scenario 7: User switches models → Easy dropdown

---

This design ensures **every possible user journey works without confusion.**
