# MOLY DEEP INVESTIGATION: Evidence Mapping & Root Cause Analysis

**Scope**: Analyze why Moly failed to understand and execute user's core goal across three messages, despite having all necessary information extracted correctly.

**Thesis**: Architecture is sound, but implementation breaks at goal tracking, information synthesis, and response generation stages.

---

## STAGE 1: FIRST MESSAGE - GOAL EXTRACTION

### What User Said
```
"I want to talk to you about a girl I am interested to. I saw her profile on fetlife... 
I was thinking of starting a chat with her and I need a first message for that. 
I don't want a simple 'hello' like the dozens she probably take every month, 
but something smart and playful. Can you help?"
```

### What Should Have Been Extracted
| Element | Value | Priority |
|---------|-------|----------|
| **PRIMARY GOAL** | Craft a first message to Christine_sub | **CRITICAL** |
| **Goal Type** | Task execution (not exploration) | **CRITICAL** |
| **Context** | FetLife platform, existing profile match | High |
| **Requirements** | Smart, playful, distinctive | High |
| **Constraints** | Avoid generic "hello" | High |
| **User Need** | Help with message composition | **CRITICAL** |

### What Was Actually Extracted (from logs)
```
[MessageProcessor] ✓ Extracted contact: Christine_sub (other, confidence=1.00)
[MessageProcessor] ✓ Extracted style: playful (confidence=1.00)
[Selector] Contact unknown - ambiguity: stakeholder
[Selector] Intention low confidence (0.00 < 0.6)
```

### ROOT CAUSE #1: GOAL NOT CAPTURED IN PRIMARY GOAL FIELD

**Evidence**: Logs show "intention=seek advice on initiating romantic interest" (vague) instead of "compose a smart, playful first message to Christine_sub" (specific).

**Impact**: System treats this as "gathering context" rather than "help me execute this task."

**Where It Breaks**: Layer 1-3 (extraction) extracted contact and style, but NOT the core deliverable the user wanted.

---

### What Moly Did
```
[ConversationAgent] HIGH-CONFIDENCE INTENT DETECTED: asking (confidence=0.90)
[ConversationAgent] Routing to response type: direct_answer
Moly Response: "Help me understand. Have you had any successful experiences 
initiating conversations with someone on Fetlife before..."
```

### Why This Response Was Correct (But Only Partially)
✅ **Right**: Asking about previous experience is sensible context-gathering
❌ **Wrong**: Doesn't acknowledge the core goal ("I need a first message")
❌ **Wrong**: Doesn't ask clarifying questions about Christine_sub specifically
❌ **Wrong**: Doesn't ask about the user's preferences for the message tone/approach

### What Better Question Should Have Been Asked
Instead of generic "have you had experience" → Should have asked:
- "Given that you want a smart and playful message, what's your usual opening style?"
- "From Christine_sub's profile, what stands out that you could reference?"
- "Should the message acknowledge you're both on a BDSM platform, or start more casually?"

**ROOT CAUSE**: Moly detected the user wanted clarification about context, but failed to recognize that the **goal is not to gather context, but to use context to execute a task.**

---

## STAGE 2: SECOND MESSAGE - STRATEGY DISCLOSURE

### What User Said
```
"I have no previous experience on fetlife or another online platform. 
I have only in real life though. Usually I start a conversation about sexual preferences, 
then I refer to bdsm, if the other person is interested I mention that I am dominant 
and some of my basic kinks. If the other person is interested I listen carefully 
and I ask more specifically about their kinks. Then I swift the topic on consensus 
and safety...
[... 7 more points of detailed methodology ...]
So, how do I start it?"
```

### What Should Have Been Extracted
| Element | Value | Why It Matters |
|---------|-------|----------------|
| **Methodology** | 9-step proven approach to BDSM conversations | Goldmine for message design |
| **Key Principle** | Questions > Statements | Core to approach |
| **Tone** | Calm, casual, theoretical | For first message style |
| **Progression** | Safety/consent comes LATER, after interest established | Sequential logic |
| **User Characteristic** | Experienced (IRL), thoughtful, safety-conscious | Confidence level |
| **Platform Adaptation** | "In online version of that the first part is not needed because preferences and kinks are already declared in our profiles" | KEY: User already understands FetLife |

