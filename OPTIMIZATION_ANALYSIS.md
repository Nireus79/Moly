# MOLY ARCHITECTURE - OPTIMIZATION ANALYSIS

**Date**: Sept 24, 2026  
**Focus**: Latency reduction & token cost optimization  
**Current estimate**: ~7-9 LLM calls per message

---

## LLM CALL MAP - Current Flow

### Phase 1: Main.go (Message Processing)
```
Request received
    ↓
[1] ConstitutionalEvaluator.Evaluate(userMessage)
    - Tier 1a/1b: deterministic (no LLM)
    - Tier 2: LLM if context mature (>= 0.5)
    ↓
[2] ContextExtractor.Extract(userMessage)
    - LLM call: Always
    - Extracts: contact, style, intention, values
    ↓
→ Pass to ConversationAgent with precomputed safety verdict
```

**Main.go LLM calls**: 1-2 (ConstitutionalEvaluator conditional on maturity)

---

### Phase 2: ConversationAgent.Run() (Response Generation)
```
Load structured context
    ↓
[3] MessageClarityAnalyzer.Analyze(userMessage, history)
    - LLM call: Always
    - Checks: clarity, priority, required clarifications
    ↓
[4] IntentDetector.Detect*(userMessage, ...)
    - LLM call: Always (unless fallback)
    - Detects: benign, harmful, unclear
    ↓
[5] SubjectShiftDetector.DetectShifts(userMessage, previousSubject)
    - LLM call: Always (unless fallback)
    - Detects: topic/contact changes
    ↓
[6] ResponseGenerator.Generate*(ctx, userMessage)
    - LLM call: Always
    - Generates: clarification or response
    ↓
[7] ConstitutionalEvaluator.Evaluate(generatedResponse)
    - LLM call: Always
    - Checks: response safety before sending
    ↓
→ Return response
```

**ConversationAgent LLM calls**: 5-6 per run

---

## TOTAL: 6-8 LLM CALLS PER MESSAGE

| Phase | Component | LLM Calls | Tokens Est. | Opportunity |
|-------|-----------|-----------|-------------|------------|
| Main | ConstitutionalEvaluator | 0-1 | 200-500 | Conditional on maturity ✅ |
| Main | ContextExtractor | 1 | 1000-1500 | **Parallelize with safety?** |
| Agent | MessageClarityAnalyzer | 1 | 500-800 | **Parallelize with intent?** |
| Agent | IntentDetector | 1 | 200-400 | **Early exit if unclear?** |
| Agent | SubjectShiftDetector | 1 | 200-400 | **Skip on first msg?** |
| Agent | ResponseGenerator | 1 | 800-1200 | **Depend on clarity check** |
| Agent | ConstitutionalEvaluator | 1 | 200-500 | **Batch with input check?** |
| **TOTAL** | | **6-8** | **4000-6400** | |

---

## OPTIMIZATION OPPORTUNITIES

### 🟢 QUICK WINS (Low effort, high impact)

#### 1. **Skip SubjectShiftDetector on First Message**
**Impact**: -1 LLM call (12-15% reduction)  
**Why**: First message has no previous subject to compare against  
**Effort**: 2 lines  
**Token savings**: ~200-400 tokens

```go
// In conversation_agent.go, Layer 9 check:
if ca.subjectShiftDetector != nil && structuredCtx != nil && !isFirstMessage {
    shifts := ca.subjectShiftDetector.DetectShifts(userMessage, previousSubject)
    // ...
}
```

**Status**: Ready to implement ✅

---

#### 2. **Early Exit on Unambiguous Clarity**
**Impact**: -1 LLM call in common cases (10-15% reduction)  
**Why**: If message is clear + no clarifications needed, skip intent detection for simple acknowledgments  
**Effort**: 3-5 lines  
**Token savings**: ~200-400 tokens

```go
// Current: Always call IntentDetector
// Better: If clarity.CanProceed && len(clarity.RequiredClarifications) == 0
//         → Go directly to AckOnly workflow, skip IntentDetector
if clarity.CanProceed && len(clarity.RequiredClarifications) == 0 {
    // Message is clear, no questions needed
    workflow = WorkflowAckOnly
    // Skip IntentDetector, go straight to response generation
} else {
    intentAnalysis = ca.intentDetector.Detect*() // Still call if needed
}
```

**Status**: Ready to implement ✅

---

