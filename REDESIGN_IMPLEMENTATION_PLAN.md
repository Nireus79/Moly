# MOLY REDESIGN: Implementation Plan & Risk Analysis

**Date**: Sept 30, 2026  
**Scope**: Refactor from 11-layer scattered architecture to unified extraction→validation→response pipeline  
**Effort Estimate**: 8-12 weeks (phased)  
**Risk Level**: MEDIUM (high complexity, but low risk with proper phasing)

---

## PART 1: WHAT NEEDS TO CHANGE

### Phase 1A: Extraction Lock & Immutability (WEEKS 1-2)

**What to build**:
```go
// NEW: Immutable extraction artifact with principle metadata
type LockedExtraction struct {
    ID              string
    CreatedAt       time.Time
    IsImmutable     bool  // Once set, never changes
    ExtractedEntities []ExtractedEntity
    ValidationErrors  []string
    
    // Lock & seal
    func (le *LockedExtraction) Lock() error {
        if le.IsImmutable {
            return errors.New("already locked")
        }
        le.IsImmutable = true
        le.CreatedAt = time.Now()
        return nil
    }
}

// ENHANCED: Extraction entity with principle awareness
type ExtractedEntity struct {
    // Existing fields kept:
    Type        string
    Value       string
    Subject     string
    Confidence  float64
    Evidence    string
    
    // NEW: Principle metadata (computed at extraction time)
    Principles      []string   // ["Harm Prevention", "User Autonomy"]
    Antonyms        []string   // ["submissive", "passive"] - computed by LLM
    AntonympSource  string     // "llm_reasoning" (not hardcoded)
    Bidirectional   bool       // Does "dominant" contradict "submissive" both ways?
}
```

**What to change**:
- `intent_detector.go`: Add principle extraction to LLM prompt
- `models/agent_types.go`: Add Principles, Antonyms to ExtractedEntity
- `database/migrations/`: New migration for extraction_metadata table

**What CAN reuse**:
- ✅ Existing LLM extraction logic (just add principle prompt)
- ✅ Existing ExtractionArtifact structure (extend it)
- ✅ Existing confidence scoring (keep as-is)

**Files touched**: 2 agent files, 1 model file, 1 migration

---

### Phase 1B: Lock Enforcement (WEEKS 2-3)

**What to build**:
```go
// NEW: Once extraction is locked, no re-parsing allowed
type ExtractionRepository struct {
    func (er *ExtractionRepository) GetLocked(extractionID string) (*LockedExtraction, error) {
        // Returns locked extraction
        // Caller CANNOT modify it
        return &LockedExtraction{IsImmutable: true}, nil
    }
    
    func (er *ExtractionRepository) TryRelock(extractionID string) error {
        return errors.New("extraction already locked, cannot modify")
    }
}

// Refactor all downstream code to work with locked extraction
// OLD: extraction.Entities = reparseWithLinguisticParser()
// NEW: extraction.Entities = lockedExtraction.ExtractedEntities (immutable)
```

**What to change**:
- `clarification_capture.go`: Remove multi-pass parsing, use locked extraction only
- `contact_deduplicator.go`: Use locked extraction subject attribution
- `main.go` (lines 2300-2600): Remove re-parsing logic

**What CAN reuse**:
- ✅ Existing database infrastructure
- ✅ Existing extraction logic
- ✅ Existing confidence scoring

**Files touched**: 5-7 files, refactor extraction usage

**RISK**: 
- ⚠️ **MEDIUM**: Many files call extraction logic independently. Must ensure all callsites use locked version.
- ⚠️ **Mitigation**: Introduce `GetLockedExtraction()` helper, make old methods error out with clear messages

---

### Phase 2: Unified Clarification Orchestrator (WEEKS 4-6)