### What Was Actually Extracted (from logs)
```
[SmartExtraction] Parsed 18 entities from pipe-delimited format
- characteristic|dominant|user|0.9
- preference|interested in bdsm|user|0.85
- preference|consensual and safe sex|user|0.95
[... 15 more entity extractions ...]
[ExtractionPhase] ✓ Extracted 18 entities (source=llm, avg_confidence=0.91)
```

### ROOT CAUSE #2: INFORMATION EXTRACTED BUT NOT SYNTHESIZED

**Evidence**:
- System extracted 18 individual entities ✅
- BUT failed to recognize this as a **unified methodology** ❌
- Treated as "collection of preferences" instead of "step-by-step approach" ❌

**Impact**: When generating response, system has data but not structure.

### What Moly Did
```
[ConversationAgent] Selected Socratic question: q_stakeholder_001 
"Besides you and them, who else might be affected by this situation?"
[ConversationAgent] [✓] Generated direct_answer response
"Given your experience in real-life interactions, transitioning to online platforms 
like FetLife requires a slightly different approach due to the anonymity and 
immediacy of digital communication."
```

### ROOT CAUSE #3: RESPONSE GENERATOR IGNORES USER'S STATED ADAPTATION

**Evidence**: User explicitly said: "In online version of that the first part is not needed because preferences and kinks are already declared in our profiles. So, how do I start it?"

**What Moly Did**: Suggested "transitioning to online platforms" as if this was a discovery the user needed.

**Impact**: User already adapted their methodology for online. Moly's response treats them as if they haven't thought about this.

### ROOT CAUSE #4: GOAL SWITCHED MID-CONVERSATION

**Evidence**:
- Message 1: "I need a first message"
- Moly's approach: "Gather context about your experience"
- Message 2: User provides context
- Moly's response: "Here's general online dating strategy" (NOT: "Let's craft your message")

**What Changed?** System lost track of the original goal.

**Where It Breaks**: There's no "goal persistence" mechanism. After gathering context, system doesn't return to original goal.

---

## STAGE 3: THIRD MESSAGE - PROFILE DATA (THE GOLDMINE)

### What User Said
```
"So, then, let's be more specific. Here are some insights from my profile:
46 male dominant. 100% Rigger 98% Dominant...
[23 detailed preferences/characteristics]
Here are some from her's:
Genders FemaleCisgender Roles submissive...
[14 detailed preferences/characteristics]
NOT INTERESTED IN: LONG DISTANCE, MALE SUBS/SWITCHES, CASUAL SEX, CYBERFUN"
```

### What Should Have Happened
**This is the moment to say**: "Perfect! I have everything I need. Let me help you craft a message that..."
- References something from HER profile (shows you read it)
- Establishes your dominance without being aggressive
- Creates opening for her to engage with her stated interests
- Follows your methodology: interest → establish relevance → safety later

### What Actually Happened
```
[SmartExtraction] Parsed 23 entities from pipe-delimited format
[ConflictDetector] Checking 23 current + 0 accumulated entities
[MessageProcessor] ✓ PHASE 4: Marked goal_extracted (intention=seek advice on online dating strategy)
[MessageProcessor] ⚠ RESPONSE BLOCKED: dominant - will ask clarification instead
[ConversationAgent] [✓] Generated direct_answer response:
"Hello, thank you for sharing your insights. It seems you're seeking 
a long-term relationship with a caring Dominant who can also act as a mentor..."
```

### ROOT CAUSE #5: RESPONSE GENERATOR MISUNDERSTANDS USER PROFILE

**Critical Error**: System treated the user's PROFILE DATA (what he's looking for) as HER profile data.

**Evidence**: 
- User said "Here are some insights from MY profile"
- Response: "It seems YOU'RE seeking a long-term relationship with a caring Dominant"
- Reality: HE'S seeking that? No. HE'S dominant. SHE'S seeking that.

