# MOLY: GREENFIELD ARCHITECTURE DESIGN

## CORE INSIGHT

Moly's job: **Transform messy human input into structured understanding, ask clarifying questions, detect inconsistencies, respond helpfully, and learn.**

The core pattern: **Request → Build Context → Handle Pending Input → Generate Response → Save Learnings**

---

## PART 1: DATA MODEL (Normalized, Query-Friendly)

### Core Entities

```
User
├── id
├── created_at
└── metadata (JSON: preferences, flags)

Context (cached snapshot of user's understanding at a point in time)
├── id
├── user_id
├── conversation_id
├── created_at
├── state: {
│   ├── aboutMe: {style, values, tone, goals}
│   ├── contacts: [{name, relationship, traits, confidence}]
│   ├── pastIntention: string
│   └── timestamp: when this snapshot was created
│   }

Conversation
├── id
├── user_id
├── created_at
├── updated_at

Message
├── id
├── conversation_id
├── role: "user" | "assistant"
├── content
├── extracted_context: {contact, style, intention, goals}
├── metadata: {stage_reached, pending_input_type}
├── created_at

PendingInput (UNIFIED - replaces clarifications + conflicts + approvals)
├── id
├── user_id
├── conversation_id
├── type: "clarification" | "conflict" | "approval" | "confirmation"
├── subtype: "style_conflict" | "intention_conflict" | "reflection_approval"
├── question: what to ask user
├── expected_answer_type: "text" | "choice" | "yes_no"
├── context: JSON with full details (old_value, new_value, etc)
├── created_at
├── resolved_at: null until user answers
├── resolution: how user answered
├── applied: whether decision was applied to user model
├── metadata: {confidence, source}

Insight (replaces Reflection - broader concept)
├── id
├── user_id
├── conversation_id
├── type: "characteristic" | "goal" | "value" | "communication_style" | "relationship"
├── value: what we learned
├── confidence: 0-1
├── source_message_id
├── status: "extracted" | "pending_approval" | "approved" | "rejected"
├── created_at
├── approved_at: null until user approves
```

### Indexes

```
-- Query patterns to optimize for:
Conversation(user_id, updated_at DESC) -- Load recent conversation
PendingInput(user_id, resolved_at IS NULL) -- Check pending
Insight(conversation_id, status) -- Load pending approvals
Message(conversation_id, created_at DESC) -- Load history
```

### Why This Design

✅ **No data duplication** — Each fact stored once
✅ **Clear relationships** — Foreign keys, not embedded documents
✅ **Queryable** — Can batch queries efficiently
✅ **Extensible** — New question types just add to `type` enum
✅ **Auditable** — Timestamps, source references everywhere

---

## PART 2: REQUEST PIPELINE (4 Stages, Not 8)

```
┌─────────────────────────────────────────────────────┐
│ STAGE 1: VALIDATE & BUILD CONTEXT                   │
├─────────────────────────────────────────────────────┤
│ Input: HTTP request                                 │
│ Actions:                                            │
│   1. Validate token → get user_id                   │
│   2. Load conversation (or create)                  │
│   3. Single query: load all user context:           │
│      - About Me (style, values, goals)              │
│      - Recent messages (10)                         │
│      - Recent insights (5)                          │
│      - Pending inputs (all unresolved)              │
│   4. Parse request → message object                 │
│ Output: (userID, conversation, context, message)    │
└─────────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────────┐
│ STAGE 2: HANDLE PENDING INPUT                       │
├─────────────────────────────────────────────────────┤
│ Input: pending_inputs from context, user message    │
│ Actions:                                            │
│   If pending_inputs.length > 0:                     │
│     1. Check if message answers first pending       │
│     2. If yes:                                      │
│        - Parse answer                              │
│        - Apply resolution (save to user model)      │
│        - Mark as resolved                           │
│        - Continue to Stage 3                        │
│     3. If no:                                       │
│        - Log (unusual: user ignores question)       │
│        - Continue to Stage 3 anyway                 │
│ Output: updated_context (with resolved conflicts)   │
│         message (unchanged)                         │
└─────────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────────┐
│ STAGE 3: GENERATE RESPONSE                          │
├─────────────────────────────────────────────────────┤
│ Input: updated_context, message                     │
│ Actions:                                            │
│   1. Extract context from message (LLM): 1 call     │
│      → contact, style, intention, goals             │
│   2. Detect conflicts with stored context           │
│   3. Generate appropriate response:                 │
│      a. If conflicts: ask conflict question         │
│      b. Else if clarification needed: ask question  │
│      c. Else: generate Socratic response            │
│   4. Run ethical analysis (1 LLM call)              │
│      → check for harmful content                    │
│   5. Extract insights (1 LLM call)                  │
│      → what we learned about user                   │
│ Output: response_object {                           │
│           text: string,                             │
│           type: "response" | "question",            │
│           metadata: {ethical?, insights[]},         │
│           pending_input: {type, question}?          │
│         }                                           │
│ LLM Calls: 3 total (extract + ethics + insights)   │
└─────────────────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────────────────┐
│ STAGE 4: SAVE & RESPOND                             │
├─────────────────────────────────────────────────────┤
│ Input: message, response_object, insights           │
│ Actions:                                            │
│   1. Batch insert:                                  │
│      - Save message                                 │
│      - Save response                                │
│      - Save insights (if any)                       │
│      - Save pending input (if conflict/clarif)      │
│      - Update user context snapshot                 │
│   2. Return response to frontend                    │
│ DB Calls: 1 transaction (5 inserts/updates)        │
│ Output: HTTP response                               │
└─────────────────────────────────────────────────────┘
```

