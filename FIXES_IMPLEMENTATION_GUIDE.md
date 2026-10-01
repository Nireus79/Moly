# CRITICAL BUGS - IMPLEMENTATION GUIDE

**Date**: October 1, 2026  
**Status**: Ready for implementation in next sessions  
**Total Issues**: 4 CRITICAL + 3 HIGH priority

---

## ISSUE #1: DATABASE CONNECTION LEAK FIX

**Priority**: CRITICAL  
**Locations**: 62 GetConnection() calls throughout codebase  
**Effort**: LOW (repetitive pattern)  
**Impact**: HIGH (prevents pool exhaustion)

### Strategy: Systematic Pattern Replacement

```go
// BEFORE (No cleanup)
conn := srv.database.GetConnection()
err := conn.QueryRow("SELECT ...").Scan(&result)

// AFTER (With cleanup)
conn := srv.database.GetConnection()
defer conn.Close()
err := conn.QueryRow("SELECT ...").Scan(&result)
if err != nil {
    log.Printf("Error: %v", err)
}
```

### Locations to Fix (Main.go)
- Line 164: `conn := db.GetConnection()`
- Line 182: `intentDetector.SetDatabase(db.GetConnection())`
- Line 309: `conn := db.GetConnection()`
- Line 563: `conn := srv.database.GetConnection()`
- Line 934: `conn := srv.database.GetConnection()` (inside clarification)
- Lines 1159-1163: AboutMe query without defer
- Lines 1337-1340: Intention query without defer
- Lines 1425-1428: Contact query without defer
- ... and 54 more locations

### Search Strategy
```bash
cd /home/nireus79/vs_projects/Moly/Moly/moly-go
grep -n "GetConnection()" main.go | grep -v "defer" | grep -v "Close"
```

### Fix Template
For each location:
1. Find the GetConnection() call
2. Add `defer conn.Close()` on next line
3. Verify error checking follows
4. Build and test

---

## ISSUE #2: UNCHECKED DATABASE ERRORS FIX

**Priority**: CRITICAL  
**Locations**: 38 database query assignments  
**Effort**: MEDIUM (needs context analysis)  
**Impact**: HIGH (prevents data corruption)

### Strategy: Add Error Checks After Every Query

```go
// BEFORE (Error checked with else-if)
err := conn.QueryRow(...).Scan(&value)
if err == nil && value != "" {
    // Use value
}
// Value might be nil if error occurred!

// AFTER (Error checked first)
err := conn.QueryRow(...).Scan(&value)
if err != nil {
    log.Printf("Query failed: %v", err)
    // Set default value
    value = ""
} else if value != "" {
    // Use value
}
```

### Locations to Check
- Line 1165: AboutMe query
- Line 1341: Intention query  
- Line 1429: Contact query
- All QueryRow() calls in main.go (grep for them)

### Pattern to Search
```bash
grep -n "QueryRow.*Scan" main.go | grep -v "if err"
```

### Fix for Each Location
1. Find QueryRow().Scan() call
2. Add error check immediately after
3. Set safe default value if error
4. Add log.Printf for debugging

---

## ISSUE #3: GOROUTINE COORDINATION FIX

**Priority**: CRITICAL  
**Locations**: Lines 2153, 2172, 2376, 4794  
**Effort**: MEDIUM (needs verification)  
**Impact**: HIGH (prevents goroutine leak)

### Strategy: Verify WaitGroup Usage

**Location 1 (Lines 2153-2180)**: Response generation goroutines
```go
// Current: Two goroutines with WaitGroup
wg.Add(2)
go func() {
    defer wg.Done()
    // Goroutine 1
}()
go func() {
    defer wg.Done()
    // Goroutine 2
}()
wg.Wait()  // ← MUST be present
```

**Verify**: 
- [ ] wg.Add(N) matches number of goroutines
- [ ] Each goroutine calls defer wg.Done()
- [ ] wg.Wait() called before proceeding

**Location 2 (Line 2376)**: Summary update goroutine
```go
// Current: Non-blocking summary update
go func(fullHistory []models.Message) {
    // No WaitGroup here!
}(history)
```