**Impact**: System suggests user profile-swap or misunderstood the profiles entirely.

### ROOT CAUSE #6: GOAL COMPLETELY LOST

**Original Goal**: "I need a first message for that"

**What System Generated**: Response about finding a RELATIONSHIP (which is HER goal, not his immediate ask)

**What Was Needed**: Help writing a MESSAGE to start a conversation

**Missing**: Any actual message draft or message-writing guidance

### ROOT CAUSE #7: RESPONSE CONTRADICTION DETECTION (TOO LATE)

```
[MessageProcessor] ⚠ CRITICAL: Response contradicts extracted user characteristic 'dominant'
[MessageProcessor] ⚠ RESPONSE BLOCKED: dominant - will ask clarification instead
```

This is good safety mechanism, BUT:
- ❌ Only triggered AFTER response was already generated
- ❌ Should catch this during response planning, not after
- ❌ By now, user is confused about what Moly understood

---

## CROSS-STAGE ANALYSIS: WHERE THE SYSTEM BREAKS

### Layer 1-3: Extraction & Context Building ✅ Works
- Correctly extracts entities
- Identifies contacts
- Recognizes style preferences
- Builds analysis context

### Layer 4-5: Gap Detection & Conflict Resolution ⚠️ Partially Works
- Detects gaps but doesn't prioritize them against goal
- Conflicts detected too late (after response generation)

### Layer 6-7: Response Generation ❌ FAILS
- Loses sight of original goal
- Generates responses that don't match extraction
- Creates contradictions with extracted data
- Generic advice instead of goal-specific help

### Layer 8-11: Deepening & Denial ❌ FAILS
- No opportunity to deepen because response already wrong
- Denial detection triggers but response already sent

---

## ROOT CAUSES (VERIFIED FROM CODE INSPECTION - FINAL)

### ROOT CAUSE #A: Loop Pattern Detection Logic is Provably Wrong
**Problem**: Code checks for entity presence instead of actual pending clarifications.

**Evidence from Code** (`unified_orchestrator.go` lines 141-143):
```go
isAnsweringClarification := analysisCtx.CurrentMessage != "" &&
    analysisCtx.ExtractedConfidence > 0 &&
    len(analysisCtx.ExtractedEntities) > 0
```

This logic fires on ANY message that has:
- Non-empty content (always true for valid messages)
- Some extraction confidence (>0)
- Some entities extracted (which is nearly always true)

**Result**: Triggers on EVERY message after the first

**Evidence from Logs**:
```
Message 1: [Normal processing through all layers] ✓
Message 2: [🔄 LOOP PATTERN: Clarification detected] ✗ FALSE POSITIVE
           Message was: "I have no previous experience... here's my 9-step methodology"
           System thinks: answering a clarification question
           
Message 3: [🔄 LOOP PATTERN: Clarification detected] ✗ FALSE POSITIVE
           Message was: "Here are insights from my profile + her profile"
           System thinks: answering a clarification question
```

**What Should Happen**:
```go
// Check if there IS actually a pending clarification question
pendingClarifications := db.GetPendingClarifications(userID, conversationID)
isAnsweringClarification := len(pendingClarifications) > 0 && 
    messageAddresses(currentMessage, pendingClarifications)
```

**Impact When False**:
- When M2 arrives: `isAnsweringClarification = TRUE` (wrong!)
- Causes `startLayer = 3` (skip Layers 1, 2, 3)
- Layer 1 (extraction) doesn't run
- Layer 3 (maturity) doesn't run
- Maturity stays 0.00 forever
- Previous goals never loaded (see Root Cause B)

**Code Location**: `unified_orchestrator.go` line 141-148

---

### ROOT CAUSE #B: SetAccumulatedContext Exists But Is NEVER CALLED
**Problem**: System CAN load previous goal, but the code that does this is never invoked.

**Evidence from Code**: 

