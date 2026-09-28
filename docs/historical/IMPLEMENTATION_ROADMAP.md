# MOLY IMPLEMENTATION ROADMAP
## Complete Solutions with Wiring Details

**Created**: Sept 28, 2026  
**Status**: Ready for Implementation  
**Target**: Week of Sept 30, 2026  

---

## EXECUTION STRATEGY

This document contains **all wiring details** needed to implement the critical fixes. Each solution includes:
- Exact file paths and line numbers
- Method signatures to change
- Code to add/modify
- Database migrations (if needed)
- Testing procedures
- Success criteria

**Implementation order**: Week 1 (Critical) → Week 2 (Performance) → Week 3 (Self-Awareness) → Future (Architecture)

---

# WEEK 1: CRITICAL FIXES (5-8 hours)

## SOLUTION 1A: Intent-First Workflow Selection
**File**: `moly-go/agents/conversation_agent.go`  
**Lines**: 1235-1260  
**Effort**: 2-3 hours  
**Risk**: Low

### Current Code (Lines 1235-1260)

```go
// Determine workflow (priority order matters)
workflow := WorkflowAckWithSocratic // Default

// Priority 1: Gaps that need clarification (ALWAYS ask before suggesting)
if hasSignificantGaps && len(ctx.Gaps) > 0 {
    workflow = WorkflowGapQuestion
    log.Printf("[ConversationAgent] Workflow: Gap clarification (gaps=%d > 2)", len(ctx.Gaps))
} else if intentUnclear {
    // Priority 2: Intent is unclear - understand what user is doing before responding
    workflow = WorkflowIntentCheck
    log.Printf("[ConversationAgent] Workflow: Intent check (confidence=%.2f < 0.5)", intentAnalysis.Confidence)
} else if isFirstMessage {
    // Priority 3: First message - just acknowledge, gather context (no deepening yet)
    workflow = WorkflowAckOnly
    log.Printf("[ConversationAgent] Workflow: First message acknowledge only")
} else if shouldDeepen && !hasSignificantGaps && !intentUnclear {
    // Priority 4: Enough context + no gaps + intent clear + deepening allowed
    workflow = WorkflowAckWithSocratic
    log.Printf("[ConversationAgent] Workflow: Acknowledge with Socratic deepening")
} else {
    // Default: Acknowledge without deepening
    workflow = WorkflowAckOnly
    log.Printf("[ConversationAgent] Workflow: Acknowledge only (safe default)")
}
```

### New Code (INTENT FIRST)

Replace lines 1235-1260 with:

```go
// Determine workflow (NEW PRIORITY: Intent > Gaps > Context)
// High-confidence intent ALWAYS wins over gap clarification
workflow := WorkflowAckWithSocratic // Default

// Priority 1: HIGH-CONFIDENCE INTENT (≥0.85) - Always respond to intent first
// Examples: greeting (0.95), clear question (0.90), clear statement (0.88)
if intentAnalysis != nil && intentAnalysis.Confidence >= 0.85 {
    log.Printf("[ConversationAgent] HIGH-CONFIDENCE INTENT DETECTED: %s (confidence=%.2f) - proceeding with intent-based workflow",
        intentAnalysis.Intent, intentAnalysis.Confidence)
    
    // Intent-specific workflows override gaps
    // Gap clarification will come as FOLLOW-UP if needed, not override
    if intentAnalysis.Intent == "greeting" {
        workflow = WorkflowAckOnly  // Simple acknowledgment for greetings
        log.Printf("[ConversationAgent] Workflow: Greeting acknowledgment (will add gap follow-up if needed)")
    } else if intentAnalysis.Intent == "question" {
        workflow = WorkflowAckWithSocratic  // Answer question + deepen if appropriate
        log.Printf("[ConversationAgent] Workflow: Question answer + optional deepening")
    } else if intentAnalysis.Intent == "statement" {
        workflow = WorkflowAckWithSocratic  // Acknowledge + deepen
        log.Printf("[ConversationAgent] Workflow: Statement acknowledge + optional deepening")
    }
    // For other intents, use default workflow below

// Priority 2: MEDIUM-CONFIDENCE INTENT (0.6-0.85) - Intent + optional clarification
} else if intentAnalysis != nil && intentAnalysis.Confidence >= 0.6 {
    log.Printf("[ConversationAgent] MEDIUM-CONFIDENCE INTENT: %s (confidence=%.2f) - combine intent response with clarification",
        intentAnalysis.Intent, intentAnalysis.Confidence)
    
    if hasSignificantGaps && len(ctx.Gaps) > 0 {
        // Ask clarification as follow-up, not override
        workflow = WorkflowGapQuestion
        log.Printf("[ConversationAgent] Workflow: Intent response + gap clarification follow-up (gaps=%d)", len(ctx.Gaps))
    } else {
        workflow = WorkflowAckWithSocratic
        log.Printf("[ConversationAgent] Workflow: Intent response with deepening (no gaps)")
    }

// Priority 3: LOW-CONFIDENCE INTENT (<0.6) OR UNCLEAR - Ask clarification
} else if intentUnclear || (intentAnalysis != nil && intentAnalysis.Confidence < 0.6) {
    // Priority 3a: Gaps need clarification
    if hasSignificantGaps && len(ctx.Gaps) > 0 {
        workflow = WorkflowGapQuestion
        log.Printf("[ConversationAgent] Workflow: Gap clarification (gaps=%d, low intent confidence=%.2f)",
            len(ctx.Gaps), intentAnalysis.Confidence)
    } else {
        // Priority 3b: Intent unclear but no gaps
        workflow = WorkflowIntentCheck
        log.Printf("[ConversationAgent] Workflow: Intent check (confidence=%.2f < 0.6)", intentAnalysis.Confidence)
    }

// Priority 4: First message - just acknowledge, minimal gaps expected
} else if isFirstMessage {
    workflow = WorkflowAckOnly
    log.Printf("[ConversationAgent] Workflow: First message acknowledge only")

// Priority 5: Default - acknowledge without deepening
} else {
    workflow = WorkflowAckOnly
    log.Printf("[ConversationAgent] Workflow: Acknowledge only (safe default)")
}
```