#### 3. **Tier 1b Optimization - Combine Signal Scan with Clarity Check**
**Impact**: Small but consistent (5-10% on clarity-checking tokens)  
**Why**: Both look at keywords - could be combined  
**Effort**: Medium (refactor MessageClarityAnalyzer)  
**Token savings**: ~100-150 tokens

```go
// Instead of:
// [1] ConstitutionalEvaluator scans principles
// [3] MessageClarityAnalyzer scans keywords

// Could: 
// Combined check that looks for both safety signals AND clarity signals in one pass
```

**Status**: Needs design review

---

### 🟡 MEDIUM EFFORT, HIGH IMPACT

#### 4. **Parallelize Main.go Phase (Context + Safety)**
**Impact**: -1 sequential call (20-30% latency reduction)  
**Why**: ContextExtractor and ConstitutionalEvaluator are independent  
**Effort**: Medium (goroutines, sync)  
**Token cost**: Same (~1000-1500 + 200-500)

```go
// Current (sequential):
// verdict := ConstitutionalEvaluator.Evaluate(msg)    // ~0.5s
// extractedContext := ContextExtractor.Extract(msg)    // ~1s
// Total: ~1.5s

// Better (parallel):
// go func() { verdict = ConstitutionalEvaluator.Evaluate(msg) }()
// extractedContext = ContextExtractor.Extract(msg)
// wait for goroutine
// Total: ~1s (saves ~0.5s)
```

**Status**: Implementation candidate

**Latency impact**: 
- Sequential: ~1.5s
- Parallel: ~1s
- Savings: 33% reduction on input phase

---

#### 5. **Parallelize Agent Phase (Clarity + Intent)**
**Impact**: -1 sequential call (15-25% latency reduction on agent phase)  
**Why**: MessageClarityAnalyzer and IntentDetector are mostly independent  
**Effort**: Medium  
**Token cost**: Same (~500-800 + 200-400)

```go
// Current (sequential):
// clarity := clarityAnalyzer.Analyze()       // ~0.6s
// intent := intentDetector.Detect()          // ~0.4s
// Total: ~1.0s

// Better (parallel):
// go func() { clarity = clarityAnalyzer.Analyze() }()
// intent = intentDetector.Detect()
// wait
// Total: ~0.6s
// Savings: ~0.4s
```

**Status**: Implementation candidate

**Latency impact**:
- Sequential: ~1.0s
- Parallel: ~0.6s
- Savings: 40% reduction on clarity/intent phase

---

#### 6. **Conditional Response Generation Based on Clarity**
**Impact**: -0 LLM calls in some paths but better response quality  
**Why**: If we're asking clarification, generate different response optimized for clarification  
**Effort**: Low-medium  
**Token savings**: Potentially -200 tokens (shorter clarifications than full responses)

```go
// Current: Always generate via ResponseGenerator
// Better: 
// - If needs clarity → Use optimized ClarificationGenerator (lighter)
// - If clear → Use full ResponseGenerator

// ClarificationGenerator: "Tell me more about X" (50 tokens)
// ResponseGenerator: Full conversation response (800 tokens)
// Savings: ~750 tokens when asking clarification
```

**Status**: Partially implemented (gap clarifications are already optimized)

---

### 🔴 ARCHITECTURAL CHANGES (High impact, design trade-offs)

#### 7. **Cache Context Extraction for User (1-hour TTL)**
**Impact**: -1 LLM call for 50% of follow-up messages  
**Why**: User communication style doesn't change every message  
**Trade-off**: Might miss newly mentioned contacts in some cases  
**Effort**: Medium (cache layer + invalidation logic)

```go
// Cache key: userID + "context_extraction"
// TTL: 1 hour (or on specific keywords like "actually", "changed my mind")
// Hit rate: ~50% of messages in same conversation
// Token savings: ~1000-1500 tokens per cache hit
```

**Status**: Design needed (need to handle "changed my mind" scenarios)

---

#### 8. **Skip SubjectShiftDetector Entirely, Use Simpler Detection**
**Impact**: -1 LLM call (12-15% reduction)  
**Why**: Could use heuristic pattern matching for 80% of cases  
**Trade-off**: Less sophisticated shift detection  
**Effort**: Low  
**Token savings**: ~200-400 tokens

```go
// Current: LLM-based detection
// Alternative: 
// - If message mentions different person name → shift detected
// - If pronouns change (he→she, etc.) → potential shift
// - Pattern: "Also about X" or "By the way" → shift detected
// 
// LLM fallback only if heuristics uncertain
```

**Status**: Implementation candidate (low risk)

---