File `tools/layer_context.go` (lines 181-191):
```go
// SetAccumulatedContext sets context from previous messages
func (lc *LayerContext) SetAccumulatedContext(
    previousEntities []models.ExtractedEntity,
    previousGoal string,
    previousValues []string,
) {
    lc.AccumulatedExtractedEntities = previousEntities
    lc.PreviousGoal = previousGoal
    lc.PreviousValues = previousValues
}
```

**Code Search Result**:
```bash
$ grep -rn "SetAccumulatedContext" /moly-go --include="*.go"
/tools/layer_context.go:181:// SetAccumulatedContext sets context from previous messages
/tools/layer_context.go:183:func (lc *LayerContext) SetAccumulatedContext(

TOTAL MATCHES: 2 (definition and comment)
ACTUAL CALLS: 0 (ZERO)
```

**What This Means**:
1. Method exists: YES ✅
2. Method is called anywhere: NO ❌
3. Therefore: `PreviousGoal` is NEVER populated
4. Therefore: Previous goals are NEVER loaded
5. Therefore: Each message starts with empty goal state

**Evidence from Logs** (consequence of never calling SetAccumulatedContext):
```
Message 1: [MessageProcessor] intention=seek advice on initiating romantic interest
Message 2: [MessageProcessor] intention=seek advice on online dating strategy     ← NEW INTENTION EXTRACTED (overwrites)
Message 3: [MessageProcessor] intention=seek a long-term relationship...         ← NEW INTENTION EXTRACTED AGAIN (overwrites)
```

**What SHOULD Happen**:
1. M1 extracts and locks: `primary_goal = "Write smart, playful first message to Christine_sub"`
2. M2 should load: `lc.SetAccumulatedContext(..., previousGoal, ...)`
3. M2 should compare: Is current intent same/different/progressive?
4. M3 should load previous context again
5. Response generation should reference primary goal, not current intent

**Where It Breaks**: 
- Called nowhere: System can't load previous state
- Consequence: Fresh intention extracted M2 and M3 (overwrites)
- Consequence: System treats each message as independent goal
- Consequence: Original "write a message" goal forgotten after M1

**Code Location**: Defined in `tools/layer_context.go` line 181, should be called in `unified_orchestrator.go` ~line 152 (after LayerContext creation), but ISN'T

---

### ROOT CAUSE #C: Maturity Stuck at 0.00 Because Layer 3 Skipped
**Problem**: Maturity should accumulate across messages, but Layer 3 is skipped in false loop pattern.

**Evidence from Logs**:
```
Message 1:
[Layer3] ✓ Maturity calculated: 0.00 (new conversation)

Message 2:
[UnifiedOrchestrator] 🔄 LOOP PATTERN: Clarification detected - jumping to Layer 4
[Layer4] ▶ Starting gap detection (maturity=0.00) ← STILL 0.00!
```

Message 2 maturity should be higher (more context accumulated), but it's still 0.00.

**Why This Happens**:
1. Root Cause A triggers: false clarification detected
2. `startLayer = 3` (skip L1, L2, L3)
3. Layer 3 is skipped
4. Maturity never recalculated
5. Maturity stays 0.00

**Effect Chain**:
```
Maturity stuck at 0.00
  ↓
System thinks user context is always "immature" 
  ↓
Gap detection thinks more clarifications needed
  ↓
Response keeps asking for more context
  ↓
Even though context IS being provided, it's not being accumulated
```

**What Should Happen**:
```
Message 1: Layer 3 calculates maturity = 0.00 (minimal context)
Message 2: Layer 3 should recalculate with accumulated entities → 0.33 (gathered context)
Message 3: Layer 3 should recalculate again → 0.66+ (comprehensive context)
```

**Code Location**: 
- Layer 3 skipping: `unified_orchestrator.go` lines 166-172
- Maturity recalc needed: `maturity_calculator.go` (Layer 3) - but needs accumulated entities from SetAccumulatedContext

**Dependency**: Requires Root Cause B to be fixed (SetAccumulatedContext must be called) so Layer 3 has accumulated entities to work with

---

### ROOT CAUSE #D: Entity Extraction Doesn't Link Subjects Properly
**Problem**: LLM extracts entities without tagging WHO they belong to (USER vs CONTACT).