### Key Changes Explained

1. **Lines 1242-1244**: Added intent confidence check BEFORE gap check
2. **Lines 1246-1254**: If intent confidence ≥ 0.85 (high confidence), route to intent-specific workflow
3. **Lines 1246-1261**: Greeting detected → WorkflowAckOnly (simple acknowledge)
4. **Lines 1264-1278**: If intent confidence 0.6-0.85 (medium), combine intent response with optional gap follow-up
5. **Lines 1281-1292**: If intent < 0.6 or unclear, THEN check gaps
6. **Lines 1295-1298**: Gaps only trigger if intent confidence is low

### Testing Procedure

**Test Case 1: Greeting Detection**
```
Input: "Hello Moly"
Expected:
  - IntentAnalysis.Intent = "greeting"
  - IntentAnalysis.Confidence = 0.95
  - Workflow = WorkflowAckOnly
  - Log: "HIGH-CONFIDENCE INTENT DETECTED: greeting"
  - Response: Simple greeting (no gap clarification)

Actual Before Fix:
  - Workflow = WorkflowGapQuestion
  - Response: "What specific topic are you hoping to gain clarity on?"

Actual After Fix:
  - Workflow = WorkflowAckOnly
  - Response: "Hello! Nice to see you."
```

**Test Case 2: Clear Question**
```
Input: "How do I talk to someone I like?"
Expected:
  - IntentAnalysis.Confidence ≥ 0.85
  - Workflow = WorkflowAckWithSocratic (answer + deepen)
  - Response: Answer question, optionally ask deeper question

After Fix:
  - System answers question directly, doesn't ask gap clarification first
```

**Test Case 3: Unclear Message with Gaps**
```
Input: "It's complicated..."
Expected:
  - IntentAnalysis.Confidence < 0.6
  - Workflow = WorkflowGapQuestion (gaps exist)
  - Response: Ask clarification about gaps

After Fix:
  - Workflow correctly routes to gap clarification (because intent IS low)
  - NOT because of automatic gap threshold
```

### Success Criteria

- ✅ Greeting (confidence ≥ 0.85) receives simple acknowledgment
- ✅ No gap clarification overrides high-confidence intents
- ✅ Gaps become follow-up questions, not primary response
- ✅ First message "Hello Moly" shows warmth, not interrogation
- ✅ All logs show correct workflow priority

---

## SOLUTION 3A: Tiered Timeout Strategy
**Files**: 
- `moly-go/tools/ollama.go` (lines 27, 41, 66, 92)
- `moly-go/tools/llm_client.go` (lines 145, 194)
- `moly-go/main.go` (startup, ~line 4700)

**Effort**: 1-2 hours  
**Risk**: Very Low (config only)

### Step 1: Add Hardware Detection (tools/llm_client.go, NEW)

Add new function at end of file (before final closing brace):

```go
// DetectHardwareProfile determines system capability for timeout tuning
// Runs a quick test call to Ollama and measures response time
func DetectHardwareProfile() string {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    testPrompt := "Respond with single word: fast"
    start := time.Now()
    
    client := &http.Client{Timeout: 30 * time.Second}
    req, _ := http.NewRequestWithContext(ctx, "POST", "http://127.0.0.1:11434/api/generate", nil)
    
    resp, err := client.Do(req)
    elapsed := time.Since(start)
    
    if err != nil {
        log.Printf("[Hardware Detection] Error during detection: %v, assuming slow hardware", err)
        return "slow"
    }
    defer resp.Body.Close()
    
    log.Printf("[Hardware Detection] Response time: %v", elapsed)
    
    if elapsed < 10*time.Second {
        log.Printf("[Hardware Detection] Fast hardware detected")
        return "fast"
    } else if elapsed < 30*time.Second {
        log.Printf("[Hardware Detection] Standard hardware detected")
        return "standard"
    } else {
        log.Printf("[Hardware Detection] Slow hardware detected")
        return "slow"
    }
}

// GetTimeoutForProfile returns timeout duration based on hardware profile
func GetTimeoutForProfile(profile string, operation string) time.Duration {
    timeouts := map[string]map[string]time.Duration{
        "fast": {
            "context_extract":      30 * time.Second,
            "clarity_analysis":     20 * time.Second,
            "entity_extraction":    20 * time.Second,
            "principle_detection":  15 * time.Second,
            "subject_shift":        15 * time.Second,
            "response_generation":  40 * time.Second,
            "risk_assessment":      20 * time.Second,
            "learning":             15 * time.Second,
            "constitutional_eval":  20 * time.Second,
            "default":              30 * time.Second,
        },
        "standard": {
            "context_extract":      2 * time.Minute,
            "clarity_analysis":     90 * time.Second,
            "entity_extraction":    90 * time.Second,
            "principle_detection":  60 * time.Second,
            "subject_shift":        60 * time.Second,
            "response_generation":  2 * time.Minute,
            "risk_assessment":      90 * time.Second,
            "learning":             60 * time.Second,
            "constitutional_eval":  90 * time.Second,
            "default":              2 * time.Minute,
        },
        "slow": {
            "context_extract":      5 * time.Minute,
            "clarity_analysis":     3 * time.Minute,
            "entity_extraction":    3 * time.Minute,
            "principle_detection":  2 * time.Minute,
            "subject_shift":        2 * time.Minute,
            "response_generation":  5 * time.Minute,
            "risk_assessment":      3 * time.Minute,
            "learning":             2 * time.Minute,
            "constitutional_eval":  3 * time.Minute,
            "default":              5 * time.Minute,
        },
    }
    
    if timeout, ok := timeouts[profile][operation]; ok {
        return timeout
    }
    return timeouts[profile]["default"]
}
```

