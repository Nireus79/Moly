# PHASE 6: Extended Testing & Verification

**Date:** October 3, 2026  
**Session:** 25 (Starting)  
**Status:** Phase 5 fix complete, Phase 6 ready to start

---

## 🎯 What Happened in Phase 5

### Root Cause Identified & Fixed ✅
**Issue:** Response metadata was missing `phase` field
**Root Cause:** `maturityCalc` was only initialized when `req.Message != ""`
**Fix:** Moved maturity loading outside message condition (commit b310f78)

**Result:** 
- Phase metadata now always present in responses
- `metadata.phase` structure guaranteed in all responses
- Phase progression tracking ready for testing

---

## 🚀 Phase 6 Objectives

Phase 6 is comprehensive end-to-end verification and optimization:

### 1. Complete Integration Testing (60 min)
- [x] Fix Phase 5 blocking issue
- [ ] Run full M1→M2→M3 sequence test
- [ ] Verify phase doesn't reset between messages
- [ ] Check maturity increases monotonically
- [ ] Confirm database persistence

### 2. Multi-Message Testing (45 min)
- [ ] M4→M5 extended conversation test
- [ ] Verify phase transition rules working
- [ ] Test edge cases (short/long messages)
- [ ] Concurrent message handling

### 3. Database Verification (30 min)
- [ ] Check `conversation_execution_state` table
- [ ] Verify phase persistence across server restart
- [ ] Validate maturity calculations in DB
- [ ] Ensure no duplicate/conflicting records

### 4. Safety & Compliance Testing (45 min)
- [ ] Constitutional principle evaluation
- [ ] Crisis detection accuracy
- [ ] Gate enforcement (all 11 layers)
- [ ] False positive rate verification

### 5. Performance Benchmarking (30 min)
- [ ] Response time per message (<2s target)
- [ ] Maturity calculation overhead
- [ ] Database query performance
- [ ] Memory usage profiles

---

## ✅ Phase 6 Success Criteria

### Core Requirements
- [ ] M1→M2→M3→M4 sequence completes without phase resets
- [ ] Maturity: ~0.2 → ~0.45 → ~0.65 → ~0.80 (monotonic increase)
- [ ] Database shows correct phase progression
- [ ] Response metadata includes all required fields
- [ ] No errors in server logs

### Additional Checks
- [ ] Phase persists after server restart
- [ ] Multiple concurrent conversations work
- [ ] Conversation history intact
- [ ] Contact relationships maintained
- [ ] Accomplishments tracked correctly

---

## 📋 Phase 6 Test Plan

### Test 6.1: Extended Message Sequence (90 min)
```
M1: "I want to improve my relationship with John. He's someone I work with closely."
    Expected: phase=initial, maturity~0.2

M2: "We often disagree on how to approach technical decisions. I try to be collaborative, but he seems dismissive."
    Expected: phase=gathering, maturity~0.45, transitioned=true

M3: "I've tried being direct about my concerns, but it doesn't seem to work. Should I escalate to management or try a different approach?"
    Expected: phase=analysis, maturity~0.65, transitioned=true

M4: "I think the core issue is that we have different communication styles. He's more assertive, while I prefer a collaborative approach."
    Expected: phase=help, maturity~0.80, transitioned=true
```

### Test 6.2: Database Persistence (30 min)
```sql
-- After M1-M4 complete, verify:
SELECT phase, maturity, updated_at FROM conversation_execution_state 
WHERE conversation_id='test_conv_123' 
ORDER BY updated_at;

-- Expected: 4 rows with phases [initial, gathering, analysis, help]
-- Maturity should be: [0.2-0.3, 0.4-0.5, 0.6-0.7, 0.75-0.85]
```

### Test 6.3: Server Restart Persistence (15 min)
1. Run M1→M2, record conversation_id
2. Kill server
3. Restart server  
4. Check if phase/maturity loaded correctly
5. Send M3, verify phase transitions properly

---

## 🔧 Phase 6 Execution Steps

### Step 1: Prepare Test Environment
```bash
# Kill old server
pkill -f "./bin/moly"

# Clean database
rm -f /tmp/moly-v2.db

# Start fresh server
cd ~/vs_projects/Moly/Moly/moly-go
nohup ./bin/moly > /tmp/moly.log 2>&1 &
sleep 3

# Verify server running
curl http://localhost:8080/api/auth/register \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","email":"test@test.com","password":"testpass123"}' | jq '.token'
```

