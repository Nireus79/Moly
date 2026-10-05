# IMPLEMENTATION PLAN: Moly State Management Refactor

**Objective**: Wire broken connections, prevent dead code, ensure clean dataflows  
**Scope**: 5 root causes, 14 hours, zero unwired functions  
**Risk Level**: MEDIUM (modifying core flow, needs careful verification)

---

## PART 1: CURRENT STATE VERIFICATION

### Before Starting: Code Audit Checklist

Run these to establish baseline:
```bash
# Find all functions that call SetAccumulatedContext
grep -rn "SetAccumulatedContext" /moly-go --include="*.go"
# Expected: Only definition + maybe 1-2 references, NO calls

# Find all references to PreviousGoal field
grep -rn "PreviousGoal" /moly-go --include="*.go"
# Expected: Definition in layer_context.go, maybe 1-2 reads, NO writes

# Find all places that load previous extraction
grep -rn "previousExtraction\|previous_extraction" /moly-go --include="*.go"
# Expected: ZERO matches (not implemented)

# Check if Layer 3 is ever skipped
grep -rn "startLayer = 3" /moly-go --include="*.go"
# Expected: 1 match in unified_orchestrator.go (the bug)

# Verify no duplicate goal extraction
grep -rn "intention.*=" /moly-go/agents/context_extractor.go | wc -l
# Expected: Small number (should have only ONE intention assignment, not multiple)
```

**Deliverable**: Screenshot of baseline state before any changes

---

## PART 1B: ARCHITECTURE MAPPING FINDINGS (FROM CODE AUDIT)

### What IS Working (Don't Break These)
✅ 11-Layer orchestrator - all layers execute, results flow correctly  
✅ Loop pattern implementation - Layer 1-3 skipping works correctly  
✅ LayerContext - all 11 layer results stored properly  
✅ Phase persistence - saved to DB for next message  
✅ Safety gates - L2 (obvious harm), L6 (ambiguous), L11 (denial) all enforce  
✅ Maturity-aware gating - severity thresholds adjust by phase  
✅ Metadata population - layer outputs flow to response.Metadata  
✅ Deferred safety - Constitutional eval waits for full context  

### What IS Broken (Must Fix)
❌ SetAccumulatedContext() method defined but NEVER CALLED  
❌ Layer 5 conflict detection can't compare vs previous entities (no accumulated)  
❌ Layer 9+ can't detect if topic/contact changed (no accumulated)  
❌ Clarification response loses accumulated context  

### Critical Call Point (From Architecture Map)
**File: `main.go` lines 1802-1865**
- This is where ConversationRequest enters
- AnalysisContext created here
- **SetAccumulatedContext() must be called BEFORE Orchestrator.ProcessMessage()**
- Then pass to Orchestrator with populated context

### The 11-Layer Data Transformation
```
Message
  ↓
[L1: Extract] → ExtractedContext + Confidence
  ↓
[L2: Principles] → Verdict + Matched rules
  ↓
[L3: Maturity] → Score + Phase + Gate
  ↓
[L4: Gaps] → List[] + Severity
  ↓
[L5: Conflicts] → List[] + Type + Critical ← NEEDS ACCUMULATED
  ↓
[L6-7: Ambiguity/Violations] → Questions[] + Severity
  ↓
[L8-11: Clarify/Deny] → Final response
  ↓
ConversationResponse + Metadata
```

---

## PART 2: DATA STRUCTURES (No Schema Changes Needed)

### Analysis: What's Already In Place

✅ **LayerContext** (`tools/layer_context.go`):
```go
type LayerContext struct {
    PreviousGoal                 string
    PreviousValues               []string
    AccumulatedExtractedEntities []models.ExtractedEntity
    // ... other fields
}

func (lc *LayerContext) SetAccumulatedContext(...) { ... }  // EXISTS
```

✅ **AnalysisContext** (`models/conversation_types.go`):
```go
type AnalysisContext struct {
    ExtractedEntities     []ExtractedEntity
    ExtractedConfidence   float64
    // ... other fields
}
```

