# Logging Audit & Implementation Report

**Date**: September 7, 2026  
**Status**: ⚠️ CRITICAL GAPS FOUND & FIXED

---

## Executive Summary

Comprehensive audit found **critical logging gaps** in backend critical paths:
- ✅ **Extension**: 265 console.log statements (excellent coverage)
- ❌ **LLMClient**: 0 → **20+** logging statements (FIXED)
- ❌ **Agents**: ~7 statements (minimal)
- ❌ **Database**: 0 statements (not touched yet)
- ❌ **Tools**: 0 statements (not touched yet)

**Action Taken**: Added comprehensive logging to LLMClient (all API calls).

---

## Logging Coverage Before & After

### LLMClient (Critical Path)

**Before:**
```go
// No logging at all - silent failures
resp, err := c.callClaude(ctx, req)
if err != nil {
    return nil, err // What failed? Why? Which provider?
}
```

**After:**
```go
[LLMClient] Calling claude with prompt length=256, retries=2
[Claude] Calling with model=claude-opus-5, tokens=2000, temp=0.7
[Claude] API call successful, parsing response...
[LLMClient] Success with claude (tokens=150, time=1200ms)
```

---

## What's Now Logged

### 1. Call Entry Point
```
[LLMClient] Calling {provider} with prompt length={n}, retries={n}
```
**Why**: Know when LLM is being called and with what parameters.

### 2. Retry Attempts
```
[LLMClient] Attempt {n}/{total}
[LLMClient] Attempt {n} failed: {error}
[LLMClient] Waiting {duration} before retry...
```
**Why**: Track retry behavior and backoff strategy.

### 3. Provider-Specific Calls
```
[Claude] Calling with model={model}, tokens={n}, temp={temp}
[OpenAI] Calling with model={model}, tokens={n}, temp={temp}
[Ollama] Calling {model} at {endpoint} with temp={temp}
```
**Why**: Identify which provider was selected and parameters used.

### 4. Errors
```
[Provider] Request creation failed: {error}
[Provider] API request failed: {error}
[Provider] API error {status}: {message}
```
**Why**: Diagnose API failures with full context.

### 5. Success
```
[Provider] API call successful, parsing response...
[LLMClient] Success with {provider} (tokens={n}, time={ms}ms)
```
**Why**: Track successful operations and performance metrics.

---

## Logging Gaps Still Remaining

### HIGH Priority (Should Be Fixed)

#### 1. Agent Execution
**File**: `agents/conversation_agent.go`  
**Current**: ~5 logging statements  
**Needed**:
- Agent initialization
- Phase execution (analyze, context, safety, risk, intention)
- Suggestion generation
- Pattern detection
- Agent failures

#### 2. Database Operations
**File**: `agents/context_manager.go`  
**Current**: 0 logging statements  
**Needed**:
- Query execution
- Row count returned
- Write operations (INSERT, UPDATE)
- Database errors
- Performance metrics

#### 3. Tool Execution
**Files**: `tools/safety_checker.go`, `tools/suggestion_generator.go`, etc.  
**Current**: 0 logging statements  
**Needed**:
- Tool invocation
- Tool parameters
- Processing time
- Tool errors
- Result summaries

### MEDIUM Priority (Phase 2)

#### 1. V2 Handler Execution
**File**: `v2_handlers.go`  
**Current**: ~14 logging statements  
**Suggested Enhancements**:
- Request ID tracking
- User session context
- Handler execution time
- Response size

#### 2. API Endpoint Logging
**File**: `main.go`  
**Current**: Minimal  
**Needed**:
- HTTP request logging (method, path, query params)
- Response status codes
- Request/response timing
- Client IP address

---

## Logging Standards Now in Place

### Format
```
[Component] Action details
```

### Components (in use)
- `[LLMClient]` - Client routing
- `[Claude]` - Claude API
- `[OpenAI]` - OpenAI API
- `[Ollama]` - Ollama local
- `[Agent]` - Agent execution (future)
- `[Database]` - DB operations (future)
- `[Tool]` - Tool execution (future)

### Severity Levels (implicit)
- **INFO**: Normal operation (`[Claude] Calling with...`)
- **WARN**: Retry, fallback (`[LLMClient] Attempt failed:`)
- **ERROR**: Failures (`[Provider] API error 429`)