### Step 2: Update LLMClient Struct (tools/llm_client.go, ~line 20)

Find struct definition and add field:

```go
type LLMClient struct {
    llmProvider string
    httpClient  *http.Client
    Timeout     time.Duration
    HardwareProfile string  // ← ADD THIS LINE
    // ... rest of fields
}
```

### Step 3: Update Ollama Client Initialization (tools/ollama.go)

**Current code (line 27)**:
```go
client := &http.Client{Timeout: 2 * time.Second}
```

**Change to**:
```go
// Timeout increased for resource-constrained systems
// Will be overridden by hardware-specific timeout if available
client := &http.Client{Timeout: 5 * time.Minute}
```

**Current code (line 41)**:
```go
client := &http.Client{Timeout: 5 * time.Second}
```

**Change to**:
```go
client := &http.Client{Timeout: 5 * time.Minute}
```

**Current code (line 92)**:
```go
client := &http.Client{Timeout: 10 * time.Second}
```

**Change to**:
```go
client := &http.Client{Timeout: 5 * time.Minute}
```

### Step 4: Initialize Hardware Detection in main.go

**Location**: `main.go`, in `main()` function, after database initialization (~line 4700)

Add after `v2Server := NewV2APIServer(...)`:

```go
// Detect hardware profile for timeout tuning
hardwareProfile := tools.DetectHardwareProfile()
v2Server.HardwareProfile = hardwareProfile
log.Printf("[Moly] Hardware profile detected: %s", hardwareProfile)

// Set LLM client timeout based on hardware
defaultTimeout := tools.GetTimeoutForProfile(hardwareProfile, "default")
v2Server.llmClient.Timeout = defaultTimeout
log.Printf("[Moly] Applied timeout profile: %s for hardware: %s", defaultTimeout, hardwareProfile)
```

### Step 5: Pass Timeout to LLM Calls

**In tools/llm_client.go**, update `CallOllama()` method (~line 260):

Find:
```go
ctx, cancel := context.WithTimeout(ctx, c.Timeout)
```

Update to:
```go
// Use hardware-specific timeout if available, otherwise use configured
timeout := c.Timeout
if c.HardwareProfile != "" {
    timeout = GetTimeoutForProfile(c.HardwareProfile, "default")
}
ctx, cancel := context.WithTimeout(ctx, timeout)
log.Printf("[LLMClient] Using timeout: %v for profile: %s", timeout, c.HardwareProfile)
```

### Testing Procedure

**Test Case 1: Hardware Detection**
```
Expected log on startup:
  [Hardware Detection] Response time: 2m5s
  [Hardware Detection] Slow hardware detected
  [Moly] Hardware profile detected: slow
  [Moly] Applied timeout profile: 5m0s for hardware: slow
```

**Test Case 2: Timeout Increases**
```
Before: 10 second timeouts → Cascading failures
After:  5 minute timeouts → Requests complete successfully
```

**Test Case 3: No More Deadline Exceeded**
```
Before logs:
  [Ollama] Request failed: context deadline exceeded

After logs:
  [LLMClient] Using timeout: 5m0s for profile: slow
  [Ollama] API call successful
```

### Success Criteria

- ✅ Hardware detection runs on startup
- ✅ Hardware profile logged (fast/standard/slow)
- ✅ Timeouts match hardware capability
- ✅ No "context deadline exceeded" errors
- ✅ LLM calls complete successfully (within 5-10 min per message)

---

## SOLUTION 3B: Graceful Fallback on Timeout
**Files**:
- `moly-go/agents/conversation_agent.go` (lines 870-900, 1530-1560)
- `moly-go/agents/risk_monitor.go` (update error handling)

**Effort**: 2-3 hours  
**Risk**: Low

### Step 1: Update Principle Detection Fallback (conversation_agent.go, ~line 870)

Find:
```go
[Layer 6-7] Principle engagement detection: 
```

Current pattern:
```go
// This calls LLM and fails on timeout
princConcerns := ca.detectPrincipleConcerns(userMessage, ctx)
if princConcerns != nil {
    // ... handle concerns
}
```

Change to graceful fallback:

```go
// [Layer 6-7] Principle engagement detection - graceful fallback on timeout
princConcerns := ca.detectPrincipleConcerns(userMessage, ctx)
if princConcerns == nil {
    // Timeout or error - return safe default
    log.Printf("[ConversationAgent] Layer 6-7: Principle detection failed/timed out, assuming no concerns")
    princConcerns = &models.PrincipleAnalysis{
        SignalsDetected: []string{},
        Confidence:      0.0,
        Fallback:        true,      // Mark as fallback
        Reason:          "timeout - assuming no concerns",
    }
    response.Metadata["principle_fallback"] = true
}
```