**What to build**:
```go
// NEW: Single orchestrator for all clarification logic
type ClarificationOrchestrator struct {
    facts            FactSet
    contradictions   []Contradiction
    gaps             []ContextGap
    ambiguities      []Ambiguity
    principleIssues  []PrincipleIssue
    
    // Single method to get next question
    func (co *ClarificationOrchestrator) GetNextQuestion(ctx context.Context) (*Question, error) {
        // Priority order (principle-based):
        // 1. Self-harm (tier 1 blocks)
        // 2. Principle violations (tier 2)
        // 3. Contradictions (user consistency)
        // 4. Gaps (context completeness)
        // 5. Ambiguities (request clarity)
        // 6. Socratic (deepening, maturity >= 0.7)
        
        return co.selectByPriority()
    }
    
    // Every question remembers why it was asked
    func (co *ClarificationOrchestrator) CreateQuestion(reason string, principle string) Question {
        return Question{
            Text:            generateByLLM(reason),  // LLM, not hardcoded
            PrincipleBasis:  principle,              // Why we asked
            Type:            reason,                 // "gap" | "contradiction" | etc
            GeneratedAt:     time.Now(),
        }
    }
}

// NEW: Track question history to avoid re-asking
type QuestionHistory struct {
    func (qh *QuestionHistory) WasAskedRecently(q *Question) bool {
        // Check if similar question asked in last N messages
        // Use semantic similarity, not exact text match
    }
}
```

**What to change**:
- `agents/`: Consolidate clarification logic from 5+ files into ClarificationOrchestrator
- `database/clarification_repository.go`: Add principle_basis, question_reason fields
- `main.go`: Route all clarification through orchestrator (not scattered layer logic)

**What CAN reuse**:
- ✅ Existing question generation logic (consolidate it)
- ✅ Existing gap detection logic (feed into orchestrator)
- ✅ Existing contradiction detection (feed into orchestrator)
- ✅ Existing database storage (extend with principle_basis)

**Files touched**: 8-10 agent files, 2 database files, 1 migration

**RISK**:
- ⚠️ **HIGH**: This is the highest-risk change. 11 layers of question logic consolidated into 1 orchestrator.
- ⚠️ **Mitigation**: 
  - Build orchestrator alongside existing code first (parallel implementations)
  - Add feature flag: `useNewOrchestrator` (default false)
  - Test with 100% of test suite + 20 new integration tests
  - Gradual rollout: 10% users → 50% → 100%

---

### Phase 3: Constrained Response Generation (WEEKS 7-9)

**What to build**:
```go
// NEW: Response generator that validates before returning
type ConstrainedResponseGenerator struct {
    facts           FactSet
    extractedNow    []ExtractedEntity
    userMaturity    float64
    llmClient       LLMProvider
    
    // Generate response with constraints
    func (crg *ConstrainedResponseGenerator) Generate(ctx context.Context) (*Response, error) {
        // Step 1: Build constraints from facts
        constraints := crg.buildConstraints()
        // Output: "User is dominant, don't suggest submission"
        
        // Step 2: Generate WITH constraints in prompt
        resp, err := crg.llm.Call(ctx, &LLMRequest{
            SystemPrompt: crg.buildConstrainedPrompt(constraints),
            UserPrompt:   crg.buildUserPrompt(),
        })
        
        // Step 3: VALIDATE response against constraints
        violations := crg.validateResponse(resp, constraints)
        if violations > 0 {
            // Don't return bad response
            return nil, NewResponseContradictionError(violations)
        }
        
        // Step 4: Save learning
        crg.saveLearning(resp)
        return resp, nil
    }
    
    // Validation logic (NEW)
    func (crg *ConstrainedResponseGenerator) validateResponse(resp *Response, constraints []string) int {
        // Check: Does response imply characteristics contradicting facts?
        // Check: Does response suggest actions conflicting with user values?
        // Check: Does response respect contact profiles?
        return violationCount
    }
}
```

**What to change**:
- `tools/response_generator.go`: Add constraint building & validation
- `agents/conversation_agent.go`: Use ConstrainedResponseGenerator
- `tools/context_aware_conflict_handler.go`: Integrate validation into generation (not post-hoc)

**What CAN reuse**:
- ✅ Existing ResponseGenerator (wrap it with constraints)
- ✅ Existing LLM client & prompt building
- ✅ Existing fact loading & processing
- ✅ ValidateResponseAgainstCharacteristics() (integrate it here)

**Files touched**: 3 files, mostly additions

**RISK**:
- ⚠️ **MEDIUM**: LLM might reject constraints and fail to generate. Need fallback.
- ⚠️ **Mitigation**:
  - If constrained generation fails, retry with relaxed constraints
  - If still fails, return clarification question instead of generic fallback
  - Log every constraint violation for learning

