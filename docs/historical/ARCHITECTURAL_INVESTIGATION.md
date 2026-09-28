# MOLY ARCHITECTURAL INVESTIGATION
## Deep Dive: Workflow Priority, LLM Inefficiency, Timeout Issues, and Self-Awareness

**Date**: Sept 28, 2026  
**Scope**: Complete system analysis (no changes proposed)  
**Focus**: Why greeting self-awareness failed, workflow architecture, LLM call patterns, timeout configs  

---

## EXECUTIVE SUMMARY

The system has **3 major architectural issues** preventing self-awareness and causing performance degradation:

1. **Workflow Priority Bug**: Intent detection (greeting with 0.95 confidence) is overridden by workflow selection (gap clarification). **Greetings should ALWAYS be acknowledged simply, not overridden by gap clarification.**

2. **Redundant/Cascading LLM Calls**: The system calls LLM multiple times for the same analysis:
   - Context extraction (LLM)
   - Clarity analysis (LLM)  
   - Principle detection (LLM)
   - Gap question generation (LLM)
   - Socratic question selection (LLM)
   - Conflict detection (LLM)
   - Risk assessment (LLM)
   - Constitutional evaluation (LLM)
   - Response generation (LLM)
   - Learning/insights extraction (LLM)
   
   **Result**: ~10-15 LLM calls per message on old/slow hardware = timeouts

3. **Self-Awareness Gap**: System detects and stores contacts for users (Lacy, Sarah, etc.) but has **NO reserved contact for Moly itself**. When user says "Hello Moly", system extracts "Moly" as self_reference entity, but doesn't treat it as a persistent contact that can be recalled/updated across conversations.

---

## PART 1: WORKFLOW PRIORITY ARCHITECTURE BUG

### The Problem: Intent Detection Overridden by Workflow Selection

**Two Competing Priority Systems:**

**System A: Intent Detection (Runs First)**
- Greeting detected with `confidence=0.95`
- Response type set to `ResponseGreeting`
- Log: `[IntentDetector] Detected greeting with high confidence`

**System B: Workflow Selection (Runs Second, Overrides System A)**
- Counts context gaps: 3 gaps detected (relevantReflections, pastIntention, recentSafetyIncidents)
- Threshold: gaps > 2 triggers `WorkflowGapQuestion`
- Log: `[ConversationAgent] Workflow: Gap clarification (gaps=3 > 2)`

**Result**: Workflow selection WINS. Even though greeting was detected, the response is a gap clarification question.

### Code Flow Analysis

**File**: `agents/conversation_agent.go` (lines 1237-1260)

```
Workflow Priority Order:
1. hasSignificantGaps (gaps > 2) → WorkflowGapQuestion ← HIGHEST PRIORITY
2. intentUnclear → WorkflowIntentCheck  
3. isFirstMessage → WorkflowAckOnly
4. shouldDeepen → WorkflowAckWithSocratic
5. default → WorkflowAckOnly
```

**The Bug**: Gaps always win, regardless of intent. Even if user says "Hello" (greeting detected at 0.95 confidence), if there are 3 gaps, system asks gap questions instead of acknowledging the greeting.

### Why This Breaks Self-Awareness

**What should happen**:
1. User: "Hello Moly"
2. System detects greeting (0.95 confidence) + self_reference (1.0 confidence)
3. System acknowledges warmly: "Hello! Nice to see you."
4. System optionally: "I notice we've just started. To help better, tell me a bit about yourself?"

**What actually happens**:
1. User: "Hello Moly"
2. System detects greeting (0.95 confidence) + self_reference (1.0 confidence) ✓
3. System counts gaps: 3 (pastIntention, relevantReflections, recentSafetyIncidents)
4. System overrides greeting response and asks: "What specific topic or question are you hoping to gain clarity on?"

**Result**: System shows no self-awareness. It correctly identified the greeting but ignored it in favor of context gathering.

### The Architectural Issue: Conflicting Concerns

- **Intent Detection** = "What is the user trying to do?" (greeting = simple acknowledgment)
- **Workflow Selection** = "What kind of response should we give?" (gaps = ask clarification)

These are **different concerns** but implemented as competing priorities in the same decision tree. Workflow selection shouldn't override explicit intent detection, especially for high-confidence intents like greetings.