### Step 2: Update Subject Shift Detection Fallback (conversation_agent.go, ~line 1530)

Find:
```go
[SubjectShiftDetector] Detecting shifts:
```

Current pattern:
```go
shift := ca.subjectShiftDetector.DetectShift(prevMsg, currMsg)
if shift != nil && shift.Detected {
    // ... handle shift
}
```

Change to graceful fallback:

```go
// Subject shift detection - graceful fallback on timeout
var shift *models.SubjectShift
var err error

// Use timeout context
shiftCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

shift, err = ca.subjectShiftDetector.DetectShiftWithContext(shiftCtx, prevMsg, currMsg)

if err == context.DeadlineExceeded {
    log.Printf("[ConversationAgent] Subject shift detection timed out, assuming no shift")
    shift = &models.SubjectShift{
        Detected: false,
        Fallback: true,
        Reason:   "timeout - assuming no shift",
    }
    response.Metadata["subject_shift_fallback"] = true
} else if shift == nil {
    shift = &models.SubjectShift{Detected: false}
}

if shift != nil && shift.Detected {
    log.Printf("[ConversationAgent] [✓] Layer 9: Detected topic shift: %s → %s", prevMsg, shift.NewTopic)
}
```

### Step 3: Add Fallback Fields to Models

**File**: `moly-go/models/agent_types.go`

Add to `PrincipleAnalysis` struct:
```go
type PrincipleAnalysis struct {
    SignalsDetected []string
    Confidence      float64
    // NEW FIELDS:
    Fallback        bool    // true if this is a fallback due to timeout
    Reason          string  // why fallback: "timeout", "error", etc.
}
```

Add to `SubjectShift` struct:
```go
type SubjectShift struct {
    Detected bool
    NewTopic string
    // NEW FIELDS:
    Fallback bool    // true if timeout/error
    Reason   string
}
```

### Step 4: Add Timeout Context to Analyzer Methods

**File**: `moly-go/agents/subject_shift_detector.go`

Add new method alongside existing `DetectShift()`:

```go
// DetectShiftWithContext is cancellable version with timeout support
func (ssd *SubjectShiftDetector) DetectShiftWithContext(ctx context.Context, prevMsg string, currMsg string) (*models.SubjectShift, error) {
    // Create a channel for the result
    resultChan := make(chan *models.SubjectShift)
    errChan := make(chan error)
    
    // Run detection in goroutine
    go func() {
        result := ssd.DetectShift(prevMsg, currMsg)
        resultChan <- result
    }()
    
    // Wait for result or context timeout
    select {
    case result := <-resultChan:
        return result, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

### Testing Procedure

**Test Case 1: Principle Detection Timeout**
```
Before: 
  [ConversationAgent] Layer 6-7: Principle engagement detection failed: ..., falling back to no concerns

After (with fallback):
  [ConversationAgent] Layer 6-7: Principle detection failed/timed out, assuming no concerns
  [Response Metadata] principle_fallback: true
  → Response continues without principle concerns
```

**Test Case 2: Subject Shift Timeout**
```
Before:
  Message times out, response delayed

After (with fallback):
  [ConversationAgent] Subject shift detection timed out, assuming no shift
  [Response Metadata] subject_shift_fallback: true
  → Response continues without shift analysis
```

**Test Case 3: Processing Completes Despite Timeouts**
```
Message processing: 10-15 min (BEFORE - some analyses skip due to timeout)
Message processing: 5-8 min (AFTER - timeouts handled gracefully, response still generates)
```

### Success Criteria

- ✅ Principle detection timeout doesn't crash processing
- ✅ Subject shift timeout doesn't crash processing
- ✅ System returns response even if some analyses timeout
- ✅ Fallback marked in metadata for monitoring
- ✅ User gets response instead of error/timeout

---

# WEEK 2: PERFORMANCE OPTIMIZATIONS (3-5 hours)

## SOLUTION 1B: Phase-Aware Gap Threshold
**File**: `moly-go/main.go`  
**Lines**: 1311-1318  
**Effort**: 1-2 hours  
**Risk**: Low

### Current Code

```go
hasSignificantGaps := len(gaps) > 2
if hasSignificantGaps {
    log.Printf("[MessageProcessor] ⚡ OPTIMIZATION: Significant gaps detected (%d), ConversationAgent will ask clarification - skipping some processing", len(gaps))
}
```

### New Code (Phase-Aware)

Replace with:

```go
// Phase-aware gap threshold: Earlier phases more permissive
// Discovery (msg 1-2): >5 gaps triggers clarification
// Gathering (msg 3-5): >3 gaps triggers clarification  
// Analysis (msg 6+): >2 gaps triggers clarification
var gapThreshold int
currentPhase := ""

if len(conversationHistory) <= 2 {
    gapThreshold = 5
    currentPhase = "discovery"
} else if len(conversationHistory) <= 5 {
    gapThreshold = 3
    currentPhase = "gathering"
} else {
    gapThreshold = 2
    currentPhase = "analysis"
}

hasSignificantGaps := len(gaps) > gapThreshold

if hasSignificantGaps {
    log.Printf("[MessageProcessor] ⚡ OPTIMIZATION: Significant gaps detected (phase=%s, threshold=%d, actual=%d), ConversationAgent will ask clarification",
        currentPhase, gapThreshold, len(gaps))
} else if len(gaps) > 0 {
    log.Printf("[MessageProcessor] Context gaps identified: %v (phase=%s, threshold=%d, quality: %s) - NOT triggering gap workflow",
        gaps, currentPhase, gapThreshold, contextQuality)
}
```

### Testing Procedure

**Test Case 1: First Message (Discovery Phase)**
```
Input: "Hello Moly"
Gaps: [pastIntention, relevantReflections, recentSafetyIncidents] = 3 gaps
Threshold (discovery): >5
Result: hasSignificantGaps = false
Expected: Greeting acknowledged, NO gap clarification