**Evidence from Logs**:
```
Message 3:
[SmartExtraction] Parsed 23 entities from pipe-delimited format
- characteristic|submissive|contact_name|0.90
- preference|good girl|contact_name|0.90
```

Notice: `contact_name` is a PLACEHOLDER STRING, not an actual contact name!

**Evidence from System**:
```
[MessageProcessor] ✓ Extracted contact: Christine_sub (submissive) ✓ CORRECT
[MessageProcessor] ✓ Extracted contact: Good girl (potential romantic) ✗ WRONG
```

System creates a new contact "Good girl" when it should link the characteristic to Christine_sub.

**Why This Happens**:
The LLM extraction prompt doesn't specify:
```
When extracting characteristics:
  - If about the USER: tag as USER
  - If about CONTACT_NAME: tag as CONTACT_Christine_sub
  NOT: contact_name (undefined placeholder)
```

**Current Behavior**:
- Extracts: "good girl" characteristic
- Tags as: "contact_name" (which LLM left as undefined)
- System creates: New contact "good girl" (deduplication fails)

**Should Be**:
- Extracts: "good girl" characteristic  
- Tags as: "CONTACT_Christine_sub"
- System links: To existing contact

**Impact**:
- False contacts created ("Good girl" as separate entity)
- Entity linking corrupted
- Contact deduplication impossible
- Response logic can't tell which traits belong to which person

**Code Location**: 
- LLM prompt: `context_extractor.go` (Layer 1) - needs to include subject context
- Entity validation: Entity parsing doesn't validate subject binding

**Example Fix Needed**:
```go
// In extraction prompt, add:
"When extracting characteristics, always specify:
 - characteristic|dominant|USER|0.9
 - characteristic|submissive|CONTACT_Christine_sub|0.9
 NOT: contact_name (use actual name)"
```

---

### ROOT CAUSE #E: Contradiction Detection Happens AFTER Response Generated
**Problem**: Response is fully generated THEN checked for contradictions. Too late to prevent bad response.

**Evidence from Logs**:
```
[ConversationAgent] [✓] Generated direct_answer response: "It seems you're seeking 
    a long-term relationship with a caring Dominant..."
[ConversationAgent] [✓] Learned about user: 6 characteristics
[ConversationAgent] Running constitutional analysis on generated response
[ConflictHandler] ⚠ RESPONSE CONTRADICTION DETECTED: 
    User extracted as 'dominant' but response suggests 'submissive'
[MessageProcessor] ⚠ RESPONSE BLOCKED: dominant - will ask clarification instead
```

Timeline: Response generated (624 seconds of work) → Constitutional analysis (after) → Contradiction found → Response blocked

**The Fundamental Problem**:
```
Current order:
  Generate response → THEN check for contradictions

Required order:
  Plan response → Check claims against facts → THEN generate response
```

**Code Location**:
- Constitutional evaluation called: Line 7 of orchestrator (POST-generation)
- Should be: Pre-generation planning phase (BEFORE response generation)

**Why This Matters**:
- When contradiction detected after: all generation work wasted, user sees nothing/clarification
- When contradiction detected before: wrong strategy avoided, correct response generated

**Consequence**:
- User experience: System took 10+ minutes to generate response, then blocked it
- User confusion: Why generate if you're going to block it?
- System loss: All context that was gathered is wasted

**Required Fix**:
Move contradiction detection to BEFORE Layer 6-7 response generation, not after

---

## SUMMARY: The System is Architecturally Inverted

The real problem is not "missing pieces" but **processing order is backwards**:

```
CURRENT (BROKEN):
Extract goal → Ask clarifications → Generate response → Check for contradictions → FAIL

REQUIRED:
Lock primary goal → Accumulate context across messages → 
Before response: verify no contradictions → Plan response strategy → Generate → Output
```

Contradiction detection, goal tracking, and loop pattern logic all need to move UPSTREAM in the pipeline.

---

## SYSTEM CAPABILITIES VS. ACTUAL BEHAVIOR