---

### Phase 4: Unified Database Schema (WEEKS 10-12)

**What to build**:
```sql
-- NEW: Immutable extraction artifacts
CREATE TABLE extractions (
    id UUID PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    locked_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

-- NEW: Entities with principle metadata
CREATE TABLE extracted_entities (
    id UUID PRIMARY KEY,
    extraction_id UUID NOT NULL REFERENCES extractions(id),
    type TEXT NOT NULL,        -- User | Contact | Property
    value TEXT NOT NULL,
    subject TEXT,              -- "user" | contact_name
    confidence FLOAT,
    evidence TEXT,
    principles TEXT[],         -- Array of principle names
    antonyms TEXT[],          -- Pre-computed contradictions
    source TEXT,              -- LLM | UserCorrected | Clarification
    extracted_at TIMESTAMP NOT NULL,
    UNIQUE(extraction_id, value)  -- No duplicate entities
);

-- CHANGED: Questions now track why they were asked
ALTER TABLE clarification_questions ADD COLUMN (
    principle_basis TEXT,     -- "Harm Prevention", "User Autonomy"
    question_reason TEXT,     -- "gap", "contradiction", "ambiguity"
    orchestrator_id UUID,     -- Links to ClarificationOrchestrator run
    similar_asked_count INT   -- How many similar Qs already asked?
);

-- NEW: Track principle violations explicitly
CREATE TABLE principle_violations (
    id UUID PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    extracted_entity_id UUID REFERENCES extracted_entities(id),
    principle TEXT NOT NULL,
    violation_type TEXT,      -- "self_harm" | "autonomy" | "consent"
    severity TEXT,            -- "low" | "medium" | "high" | "critical"
    detected_at TIMESTAMP,
    resolved_at TIMESTAMP,
    resolution TEXT,          -- "clarification_asked" | "rephrased" | "blocked"
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

-- NEW: Response validation audit trail
CREATE TABLE response_validations (
    id UUID PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    response_text TEXT,
    constraints TEXT[],
    violations INT,
    passed BOOLEAN,
    fallback_used BOOLEAN,
    validated_at TIMESTAMP,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);
```

**What to change**:
- Create 3 new migrations (extractions, extracted_entities, principle_violations, response_validations)
- Backfill existing extraction data into new tables
- Update all queries to use new schema

