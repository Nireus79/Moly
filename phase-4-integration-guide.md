# PHASE 4: Main.go Integration - Detailed Guide

**Date:** October 3, 2026  
**Session:** 23 (Phase 4 Implementation)  
**Status:** Ready to implement  

---

## Overview

Phase 4 integrates the maturity redesign (Phases 0-3) into the main message processing pipeline. Key work:

1. **✅ ALREADY DONE:** Maturity loading and calculation (lines 667, 1876)
2. **⏳ IN PROGRESS:** Phase persistence (lines 1947-1954 - incomplete)
3. **❌ TO DO:** Accomplishment tracking (orchestrator → DB)
4. **❌ TO DO:** Response metadata with phase info

---

## Integration Points

### 1. Maturity Load (Line ~667) ✅ WORKING

```go
maturityCalc, matErr := srv.maturityService.LoadOrCreateMaturityContext(userID, req.ConversationID)
if matErr != nil {
    log.Printf("[MessageProcessor] Warning: Failed to load maturity context: %v", matErr)
    initialContextMaturity = 0.0
} else if maturityCalc != nil {
    initialContextMaturity = maturityCalc.CalculateOverallMaturity()
    log.Printf("[MessageProcessor] ✓ Loaded maturity context: initial=%.2f", initialContextMaturity)
}
```

✅ Already working - loads maturity from database

---

### 2. Accomplishment Tracking (NEW - After Orchestrator)

**Location:** Between line 1840 (orchestrator complete) and line 1842 (safety check)

**What to add:**

```go
// PHASE 4: Track accomplishments from orchestrator run
// These are achievements that improve maturity
var accomplishments *models.PhaseAccomplishments

if layerCtx != nil {
    accomplishments = &models.PhaseAccomplishments{
        MessageID: userMessageID,
        
        // Layer 1: Extracted entities count
        EntitiesExtracted: len(extractedEntities),
        ExtractionConfidence: extractedConfidence,
        
        // Layer 3: Clarifications answered
        ClarificationsAnswered: 0, // Count from processedClarificationAnswer
        
        // Layer 4: Gaps detected and resolved
        GapsDetected: layerCtx.Layer4.GapCount if layerCtx.Layer4 != nil,
        GapsResolved: 0, // Count from this message
        
        // Layer 5: Conflicts handled
        ConflictsDetected: layerCtx.Layer5.ConflictCount if layerCtx.Layer5 != nil,
        ConflictsResolved: 0, // Count from this message
        
        // Layer 6-7: Ambiguities and violations clarified
        AmbiguitiesCleared: 0, // True if was ambiguous, now clear
        ViolationsClarified: 0, // Count of clarifications asked
        
        // Recording metadata
        Timestamp: time.Now().Unix(),
    }
    
    // Track if answering clarification (improvement to maturity)
    if processedClarificationAnswer {
        accomplishments.ClarificationsAnswered = 1
    }
    
    log.Printf("[MessageProcessor] ✓ Recorded accomplishments: entities=%d, clarifications=%d, conflicts=%d",
        accomplishments.EntitiesExtracted,
        accomplishments.ClarificationsAnswered,
        accomplishments.ConflictsDetected)
}
```

**Why:** Tracks what was achieved in this message to guide maturity calculation. Essential for phase progression.

---

### 3. Phase Update and Persistence (Line ~1937-1954) ⏳ ENHANCE

**Current code (partially working):**

```go
// Fix S: Determine new phase based on recalculated maturity
newPhase := currentPhase
if newMaturity < 0.3 {
    newPhase = "discovery"
} else if newMaturity < 0.6 {
    newPhase = "gathering"
} else {
    newPhase = "analysis"
}

if newPhase != currentPhase {
    log.Printf("[MessageProcessor] ✓ Phase advancement: %s to %s (maturity: %.2f)", currentPhase, newPhase, newMaturity)
    // Update conversation phase in execution state for agent
    if execState != nil {
        execState.Phase = agents.ExecutionPhase(newPhase)
        log.Printf("[MessageProcessor] ✓ Updated ConversationPhase to %s", newPhase)
    }
}
```