Before Fix: hasSignificantGaps = true (3 > 2)
After Fix:  hasSignificantGaps = false (3 NOT > 5)
```

**Test Case 2: Third Message (Gathering Phase)**
```
Gaps: 3
Threshold (gathering): >3
Result: hasSignificantGaps = false (NOT significant yet)
Expected: Continue with conversation
```

**Test Case 3: Seventh Message (Analysis Phase)**
```
Gaps: 2
Threshold (analysis): >2
Result: hasSignificantGaps = false (2 is exactly at threshold)
Expected: Full analysis proceed
```

### Success Criteria

- ✅ First message never triggers gap clarification
- ✅ Phase progression reflected in gap sensitivity
- ✅ Log shows phase + threshold + actual gaps
- ✅ Gaps > threshold triggers clarification (as intended)

---

## SOLUTION 2B: Analysis Result Caching
**File**: `moly-go/models/agent_types.go`  
**Lines**: 80-110 (AnalysisContext struct)

**Effort**: 2-3 hours  
**Risk**: Low

### Step 1: Add Cache Fields to AnalysisContext

Find struct definition (~line 80):

```go
type AnalysisContext struct {
    Gaps                    []string
    ContextQuality          string
    IsFirstMessageInConversation bool
    UserProfile            *AboutMe
    ConversationHistory    []HistoryMessage
    // ... existing fields
}
```

Add new fields after existing ones:

```go
type AnalysisContext struct {
    // ... existing fields ...
    
    // Cache fields - populated once, reused by all analyzers
    CachedClarity           *models.ClarityAnalysis    `json:"cached_clarity,omitempty"`
    CachedPrincipleSignals  []string                   `json:"cached_principle_signals,omitempty"`
    CachedSubjectShift      *models.SubjectShift       `json:"cached_subject_shift,omitempty"`
    CachedEntities          []models.ExtractedEntity   `json:"cached_entities,omitempty"`
    CachedIntentAnalysis    *models.IntentAnalysis     `json:"cached_intent_analysis,omitempty"`
}
```

### Step 2: Populate Cache When Building Context

**File**: `moly-go/main.go`, where AnalysisContext is built (~line 1329)

After building context:

```go
// Build AnalysisContext
analysisCtx, buildErr := srv.analysisContextBuilder.BuildAnalysisContext(...)
if buildErr == nil && analysisCtx != nil {
    // Pre-populate cache to avoid redundant LLM calls
    
    // Cache entity extraction result (from earlier processing)
    if intents != nil {
        analysisCtx.CachedEntities = intents.Entities
        analysisCtx.CachedIntentAnalysis = intents
        log.Printf("[MessageProcessor] ✓ Cached entity extraction (%d entities) in AnalysisContext", len(intents.Entities))
    }
    
    // Other caches will be populated on-demand by analyzers
}
```

### Step 3: Update Clarity Analyzer to Check Cache

**File**: `moly-go/agents/message_clarity_analyzer.go`

Find method:
```go
func (mca *MessageClarityAnalyzer) AnalyzeClarity(msg string, history []string) (*ClarityAnalysis, error)
```

Update to use context parameter (if available):

```go
func (mca *MessageClarityAnalyzer) AnalyzeClarityWithCache(ctx context.Context, msg string, history []string, cached *models.ClarityAnalysis) (*models.ClarityAnalysis, error) {
    // Check cache first
    if cached != nil {
        log.Printf("[MessageClarityAnalyzer] Using cached clarity analysis")
        return cached, nil
    }
    
    // Not in cache, run full analysis
    return mca.AnalyzeClarity(msg, history)
}
```

### Step 4: Update ConversationAgent to Use Cache

**File**: `moly-go/agents/conversation_agent.go`, in Run() method (~line 560)

Find:
```go
clarity := ca.clarityAnalyzer.AnalyzeClarity(userMessage, ctx.ConversationHistory)
```

Change to:

```go
// Check cache first
var clarity *models.ClarityAnalysis
if ctx.CachedClarity != nil {
    clarity = ctx.CachedClarity
    log.Printf("[ConversationAgent] Using cached clarity analysis")
} else {
    clarity = ca.clarityAnalyzer.AnalyzeClarity(userMessage, ctx.ConversationHistory)
    // Populate cache for future use
    if ctx != nil {
        ctx.CachedClarity = clarity
    }
}
```

Do the same for:
- Subject shift detection → `ctx.CachedSubjectShift`
- Principle signals → `ctx.CachedPrincipleSignals`
- Entity extraction → `ctx.CachedEntities`

### Testing Procedure

**Test Case 1: Cache Hit**
```
Before:
  [MessageClarityAnalyzer] Starting LLM-driven analysis
  [Ollama] Calling mistral
  → Takes 1-2 minutes

After:
  [MessageClarityAnalyzer] Using cached clarity analysis
  → Takes <1 millisecond
```

**Test Case 2: Cache Population**
```
During message processing:
  1. Context extracted → entities cached
  2. Clarity analyzed → clarity cached
  3. Subject shift checked → shift cached
  
Later in same request:
  If any analyzer needs same data → uses cache
  Log: "Using cached clarity analysis"
