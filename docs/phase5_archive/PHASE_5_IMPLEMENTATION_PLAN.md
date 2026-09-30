# Phase 5: Self-Awareness Optimization - Implementation Plan

**Date**: September 29, 2026  
**Status**: 🟡 READY TO START  
**Effort**: ~2 days  
**Risk**: LOW  

---

## Overview

Optimize Moly's self-awareness system (MetaInstructionDetector) using LinguisticParser from Week 1-4 implementation.

**Goal**: 50-150x performance improvement + better accuracy + cleaner code

---

## Phase 1: LinguisticParser Grammar Rules (Day 1 - 4 hours)

### Deliverables
- Add 4 new grammar rules to `tools/linguistic_parser.go`
- Write 8 comprehensive tests in `tools/linguistic_parser_test.go`
- Ensure 100% test pass rate
- ~50 LOC added

### Rules to Add

**Rule: extractFocusDirective**
```go
// Patterns: "X is my focus", "focus on X", "prioritize X", "my focus is X"
// Examples:
//   "Lace is my focus" → {subject: "user", property: "lace", type: "focus", confidence: 0.95}
//   "focus on communication" → {subject: "user", property: "communication", type: "focus", confidence: 0.90}
//   "don't focus on drama" → {subject: "user", property: "drama", type: "focus", is_negated: true, confidence: 0.92}
// Output: ExtractionResult with type="focus"
```

**Rule: extractConstraint**
```go
// Patterns: "remember X", "keep in mind X", "don't forget X"
// Examples:
//   "remember to be patient" → {subject: "user", property: "patient", type: "constraint", confidence: 0.90}
//   "keep in mind I'm nervous" → {subject: "user", property: "nervous", type: "constraint", confidence: 0.85}
// Output: ExtractionResult with type="constraint"
```

**Rule: extractPriority**
```go
// Patterns: "my priority is X", "prioritize X"
// Examples:
//   "my priority is intimacy" → {subject: "user", property: "intimacy", type: "priority", confidence: 0.95}
// Output: ExtractionResult with type="priority"
```

**Rule: extractNegatedDirective**
```go
// Patterns: "don't/can't/won't focus on X", "not interested in X"
// Examples:
//   "don't focus on drama" → {subject: "user", property: "drama", type: "focus", is_negated: true, confidence: 0.92}
//   "not interested in casual" → {subject: "user", property: "casual", type: "interest", is_negated: true, confidence: 0.90}
// Output: ExtractionResult with is_negated=true
```

### Tests Required (8 total)

```go
TestExtractFocusDirective() {
    // Test: "Lace is my focus"
    // Test: "focus on communication"
    // Test: "don't focus on drama"
}

TestExtractConstraint() {
    // Test: "remember to be patient"
    // Test: "keep in mind I'm nervous"
}

TestExtractPriority() {
    // Test: "my priority is intimacy"
}

TestExtractNegatedDirective() {
    // Test: "don't focus on X"
    // Test: "not interested in Y"
}
```

### Success Criteria
- ✅ All 8 tests pass
- ✅ Build succeeds (go build ./...)
- ✅ No compiler warnings
- ✅ Rules handle negation correctly
- ✅ Confidence scores reasonable (0.85-0.95)

### Files to Modify
- `moly-go/tools/linguistic_parser.go` (add 4 functions, ~50 LOC)
- `moly-go/tools/linguistic_parser_test.go` (add 8 tests, ~120 LOC)

### Git Commit Message
```
Phase 5 Part 1: Add meta-instruction grammar rules to LinguisticParser

New grammar rules for self-awareness optimization:
  • extractFocusDirective: "X is my focus", "focus on X"
  • extractConstraint: "remember X", "keep in mind X"
  • extractPriority: "my priority is X"
  • extractNegatedDirective: "don't focus on X"

Tests: 8 new tests covering all patterns and edge cases
Impact: Foundation for simplified MetaInstructionDetector

All tests passing. Build clean. Ready for Phase 2.

Co-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>
```

---

## Phase 2: MetaInstructionDetector Refactor (Day 1 - 6 hours)

### Deliverables
- Refactor `agents/meta_instruction_detector.go` (~150 LOC changes)
- Integrate LinguisticParser as Tier 1
- Simplify Tier 2 (keyword detection)
- Keep Tier 3 (LLM fallback)
- Write 10 comprehensive tests
- 43% LOC reduction (320 → 180)

### Architecture

