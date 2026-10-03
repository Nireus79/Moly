# PHASE 5: Test Execution Plan

**Date:** October 3, 2026  
**Session:** 24 (Starting)  
**Status:** Ready for execution

---

## Test Environment Setup

### Prerequisites
- ✅ Phase 4 built and tested (21MB binary ready)
- ✅ All 364 unit tests passing
- ✅ Database schema includes conversation_execution_state
- ✅ LLM (Ollama) available at http://127.0.0.1:11434

### Database Check
```bash
sqlite3 ~/.moly/moly.db ".schema conversation_execution_state"
```

Expected columns:
- user_id (TEXT)
- conversation_id (TEXT)
- phase (TEXT) - initial/gathering/analysis/help
- maturity (REAL) - 0.0-1.0
- updated_at (INTEGER)

---

## TEST 1: Single Conversation Phase Progression

**Objective:** Verify phase progression works end-to-end (M1→M2→M3)

### Setup Phase
1. Start fresh database: `rm ~/.moly/moly.db` (or use new test file)
2. Create test user account
3. Record conversation_id for verification

### Execution

#### Step 1.1: First Message (M1 - Initial)

**Input:**
```json
{
  "message": "I want to improve my relationship with John. He's someone I work with closely.",
  "conversationId": "test_phase5_001",
  "aboutMe": {
    "communicationStyle": "",
    "coreValues": [],
    "tonePreference": ""
  }
}
```

**Expected Response:**
```json
{
  "metadata": {
    "phase": {
      "current": "initial",
      "previous": "initial",
      "maturity": 0.2,
      "transitioned": false
    }
  }
}
```

**Database Verification (M1):**
```sql
SELECT phase, maturity, updated_at FROM conversation_execution_state 
WHERE conversation_id='test_phase5_001' LIMIT 1;
-- Expected: initial | 0.2-0.3 | [current timestamp]
```

✅ **Assertion 1.1:** Phase is "initial", maturity ~0.2-0.3

---

#### Step 1.2: Second Message (M2 - Clarification/Gathering)

**Input:**
```json
{
  "message": "We often disagree on how to approach technical decisions. I try to be collaborative, but he seems dismissive of my ideas.",
  "conversationId": "test_phase5_001",
  "aboutMe": {
    "communicationStyle": "collaborative",
    "coreValues": ["respect", "clear_communication"],
    "tonePreference": "constructive"
  }
}
```

**Expected Response:**
```json
{
  "metadata": {
    "phase": {
      "current": "gathering",
      "previous": "initial",
      "maturity": 0.45,
      "transitioned": true
    }
  }
}
```

**Database Verification (M2):**
```sql
SELECT phase, maturity, updated_at FROM conversation_execution_state 
WHERE conversation_id='test_phase5_001' LIMIT 1;
-- Expected: gathering | 0.4-0.5 | [new timestamp]
-- CRITICAL: Must NOT be "initial" (that would indicate reset bug!)
```

✅ **Assertion 1.2:** Phase is "gathering", maturity ~0.4-0.5, transitioned=true

---

#### Step 1.3: Third Message (M3 - Analysis)

**Input:**
```json
{
  "message": "I've tried being direct about my concerns, but it doesn't seem to work. Should I escalate to management or try a different approach?",
  "conversationId": "test_phase5_001",
  "aboutMe": {
    "communicationStyle": "collaborative",
    "coreValues": ["respect", "clear_communication", "problem_solving"],
    "tonePreference": "constructive"
  }
}
```

**Expected Response:**
```json
{
  "metadata": {
    "phase": {
      "current": "analysis",
      "previous": "gathering",
      "maturity": 0.65,
      "transitioned": true
    }
  }
}
```

**Database Verification (M3):**
```sql
SELECT phase, maturity, updated_at FROM conversation_execution_state 
WHERE conversation_id='test_phase5_001' LIMIT 1;
-- Expected: analysis | 0.6-0.7 | [new timestamp]
-- CRITICAL: Must NOT be "initial" or "gathering" (reset bug)
```

✅ **Assertion 1.3:** Phase is "analysis", maturity ~0.6-0.7, transitioned=true

---

### Test 1 Success Criteria

✅ M1: Phase loaded as "initial", maturity ~0.2  
✅ M2: Phase progressed to "gathering", maturity ~0.45, DB updated  
✅ M3: Phase progressed to "analysis", maturity ~0.65, DB updated  
✅ Phase never reset (critical bug check)  
✅ Maturity always increased (never decreased)  
✅ Timestamps updated after each message  

---

## TEST 2: Database Consistency

**Objective:** Verify database state is consistent

### Queries to Run

```sql
-- 1. Check phase progression
SELECT conversation_id, phase, maturity, updated_at 
FROM conversation_execution_state 
ORDER BY updated_at DESC 
LIMIT 10;
-- Verify: phases are in order, maturity increases

-- 2. Check for phase resets
SELECT conversation_id, phase, COUNT(*) as phase_count
FROM conversation_execution_state
GROUP BY conversation_id, phase
HAVING COUNT(*) > 1;
-- Verify: No duplicates (phase shouldn't appear twice for same conv)

-- 3. Check maturity progression
SELECT conversation_id, 
       GROUP_CONCAT(maturity, ' → ') as progression
FROM conversation_execution_state
GROUP BY conversation_id;
-- Verify: Maturity only increases (0.2 → 0.45 → 0.65)

-- 4. Check accomplishments
SELECT conversation_id, phases 
FROM conversation_maturity
ORDER BY updated_at DESC
LIMIT 5;
-- Verify: Phases populated with accomplishments
```

