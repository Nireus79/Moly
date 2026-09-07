# Moly Error Handling & Resilience Specification

**Date**: September 6, 2026  
**Status**: Reference  
**Version**: 1.0

---

## Principles

1. **Never block the user** — Always return something useful
2. **Graceful degradation** — Partial results > no results
3. **Clear error messages** — Help user understand & recover
4. **Transparent logging** — Debug issues quickly
5. **Fail fast** — Don't waste time on failing paths

---

## Error Hierarchy

### Level 1: Immediate Safety Issues (STOP)

**Crisis/Illegal** → Return resources, don't proceed
```
User message contains suicidal language
→ SafetyChecker returns crisis alert
→ Stop all processing
→ Return crisis resources to user
→ User can proceed or get help
```

### Level 2: High Risk (EDUCATE)

**Manipulation/Fraud/Harm** → Educate first, still offer suggestions
```
User wants to manipulate contact
→ RiskMonitor flags as HIGH severity
→ Generate educational questions
→ Return questions + suggestions
→ User thinks it through + chooses
```

### Level 3: Warnings (ADVISE)

**Ethics concerns** → Show principles, proceed
```
User message has minor ethical concern
→ ConstitutionEvaluator notes concern
→ Include principle explanation
→ Return full suggestions
→ User can choose to follow advice or not
```

### Level 4: Technical Issues (RECOVER)

**API/Database/Agent failures** → Use fallback
```
Backend temporarily unavailable
→ Extension detects timeout
→ Shows "Backend busy" message
→ Returns cached suggestions if available
→ User still gets help (just not personalized)
```

---

## Error Response Format

All errors include recovery guidance:

```json
{
  "error": {
    "type": "BACKEND_UNAVAILABLE",
    "message": "Backend is processing other requests. Showing cached suggestions.",
    "severity": "warning",
    "userAction": "try_again",
    "retryAfter": 5,
    "fallback": {
      "available": true,
      "source": "cached",
      "dataAge": "2 minutes old"
    }
  },
  "suggestions": [
    // Cached suggestions from last time
  ]
}
```

---

## Common Errors & Recovery

### 1. Backend Unavailable (504)

**Cause**: Backend down, overwhelmed, or network issue

**User sees**:
```
"Backend is busy. Showing recent suggestions while we reconnect..."
```

**Recovery**:
```typescript
if (backendTimeout) {
  // Try 3 times with exponential backoff
  for (let i = 0; i < 3; i++) {
    try {
      return await callBackend();
    } catch (e) {
      await delay(Math.pow(2, i) * 1000); // 1s, 2s, 4s
    }
  }
  
  // Fall back to cached suggestions
  return getCachedSuggestions();
}
```

**User action**: "Try again" button

---

### 2. API Key Invalid (401)

**Cause**: User's API key wrong, revoked, or quota exceeded

**User sees**:
```
"API key invalid. Check Settings and try a new key."
```

**In Settings**:
```
[!] API Key Error: Invalid or Expired
[Get a new key] → Opens provider website
```

**Recovery**:
```typescript
if (apiKeyError) {
  // Don't retry forever
  settingsStore.setError("API key invalid");
  showSettingsModal();
  
  // Provide direct link to fix it
  redirectTo(getProviderAuthUrl());
}
```

---

### 3. Model Not Found (400)

**Cause**: Selected model doesn't exist or user doesn't have access

**User sees**:
```
"Model not available. Discovering available models..."
```

**Recovery**:
```typescript
if (modelNotFound) {
  // Auto-discover
  const availableModels = await discoverModels(selectedProvider);
  
  if (availableModels.length > 0) {
    // Use first available
    selectModel(availableModels[0]);
    retry();
  } else {
    // No models available
    showError("No models available for this provider");
    redirectToSettings();
  }
}
```

---

### 4. Timeout (504)

**Cause**: Request took > 30 seconds

**User sees** (after 2 seconds):
```
"Processing... (this can take a while on slow systems)"
```

**After 30 seconds**:
```
"Request timed out. Try again?"
[Try Again] [Cancel]
```

**Recovery**:
```typescript
const timeout = 30000; // 30 seconds

try {
  return await Promise.race([
    callBackend(),
    delay(timeout).then(() => {
      throw new TimeoutError();
    })
  ]);
} catch (e) {
  if (e instanceof TimeoutError) {
    return showRetryPrompt();
  }
}
```

---

### 5. Rate Limited (429)

**Cause**: User exceeded rate limit (100 req/min)

**User sees**:
```
"Too many requests. Please wait 30 seconds."
[Status: 28 seconds remaining]
```

**Recovery**:
```typescript
if (rateLimited) {
  const resetTime = response.headers['X-RateLimit-Reset'];
  const waitSeconds = Math.ceil((resetTime - now()) / 1000);
  
  showCountdown(waitSeconds);
  
  // Auto-retry after limit resets
  setTimeout(() => retry(), waitSeconds * 1000);
}
```

---

### 6. Database Error (500)

**Cause**: Database offline, corrupted, or connection lost

**User sees**:
```
"Data service unavailable. Working offline."
```

