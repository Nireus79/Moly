# Phase 1.2: Backend Agent Integration
**Duration**: Sep 20 - Oct 4, 2026 (2 weeks)  
**Status**: Starting Sep 7, 2026 (early start - Phase 1.1 complete)

---

## Goals
1. Connect extension to backend v2 agents
2. Implement feature flags for gradual rollout
3. Graceful fallback to old logic
4. A/B testing infrastructure
5. Monitoring & error handling

---

## Architecture: Dual Path

```
Extension User Request
    ↓
[Feature Flag Check]
    ├─→ New Path (10-100%): /api/v2/conversation/generate
    │   └─→ Agent System (Conversation → Learning → Context → Risk)
    │
    └─→ Old Path (0-90%): Inline logic
        └─→ Direct Claude API call
```

---

## Implementation Tasks

### Week 1: V2 API Client (Sep 20-26)

#### 1. Create V2AgentClient (`src/api/v2AgentClient.ts`)
- [ ] Connect to backend /api/v2/ endpoints
- [ ] Request/response types
- [ ] Error handling & retries
- [ ] Timeout management (2s timeout)
- [ ] Request logging

#### 2. Feature Flags (`src/config/featureFlags.ts`)
- [ ] Enable/disable V2 agents
- [ ] Percentage rollout (0-100%)
- [ ] User-based targeting
- [ ] Override via settings

#### 3. Update BackendManager (`src/api/backendManager.ts`)
- [ ] Dual-path routing
- [ ] Feature flag integration
- [ ] Fallback logic
- [ ] Performance metrics

#### 4. Error Handling & Fallback
- [ ] Network error recovery
- [ ] Timeout fallback to old logic
- [ ] Partial failure handling
- [ ] Error reporting

### Week 2: Testing & Integration (Sep 27 - Oct 4)

#### 5. Integration Tests (`src/api/__tests__/`)
- [ ] V2AgentClient tests
- [ ] Feature flag tests
- [ ] Fallback mechanism tests
- [ ] End-to-end integration

#### 6. Monitoring & Alerting
- [ ] Success/failure rates
- [ ] Latency metrics
- [ ] Error categorization
- [ ] User feedback collection

#### 7. A/B Testing Setup
- [ ] Bucket assignment
- [ ] Metrics comparison
- [ ] Segment analysis

---

## Files to Create/Modify

### New Files
```
src/api/v2AgentClient.ts         ← V2 backend client
src/config/featureFlags.ts        ← Feature flag management
src/types/v2ApiTypes.ts           ← API request/response types
src/api/__tests__/v2AgentClient.test.ts
src/config/__tests__/featureFlags.test.ts
```

### Modified Files
```
src/api/backendManager.ts         ← Add dual-path routing
src/hooks/useMolyAgent.ts         ← Update to use feature flags
src/settings/Settings.tsx         ← Add V2 agent settings
```

---

## Feature Flag Strategy

### Levels
1. **Global**: Enable/disable all V2 features
2. **User Percentage**: 0%, 10%, 25%, 50%, 75%, 100%
3. **User Specific**: Override for testing users
4. **Settings Override**: User can enable beta features

### Rollout Schedule
- Sep 20: 0% (disabled)
- Sep 23: 10% (internal testing)
- Sep 27: 25% (expanded testing)
- Oct 1: 50% (half users)
- Oct 4: 100% (full rollout)

---

## Fallback Mechanism

### Triggers
- Network error
- Timeout (2s)
- HTTP error (5xx)
- Missing required context

### Fallback Behavior
1. Log event with metrics
2. Fallback to old logic
3. Notify user of degraded mode
4. Report to backend

---

## Success Criteria

- [ ] Extension connects to backend agents
- [ ] Feature flags working (0% → 100%)
- [ ] Fallback mechanism tested
- [ ] Error rate < 1%
- [ ] Latency p95 < 2s
- [ ] All tests passing
- [ ] Monitoring dashboard live
- [ ] Zero user-facing bugs in testing

---

## Testing Plan

### Manual Testing
1. Test with feature flag at 0%, 10%, 50%, 100%
2. Force error conditions (network down, timeout)
3. Verify fallback behavior
4. Check metrics accuracy

### Automated Testing
1. Unit tests for V2AgentClient
2. Unit tests for feature flags
3. Integration tests
4. End-to-end tests

---

## Monitoring Metrics

### Key Metrics
- V2 request success rate
- V2 response latency (p50, p95, p99)
- Fallback rate (when fallback triggered)
- Error categories
- Feature flag distribution

### Dashboards
- Real-time agent performance
- Error rate by endpoint
- User cohort comparison
- Fallback analysis

---

## Rollback Plan

If error rate > 2%:
1. Reduce feature flag to 50%
2. Investigate root cause
3. If no fix found, disable completely
4. Revert to 100% old logic

---

## Success Metrics for Phase

✅ 10% users on V2 agents by Sep 23  
✅ Error rate < 1%  
✅ Latency p95 < 2s  
✅ No critical bugs  
✅ Fallback mechanism working  

---

## Next Steps (Phase 1.3)
- Gradual rollout from 10% → 100%
- Monitor metrics continuously
- Iterate on agent prompts
- Gather user feedback
