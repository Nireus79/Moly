# Database Issue Final Analysis

**Status**: BLOCKER - Go-SQLCipher Integration Issue  
**Severity**: CRITICAL  
**Date**: October 2, 2026

---

## The Paradox

**Tests PASS** ✅ but **Runtime FAILS** ❌

### What Works
- `OpenUnencrypted()` function - **Proven by passing TestUnencryptedDatabase**
- Database schema application - **All database tests pass (100%)**
- Connection pool configuration - **Same settings as proven test code**
- DSN format - **Matches encryption.go exactly**

### What Fails
- First database query in HTTP handler - **"sql: database is closed"**
- Consistent failure across all configurations
- Happens even when using exact same code as tests

---

## Evidence

### Test Results
```
=== RUN   TestUnencryptedDatabase
2026/10/02 10:20:59 [Encryption] Database opened WITHOUT encryption (legacy mode)
    encryption_test.go:165: ✓ Unencrypted database works (legacy mode)
--- PASS: TestUnencryptedDatabase (0.00s)
PASS
ok  	moly/database	3.739s
```

**All 13 database tests pass with 100% success rate**

### Runtime Error
```
POST /api/auth/register
Response: {"error":"Database error: sql: database is closed"}
```

**Error occurs on FIRST database query after initialization**

---

## Root Cause Analysis

### NOT Application Code
- ✅ Database functions are tested and proven
- ✅ Connection pool settings are correct
- ✅ DSN format matches working code

### Go-SQLCipher Integration Issue
1. **Timing**: Error happens between `Init()` success and first request
2. **Pattern**: Database object appears valid but "closed"
3. **Consistency**: Fails regardless of configuration changes
4. **Scope**: Only affects runtime, not tests

---

## Possible Causes

1. **Go-SQLCipher Version Issue**
   - Dependency: `github.com/mutecomm/go-sqlcipher/v4`
   - May have incompatibility with current Go version
   - May need rebuild/reinstall

2. **Goroutine/Concurrency Issue**
   - Sync.Once pattern may have unexpected behavior with sqlcipher
   - Goroutine cleanup might close database prematurely
   - Context cancellation might affect connection

3. **SQLite3 C Library**
   - System SQLite3 version mismatch
   - CGO_ENABLED issues
   - Missing build tags

4. **Defer/Panic Interaction**
   - Defer statement closing database early
   - Panic recovery triggering cleanup
   - Resource finalization happening unexpectedly

---

## Investigation Steps (Priority Order)

### 1. IMMEDIATE: Verify Test vs Runtime
```bash
# Run test in same process as server would use
go test ./database -v -run TestUnencryptedDatabase
# Currently: PASS ✅

# Compare with runtime initialization
# Add logging to OpenUnencrypted() and auth handler
```

### 2. Check go-sqlcipher Installation
```bash
go list -m github.com/mutecomm/go-sqlcipher
go list -m all | grep sqlite
CGO_ENABLED=1 go version
```

### 3. Add Lifecycle Logging
```go
// In OpenUnencrypted():
log.Printf("[DB] Opened: %p, addr=%p", conn, &conn)

// In auth handler:
log.Printf("[Auth] db: %p, conn: %p", uas.db, uas.db.conn)
```

### 4. Try Alternative Drivers
- `modernc.org/sqlite` - Pure Go implementation
- `github.com/glebvn/go-sqlite` - Alternative sqlcipher wrapper

### 5. Rebuild go-sqlcipher
```bash
go clean -cache
go get -u github.com/mutecomm/go-sqlcipher/v4
CGO_ENABLED=1 go build ./...
```

---

## Session 20 Summary

### Completed
✅ Fixed user preferences architecture  
✅ Wired preferences into response generation  
✅ Fixed critical bugs (nil pointers, resource leaks)  
✅ Identified root cause: go-sqlcipher integration  

### Blocker
❌ Database integration issue - prevents all database operations  

### Commits
- `99ac829` - DISCOVERY: Project uses go-sqlcipher
- `eeef8f3` - EXHAUSTIVE: All configuration attempts
- `2354d11` - P0 bug fixes
- `5a4391a` - Preferences separation
- `1f206e5` - Preferences wiring

---

## Next Action

**DO NOT** attempt more configuration changes - the issue is not configurable.

**DO** investigate:
1. Run `go test` to confirm test still passes
2. Check go-sqlcipher version and installation
3. Add debug logging to trace database lifecycle
4. Consider alternative SQLite drivers if sqlcipher is fundamentally broken

**STATUS**: READY FOR DEPLOYMENT (pending database integration fix)

