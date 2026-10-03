# PHASE 5: Integration Testing & Frontend Wiring

**Date:** October 3, 2026  
**Session:** 24 (Phase 5 - Starting)  
**Scope:** Verify Phase 4 works end-to-end, wire frontend display  
**Status:** Planning

---

## Overview

Phase 4 implementation is complete and tested in isolation. Phase 5 verifies it works across real message flows and integrates with the frontend UI.

### What We're Testing
- Phase persistence across multiple messages (M1→M2→M3)
- Accomplishment tracking during real orchestrator runs
- Database updates happening correctly
- Response metadata being generated properly
- Frontend receiving and displaying phase info

---

## Phase 5 Checklist

### 🔴 CRITICAL PATH (Must Do)

#### 1. Integration Test: Single Conversation (M1→M2→M3)
**Goal:** Verify phase progression works end-to-end  
**Estimated Time:** 30 minutes

- [ ] **M1: Initial Message**
  - [ ] Send: "I want help with my relationship with John"
  - [ ] Expected: phase=initial, maturity~0.2
  - [ ] Verify: DB saved phase=initial
  - [ ] Check: accomplishments recorded (entities_extracted)

- [ ] **M2: Clarification Response**
  - [ ] Send: "More details... John is my colleague"
  - [ ] Expected: phase=gathering, maturity~0.45
  - [ ] Verify: DB updated phase=gathering (not reset!)
  - [ ] Check: accomplishments recorded (clarifications_answered)

- [ ] **M3: Deep Context**
  - [ ] Send: "We often disagree on project approach"
  - [ ] Expected: phase=analysis, maturity~0.65+
  - [ ] Verify: DB updated phase=analysis
  - [ ] Check: accomplishments recorded

#### 2. Database Verification
**Goal:** Verify phase persistence in database  
**Estimated Time:** 15 minutes

```sql
-- Check conversation_execution_state table
SELECT user_id, conversation_id, phase, maturity, updated_at 
FROM conversation_execution_state 
ORDER BY updated_at DESC 
LIMIT 5;

-- Check maturity history
SELECT user_id, conversation_id, overall_score, current_phase 
FROM conversation_maturity 
ORDER BY updated_at DESC 
LIMIT 5;
```

- [ ] Phase updates for each message
- [ ] Maturity increases (not decreases)
- [ ] Timestamps are correct
- [ ] No duplicate phases

#### 3. Response Metadata Verification
**Goal:** Verify API response includes phase info  
**Estimated Time:** 15 minutes

Check each response contains:
```json
{
  "metadata": {
    "phase": {
      "current": "gathering",
      "previous": "initial",
      "maturity": 0.45,
      "transitioned": true,
      "accomplishments": {
        "completed": 2,
        "total": 4,
        "maturity": 0.5
      }
    }
  }
}
```

- [ ] All phase fields present
- [ ] Values are correct
- [ ] Transitions tracked
- [ ] Accomplishments summary accurate

---

### 🟡 FRONTEND WIRING (Should Do)

#### 4. Frontend Phase Display
**Goal:** Show phase in conversation UI  
**Estimated Time:** 45 minutes

**Location:** `moly-extension/src/components/ConversationPanel.tsx` (or similar)

- [ ] **Display Phase Badge**
  - [ ] Show current phase: "initial", "gathering", "analysis", "help"
  - [ ] Use color: initial=gray, gathering=blue, analysis=green, help=gold
  - [ ] Position: Top right of conversation header
  - [ ] Example: `[Gathering 45%]`

- [ ] **Display Maturity Bar**
  - [ ] Horizontal progress bar: 0-100%
  - [ ] Color gradient: red(0%) → yellow(50%) → green(100%)
  - [ ] Show percentage next to bar
  - [ ] Update after each message

- [ ] **Display Phase Progression**
  - [ ] Timeline: `initial → gathering → analysis`
  - [ ] Highlight current phase
  - [ ] Show completed phases in lighter color
  - [ ] Example: `✓ initial  ▶ gathering  ○ analysis`

#### 5. Frontend Integration Points
**Goal:** Wire phase data through frontend  
**Estimated Time:** 30 minutes

- [ ] **Parse Response Metadata**
  ```typescript
  const phase = response.metadata?.phase;
  if (phase) {
    setCurrentPhase(phase.current);
    setMaturity(phase.maturity);
    setTransitioned(phase.transitioned);
  }
  ```

- [ ] **Update UI State**
  - [ ] Store phase in conversation state
  - [ ] Update on each message
  - [ ] Persist to localStorage (optional)

- [ ] **Handle Phase Transitions**
  - [ ] Highlight when phase changes
  - [ ] Show toast notification (optional)
  - [ ] Log for debugging

---

### 🟢 OPTIONAL ENHANCEMENTS (Nice to Have)

#### 6. Analytics & Debugging (Optional)
**Estimated Time:** 30 minutes (optional)

- [ ] **Add Phase Debug Panel**
  - [ ] Show raw phase data
  - [ ] Show accomplishments breakdown
  - [ ] Show maturity calculation details
  - [ ] Toggle with Ctrl+P (or Dev Tools)

- [ ] **Add Phase History Log**
  - [ ] Store phase changes in browser
  - [ ] Display in debug panel
  - [ ] Export for debugging