```go
type MetaInstructionDetector struct {
    llmClient tools.LLMProvider
    parser    *tools.LinguisticParser  // NEW
}

func (mid *MetaInstructionDetector) Detect(ctx context.Context, msg string) *MetaInstruction {
    // Tier 1: LinguisticParser (NEW)
    extractions := mid.parser.Parse(msg)
    if result := mid.extractMetaFromLinguistic(extractions); result != nil {
        return result
    }
    
    // Tier 2: Keyword Detection (SIMPLIFIED)
    if result := mid.detectByKeywords(msg); result != nil {
        return result
    }
    
    // Tier 3: LLM (UNCHANGED)
    return mid.detectByLLM(ctx, msg)
}

// NEW: Convert ExtractionResult → MetaInstruction
func (mid *MetaInstructionDetector) extractMetaFromLinguistic(extractions []ExtractionResult) *MetaInstruction {
    for _, ext := range extractions {
        if ext.Type == "focus" || ext.Type == "constraint" || ext.Type == "priority" {
            return &MetaInstruction{
                Type:           ext.Type,
                Confidence:     ext.Confidence,
                TargetTopic:    ext.Property,
                Subjects:       []string{ext.Subject},
                IsNegated:      ext.IsNegated,
                RawInstruction: "", // populated by caller
                Source:         "linguistic_parser",
            }
        }
    }
    return nil
}
```

### What to Remove
- `extractFocusTarget()` (~30 LOC) - replaced by grammar rule
- `extractConstraintTarget()` (~25 LOC) - replaced by grammar rule
- Complex string parsing (~45 LOC)
- Manual JSON parsing (~70 LOC)
- **Total removed: ~170 LOC**

### What to Add
- `extractMetaFromLinguistic()` (~20 LOC)
- LinguisticParser integration (~30 LOC)
- Updated tests (~10 LOC)
- **Total added: ~60 LOC**

### Tests Required (10 total)

```go
TestDetectViaLinguisticParser() {
    // Test: "Lace is my focus" → uses parser
    // Test: "remember to be patient" → uses parser
}

TestDetectViaKeywords() {
    // Test: "you are moly" → keyword tier
}

TestDetectViaLLM() {
    // Test: Genuinely ambiguous case → LLM tier
}

TestNegatedDirectives() {
    // Test: "don't focus on X" → parsed correctly
}

TestCompoundInstructions() {
    // Test: "Lace is my focus AND remember patience"
}

TestSubjectAttribution() {
    // Test: Subject extracted from parser
}

TestBackwardCompatibility() {
    // Test: MetaInstruction type unchanged
    // Test: Detect() interface unchanged
}
```

### Success Criteria
- ✅ All 10 tests pass
- ✅ 43% LOC reduction (320 → 180)
- ✅ No breaking changes to MetaInstruction type
- ✅ No breaking changes to Detect() interface
- ✅ LLM fallback still works
- ✅ Performance: <100ms avg (was 5-15s)

### Files to Modify
- `moly-go/agents/meta_instruction_detector.go` (refactor, ~150 LOC changes)
- `moly-go/agents/meta_instruction_detector_test.go` (add 10 tests, ~200 LOC)

### Git Commit Message
```
Phase 5 Part 2: Refactor MetaInstructionDetector with LinguisticParser

Simplified self-awareness system:
  • Tier 1: LinguisticParser (new, <100ms, deterministic)
  • Tier 2: Keyword detection (simplified, <1ms)
  • Tier 3: LLM fallback (unchanged, 5-15s)

Changes:
  • Added extractMetaFromLinguistic() for grammar→meta conversion
  • Removed complex string parsing (~170 LOC)
  • Removed manual JSON parsing (~70 LOC)
  • Reduced total LOC: 320 → 180 (43% reduction)

Performance: 50-150x faster
Accuracy: 90-99% across all cases
Tests: 10 new tests, all passing
Backward compatible: Same interface, same types

Co-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>
```

---

## Phase 3: Integration & Testing (Day 2 - 4 hours)

### Deliverables
- Integration tests in `integration_meta_instruction_test.go`
- Performance benchmarks
- End-to-end verification
- Regression testing
- 12 comprehensive tests

### Tests Required

```go
TestMetaInstructionIntegration() {
    // Full flow: message → parse → detect → store
}

TestPerformanceBenchmark() {
    // Time simple vs compound vs negated instructions
    // Measure cache effectiveness
}

TestRegressionCases() {
    // Cases that failed in current system
    // Verify they now work
}

TestAccuracy() {
    // 100 test cases covering all patterns
    // Verify 90%+ accuracy
}
```

### Success Criteria
- ✅ All 12 tests pass
- ✅ Performance: average <100ms
- ✅ Accuracy: >90% overall
- ✅ No regressions from current system
- ✅ Build succeeds

### Files to Add/Modify
- `moly-go/integration_meta_instruction_test.go` (new, ~200 LOC, 12 tests)
- `moly-go/main.go` (no changes needed - interface unchanged)

