# P0 FIXES - CONTINUATION GUIDE

**Status**: Started (3/26 connection fixes complete)  
**Last Commit**: `3a21fcd`  
**Next Task**: Continue database connection leak fixes  
**Token Limit**: Reached previous session

---

## PROGRESS

✅ **Completed** (3 fixes):
- Line 164: Initial context setup
- Line 310: Token verification
- Line 563: Message processing main

❌ **Remaining** (23 fixes):
- Lines: 937, 1013, 1555, 1988, 2463, 2569, 2648, 3355, 3411, 3461, 3561, 3742, 3850, 3935, 4017, 4236, 4334, 4445, 5031, 5153, 5391, 5465, 5536

---

## HOW TO CONTINUE

### Option 1: Manual Fixes (Most reliable)
For each line number, add `defer conn.Close()` after `GetConnection()`:

```go
// Pattern to apply
conn := srv.database.GetConnection()
defer conn.Close()  // ← ADD THIS LINE
```

### Option 2: Script Fix (If quick fix works)
```bash
cd /home/nireus79/vs_projects/Moly/Moly/moly-go

# Run script to fix all remaining
./apply_connection_fixes.sh
```

### Option 3: Batch Find-Replace
Use exact line numbers from list above to fix each location

---

## VERIFICATION STEPS

1. **Build**: `go build ./...` (must pass)
2. **Test**: `go test ./...` (must pass)
3. **Commit**: Descriptive commit message
4. **Push**: `git push origin master`

---

## NEXT AFTER CONNECTIONS

Once all 26 connection fixes done:

### P0 Task 2: Unchecked Database Errors (38 queries)
Locations to check:
- Line 1165: AboutMe query
- Line 1341: Intention query
- Line 1429: Contact query
- All QueryRow() in main.go

Pattern: Add error check after every `.Scan()`
```go
err := conn.QueryRow(...).Scan(&value)
if err != nil {
    log.Printf("Query failed: %v", err)
    value = ""  // Set safe default
}
```

### P0 Task 3: Goroutine Coordination (4 locations)
Verify at lines: 2153, 2172, 2376, 4794
- Ensure wg.Add() matches goroutine count
- Verify defer wg.Done() on each goroutine
- Confirm wg.Wait() called before proceeding

---

## FILES MODIFIED

- `main.go` - Database connection leak fixes

---

## ESTIMATED REMAINING EFFORT

- Connection fixes: ~30 minutes
- Error check fixes: ~1 hour  
- Goroutine verify: ~30 minutes
- Testing: ~30 minutes
- **Total P0**: ~2.5 hours

---

**Next Session**: Start with continuing connection fixes at line 937