### What Moly's Architecture Can Do ✅

| Capability | Used? | Evidence |
|-----------|-------|----------|
| Extract specific contacts | ✅ Yes | "Christine_sub" extracted |
| Extract characteristics | ✅ Yes | "dominant" extracted |
| Extract user methodology | ✅ Yes | 18 entities extracted |
| Store in context | ✅ Yes | Appears in AnalysisContext |
| Detect contradictions | ✅ Yes | Detected dominant/submissive mismatch |
| Generate responses | ✅ Yes | Responses generated |
| Ask clarifying questions | ✅ Yes | Asked about experience |

### What Moly's Architecture Should Do But Doesn't ❌

| Capability | Missing? | Impact |
|-----------|----------|--------|
| Track original goal through all layers | ❌ Yes | Goal lost after extraction |
| Synthesize entities into strategies | ❌ Yes | Data extracted but not structured |
| Reference goal during response generation | ❌ Yes | Generic response instead of goal-aligned |
| Map entities to "who owns them" | ❌ Yes | Profile confusion |
| Use clarification answers to execute task | ❌ Yes | Clarifications don't progress toward goal |
| Generate task-specific responses | ❌ Yes | Advice-giving instead of task execution |

---

## THE CORE ARCHITECTURAL GAP

### Current Flow (What Happens)
```
L1-3: Extract context → L4-5: Detect gaps → L6-7: Generate generic advice → L8-11: (Never reached)
```

### Required Flow (What Should Happen)
```
L1-3: Extract context + PERSIST GOAL → L4-5: Detect gaps relative to GOAL → L6-7: Generate GOAL-ALIGNED response → L8-11: Deepen if goal-execution continues
```

### Missing Piece
There is NO mechanism that says:
- "User's goal is X"
- "I've gathered context A, B, C"
- "Now I generate response that USES A, B, C to accomplish X"
- "If successful, goal moves forward; if not, ask more specific question"

---

## IMPLEMENTATION ISSUES (Not Architectural)

### Issue #1: Goal Not Maintained in LayerContext
- Goal extracted in Layer 1
- Lost before Layer 6
- Response generator has no reference to it

### Issue #2: Response Type Selection
- "direct_answer" type is too generic
- Should have "task_execution" type for goals like "write X"
- "task_execution" would generate drafts, not advice

### Issue #3: No Synthesis Step
- Extraction creates entities
- Nothing transforms entities into "actionable insights"
- L4 detects gaps but doesn't create strategies

### Issue #4: Clarification Loop Never Closes
- Clarification question asked ("Have you had experience?")
- Answer provided ("Here's my 9-step approach")
- Never revisits original goal in light of answer
- Should say: "Got it! Using your approach + her profile, let me craft a message..."

---

## REQUIRED FIXES (ARCHITECTURAL RESTRUCTURING)

### Fix #1: Lock Primary Goal Mechanism
**Where**: Layer 1 extraction, maintain through Layer 11
**What**: 
- In Message 1: Extract and LOCK primary_goal
- In Messages 2+: Extract current_message_intent but DON'T override primary_goal
- Add `Conversation.PrimaryGoal` (immutable after M1) and `Message.CurrentIntent` (mutable)
**Files**: 
- `context_extractor.go` (L1) - Add primary goal detection
- `conversation_agent.go` (L6-7) - Check primary goal, use current intent as filter only
- `models/conversation.go` - Add PrimaryGoal field
**Impact**: Goal stays consistent; each message provides context for that goal, not new goal

### Fix #2: Loop Pattern Detection (Clarification vs. Continuation)
**Where**: `unified_orchestrator.go` - clarification detection logic
**What**: 
- Current: "if has entities + context → assume clarification"
- Required: "did user actually receive a clarification question?" 
- Check: `TemporaryFactStore` - are there pending clarification questions?
- If yes AND current message addresses them → clarification
- If no OR current message is new context → continuation (use full pipeline)
**Impact**: Stops false loop pattern triggering, allows maturity to accumulate properly

