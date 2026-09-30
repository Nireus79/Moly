# MOLY REDESIGN: Quick Reference

## The Transformation

### BEFORE: Scattered Architecture (Current)
```
User Message
    ↓
[Layer 1: Extract]
    ↓
[Layer 2: Evaluate] (scattered)
[Layer 3: Evaluate] (scattered)
    ↓
[Layer 4: Ask questions] (scattered across 5 files)
[Layer 5: Ask questions] (scattered across 5 files)
[Layer 6: Ask questions] (scattered across 5 files)
    ↓
[Layer 7: Generate response]
    ↓
[Layer 8-11: Validate?] (No, response not validated)
    ↓
Send to user ← ROLE REVERSAL BUG POSSIBLE HERE
```

**Problems**:
- Extraction re-parsed 7 times (subject attribution corruption)
- Clarification logic scattered (1500 LOC across 5 files)
- Response not validated against facts
- 11 layers feel separate, hard to modify

---

### AFTER: Unified Pipeline (Redesigned)
```
User Message
    ↓
[EXTRACTION: Locked]
├─ Extract entities with LLM
├─ Add principle metadata (antonyms, relationships)
├─ Lock immediately (NO RE-PARSING)
└─ Store in immutable artifact
    ↓
[VALIDATION: Unified]
├─ Check context maturity
├─ Detect gaps, contradictions, ambiguities
├─ Check principle violations
└─ Route to ClarificationOrchestrator
    ↓
[CLARIFICATION ORCHESTRATOR: Centralized]
├─ Single method: GetNextQuestion()
├─ Priority-based: Self-harm → Principles → Contradictions → Gaps
├─ Deduplication: Don't ask same question twice
├─ Tracking: Every question logs "why we asked"
└─ Uses LLM-generated questions (no hardcoding)
    ↓
IF questions needed → Ask & loop back
ELSE → Continue
    ↓
[RESPONSE: Constrained Generation]
├─ Build constraints from facts: "User is dominant, don't suggest submission"
├─ Generate WITH constraints: LLM sees them in system prompt
├─ Validate BEFORE sending: Does response violate constraints?
├─ If violated → Ask clarification instead
└─ If valid → Send response
    ↓
Send to user (GUARANTEED CONSISTENT)
    ↓
[LEARNING: Automated]
└─ Save what user learned, what changed
```

**Benefits**:
- No multi-pass parsing (one lock)
- Clarification logic centralized (300 LOC in one orchestrator)
- Response validated BEFORE sending (no role reversal)
- Every question reason logged (transparent reasoning)

---

## Changes by Category

### What MUST Change (High Effort)
| Component | Current | Redesigned | Effort |
|-----------|---------|-----------|--------|
| Extraction | Multi-pass | Single-pass locked | 2 weeks |
| Clarification | 5 files, scattered | 1 orchestrator | 3 weeks |
| Response Gen | Generate → Validate | Constrained generation | 2 weeks |
| Database | Current schema | Enhanced schema | 2 weeks |

**Total**: ~9 weeks for one developer

### What CAN Stay (High Reuse)
| Component | Reuse |
|-----------|-------|
| LLM client | 100% (just add principle prompt) |
| Database infrastructure | 100% (just extend schema) |
| Auth/Sessions | 100% (no changes) |
| Extension UI | 100% (no changes) |
| Learning system | 100% (no changes) |

**Reuse**: ~70% of existing infrastructure

---

## Risk Matrix

```
           EFFORT    |
           High ↑    |
                |  Phase 2 (Orchestrator)
                |  ■ HIGH COMPLEXITY
                |  ■ 11-layer → 1-layer
                |  ■ MITIGATE: Feature flag, gradual rollout
                |
                |  Phase 4 (Database)    Phase 3 (Response)
                |  ■ HIGH DATA RISK      ■ MEDIUM LLM RISK
                |  ■ Schema migration    ■ Constraint handling
                |  ■ MITIGATE: Dual-write ■ MITIGATE: Fallback
                |
           Low  |  Phase 1 (Extraction)
                |  ■ LOW RISK
                |  ■ Mechanical change
                |  ■ MITIGATE: Testing
                |_________________
                         →
                    IMPACT (Benefit)
```

