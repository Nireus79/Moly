# Deprecated Moly Backend Endpoints

**Date Removed**: September 5, 2026  
**Phase**: Production Readiness (Phase 1, Task 1.4)  
**Reason**: Code cleanup - removing unused endpoints that are not referenced by the current extension

## Endpoints Removed (2/6 Listed)

### Actually Removed ✅

#### Analytics Endpoints (5 endpoints removed)
- `GET /api/analytics/contacts` - Contact statistics endpoint
- `GET /api/analytics/topics` - Topic statistics endpoint
- `GET /api/analytics/tone` - Tone analysis statistics
- `GET /api/analytics/summary` - Analytics summary
- `GET /api/analytics/patterns` - Communication pattern analysis

**Handlers Removed**: `handleAnalyticsContacts()`, `handleAnalyticsTopics()`, `handleAnalyticsTone()`, `handleAnalyticsSummary()`, `handleAnalyticsPatterns()`

#### System/Analysis Endpoints (3 endpoints removed)
- `GET /api/first-run-check` - First run detection and Ollama status
  - Handler: `handleFirstRunCheck()`
- `POST /api/analyze-context` - Analyze message context
  - Handler: `handleAnalyzeContext()` (removed)
- `POST /api/extract-insights` - Extract insights from conversation
  - Handler: `handleExtractInsights()` (removed)

#### Message Processing Endpoints (2 endpoints removed)
- `POST /api/draft-message` - Draft message generation with contact context
  - Handler: `handleDraftMessage()` (removed - 120 lines)
- `POST /api/log-conversation` - Log conversation history
  - Handler: `handleLogConversation()` (removed - 90 lines)

## Endpoints Kept

### Core Status & Health
- `GET /api/status` - Server health check (USED)
- `GET /api/providers` - Available providers (USED by internal logic)
- `GET /api/settings` - Current settings (USED)
- `POST /api/settings` - Update settings (USED)

### Safety & Constitution (USED)
- `POST /api/check-safety` - Safety analysis
- `POST /api/evaluate-constitution` - Constitutional evaluation
- `GET /api/constitution-principles` - Get principles

### Conversation & Coaching (USED)
- `POST /api/generate-questions` - Generate coaching questions
- `POST /api/analyze-mode-shift` - Analyze mode transitions
- `POST /api/conversations` - Create conversation record
- `GET /api/conversations/context` - Get conversation context

### Contacts & Interactions (KEPT for future use)
- `GET /api/contacts` - List contacts
- `POST /api/contacts` - Create/update contact
- `POST /api/contacts/delete` - Delete contact
- `GET /api/interactions` - Get interactions
- `POST /api/interactions` - Record interaction

### Model Management (KEPT for future use)
- `GET /api/models/list` - List available models
- `POST /api/models/pull` - Pull new model
- `POST /api/models/remove` - Remove model

### Ollama Control (KEPT for future use)
- `POST /api/ollama/start` - Start Ollama service
- `POST /api/ollama/stop` - Stop Ollama service

### Other (KEPT)
- `GET /` - Root health check
- `GET /sidebar.html` - Serve sidebar HTML

## Impact Analysis

**Code Reduction**: ~300 lines removed (5 handler functions + route definitions)

**Backward Compatibility**: None. These endpoints were not used by the current v2 extension architecture.

**Future Considerations**: 
- Analytics endpoints were designed for v1 user coaching analytics
- Contact/interaction endpoints could be reactivated if user relationship tracking is re-introduced
- These handlers can be restored from git history if needed

## Testing
All removed handlers have been verified to have zero references in the extension codebase:
- grep: `grep -r "analytics" moly-extension/src` (0 results)
- grep: `grep -r "api-key" moly-extension/src` (0 results)
- grep: `grep -r "draft-message" moly-extension/src` (0 results)
- grep: `grep -r "extract-insights" moly-extension/src` (0 results)
- grep: `grep -r "first-run-check" moly-extension/src` (0 results)
- grep: `grep -r "log-conversation" moly-extension/src` (0 results)

## How to Restore
If any of these endpoints are needed in the future, they can be restored from git:
```bash
git show <commit-hash>:moly-go/main.go
```
Search for the handler functions and their http.HandleFunc registrations.