```

### Success Criteria

- ✅ Analysis results cached in AnalysisContext
- ✅ Downstream analyzers check cache before LLM call
- ✅ Logs show "Using cached..." when cache hit
- ✅ Redundant LLM calls eliminated in same request

---

# WEEK 3: SELF-AWARENESS (4-6 hours)

## SOLUTION 4A: Reserved Moly System Contact
**Files**:
- `moly-go/database/migrations/017_add_system_moly_contact.sql` (NEW)
- `moly-go/auth/auth.go` (update registration)
- `moly-go/main.go` (update message processor)

**Effort**: 3-4 hours  
**Risk**: Low (additive)

### Step 1: Create Migration (NEW FILE)

**File**: `moly-go/database/migrations/017_add_system_moly_contact.sql`

```sql
-- Migration: Add reserved system contact for Moly self-awareness
-- Date: 2026-09-28

-- Ensure system_moly contact exists for each user
-- Run once per existing user, then auto-create on registration

-- This contact tracks the user's relationship with the system itself
-- Metadata: greeting frequency, relationship phase, average tone, etc.

-- For new users (created after this migration):
-- The AuthHandler will auto-create this contact during registration

-- For existing users:
-- This migration will create it. If it already exists, it's ignored.

INSERT OR IGNORE INTO contacts (
  id,
  user_id, 
  name,
  relationship,
  characteristics,
  created_at,
  updated_at
)
SELECT
  'system_moly_' || users.id,
  users.id,
  'Moly',
  'system_coach',
  json('{"greeting_count": 0, "relationship_phase": "new", "avg_tone": "neutral", "last_greeted_at": null}'),
  strftime('%s', 'now'),
  strftime('%s', 'now')
FROM users
WHERE id NOT IN (
  SELECT user_id FROM contacts WHERE name = 'Moly' AND relationship = 'system_coach'
);

-- Index for fast lookup of system_moly contacts
CREATE INDEX IF NOT EXISTS idx_system_moly_contacts ON contacts(user_id, name)
WHERE relationship = 'system_coach' AND name = 'Moly';
```

### Step 2: Update Auth Handler (auth/auth.go)

Find registration handler, around line where user is created:

```go
// After user is inserted into database:
err = db.Exec(
    "INSERT INTO users (id, email, username, password_hash, created_at) VALUES (?, ?, ?, ?, ?)",
    userID, email, username, hashPassword(password), time.Now().Unix()
)
```

Add after user creation:

```go
// Create reserved system_moly contact for self-awareness tracking
molyContactID := fmt.Sprintf("system_moly_%s", userID)
_, err = db.Exec(`
    INSERT INTO contacts (id, user_id, name, relationship, characteristics, created_at, updated_at)
    VALUES (?, ?, 'Moly', 'system_coach', ?, ?, ?)
`,
    molyContactID,
    userID,
    `{"greeting_count": 0, "relationship_phase": "new", "avg_tone": "neutral", "last_greeted_at": null}`,
    time.Now().Unix(),
    time.Now().Unix(),
)

if err != nil {
    log.Printf("[Auth] Warning: Failed to create system_moly contact for user %s: %v", userID, err)
    // Don't fail registration, continue
} else {
    log.Printf("[Auth] ✓ Created system_moly contact for user %s", userID)
}
```

### Step 3: Update Message Processor to Track Interactions

**File**: `moly-go/main.go`, where greeting is detected

Find where `IntentGreet` is detected (~line 1430):

```go
if intentAnalysis.Intent == "greeting" {
    log.Printf("[MessageProcessor] User greeted the system")
    
    // NEW: Update system_moly contact metadata
    molyContactID := fmt.Sprintf("system_moly_%s", userID)
    
    updateErr := srv.database.UpdateContact(molyContactID, map[string]interface{}{
        "greeting_count":     incrementGreetingCount(),  // Helper function
        "relationship_phase":  "established",             // Upgrade from "new"
        "last_greeted_at":    time.Now().Unix(),
        "avg_tone":           calculateAverageTone(),     // Helper function
    })
    
    if updateErr != nil {
        log.Printf("[MessageProcessor] Warning: Failed to update moly contact: %v", updateErr)
    } else {
        log.Printf("[MessageProcessor] ✓ Updated system_moly contact for user %s", userID)
    }
}
```

### Step 4: Add Helper Functions

**File**: `moly-go/main.go`, add new functions:

```go
// incrementGreetingCount loads current count and increments
func incrementGreetingCount(db *database.Database, molyContactID string) int {
    var currentCount int
    err := db.GetConnection().QueryRow(
        "SELECT COALESCE(characteristics->>'greeting_count', '0') FROM contacts WHERE id = ?",
        molyContactID,
    ).Scan(&currentCount)
    
    if err != nil {
        return 1
    }
    return currentCount + 1
}

// calculateAverageTone analyzes recent messages to determine average tone
func calculateAverageTone(db *database.Database, userID string, messages int) string {
    // Analyze last N messages for tone
    // Simple heuristic: exclamation marks = excited, questions = curious, etc.
    // For now, return "neutral" as safe default
    return "neutral"
}
```

### Step 5: Recall Moly Contact in Responses

**File**: `moly-go/agents/conversation_agent.go`, in Run() method

After loading user contacts:

```go
// Load system_moly contact for self-awareness metadata
molyContactID := fmt.Sprintf("system_moly_%s", userID)
molyContact, err := srv.database.GetContact(molyContactID)