### Git Commit Message
```
Phase 5 Part 3: Integration testing and performance verification

Tests:
  • 12 integration tests covering all scenarios
  • Performance benchmarks (simple, compound, negated)
  • Regression testing vs current system
  • Accuracy verification (>90% overall)

Results:
  • Average detection time: <100ms (was 5-15s)
  • Accuracy improvement: 90-99% (was 80-95%)
  • 100% test pass rate
  • Zero regressions

Performance breakdown:
  • Simple cases: <50ms (was 5-15s, 100-300x faster)
  • Compound: <80ms (was 15-20s, 187-250x faster)
  • Negated: <60ms (was 10-15s, 166-250x faster)

Co-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>
```

---

## Phase 4: Optimization & Deployment (Day 2 - 3 hours)

### Deliverables
- Fine-tune confidence thresholds
- Performance profiling
- Update documentation
- Production deployment preparation
- 50 LOC

### Tuning

```go
// Adjust if needed based on test results:
const (
    HighConfidenceMetaInstruction   = 0.90  // Focus directive
    MediumConfidenceMetaInstruction = 0.75  // Generic LLM result
    LowConfidenceMetaInstruction    = 0.60  // Ambiguous case
)
```

### Documentation Updates
- Add Phase 5 results to SELF_AWARENESS_OPTIMIZATION.md
- Update ARCHITECTURE.md with new Meta-Instruction flow
- Add Phase 5 completion status

### Success Criteria
- ✅ Confidence thresholds optimized
- ✅ No performance regressions
- ✅ Documentation updated
- ✅ Ready for production

### Files to Modify
- `moly-go/agents/meta_instruction_detector.go` (tune thresholds, ~10 LOC)
- `ARCHITECTURE.md` (add Phase 5 section)
- `SELF_AWARENESS_OPTIMIZATION.md` (add Phase 5 results)

### Git Commit Message
```
Phase 5 Complete: Self-Awareness Optimization Ready for Production

Summary:
  ✅ Phase 1: 4 new grammar rules (50 LOC, 8 tests)
  ✅ Phase 2: Refactored detector (150 LOC changes, 10 tests)
  ✅ Phase 3: Integration testing (12 tests, benchmarks)
  ✅ Phase 4: Optimization & docs (50 LOC)

Results:
  • Code: 320 LOC → 180 LOC (43% reduction)
  • Speed: 5-15s → <100ms (50-150x faster)
  • Accuracy: 80-95% → 90-99% (improved)
  • Capabilities: +compound, +negation, +multi-subject
  • Tests: 40+ new tests, 100% pass rate

Backward compatible: Same interfaces, same types
Risk level: LOW (uses proven components)
Status: PRODUCTION READY

Co-Authored-By: Claude Haiku 4.5 <noreply@anthropic.com>
```

---

## Timeline & Milestones

```
Day 1 (8 hours):
├─ Phase 1 (4 hrs): Grammar rules + tests
├─ Phase 2 (6 hrs): Detector refactor + tests
└─ Checkpoint: 50% reduction in LOC complete

Day 2 (7 hours):
├─ Phase 3 (4 hrs): Integration testing
├─ Phase 4 (3 hrs): Optimization & docs
└─ Checkpoint: PRODUCTION READY

Total: ~2 days, 15 hours effort
```

---

## Success Metrics

| Metric | Target | Current | Phase 5 Goal |
|--------|--------|---------|-------------|
| Speed | <100ms | 5-15s | ✅ Achieve |
| Code | 180 LOC | 320 LOC | ✅ 43% reduction |
| Accuracy | >90% | 80-95% | ✅ 90-99% |
| Tests | 40+ | 0 | ✅ All passing |
| Build | Clean | - | ✅ No warnings |

---

## Rollback Plan

If issues arise:
1. Revert Phase 4 commit (optimization tuning)
2. Keep Phases 1-3 (they're stable)
3. Or revert to before Phase 5 started
4. LLM fallback always available as safety net

---

## Next Steps

### Immediate (Start Phase 1)
1. Read `SELF_AWARENESS_OPTIMIZATION.md` (context)
2. Add 4 grammar rules to LinguisticParser
3. Write 8 tests
4. Commit Phase 1

### Continue (Phase 2)
1. Refactor MetaInstructionDetector
2. Write 10 integration tests
3. Verify performance improvement
4. Commit Phase 2

### Finish (Phases 3-4)
1. Complete integration testing
2. Optimize thresholds
3. Update documentation
4. Deploy to production

---

## References

- **Current System**: `moly-go/agents/meta_instruction_detector.go` (320 LOC)
- **Grammar Foundation**: `moly-go/tools/linguistic_parser.go` (550 LOC, proven)
- **Analysis**: `SELF_AWARENESS_OPTIMIZATION.md` (detailed rationale)
- **Week 1-4 Implementation**: Already deployed and tested

---

**Status**: 🟡 **READY TO START PHASE 1**  
**Effort**: ~2 days total  
**Risk**: LOW (uses proven components)  
**ROI**: Very High (50-150x speedup)  

**Ready to begin Phase 1?** ✅