**Recovery**:
```typescript
if (databaseError) {
  // Switch to offline mode
  useLocalStorage = true;
  
  // Cache current data
  saveToLocalStorage(userData);
  
  // Show what we have
  showCachedConversations();
  
  // Auto-sync when backend returns
  monitorBackendHealth();
}
```

---

### 7. Malformed Request (400)

**Cause**: Developer error, missing required fields

**User sees**: Nothing (should never happen to user)

**Backend logs**:
```
[ERROR] Malformed request: missing field 'conversationId'
[ERROR] Request: { "userId": "...", "userMessage": "..." }
[ERROR] Required fields: conversationId, userId, userMessage, mode, tone
```

**Fix**: Code review, tests, input validation

---

### 8. Reflection Merge Failed (500)

**Cause**: Reflection didn't merge to contact profile properly

**User sees**:
```
"Couldn't save insights. Try again?"
```

**Recovery**:
```typescript
if (reflectionMergeFailed) {
  // Keep reflection in memory
  store.reflection.status = 'pending_retry';
  
  // Show retry button
  showRetryButton();
  
  // Retry on next sync
  onBackendReady(() => {
    retryReflectionMerge();
  });
}
```

---

## Agent-Specific Error Handling

### Conversation Agent Fails

```go
func (a *ConversationAgent) Run(ctx Context) (*Response, error) {
  // If any phase fails, return partial result
  
  result := &Response{
    phase: "partial",
    suggestions: getCachedSuggestions(), // Have something
    questions: getSimpleFallbackQuestions(),
    error: err.Error(),
  }
  
  return result, nil // Don't error out, return what we have
}
```

### Learning Agent Fails

```go
// If learning fails, suggestions still work
// Learning is best-effort

if learningFailed {
  logger.Warn("Learning agent failed, proceeding without personalization")
  
  // Suggestions generated with generic context
  // User still gets help (just less personalized)
  // Try learning again next time
}
```

### Context Manager Fails

```go
// If context retrieval fails, alert user
if contextFailed {
  return &Response{
    error: "Missing About Me. Please complete your profile.",
    suggestions: nil,
    message: "We need to know more about you first.",
  }
}
```

---

## Retry Logic

**Exponential backoff**:
```go
func RetryWithBackoff(fn func() error, maxRetries int) error {
  for i := 0; i < maxRetries; i++ {
    err := fn()
    if err == nil {
      return nil
    }
    
    // Wait before retry: 1s, 2s, 4s, 8s...
    wait := time.Duration(math.Pow(2, float64(i))) * time.Second
    time.Sleep(wait)
  }
  
  return fmt.Errorf("failed after %d retries", maxRetries)
}
```

**When to retry**:
- ✅ Temporary network issue (timeout)
- ✅ Rate limiting (wait, then retry)
- ✅ Service temporarily unavailable
- ✅ Database connection pool exhausted

**Never retry**:
- ❌ Invalid API key (fix first)
- ❌ Malformed request (code error)
- ❌ User not authorized (fix permissions)
- ❌ Resource not found (404)

---

## Logging Errors

**Levels**:
```
DEBUG: Low-level details, decision trees
INFO: Completed operations, normal flow
WARN: Recovered errors, degraded performance
ERROR: Failed operations, user-facing issues
FATAL: System-level failures
```

**Example**:
```go
logger.WithFields(logrus.Fields{
  "userId": userId,
  "conversationId": conversationId,
  "phase": "suggestion_generation",
  "errorType": "timeout",
  "processingTime": 32000, // ms
  "requestId": requestId,
}).Error("Suggestion generation timed out")
```

**Never log**:
- ❌ API keys
- ❌ User messages (personally identifying)
- ❌ Contact names in error logs
- ❌ Sensitive data

---

## User-Facing Error Messages

**Good** (clear, actionable):
```
"API key invalid. Get a new one here: [link]"
"Backend is busy. Trying again in 5 seconds..."
"Missing About Me. Let's set that up: [Go to Settings]"
```

**Bad** (confusing, unhelpful):
```
"Error: ECONNREFUSED"
"500 Internal Server Error"
"Null pointer exception"
```

---

## Monitoring & Alerting

**Track**:
- Error rate by type
- Recovery time
- User impact (how many affected)
- Fallback usage (% of requests using cache)

**Alert when**:
- Error rate > 5% for 5 minutes
- Timeout rate > 10%
- Fallback usage > 30%
- Any database error
- Repeated same error (pattern)

---

## Testing Errors

Every error path should be tested:

```go
func TestBackendUnavailable(t *testing.T) {
  // Mock backend offline
  backend.Close()
  
  // Call should use fallback
  response, err := client.GenerateSuggestions(msg)
  
  // Should have suggestions (cached)
  assert.NotNil(response.Suggestions)
  
  // Should note it's offline
  assert.Contains(response.Message, "offline")
}

func TestAPIKeyInvalid(t *testing.T) {
  // Mock invalid key error
  backend.SetApiKeyError()
  
  // Should redirect to settings
  response := client.GenerateSuggestions(msg)
  assert.Equal("invalid_api_key", response.Error.Type)
  assert.True(response.ShouldRedirectToSettings)
}
```

---

This framework ensures Moly is resilient, helpful, and never leaves users stranded.