**What to add after this:**

```go
// PHASE 4: PERSIST phase progression to database
// This ensures next message loads the NEW phase (not old one)
if newPhase != currentPhase && execState != nil {
    // Save phase progression to conversation record
    conn := srv.database.GetConnection()
    now := time.Now().Unix()
    
    // Update execution state in database
    updateErr := conn.QueryRow(`
        UPDATE conversation_execution_state 
        SET phase = ?, updated_at = ?
        WHERE user_id = ? AND conversation_id = ?
        RETURNING id
    `, newPhase, now, userID, conversationID).Scan()
    
    if updateErr != nil {
        log.Printf("[MessageProcessor] Warning: Failed to persist phase to database: %v", updateErr)
    } else {
        log.Printf("[MessageProcessor] ✓ Persisted phase progression: %s → %s (maturity: %.2f)", 
            currentPhase, newPhase, newMaturity)
    }
    
    // Also update maturity progression record for audit trail
    if maturityCalc != nil {
        phaseProgErr := maturityCalc.RecordPhaseProgression(newPhase, newMaturity)
        if phaseProgErr != nil {
            log.Printf("[MessageProcessor] Warning: Failed to record phase progression: %v", phaseProgErr)
        }
    }
}
```

**Why:** Ensures phase persists across messages. Without this, next message loads old phase.

---

### 4. Response Metadata (Line ~2250+) ❌ TO DO

**Location:** When building final response to send to frontend (around response generation)

**What to add:**

```go
// PHASE 4: Include phase information in response metadata
if agentResp != nil && agentResp.Metadata == nil {
    agentResp.Metadata = make(map[string]interface{})
}

if agentResp != nil && maturityCalc != nil {
    // Add phase progression tracking
    agentResp.Metadata["phase"] = map[string]interface{}{
        "current": newPhase,
        "previous": currentPhase,
        "maturity": finalContextMaturity,
        "transitioned": (newPhase != currentPhase),
        "accomplishments": map[string]interface{}{
            "entitiesExtracted": accomplishments.EntitiesExtracted,
            "clarificationsAnswered": accomplishments.ClarificationsAnswered,
            "conflictsResolved": accomplishments.ConflictsResolved,
        },
    }
    
    log.Printf("[MessageProcessor] ✓ Added phase metadata to response: %s", newPhase)
}
```

**Why:** Informs frontend of current phase so UI can adapt (e.g., show "Gathering info" vs "Analyzing options")

---

## Implementation Checklist

### Step 1: Add Accomplishment Tracking
- [ ] Insert accomplishment tracking code after line 1840
- [ ] Link `processedClarificationAnswer` flag (already exists at line 941)
- [ ] Ensure `extractedEntities`, `extractedConfidence` are available (they are - line ~860)
- [ ] Log accomplishments for debugging

### Step 2: Enhance Phase Persistence
- [ ] Add database UPDATE after phase detection (line ~1954)
- [ ] Ensure `conversation_execution_state` table has `phase` column
- [ ] Add phase progression recording to maturity service

### Step 3: Add Response Metadata
- [ ] Insert phase metadata into `agentResp.Metadata` 
- [ ] Include phase transition info
- [ ] Include accomplishments summary

### Step 4: Testing
- [ ] Single message: Verify maturity improves (M1 to M2)
- [ ] Two messages: Verify phase loads from DB (not reset)
- [ ] Conversation flow: initial → gathering → analysis progression
- [ ] Response: Contains phase field in metadata

### Step 5: Verification
- [ ] Build: `go build ./...` - no errors
- [ ] Tests: `go test ./... -race` - no race conditions
- [ ] End-to-end: Send message, check response has phase field

