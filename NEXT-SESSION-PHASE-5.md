# PHASE 5: Quick Start Checklist

**Status:** Ready to start  
**Estimated Duration:** 2-3 hours  
**Scope:** Integration testing + frontend wiring  

---

## Quick Checklist

### 🔴 MUST DO (Critical Path - 60 min)

- [ ] **Test 1: Single Conversation (M1→M2→M3)**
  - M1: Send "Help with relationship" → Check: phase=initial, maturity~0.2
  - M2: Send clarification → Check: phase=gathering, maturity~0.45, DB updated
  - M3: Send more context → Check: phase=analysis, maturity~0.65+, DB updated
  - Verify: Phase persists (doesn't reset to initial)

- [ ] **Test 2: Database Verification**
  - Query conversation_execution_state table
  - Verify: phase column updates for each message
  - Verify: maturity increases
  - Verify: No duplicate phases

- [ ] **Test 3: Response Metadata**
  - Send message via API
  - Check response.metadata.phase field
  - Verify: current, previous, maturity, transitioned, accomplishments
  - All fields present and correct

### 🟡 SHOULD DO (Frontend - 75 min)

- [ ] **Add Phase Display to UI**
  - Phase badge (top right)
  - Maturity bar (0-100%)
  - Phase timeline (initial → gathering → analysis)

- [ ] **Wire Response Metadata**
  - Parse response.metadata.phase
  - Update UI state on each message
  - Handle phase transitions

### 🟢 NICE TO HAVE (Optional - 60 min)

- [ ] **Debug Panel** (Ctrl+P)
  - Show phase data
  - Show accomplishments
  - Export for debugging

- [ ] **Extended Testing**
  - Multiple users
  - Edge cases (short/long messages)
  - Load testing

---

## Success Criteria

✅ Phase progression M1→M2→M3 works  
✅ Database persists phase correctly  
✅ API responses include metadata  
✅ Frontend displays phase badge + bar  
✅ No errors in console or database  

---

## If Things Go Wrong

| Problem | Likely Cause | Fix |
|---------|---|---|
| Phase resets to initial | currentPhase overwrite (line 1695) | Already fixed in Phase 4 |
| Metadata missing | Response generation (line 3223) | Check agentResp.Metadata |
| Frontend not updating | Response parsing | Check ConversationPanel.tsx |
| Maturity decreases | Accomplishment logic | Check MarkAccomplished() |

---

## Files to Reference

- `PHASE-5-INTEGRATION-GUIDE.md` - Detailed guide
- `moly-go/main.go` - Lines 1847-1876 (accomplishment tracking)
- `moly-go/main.go` - Lines 1950-1969 (phase persistence)
- `moly-go/main.go` - Lines 3223-3237 (response metadata)
- `moly-extension/src/components/ConversationPanel.tsx` - UI component

---

## Test Commands

```bash
# Build without starting server
cd /home/nireus79/vs_projects/Moly/Moly/moly-go
go build -o ../bin/moly .

# Start server (manually)
./bin/moly

# Test with curl (from another terminal)
curl -X POST http://localhost:8080/api/v2/message \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Help with relationship","conversationId":"test_conv_1"}'

# Check database
sqlite3 ~/.moly/moly.db
> SELECT phase, maturity FROM conversation_execution_state WHERE conversation_id='test_conv_1';
```

---

## Next: Begin Integration Test

Ready to start Test 1: Single Conversation (M1→M2→M3)

**See:** PHASE-5-INTEGRATION-GUIDE.md for detailed test plan
