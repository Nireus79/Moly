# DETAILED IMPLEMENTATION GUIDE: Phases 1-4 with Feature Flags

**Status**: Ready to Implement  
**Date**: Sept 30, 2026  
**Approach**: Feature flag based, staged rollout (10% → 50% → 100%)

---

## Overview

This guide implements Phases 1-4 EXACTLY as specified in `IMPLEMENTATION_PLAN_PHILOSOPHY_FIRST.md` with:
- Feature flags for each phase
- Staged rollout strategy
- Performance monitoring
- SLA thresholds with rollback triggers
- Comprehensive logging

---

## Phase 1: Extraction Lock - Detailed Implementation

### Step 1A: Feature Flag Integration

**File**: `config/feature_flags.go` ✅ CREATED

**Usage in main.go**:
```go
// At top of message processing
flags := config.GetFeatureFlags()

if flags.UseExtractionLock {
    log.Printf("[Phase1] Extraction lock enabled (staged rollout)")
    // Use extraction lock flow
} else {
    log.Printf("[Phase1] Extraction lock disabled (using fallback)")
    // Use fallback (old re-parsing flow)
}
```

### Step 1B: Extract and Lock Implementation

**File**: `agents/extraction_phase.go` ✅ ALREADY HAS

**Code pattern**:
```go
// Extract once
entities, conflicts := extractionPhase.Run(ctx, input)
extraction := &ExtractionArtifact{
    ID:       generateID(),
    Entities: entities,
    Conflicts: conflicts,
}

// LOCK IT (permanent)
extraction.Lock("extraction complete")

// Now it's immutable - all downstream uses same extraction
```

### Step 1C: Remove Re-Parsing Paths (main.go)

**Location**: Lines 687-800 (according to plan)

**Current state**: Extraction is called but may have fallbacks