### Fix #3: Maturity Accumulation (Not Reset)
**Where**: Layer 3 (maturity calculator)
**What**:
- Message 1: Calculate initial maturity (0.00-0.33)
- Message 2: Add to accumulated entities, recalculate (should be 0.33+)
- Message 3: Add more entities, recalculate (should be higher)
- NOT: `maturity=0.00` every message
**Files**: `maturity_calculator.go` (L3)
**Impact**: Maturity properly tracks context accumulation across messages

### Fix #4: Entity Subject Binding
**Where**: Layer 1 entity extraction
**What**:
- LLM prompt must include current subject context
- When extracting from profile section: "These are characteristics of [Contact Name]"
- Tag every entity with owner: USER | CONTACT_[name] | CONTEXT
- Validate subject binding before returning entities
**Files**: 
- `smart_extraction.go` (L1) - Improve LLM prompt, add subject tracking
- Entity validation layer - Check no unbound subjects
**Impact**: "Good girl" extracted as Christine_sub.good_girl characteristic, not separate contact

### Fix #5: Pre-Generation Contradiction Check
**Where**: Before Layer 6 response generation
**What**:
- Create `ResponsePlanner` that occurs BEFORE response generation
- Plan extracts key claims that response will make
- Checks each claim against: extracted facts + conversation history
- If contradiction found: abort this response strategy, try alternative
- Only THEN generate response
**Files**: New: `response_planner.go`, called before response generation
**Impact**: Contradictions prevented, not detected after generation

---

## IMPLEMENTATION WORK ITEMS

### Priority 1: Loop Pattern Fix (HIGH - Blocks everything else)
- [ ] Rewrite clarification detection in `unified_orchestrator.go`
- [ ] Check: `TemporaryFactStore.GetPendingClarifications(userID)` 
- [ ] If count > 0 AND message addresses them → clarification (L4 onwards)
- [ ] If count == 0 OR message is new context → full pipeline (L1 onwards)
- [ ] Estimated: Medium effort, HIGH impact
- [ ] Files: `unified_orchestrator.go` line ~180

### Priority 2: Primary Goal Locking (HIGH - Enables everything else)
- [ ] Add `PrimaryGoal` field to `Conversation` model
- [ ] In L1 extraction (Message 1): Detect and lock primary goal
- [ ] In L1 extraction (Message 2+): Extract `CurrentMessageIntent` separately
- [ ] In L6-7 response generation: Reference `PrimaryGoal`, use `CurrentIntent` as filter
- [ ] Estimated: Medium effort, HIGH impact
- [ ] Files: `context_extractor.go`, `conversation_agent.go`, `models/conversation.go`

### Priority 3: Entity Subject Binding (MEDIUM - Fixes false contacts)
- [ ] Update LLM extraction prompt to include subject context
- [ ] Add subject field to extracted entities
- [ ] Validate no unbound subjects before returning
- [ ] Estimated: Low-medium effort, MEDIUM impact
- [ ] Files: `smart_extraction.go`, entity models

### Priority 4: Response Pre-Planning (MEDIUM - Prevents contradictions early)
- [ ] Create new `response_planner.go` module
- [ ] Call BEFORE generation in Layer 6
- [ ] Extract claims from proposed response
- [ ] Validate claims against facts
- [ ] Estimated: Medium effort, MEDIUM impact
- [ ] Files: New file, called from `conversation_agent.go` L6