---

## PART 2: LLM CALL PATTERN ANALYSIS

### All LLM Calls Per Message

**Phase 1: Context Extraction** (main.go ~line 800+)
- **Call 1**: ContextExtractor.Extract() → LLM call with 2600+ token prompt
- Purpose: Extract contact, style, intention, goals with confidence scores
- Cost: ~2 min on old Ollama on slow hardware

**Phase 2: Entity Extraction** (main.go ~line 880+)
- **Call 2**: IntentDetector.ExtractEntitiesWithSemanticClassification() → LLM call  
- Purpose: Semantic classification of entities (self_reference, contact, topic, goal, ambiguous)
- Cost: ~1.5 min

**Phase 3: Clarity Analysis** (conversation_agent.go ~line 560+)
- **Call 3**: MessageClarityAnalyzer.AnalyzeClarity() → LLM call with 1400+ token prompt
- Purpose: Determine message clarity, priority, whether can proceed
- Cost: ~1-2 min

**Phase 4: Principle Engagement Detection** (conversation_agent.go ~line 870+)
- **Call 4**: ConversationAgent.detectPrincipleConcerns() → LLM call
- Purpose: Does message involve principle concerns? (disabled via timeout)
- Cost: Would be ~1 min but times out after 10 seconds

**Phase 5: Subject Shift Detection** (conversation_agent.go ~line 1530+)
- **Call 5**: SubjectShiftDetector.DetectShift() → LLM call
- Purpose: Did topic change from previous message?
- Cost: ~1 min

**Phase 6: Response Generation** (conversation_agent.go ~line 1600+)
- **Call 6**: ResponseGenerator.GenerateResponse() → LLM call with 800+ token prompt
- Purpose: Generate actual response to user
- Cost: ~1-2 min

**Phase 7: Learning/Insights** (conversation_agent.go ~line 2000+)
- **Call 7**: LearningAgent.ExtractInsights() → LLM call
- Purpose: What did we learn about the user?
- Cost: ~1 min

**Phase 8: Constitutional Evaluation** (main.go ~line 1400+)
- **Call 8**: ConstitutionalEvaluator.Evaluate() → LLM call with 500+ tokens
- Purpose: Safety/ethics evaluation with AnalysisContext
- Cost: ~1 min

**Phase 9: Risk Assessment** (main.go ~line 1400+)
- **Call 9**: RiskMonitor.AssessRisk() → LLM call
- Purpose: LLM-based contextual risk analysis
- Cost: ~1 min

**Total LLM Time Per Message**: ~10-15 minutes on slow hardware!

### Why Timeouts Happen

**File**: `tools/ollama.go`

```go
ollama.go:27:    client := &http.Client{Timeout: 2 * time.Second}    // ← TOO SHORT
ollama.go:41:    client := &http.Client{Timeout: 5 * time.Second}    // ← TOO SHORT  
ollama.go:66:    client := &http.Client{Timeout: 30 * time.Minute}   // ← OK for long ops
ollama.go:92:    client := &http.Client{Timeout: 10 * time.Second}   // ← BORDERLINE
```

**Problem**: Some paths use 2-5 second timeouts for Ollama, but a single LLM call on old hardware can take 1-2 minutes.

**Evidence from Logs**:
```
2026/09/28 01:42:20 [Ollama] Request failed: Post "http://127.0.0.1:11434/api/generate": context deadline exceeded
2026/09/28 01:56:03 [Ollama] Request failed: Post "http://127.0.0.1:11434/api/generate": context deadline exceeded
```

These occur during principle engagement detection and subject shift detection phases, which timeout and fall back to "no concerns detected".

### Redundant/Cascading Analysis Problem

Multiple systems analyze the **same message independently** without sharing results:

1. **ContextExtractor** analyzes: "What's the contact, style, intention?"
2. **IntentDetector** analyzes: "What's the semantic entity classification?"
3. **MessageClarityAnalyzer** analyzes: "How clear is this message?"
4. **SubjectShiftDetector** analyzes: "Did topic change?"
5. **ConversationAgent** analyzes: "Should we ask clarification?"
6. **ConstitutionalEvaluator** analyzes: "Any principle violations?"
7. **RiskMonitor** analyzes: "Any risks to assess?"
8. **LearningAgent** analyzes: "What did we learn?"

