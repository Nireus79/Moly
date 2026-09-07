# Moly Complete Architecture & Workflows

## System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                   MOLY EXTENSION (Chrome)                       │
│  moly-extension/src → TypeScript/React → Compiled to dist/     │
└─────────────────────────────────────────────────────────────────┘
                     ↓                              ↓
        ┌────────────────────────┐    ┌─────────────────────────┐
        │  Port 11435            │    │  Port 11436             │
        │  CORS Proxy            │    │  Go Backend             │
        │  (moly-proxy)          │    │  (moly-go)              │
        │  Node.js               │    │  Safety/Ethics/Analysis │
        └────────────────────────┘    └─────────────────────────┘
                     ↓
        ┌────────────────────────┐
        │  Port 11434            │
        │  Ollama (Local Models) │
        │  Mistral/Llama2/etc    │
        └────────────────────────┘
```

## Complete Workflow: User Clicks Extension

### **Phase 1: Extension Launch & Backend Check**

```
User clicks Moly icon
    ↓
chrome.action.onClicked (serviceWorker.ts:20)
    ↓
BackendManager.ensureRunning() (backendManager.ts)
    ↓
    ├─ Health check: GET http://127.0.0.1:11436/api/status
    │  ├─ If healthy → Continue
    │  └─ If unhealthy → Try to start via native host
    │
    ├─ Native host attempt: com.moly.backend_host
    │  ├─ Spawns: cd moly-go && ./moly
    │  └─ Waits: up to 5 seconds for startup
    │
    └─ Fallback: If not started, show instructions
       └─ Extension continues (graceful degradation)
         
Result: BackendStatus component shows:
  ✓ Green: "Backend connected" (11436 healthy)
  ⏳ Yellow: "Starting backend..." (checking)
  ✗ Red: Instructions to start manually
```

### **Phase 2: Sidebar Rendering**

```
Backend status checked → Sidebar rendered
    ↓
BackendStatus component displays status
    ↓
