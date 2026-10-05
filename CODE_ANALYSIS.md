# MOLY CODE ANALYSIS: What System Actually Manages

**Date**: October 4, 2026  
**Method**: Direct code inspection, not assumptions  
**Finding**: System has capability structures but critical execution paths are broken/missing

---

## PART 1: WHAT SYSTEM CAN MANAGE (Defined But Not Implemented)

### Data Structures Exist For State Management

**LayerContext has these fields** (in `tools/layer_context.go`):
```go
AccumulatedExtractedEntities []models.ExtractedEntity  // All entities from previous messages
PreviousGoal                 string                    // Goal from previous message(s)
PreviousValues               []string                  // Values from previous message(s)
```

**SetAccumulatedContext method exists** (line 183-191 in `layer_context.go`):
```go
func (lc *LayerContext) SetAccumulatedContext(
    previousEntities []models.ExtractedEntity,
    previousGoal string,
    previousValues []string,
) {
    lc.AccumulatedExtractedEntities = previousEntities
    lc.PreviousGoal = previousGoal
    lc.PreviousValues = previousValues
}
```

**ExtractedContext can hold goals** (in `models/agent_types.go`):
```go
type ExtractedContext struct {
    Intention            string    // Single intention
    Goals                []string  // Multiple goals
    UserValues           []string  // User's expressed values
    IntentionConfidence  float64   // 0-1 confidence in intention
}
```

✅ **Capability EXISTS** to maintain previous goal, previous values, accumulated entities

---

## PART 2: WHY IT'S NOT WORKING - The Broken Link

### SetAccumulatedContext is NEVER CALLED

**Search result**:
```bash
$ grep -rn "SetAccumulatedContext" /moly-go --include="*.go"
/tools/layer_context.go:181:// SetAccumulatedContext sets context from previous messages
/tools/layer_context.go:183:func (lc *LayerContext) SetAccumulatedContext(
```

**FACT**: Only 2 matches - the definition and one reference. **Zero calls to this function exist in the codebase.**

**Impact**: 
- PreviousGoal is never loaded from previous message
- AccumulatedExtractedEntities is never populated
- Every message starts fresh with empty state

### The False Clarification Detection

**Location**: `unified_orchestrator.go` line 141-143:

```go
isAnsweringClarification := analysisCtx.CurrentMessage != "" &&
    analysisCtx.ExtractedConfidence > 0 &&
    len(analysisCtx.ExtractedEntities) > 0
```

**Logic**: If message has content + some extraction happened → assume it's a clarification

