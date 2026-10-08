# Workflow Integration Analysis - Where to Wire Contact Workflow

**Question:** Should contact workflow be PARALLEL or IN-MAIN-FLOW?  
**Answer:** IN-MAIN-FLOW, before Layer 1, STOPPING if clarifications needed

---

## CURRENT WORKFLOW (as traced in code)

### Flow in main.go (lines 2000-2090)

```
1. Line 2001-2010:   Build AnalysisContext
                      └─ Extract entities, history, contacts from message

2. Line 2016-2020:   Wire extraction clarifications to AnalysisContext
                      └─ Attach any LLM-detected gaps

3. Line 2024-2048:   Enhance AnalysisContext with ExtractionArtifact
                      └─ Add metadata, expiry, confidence

4. Line 2050-2055:   Cache entity extraction
                      └─ Pre-populate cache to avoid re-extraction

5. Line 2060-2069:   Load previous extraction state
                      └─ Accumulated entities, goals, progression

6. Line 2076:        ◀─────── CALL unifiedOrchestrator.ProcessMessage
                      └─ Orchestrator takes analysisCtx
```

### Flow inside UnifiedOrchestrator.ProcessMessage (lines 128-390+)

```
1. Line 147-170:     Detect if answering clarification
                      └─ Check if pending clarifications exist
                      └─ Check if message addresses them
                      └─ Set flag: isAnsweringClarification

2. Line 208:         Create LayerContext
                      └─ New context object for layers

3. Line 211-286:     Wire all data to LayerContext
                      ├─ ClarificationQuestions (line 211)
                      ├─ PendingClarifications (line 244)
                      ├─ SentenceAnalyses (line 260)
                      ├─ AccumulatedContext (line 268)
                      ├─ PrimaryGoal (line 298)
                      └─ Message summaries (line 282)

4. Line 318-390+:    Run Layers 1-11 sequentially
                      ├─ Skip Layer 2 if clarifying (line 322)
                      ├─ Skip layers that CanSkip() (line 339)
                      └─ Execute layer.Process() (line 351)
```

---

## THREE INTEGRATION OPTIONS

### Option A: PARALLEL (Separate workflow before orchestrator)

```
main.go:
  ├─ Build AnalysisContext
  ├─ Load previous extraction
  ├─ [NEW] Run ContactContextBuilder (SEPARATE)
  │  ├─ Detect contacts
  │  ├─ Resolve pronouns
  │  ├─ Detect ambiguities
  │  └─ Show clarification modal ← BLOCKING CALL
  ├─ If clarifications needed, return early
  ├─ Otherwise, wire contact context to analysisCtx
  └─ Call unifiedOrchestrator.ProcessMessage
```

**Pros:**
- Separate concerns (contact workflow independent)
- Clear responsibility boundary
- Can test independently

**Cons:**
- ❌ Duplicates context loading (already done in orchestrator)
- ❌ Orchestrator doesn't know about contacts yet
- ❌ Need to rebuild orchestrator's context-wiring after contact workflow
- ❌ More complex flow in main.go
- ❌ Contact results not available to all layers (some layers loaded before contact workflow)

---

### Option B: INLINE in Orchestrator, BEFORE Layer 1

```
UnifiedOrchestrator.ProcessMessage:

1. Line 147-170:     [EXISTING] Detect if answering clarification
                      └─ Check pending clarifications

2. Line 208:         Create LayerContext
                      └─ New context object

3. [NEW] Add Contact Workflow:
         ├─ Detect contacts in message
         ├─ Resolve pronouns using existing tools
         ├─ Detect ambiguities
         └─ Build contact context string

4. [NEW] Wire contact context to LayerContext:
         ├─ ActiveContacts
         ├─ PronounResolutions
         └─ ContactContext for extraction

5. [CONDITIONAL] If ambiguous contacts:
         ├─ Show clarification modal
         ├─ Get user response
         ├─ Update resolutions
         └─ Clear flag or return early?

6. Line 318-390+:    Run Layers 1-11 (Layer 1 gets contact context)
```