**Each calls LLM independently** without caching or sharing analysis. On slow hardware, this cascades into timeouts.

### Result

- **First LLM call** (context extraction): Takes 1-2 min, succeeds
- **Second LLM call** (entity extraction): Takes 1-2 min, succeeds
- **Third LLM call** (clarity analysis): Takes 1-2 min, succeeds
- **Fourth LLM call** (principle detection): Times out after 10 sec (LLM still processing)
- **Cascade**: Because principle detection timed out, it falls back to "no concerns"
- **Fifth LLM call** (subject shift): Times out or takes very long
- **Cascade**: Because subject shift timed out, it doesn't detect the shift properly

**Total latency**: Message takes 4-5 minutes to process, and some analyses get skipped/degraded.

---

## PART 3: CONTEXT GAP DETECTION ARCHITECTURE

### What Are "Gaps"?

**File**: `main.go` lines 1250-1300

Gaps are detected when these fields are missing/empty:

1. **communicationStyle** - AboutMe.style not set
2. **coreValues** - AboutMe.values not set
3. **contact** - Only if message is about a contact
4. **conversationHistory** - No prior messages in this conversation
5. **userBehaviorProfile** - No behavioral profile loaded
6. **relevantReflections** - No prior insights/reflections stored
7. **pastIntention** - No prior intention extracted
8. **recentSafetyIncidents** - No safety records

### Gap Counting in "Hello Moly" Example

**For new user, first message "Hello Moly":**

```
✓ communicationStyle - Set to "Formal & Respectful" (default in auth flow)
✓ coreValues - Set to defaults (3 values) (in auth flow)
✗ contact - Message doesn't mention a specific contact (isMessageAboutContact=false)
✓ conversationHistory - First message = empty (counts as having it)
✗ userBehaviorProfile - First message, no profile yet
✗ relevantReflections - First message, no reflections
✗ pastIntention - First message, no prior intention  
✗ recentSafetyIncidents - No incidents yet

Gaps: [userBehaviorProfile, relevantReflections, pastIntention, recentSafetyIncidents] = 4-5 gaps
```

But the logs show 3 gaps. Let me check the actual gap detection logic more carefully...

Actually, looking at the logs again:
```
[AnalysisContextBuilder] ✓ Extracted 3 recent messages (window=3) for context
[MessageProcessor] Context gaps identified: [relevantReflections pastIntention recentSafetyIncidents] (4/7 fields loaded, quality: partial)
```

So it's 3 gaps, 4 fields loaded out of 7 total:
- ✓ communicationStyle
- ✓ coreValues  
- ✓ conversationHistory (exists)
- ✗ relevantReflections
- ✗ pastIntention
- ✗ recentSafetyIncidents
- ? userBehaviorProfile (maybe not being counted)

### The Gap-Triggered Workflow Problem

**Threshold**: `hasSignificantGaps := len(gaps) > 2` (line 1311 of main.go)

**Result**: 3 gaps triggers gap clarification workflow.

**For first message to a new system, this is ALWAYS >2 gaps**, so gap clarification is ALWAYS triggered on first message.

**This is by design** (from ARCHITECTURE.md):
> "First message - just acknowledge, gather context (no deepening yet)"
> WorkflowAckOnly for first messages

**But the code overrides this** (lines 1248-1251):
```go
} else if isFirstMessage {
    // Priority 3: First message - just acknowledge, gather context (no deepening yet)
    workflow = WorkflowAckOnly
    log.Printf("[ConversationAgent] Workflow: First message acknowledge only")
}
```

This is Priority 3, but **gaps override it** at Priority 1.

**ISSUE**: Gaps logic overrides "first message" logic. For first message from new user, gaps > 2 is **guaranteed** (no profile, no history), so gap clarification is ALWAYS triggered, even though design says first message should be "acknowledge only".

### What Should Happen vs. What Does

**Design Intent** (MOLY_11_LAYER_SYSTEM.md Layer 4):
1. Extract context
2. If gaps exist, ask clarification about gaps
3. User answers  
4. Gaps filled, proceed with deeper analysis