### Why 4 Stages

- **Clear separation** — each stage has one job
- **Testable** — can mock each stage independently
- **Efficient** — minimal database round-trips
- **Understandable** — flow is obvious to newcomers
- **Extensible** — new logic goes into right stage

---

## PART 3: CODE ARCHITECTURE (Layered, Testable)

```
handlers/
├── message_handler.go          -- HTTP layer only (validation, routing)
└── types.go                    -- Request/Response DTOs

orchestration/
├── message_pipeline.go         -- Stages 1-4 orchestration
├── pending_input_handler.go    -- Detects & applies pending answers
└── response_generator.go       -- Stages 3a-3c decision logic

context/
├── context_loader.go           -- Load user context (1 query)
├── context_builder.go          -- Build Context object from data
└── types.go                    -- Context struct

extraction/
├── message_extractor.go        -- Extract contact, style, intention, goals
├── conflict_detector.go        -- Find inconsistencies
└── insight_extractor.go        -- Learn new facts

generation/
├── response_generator.go       -- Generate natural responses
├── question_generator.go       -- Generate Socratic/clarification Qs
├── ethical_gate.go             -- Apply ethical constraints
└── suggestion_generator.go     -- Generate message suggestions

storage/
├── message_store.go            -- Save messages
├── context_store.go            -- Save context snapshots
├── pending_input_store.go      -- Save pending inputs
├── insight_store.go            -- Save insights
└── batch_writer.go             -- Batch inserts in transaction

llm/
├── extractor_client.go         -- LLM extraction calls
├── ethics_client.go            -- LLM ethical analysis
├── insight_client.go           -- LLM insight extraction
└── response_client.go          -- LLM response generation

models/
├── user.go
├── conversation.go
├── message.go
├── context.go
├── pending_input.go            -- UNIFIED (clarification + conflict + approval)
├── insight.go
└── response.go
```

### Key Pattern: Dependency Injection

```go
type MessagePipeline struct {
  loader        *ContextLoader
  conflictDetector *ConflictDetector
  generator     *ResponseGenerator
  extractor     *MessageExtractor
  store         *BatchWriter
  llm           LLMClient
}

// Easy to test: swap LLMClient with mock
```

---

## PART 4: UNIFIED PENDING INPUT SYSTEM

### The Big Insight

**Stop treating clarifications, conflicts, and approvals as separate systems.**

They're all the same pattern:
1. Ask user something
2. User answers
3. Apply the answer to their model
4. Move forward

### Single Table: PendingInput

```sql
CREATE TABLE pending_input (
  id BIGINT PRIMARY KEY,
  user_id BIGINT,
  conversation_id BIGINT,
  type TEXT, -- "clarification" | "conflict" | "approval"
  subtype TEXT, -- "style_conflict" | "reflection_approval" etc
  question TEXT,
  context JSONB, -- {old_value, new_value, reasoning, source_msg_id}
  created_at TIMESTAMP,
  resolved_at TIMESTAMP,
  resolution TEXT, -- what user answered
  applied BOOLEAN, -- whether we applied it to user model
  metadata JSONB
);
```

### Single Handler

```go
type PendingInputHandler struct {
  store         PendingInputStore
  conflictResolver *ConflictResolver
  clarificationApplier *ClarificationApplier
  approvalProcessor *ApprovalProcessor
}

func (p *PendingInputHandler) HandleAnswer(pending PendingInput, userAnswer string) {
  switch pending.Type {
  case "conflict":
    resolution := parseConflictAnswer(userAnswer, pending.Context)
    p.conflictResolver.Apply(pending, resolution)
  case "clarification":
    p.clarificationApplier.Apply(pending, userAnswer)
  case "approval":
    p.approvalProcessor.Apply(pending, userAnswer)
  }
}
```