**Pros:**
- ✅ All context already loaded
- ✅ No duplication
- ✅ Contact results available to ALL layers
- ✅ Single entry point (ProcessMessage)
- ✅ Reuses existing infrastructure
- ✅ Follows pattern of clarification detection (already in code)

**Cons:**
- Makes orchestrator slightly larger
- Contact workflow AND layer execution both in same function

---

### Option C: INLINE + BLOCKING CLARIFICATIONS (Recommended)

```
UnifiedOrchestrator.ProcessMessage:

1. Line 147-170:     [EXISTING] Detect if answering clarification

2. Line 208:         Create LayerContext

3. [NEW] Contact Workflow (CRITICAL SECTION):
         ├─ Detect contacts
         ├─ Resolve pronouns
         ├─ Calculate confidence
         ├─ Identify ambiguities
         │
         ├─ If ambiguity confidence < 0.50:
         │  ├─ Show clarification modal ← BLOCKING
         │  ├─ Wait for user response
         │  ├─ Update contact resolutions
         │  └─ Continue to step 4
         │
         └─ Otherwise: Continue to step 4

4. Wire ALL context to LayerContext (including contact results)

5. Run Layers 1-11
   └─ All layers have complete contact context
```

**Pros:**
- ✅ Contact ambiguities resolved BEFORE layers
- ✅ All downstream layers have clean data
- ✅ No false contradictions in Layer 5+
- ✅ Clarifications happen early, not deep in layers
- ✅ Follows existing pattern (like pending clarifications)
- ✅ User gets clarification modal early

**Cons:**
- Main flow "pauses" for clarifications
- Slightly different from existing clarification pattern

---

## COMPARISON TABLE

| Aspect | Option A (Parallel) | Option B (Inline) | Option C (Inline+Block) |
|--------|---------------------|------------------|------------------------|
| **Context loading** | Duplicated | Single ✓ | Single ✓ |
| **Available to all layers** | ❌ No | ✅ Yes | ✅ Yes |
| **Code location** | main.go | orchestrator | orchestrator |
| **Blocking clarifications** | ✅ Yes | ❌ No | ✅ Yes |
| **Complexity** | High (dual flows) | Medium | Medium |
| **False contradictions** | ❌ Possible | ⚠️ Possible | ✅ Prevented |
| **Test independence** | ✅ Easy | ⚠️ Coupled | ⚠️ Coupled |

---

## RECOMMENDATION: Option C (Inline + Blocking)

**Why:**

1. **Data Integrity First**
   - Contact ambiguities resolved BEFORE layers analyze
   - No false contradictions in Layer 5
   - All layers get clean, unambiguous contact data

2. **Follows Existing Pattern**
   - Already has clarification detection (line 147-170)
   - Contact workflow is similar: detect problem → ask if needed
   - Not introducing new pattern, extending existing

3. **Architectural Consistency**
   - All context wiring happens in ProcessMessage
   - Single entry point, single exit point
   - LayerContext is fully prepared before layers run

4. **User Experience**
   - Clarifications surfaced early (not mid-layer)
   - Modal appears before main response
   - Natural interaction flow

5. **Performance**
   - No context reloading
   - Minimal overhead
   - Contact resolution reuses existing tools

---

## IMPLEMENTATION LOCATION

### In unified_orchestrator.go, after line 206 (after IsMessageOne setup)

```go
// === EXISTING CODE (line 147-206) ===
// Detect clarifications, create LayerContext, wire basic context

// === NEW: Contact Workflow (add here) ===
// [Before any layer execution]

var activeContacts []Contact
var pronouResolutions map[string]PronounResolution
var contactContext string

// 1. Detect contacts
contactDetector := NewContactDetector()
detectedContacts := contactDetector.DetectInMessage(message, analysisCtx)

// 2. Resolve pronouns
pronounResolver := NewPronounResolver(db)
for _, contact := range detectedContacts {
    resolutions := pronounResolver.ResolveAntecedent(contact, detectedContacts)
    // Calculate confidence...
    // Check if ambiguous (< 0.50)...
    // If ambiguous: show modal and wait
}

// 3. Wire to LayerContext
lc.ActiveContacts = activeContacts
lc.ContactContext = buildContactContextString(activeContacts)

// === THEN: Existing layer execution (line 318+) ===
// Layer 1 receives contact context in lc
```

