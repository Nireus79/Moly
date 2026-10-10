# FIX #5 & #6 COMPLETION GUIDE
**Production Readiness Final Fixes**  
**Date:** October 8, 2026  
**Effort:** ~2 hours remaining

---

## ALREADY COMPLETED ✅

### chat_repository.go
- ✅ GetMessage(userID, messageID) - DONE
- ✅ DeleteMessage(userID, messageID) - DONE with error handling
- ✅ DeleteConversation(userID, conversationID) - DONE with error handling
- ✅ Error logging added
- ✅ RowsAffected verification

### clarification_repository.go  
- ✅ GetQuestion(userID, questionID) - DONE
- ✅ Error handling pattern established

---

## REMAINING: 5 REPOSITORIES (2 hours estimated)

### 1. pending_input_repository.go (30 min)

**File:** `database/pending_input_repository.go`

**Methods to update:**
```go
// Line 253: MarkApplied
// OLD:
func (r *PendingInputRepository) MarkApplied(id int64) error {
    query := `UPDATE pending_inputs SET status = 'applied', resolved_at = ? WHERE id = ?`
    _, err := r.db.Exec(query, time.Now().Unix(), id)
    return err
}

// NEW:
func (r *PendingInputRepository) MarkApplied(userID string, id int64) error {
    query := `UPDATE pending_inputs SET status = 'applied', resolved_at = ? WHERE id = ? AND user_id = ?`
    result, err := r.db.Exec(query, time.Now().Unix(), id, userID)
    if err != nil {
        log.Printf("[PendingInputRepository] Error marking applied: %v", err)
        return err
    }
    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("input not found or access denied")
    }
    return nil
}
```

**Methods:** MarkApplied, MarkProcessed, Delete (all 3 need userID parameter + WHERE clause)

---

### 2. clarification_history.go (30 min)

**File:** `database/clarification_history.go`

**Methods to update:**
- DeleteHistory(userID, questionID) - Add userID to WHERE
- GetQuestionHistory(userID, questionID) - Add userID to WHERE
- RecordStatusChange(userID, questionID, status) - Add userID to WHERE

**Pattern:** Every query has `WHERE` clause - add `AND user_id = ?` or `AND ch.user_id = ?`

---

### 3. context_attribute_repository.go (20 min)

**File:** `database/context_attribute_repository.go`

**Method:**
```go
// Line 279: DeleteAttribute
// OLD:
func (r *ContextAttributeRepository) DeleteAttribute(attributeID string) error {
    query := `DELETE FROM context_attributes WHERE id = ?`
    _, err := r.db.Exec(query, attributeID)
    return err
}

// NEW:
func (r *ContextAttributeRepository) DeleteAttribute(userID, attributeID string) error {
    query := `DELETE FROM context_attributes WHERE id = ? AND user_id = ?`
    result, err := r.db.Exec(query, attributeID, userID)
    if err != nil {
        log.Printf("[ContextAttributeRepository] Error deleting: %v", err)
        return err
    }
    rows, _ := result.RowsAffected()
    if rows == 0 {
        return fmt.Errorf("attribute not found or access denied")
    }
    return nil
}
```

---

### 4. sentence_analysis_repository.go (15 min)

**File:** `database/sentence_analysis_repository.go`

**Method:** UpdateResolution (line ~171)
- Add userID parameter
- Add `AND user_id = ?` to WHERE clause
- Add error handling

**Pattern:** Same as others

---

### 5. Other repositories (25 min)

**Quick scan for missed methods:**
```bash
grep -n "DELETE FROM\|UPDATE.*WHERE id = " database/*.go | grep -v "user_id"
```

**Files to check:**
- conversation_summary_repository.go (check UpdateSummary, ResetMessagesSinceUpdate)
- contact_repository.go (verify already has userID checks)
- clarification_repository.go responses (verify already has userID checks)

---

## EXECUTION CHECKLIST

### Per Method (Apply consistently):
- [ ] Add `userID string` parameter (first parameter after receiver)
- [ ] Add `AND user_id = ?` to WHERE clause
- [ ] Add `userID` to exec/query arguments
- [ ] Add error logging: `log.Printf("[Repository] Error: %v", err)`
- [ ] For DELETE: Verify `RowsAffected() > 0`, return error if 0
- [ ] For UPDATE: Log success with affected rows count
- [ ] Add import "log" if not already present

### After Each Fix:
```bash
cd moly-go && go build . 2>&1 | grep error
```
Should return nothing (or only pre-existing errors)

### Final:
```bash
cd moly-go && go test ./... 2>&1 | tail -5
```
Verify tests still pass

---

## COMMIT TEMPLATE

```
FIX #5 + #6: Complete data isolation and error handling

SECURITY FIX #6 - Data Isolation:
- pending_input_repository: Added userID to Mark/Delete methods
- clarification_history: Added userID to all query methods
- context_attribute_repository: Added userID to Delete
- sentence_analysis_repository: Added userID to Update
- ALL DELETE operations now verify RowsAffected > 0

ERROR HANDLING FIX #5:
- Added logging to all DELETE/UPDATE operations
- All operations now return errors instead of silently failing
- Prevented unauthorized data access across all repositories

SECURITY IMPACT:
- Users cannot delete/modify other users' data
- All error conditions properly logged and reported
- Data integrity verified with RowsAffected checks

BUILD: ✅ Clean

PRODUCTION READY: ✅ YES
```

---

## WHAT THIS COMPLETES

✅ **Production Readiness Achieved**
- All critical bugs fixed (FIX #1-11)
- All data isolation verified (FIX #6)
- All error handling complete (FIX #5)
- Schema consolidated (Migration 044)
- Validation tests pass
- Build clean

**Next phases (not blocking production):**
- Phase 3: Dead code cleanup (5.5 hours, with proper tools)
- Phase 4: Database consolidation (1.5 hours)
- Phase 5: Final cleanup (1.25 hours)

---

## QUICK REFERENCE: Where to Make Changes

| File | Lines | Change |
|------|-------|--------|
| pending_input_repository.go | 253, 262, 273 | Add userID param + AND user_id = ? |
| clarification_history.go | ~50, ~100, ~150 | Add userID param + AND user_id = ? |
| context_attribute_repository.go | 279 | Add userID param + AND user_id = ? |
| sentence_analysis_repository.go | 171 | Add userID param + AND user_id = ? |

---

## VERIFICATION AFTER COMPLETION

1. Build must pass: `go build .`
2. Tests must pass: `go test ./...`
3. No new compilation errors
4. No new test failures
5. All methods have error logging

---

**Once complete: PRODUCTION READY ✅**

System is then ready for deployment with all critical security and reliability fixes in place.