#### 9. **Combine ConstitutionalEvaluator Calls (Input + Output)**
**Impact**: Not reducing calls, but batching/optimization  
**Why**: Same principles, same constitution data  
**Trade-off**: Would need bidirectional evaluation prompt  
**Effort**: High  
**Token savings**: ~0 tokens (same analysis) but could optimize prompt

```go
// Current:
// [1] Evaluate(userMessage)
// [7] Evaluate(generatedResponse)

// Could combine into single analysis pass with two evaluations
// But: Adds complexity, may not be worth it
```

**Status**: Lower priority (diminishing returns)

---

## RECOMMENDATION RANKING

### Priority 1 (Implement Now)
1. ✅ **Skip SubjectShiftDetector on first message** (1 LLM call, 2 lines)
2. ✅ **Early exit on unambiguous clarity** (1 LLM call, 5 lines)
3. ✅ **Parallelize Main.go phase** (0 calls, 20-30% latency reduction)
4. ✅ **Parallelize Agent phase** (0 calls, 40% latency reduction on that phase)

**Expected impact**: 
- Token cost: -200-800 tokens (10-15% reduction)
- Latency: -0.9s (40% reduction on processing)

---

### Priority 2 (Implement Next Sprint)
1. 🟡 **Simpler subject shift detection** (1 LLM call, pattern-based)
2. 🟡 **Context cache with TTL** (1000+ tokens savings, requires invalidation logic)
3. 🟡 **Combined Tier 1b scan** (100-150 tokens, refactoring)

**Expected impact**:
- Token cost: -1200-1500 tokens (20-25% reduction)
- Latency: Minimal additional savings

---

### Priority 3 (Monitor, Not Required)
1. 🔴 Batch ConstitutionalEvaluator calls (complex, minimal gain)
2. 🔴 Major architectural changes (risk vs. reward)

---

## IMPLEMENTATION ROADMAP

### Phase 1: Quick Wins (1-2 hours)
```
commit 1: Skip SubjectShiftDetector on first message
commit 2: Early exit on clear messages (skip intent detection)
commit 3: Parallelize Main.go (ContextExtractor + ConstitutionalEvaluator)
commit 4: Parallelize ConversationAgent (Clarity + Intent)
```

**Expected result**: 
- 10-15% token cost reduction
- 40% latency reduction on processing

### Phase 2: Pattern-Based Optimization (4-6 hours)
```
commit 5: Replace LLM SubjectShiftDetector with pattern matching
commit 6: Add context cache with smart invalidation
commit 7: Optimize tier 1b scanning
```

**Expected result**:
- Additional 15-20% token cost reduction
- Improved system responsiveness

---

## CURRENT STATE ASSESSMENT

**Strengths**:
- ✅ Conditional safety evaluation (Layer 3 maturity check saves LLM calls)
- ✅ Pre-computed verdicts reduce double-checking
- ✅ Deterministic Tier 1a/1b reduces LLM dependency

**Inefficiencies**:
- ⚠️ SubjectShiftDetector runs every message (skip first)
- ⚠️ IntentDetector always called (even for clear messages)
- ⚠️ Sequential LLM calls in Main.go (could parallelize)
- ⚠️ Sequential LLM calls in Agent (could parallelize)
- ⚠️ Context extraction not cached (same user, same info)

**Risk Assessment**:
- Quick wins (Priority 1): Very low risk, proven patterns
- Medium effort (Priority 2): Low-medium risk, good payoff
- Architectural (Priority 3): High risk, diminishing returns

---

## COST-BENEFIT SUMMARY

| Optimization | Tokens/Call | Est. Calls/Day | Token Cost/Day | Implementation Hours | Risk |
|--------------|-------------|----------------|-----------------|-------------------|------|
| Skip SubjectShift first msg | 300 | 1000/5 | 60K | 0.5 | None |
| Early exit clear messages | 300 | 1000/10 | 30K | 1 | Low |
| Parallelize Main | 0 | - | 0 | 2 | Low |
| Parallelize Agent | 0 | - | 0 | 2 | Low |
| Pattern-based shifts | 300 | 1000/5 | 60K | 3 | Low |
| Context cache (1hr) | 1000 | 1000/2 | 500K | 4 | Medium |
| **TOTAL Priority 1-2** | | | **650K/day** | **6-8 hrs** | **Low** |

---

**Conclusion**: Priority 1 optimizations are low-risk, high-impact quick wins. Implement in order. Priority 2 optimizations need careful testing of cache invalidation logic.