---

## DATA FLOW DIAGRAM

```
Message Input
    │
    ▼
┌─────────────────────────────────┐
│ main.go                         │
│ ├─ Build AnalysisContext       │
│ ├─ Load previous extraction     │
│ └─ Call ProcessMessage          │
└──────────┬──────────────────────┘
           │
           ▼
┌─────────────────────────────────┐
│ UnifiedOrchestrator             │
│ .ProcessMessage()               │
├─────────────────────────────────┤
│                                 │
│ 1. Create LayerContext          │
│                                 │
│ 2. Wire base context            │
│                                 │
│ 3. [NEW] Contact Workflow:      │
│    ├─ Detect contacts           │
│    ├─ Resolve pronouns          │
│    ├─ Detect ambiguities        │
│    │                            │
│    ├─ If ambiguous (< 0.50):    │
│    │  └─ Show modal ← BLOCKING  │
│    │     Wait for response      │
│    │     Update resolutions     │
│    │                            │
│    └─ Wire to LayerContext      │
│                                 │
│ 4. Run Layers 1-11              │
│    ├─ Layer 1: Extract (HAS     │
│    │            contact context)│
│    ├─ Layer 2-4: Process        │
│    └─ Layer 5-11: Use clean data│
│                                 │
└──────────┬──────────────────────┘
           │
           ▼
Generate Response (all contact data is clean)
```

---

## INTEGRATION CHECKLIST

- [ ] Add ContactDetector to orchestrator
- [ ] Add PronounResolver (already exists)
- [ ] Implement confidence scoring
- [ ] Implement ambiguity detection
- [ ] Wire modal for user response
- [ ] Update LayerContext to hold contact data
- [ ] Update Layer 1 extraction prompt to receive contact context
- [ ] Test with multi-contact conversations
- [ ] Verify no false contradictions in Layer 5

---

## TIMING: BLOCKING vs NON-BLOCKING?

**Question:** Should clarifications BLOCK the main flow?

**Answer:** YES - here's why:

1. **Data dependency**: Layers 1-11 all depend on knowing WHO
2. **False positives**: Without clear contact knowledge, Layer 5 flags false contradictions
3. **User experience**: Better to ask clarification upfront than bury it in response
4. **Architecture**: Existing pattern: detect problem → ask if needed → continue
5. **Efficiency**: Don't run 11 layers with ambiguous data, clarify first

**Comparison:**

**Non-blocking (bad):**
```
Contact ambiguous (confidence 0.40)
  → Don't ask
  → Run layers with ambiguous data
  → Layer 5 finds false contradictions
  → Response includes unnecessary clarifications
  → User confused
```

**Blocking (good):**
```
Contact ambiguous (confidence 0.40)
  → Ask in modal immediately
  → User responds
  → Proceed with clear contact data
  → All layers have clean data
  → Response focused on actual gaps
```

---

## CONCLUSION

**Implementation approach: Option C (Inline + Blocking)**

**Location:** UnifiedOrchestrator.ProcessMessage, between line 206 and 318

**Key characteristics:**
- ✅ Runs BEFORE layers
- ✅ Blocks main flow if clarifications needed
- ✅ Wires results to LayerContext
- ✅ All layers receive clean contact data
- ✅ Reuses existing infrastructure
- ✅ Follows existing patterns

**Expected outcome:**
- ✅ No false contradictions
- ✅ Correct subject attribution
- ✅ Progressive naming support
- ✅ Proper pronoun resolution
- ✅ Clean downstream data