### Priority 5: Maturity Accumulation (LOW - Depends on #1)
- [ ] Fix Layer 3 to accumulate entities instead of reset
- [ ] Recalculate from combined entity set each time
- [ ] Estimated: Low effort (once #1 is fixed)
- [ ] Files: `maturity_calculator.go`

---

## VERIFICATION CHECKPOINTS (Updated)

After implementing fixes in priority order, test with same conversation:

### After Fix #1 + #2: Loop Pattern & Primary Goal
**Test Message 1**:
- ✅ Primary goal locked: "Write smart, playful first message to Christine_sub"
- ✅ Maturity: 0.00
- ✅ Response: Asks clarification about methodology

**Test Message 2**:
- ✅ Loop detection: NO false "clarification" trigger (should be full L1-11)
- ✅ Maturity: 0.33 (accumulated, not reset to 0.00)
- ✅ Primary goal: STILL "Write smart, playful first message" (not changed to "online dating strategy")
- ✅ Response: References user's 9-step approach, asks about Christine_sub

**Test Message 3**:
- ✅ Loop detection: NO false trigger
- ✅ Maturity: 0.66+ (accumulated further)
- ✅ Contacts: Only "Christine_sub" exists (no false "Good girl" contact)
- ✅ Primary goal: UNCHANGED
- ✅ Response: Applies both profiles to original goal

### After Fix #4: Entity Subject Binding
**Test Message 3 Extraction**:
- ✅ Christine_sub's characteristics tagged as `owner=Christine_sub`
- ✅ No "Good girl" created as separate contact
- ✅ All profile data correctly linked to subject

### After Fix #5: Response Pre-Planning
**All Messages**:
- ✅ Contradictions caught during planning, not after generation
- ✅ No blocked responses (prevented before generation)
- ✅ Response claims match extracted facts

---

## CRITICAL REALIZATION

**The system isn't broken at extraction—it's broken at MANAGEMENT.**

```
What Moly has:
✅ Can extract "dominant" correctly
✅ Can extract user methodology
✅ Can extract profile data
✅ Can detect contradictions

What Moly's missing:
❌ Can't keep primary goal consistent
❌ Can't distinguish clarification from continuation
❌ Can't accumulate maturity
❌ Can't bind entities to subjects
❌ Can't catch contradictions before generation

Result: Extraction works, but system loses the plot.
```

This is why it's architectural, not just implementation. The fundamental message-processing loop needs restructuring:

**Current flow** (broken):
```
M1: Extract goal+intent → Ask clarifications
M2: Extract goal+intent (overwrites!) → Ask more clarifications
M3: Extract goal+intent (overwrites again!) → False response, block it
```

**Required flow**:
```
M1: Extract primary_goal (lock it) + current_intent → Ask clarifications
M2: Reuse primary_goal + extract current_intent → Check clarification answered?
    Yes: L4-11 with accumulated context
    No: L1-11 with full fresh pipeline
M3: Reuse primary_goal + extract current_intent → Check clarification answered?
    Yes: L4-11 with accumulated context
    No: L1-11
```

---

## CONCLUSION: Verified from Code Inspection

**Core Finding**: System is NOT architecturally broken—it has disconnected wiring.

**What the Code Shows**:

✅ **Capability Exists**:
- `SetAccumulatedContext()` method defined for loading previous state
- `LayerContext.PreviousGoal` field for tracking goals
- `ExtractedContext.Goals` array for multiple goals
- Layer 3 can recalculate maturity

❌ **But Wiring is Missing**:
- `SetAccumulatedContext()` defined but NEVER CALLED (verified: 0 calls in codebase)
- Clarification detection logic wrong (checks entities, not pending questions)
- Maturity recalculation never happens (Layer 3 skipped)
- Entity subjects not tagged (LLM prompt doesn't specify)

**Severity**: CRITICAL

**Root Causes** (Code-Verified):
1. **Loop Pattern Detection**: Wrong logic (line 141-143 `unified_orchestrator.go`) fires on EVERY message
2. **SetAccumulatedContext Never Called**: Function exists but zero invocations anywhere
3. **Maturity Stuck at 0.00**: Because Layer 3 skipped by false loop pattern
4. **Entity Subjects Not Linked**: LLM extraction doesn't tag USER vs CONTACT
5. **Contradiction Check Too Late**: Happens post-generation, can't prevent response

**Fix Strategy**: Wire existing functionality together (14 hours of focused work)
1. Call SetAccumulatedContext (2h)
2. Fix clarification detection logic (4h)
3. Load and persist previous goals (2h)
4. Link entity subjects in extraction (3h)
5. Move contradiction checking upstream (3h)

**Expected Outcome**: Same conversation produces consistent, goal-aligned responses with properly accumulated context