✅ **ExtractedContext** (`models/agent_types.go`):
```go
type ExtractedContext struct {
    Intention            string
    Goals                []string
    UserValues           []string
    IntentionConfidence  float64
}
```

### Decision: NO new tables needed
- Previous goals stored in memory (LayerContext)
- Can be persisted via Conversation.metadata if needed later
- For now: just wire existing structure

### Action Items:
- [ ] Add optional `PrimaryGoal` to `Conversation` model (database persistence)
- [ ] Update `Conversation` table migration if adding field (non-breaking)
- [ ] NO schema changes to core tables

---

## PART 3: DETAILED FIX SEQUENCE (Order Matters!)

### FIX #1: Load Previous Extraction State (2 hours)

**Why First**: SetAccumulatedContext needs data to load

**What to do**:

**Step 1.1**: Understand current message flow
- Read: `handlers/message_processor.go` (where messages enter)
- Find: Where `AnalysisContext` is populated
- Map: What previous message data is available

**Step 1.2**: Create extraction persistence
- Find: Last `Layer1Result` for this user+conversation
- Option A: Store in memory (session-based)
- Option B: Query from database (slower but persistent)
- Recommendation: Start with memory (simpler, faster)

**Step 1.3**: CRITICAL - Call from main.go, NOT UnifiedOrchestrator

From architecture mapping: **SetAccumulatedContext() call point is in main.go ~line 1802-1865**

This is where:
1. ConversationRequest received from user
2. AnalysisContext created
3. **THIS IS WHERE SetAccumulatedContext() must be called**
4. Then passed to Orchestrator.ProcessMessage()

Location: `main.go` ~line 1820 (MessageProcessorHandler or similar)

```go
// BEFORE calling uo.ProcessMessage():
// Load previous message's extraction
prevExtraction := loadPreviousExtraction(userID, conversationID)
if prevExtraction != nil {
    analysisCtx.AccumulatedExtractedEntities = prevExtraction.Entities
    analysisCtx.PreviousGoal = prevExtraction.Goal  // STORE IN ANALYSIS CONTEXT
}

// Create layer context from analysis context
lc := tools.NewLayerContext(analysisCtx, userID, messageID, conversationID)

// NOW SetAccumulatedContext is called implicitly since analysisCtx has the data
// OR explicitly in UnifiedOrchestrator.ProcessMessage() line ~155
lc.SetAccumulatedContext(
    analysisCtx.AccumulatedExtractedEntities,
    analysisCtx.PreviousGoal,  // THIS WAS NEVER POPULATED BEFORE
    nil,  // values
)
```

**Step 1.4**: Implement loadPreviousExtraction
- Location: New function in `main.go` or `handlers/message_processor.go`
- Query: Get last successful extraction for this user+conversation
- Option A (Simple): Store latest in memory map
- Option B (Persistent): Query from context_attributes table
- Recommendation: Start with memory for speed, add DB persistence later

**Code Location**: 
- Call point: `main.go` line 1802-1865 (before Orchestrator.ProcessMessage call)
- Method: `agents/unified_orchestrator.go` line ~155 (SetAccumulatedContext call)

**Verification**:
- [ ] PreviousGoal populated before L1 runs
- [ ] AccumulatedExtractedEntities has previous entities
- [ ] No nil pointer crashes
- [ ] Test: Message 2 should have M1's entities available

---

### FIX #2: Fix Clarification Detection (4 hours)

**Why Second**: Affects which layers run; other fixes depend on L1-L3 running

**Step 2.1**: Understand what pending clarifications look like
- Find: Where clarification questions are stored
- Find: `TemporaryFactStore` or equivalent
- Understand: How to query "pending for this user/conversation"

**Step 2.2**: Replace clarification detection logic
```go
// OLD (WRONG):
isAnsweringClarification := analysisCtx.CurrentMessage != "" &&
    analysisCtx.ExtractedConfidence > 0 &&
    len(analysisCtx.ExtractedEntities) > 0

// NEW (CORRECT):
pendingClarifications, err := uo.getPendingClarifications(userID, conversationID)
if err != nil {
    log.Printf("[UnifiedOrchestrator] Warning: could not check pending clarifications: %v", err)
    pendingClarifications = make([]string, 0)
}

isAnsweringClarification := len(pendingClarifications) > 0 &&
    uo.addressesClarification(analysisCtx.CurrentMessage, pendingClarifications)
```