ContactSelector → (user selects who they're talking to)
    ↓
ChatHistory → (displays conversation)
    ↓
MessageInput → (where user types)
```

### **Phase 3: User Types Message**

```
User types message in MessageInput
    ↓
handleSendMessage() called (Sidebar.tsx:80)
    ↓
MolyAgent.analyze() called (if backend available)
    ├─ POST http://127.0.0.1:11436/api/check-safety
    │  └─ SafetyAlert component shows if crisis detected
    │
    ├─ POST http://127.0.0.1:11436/api/evaluate-constitution
    │  └─ Show ethics violations if found
    │
    └─ POST http://127.0.0.1:11436/api/generate-questions
       └─ Show contextual questions for user reflection
```

### **Phase 4: LLM Suggestion Generation**

```
Message sent from Sidebar
    ↓
chrome.runtime.sendMessage({type: 'GENERATE_SUGGESTIONS'})
    ↓
serviceWorker.ts: generateSuggestions()
    ↓
Get active LLM provider from Settings:
    ├─ If Ollama (default):
    │  ├─ Try proxy at 11435: http://localhost:11435/api/generate
    │  │  (CORS proxy forwards to Ollama on 11434)
    │  │
    │  └─ Fallback to direct: http://localhost:11434/api/generate
    │     (May fail with 403 if no proxy)
    │
    ├─ If Claude: Use API key → Claude API
    │
    └─ If OpenAI: Use API key → OpenAI API
       
Provider returns suggestions
    ↓
Display in Suggestions component
    ↓
User copies suggestion to clipboard
    ↓
User sends in original app
```

## Component Interconnections

### **Service Worker (Background Process)**

**File:** `moly-extension/src/background/serviceWorker.ts`

**On Extension Launch:**
1. Logs: "Background service worker loaded"
2. Prepares message listeners
3. Does NOT auto-start backend (waits for first click)

**On Extension Icon Click:**
1. Calls `BackendManager.ensureRunning()`
2. Waits for backend health check
3. Injects sidebar on active tab

**On Message `GENERATE_SUGGESTIONS`:**
1. Gets user settings (active LLM, API keys)
2. Calls active provider's `generateSuggestions()`
3. Returns suggestions to Sidebar

### **BackendManager (11436 Health Check)**

**File:** `moly-extension/src/api/backendManager.ts`

**Responsibilities:**
- Detects if Go backend is running
- Attempts to start via native messaging if needed
- Retries for up to 5 seconds
- Reports status to Sidebar
- Graceful fallback if backend unavailable

**Ports:**
- Checks: `http://127.0.0.1:11436/api/status`
- Native host message: `com.moly.backend_host`

### **MolyAgent (Go Backend Bridge)**

**File:** `moly-extension/src/api/molyAgent.ts`

**Methods:**
- `checkSafety(message)` → Safety/crisis detection
- `evaluateConstitution(message)` → Ethics assessment
- `analyzeModeShift(current, proposed, context)` → Relationship analysis
- `generateQuestions(contactName, context)` → Contextual questions
- `analyzeMessage(msg, contact, context)` → Full pipeline

**Port:** 11436 (Go backend)

**Error Handling:**
- Throws if backend unavailable
- Hook catches and logs gracefully
- Extension continues with LLM only

### **Sidebar Component**

**File:** `moly-extension/src/sidebar/Sidebar.tsx`

**Flow:**
1. Mount → Show BackendStatus
2. User picks contact → ChatHistory loads
3. User types → handleSendMessage()
4. Analysis runs → Show SafetyAlert + Ethics warnings
5. LLM call → Show Suggestions
6. User copies → Store in history

**Hooks Used:**
- `useChatStore()` - conversation state
- `useSettingsStore()` - user config
- `useMolyAgent()` - backend analysis
- `useState()` - UI state

### **Ollama Provider (LLM)**

**File:** `moly-extension/src/api/providers/ollama.ts`

**Flow:**
1. Default baseUrl: `http://localhost:11435` (CORS proxy)
2. Try to discover models from proxy
3. If proxy unreachable → Fallback to direct: `http://localhost:11434`
4. Generate suggestions via active model
5. Return to service worker

**Ports:**
- Primary: 11435 (CORS proxy)
- Fallback: 11434 (direct Ollama)

### **CORS Proxy**

**File:** `moly-proxy/bin/moly-proxy.js`

**Purpose:**
- Listens on 11435
- Forwards to Ollama on 11434
- Adds CORS headers to responses
- Handles preflight OPTIONS requests

**Install & Run:**
```bash
cd moly-proxy
npm install
node bin/moly-proxy.js
```

### **Go Backend**

**File:** `moly-go/main.go`

**Port:** 11436

**Systems:**
1. SafetyChecker (safety_check.go)
   - Detects crisis/self-harm language
   - Detects illegal activity
   - Returns resources if crisis detected

2. ConstitutionEvaluator (constitution.go)
   - Evaluates against 10 ethical principles
   - Returns violations & aligned principles

3. ModeTransitionEngine (mode_transition.go)
   - Analyzes relationship mode shifts
   - Returns risk level & mitigation strategies

4. QuestionAgent (question_agent.go)
   - Generates contextual questions
   - Uses LLM to determine what needs clarifying

**CORS:** All responses include `Access-Control-Allow-Origin: *`

## Data Flow: Complete Message Journey

```
User types: "Hey, how are you?"
    ↓
handleSendMessage() called
    ├─ UI: Show message in chat
    ├─ Storage: Save to chatStore
    │
    └─ BackendAnalysis (optional if 11436 available):
       ├─ POST /api/check-safety
       │  └─ Result: {alert_type: "none", ...}
       │     (UI: Hide SafetyAlert)
       │
       ├─ POST /api/evaluate-constitution
       │  └─ Result: {violations: [], aligned_principles: [...]}
       │     (UI: Show "Ethics: All aligned")
       │
       └─ POST /api/generate-questions
          └─ Result: {questions: ["...", "...", "..."], ...}
             (UI: Show in context panel)
    
LLM Suggestion Generation:
    ├─ chrome.runtime.sendMessage({type: 'GENERATE_SUGGESTIONS'})
    │
    └─ serviceWorker.ts:
       ├─ Get settings → activeProvider = 'ollama'
       ├─ Get Ollama config → baseUrl = 'http://localhost:11435'
       │
       └─ OllamaProvider.generateSuggestions():
          ├─ POST http://localhost:11435/api/generate
          │  └─ CORS Proxy intercepts
          │     └─ Forward to http://localhost:11434/api/generate
          │        └─ Ollama returns response
          │           └─ Proxy forwards with CORS headers
          │              └─ Extension receives response
          │
          ├─ Parse suggestions
          └─ Return to Sidebar

Sidebar Display:
    ├─ Show suggestions in Suggestions component
    ├─ Add copy button for each
    └─ On copy: Save to history & store

User Action:
    └─ Paste to messaging app
```

## Port Map

| Port | Service | Purpose |
|------|---------|---------|
| **11434** | Ollama | Local LLM models (Mistral, Llama2, etc.) |
| **11435** | moly-proxy | CORS proxy forwarding to 11434 |
| **11436** | moly-go | Backend: safety, ethics, analysis |

**Extension:** Runs in browser context (no fixed port)

## Starting Everything

### **Minimal Setup (Ollama + Proxy + Extension)**

```bash
# Terminal 1: Start Ollama
ollama serve

# Terminal 2: Start CORS Proxy
cd moly-proxy
node bin/moly-proxy.js

# Terminal 3: Start Go Backend (optional, for safety/ethics)
cd moly-go
./moly

# Terminal 4: Load extension in Chrome
# chrome://extensions → Load unpacked → moly-extension/dist/
```

### **Systemd Setup (Linux, auto-start)**

```bash
# Set up services
sudo systemctl --user enable moly-proxy
sudo systemctl --user enable moly-go
sudo systemctl --user start moly-proxy
sudo systemctl --user start moly-go

# Extension loads, auto-detects running services
```

## Graceful Degradation

| Component Down | Extension Behavior |
|---|---|
| Go backend (11436) | ✓ Works - uses LLM only, no safety/ethics |
| CORS Proxy (11435) | ✓ Tries direct Ollama (11434), may fail with CORS |
| Ollama (11434) | ✓ Falls back to Claude/OpenAI if configured |
| All backends down | ✓ Works if Claude/OpenAI API key configured |

Extension never crashes - always has a fallback path.

## Verification Checklist

- [ ] CORS Proxy running on 11435: `curl http://localhost:11435/api/tags`
- [ ] Ollama running on 11434: `curl http://localhost:11434/api/tags`
- [ ] Go Backend running on 11436: `curl http://localhost:11436/api/status`
- [ ] Extension loads without errors: `chrome://extensions`
- [ ] Extension detects backend: BackendStatus shows ✓ or ⏳ or ✗
- [ ] Sidebar shows after clicking icon
- [ ] Typing message → suggestions appear (LLM working)
- [ ] Console shows no errors: F12 → Console tab
- [ ] Backend analysis works: console shows `[Moly]` logs

## Debugging

**If extension crashes:**
1. Check console: F12 → Extensions → Inspect
2. Look for errors about missing ports
3. Check if all services are running

**If no suggestions appear:**
1. Check if CORS proxy running on 11435
2. Check if Ollama running on 11434
3. Check if Claude/OpenAI API key configured
4. See console for provider errors

**If 403 Forbidden error:**
1. CORS proxy not running
2. Start: `cd moly-proxy && node bin/moly-proxy.js`

**If backend analysis not working:**
1. Go backend might not be running
2. Check: `curl http://localhost:11436/api/status`
3. Start: `cd moly-go && ./moly`

---

**Status:** All interconnections verified ✓  
**Last Updated:** 2026-09-05  
**Version:** v1.0