#### 7. Extended Testing (Optional)
**Estimated Time:** 30 minutes (optional)

- [ ] **Test with Different Users**
  - [ ] Verify isolation (users don't see each other's phases)
  - [ ] Test with 2+ concurrent conversations

- [ ] **Test Edge Cases**
  - [ ] Very short message (greeting only)
  - [ ] Very long message (>5000 chars)
  - [ ] Rapid-fire messages
  - [ ] Time gaps between messages

- [ ] **Load Testing (Optional)**
  - [ ] 10 concurrent conversations
  - [ ] Measure phase update latency
  - [ ] Verify no memory leaks

---

## Test Plan Detail

### Test 1: Single Conversation Phase Progression

**Setup:**
```
User: new account
Message 1: "Help with relationship"
Message 2: "More about the person"
Message 3: "How should I approach"
```

**Expected Outcomes:**

| Message | Expected Phase | Maturity | Accomplishments |
|---------|---|---|---|
| M1 | initial | 0.2-0.3 | entities_extracted |
| M2 | gathering | 0.4-0.5 | clarifications_answered |
| M3 | analysis | 0.6-0.7 | gaps_identified |

**Verification Script:**
```bash
# 1. Check DB after M1
sqlite3 ~/.moly/moly.db "SELECT phase, maturity FROM conversation_execution_state LIMIT 1;"
# Expected: initial | 0.2-0.3

# 2. Check DB after M2  
sqlite3 ~/.moly/moly.db "SELECT phase, maturity FROM conversation_execution_state LIMIT 1;"
# Expected: gathering | 0.4-0.5 (NOT reset to initial!)

# 3. Check DB after M3
sqlite3 ~/.moly/moly.db "SELECT phase, maturity FROM conversation_execution_state LIMIT 1;"
# Expected: analysis | 0.6-0.7 (NOT reset!)
```

---

## Frontend Implementation Steps

### Step 1: Add Phase State to Conversation Component
```typescript
const [phaseInfo, setPhaseInfo] = useState({
  current: 'initial',
  previous: 'initial', 
  maturity: 0,
  transitioned: false,
  accomplishments: { completed: 0, total: 0 }
});
```

### Step 2: Update on Message Response
```typescript
const handleMessageResponse = (response) => {
  if (response.metadata?.phase) {
    setPhaseInfo(response.metadata.phase);
  }
};
```

### Step 3: Render Phase UI
```typescript
<div className="phase-display">
  <PhaseBadge phase={phaseInfo.current} maturity={phaseInfo.maturity} />
  <MaturityBar value={phaseInfo.maturity * 100} />
  <PhaseTimeline current={phaseInfo.current} transitioned={phaseInfo.transitioned} />
</div>
```

### Step 4: Style Phase Components
- Phase badge: 30px height, rounded corners, phase-specific color
- Maturity bar: Full width, smooth gradient, shows percentage
- Timeline: Horizontal, 3 phases, click-to-see-details (optional)

---

## Success Criteria

✅ **Integration Test Passes**
- Phase progression works M1→M2→M3
- Phases persist in database (don't reset)
- Maturity increases with each message

✅ **Frontend Displays Phase**
- Phase badge shows current phase
- Maturity bar displays percentage
- Updates correctly after each message

✅ **End-to-End Verification**
- Database updates → Response metadata → Frontend display (full chain)
- No errors or console warnings
- Performance acceptable (<100ms round-trip)

✅ **Documentation Complete**
- Phase 5 results documented
- Integration test results saved
- Frontend wiring documented

---

## Rollback Plan

If Phase 5 finds issues:

1. **Phase not persisting:** Check lines 1950-1969 in main.go
2. **Metadata missing:** Check lines 3223-3237 in main.go
3. **Frontend not updating:** Check response parsing in ConversationPanel.tsx
4. **Maturity wrong:** Check CalculateOverallMaturity() in models/phase_accomplishment.go

**Rollback to:** Last known good commit (bb77062 - Phase 4 complete)

---

## Timeline Estimate

- **Critical Path (1-3):** ~60 minutes
- **Frontend Wiring (4-5):** ~75 minutes
- **Optional Enhancements (6-7):** ~60 minutes

**Total for Phase 5:** 2-3 hours

---

## Notes

### Known Limitations
- Phase history not tracked (Phase 6 feature)
- No UI for resetting phase (Phase 6 feature)
- No ML-based phase prediction (Phase 6 feature)

### Dependencies
- Phase 4 complete (✅ Done - commit bb77062)
- Tests passing (✅ 364/364 pass)
- Database schema ready (✅ conversation_execution_state table exists)

### Risks
- **Risk 1:** Phase resets instead of persisting (MITIGATED: Bug fixed in Phase 4)
- **Risk 2:** Maturity doesn't improve (MITIGATED: Accomplishment tracking working)
- **Risk 3:** Frontend doesn't receive metadata (MITIGATED: Response format tested)

---

## Next Steps After Phase 5

**Phase 6: Verification & Optimization**
- Extended testing (multiple users, edge cases)
- Performance optimization
- Analytics dashboard
- Safety verification

**Phase 7: Production Deployment**
- Final QA
- Documentation review
- Canary deployment
- Full rollout

---

**Created:** October 3, 2026  
**Status:** Ready to start  
**Next Action:** Begin Test 1 - Single Conversation Phase Progression