**What CAN reuse**:
- ✅ Existing SQLite infrastructure
- ✅ Existing migration system
- ✅ Existing repositories (extend, don't replace)

**Files touched**: 4 migration files, 4 repository files

**RISK**:
- ⚠️ **HIGH**: Schema changes affect production data.
- ⚠️ **Mitigation**:
  - Add dual-write during transition (write to both old and new tables)
  - Backfill during low-traffic window
  - Add rollback migration (can revert if needed)
  - Test with copy of production database first

---

## PART 2: WHAT CAN BE REUSED

### Infrastructure (HIGH Reuse)
| Component | Current Status | Redesign Use | Effort |
|-----------|---|---|---|
| LLM Client | ✅ Working | Use as-is for extraction + generation | **MINIMAL** |
| Database Layer | ✅ SQLite + Migrations | Extend schema, keep queries | **LOW** |
| API Handlers | ✅ Implemented | Keep routing, refactor internals | **MEDIUM** |
| Extension | ✅ Working | No changes needed | **NONE** |
| Auth/Sessions | ✅ Working | Keep as-is | **NONE** |

### Code (MEDIUM Reuse)
| Component | Current | Redesign | Reuse |
|-----------|---------|----------|-------|
| extraction logic | `intent_detector.go` (1000 LOC) | Enhanced with principle extraction | **60%** |
| clarification logic | 5 files, 1500 LOC scattered | 1 orchestrator, 300 LOC | **40%** (reorganized) |
| validation | `response_validation_fix.go` (100 LOC) | Integrated into generation | **90%** |
| gap detection | `analysis_context_builder.go` (400 LOC) | Feed into orchestrator | **80%** |
| conflict detection | 3 files, 800 LOC | Feed into orchestrator | **70%** |
| learning system | `learning_agent.go` | Keep, no changes | **100%** |

### Database (HIGH Reuse)
| Table | Current | Redesign | Action |
|-------|---------|----------|--------|
| users | ✅ | Keep as-is | Reuse |
| sessions | ✅ | Keep as-is | Reuse |
| about_me | ✅ | Keep as-is | Reuse |
| contacts | ✅ | Keep as-is | Reuse |
| conversations | ✅ | Keep as-is | Reuse |
| messages | ✅ | Keep as-is | Reuse |
| clarification_questions | ✅ | Add principle_basis column | EXTEND |
| context_attributes | ✅ | Keep as-is | Reuse |
| contexts_conflicts | ✅ | Keep as-is | Reuse |

**Total reuse**: ~70% of existing infrastructure, ~40% of existing logic

---

## PART 3: RISKS & MITIGATION

### Risk 1: Multi-Pass Parsing Removal (CRITICAL)

**Risk**: Current system re-parses to ensure accuracy. Removing this might miss nuances.

**Why it's needed**: Multi-pass parsing causes subject attribution corruption (the role-reversal bug).

**Mitigation**:
- ✅ Use LLM confidence scores (trust 0.9+ confidence entities)
- ✅ For low-confidence entities, ask clarification (don't re-parse)
- ✅ Add explicit field: `needsClarification: bool` to extraction
- ✅ Test: Compare single-pass vs multi-pass accuracy on 100 sample messages
- ✅ Monitor: Track accuracy metrics post-deploy

**Implementation**: Phase 1B, add logging to measure accuracy impact

---

### Risk 2: Consolidated Clarification Logic (HIGH)

**Risk**: 11 layers of question logic compressed into 1 orchestrator. Edge cases might be missed.

**Failure mode**: 
- New user hits unexpected question ordering
- Question priority logic breaks for edge cases
- Some question types not triggered

**Mitigation**:
- ✅ Build orchestrator in parallel (don't delete old code initially)
- ✅ Add feature flag `useNewOrchestrator` (gradual rollout: 10% → 50% → 100%)
- ✅ Add 20 new integration tests (edge cases from current 11-layer system)
- ✅ Add monitoring: Track question types asked, order, user skip rates
- ✅ Failover: If orchestrator crashes, fall back to old layered system

**Implementation**: Phase 2, with A/B testing infrastructure

---

### Risk 3: Response Validation Blocking (MEDIUM)

**Risk**: New validation might block valid responses (false positives).

**Failure mode**: 
- Legitimate advice rejected due to over-strict constraint checking
- User gets stuck in clarification loop
- User gives up, thinks Moly is broken

**Mitigation**:
- ✅ Validation has 2 levels: HARD (block) vs WARN (ask user)
- ✅ HARD only for: self-harm, violence, consent violations
- ✅ WARN for: principle edge cases, user can override ("actually, this is fine")
- ✅ Monitor: Track override rate, false positive rate
- ✅ Human review: Sample check 10 blocked/warned responses weekly

**Implementation**: Phase 3, with override mechanism + monitoring

---

### Risk 4: Database Migration (HIGH)

**Risk**: Production data loss or corruption during schema changes.

**Failure mode**:
- Backfill job fails mid-way
- Old queries still run against partial schema
- Rollback corrupts existing data

**Mitigation**:
- ✅ Test migration on copy of production DB first (1 week)
- ✅ Add rollback migration (can revert all changes)
- ✅ Dual-write phase (write to both old + new tables for 1 week)
- ✅ Dual-read phase (read from both, compare results)
- ✅ Gradual cutover (20% → 50% → 100% users on new schema)
- ✅ Automated revert if data corruption detected

**Implementation**: Phase 4, with careful staging

---

### Risk 5: LLM Constraint Handling (MEDIUM)

**Risk**: LLM might not respect constraints in prompt, or fail to generate with them.

**Failure mode**:
- LLM ignores constraint "don't suggest submission"
- Constrained generation times out
- Response violates constraint anyway

**Mitigation**:
- ✅ Add constraint validation to LLM prompt (explicit: "You MUST not...")
- ✅ Add retry logic: If constrained fails, ask clarification instead
- ✅ Add confidence check: If response confidence < 0.8, ask clarification
- ✅ Monitor: Track constraint violation rate
- ✅ Test: Run 1000 constrained responses, measure violation rate

**Implementation**: Phase 3, with fallback logic

---

### Risk 6: Performance Degradation (MEDIUM)

**Risk**: New validation adds latency.

**Failure mode**:
- Constrained generation takes 5+ seconds
- User sees long delays
- Orchestrator priority logic is slow

**Mitigation**:
- ✅ Benchmark before/after: Track response latency
- ✅ Add caching: Principle metadata, constraint lists, LLM responses
- ✅ Async validation: Validate while generating next response
- ✅ SLA: Keep p95 latency < 3 seconds
- ✅ If degradation > 20%, rollback

**Implementation**: Phase 3-4, with monitoring

---

## PART 4: IMPLEMENTATION TIMELINE

### Phase 1: Foundation (Weeks 1-3) - **LOW RISK**
```
Week 1: Extraction lock + principle metadata
├─ Add Principles, Antonyms to ExtractedEntity
├─ Add principle extraction to LLM prompt
├─ Add lock mechanism
└─ Tests: 5 unit tests

Week 2-3: Lock enforcement
├─ Refactor clarification_capture.go
├─ Refactor contact_deduplicator.go
├─ Add "locked extraction" requirement throughout
└─ Tests: 10 integration tests + benchmarks
```

**Deliverable**: Extraction locked, no more multi-pass parsing. Can merge to main. No user-facing changes.

---

### Phase 2: Orchestration (Weeks 4-6) - **HIGH RISK, CAREFUL ROLLOUT**
```
Week 4: Build ClarificationOrchestrator in parallel
├─ New file: orchestration/clarification_orchestrator.go
├─ Feature flag: useNewOrchestrator (default false)
├─ Route clarifications through it (experimental)
└─ Tests: 20 tests covering all question types

Week 5: Question deduplication & principle tracking
├─ Add principle_basis to questions
├─ Implement question history check
├─ Add "why did we ask?" logging
└─ Tests: 10 tests for deduplication

Week 6: Gradual rollout
├─ 10% users on new orchestrator
├─ Monitor for 48 hours
├─ 50% users
├─ Monitor for 1 week
├─ 100% users
└─ Keep old layered system as fallback
```

**Deliverable**: Unified clarification orchestrator live. Questions now track why they were asked. Can rollback if needed.

---

### Phase 3: Response Validation (Weeks 7-9) - **MEDIUM RISK**
```
Week 7: Constraint building & validation logic
├─ New: ConstrainedResponseGenerator.buildConstraints()
├─ New: ConstrainedResponseGenerator.validateResponse()
├─ Integration with existing ResponseGenerator
└─ Tests: 15 tests for constraint validation

Week 8: Response generation with constraints
├─ Modify LLM prompt to include constraints
├─ Add fallback logic (if constrained fails, ask clarification)
├─ Add override mechanism (user can say "this is fine")
└─ Tests: 20 integration tests

Week 9: Monitoring & gradual rollout
├─ 10% users on constrained generation
├─ Monitor violation rate, latency, user satisfaction
├─ If good: 50% → 100%
└─ Keep old generation as fallback
```

**Deliverable**: Responses now validated against facts before sending. Role-reversal bug impossible.

---

### Phase 4: Database Schema (Weeks 10-12) - **HIGH RISK, CAREFUL STAGING**
```
Week 10: Schema design & test migration
├─ Create 3 migrations (extractions, entities, validations)
├─ Test on copy of production DB
├─ Create rollback migrations
├─ Add backfill logic
└─ Tests: Migration tests

Week 11: Dual-write & staging
├─ Deploy migrations (non-destructive)
├─ Start dual-write (both old + new tables)
├─ Verify data consistency
├─ Backfill historical data
└─ Tests: Data consistency checks

Week 12: Cutover & cleanup
├─ Switch to new schema (read from new)
├─ Monitor for 1 week
├─ Remove dual-write code
├─ Archive old tables (keep for 30 days)
└─ Tests: Full regression test suite
```

**Deliverable**: New schema live. All principle metadata, validation logs, question history tracked.

---

## PART 5: GO/NO-GO DECISION POINTS

### After Phase 1 (Week 3)
**Decision**: Proceed to Phase 2?
- ✅ **GO if**: Lock enforcement working, no multi-pass parsing bugs
- ❌ **NO-GO if**: Lock enforcement causes regressions, users report accuracy loss

### After Phase 2 (Week 6)
**Decision**: Replace old layered system entirely?
- ✅ **GO if**: Orchestrator handles all question types correctly, no new edge cases, rollout stable at 100%
- ❌ **NO-GO if**: Orchestrator misses question types, crashes on edge cases, users skip clarifications more

### After Phase 3 (Week 9)
**Decision**: Enable constrained generation for all users?
- ✅ **GO if**: Constraint violation rate < 5%, latency increase < 10%, user satisfaction up
- ❌ **NO-GO if**: Constraints cause timeouts, too many false positives, response quality degrades

### After Phase 4 (Week 12)
**Decision**: Keep new schema permanently?
- ✅ **GO if**: Data migration clean, no corruption, queries performant, monitoring shows improvement
- ❌ **NO-GO if**: Data corruption found, queries slow, operational overhead too high

---

## PART 6: EFFORT BREAKDOWN

### Development
| Phase | Files | LOC Added | LOC Modified | Estimated Days |
|-------|-------|-----------|--------------|---|
| 1A | 2 | 300 | 200 | 3 |
| 1B | 5 | 100 | 400 | 4 |
| 2 | 10 | 800 | 1000 | 10 |
| 3 | 3 | 500 | 300 | 6 |
| 4 | 4 | 0 | 200 (SQL) | 4 |
| **TOTAL** | **24** | **1700** | **2100** | **27 days (6 weeks)** |

### Testing
| Phase | Unit Tests | Integration Tests | Days |
|-------|---|---|---|
| 1 | 15 | 10 | 3 |
| 2 | 20 | 20 | 5 |
| 3 | 15 | 20 | 5 |
| 4 | 5 | 15 | 4 |
| **TOTAL** | **55** | **65** | **17 days** |

### Total Effort: **44 days** (~9 weeks, 1 person) or **6 weeks** (2 people parallel)

---

## PART 7: SUCCESS METRICS

### Phase 1
- ✅ Zero multi-pass parsing
- ✅ Extraction accuracy maintained or improved
- ✅ No data corruption from locking

### Phase 2
- ✅ All question types triggered correctly
- ✅ Question deduplication working (no repeated questions)
- ✅ Principle basis logged for 100% of questions
- ✅ User skip rate on clarifications < 20%

### Phase 3
- ✅ Role-reversal bug: ZERO occurrences
- ✅ Response constraint violation rate < 5%
- ✅ Response latency increase < 10%
- ✅ User override rate < 10% (false positives)

### Phase 4
- ✅ Migration: ZERO data corruption
- ✅ Query performance: p95 latency < 3s (same as before)
- ✅ Principle violation tracking: 100% logged
- ✅ Audit trail: All question reasons tracked

---

## PART 8: ROLLBACK STRATEGY

Each phase has a rollback plan:

**Phase 1**: Remove lock enforcement, fall back to current extraction
- Effort: 1 day
- Risk: Very low (just revert code)

**Phase 2**: Disable feature flag, route to old layered system
- Effort: < 1 hour (flip flag, failover)
- Risk: Low (old system proven)

**Phase 3**: Disable constrained generation, use old response generator
- Effort: < 1 hour (flip flag, failover)
- Risk: Low (old system proven, but role-reversal bug could return)

**Phase 4**: Run rollback migration, read from old schema
- Effort: 2 hours (migration + data verification)
- Risk: Medium (must verify data consistency)

---

## CONCLUSION

**Redesign is achievable with careful phasing and rollout.**

**Key risks** are all manageable with proper mitigation:
1. Multi-pass removal → mitigated by confidence scores + testing
2. Orchestrator consolidation → mitigated by feature flag + A/B testing
3. Response validation → mitigated by HARD/WARN levels + monitoring
4. Database migration → mitigated by dual-write + staged cutover
5. LLM constraint handling → mitigated by retry logic + fallback
6. Performance → mitigated by caching + SLA monitoring

**Sweet spot**: Start with Phase 1 (low risk, high value). Evaluate before Phase 2.

**Current system can run indefinitely**, but redesign eliminates:
- Multi-pass parsing (root cause of bugs)
- Scattered clarification logic (1500 LOC → 300 LOC)
- Post-hoc response validation (validate during generation)
- Opaque decision making (every question reason logged)

