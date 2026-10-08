# Investigation: Why Message 2 Shows Goal Shift (Root Cause Analysis)

**Date:** October 8, 2026  
**Issue:** Message 2 is incorrectly analyzed as a goal shift when it's actually context provision for the same goal

---

## The Evidence

### Message 1
```
"Help me craft a smart, playful opening message to Christine_sub"
Goal extracted: "craft_message"
```

### Message 2
```
"Here are some insights from my profile. [his kinks]
Here are some from her's: [her kinks]
[his communication strategy]
So, how do I start it?"
```

**Expected:** Goal = "craft_message" (SAME - answering clarification questions)  
**Actual:** Goal = "initiate BDSM relationship" (WRONG - false shift)

---

## ROOT CAUSE: Extraction Has No Previous Context

### The Problem Flow

**Layer 1 (Extraction Phase)** - extraction_phase.go line 92:
```go
extractedCtx, extractErr = ep.contextExtractor.Extract(ctx, input.Message)
```

This calls the extraction with ONLY the current message.

**ContextExtractor.Extract** - context_extractor.go line 32:
```go
func (ce *ContextExtractor) Extract(ctx context.Context, userMessage string) (*models.ExtractedContext, error)
```

Signature takes ONLY:
- `ctx context.Context`
- `userMessage string`

**No parameters for:**
- Previous goal
- Recent messages
- Conversation history
- Message count

**When building the LLM prompt** - context_extractor.go line 48:
```go
prompt := ce.buildExtractionPrompt(userMessage)
```

The prompt is built with ONLY the current message!

**Result:** The LLM sees Message 2 with BDSM information and infers goal = "initiate BDSM relationship" because it has NO CONTEXT that:
1. Previous message goal was "craft message"
2. This is a clarification response, not a new request
3. BDSM info is CONTEXT about people, not the user's new goal

### Timeline in Code

1. **extraction_phase.go line 92:** Extract(userMessage) called WITHOUT previous goal
2. **context_extractor.go line 48:** LLM prompt built WITHOUT previous goal
3. **goal_coherence_handler.go line 25:** Compares `lc.UserGoal` (wrongly extracted as "initiate BDSM") vs `lc.PrimaryGoal` ("craft message")
4. **Result:** Marks as "different" goal

---

## The Missing Information

The ExtractionPhaseInput (line 16-26) HAS this information:

```go
type ExtractionPhaseInput struct {
    UserID         string                   // ✓ have it
    Message        string                   // ✓ have it
    MessageCount   int                      // ✓ have it (could detect Message 2)
    RecentMessages []models.Message         // ✓ have it (previous messages!)
    PreviousExtraction interface{}          // ✓ have it (contains Message 1 goal!)
    // ... other fields
}
```

But this information is **NOT PASSED** to the extraction LLM!

---

## Why This Happens

### Current Flow
```
main.go line 998: Build ExtractionPhaseInput (includes PreviousExtraction)
  ↓
extraction_phase.go line 92: Call Extract(message) ← ONLY passes current message!
  ↓
context_extractor.go line 48: Build prompt from message ONLY ← No previous context!
  ↓
LLM extracts without knowing previous goal ← Makes wrong inference
```

### What Should Happen
```
extraction_phase.go line 92: Call Extract(message, previousGoal, recentMessages)
  ↓
context_extractor.go line 48: Build prompt INCLUDING previous goal
  ↓
LLM prompt includes: "Previous goal was 'craft message'. Is the user changing goals or providing context?"
  ↓
LLM correctly identifies: "Still crafting message, now providing context about both people"
```

---

## The Solution (Not Hardcoding)

**NOT:** Add rule "if message says 'Here's my profile' then assume same goal" (hardcoding)

**YES:** Pass previous context to LLM so it can make informed decision

**Change Extract function signature:**
```go
// Before
func (ce *ContextExtractor) Extract(ctx context.Context, userMessage string) (*models.ExtractedContext, error)

// After
func (ce *ContextExtractor) Extract(
    ctx context.Context, 
    userMessage string,
    previousGoal string,           // ← NEW: from Message 1
    messageCount int,              // ← NEW: detect if this is Message 2+
) (*models.ExtractedContext, error)
```

**Update extraction prompt** to include:
- "Previous message's goal was: X"
- "Is this message changing the goal, or providing context for the same goal?"
- "If uncertain, ask for clarification"

This lets the LLM understand Message 2 correctly WITHOUT hardcoded rules.

---

## Architecture Insight

This reveals a deeper pattern:

- **Layer 1 (Extraction)** should have multi-message context
- **Not** just the current message in isolation
- **Goal determination** needs to consider conversation history
- **Loop pattern** can help (skip Layer 1 if detecting clarification)
- **But extraction itself** should be smarter

---

## Why FIX #72 Can't Solve This Alone

FIX #72 (Gap Tagging & Filtering) assumes goals are correctly identified.  
But if Layer 1 extracts the wrong goal, FIX #72 just works with bad data.

The order matters:
1. **FIX Layer 1 Extraction** ← Identify correct goal (CRITICAL)
2. **FIX #72 Phase 1A** ← Calculate goal coherence correctly
3. **FIX #72 Phase 2** ← Filter gaps based on correct goal

Without fixing Layer 1, goal coherence and gap filtering are working with false information.

---

## Current Status

**What's Broken:**
- ❌ Message 2 extraction doesn't know previous goal
- ❌ LLM infers new goal from BDSM context alone
- ❌ Goal coherence compares wrong values
- ❌ Gap filtering can't help (bad goal already set)

**What Would Fix It:**
- ✅ Pass previous goal to extraction
- ✅ Let LLM decide goal continuity (not hardcoding)
- ✅ Then goal coherence works correctly
- ✅ Then FIX #72 works correctly

---

## Recommendation

Before implementing any gap filtering or goal coherence changes:

**FIX #72 Phase 0: Goal Extraction (FIRST)**
1. Update Extract() to receive previousGoal and messageCount
2. Update buildExtractionPrompt() to include: "Previous goal was X. Is the user changing goals?"
3. Let LLM determine goal continuity from conversation context
4. THEN goal coherence can correctly compare goals
5. THEN gap filtering works on correct data

This is NOT hardcoding - this is giving the LLM the context it needs to make the right decision.

---

**Next Step:** Test what extraction actually returns for Message 2 if it had previous context about Message 1's goal.