**Fix Options**:
- Option A: Add WaitGroup coordination
- Option B: Add channel for completion notification
- Option C: Remove goroutine (run synchronously)

### Action Items
1. Search for all `go func` in main.go
2. Verify each is either:
   - Under WaitGroup with defer wg.Done()
   - Has completion channel
   - Has cleanup logic
3. Add wg.Wait() call before proceeding past goroutine block

---

## ISSUE #4: RACE CONDITION FIX

**Priority**: CRITICAL  
**Locations**: 209 concurrent maps  
**Effort**: HIGH (needs refactoring)  
**Impact**: HIGH (prevents data corruption)

### Strategy: Add Synchronization

```go
// BEFORE (Concurrent access without lock)
var contextMap = make(map[string]interface{})

// Goroutine 1
contextMap["key"] = value

// Goroutine 2  
val := contextMap["key"]  // RACE CONDITION!

// AFTER (With RWMutex)
var contextMapMutex sync.RWMutex
var contextMap = make(map[string]interface{})

// Goroutine 1 (write)
contextMapMutex.Lock()
contextMap["key"] = value
contextMapMutex.Unlock()

// Goroutine 2 (read)
contextMapMutex.RLock()
val := contextMap["key"]
contextMapMutex.RUnlock()
```

### Maps to Protect (Identify with `go test -race`)
1. Run race detector first to identify which maps have issues
2. Add sync.RWMutex for each concurrent map
3. Wrap all accesses with Lock/Unlock or RLock/RUnlock

### Execution Steps
1. Run: `go test -race ./...`
2. Identify which maps trigger race condition warnings
3. Add corresponding mutexes
4. Wrap all accesses with proper locking
5. Run race detector again to verify

---

## IMPLEMENTATION CHECKLIST

### Session 1 (Immediate - P0)
- [ ] Add defer Close() to all 62 GetConnection() calls
  - Estimated effort: 1-2 hours
  - Search locations first, then batch apply fixes
  
- [ ] Add error checks to 38 database queries
  - Estimated effort: 2-3 hours  
  - Highest impact area for data corruption

- [ ] Verify goroutine coordination at 4 locations
  - Estimated effort: 1 hour
  - Just verify existing patterns are correct

### Session 2 (P1)
- [ ] Run `go test -race ./...` to identify race conditions
- [ ] Add sync.RWMutex to concurrent maps
- [ ] Replace error suppressions with proper handling
- [ ] Add nil guards to hot paths

### Session 3+ (P2)
- [ ] Implement connection pooling
- [ ] Implement worker pool for goroutines
- [ ] Refactor context propagation

---

## TESTING STRATEGY

### After P0 Fixes
```bash
# Test compilation
go build ./...

# Test functionality (no new issues)
go test ./...

# Run with race detector
go test -race ./...
```

### Performance Test
```bash
# Monitor connection count during load test
# Should stay stable, not grow unbounded
SELECT COUNT(*) FROM active_connections;

# Monitor goroutine count
# Should peak and return to baseline
runtime.NumGoroutine()
```

---

## PRIORITY ROADMAP

**This Week (P0 - Production Stability)**
- Fix database connection leak (62 locations)
- Fix unchecked errors (38 queries)
- Verify goroutine coordination (4 locations)
- Build + test

**Next Week (P1 - Data Corruption Prevention)**
- Race condition detection and fixes
- Nil pointer guards
- Error handling improvements

**Week 3+ (P2 - Architecture Improvement)**
- Connection pooling refactor
- Goroutine pooling refactor
- Context propagation standardization

---

## SUCCESS CRITERIA

✅ All 62 GetConnection() have defer Close()  
✅ All 38 database queries check errors  
✅ All 4 goroutine locations properly coordinated  
✅ `go test -race ./...` runs with zero race condition warnings  
✅ Build passes: `go build ./...` ✅  
✅ Tests pass: `go test ./...` ✅  
✅ Load test shows stable connection/goroutine counts  

---

**Status**: Ready for implementation  
**Next Step**: Start with Issue #1 (database connection leak) in next session  
**Blocking**: P0 fixes must complete before production deployment