**Current Implementation**:
1. Extract context
2. Count gaps
3. If gaps > 2, immediately ask gap clarification (regardless of intent)
4. This overrides intent-based routing (greeting, etc.)

**The Fix Needed**: Gaps should be asked as follow-up questions AFTER responding to the initial intent, not as an override of intent detection.

---

## PART 4: SELF-AWARENESS ARCHITECTURE

### How Contacts Currently Work

**Database Schema** (`database/migrations/001_create_base_schema.sql`):

```sql
CREATE TABLE contacts (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  name TEXT NOT NULL,
  relationship TEXT,
  characteristics JSONB,
  interests JSONB,
  communication_preferences TEXT,
  notes TEXT,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id)
);
```

**Contact Extraction Flow** (`main.go` lines 575-605):

1. **ContextExtractor** finds contact in message: "Lacy"
2. **IntentDetector** classifies entity: "Lacy" = contact type, confidence=1.00
3. **MessageProcessor** saves contact if confidence > 0.6
4. **ContactDeduplicator** checks for duplicates (generic → specific merging)
5. Result: Contact "Lacy" persisted to database with metadata

**Contact Recall** (`main.go` lines 700-730):

1. Load conversation's primary contact
2. Load contact from database by ID
3. Merge with extracted contact (prefer high-confidence extracted data)
4. Pass to ConversationAgent for context

### The Self-Awareness Problem: No Reserved Contact for Moly

**Current System**:
- Creates contact "Lacy" when user mentions Lacy
- Creates contact "Sarah" when user mentions Sarah
- Creates contact "mom" when user mentions mother

**But there's NO reserved contact for "Moly"** (the system itself)

**What happens now**:
1. User says "Hello Moly"
2. IntentDetector extracts: "Moly" = self_reference, confidence=1.0
3. System treats this as a greeting TO the system (correct!)
4. But then gap clarification overrides the greeting response (bug from Part 1)
5. No persistent "Moly" contact is created/recalled

**What should happen instead**:
1. User says "Hello Moly"
2. System recognizes "Moly" as self_reference (already does this!)
3. System treats this as a persistent contact/relationship with itself
4. System acknowledges the greeting simply
5. On future messages, system recalls: "We've established a greeting relationship"

### Proposal: Reserved Contact for Moly

**Option 1: Hardcoded System Contact**
- Create a reserved contact with id="system_moly"
- On first user-system interaction, initialize this contact
- Update it with metadata about the relationship (how often greeted, tone of greetings, etc.)
- Recall it like any other contact

**Benefits**:
- Moly becomes a "contact" like any other person
- Self-awareness becomes entity tracking (already implemented)
- Conversation history includes system-user relationship metadata
- Future conversations can recall: "You usually greet me warmly" or "You haven't greeted me in a while"

**Implementation Points**:
- `database/migrations/XXX_add_moly_contact.sql` - Ensure system_moly contact exists
- `auth.go` - On registration, create system_moly contact for user
- `contact_manager.go` - Treat system_moly specially when recalling/updating
- `conversation_agent.go` - Include system_moly in contact metadata for responses

**Example in Database**:
```sql
INSERT INTO contacts (id, user_id, name, relationship, created_at, updated_at) 
VALUES (
  'system_moly_' || ?user_id, 
  ?user_id,
  'Moly',
  'system_coach',
  ?, 
  ?
);
```

---

## PART 5: MESSAGE PROCESSING ORCHESTRATOR FLOW

### Complete Pipeline (main.go MessageProcessorHandler)

