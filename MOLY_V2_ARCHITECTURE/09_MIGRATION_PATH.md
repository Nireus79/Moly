# Moly Migration Path: Current → Agent-Based Architecture

**Date**: September 6, 2026  
**Status**: Implementation Roadmap  
**Version**: 1.0

---

## Current State

- ✅ Extension working (React, Zustand, Chrome sidePanel)
- ✅ Basic conversation history
- ✅ Contact management
- ✅ Hardcoded suggestion generation
- ✅ Basic safety checks
- ❌ No agents (all logic inline)
- ❌ No learning (no behavioral profile)
- ❌ No dynamic questions (templates only)

---

## Target State

- ✅ Extension enhanced (better UX, offline mode)
- ✅ Backend agents orchestrating everything
- ✅ Dynamic question generation
- ✅ User behavioral learning
- ✅ Risk pattern detection
- ✅ Richer context over time

---

## Phase 1: Parallel Backend (Weeks 1-2)

**Goal**: Get backend agents running in parallel without breaking current extension.

### Step 1.1: Implement Backend Agents (Week 1)

```bash
# Create Go backend
mkdir moly-backend
cd moly-backend

# Implement agent framework
touch cmd/server/main.go
touch internal/agents/conversation.go
touch internal/agents/learning.go
touch internal/agents/context_manager.go
touch internal/agents/risk_monitor.go

# Implement tools
touch internal/tools/suggestion_generator.go
touch internal/tools/question_generator.go
touch internal/tools/safety_checker.go

# Implement database layer
touch internal/database/postgres.go
touch internal/database/migrations.go

# Run migrations
go run cmd/migrate/main.go up
```

### Step 1.2: Implement API Endpoints

```go
// cmd/server/main.go
func main() {
  router := setupRouter()
  
  // Health check
  router.GET("/api/status", statusHandler)
  
  // Main conversation endpoint
  router.POST("/api/conversation/generate", conversationHandler)
  
  // Feedback endpoint
  router.POST("/api/conversation/feedback", feedbackHandler)
  
  // Context endpoint
  router.GET("/api/context", contextHandler)
  
  router.Run(":11436")
}
```

### Step 1.3: Database Setup

```bash
# PostgreSQL
createdb moly

# Run migrations
go run cmd/migrate/main.go up

# Verify
psql moly -c "SELECT COUNT(*) FROM information_schema.tables;"
```

### Step 1.4: Test Locally

```bash
# Start backend
go run cmd/server/main.go

# Test endpoint
curl -X POST http://localhost:11436/api/conversation/generate \
  -H "Content-Type: application/json" \
  -d '{
    "conversationId": "test",
    "userId": "test",
    "userMessage": "Tell me about Sarah",
    "mode": "direct",
    "tone": "friendly"
  }'
```

### Step 1.5: Deploy to Staging

```bash
# Docker
docker build -t moly-backend:latest .
docker push moly-backend:latest

# Kubernetes
kubectl apply -f deployment.yaml -n staging

# Verify
kubectl get pods -n staging
curl https://staging-api.moly.dev/api/status
```

**Outcome**: Backend agents running, API working, tests passing. Extension still uses old logic.

---

## Phase 2: Gradual Migration (Weeks 3-4)

**Goal**: Gradually move extension traffic to backend agents while monitoring.

### Step 2.1: Add Backend URL Detection to Extension

```typescript
// src/api/backendManager.ts
export class BackendManager {
  async detectBackendUrl(): Promise<string> {
    // Check if backend is available
    const ports = [11436, 11437, 8000, 5000];
    
    for (const port of ports) {
      try {
        const response = await fetch(`http://localhost:${port}/api/status`);
        if (response.ok) {
          return `http://localhost:${port}`;
        }
      } catch (e) {}
    }
    
    // Fall back to production
    return 'https://api.moly.dev';
  }
}
```

### Step 2.2: Create Feature Flag

```typescript
// src/config/flags.ts
export const FEATURE_FLAGS = {
  USE_BACKEND_AGENTS: localStorage.getItem('useBackendAgents') === 'true',
  BACKEND_URL: await getBackendUrl(),
  FALLBACK_TO_OLD_LOGIC: true, // Safety fallback
};
```

### Step 2.3: Dual-Path Implementation

```typescript
// src/sidebar/Sidebar.tsx
async function handleSendMessage(message: string) {
  if (FEATURE_FLAGS.USE_BACKEND_AGENTS && FEATURE_FLAGS.FALLBACK_TO_OLD_LOGIC) {
    try {
      // Try new agent-based path
      const response = await conversationAPI.generateWithAgents(message);
      displaySuggestions(response.suggestions);
    } catch (error) {
      console.warn('Agent generation failed, falling back to old logic');
      // Fall back to old inline logic
      const oldResponse = await generateSuggestionsLocally(message);
      displaySuggestions(oldResponse);
    }
  } else {
    // Use old logic
    const response = await generateSuggestionsLocally(message);
    displaySuggestions(response);
  }
}
```

### Step 2.4: Gradual Rollout

**Week 3**:
- 10% of users to new backend (opt-in)
- Monitor error rates, latency
- Gather feedback

**Week 4**:
- 50% of users to new backend
- A/B test suggestion quality
- Monitor behavioral metrics

**After validation**:
- 100% to new backend
- Keep fallback for safety

### Step 2.5: Data Migration

```go
// cmd/migrate-data/main.go
// Migrate existing conversation history from local storage to database
func migrateConversations(userId string) error {
  // Load from chrome.storage.local
  // Parse conversation data
  // Insert into PostgreSQL
  // Verify
  return nil
}
```

**Outcome**: Extension talking to backend agents with automatic fallback. Gradual migration underway.

---

## Phase 3: Full Migration (Week 5+)

**Goal**: Sunset old inline logic, fully embrace agents.

### Step 3.1: Remove Old Logic

```typescript
// BEFORE: Two paths
async function generateSuggestions(message) {
  if (useBackendAgents) {
    return await backendAgents.generate(message);
  } else {
    return await inlineLogic.generate(message); // OLD
  }
}