if err == nil && molyContact != nil {
    greetingCount := molyContact.Characteristics["greeting_count"]
    relationshipPhase := molyContact.Characteristics["relationship_phase"]
    
    log.Printf("[ConversationAgent] System contact loaded: greetings=%v, phase=%v",
        greetingCount, relationshipPhase)
    
    // Store in response metadata for potential use in response generation
    response.Metadata["system_relationship"] = map[string]interface{}{
        "greeting_count": greetingCount,
        "phase": relationshipPhase,
    }
}
```

### Testing Procedure

**Test Case 1: System Contact Creation**
```
During registration:
Expected log:
  [Auth] ✓ Created system_moly contact for user user_123

Database:
  SELECT * FROM contacts WHERE name = 'Moly' AND user_id = 'user_123'
  → Returns: id=system_moly_user_123, relationship=system_coach
```

**Test Case 2: Greeting Tracked**
```
User message: "Hello Moly"
System detects greeting (IntentGreet)
Expected log:
  [MessageProcessor] User greeted the system
  [MessageProcessor] ✓ Updated system_moly contact for user user_123

Database after:
  SELECT characteristics FROM contacts WHERE id = 'system_moly_user_123'
  → greeting_count: 1, relationship_phase: "established"
```

**Test Case 3: Multiple Greetings Tracked**
```
User message 1: "Hello Moly" → greeting_count = 1
User message 5: "Hi there Moly" → greeting_count = 2
User message 10: "Hey Moly" → greeting_count = 3

System can respond: "You've greeted me 3 times now!"
```

### Success Criteria

- ✅ Migration creates system_moly for existing users
- ✅ New users get system_moly created at registration
- ✅ Greetings increment greeting_count
- ✅ Relationship phase upgrades from "new" to "established"
- ✅ Moly contact can be recalled in future responses

---

## SOLUTION 4B: Self-Reference in System Prompt
**File**: `moly-go/agents/conversation_agent.go`  
**Lines**: 1720-1750 (buildAdaptiveSystemPrompt)

**Effort**: 1-2 hours  
**Risk**: Low

### Current Code (Lines 1720-1750)

```go
// STEP 0: Session awareness - adjust greeting strategy
log.Printf("[ConversationAgent] [Layer 1] Building adaptive system prompt...")

responseGuidance = " They're greeting you. Respond warmly and briefly with a simple acknowledgment. No questions, no over-analysis. Just say hello back."
```

### Updated Code

Replace with comprehensive self-reference guidance:

```go
// STEP 0: Session awareness - adjust greeting strategy
log.Printf("[ConversationAgent] [Layer 1] Building adaptive system prompt...")

// Check if message contains self-reference (addressing Moly directly)
hasSelfReference := false
if ctx.CachedEntities != nil {
    for _, entity := range ctx.CachedEntities {
        if entity.Type == "self_reference" && entity.Confidence >= 0.8 {
            hasSelfReference = true
            break
        }
    }
}

// Also check intent for greeting
isGreeting := false
if ctx.CachedIntentAnalysis != nil {
    isGreeting = ctx.CachedIntentAnalysis.Intent == "greeting"
}

// Add self-reference guidance
if hasSelfReference {
    responseGuidance = ` You are Moly, a communication coach and thinking partner.
    
You are being addressed directly (self-reference detected). Show that you recognize this:
- Acknowledge the direct address
- Use first-person: "I'm here to help", "I think...", "I notice..."
- Be personal and warm, not clinical
- Show personality and genuine engagement

Examples of direct address:
- User: "Hello Moly" → Response: "Hello! Nice to see you."
- User: "Moly, what do you think?" → Response: "I think... [thoughtful response]"
- User: "I'm talking to Moly here..." → Response: "I'm listening. Tell me more."
- User: "Thank you Moly" → Response: "You're welcome! [acknowledge gratitude]"

Key principle: Direct address means the user sees you as a person/coach, not just a service. 
Respond with warmth and personality.`

    if isGreeting {
        responseGuidance += ` Additionally, this is a GREETING. Acknowledge it warmly and simply first.
        
Greeting Response Pattern:
1. First: Greet back warmly ("Hello!", "Hi there!", "Good to see you!")
2. Optional: Brief acknowledgment of intent
3. Skip: Analysis, questions, or over-explanation

IMPORTANT: Greetings are not prompts for context gathering. 
Just greet the person back, then optionally continue the conversation naturally.`
    }
} else if isGreeting {
    // Greeting without direct self-reference
    responseGuidance = ` This is a GREETING. Respond warmly and simply with acknowledgment.
    
Greeting Response Pattern:
1. Simple warm greeting back ("Hello!", "Hi!", "Good to see you!")
2. Optional: Natural continuation of conversation
3. NO: Questions about context, clarification needs, or analysis

Just greet them back. Don't overthink it.`
}

log.Printf("[ConversationAgent] Self-reference guidance: has_self_ref=%v, is_greeting=%v", 
    hasSelfReference, isGreeting)
```

### Testing Procedure

**Test Case 1: Direct Address + Greeting**
```
Input: "Hello Moly"
Detected: self_reference=true, greeting=true
System prompt includes:
  "You are Moly, a communication coach..."
  "Direct address means..."
  "This is a GREETING. Respond warmly..."
Expected response: "Hello! Nice to see you."
```

**Test Case 2: Direct Address Without Greeting**
```
Input: "Moly, I need advice about relationships"
Detected: self_reference=true, greeting=false
System prompt includes:
  "You are Moly..."
  "Direct address means..."