✅ **Assertion 2.1:** Phase progression matches expected (initial→gathering→analysis)  
✅ **Assertion 2.2:** No duplicate phase entries  
✅ **Assertion 2.3:** Maturity always increases  
✅ **Assertion 2.4:** Accomplishments recorded in phases  

---

## TEST 3: Response Metadata Format

**Objective:** Verify API response includes complete phase metadata

### Check Each Response

For each message (M1, M2, M3), verify response contains:

```json
{
  "response": "...",
  "metadata": {
    "phase": {
      "current": "...",        // ✅ Present
      "previous": "...",       // ✅ Present
      "maturity": 0.0,         // ✅ Present, numeric
      "transitioned": false,   // ✅ Present, boolean
      "accomplishments": {     // ✅ Present
        "completed": 0,        // ✅ Integer
        "total": 0,            // ✅ Integer
        "maturity": 0.0        // ✅ Numeric
      }
    }
  }
}
```

### Validation Checks

- [ ] All phase fields present (current, previous, maturity, transitioned)
- [ ] accomplishments present and valid
- [ ] Values are correct type (string for phase, number for maturity)
- [ ] No null or undefined values
- [ ] No error messages in response

✅ **Assertion 3.1:** All phase fields present  
✅ **Assertion 3.2:** All field types correct  
✅ **Assertion 3.3:** Values match database state  

---

## TEST 4: Accomplishment Tracking

**Objective:** Verify accomplishments are tracked correctly

### Expected Accomplishments by Phase

**M1 (initial phase):**
- entities_extracted: YES (contact: John, goals: improve relationship)
- clarifications_answered: NO
- gaps_identified: NO
- conflicts_handled: NO

**M2 (gathering phase):**
- entities_extracted: YES (conflict detected)
- clarifications_answered: YES (answered about relationship)
- gaps_identified: YES (about Me fields filled)
- conflicts_handled: NO

**M3 (analysis phase):**
- entities_extracted: YES
- clarifications_answered: YES (answered about approach)
- gaps_identified: YES
- conflicts_handled: NO

### Verification

```sql
-- Check accomplished tasks
SELECT phase, accomplishments 
FROM conversation_maturity 
WHERE conversation_id='test_phase5_001';
```

✅ **Assertion 4.1:** Accomplishments recorded for each phase  
✅ **Assertion 4.2:** Accomplishments match message content  

---

## TEST 5: Performance Check

**Objective:** Verify no performance regression

### Measurements

- [ ] M1 response time: <2 seconds (first message, includes setup)
- [ ] M2 response time: <1.5 seconds (phase progression)
- [ ] M3 response time: <1.5 seconds
- [ ] Database query time: <100ms
- [ ] Metadata generation: <50ms

### Tools

```bash
# Using curl with time measurement
time curl -X POST http://localhost:8080/api/v2/message \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"...","conversationId":"..."}'
```

✅ **Assertion 5.1:** All response times within budget  
✅ **Assertion 5.2:** No timeouts or slow responses  

---

## TEST FAILURE TROUBLESHOOTING

### Issue: Phase Resets (current = previous, not transitioned)

**Symptom:** M2 shows current="initial" instead of "gathering"

**Likely Cause:** Phase overwrite bug (lines 1695-1701 in main.go)

**Debug:**
```bash
# Check if bug fix was applied
grep -n "currentPhase already set from maturityCalc" moly-go/main.go
# Should show lines with comment, not assignment
```

**Fix:** Phase overwrite bug was already fixed in Phase 4. If this appears, rerun fix.

---

### Issue: Metadata Missing

**Symptom:** Response has no metadata.phase field

**Likely Cause:** Response generation skipped or metadata not built

**Debug:**
```go
// Check main.go lines 3223-3237
// Verify: agentResp.Metadata["phase"] is being set
```

**Fix:** Ensure maturityCalc is not nil and phase persistence succeeded.

---

### Issue: Maturity Doesn't Increase

**Symptom:** M2 maturity = M1 maturity (no improvement)

**Likely Cause:** Accomplishments not being tracked

**Debug:**
```bash
# Check logs for "PHASE 4: Recorded" messages
# Should show accomplishments recorded for each message
```

**Fix:** Verify MarkAccomplished() is being called for each message.

---

## Test Execution Order

1. ✅ **Test 1: Phase Progression** - Start here (most critical)
2. ✅ **Test 2: Database Consistency** - Verify DB state
3. ✅ **Test 3: Response Metadata** - Check API output
4. ✅ **Test 4: Accomplishment Tracking** - Verify tracking
5. ✅ **Test 5: Performance** - Measure speed

---

## Expected Results Summary

| Test | Status | Expected | Actual | Pass |
|------|--------|----------|--------|------|
| Phase M1→M2→M3 | Ready | initial→gathering→analysis | ??? | ??? |
| DB Persistence | Ready | Phase updates, no resets | ??? | ??? |
| Metadata Format | Ready | All fields present | ??? | ??? |
| Accomplishments | Ready | Tracked for each phase | ??? | ??? |
| Performance | Ready | <2s per message | ??? | ??? |

---

## Next Steps

1. Start Test 1 (Phase Progression)
2. Run 3-message conversation (M1→M2→M3)
3. Verify each assertion passes
4. Document any issues found
5. Proceed to Test 2 (Database) if Test 1 passes

---

**Created:** October 3, 2026  
**Status:** Ready to execute  
**Start:** Test 1 - Single Conversation Phase Progression