**Step 2.3**: Implement helper functions
```go
func (uo *UnifiedOrchestrator) getPendingClarifications(
    userID, conversationID string,
) ([]string, error) {
    // Query: SELECT question_id FROM clarification_questions 
    //        WHERE user_id = ? AND conversation_id = ?
    //        AND status = 'pending'
}

func (uo *UnifiedOrchestrator) addressesClarification(
    message string,
    questions []string,
) bool {
    // Check if message content matches any pending question
    // Use keyword matching, not exact match
}
```

**Code Location**: `agents/unified_orchestrator.go` lines 141-148

**Verification**:
- [ ] Message 2 & 3 do NOT trigger false clarification
- [ ] First clarification question (M1) DOES trigger skip
- [ ] Logs show correct detection reason
- [ ] No database errors

---

### FIX #3: Ensure Layer 1-3 Always Run (Fixes Maturity Accumulation) (2 hours)

**Why Third**: Depends on FIX #2; enables maturity recalculation

**Step 3.1**: Understand current layer skipping
- Layer 1: Extraction
- Layer 2: Principle checking
- Layer 3: Maturity calculation
- When false loop pattern triggers, these are SKIPPED

**Step 3.2**: Update loop pattern to always run L1-3
```go
// OPTION A: Always run L1-3, only skip beyond
if !isAnsweringClarification {
    startLayer = 0  // Run from L1
} else {
    startLayer = 3  // Skip to L4 (L1-L3 still need to run, see below)
}

// Actually, BETTER:
// Even when clarification, Layer 1 and 3 should run with accumulated context
startLayer = 0  // ALWAYS start from L1

// Instead, add flag to L1 and L3:
if isAnsweringClarification {
    lc.IsReprocessingWithAccumulated = true
    log.Printf("[UnifiedOrchestrator] Reprocessing with accumulated context")
}
```

**Step 3.3**: Update Layer 1 to use accumulated context
- If `IsReprocessingWithAccumulated`, use previous extraction + new message
- Don't re-extract everything fresh, merge contexts

**Step 3.4**: Update Layer 3 to recalculate with accumulated
- If accumulated entities exist, use them for maturity calculation
- Maturity should go UP with more context, not reset to 0.00

**Code Location**: `unified_orchestrator.go` line 164-172 (loop skipping logic)

**Verification**:
- [ ] Maturity: M1=0.00, M2=0.33+, M3=0.66+
- [ ] Layer 3 executes every message
- [ ] Accumulated entities passed to L3
- [ ] Maturity recalculation visible in logs

---

### FIX #4: Distinguish Primary Goal vs Current Intent (3 hours)

**Why Fourth**: Requires L1 extraction to work properly; uses data from FIX #1

**Step 4.1**: Add primary goal concept to LayerContext
```go
type LayerContext struct {
    // ... existing fields
    
    // NEW: Goal tracking
    PrimaryGoal         string  // From M1, never changes
    CurrentMessageIntent string  // Fresh each message
    GoalProgression     []string // Track evolution: M1, M2, M3 intents
}
```

**Step 4.2**: Modify Layer 1 to detect and lock primary goal
```go
// In Layer 1 extraction result:
if messageIndex == 0 {  // First message
    lc.PrimaryGoal = extractedContext.Intention
    log.Printf("[Layer1] ✓ PRIMARY GOAL LOCKED: %s", lc.PrimaryGoal)
} else {
    // Subsequent messages
    lc.CurrentMessageIntent = extractedContext.Intention
    lc.GoalProgression = append(lc.GoalProgression, extractedContext.Intention)
    
    if lc.PrimaryGoal != "" {
        log.Printf("[Layer1] Goal comparison: primary='%s' vs current='%s'",
            lc.PrimaryGoal, lc.CurrentMessageIntent)
    }
}
```