// AFTER: One path
async function generateSuggestions(message) {
  return await backendAgents.generate(message);
}
```

### Step 3.2: Clean Up Extension

```bash
# Remove old code
rm src/api/oldSuggestionGenerator.ts
rm src/utils/oldContextExtractor.ts
rm src/hooks/useOldLogic.ts

# Remove old tests
rm tests/oldLogic.test.ts

# Update documentation
```

### Step 3.3: Optimize for Agents

```typescript
// Simplify extension with agents doing heavy lifting
export const Sidebar = () => {
  // Extension now just:
  // 1. Collects user input
  // 2. Sends to backend
  // 3. Displays response
  // 4. Manages offline fallback
};
```

### Step 3.4: Enable Advanced Features

With agents, now possible:
- ✅ Dynamic question generation
- ✅ Behavioral learning
- ✅ Risk pattern detection
- ✅ Socratic conversation mode
- ✅ Context-aware suggestions
- ✅ Educational safety checks

```typescript
// Now available in extension
// No inline implementation needed
// All via backend agents

export const FEATURES = {
  socraticMode: true,
  learningEnabled: true,
  riskMonitoring: true,
  dynamicQuestions: true,
};
```

**Outcome**: Old logic completely removed, fully agent-driven.

---

## Phase 4: Optimization (Week 6+)

**Goal**: Performance tuning, feature completion, hardening.

### Step 4.1: Performance Optimization

```go
// Agent processing time optimization
// Target: < 1 second for most requests
// Current: 1-3 seconds

// Caching strategies
type Cache struct {
  userProfiles map[string]*UserProfile // TTL: 5 min
  contactProfiles map[string]*Contact // TTL: 10 min
  suggestions map[string][]Suggestion // TTL: 1 min
}
```

### Step 4.2: Quality Improvements

- Refine agent prompts based on real usage
- Improve question generation
- Tune safety thresholds
- Optimize suggestion accuracy

### Step 4.3: Feature Completion

- Reflection modal refinement
- Offline mode enhancement
- Settings UI polish
- Error message clarity

### Step 4.4: Load Testing

```bash
# k6 load test
k6 run tests/load.js

# Apache Bench
ab -n 1000 -c 10 http://localhost:11436/api/status

# Monitor
- API latency p50, p95, p99
- Error rates
- Agent processing times
- Database connection pool
```

**Outcome**: Production-ready, optimized, feature-complete.

---

## Rollback Plan

If agents break: instant fallback to old logic.

```typescript
// In extension
if (backendFailing) {
  FEATURE_FLAGS.USE_BACKEND_AGENTS = false;
  reload(); // Restart with old logic
}

// Metrics watched:
// - API error rate > 5% → Rollback
// - Latency p95 > 5s → Rollback  
// - Agent failures > 100 per min → Rollback
```

---

## Data Consistency During Migration

**Challenge**: User's local data vs. backend data

**Solution**:
```typescript
// Phase 1: Parallel read
// Backend has its own DB
// Extension keeps local data
// No sync issues yet

// Phase 2: Gradual sync
// Pull server-side about_me on login
// Merge with local data
// Use server as source of truth

// Phase 3: Full sync
// All reads from server
// Local data backup only
```

---

## Success Criteria

Each phase must pass before proceeding:

**Phase 1**:
- ✅ Backend running, all endpoints working
- ✅ Database migrations successful
- ✅ Manual testing passed
- ✅ Error rates < 1%

**Phase 2**:
- ✅ 10% users on backend, no issues
- ✅ Feature parity with old logic
- ✅ Latency acceptable (< 2s)
- ✅ Fallback mechanism works

**Phase 3**:
- ✅ 100% users on backend
- ✅ Old logic removed
- ✅ Monitoring stable
- ✅ No rollback needed

**Phase 4**:
- ✅ Performance optimized
- ✅ All features working
- ✅ User feedback positive
- ✅ Production hardened

---

## Timeline

| Phase | Duration | Start | End | Status |
|-------|----------|-------|-----|--------|
| 1: Backend Implementation | 2 weeks | Sep 6 | Sep 20 | To do |
| 2: Gradual Migration | 2 weeks | Sep 20 | Oct 4 | To do |
| 3: Full Migration | 1 week | Oct 4 | Oct 11 | To do |
| 4: Optimization | 2+ weeks | Oct 11 | Oct 25 | To do |

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Backend bugs | Broken suggestions | Gradual rollout, fallback |
| Data loss | User data gone | Daily backups, verification |
| Performance degradation | Slow experience | Load testing, caching |
| Agent hallucinations | Bad suggestions | Safety checks, testing |
| Database issues | Data unavailable | Connection pooling, health checks |

---

This plan enables a safe, gradual transition from inline logic to intelligent agents.