**Reality**:
- Message 1: Has entities → marked as clarification? (depends on AnalysisContext state)
- Message 2: Has entities → MARKED AS CLARIFICATION (even though it's new info)
- Message 3: Has entities → MARKED AS CLARIFICATION (even though it's profile data)

**What SHOULD check**:
- Is there a pending clarification question in database?
- Does current message address that specific question?
- Not just "are there entities"

### Loop Pattern Effect When False Fires

When `isAnsweringClarification = true`:
```go
startLayer = 3  // Skip Layers 0,1,2 (Layer 1,2,3)
```

**Skips**:
- ❌ Layer 1: Context Extraction - NO RE-EXTRACTION
- ❌ Layer 2: Principle Checking - not relevant
- ❌ Layer 3: Maturity Assessment - NO MATURITY RECALCULATION

**Result**:
- Fresh entities from current message never extracted with full pipeline
- Maturity recalculation never happens (stays 0.00)
- Previous context never loaded (SetAccumulatedContext never called)
- Layer 4+ only has current message data

---

## PART 3: DATA FLOW - WHERE INFORMATION GETS LOST

### Message 1 Processing

```
Input: User message with goal + contact + style
    ↓
Layer 1 extracts: 
  - Intention: "seek advice on initiating romantic interest"
  - Goals: ["compose first message", "be distinctive"]
  - Contact: "Christine_sub"
  - Style: "playful"
    ↓
Layer3 calculates: maturity = 0.00 (new conversation)
    ↓
Layers 6-7 generate: clarification questions
    ↓
ConversationResponse sent back
    ↓
??? Extracted data not persisted anywhere ???
    ↓
Conversation continues...
```

### Message 2 Processing

```
Input: User message with methodology (9-step approach, 18 entities)
    ↓
AnalysisContext created with CurrentMessage + some extraction
    ↓
unified_orchestrator checks: CurrentMessage != "" && ExtractedConfidence > 0
    → isAnsweringClarification = TRUE (false positive!)
    ↓
startLayer = 3 (skip L1, L2, L3)
    ↓
Layer 1 (extraction) SKIPPED - fresh message entities not extracted
Layer 3 (maturity) SKIPPED - maturity not recalculated
    ↓
Layer 4 runs with:
  - AccumulatedExtractedEntities: [] (EMPTY - SetAccumulatedContext never called)
  - PreviousGoal: "" (EMPTY - never loaded)
  - Current message data only
    ↓
Gap detection sees goal is unclear (because previous goal never loaded!)
    ↓
Response generation creates generic advice
    ↓
??? New entities, new goals still not persisted for next message ???
```

### Message 3 Processing

```
Same pattern repeats:
- isAnsweringClarification = TRUE (false positive)
- Layers 1-3 skipped
- Accumulated context empty
- Fresh goal extracted (different from previous)
- Generic response
- New goal overwritten again (not persisted)
```

---

## PART 4: WHAT'S ACTUALLY MANAGED vs. WHAT'S LOST

### ✅ What System CAN Extract & Store (Happens)

```
Per Message:
- Contacts (name, relationship)
- Style (casual, formal, playful)
- Characteristics (dominant, submissive)
- Entities (39+ entity types)
- Extraction confidence
- Constitutional violations

Per Conversation:
- Messages (stored in database)
- Reflections (if explicitly saved)
- Conversation history
```

### ❌ What System CANNOT Maintain (Not Implemented)

```
Across Messages:
- Original/Primary goal (extracted but not locked)
- Previous goals (structure exists, never loaded)
- Accumulated entities (structure exists, never loaded)
- Entity history (each message replaces, doesn't accumulate)
- Goal progression (should track if goal achieved/changed)
- Entity subjects (who owns each entity - USER vs CONTACT)

During Message Processing:
- Which clarification question was asked (not tracked)
- Did user answer it (not validated)
- What new info came in vs clarification response (not distinguished)
- Entity ownership mapping (extracted as "contact_name", not linked to actual contact)
```

---

## PART 5: THE CORE SYSTEM ISSUES

### Issue #1: Broken Clarification Detection Logic

**Current**: `if (message exists) AND (has entities) AND (confidence > 0) → clarification`

**Problem**: This fires on EVERY message after first

**Should be**: `if (pending_question in database) AND (message addresses it) → clarification`

**File**: `unified_orchestrator.go` line 141-143

**Fix Required**: Check database for pending clarifications, not just presence of entities

---

### Issue #2: SetAccumulatedContext Defined But Never Called

**Current**: Function exists, never invoked

**Problem**: Previous goal/values/entities never loaded

**Should be**: Called before running layers, populate from previous message's extraction

**Files**:
- Definition: `tools/layer_context.go` line 181-191
- Should be called: `unified_orchestrator.go` line ~152 (after LayerContext created)

**Fix Required**: After creating LayerContext, load previous message's extracted goal/entities and call SetAccumulatedContext

---

### Issue #3: Fresh Intention Extracted Every Message Without Checking Previous

**Current**: Layer 1 extracts Intention fresh each time, overwrites `ExtractedContext.Intention`

**Problem**: System forgets "write a first message" goal from M1 when M2 extracts new intention

**Should be**: 
- M1: Extract intention, mark as PRIMARY_GOAL (lock it)
- M2+: Extract intent, keep separate as CURRENT_MESSAGE_INTENT
- Compare: is current intent same as primary? Different? Progressive?

**Files**:
- Layer 1 extraction: `agents/context_extractor.go`
- Should distinguish: primary vs current intent

**Fix Required**: Add `primary_goal` field to conversation state, don't overwrite with new intent

---

### Issue #4: Entity Extraction Doesn't Link Subjects Properly

**Current**: LLM extracts entities without tagging owner (USER vs CONTACT)

**Example from logs**:
```
[SmartExtraction] characteristic|good girl|contact_name|0.90
```

Problem: `contact_name` is undefined - should be `Christine_sub`

**Result**: System creates false contact "Good girl" instead of linking to Christine_sub

**File**: `agents/context_extractor.go` (LLM prompt)

**Fix Required**: 
- Add subject context to LLM extraction prompt
- Validate all entities have subject attached
- Link "good girl" to existing contact

---

### Issue #5: Maturity Recalculation Requires Layer 3, But Layer 3 Skipped

**Current**: 
- Layer 3 calculates maturity
- Loop pattern skips Layer 3 on alleged clarifications
- Maturity = 0.00 every time

**Problem**: No maturity accumulation, stays immature forever

**File**: `unified_orchestrator.go` line 166-172 (layer skipping)

**Fix Required**: Even in loop pattern, Layer 3 must run to recalculate maturity with accumulated entities

---

## PART 6: PRIORITIZED IMPACT ASSESSMENT

### Impact Hierarchy (from logs evidence)

| Issue | Impact | Why | Affected |
|-------|--------|-----|----------|
| **False Clarification Detection** | CRITICAL | Breaks loop pattern, skips Layers 1-3 every time | ALL messages |
| **SetAccumulatedContext Not Called** | CRITICAL | Accumulated state never loads | Goal persistence impossible |
| **Maturity Stays 0.00** | HIGH | System never matures, treats M3 like M1 | Context quality assessment |
| **Fresh Intent Overwrites Previous** | HIGH | Goal lost after M1 | Goal tracking |
| **Entity Subjects Not Linked** | HIGH | Creates false contacts, corrupts linking | Entity deduplication |
| **Contradiction Detection After Generation** | MEDIUM | Prevention impossible, only detection | Response safety |

---

## PART 7: WHAT NEEDS TO HAPPEN (NOT WHAT'S MISSING)

### The System Doesn't Need New Components

The system already has:
- ✅ AnalysisContext to hold extraction results
- ✅ LayerContext to pass data between layers
- ✅ SetAccumulatedContext method (just not called)
- ✅ PreviousGoal, PreviousValues fields (just not populated)
- ✅ Layer 1 extraction (just skipped in loop pattern)
- ✅ Layer 3 maturity (just skipped in loop pattern)

### What Needs to Change

1. **Call SetAccumulatedContext** - Populate PreviousGoal/PreviousValues before layer execution
2. **Fix clarification detection** - Check database for actual pending questions
3. **Don't skip Layer 1-3** - Always extract and calculate maturity, even in loop pattern
4. **Distinguish intent types** - Primary goal vs. current message intent
5. **Bind entity subjects** - Tag each entity with USER or CONTACT owner
6. **Move contradiction checks upstream** - Before response generation

### The Real Question: Extract More or Better Management?

**Current extraction captures**:
- 39+ entity types ✅
- Confidence scores ✅
- Subject attribution (partially) ⚠️
- Negation preservation ✅
- Style, tone, contact, goals ✅

**The problem is not extraction scope** - the system extracts plenty of data.

**The problem is state management** - extracted data is not accumulated, compared, or prioritized across messages.

**Solution**: Better management of what's already extracted, not more extraction.

---

## PART 8: CONCRETE IMPLEMENTATION REQUIREMENTS

### Requirement #1: Actually Call SetAccumulatedContext

**Where**: `unified_orchestrator.go` ProcessMessage function, around line 152

**Before**:
```go
lc := tools.NewLayerContext(analysisCtx, userID, messageID, conversationID)
```

**After**:
```go
lc := tools.NewLayerContext(analysisCtx, userID, messageID, conversationID)

// Load previous message's context
if previousExtraction != nil {
    lc.SetAccumulatedContext(
        previousExtraction.Entities,
        previousExtraction.PrimaryGoal,
        previousExtraction.Values,
    )
}
```

**Dependency**: Need to load previous message's Layer1Result from somewhere (database or context)

---

### Requirement #2: Fix Clarification Detection

**Where**: `unified_orchestrator.go` line 141-143

**Before**:
```go
isAnsweringClarification := analysisCtx.CurrentMessage != "" &&
    analysisCtx.ExtractedConfidence > 0 &&
    len(analysisCtx.ExtractedEntities) > 0
```

**After**:
```go
// Check if there's actually a pending clarification question
pendingClarifications := db.GetPendingClarifications(userID, conversationID)
isAnsweringClarification := len(pendingClarifications) > 0 && 
    addressesClarificationQuestion(analysisCtx.CurrentMessage, pendingClarifications)
```

---

### Requirement #3: Distinguish Primary Goal vs Current Intent

**Where**: Layer 1 extraction and LayerContext

**Add to LayerContext**:
```go
type LayerContext struct {
    ...
    PrimaryGoal         string  // From Message 1, never overwritten
    CurrentMessageIntent string // Fresh each message
    GoalProgression     []Goal  // Track how goal evolves
}
```

**In extraction**: Mark goal from Message 1 as primary, subsequent messages as variant/evolved

---

### Requirement #4: Link Entities to Subjects

**Where**: `agents/context_extractor.go` (LLM prompt)

**Add to extraction prompt**:
```
When extracting characteristics, always tag:
- USER: for characteristics about the person writing the message
- CONTACT_[name]: for characteristics about someone they're describing
- CONTEXT: for general context facts

Example:
User writes: "I'm dominant. She wants a mentor."
Extract as:
- characteristic|dominant|USER|0.9
- preference|wants a mentor|CONTACT_Christine_sub|0.9
NOT:
- preference|wants a mentor|contact_name|0.9
```

---

## PART 9: ESTIMATED EFFORT

| Change | Effort | Complexity | Risk |
|--------|--------|-----------|------|
| Call SetAccumulatedContext | 2 hours | Low | Low (just wire existing code) |
| Fix clarification detection | 4 hours | Medium | Medium (logic change) |
| Load previous extraction | 2 hours | Low | Low (DB query) |
| Distinguish intent types | 3 hours | Medium | Medium (affects multiple layers) |
| Link entity subjects | 3 hours | Medium | Medium (changes extraction format) |
| **TOTAL** | **14 hours** | Medium | Medium |

---

## CONCLUSION

**The system is not missing capabilities - it's failing at execution.**

The architecture provides:
- ✅ State persistence fields (PreviousGoal, PreviousValues, etc.)
- ✅ Methods to load state (SetAccumulatedContext)
- ✅ Extraction of rich data (39+ entity types)
- ✅ Layers to process data (11 layers, each with defined purpose)

But these capabilities are **disconnected**:
- SetAccumulatedContext method exists but never called
- Previous goal never loaded because context never populated
- Fresh extraction overwrites previous because loop pattern skips Layer 1
- Clarification detection false-fires because logic is too broad

**Fix Strategy**: Wire existing capability together, don't add new components.

**Key Insight**: Information IS being extracted correctly. It's just not being accumulated, prioritized, or persisted across messages. The management layer is broken, not the extraction layer.