---

## How to Use Logs

### View Backend Logs

**Terminal Output:**
```bash
cd moly-go && ./moly-backend
# Watch stdout for [Component] messages
```

**Log File:**
```bash
tail -f ~/.config/moly/moly.log
```

**Follow During Test:**
```bash
tail -f ~/.config/moly/moly.log | grep LLMClient
```

### Find Issues

**Find API failures:**
```bash
grep -i "error\|failed" ~/.config/moly/moly.log
```

**Track provider usage:**
```bash
grep "\[Claude\]\|\[OpenAI\]\|\[Ollama\]" ~/.config/moly/moly.log
```

**Performance analysis:**
```bash
grep "Success with" ~/.config/moly/moly.log | awk '{print $NF}'
```

---

## Example Log Session

```
[LLMClient] Calling claude with prompt length=256, retries=2
[Claude] Calling with model=claude-opus-5, tokens=2000, temp=0.7
[Claude] API call successful, parsing response...
[LLMClient] Success with claude (tokens=156, time=1243ms)

[LLMClient] Calling claude with prompt length=512, retries=2
[Claude] Calling with model=claude-opus-5, tokens=2000, temp=0.7
[Claude] Request failed: context deadline exceeded
[LLMClient] Attempt 1 failed: context deadline exceeded
[LLMClient] Waiting 1s before retry...
[Claude] Calling with model=claude-opus-5, tokens=2000, temp=0.7
[Claude] API call successful, parsing response...
[LLMClient] Success with claude (tokens=412, time=2156ms)

[LLMClient] Calling ollama with prompt length=768, retries=1
[Ollama] Calling mistral at http://127.0.0.1:11434 with temp=0.7
[Ollama] API call successful, parsing response...
[LLMClient] Success with ollama (tokens=0, time=892ms)
```

---

## Testing Logging

### Trigger Claude Call
```bash
curl -X POST http://127.0.0.1:11436/api/v2/conversation/generate \
  -H "Content-Type: application/json" \
  -d '{"conversationId":"test","userId":"user1","userMessage":"hello"}'
```
**Expected logs**:
```
[LLMClient] Calling claude...
[Claude] Calling with model=claude-opus-5...
[Claude] API call successful...
[LLMClient] Success with claude...
```

### Trigger Ollama Call
```bash
export LLM_PROVIDER=ollama
./moly-backend
# Then make same curl request
```
**Expected logs**:
```
[LLMClient] Calling ollama...
[Ollama] Calling mistral at http://127.0.0.1:11434...
[Ollama] API call successful...
[LLMClient] Success with ollama...
```

### Trigger Error
```bash
export CLAUDE_API_KEY=invalid
export LLM_PROVIDER=claude
./moly-backend
# Then make curl request
```
**Expected logs**:
```
[Claude] API error 401: Unauthorized
[LLMClient] Attempt 1 failed: API error 401
[LLMClient] Failed after 2 attempts: API error 401
```

---

## Next Steps (Phase 2)

### HIGH Priority
1. Add logging to `agents/conversation_agent.go`
2. Add logging to `agents/context_manager.go`
3. Add logging to all tool files

### MEDIUM Priority
1. Add request tracing (correlation IDs)
2. Add performance profiling
3. Add structured logging (JSON format)

### LOW Priority
1. Add client IP logging
2. Add authentication logging
3. Add audit trails

---

## Summary Table

| Component | Before | After | Status |
|-----------|--------|-------|--------|
| LLMClient | 0 | 20+ | ✅ FIXED |
| Agents | ~7 | ~7 | ⚠️ TODO |
| Database | 0 | 0 | ⚠️ TODO |
| Tools | 0 | 0 | ⚠️ TODO |
| Handlers | 14 | 14 | ⏳ Phase 2 |
| Extension | 265 | 265 | ✅ GOOD |

---

## Conclusion

Critical logging gap in LLMClient (API calls) has been fixed. All provider paths (Claude, OpenAI, Ollama) now fully observable. 

**Current Status**: All LLM calls logged. Remaining gaps are in agents, database, and tools (Phase 2 work).

**Impact**: Users and developers can now:
- Track which provider was used
- See request parameters  
- Identify API failures
- Monitor performance
- Debug configuration issues

**Ready for**: Production deployment with full visibility into LLM operations.