---

## Key Files to Update

| File | Lines | Change |
|------|-------|--------|
| `main.go` | ~1840 | Add accomplishment tracking |
| `main.go` | ~1954 | Add phase persistence |
| `main.go` | ~2250 | Add response metadata |
| `models/phase_definitions.go` | Already exists | Use `PhaseAccomplishments` struct |
| `storage/maturity_service.go` | Already exists | Methods are ready |

---

## Success Criteria

1. ✅ Build passes: `go build ./... && go test -race ./...`
2. ✅ Single message: Maturity improves from 0.0 → 0.3+
3. ✅ Phase detected: Message 1 shows "discovery", message 2+ shows correct phase
4. ✅ Response metadata: Phase field present in all responses
5. ✅ Persistence: Closing app and reopening shows saved phase
6. ✅ No race conditions: `go test -race ./...` passes

---

## Integration Flow Summary

```
Message Arrives
    ↓
Load Phase from DB (Line 667) ← DONE
    ↓
Orchestrator Runs (Line 1764)
    ↓
Track Accomplishments (NEW: After 1840) ← TO DO
    ↓
Recalculate Maturity (Line 1876) ← DONE
    ↓
Detect Phase Progression (Line 1937) ← DONE (partially)
    ↓
Persist Phase to DB (NEW: After 1954) ← TO DO
    ↓
Generate Response (Line 2182)
    ↓
Add Phase to Metadata (NEW: Line 2250+) ← TO DO
    ↓
Send Response with Phase
```

---

## Common Issues & Fixes

### Issue: Phase not persisting (always shows "discovery")
**Fix:** Check `conversation_execution_state` table has `phase` column. Run:
```sql
ALTER TABLE conversation_execution_state ADD COLUMN phase TEXT DEFAULT 'initial';
```

### Issue: Maturity not improving
**Fix:** Verify accomplishments tracking is recording achievements. Check logs for:
```
[MessageProcessor] ✓ Recorded accomplishments: entities=...
```

### Issue: Response doesn't have phase field
**Fix:** Verify metadata update at response generation. Check:
1. `agentResp.Metadata` is not nil
2. Phase field is being added before response sent
3. JSON encoding includes metadata

### Issue: Race condition on phase update
**Fix:** Use transaction when updating both execution state and maturity:
```go
tx, err := db.BeginTx(context.Background(), nil)
// Update both tables in transaction
tx.Commit()
```

---

## Testing Commands

### Test 1: Build Clean
```bash
cd moly-go && go build ./...
```

### Test 2: Run All Tests
```bash
cd moly-go && go test ./... -race -timeout 30s
```

### Test 3: Manual Test (Single Message)
```bash
curl -X POST http://localhost:8080/api/v2/message \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "I want to improve my relationship with John",
    "conversationId": "conv_test_123"
  }'
```

Check response for:
```json
{
  "metadata": {
    "phase": {
      "current": "gathering",
      "maturity": 0.45,
      "transitioned": true
    }
  }
}
```

---

## Database Schema Check

Verify these columns exist:

```sql
-- conversation_execution_state table
SELECT phase FROM conversation_execution_state LIMIT 1;

-- OR create column if missing
ALTER TABLE conversation_execution_state ADD COLUMN phase TEXT DEFAULT 'initial';
ALTER TABLE conversation_execution_state ADD COLUMN maturity REAL DEFAULT 0.0;
```

---

## References

- `NEXT-SESSION-PHASE-4.md` - Quick start guide
- `storage/maturity_service.go` - Maturity service implementation (line ~1-250)
- `models/phase_definitions.go` - Phase structs and constants
- `ARCHITECTURE.md` - System data flow
- Session 22 memory: `[[session-22-maturity-redesign]]` for design rationale

---

**Last Updated:** October 3, 2026  
**Session:** 23 (Phase 4 - Main.go Integration)  
**Status:** Ready for implementation