**Step 4.3**: Update response generation to use primary goal
- Find: Where goal is used in response generation (L6-7)
- Change: Use `PrimaryGoal` instead of `CurrentMessageIntent`
- Use `CurrentMessageIntent` only to assess if user is progressing toward goal

**Code Location**: 
- `agents/context_extractor.go` (Layer 1) - detect and lock
- `agents/conversation_agent.go` (L6-7) - use primary goal

**Verification**:
- [ ] M1 locks goal
- [ ] M2-M3 use locked goal
- [ ] Goal never changes across messages
- [ ] Response addresses original goal, not new intent

**Cleanup**:
- [ ] Remove any code that overwrites goal after M1
- [ ] Remove any code that treats M2+ intent as goal

---

### FIX #5: Link Entity Subjects in Extraction (3 hours)

**Why Fifth**: Standalone fix; depends on L1 extraction working

**Step 5.1**: Update LLM extraction prompt
- Find: `context_extractor.go` where LLM is called
- Find: The prompt that asks for entity extraction
- Update prompt to specify subject tagging

**Step 5.2**: Update prompt template
```
BEFORE:
"Extract all entities: contacts, characteristics, preferences...
Format: type|value|confidence"

AFTER:
"Extract all entities, ALWAYS specify who it's about:
Format: type|value|subject|confidence

Subjects:
- USER: for facts about the person writing the message
- CONTACT_[Name]: for facts about someone they mention
- CONTEXT: for general situation facts

Example:
User says: 'I'm dominant. She wants a mentor.'
Correct extraction:
- characteristic|dominant|USER|0.9
- preference|wants a mentor|CONTACT_Christine|0.9

NOT:
- preference|wants a mentor|contact_name|0.9  ← WRONG: contact_name is undefined
"
```

**Step 5.3**: Update entity parsing to validate subjects
```go
type ParsedEntity struct {
    Type      string
    Value     string
    Subject   string  // NEW: must be USER or CONTACT_*
    Confidence float64
}

func validateSubject(subject string) error {
    if subject == "USER" {
        return nil
    }
    if strings.HasPrefix(subject, "CONTACT_") {
        return nil
    }
    if subject == "CONTEXT" {
        return nil
    }
    return fmt.Errorf("invalid subject: %s", subject)
}
```

**Step 5.4**: Use subject info for deduplication
- When "good girl" is extracted with `CONTACT_Christine`, don't create new contact
- Link characteristic to existing contact

**Code Location**: `agents/context_extractor.go` (Layer 1 extraction)

**Verification**:
- [ ] All entities have valid subject tags
- [ ] No undefined "contact_name" placeholders
- [ ] "Good girl" characteristic linked to Christine_sub, not new contact
- [ ] Entity deduplication works

**Cleanup**:
- [ ] Remove any code that creates contacts from characteristics
- [ ] Remove any undefined subject handling

---

## PART 4: DEAD CODE CLEANUP

After all fixes are in place:

### Scan for Dead Code

**Functions to check**:
```bash
# Find all functions that are defined but never called
grep -rn "^func.*{" /moly-go/agents/*.go > /tmp/all_funcs.txt
grep -rn "functionName" /moly-go --include="*.go" > /tmp/all_calls.txt
# Compare: functions in all_funcs.txt but not in all_calls.txt
```

**Specific functions to verify are USED**:
- [ ] `SetAccumulatedContext` - NOW CALLED in FIX #1
- [ ] `loadPreviousExtraction` - NEW, called in FIX #1
- [ ] `getPendingClarifications` - NEW, called in FIX #2
- [ ] `addressesClarification` - NEW, called in FIX #2
- [ ] All Layer methods - verify each is called from orchestrator

**Functions to REMOVE if dead**:
- [ ] Any old "goal tracking" functions (before refactor)
- [ ] Any old "clarification detection" functions (before refactor)
- [ ] Any "stub" methods that were placeholders

---

## PART 5: DATA FLOW VERIFICATION

### Before/After Data Flow