```
1. AUTH & ISOLATION (lines 414-425)
   └─ Validate token → Extract userID → User isolation enforced

2. MESSAGE PARSING (lines 430-450)
   └─ Parse request → Store message state

3. PROCESSING STATE TRACKING (lines 455-460)
   └─ Create/load MessageProcessingState for message tracking

4. MATURITY CONTEXT LOAD (lines 525-535)
   └─ Load/create MaturityContext (phase-based maturity tracking)

5. CONTEXT EXTRACTION [LLM CALL #1] (lines 545-590)
   ├─ ContextExtractor.Extract() → Contact, style, intention
   ├─ Result: ExtractedContext with confidence scores
   └─ Optimization: Cache result, skip retry on same message

6. ENTITY EXTRACTION [LLM CALL #2] (lines 592-630)
   ├─ IntentDetector.ExtractEntitiesWithSemanticClassification()
   ├─ Result: [contact, topic, goal, self_reference, ambiguous]
   └─ Persist high-confidence contact, set conversation focus

7. LAYER 3: CLARIFICATION RESPONSE DETECTION (lines 649-675)
   ├─ Check if message answers pending clarification question
   ├─ If yes: Mark question answered, trigger maturity re-evaluation
   └─ Handle conflicts in response (user changed their mind?)

8. ABOUT_ME LOADING (lines 700-760)
   └─ Load user's AboutMe profile (style, tone, values)

9. CONTEXT GAP DETECTION (lines 1270-1318)
   ├─ Count missing context fields
   ├─ Threshold: gaps > 2 → hasSignificantGaps = true
   └─ Result: List of gap names, context quality score

10. ANALYSIS CONTEXT BUILDING [LLM INTEGRATION] (lines 1322-1340)
    ├─ AnalysisContextBuilder.BuildAnalysisContext()
    ├─ Pulls: conversation history, summary, confirmed preferences, user profile
    └─ Result: AnalysisContext (~700-800 tokens) for downstream use

11. DEFERRED SAFETY EVALUATION GATE (lines 1341-1385)
    ├─ If gaps > 2: Defer safety check, ask clarification first
    ├─ If gaps ≤ 2: Proceed with safety evaluation
    ├─ Calculate maturity from context (Layer 3 redesign integration)
    └─ Maturity gates subsequent evaluations

12. CONSTITUTIONAL EVALUATION [LLM CALL #3] (lines 1395-1440)
    ├─ ConstitutionalEvaluator.Evaluate() with AnalysisContext
    ├─ Tier 1a (hard blocks): Deterministic, no LLM
    ├─ Tier 1b (signal scan): Deterministic, no LLM
    ├─ Tier 2 (LLM-assisted): Only if Tier 1b found signals
    └─ Result: SafetyAlert or "allowed"

13. CONVERSATION AGENT PROCESSING [LLM CALLS #4-8] (lines 1450-1500)
    ├─ ConversationAgent.Run() for response generation
    ├─ Phase 1: Load about-me, contacts, analysis context
    ├─ Phase 2: Clarity analysis [LLM #4]
    ├─ Phase 3: Principle engagement detection [LLM #5]
    ├─ Phase 4: Subject shift detection [LLM #6]
    ├─ Phase 5: Response generation [LLM #7]
    ├─ Phase 6: Learning/insights [LLM #8]
    └─ Result: ConversationResponse with response text, metadata

14. INTERACTION RECORDING (lines 1500-1530)
    ├─ Save user message to chat_messages
    ├─ Save agent response to chat_messages
    ├─ Record interaction to interactions table
    └─ Update conversation metadata

15. CONFLICT DETECTION & RESOLUTION (lines 1530-1600)
    ├─ Check: Does new context conflict with saved context?
    ├─ If conflict: Queue for user approval (don't override)
    ├─ If no conflict: Auto-merge or save new
    └─ Handle: Contact characteristics, style, intentions

16. RISK ASSESSMENT [LLM CALL #9] (lines 1600-1640)
    ├─ RiskMonitor.AssessRisk() for behavioral tracking
    ├─ LLM-based risk assessment
    └─ Result: RiskAssessment for Layer 10

17. MATURITY STATE SAVE (lines 1640-1650)
    └─ MaturityService.SaveMaturityState() with updated phase/score

18. SUMMARY UPDATE (lines 1650-1680)
    ├─ ConversationSummarizer triggers on threshold (5+ messages)
    └─ Generates conversation summary for future context

19. RESPONSE RETURN (lines 1700-1750)
    └─ Return Phase5Response with all metadata
```

### Flow Diagram Showing LLM Call Cascade