### Why This Works

✅ **Single code path** — all pending inputs handled same way
✅ **No duplication** — one storage table, one retrieval, one handler
✅ **Easy to extend** — new input type just adds a case
✅ **Clear state machine** — created → resolved → applied

---

## PART 5: CONTEXT LOADING STRATEGY

### Single Batch Query

Instead of:
```
Load conversation (1 query)
Load AboutMe (1 query)
Load reflections (1 query)
Load contacts (1 query)
Load user profile (1 query)
...
```

One efficient query:

```sql
WITH user_context AS (
  SELECT * FROM about_me WHERE user_id = ?
),
recent_messages AS (
  SELECT * FROM messages 
  WHERE conversation_id = ? 
  ORDER BY created_at DESC 
  LIMIT 10
),
pending AS (
  SELECT * FROM pending_input 
  WHERE user_id = ? AND resolved_at IS NULL
),
insights AS (
  SELECT * FROM insights 
  WHERE conversation_id = ? AND status IN ('approved', 'pending_approval')
  ORDER BY created_at DESC 
  LIMIT 5
)
SELECT 
  (SELECT * FROM user_context) as about_me,
  (SELECT * FROM recent_messages) as messages,
  (SELECT * FROM pending) as pending_inputs,
  (SELECT * FROM insights) as insights
```

Or: 4 separate queries in parallel (HTTP2, connection pooling)

### Result

**1-4 database queries** instead of 20+

---

## PART 6: LLM CALL OPTIMIZATION

### Current (7 calls)
1. Extract context
2. Safety check
3. Risk assessment  
4. Socratic selection
5. Response generation
6. Reflection extraction
7. Ethical analysis

### Optimized (3 calls)

**Call 1: Context Extraction** (temperature 0.3)
```
Extract: contact, style, intention, goals from: "..."
Also check: does this create conflicts with stored values?
Return: structured extraction + conflict flags
```

**Call 2: Response Generation** (temperature 0.7)
```
You are Moly. Context: {...style, about_me, conversation_history...}
If conflicts: ask about them.
Else if needs clarification: ask.
Else: respond with Socratic question or validation.
Generate natural response.
```

**Call 3: Insight Extraction** (temperature 0.5)
```
What did we learn about this user from: "..."
Extract: new characteristics, goals, values, communication patterns
Return: structured insights
```

**Safety/Ethics**: Pattern matching + heuristics, not LLM (too slow)

### Result

**3 LLM calls** instead of 7

---

## PART 7: FLOW COMPARISON

### Current (Messy)
```
Request
  ↓ Safety check (LLM)
  ↓ Risk assessment (LLM)
  ↓ Context extraction (LLM)
  ↓ Load conversation
  ↓ Load AboutMe
  ↓ Load reflections
  ↓ Load contacts
  ↓ Load profile
  ↓ → ConversationAgent (does extraction AGAIN, safety AGAIN)
  ↓    ├→ Safety check (LLM) [REDUNDANT]
  ├→ Generate response (LLM)
  ├→ Extract insights (LLM)
  ├→ Ethical analysis (LLM)
  ├→ Detect conflicts
  ├→ Check intention conflicts
  ├→ Save message
  ├→ Save response
  ├→ Save contact
  ├→ Save style
  ├→ Save intention
  ├→ Save goals
  ├→ Save reflection
  ↓ Response

DB Queries: 20-25
LLM Calls: 7
Code Complexity: ⭐⭐⭐⭐⭐
Testability: ⭐
```

### New (Clean)
```
Request
  ↓ [STAGE 1] Validate + Load Context (1-4 DB queries in parallel)
  ↓ [STAGE 2] Handle Pending Input (if exists)
  ├→ Check if message answers pending
  ├→ Parse answer
  ├→ Apply resolution
  ↓ [STAGE 3] Generate Response (3 LLM calls)
  ├→ Extract context (1 LLM) + detect conflicts
  ├→ Generate response (1 LLM)
  ├→ Extract insights (1 LLM)
  ↓ [STAGE 4] Save All (1 batch transaction)
  ├→ Save message
  ├→ Save response
  ├→ Save insights
  ├→ Save pending input (if conflict/clarification)
  ├→ Update context snapshot
  ↓ Response

DB Queries: 4-10
LLM Calls: 3
Code Complexity: ⭐
Testability: ⭐⭐⭐⭐⭐
```