---

## Go/No-Go Gates

### Phase 1 (Weeks 1-3): Extraction Lock
```
MEASURE:
  ✓ Zero multi-pass parsing
  ✓ Accuracy maintained
  ✓ Lock mechanism solid

GO IF: All three ✓
NO-GO IF: Accuracy drops OR lock fails
EFFORT TO ROLLBACK: 1 day
```

### Phase 2 (Weeks 4-6): Orchestrator
```
MEASURE:
  ✓ All question types work
  ✓ No new edge cases
  ✓ 100% rollout stable

GO IF: All three ✓
NO-GO IF: Missing question types OR crashes
EFFORT TO ROLLBACK: < 1 hour (feature flag)
```

### Phase 3 (Weeks 7-9): Response Validation
```
MEASURE:
  ✓ Constraint violation rate < 5%
  ✓ Latency increase < 10%
  ✓ User satisfaction up

GO IF: All three ✓
NO-GO IF: Too many false positives OR slow
EFFORT TO ROLLBACK: < 1 hour (feature flag)
```

### Phase 4 (Weeks 10-12): Database
```
MEASURE:
  ✓ Zero data corruption
  ✓ Query performance same
  ✓ New schema stable

GO IF: All three ✓
NO-GO IF: Data corruption OR slow queries
EFFORT TO ROLLBACK: 2 hours (migration + verify)
```

---

## What Gets Better

### Bug: Role Reversal (Critical)
```
BEFORE:
  User: "I'm dominant"
  Extraction: dominant ✓
  Response: "explore your submissive nature" ✗ (BUG!)

AFTER:
  User: "I'm dominant"
  Extraction: dominant ✓ (LOCKED, no re-parsing)
  Validation: Response contradicts "dominant" → ASK CLARIFICATION
  Response: Never sent (caught at validation phase)
```

### Bug: Repeated Questions (Annoying)
```
BEFORE:
  Q: "Tell me about this girl"
  User: "She's intelligent"
  Q: "Tell me about this girl" ← SAME QUESTION AGAIN

AFTER:
  Q: "Tell me about this girl"
  User: "She's intelligent"
  Q: (different question, deduplication working)
```

### Issue: Mysterious Questions (UX)
```
BEFORE:
  Q: "How do you handle conflict?"
  User: *confused* "Why are you asking this?"

AFTER:
  Q: "How do you handle conflict?" [principle: User Autonomy]
  Reason logged: "Gap in communication style understanding"
  User: "Oh, that makes sense"
```

### Issue: Slow Clarification Loop (Performance)
```
BEFORE:
  11 layers check → maybe ask question → generate response
  Average: 2-3 seconds per message

AFTER:
  1 orchestrator prioritizes → ask best question → respond
  Average: 1-2 seconds (parallel logic)
```

---

## The Math

| Metric | Current | Redesigned | Improvement |
|--------|---------|-----------|------------|
| Lines of clarification logic | 1500 | 300 | 80% reduction |
| Parsing passes | 7 | 1 | 85% reduction |
| Response validation gaps | 1 major | 0 | 100% fixed |
| Question repetition rate | 5-10% | < 1% | 90% reduction |
| Latency | 2-3s p95 | 1-2s p95 | 30% faster |
| Code understandability | Hard (11 scattered) | Easy (1 orchestrator) | 3x easier |

---

## Decision

**RECOMMEND**: Start with Phase 1 (low risk, high value)

- Fixes multi-pass parsing (root cause of bugs)
- Takes only 3 weeks
- Low rollback risk (1 day)
- Can run rest of system unchanged
- Validates extraction-locking approach before committing to full redesign

**If Phase 1 succeeds**: Phase 2 unlocks the real value (orchestrator consolidation)

**If Phase 1 fails**: Only 1 week lost, system remains functional

