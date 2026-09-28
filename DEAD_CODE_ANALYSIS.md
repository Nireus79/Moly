# Dead Code Analysis Report

**Date**: Sept 28, 2026  
**Status**: Comprehensive audit of unused/dead code

---

## Summary

| Category | Count | LOC | Action |
|----------|-------|-----|--------|
| **DEAD CODE** | 1 | 870 | DELETE |
| **INCOMPLETE FEATURES** | 3 | ~400 | DELETE |
| **DEBUG TOOLS** | 6 | ~450 | KEEP (optional) |

---

## CATEGORY 1: DEAD CODE (DELETE)

### ❌ mode_transition.go (870 LOC)
- **Status**: Completely unused, 0 references
- **Purpose**: Relationship mode transition analysis (professional→romantic, etc.)
- **Why Dead**: 
  - Out of scope (Moly is general communication coach, not dating specialist)
  - Never integrated into 11-layer architecture
  - Not part of product vision
  - Speculative future feature that was abandoned
- **Action**: **DELETE**
- **Risk**: None - no code depends on it

---

## CATEGORY 2: INCOMPLETE FEATURES (DELETE)

### ❌ handleCodeBasedLogin (lines 4509-4600, ~90 LOC)
- **Status**: Defined but NOT WIRED
- **Purpose**: Alternative auth method (code-based login for devices)
- **Why Dead**:
  - Complete auth system already exists and wired:
    - ✅ `/api/auth/register` - userAuthServer.RegisterHandler
    - ✅ `/api/auth/login` - userAuthServer.LoginHandler  
    - ✅ `/api/auth/verify` - userAuthServer.VerifyTokenHandler
    - ✅ `/api/auth/logout` - userAuthServer.LogoutHandler
  - Frontend uses standard auth, not code-based
  - Incomplete feature (codes generated but can't be consumed)
- **Related Dead Code**:
  - `handleGenerateAuthCode` (~90 LOC, WIRED but useless)
  - `handleValidateAuthCode` (~80 LOC, NOT WIRED)
  - `login_codes` database table (created but never used in production flow)
- **Action**: **DELETE all three handlers + drop login_codes table**
- **Risk**: None - only generates codes that can't be used

### ❌ handleGenerateAuthCode (lines 4459-4507, ~50 LOC)
- **Status**: WIRED to `/api/generate-auth-code` but generates codes nobody consumes
- **Why**: handleCodeBasedLogin never wired, so codes are dead data
- **Action**: DELETE

### ❌ handleValidateAuthCode (lines 4601-4640, ~40 LOC)
- **Status**: Defined but NOT WIRED
- **Why**: Part of incomplete auth feature
- **Action**: DELETE

**Total Incomplete Auth Code**: ~260 LOC

---

## CATEGORY 3: DEBUG/DIAGNOSTIC TOOLS (KEEP - Optional)

These are wired and useful for development/debugging, but not part of production flow.

### ✅ handleCheckSafety (~50 LOC, WIRED)
- Endpoint: `/api/check-safety`
- Purpose: Debug safety checker on individual messages
- Frontend usage: 0 references
- Production use: No
- Value: Useful for testing safety layer in isolation
- **Action**: KEEP (optional debug tool)

### ✅ handleEvaluateConstitution (~50 LOC, WIRED)
- Endpoint: `/api/evaluate-constitution`
- Purpose: Debug constitutional evaluator
- Frontend usage: 0 references
- **Action**: KEEP (optional debug tool)

### ✅ handleAnalyzeModeShift (~50 LOC, WIRED)
- Endpoint: `/api/analyze-mode-shift`
- Purpose: Debug mode shift detection
- Frontend usage: 0 references
- **Action**: KEEP (optional debug tool)

### ✅ handleGenerateQuestions (~50 LOC, WIRED)
- Endpoint: `/api/generate-questions`
- Purpose: Debug question generation
- Frontend usage: 0 references
- **Action**: KEEP (optional debug tool)

### ✅ handleGetPrinciples (~20 LOC, WIRED)
- Endpoint: `/api/constitution-principles`
- Purpose: Get ethics principles list
- Frontend usage: 0 references
- **Action**: KEEP (might be useful for UI)

### ✅ handleFrontendErrors (~30 LOC, WIRED)
- Endpoint: `/api/frontend-errors`
- Purpose: Collect frontend errors for debugging
- Frontend usage: 1 reference
- **Action**: KEEP (useful for error tracking)

**Total Debug Tools**: ~250 LOC (useful for development)

---

## Recommended Actions

### IMMEDIATE (High Priority)
Delete these files/functions:

```bash
# Delete files
rm moly-go/mode_transition.go (870 LOC)

# Delete incomplete auth handlers from main.go
# Remove: handleGenerateAuthCode (lines 4459-4507)
# Remove: handleCodeBasedLogin (lines 4509-4600)
# Remove: handleValidateAuthCode (lines 4601-4640)

# Remove from router:
# Remove: http.HandleFunc("/api/generate-auth-code", ...)
# (handleValidateAuthCode is not wired, so no router removal needed)

# Drop database table:
# DROP TABLE login_codes;
```

**Total Code Removal**: 1,130 LOC

### OPTIONAL (Nice to Have)
Consider removing debug endpoints if you want a cleaner production API:
- `/api/check-safety`
- `/api/evaluate-constitution`
- `/api/analyze-mode-shift`
- `/api/generate-questions`

Leave `/api/frontend-errors` and `/api/constitution-principles` as they're still potentially useful.

---

## Impact Analysis

### Removing mode_transition.go
- ✅ No impact - 0 references
- ✅ No dependencies
- ✅ No production code affected

### Removing incomplete auth feature
- ✅ No impact - handlers not in production flow
- ⚠️ Must also drop `login_codes` table from database
- ✅ No frontend code affected
- ✅ Existing auth system (`userAuthServer`) fully functional

### Removing debug endpoints
- ✅ No impact - 0 production references
- ✅ Frontend still works normally
- ✅ Testing can use direct database or unit tests
- ⚠️ Will lose convenience endpoints for manual testing

---

## Build Impact

After removing all dead code:
- ✅ Build still clean (no dependencies)
- ✅ All production code remains functional
- ✅ No API breaking changes (debug endpoints rarely used)
- ✅ Codebase cleaner and easier to maintain

**Estimated LOC Reduction**: 1,130 LOC (dead code)

---

## Recommendation

**REMOVE IMMEDIATELY:**
- ✅ mode_transition.go (clear dead code)
- ✅ handleCodeBasedLogin, handleGenerateAuthCode, handleValidateAuthCode (incomplete feature, proper auth already in place)
- ✅ login_codes database table (orphaned)

**KEEP:**
- ✅ Debug endpoints (useful for development, can be documented as developer tools)
- ✅ Current auth system (userAuthServer)
- ✅ All production functionality

---

**Estimated Cleanup Time**: 30-45 minutes (code removal + migration)  
**Risk Level**: LOW (isolated changes, no dependencies)  
**Build Impact**: POSITIVE (cleaner codebase)