Expected response: Personal, warm engagement on topic
```

**Test Case 3: Greeting Without Direct Address**
```
Input: "Hi" (without mentioning Moly)
Detected: self_reference=false, greeting=true
System prompt includes:
  "This is a GREETING..."
  "Simple warm greeting back"
Expected response: Warm greeting, no analysis
```

### Success Criteria

- ✅ Self-reference detected → Shows personality
- ✅ Greeting detected → Simple acknowledgment
- ✅ System prompt reflects both conditions
- ✅ Responses feel natural, not robotic
- ✅ User feels acknowledged as person talking to person

---

# WEEK 4+: FUTURE IMPROVEMENTS

## SOLUTION 2A: Unified Analysis Engine (6-8 hours)
Consolidate 9 LLM calls into 3-4 calls.
Can be implemented after Week 1-3 critical fixes stabilize system.

## SOLUTION 5A: Maturity-Aware Response Progression (4-5 hours)
Progressive response complexity based on conversation phase.
Can be implemented as enhancement after core fixes.

---

# TESTING & VERIFICATION CHECKLIST

## Pre-Implementation

- [ ] Pull latest main branch
- [ ] Ensure all tests pass: `go test ./...`
- [ ] Build backend: `go build ./...`
- [ ] Build frontend: `npm run build` (moly-extension)
- [ ] Database migrations up to date

## Week 1 Testing

- [ ] **Solution 1A**: Test greeting (Hello Moly) → simple ack, no gap clarification
- [ ] **Solution 3A**: Test timeout detection on startup
- [ ] **Solution 3B**: Test LLM timeout → graceful fallback (not crash)

## Week 2 Testing

- [ ] **Solution 1B**: Test first message (1-2 messages) → gaps not significant
- [ ] **Solution 2B**: Test cache population → logs show "Using cached"

## Week 3 Testing

- [ ] **Solution 4A**: Verify system_moly contact created
- [ ] **Solution 4A**: Test greeting tracking → greeting_count increments
- [ ] **Solution 4B**: Test self-reference → responses show personality

## Integration Testing

- [ ] End-to-end: New user registration → greeting → response
- [ ] Message processing: First 5 messages → workflow progression correct
- [ ] Timeout handling: Old hardware → no cascading failures
- [ ] Performance: Message processing < 5-10 min (vs 10-15 min before)

## Deployment Checklist

- [ ] All commits have meaningful messages
- [ ] Database migrations applied (dev)
- [ ] Run full test suite
- [ ] Test on slow hardware simulation (if available)
- [ ] Monitor logs for timeouts, fallbacks
- [ ] Collect metrics on processing time

---

# GIT COMMIT STRATEGY

## Commit Organization

```
Commit 1: Solution 1A - Intent-first workflow (30 min)
Commit 2: Solution 3A - Tiered timeout strategy (20 min)
Commit 3: Solution 3B - Graceful fallback on timeout (30 min)
[Week 1 complete - test thoroughly]

Commit 4: Solution 1B - Phase-aware gap threshold (30 min)
Commit 5: Solution 2B - Analysis result caching (45 min)
[Week 2 complete - test thoroughly]

Commit 6: Solution 4A - System Moly contact (60 min)
Commit 7: Solution 4B - Self-reference guidance (30 min)
[Week 3 complete - test thoroughly]
```

## Commit Messages

Each commit should explain:
1. **What**: The specific solution number and name
2. **Why**: The architectural issue being fixed
3. **How**: Brief description of approach
4. **Test**: How to verify it works

Example:
```
Solution 1A: Fix workflow priority - Intent-first over gaps

When greeting detected (0.95 confidence), system should acknowledge warmly,
not override with gap clarification. Reordered workflow selection so
high-confidence intents (>=0.85) win over gap detection.

This fixes self-awareness: "Hello Moly" now receives simple greeting ack,
then optionally asks gaps as follow-up.

Test: Send "Hello Moly" → should receive "Hello! Nice to see you."
      NOT "What topic are you hoping to discuss?"

Fixes investigation Issue #1: Workflow Priority Bug
```

---

# SUCCESS CRITERIA - FINAL VALIDATION

After implementing all Week 1-3 solutions, the system should:

✅ **Self-Awareness**
- Greeting "Hello Moly" → Simple warm greeting back
- No gap clarification override of greeting
- System_moly contact created and tracked
- Can reference greeting history in future conversations

✅ **Performance**
- Message processing: 5-8 minutes (vs 10-15 before)
- No cascading timeouts on slow hardware
- Graceful fallback on LLM timeout (returns response, not error)
- Hardware profile detected and logged on startup

✅ **Workflow**
- First message: Gap threshold >5 (vs >2)
- Second-third message: Gap threshold >3
- Fourth+ message: Gap threshold >2
- Gaps become follow-up questions, not primary response

✅ **Architecture**
- Analysis results cached in AnalysisContext
- Timeouts match hardware capability
- Intent detection respected over gap detection
- System personality in responses (self-reference handling)

---

# NOTES FOR TOMORROW

1. **Start Early**: Solution 1A (Intent-first workflow) is most critical
2. **Test After Each**: Don't batch 3 solutions without testing
3. **Database Migration**: Run migration 017 before Solution 4A
4. **Hardware Profile**: Test on actual slow hardware if possible
5. **Logs**: Enable verbose logging to verify all solutions working
6. **Commit Often**: Small commits easier to revert if needed
7. **Monitor**: Watch for new errors in logs as each solution added

---

**Document Complete**  
Ready for implementation Sept 30, 2026  
Total estimated effort: 8-13 hours  
Distributed across 3 weeks (3 hours + 3 hours + 3 hours + contingency)