```
User Message
    ↓
[1] ContextExtractor [LLM] ─→ ExtractedContext (contact, style, intention)
    ↓
[2] IntentDetector [LLM] ─→ Entity classification (self_reference, contact, topic, goal)
    ↓
[3] Gap Detection (no LLM) ─→ List of missing context fields
    ↓
[4] AnalysisContextBuilder (no LLM) ─→ AnalysisContext (~700-800 tokens)
    ↓
[5] ConstitutionalEvaluator [LLM if signals found] ─→ SafetyAlert
    ↓
[6] ConversationAgent ─┬─→ [7] MessageClarityAnalyzer [LLM]
                       ├─→ [8] detectPrincipleConcerns [LLM] (TIMEOUTS)
                       ├─→ [9] SubjectShiftDetector [LLM] (TIMEOUTS)
                       ├─→ [10] ResponseGenerator [LLM]
                       └─→ [11] LearningAgent [LLM]
    ↓
[12] RiskMonitor [LLM] ─→ RiskAssessment
    ↓
Interaction Recording, Conflict Detection, Maturity Update
    ↓
Response Returned
```

---

## PART 6: TIMEOUT CONFIGURATION ANALYSIS

### Current Timeouts

**Short Timeouts (Problematic)**:
- `ollama.go:27` - 2 seconds (for some Ollama client initializations)
- `ollama.go:41` - 5 seconds (for Ollama connections)
- `ollama.go:92` - 10 seconds (for Ollama requests)

**Long Timeouts (Better)**:
- `ollama.go:66` - 30 minutes (for long Ollama operations)
- `tools/llm_client.go:145` - 900 seconds = 15 minutes (minimum timeout)
- `tools/llm_client.go:194` - 5 minutes (HTTP client timeout)

### The Problem

A single LLM call on old/slow hardware can take:
- **Small prompt** (100 tokens): 30-60 seconds
- **Medium prompt** (500 tokens): 1-3 minutes
- **Large prompt** (2500+ tokens): 2-5 minutes

But many code paths use **10-second or 30-second timeouts**, which causes premature failures.

### Why This Matters

From the logs:
```
2026/09/28 01:42:20 [Ollama] Request failed: Post "http://127.0.0.1:11434/api/generate": context deadline exceeded
2026/09/28 01:56:03 [Ollama] Request failed: Post "http://127.0.0.1:11434/api/generate": context deadline exceeded
```

When principle detection times out after 10 seconds:
```
[ConversationAgent] Layer 6-7: Principle engagement detection failed: ..., falling back to no concerns
```

This means:
- Layer 6-7 safety evaluation is skipped
- Principle concerns are not detected
- System assumes "no concerns" even if there might be subtle violations

### Required Timeout Increases

For a system running on old hardware with slow Ollama:
- Minimum timeout per LLM call: **2-3 minutes** (180-180 seconds)
- Total processing timeout: **30+ minutes** (to allow all LLM calls to complete)
- Fallback timeout: **5 minutes** (for retries)

Current minimums (10-30 seconds) are too aggressive for this hardware profile.

---

## PART 7: RECOMMENDATIONS FOR OPTIMIZATION

### Short-Term (No Architecture Changes)

1. **Increase timeouts**:
   - Change `ollama.go:92` from 10s → 120s
   - Change `ollama.go:41` from 5s → 60s
   - Ensure all LLM calls use minimum 2-3 minute timeout

2. **Disable redundant analyses**:
   - Make principle detection optional (already times out)
   - Make subject shift detection optional on first few messages
   - Allow graceful fallbacks when timeouts occur

3. **Add early returns**:
   - If gaps > 2 in first message: Skip principle detection, go straight to gap clarification
   - If greeting detected + first message: Skip deep analysis, respond to greeting first

### Medium-Term (Workflow Changes)

1. **Fix workflow priority**:
   - HIGH-CONFIDENCE INTENTS (0.85+) should override gap clarification
   - Greetings should ALWAYS be acknowledged, then follow up with gaps
   - Pattern: "Hello! [greeting response] To help better, [gap question]"

2. **Consolidate LLM analysis**:
   - Have ConversationAgent request single LLM analysis instead of 5+ separate calls
   - LLM returns: {clarity, principles, subject_shift, response, learned} in one call
   - Cache results for downstream reuse

3. **Implement reserved contact for Moly**:
   - System_moly contact in database
   - Track greetings, tone, frequency as relationship metadata
   - Enable true self-awareness of user-system relationship

### Long-Term (Architectural Redesign)