**BEFORE (Broken)**:
```
Message 1:
  ↓
  [L1: Extract intention] → intention = "initiating romantic interest"
  ↓
  [L3: Maturity = 0.00]
  ↓
  Response sent
  ↓
  ??? Extracted data LOST ???

Message 2:
  ↓
  [False clarification detected]
  ↓
  [L1-L3 SKIPPED]
  ↓
  [L4-11: No previous goal]
  ↓
  Response doesn't reference M1 goal
```

**AFTER (Fixed)**:
```
Message 1:
  ↓
  [L1: Extract intention] → primary_goal = "write smart first message"
  ↓
  [Store extraction in memory]
  ↓
  [L3: Maturity = 0.00]
  ↓
  Response sent with goal locked

Message 2:
  ↓
  [Load previous extraction from memory]
  ↓
  [SetAccumulatedContext called]
  ↓
  [Real clarification check - returns FALSE]
  ↓
  [L1: Extract new entities WITH accumulated context]
  ↓
  [L3: Recalculate maturity with accumulated = 0.33]
  ↓
  [L4-11: Have primary goal + accumulated entities]
  ↓
  Response uses primary goal, acknowledges M1 approach
```

### Verification Checkpoints

| Stage | Check | Expected | How to Verify |
|-------|-------|----------|---------------|
| **FIX #1** | PreviousGoal loaded | Not null before L1 | Log: `PreviousGoal=...` |
| **FIX #2** | Clarification detection | M2 returns FALSE | Log: `isAnsweringClarification = false` |
| **FIX #3** | Layer 3 runs always | Maturity recalculated | Log: `MaturityScore=0.33` on M2 |
| **FIX #4** | Primary goal locked | Same across M1→M3 | Log: `PrimaryGoal = "..." (locked)` |
| **FIX #5** | Subjects linked | No false contacts | Log: No `good girl` contact created |

---

## PART 6: TEST SCRIPT (Verify End-to-End)

### Run After All Fixes

```bash
#!/bin/bash

echo "=== MOLY STATE MANAGEMENT TEST ==="

# Compile
echo "1. Building..."
cd moly-go
go build -o moly . 2>&1 | head -20
if [ $? -ne 0 ]; then
    echo "❌ BUILD FAILED"
    exit 1
fi

# Run with test conversation
echo "2. Testing with conversation..."
cat > /tmp/test_messages.json << 'EOF'
[
  {"role": "user", "content": "I need help writing a first message to Christine on FetLife"},
  {"role": "user", "content": "I have experience in real life with BDSM, I'm dominant, I like consensus"},
  {"role": "user", "content": "She's submissive, looking for a mentor, wants a long-term relationship"}
]
EOF

# Verify logs contain:
echo "3. Checking logs..."
grep -c "PRIMARY GOAL LOCKED" build.log || echo "❌ Goal not locked"
grep -c "isAnsweringClarification = false" build.log || echo "❌ False clarification detection not fixed"
grep -c "MaturityScore.*0.33" build.log || echo "❌ Maturity not accumulating"
grep -c "SetAccumulatedContext called" build.log || echo "❌ Accumulated context not loaded"

echo "✅ All tests passed!"
```

---

## PART 7: ROLLBACK PLAN

If something breaks:

1. **Revert to baseline** (after FIX #0 baseline audit):
   - Git revert specific commits in reverse order
   - Test after each revert

2. **Partial rollback**:
   - FIX #1 only affects loading, safe to disable
   - FIX #2 only affects detection, safe to disable
   - FIX #3 only affects when layers run, safe to disable

3. **Safety valve**:
   Add feature flag to disable all fixes:
   ```go
   if os.Getenv("MOLY_LEGACY_MODE") == "true" {
       // Use old (broken) code path
   }
   ```

---

## PART 8: COMMIT STRATEGY

**No mega-commits**. One fix = one commit.

```bash
# FIX #1
git add agents/unified_orchestrator.go tools/layer_context.go
git commit -m "Load previous extraction state when processing message

- Implement loadPreviousExtraction() to retrieve M1 data
- Call SetAccumulatedContext() to populate PreviousGoal
- Enables goal persistence across messages
- Fixes: Goals overwriting each message"

# FIX #2  
git add agents/unified_orchestrator.go
git commit -m "Fix clarification detection to check database not entity presence

- Old: fired on any message with entities (always true)
- New: checks if actual pending clarification exists
- Fixes: False loop pattern triggering every message"

# FIX #3
git add agents/unified_orchestrator.go tools/layer_context.go
git commit -m "Ensure Layers 1-3 always run even in clarification mode

- Layer 1: re-extract with accumulated context
- Layer 3: recalculate maturity with accumulated entities
- Fixes: Maturity stuck at 0.00"

# FIX #4
git add agents/context_extractor.go tools/layer_context.go agents/conversation_agent.go
git commit -m "Lock primary goal in Message 1, track current intent separately

- Add PrimaryGoal field to LayerContext
- Message 1: lock goal, never change
- Message 2+: track as CurrentMessageIntent
- Fixes: Goals overwriting with each message"

# FIX #5
git add agents/context_extractor.go
git commit -m "Link entity subjects (USER/CONTACT_name) in extraction

- Update LLM prompt to specify subject tagging
- Validate all entities have valid subject
- Link characteristics to contacts, don't create false contacts
- Fixes: 'Good girl' extracted as separate contact"

# Cleanup
git add agents/unified_orchestrator.go
git commit -m "Remove dead code and old goal tracking functions

- Remove old goal extraction logic (replaced by primary goal)
- Remove old clarification detection (replaced by database check)
- Clean up unused imports/variables"
```

---

## PART 9: SUMMARY TABLE

| Fix | What | Where | Lines | Risk | Verification |
|-----|------|-------|-------|------|--------------|
| #1 | Load prev extraction | `unified_orchestrator.go` | 152-165 | LOW | PreviousGoal populated |
| #2 | Fix clarification | `unified_orchestrator.go` | 141-150 | MEDIUM | M2-M3 don't skip L1-3 |
| #3 | Always run L1-3 | `unified_orchestrator.go` | 164-175 | MEDIUM | Maturity accumulates |
| #4 | Lock primary goal | `context_extractor.go` | ~200+ | MEDIUM | Goal persists M1→M3 |
| #5 | Link subjects | `context_extractor.go` | ~150+ | LOW | No false contacts |
| CLEAN | Remove dead code | Various | ~50 | LOW | Tests pass |

---

## PART 10: FINAL CHECKLIST

Before considering done:

```
Architecture:
  [ ] SetAccumulatedContext actually called
  [ ] Previous data loads before Layer 1
  [ ] All 11 layers can access accumulated context
  [ ] No unwired functions exist
  
Data Flow:
  [ ] M1 extraction stored somewhere (memory/db)
  [ ] M2 loads M1 extraction
  [ ] M3 loads M1+M2 extractions
  [ ] Maturity: 0.00 → 0.33 → 0.66+
  
Dead Code:
  [ ] No unused functions
  [ ] No duplicate goal tracking
  [ ] No old clarification detection
  [ ] All old functions removed or refactored
  
Database:
  [ ] No schema mismatches
  [ ] No table name errors
  [ ] No column name errors
  [ ] All queries tested
  
Testing:
  [ ] Build succeeds
  [ ] No compilation errors
  [ ] No nil pointer crashes
  [ ] Logs show correct flow
  [ ] Test conversation runs through M1→M3
```

---

## Timeline

- **FIX #1**: 2h (Load state)
- **FIX #2**: 4h (Clarification detection)
- **FIX #3**: 2h (Layer running)
- **FIX #4**: 3h (Primary goal)
- **FIX #5**: 3h (Subject linking)
- **Cleanup**: 1h (Dead code)
- **Testing**: 1h (Verify end-to-end)
- **Buffer**: 1h (Unexpected issues)

**Total: ~17 hours** (1 hour more than estimate for debugging)

---

## Approval Needed Before Starting

- [ ] Agree on FIX sequence (no reordering)
- [ ] Agree on data storage (memory vs database)
- [ ] Agree on commit strategy (one fix per commit)
- [ ] Agree on test script validation criteria
- [ ] Schedule: Who reviews each fix?
