# Bug Tracker - Active Issues

**Updated**: September 30, 2026  
**Status**: 9 open, 2 in progress, 0 critical

---

## Open Issues

### 1. Missing rows.Err() Checks (9 remaining)

**Severity**: MEDIUM  
**Status**: IN PROGRESS (2/11 fixed)  
**Effort**: 1-2 hours  

**Issue Description**:
After SQL row iteration loops (`for rows.Next()`), errors from iteration are not checked. If iteration fails (e.g., network timeout, connection drop), the error is silently lost.

**Pattern**:
```go
defer rows.Close()
for rows.Next() {
    // process rows
}
// BUG: No rows.Err() check!
return data, nil
```

**Correct Pattern**:
```go
defer rows.Close()
for rows.Next() {
    // process rows
}
// FIX: Check for iteration errors
if err := rows.Err(); err != nil {
    return nil, fmt.Errorf("scan failed: %w", err)
}
return data, nil
```

**Fixed**:
- ✅ database/analysis_context_builder.go:160 (commit dd96376)
- ✅ storage/maturity_service.go:305 (commit dd96376)

**Remaining**:
- ⏳ main.go:1142
- ⏳ main.go:2217
- ⏳ main.go:3074
- ⏳ main.go:3478
- ⏳ main.go:3691
- ⏳ main.go:3923
- ⏳ main.go:4473
- ⏳ agents/learning_agent.go (1+ locations)
- ⏳ agents/temporary_fact_store.go (1+ locations)

**Impact**: Iteration errors silently masked, could indicate data corruption or network issues

**Fix Script**:
```bash
# Find all locations:
grep -n "defer rows.Close()" moly-go/*.go moly-go/**/*.go

# For each location, find where it returns after the for loop
# Add: if err := rows.Err(); err != nil { return ..., err }
```

---

## Completed Fixes

### ✅ Factory Pattern for ConversationAgent (Sept 30)
- Enforced initialization order via `NewFullyInitializedConversationAgent()`
- Added `IsReady()` validation
- Commit: a067ef3

### ✅ FK Constraint Timing Bugs (Sept 30)
- Conversation creation moved before message_processing_state
- Conversation creation moved before meta-instruction path
- Commits: 733c603, 3fdef48

### ✅ Nil Error Handling (Sept 30)
- Fixed spurious "Fatal error: <nil>" messages
- Commit: 3b918ea

### ✅ Dead Code Removal (Sept 30)
- Removed orphaned SuggestionGenerator, QuestionGenerator, ClarificationAsker
- Commit: d5bb738

---

## Resolution Priority

### P1 (This Week)
- [ ] Fix remaining 9 rows.Err() checks

### P2 (Next Week)
- [ ] Add comprehensive SQL error logging
- [ ] Audit other SQL patterns for similar issues

### P3 (Later)
- [ ] Consider database query wrapper for automatic error checking

---

## How to Contribute

1. Pick a location from "Remaining" list
2. Read the file and understand the context
3. Add rows.Err() check after the for loop
4. Test: `go test ./...`
5. Commit with pattern: `🐛 FIX: Add rows.Err() check in <file>:<line>`

**Example Commit Message**:
```
🐛 FIX: Add rows.Err() check in main.go:1142

Missing error check after for rows.Next() loop was silently
losing iteration errors. Added proper rows.Err() validation.

Impact: Iteration errors no longer masked.
Build: ✅ Clean
Tests: ✅ All passing
```

---

## Statistics

| Metric | Value |
|--------|-------|
| Total Issues | 10 |
| Critical | 0 |
| High | 0 |
| Medium | 9 |
| Low | 1 |
| Fixed This Session | 2 |
| Remaining | 9 |
| Avg Fix Time | 5-10 min each |

---

## Related Documentation

- [DOCUMENTATION_INDEX.md](../DOCUMENTATION_INDEX.md) - Main index
- [CURRENT_STATUS.md](CURRENT_STATUS.md) - Session status
- [BUG_TRACKER.md](BUG_TRACKER.md) - This file