**Changes needed**:
1. Find all calls to `LinguisticParser`
2. Remove fallback re-parsing paths
3. If extraction fails, return error (don't retry with parser)
4. Use locked extraction only

**Implementation**:
```go
// BEFORE: Had fallback logic
epOutput, err := srv.extractionPhase.Run(ctx, epInput)
if err != nil {
    // OLD: Try LinguisticParser fallback
    // linguisticEntities := linguisticParser.Parse(message)
}

// AFTER: No fallback, extraction is authoritative
lockedExtraction, err := srv.extractionPhase.Run(ctx, epInput)
if err != nil {
    log.Printf("[Phase1] FATAL: Extraction failed - cannot proceed (no fallback)")
    return errors.New("extraction required but failed")
}

// Verify extraction is locked
if !lockedExtraction.IsLocked {
    log.Printf("[Phase1] ERROR: Extraction not locked!")
    return errors.New("extraction must be locked")
}

// Use locked extraction for all downstream
```

### Step 1D: Update clarification_capture.go

**Location**: `database/clarification_capture.go` (as per plan)

**Remove**: All LinguisticParser usage and re-parsing logic

**Keep**: Use locked extraction only

```go
// BEFORE: Had multi-pass parsing in clarification flow
func ProcessClarificationWithExtractionArtifact(artifact *ExtractionArtifact) {
    // OLD: Re-parse if subject unclear
    if entity.Subject == "" {
        reparsedEntity := linguisticParser.Parse(message)
    }
}

// AFTER: Use extraction as-is
func ProcessClarificationWithExtractionArtifact(artifact *ExtractionArtifact) {
    for _, entity := range artifact.Entities {
        // Use extraction directly, no re-parsing
        if entity.Subject == "" {
            // Ask clarification: "Who is this about?"
            // Don't retry parsing
        }
    }
}
```

### Step 1E: Monitoring Integration

**File**: `monitoring/metrics.go` ✅ CREATED

**Integration in main.go**:
```go
metrics := monitoring.GetMetrics()

// Record extraction lock success
if flags.UseExtractionLock && lockedExtraction.IsLocked {
    metrics.RecordExtractionLockSuccess()
    log.Printf("[Phase1] ✓ Extraction locked: %s", lockedExtraction.ID)
}

// Record extraction time
start := time.Now()
epOutput := srv.extractionPhase.Run(ctx, epInput)
ms := time.Since(start).Milliseconds()
metrics.RecordExtractionTime(ms)
```

### Step 1F: Performance Monitoring

**Baseline Metrics** (before Phase 1):
```
- Extraction time: ~200ms
- Multi-pass calls: ~7 per message
- Subject attribution accuracy: 95%
```

**Phase 1 Targets**:
```
- Extraction time: <200ms (maintained)
- Multi-pass calls: 0/msg (no re-parsing)
- Subject attribution accuracy: 95%+ (maintained)
```

**Rollback Triggers**:
```
- Subject attribution accuracy drops below 94% → ROLLBACK
- Extraction time increases > 25% → ROLLBACK
- Multi-pass calls > 1/msg → ROLLBACK
```

### Step 1G: Deployment Strategy

**Week 1: 10% Rollout**
```bash
# Enable for 10% of users
export MOLY_USE_EXTRACTION_LOCK=true
# With canary deployment or feature flag at request level

# Monitor for 48 hours
- Error rate: should be 0%
- Extraction time: should match baseline
- No regressions reported
```

**Week 2: 50% Rollout**
```bash
# If 10% successful, expand to 50%
# Keep close monitoring: 1 hour checks

# Decision gate:
- If any rollback trigger hit → ROLLBACK
- If stable → proceed to 100%
```

**Week 3: 100% Rollout**
```bash
# All users on extraction lock
# Continue monitoring for 1 week

# After 1 week: declare permanent
```

---

## Phase 2: Layer 5 Conflict Channeling - Detailed Implementation

### Step 2A: Feature Flag

**File**: `config/feature_flags.go` ✅ HAS `UseLayer5ConflictGate`

### Step 2B: ConversationAgent Wiring

**Location**: `agents/conversation_agent.go`

**Required changes**:
```go
// Add Layer 5 handler field
type ConversationAgent struct {
    // ... existing fields ...
    layer5Handler *Layer5ConflictHandler
    clarificationHistory *ClarificationHistory
}

// Add setter
func (ca *ConversationAgent) SetLayer5ConflictHandler(h *Layer5ConflictHandler) {
    ca.layer5Handler = h
}

// In Run() method, ADD after Layer 4:
// ⭐ [Layer 5] CONFLICT DETECTION & RESOLUTION
if ca.layer5Handler != nil && len(epOutput.Conflicts) > 0 {
    log.Printf("[ConversationAgent] [Layer 5] %d conflicts detected", len(epOutput.Conflicts))
    
    conflictQ, hasQuestion := ca.layer5Handler.ProcessConflicts(
        ctx,
        epOutput.Conflicts,
        ctx.AboutMe.UserID,
        ctx.ConversationID,
    )
    
    if hasQuestion {
        response.Response = conflictQ.Question
        response.Metadata["layer5Conflict"] = true
        return response, nil
    }
}
```

### Step 2C: Database Migrations

**File**: `database/migrations/027_add_layer5_tracking.sql`

```sql
ALTER TABLE clarification_questions ADD COLUMN (
    conflict_id TEXT,
    conflict_previous_value TEXT,
    conflict_current_value TEXT,
    layer TEXT DEFAULT '4'
);

CREATE TABLE conflict_resolutions (
    id UUID PRIMARY KEY,
    conflict_id TEXT,
    user_id TEXT NOT NULL,
    conversation_id TEXT,
    user_explanation TEXT,
    resolved_at TIMESTAMP
);

CREATE INDEX idx_clarifications_conflict ON clarification_questions(conflict_id);
CREATE INDEX idx_clarifications_layer ON clarification_questions(layer);
```

### Step 2D: Monitoring Thresholds

**Phase 2 Targets**:
```
- Conflicts generating questions: 100% (vs current 70%)
- Question deduplication rate: > 95%
- Response latency: < 3.5s (alert), < 4s (rollback)
```

**Rollback Triggers**:
```
- Conflicts generating questions < 95%
- Deduplication rate < 90%
- Response latency p95 > 3.5s
- User complaint rate > 3 per 1000 users
```

### Step 2E: Deployment (Week 4-6)

```
Week 4:  10% users (48 hours monitoring)
Week 5:  50% users (1 week monitoring)
Week 6:  100% users (permanent)
```

---

## Phase 3: Constrained Response Generation - Detailed Implementation

### Step 3A: ConstrainedResponseGenerator

**File**: `tools/constrained_response_generator.go` (NEW)

**Implementation** (as per plan):
```go
type ConstrainedResponseGenerator struct {
    llmClient        LLMProvider
    baseResponseGen  *ResponseGenerator
    db               *Database
    validator        *ResponseValidator
}

// Generate with constraints
func (crg *ConstrainedResponseGenerator) Generate(ctx context.Context, ...) (*ConversationResponse, error) {
    // Step 1: Build constraints from facts
    constraints := crg.buildConstraints(userProfile, contacts, extracted)
    
    // Step 2: Pass to LLM in system prompt
    systemPrompt := crg.buildConstrainedSystemPrompt(constraints)
    
    // Step 3: Generate response WITH constraints
    response := crg.baseResponseGen.GenerateWithSystemPrompt(
        ctx, userMessage, userProfile, contacts, systemPrompt)
    
    // Step 4: Validate response
    violations := crg.validateResponse(response.Response, constraints)
    if violations > 0 {
        // Don't return bad response - ask clarification instead
        return crg.generateClarificationInstead()
    }
    
    // Step 5: Return validated response
    return response, nil
}
```

### Step 3B: Constraint Caching

**File**: `tools/constraint_cache.go` (NEW)

```go
type ConstraintCache struct {
    cache map[string][]Constraint
    ttl   time.Duration
    mu    sync.RWMutex
}

// Get constraints (cached)
func (cc *ConstraintCache) GetConstraints(userID, conversationID string) []Constraint {
    key := userID + ":" + conversationID
    cached, ok := cc.cache[key]
    if ok {
        return cached // < 10ms
    }
    // Miss - will rebuild (200ms)
    return nil
}
```

**Expected cache hit rate**: > 70% (saves 80% of constraint building time)

### Step 3C: Feature Flag Integration

**Locations**:
- ResponseGenerator: Check flag before validation
- ConversationAgent: Route to ConstrainedResponseGenerator if flag enabled

### Step 3D: A/B Testing Setup

**Split traffic**:
```
50% users: ConstrainedResponseGenerator (treatment)
50% users: Basic ResponseGenerator (control)

Measure:
- Role-reversal bugs (treatment should be 0%)
- Response quality ratings
- User satisfaction
```

### Step 3E: Monitoring Thresholds

**Phase 3 Targets**:
```
- Role-reversal bugs: 0 (vs 0.1% baseline)
- Constraint violations: < 5%
- Cache hit rate: > 70%
- Response latency: < 3.5s (alert), < 4s (rollback)
```

**Rollback Triggers**:
```
- Role-reversal bugs > 0 per 1000 users
- Constraint violations > 8%
- Cache hit rate < 60%
- Response latency p95 > 3.5s
- False positive blocks > 10%
```

### Step 3F: Deployment (Week 7-9)

```
Week 7:  10% users (48 hours)
Week 8:  50% users (1 week) + A/B test
Week 9:  100% users (permanent)
```

---

## Phase 4: Clean Schema Migration - Detailed Implementation

### Step 4A: Feature Flag

**File**: `config/feature_flags.go` ✅ HAS `UseCleanSchema`

**One-time migration flag** (not staged rollout)

### Step 4B: Pre-Migration Validation

**File**: `database/migration_validator.go` (NEW)

```go
func (mv *MigrationValidator) ValidateReadiness() error {
    // Check old schema is accessible
    // Check new schema can be created
    // Check export works
    // Check import works
    // Estimate migration time
    return nil
}
```

### Step 4C: Deployment Steps

**See**: `PHASE_4_CUTOVER_PLAN.md` (already created)

Summary:
- Day 1-3: Testing
- Day 4: Production (30 min downtime)
- Day 5: Verification

### Step 4D: Monitoring

```
- Migration time: < 5 minutes
- Data loss: 0 records
- Query latency: p95 < 100ms
- Error rate: 0%
```

**Rollback triggers**:
```
- Migration time > 10 minutes
- Data loss > 0 records
- Query latency p95 > 150ms
- Error rate > 1%
```

---

## SLA Monitoring & Rollback Triggers

### Consolidated SLA Thresholds

| Phase | Metric | Alert | Rollback |
|-------|--------|-------|----------|
| 1 | Extraction latency p95 | > base + 20% | > base + 30% |
| 1 | Multi-pass calls | > 1/msg | > 2/msg |
| 1 | Attribution accuracy | < 94% | < 93% |
| 2 | Response latency p95 | > base + 20% | > base + 30% |
| 2 | Conflict questions | < 95% | < 90% |
| 2 | Question repeats | > 2% | > 5% |
| 3 | Response latency p95 | > base + 20% | > base + 30% |
| 3 | Role-reversal bugs | > 0.05% | > 0.1% |
| 3 | Constraint violations | > 8% | > 10% |
| 4 | Migration time | > 5min | > 10min |
| 4 | Data loss | > 0 | > 0 |
| 4 | Query latency p95 | > 100ms | > 150ms |

### Automatic Rollback Procedure

```go
// In production monitoring
if metricsExceedThreshold(metric, rollbackThreshold) {
    log.Printf("[ROLLBACK] %s exceeded threshold, initiating rollback", metric)
    flags.SetFlag("Phase" + phase, false)  // Disable feature
    alertOncall()
    recordIncident()
}
```

---

## Deployment Checklist (All Phases)

### Pre-Deployment
- [ ] All code reviewed
- [ ] All tests passing (100%)
- [ ] Monitoring configured
- [ ] Alerting rules tested
- [ ] Runbooks prepared
- [ ] Team trained
- [ ] Rollback tested

### During Deployment
- [ ] Feature flag enabled at low % (10%)
- [ ] Metrics dashboard open
- [ ] On-call monitoring
- [ ] Hourly check-ins (first 8 hours)
- [ ] Escalation path clear

### Post-Deployment
- [ ] Metrics stable for 24 hours
- [ ] No regressions reported
- [ ] Expand to next %
- [ ] Continue monitoring (7 days)
- [ ] Declare permanent

---

## Files to Create/Modify

### NEW FILES
- `config/feature_flags.go` ✅ DONE
- `monitoring/metrics.go` ✅ DONE
- `tools/constrained_response_generator.go` ⏳ TODO
- `tools/constraint_cache.go` ⏳ TODO
- `database/migration_validator.go` ⏳ TODO

### FILES TO MODIFY
- `main.go` - Remove re-parsing paths, add feature flag checks
- `agents/extraction_phase.go` - Already has lock
- `agents/conversation_agent.go` - Add Layer 5 wiring
- `database/clarification_capture.go` - Remove multi-pass logic
- `tools/response_generator.go` - Add ConstrainedResponseGenerator integration

---

## Success Criteria

✅ **Phase 1**: Zero multi-pass parsing, 95%+ accuracy  
✅ **Phase 2**: 100% conflicts generate questions, < 1% repeats  
✅ **Phase 3**: Zero role-reversal bugs, < 5% violations  
✅ **Phase 4**: 0% data loss, < 5 min migration  

---

## Timeline

```
Week 1: Phase 1 implementation + testing
Week 2: Phase 1 deployment (10% → 50% → 100%)
Week 3: Phase 2 implementation + testing
Week 4: Phase 2 deployment
Week 5: Phase 3 implementation + testing
Week 6: Phase 3 deployment
Week 7: Phase 4 preparation + testing
Week 8: Phase 4 production migration
Week 9: Monitoring + verification
```

---

**Status**: Ready to start Phase 1 detailed implementation  
**Next**: Implement ConstrainedResponseGenerator and begin Phase 1 wiring