---

## PART 8: TESTING STRATEGY

### Unit Tests (Easy)

```go
// Test conflict detection
func TestConflictDetector_StyleConflict(t *testing.T) {
  detector := NewConflictDetector()
  conflict := detector.Detect(
    storedStyle: "casual",
    extractedStyle: "formal",
    context: "work context"
  )
  assert.Equal(t, conflict.Type, "style_conflict")
}

// Test answer parsing
func TestPendingInputHandler_ParseConflictAnswer(t *testing.T) {
  handler := NewPendingInputHandler()
  resolution := handler.ParseAnswer(
    pending: {type: "conflict", context: {...}},
    userAnswer: "both are true for different situations"
  )
  assert.Equal(t, resolution, "merge")
}

// Test response generation
func TestResponseGenerator_ConflictResponse(t *testing.T) {
  gen := NewResponseGenerator(mockLLM)
  response := gen.GenerateResponse(
    context: {...},
    hasConflict: true
  )
  assert.Contains(t, response, "different situations")
}
```

### Integration Tests

```go
// Test full pipeline with real DB but mock LLM
func TestMessagePipeline_FullFlow(t *testing.T) {
  pipeline := NewMessagePipeline(realDB, mockLLM)
  response := pipeline.ProcessMessage(
    userID: "user1",
    message: "I want to message my boss formally but I'm casual normally"
  )
  assert.True(t, response.HasConflict)
  assert.Contains(t, response.PendingInput.Question, "different")
}
```

### Each Stage Independently Testable ✅

---

## PART 9: DEPLOYMENT & SCALING

### Current Pain Points
- Complex state makes debugging hard
- Slow due to redundant queries
- Hard to add features without breaking things

### New Advantages
- **Simple state machine** — easy to debug
- **Fast** — 4 queries, 3 LLM calls
- **Testable** — each stage can be tested alone
- **Scalable** — clear where bottlenecks are
- **Maintainable** — new team member understands flow in 1 day

### Performance Profile

```
Typical message: 300-500ms
├─ Stage 1 (Context): 100-150ms (DB I/O)
├─ Stage 2 (Pending): 10-20ms (conflict checking)
├─ Stage 3 (Generate): 150-250ms (3 LLM calls)
└─ Stage 4 (Save): 30-50ms (DB transaction)

Bottleneck: LLM latency (Stage 3)
Solution: Stream responses to frontend, save when complete
```

---

## PART 10: SUMMARY TABLE

| Aspect | Current | From Scratch |
|--------|---------|--------------|
| **Stages** | 8 | 4 |
| **DB Queries** | 20-25 | 4-10 |
| **LLM Calls** | 7 | 3 |
| **Handler LOC** | 1400+ | 300-400 |
| **Testability** | Poor | Excellent |
| **Maintainability** | Poor | Excellent |
| **Complexity** | High | Low |
| **Extensibility** | Hard | Easy |
| **Pending Input Systems** | 3 (separate) | 1 (unified) |
| **Data Duplication** | Yes | No |

---

## PART 11: MIGRATION PATH

If redesigning from scratch isn't an option, migrate incrementally:

1. **Week 1**: Add PendingInput table, migrate conflicts to use it
2. **Week 2**: Migrate clarifications to PendingInput
3. **Week 3**: Unify handlers into single ProcessPendingInput
4. **Week 4**: Consolidate context loading into single query
5. **Week 5**: Extract pipeline stages from monolithic handler
6. **Week 6**: Optimize LLM calls (combine 3 calls into single smarter call)

Each week ships independently, no breaking changes.

---

## KEY DECISIONS

### Why This Design Works

✅ **Single Unified System** — one flow for all user input types
✅ **Clear State Machine** — even non-technical people understand it
✅ **Minimal Database Queries** — batch 4-10 instead of 20-25
✅ **Fewer LLM Calls** — 3 smart calls instead of 7 redundant ones
✅ **Testable** — each stage is independently testable
✅ **Maintainable** — code is clear, flow is obvious
✅ **Extensible** — adding new conflict types = 1 new enum value
✅ **Scalable** — clear where bottlenecks are (LLM latency)

### The Philosophy

**Flatten the complexity.** Every element serves a clear purpose. No redundancy. No ambiguity. Just: load context → handle pending → respond → save.

---

## CONCLUSION

**If I were building Moly from scratch today, I'd make it 60% smaller, 50% faster, and 10x more maintainable.**

The current system works, but it's evolved organically and accumulated complexity. A fresh design would make everything clearer, faster, and easier to extend.

The good news: you can migrate to this design incrementally without breaking the current system.
