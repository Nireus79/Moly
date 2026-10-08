# Moly: Unwired Components Investigation

**Date:** October 8, 2026  
**Status:** Comprehensive audit of implemented-but-unwired code

---

## Executive Summary

| Component | Status | Priority | Impact |
|-----------|--------|----------|--------|
| Layer 10 Persistence | ❌ STUB | 🔴 CRITICAL | Multi-turn questioning doesn't track state |
| ResponseContradictionHandler | ❌ UNUSED | 🟡 MEDIUM | Contradiction detection never runs |
| AnswerProcessor | ✅ WIRED | ✅ Working | Clarification response processing |
| IncomingMessageAnalyzer | ✅ WIRED | ✅ Working | Message analysis & suggestions |
| ConversationAnalyzer | ✅ WIRED | ✅ Working | Multi-message conversation analysis |
| ContextManager | ✅ WIRED | ✅ Working | User context management |
| GoalCoherence | ✅ WIRED (FIX #72) | ✅ Working | Goal-aligned gap detection |

---

## CRITICAL FINDINGS

### 1. Layer 10 Persistence: STUB IMPLEMENTATION

**File:** `agents/layer10_persistent_questioning.go:77-99`

**Current State:**
```go
// LoadOrCreateSession - STUB
// Returns new session every time, never loads from DB
session := l10.LoadOrCreateSession(lc.UserID, lc.ConversationID)  // Line 143
session.QuestionCount++  // Incremented but never persists
l10.SaveSession(...)  // Logs but doesn't persist (Line 189)
```

**Why It's a Problem:**

The system is designed to ask clarification questions up to 4 times ("Max turns"):
```
Message 1: "Why do you want to do this?" (QuestionCount=0→1, save fails)
Message 2: Loads new session (QuestionCount=0), asks same question again! ❌
Message 3: Loads new session again, asks same question AGAIN! ❌
Message 4: Loads new session again...
```

Result: User sees the same persistent question repeatedly instead of different probing questions.

**Why It's Not Wired:**

Database table never created (line 78 stub):
```go
func (l10 *Layer10PersistentQuestioning) createTableIfNotExists() {
    log.Printf("[Layer10] Database persistence stub - state will not persist across messages")
}
```

No schema defined, no save logic, no load logic.

**What Needs to Happen:**

1. Create `persistence_sessions` table in database
2. Implement `LoadOrCreateSession()` to actually query database
3. Implement `SaveSession()` to actually persist QuestionCount and PreviousAnswers
4. Test: Verify question count increments across messages

**Impact if Fixed:**
- Layer 10 can properly track "user asked 3x about this harm"
- Progressive questioning strategy becomes viable
- Users can't evade ethical questioning by just re-asking

---

### 2. ResponseContradictionHandler: COMPLETE BUT UNUSED

**File:** `agents/response_contradiction_handler.go`

**Current State:**

✅ Fully implemented with:
- `GenerateContradictionQuestion()` - Creates clarification questions
- `ExplainContradiction()` - Explains the contradiction
- `SaveContradictionQuestion()` - Saves to DB
- `GetSummary()` - Formats summary for user
- 6 comprehensive unit tests

❌ But: **Zero usage in main codebase**

Only appears in:
- Unit tests (response_contradiction_handler_test.go)
- Integration tests (phase3_integration_test.go)
- Class definition

**Why It's Not Wired:**

No call to this handler exists in:
- `main.go` - Never instantiated
- `unified_orchestrator.go` - Never called
- `conversation_agent.go` - Never referenced
- Any response generation pipeline

**What It Was Meant To Do:**

When user's response contradicts their known characteristics:
```
User says: "I value directness"
User does: Sends manipulative opening message
Handler should: Ask "I notice you value directness, but this 
               message uses indirect tactics. Can you explain?"
```

**Why It's Disconnected:**

The contradiction detection logic was moved to other layers:
- Layer 5: Conflict detection (different focus - contradictory instructions)
- Layer 2: Principle evaluation (catches some contradictions)
- Layer 7: Violation clarification (specific to principles)

This handler became redundant/specialized, but was never removed or reintegrated.

**Cost of Not Fixing:**

Users can contradict themselves without being asked about it:
- Say they value honesty, then ask how to manipulate
- Say they want consent-focused communication, then request non-consensual tactics
- Say they're cautious, then ask for reckless plans

**What Needs to Happen:**

Option A (Recommended): Integrate into conflict detection pipeline
- Add to Layer 5 or Layer 7 as specialized contradiction handler
- Use existing tests to verify integration
- Remove if truly redundant

Option B: Remove if truly redundant
- Delete handler and tests
- Document why contradiction handling is in Layer 5/7 instead

---

## WORKING COMPONENTS (Confirmed Wired)

### ✅ AnswerProcessor
- **File:** `agents/answer_processor.go`
- **Usage:** Line 3887 in `main.go` - processes clarification responses
- **Status:** FULLY OPERATIONAL
- **What it does:** Extracts context from user's answers to clarification questions

### ✅ IncomingMessageAnalyzer
- **File:** `agents/incoming_message_analyzer.go`
- **Usage:** Lines 4035, 4113, 4121 in `main.go`
- **Status:** FULLY OPERATIONAL
- **What it does:** Analyzes incoming messages, detects sender, generates suggestions

### ✅ ConversationAnalyzer
- **File:** `agents/conversation_analyzer.go`
- **Usage:** Line 5549 in `main.go` - analyzes full conversation
- **Status:** FULLY OPERATIONAL
- **What it does:** Analyzes multi-message conversations for patterns

### ✅ ContextManager
- **File:** `agents/context_manager.go`
- **Usage:** Integrated in `agents/v2_agents.go` (AgentSystem)
- **Status:** FULLY OPERATIONAL
- **What it does:** Manages user About Me profile and context

### ✅ GoalCoherence (FIXED in FIX #72)
- **File:** `agents/goal_coherence_handler.go`
- **Usage:** Now wired in `unified_orchestrator.go` line ~415
- **Status:** FULLY OPERATIONAL (as of Session 35)
- **What it does:** Analyzes how current goal relates to primary goal

---

## ROOT CAUSE ANALYSIS

### Why Components Get Unwired

**Pattern 1: Architectural Drift**
- Component designed for one approach
- System evolved in different direction
- Component never removed (legacy code accumulation)
- Example: ResponseContradictionHandler (designed for response validation, moved to extraction phase)

**Pattern 2: Incomplete Implementation**
- Component skeleton created with tests
- Core persistence/database logic stubbed out
- Never finished before moving to other work
- Example: Layer 10 persistence (schema designed, implementation stubbed)

**Pattern 3: Knowledge Loss**
- Developer who designed component left or moved on
- No documentation of integration plan
- Future developers don't know where to wire it
- Example: ContextManager had unclear DB integration

**Pattern 4: Architectural Confusion**
- Multiple components with overlapping purpose
- Unclear which should handle what
- All get implemented "just in case"
- Example: Multiple conflict detectors (Layer 5, ResponseContradictionHandler, ConflictDetector)

---

## Recommendations

### PRIORITY 1 (Fix Now)
**Layer 10 Persistence - Database Schema**
- Time: ~4 hours
- Impact: Multi-turn questioning becomes functional
- Files to change: 
  - `agents/layer10_persistent_questioning.go` - Implement LoadOrCreateSession/SaveSession
  - Database migrations - Create persistence_sessions table
  - main.go - Ensure Layer 10 has database access

### PRIORITY 2 (Clarify)
**ResponseContradictionHandler - Audit Decision**
- Time: ~1 hour
- Decision: Keep or delete?
  - If duplicate: Delete handler + tests, document why Layer 5/7 handle it instead
  - If specialized: Integrate into Layer 7 for response-specific contradictions
  - Do not leave in limbo

### PRIORITY 3 (Monitor)
**Unused Handlers Pattern**
- Add to code review checklist: "Any handler/analyzer created but not instantiated in main.go?"
- Annual audit: Remove dead code or document why it exists
- Architecture decision: When to leave experimental code vs when to prune

---

## Code Quality Impact

**Technical Debt:**
- ~150 lines of unused/untested-in-production code (ResponseContradictionHandler)
- ~50 lines of stub code (Layer 10 persistence)
- Total: ~200 lines of incomplete implementation

**Maintainability:**
- ⚠️ Developers may spend time investigating unused handlers
- ⚠️ Tests pass but feature doesn't work (Layer 10)
- ⚠️ Uncertainty about architectural intent (which layer handles contradictions?)

**Opportunity Cost:**
- Developer time spent implementing ResponseContradictionHandler → could have fixed Layer 10
- No clear integration points for future components
- Makes codebase harder to understand ("why does this exist?")

---

## Session 35 Context

**Discovered:** ResponseContradictionHandler completely unused  
**Wired:** GoalCoherence → Gap Detection (FIX #72)  
**Fixed:** Entity saturation preventing maturity improvement (FIX #71)

This investigation revealed that pattern of partial implementations and architectural drift, which explains why:
- GoalCoherence was analyzed but not used
- Layer 10 can't track persistent questioning state
- ResponseContradictionHandler has full tests but zero production usage

---

## Next Steps

1. **Immediate:** Wire Layer 10 persistence (makes persistent questioning actually work)
2. **Soon:** Audit ResponseContradictionHandler (remove or integrate)
3. **Ongoing:** Add pre-commit checks for "handler created but not wired" pattern
4. **Documentation:** Update CLAUDE.md with architectural layers and their responsibilities