### Step 2: Run Extended Message Sequence Test
Create `phase6_test.sh` with full M1→M2→M3→M4 sequence

### Step 3: Verify Database State
- Check `conversation_execution_state` table
- Verify phase column progression
- Confirm maturity increases

### Step 4: Test Server Persistence
- Document conversation_id from test
- Kill/restart server
- Verify phase/maturity reloads correctly

### Step 5: Safety Gate Testing
- Verify all 11 layers evaluating
- Check crisis detection accuracy
- Confirm denial gates working

---

## 📊 Expected Results

### Phase Progression
```
M1 (initial):   phase=initial,  maturity=0.20, transitioned=false
M2 (gathering): phase=gathering, maturity=0.45, transitioned=true
M3 (analysis):  phase=analysis,  maturity=0.65, transitioned=true
M4 (help):      phase=help,      maturity=0.80, transitioned=true
```

### Maturity Calculation
```
Each phase contributes to maturity:
- Entities extracted: +0.1 per entity (capped at 0.3/phase)
- Gaps clarified: +0.15 per gap (capped at 0.3/phase)
- Intention established: +0.2
- Profile complete: +0.15

Total across phases: initial(0.2) + gathering(0.25) + analysis(0.2) + help(0.15) = 0.80
```

### Response Metadata Structure
```json
{
  "success": true,
  "response": "...",
  "metadata": {
    "phase": {
      "current": "gathering",
      "previous": "initial",
      "maturity": 0.45,
      "transitioned": true,
      "accomplishments": {
        "completed": 2,
        "total": 4,
        "maturity": 0.50
      }
    }
  }
}
```

---

## 🛠️ Tools & Resources

### Testing Scripts Location
- `phase6_test.sh` - Extended message sequence test (to be created)
- `phase6_db_verify.sh` - Database verification script (to be created)
- `phase6_stress_test.sh` - Multi-user concurrent test (optional)

### Key Files
- `moly-go/main.go` - Lines 678-705 (maturity loading)
- `moly-go/main.go` - Lines 1900-1970 (phase persistence)
- `moly-go/main.go` - Lines 3240-3280 (response metadata)
- `moly-go/storage/maturity_service.go` - Maturity logic

### Monitoring
- Server logs: `/tmp/moly.log`
- Database: `/tmp/moly-v2.db`
- Debug logs: `grep "DEBUG\|PHASE" /tmp/moly.log`

---

## 📌 Known Issues & Workarounds

### Issue: Tests Timeout
**Cause:** LLM processing (Ollama) is slow
**Workaround:** Use `timeout 300` (5 min) for each test, monitor /tmp/moly.log

### Issue: Database Not Updating
**Cause:** Phase persistence code not reached
**Status:** Fixed in Phase 5 (commit b310f78)

### Issue: Phase Resets
**Cause:** CurrentPhase overwrite bug
**Status:** Fixed in Phase 4

---

## 🎓 Phase 6 Learning Goals

1. **Verify end-to-end system works** - All layers properly wired
2. **Understand phase progression** - How accomplishments trigger transitions
3. **Database correctness** - Phase/maturity persisted accurately
4. **Performance characteristics** - Where bottlenecks are
5. **Safety gates** - Crisis detection and denial working

---

## ⏱️ Phase 6 Timeline

- **Setup & Test 6.1:** 30 min
- **Database Verification:** 15 min
- **Persistence Testing:** 15 min
- **Safety Gate Testing:** 20 min
- **Documentation:** 15 min
- **Buffer & Issues:** 15 min

**Total Estimated:** 110 minutes (1.8 hours)

---

## ✍️ Session Notes

**Phase 5 Completion Summary:**
- Root cause: maturityCalc nil when req.Message empty
- Fix: Unconditional maturity loading
- Commit: b310f78
- Status: Ready for Phase 6 verification

**What Phase 6 Will Confirm:**
1. Phase metadata consistent across all responses
2. Phase progression follows expected pattern (initial→gathering→analysis→help)
3. Maturity increases monotonically
4. Database persistence working
5. Multi-message conversations stable

---

**Ready to Start Phase 6**

Begin with Step 1: Prepare test environment and run Test 6.1

See `phase6_test.sh` (to be created) for full test execution