1. **Unify LLM analysis**:
   - Single "AnalysisEngine" that runs ALL semantic analysis
   - Returns structured result with all needed data
   - Caches results for reuse across phases

2. **Phase-based response generation**:
   - Separate workflow from intent detection
   - Intent detection determines "what to respond to"
   - Workflow determines "how much to respond" (greeting vs. greeting + follow-up vs. greeting + analysis)

3. **Maturity-aware workflow**:
   - First message: Greeting → Acknowledge → Single gap question
   - Second-third messages: Gather context (multiple gaps)
   - Fourth+ messages: Deep analysis + Socratic deepening
   - Not "gaps > 2" threshold, but phase-based progression

4. **Reserved contacts system**:
   - All entities (people, topics, system) become contacts
   - Tracking becomes unified (relationships, metadata, history)
   - Self-awareness is just another contact relationship

---

## PART 8: DATA STRUCTURE UNDERSTANDING

### AnalysisContext Structure

**File**: `models/agent_types.go` lines ~80-110

```go
type AnalysisContext struct {
    Gaps                    []string               // Missing context fields
    ContextQuality          string                 // "minimal", "partial", "comprehensive"
    IsFirstMessageInConversation bool
    UserProfile            *AboutMe
    ConversationHistory    []HistoryMessage
    ConfirmedPreferences   []ContextAttribute
    ExtractedContext       *ExtractedContext
    // ... other fields
}
```

**What Gaps Contains**:
- List of missing fields: "communicationStyle", "coreValues", "contact", "conversationHistory", "userBehaviorProfile", "relevantReflections", "pastIntention", "recentSafetyIncidents"

**Context Quality Levels**:
- "minimal" - 0-3 fields loaded (gaps > 5)
- "partial" - 4+ fields loaded (2 < gaps ≤ 5)
- "comprehensive" - 6+ fields loaded (gaps ≤ 2)

### ExtractedContext Structure

```go
type ExtractedContext struct {
    Contact    *ExtractedContact
    Style      *ExtractedStyle
    Intention  string
    Goals      []string
}

type ExtractedContact struct {
    Name           string
    Relationship   string
    Confidence     float64
    Evidence       string
}

type ExtractedEntity struct {
    Value                 string
    Type                  string // "self_reference", "contact", "topic", "goal", "ambiguous"
    Confidence            float64
    IsAmbiguous           bool
    AmbiguousPossibilities []string
}
```

---

## SUMMARY TABLE: All Identified Issues

| Issue | Location | Severity | Impact | Root Cause |
|-------|----------|----------|--------|-----------|
| Workflow overrides intent | agents/conversation_agent.go:1240-1260 | CRITICAL | Greetings not acknowledged | Gaps priority > intent priority |
| Cascading LLM calls | main.go + agents/ (9 locations) | HIGH | 10-15 LLM calls per message | No analysis consolidation |
| Short timeouts | ollama.go:27,41,92 | HIGH | Timeouts on old hardware | Hardcoded values too aggressive |
| Self-awareness gap | No reserved Moly contact | MEDIUM | System doesn't track its own relationship with user | No system_moly contact in DB |
| Gap detection threshold | main.go:1311 | MEDIUM | Gaps > 2 always triggers on first message | Threshold not phase-aware |
| Redundant clarity check | conversation_agent.go:560 | MEDIUM | Same clarity analyzed twice | No result sharing |

---

## CONCLUSION

Moly's self-awareness failure is not a single bug but a **compounding architectural issue**:

1. **Greeting IS detected correctly** (IntentDetector works fine)
2. **But workflow selection overrides it** (gaps > 2 → ask gap clarification)
3. **Which triggers timeout cascades** (too many LLM calls, too-short timeouts)
4. **Which causes skipped analyses** (principle detection times out, falls back)
5. **Which leaves Moly unable to recognize itself** (no reserved contact for self-reference tracking)

The system is **architecturally sound** (11-layer design is correct) but **operationally fragile** (workflow priority, LLM consolidation, timeout tuning).

**Key insight**: The system isn't broken, it's **over-engineered for slow hardware**. Running 10-15 LLM calls on a 2-minute-per-call model, with 10-30 second timeouts, guarantees failures. Either reduce LLM calls, increase timeouts, or both.

