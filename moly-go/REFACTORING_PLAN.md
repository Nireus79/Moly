# MOLY GREENFIELD REFACTOR: DETAILED IMPLEMENTATION PLAN

**Scope**: Complete rewrite of message pipeline, unified pending input system, optimized LLM/DB usage  
**Duration**: ~24-31 hours of implementation  
**Risk Mitigation**: Precise function mapping, no dead code, staged verification  
**Status**: ⏳ READY TO EXECUTE

---

## PHASE 0: PREPARATION & VERIFICATION

### Step 0.1: Current Code Inventory

**Files to be refactored:**
```
main.go (MessageProcessorHandler - 1400+ lines)
├─ Lines 309-1700: Complete handler - WILL BE REPLACED
agents/conversation_agent.go (900+ lines)
├─ ConversationAgent.Run() - WILL BE REPLACED with ResponseGenerator
├─ Helper methods - WILL BE MOVED to extraction/
tools/inline_conflict_resolver.go
├─ ParseResolutionFromResponse() - WILL BE MOVED to pending_input/
├─ ApplyConflictResolution() - WILL BE MOVED to pending_input/
database/repositories.go
├─ ContextConflictRepository - WILL BE REPLACED with unified PendingInputRepository
├─ TemporaryFactStore - WILL BE REPLACED with PendingInputRepository
├─ ReflectionRepository - WILL BE ENHANCED (add Reject method, fix queries)
```

**Files to be created:**
```
orchestration/
├─ pipeline.go (4-stage orchestrator)
├─ pending_input_handler.go (unified handler)
├─ context_builder.go (build context from loaded data)

extraction/
├─ message_extractor.go (LLM extraction: contact, style, intention, goals)
├─ conflict_detector.go (find inconsistencies)

generation/
├─ response_orchestrator.go (stage 3 decision logic)
├─ suggestion_generator.go (MOVED from tools/, renamed)

storage/
├─ batch_writer.go (transactional saves)

database/
├─ pending_input_repository.go (UNIFIED: clarifications + conflicts + approvals)
```

### Step 0.2: Database Schema Changes

**New table (PendingInput - replaces 3 systems):**
```sql
CREATE TABLE pending_input (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  conversation_id BIGINT NOT NULL,
  type TEXT NOT NULL CHECK(type IN ('clarification', 'conflict', 'approval')),
  subtype TEXT, -- 'style_conflict', 'intention_conflict', 'reflection_approval', etc
  question TEXT NOT NULL,
  context JSONB NOT NULL, -- {old_value, new_value, reasoning, source_msg_id}
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  resolved_at TIMESTAMP,
  resolution TEXT,
  applied BOOLEAN DEFAULT FALSE,
  metadata JSONB,
  
  INDEX idx_user_pending (user_id, resolved_at),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);
```

**Indexes to add:**
```sql
CREATE INDEX idx_context_user_pending ON pending_input(user_id, resolved_at);
CREATE INDEX idx_conversation_pending ON pending_input(conversation_id, type);
```

**Deprecated tables (can be kept as archive for now, deleted in cleanup phase):**
```
temporary_fact_store → MIGRATE data to pending_input, then DELETE
context_conflicts → MIGRATE data to pending_input, then DELETE
```

### Step 0.3: Existing Function Mapping

| Current Location | Current Name | New Location | New Name | Action |
|------------------|--------------|--------------|----------|--------|
| main.go:309-1700 | MessageProcessorHandler | orchestration/pipeline.go | MessagePipeline.Process | REPLACE |
| agents/conversation_agent.go:X | ConversationAgent.Run | generation/response_orchestrator.go | ResponseOrchestrator.Generate | REPLACE |
| agents/conversation_agent.go:740+ | buildAdaptiveSystemPrompt | extraction/message_extractor.go | buildAdaptiveSystemPrompt | MOVE |
| agents/conversation_agent.go:770+ | detectEmotionalTone | extraction/message_extractor.go | detectEmotionalTone | MOVE |
| agents/conversation_agent.go:800+ | detectTopic | extraction/message_extractor.go | detectTopic | MOVE |
| tools/inline_conflict_resolver.go:141 | ParseResolutionFromResponse | orchestration/pending_input_handler.go | parseAnswer | MOVE & REFACTOR |
| tools/inline_conflict_resolver.go:190 | ApplyConflictResolution | orchestration/pending_input_handler.go | applyResolution | MOVE & REFACTOR |
| tools/suggestion_generator.go | SuggestionGenerator | generation/suggestion_generator.go | SuggestionGenerator | MOVE (no change) |
| tools/harm_analyzer.go | HarmAnalyzer | generation/harm_analyzer.go | HarmAnalyzer | MOVE (no change) |
| database/repositories.go:218+ | ReflectionRepository | database/reflection_repository.go | ReflectionRepository | ENHANCE (add Reject, fix queries) |
| database/repositories.go:xxx | TemporaryFactStore | database/pending_input_repository.go | PendingInputRepository (clarification type) | REPLACE |
| database/repositories.go:xxx | ContextConflictRepository | database/pending_input_repository.go | PendingInputRepository (conflict type) | REPLACE |

---

## PHASE 1: CREATE NEW INFRASTRUCTURE (4 hours)

### Step 1.1: Create Database Layer (`database/pending_input_repository.go`)

**Signature:**
```go
type PendingInputRepository struct {
  db *sql.DB
}

type PendingInput struct {
  ID            int64
  UserID        int64
  ConversationID int64
  Type          string // "clarification" | "conflict" | "approval"
  Subtype       string
  Question      string
  Context       json.RawMessage
  CreatedAt     time.Time
  ResolvedAt    *time.Time
  Resolution    *string
  Applied       bool
  Metadata      json.RawMessage
}

// Methods needed:
func (r *PendingInputRepository) Create(ctx context.Context, pi *PendingInput) error
func (r *PendingInputRepository) GetUnresolved(ctx context.Context, userID int64) ([]PendingInput, error)
func (r *PendingInputRepository) GetByID(ctx context.Context, id int64) (*PendingInput, error)
func (r *PendingInputRepository) Resolve(ctx context.Context, id int64, resolution string) error
func (r *PendingInputRepository) MarkApplied(ctx context.Context, id int64) error
func (r *PendingInputRepository) GetByConversation(ctx context.Context, conversationID int64) ([]PendingInput, error)
```

**Key points:**
- Single table, type field determines subtype
- No NULL query bugs (all fields correctly typed)
- Can query: all pending for user, all for conversation, by specific ID
- Applied flag tracks whether resolution was written to user model

### Step 1.2: Create Context Data Layer (`models/context.go` - new/enhanced)

**Structure:**
```go
type UserContext struct {
  AboutMe struct {
    Style            string
    Values           []string
    Tone             string
    Goals            []string
    CommunicationPreferences string
  }
  Contacts []Contact
  RecentMessages []Message
  PendingInputs []PendingInput
  RecentInsights []Insight
  CreatedAt time.Time
}

// Load function signature:
func LoadUserContext(db *sql.DB, userID int64, conversationID int64) (*UserContext, error)
```

**Implementation note:**
- Single function that loads ALL context in 3-4 parallel queries
- Returns comprehensive snapshot
- Used by all stages

### Step 1.3: Create Batch Storage Layer (`storage/batch_writer.go`)

**Signature:**
```go
type BatchWriter struct {
  db *sql.DB
}

type BatchWriteRequest struct {
  Message         *Message
  Response        *Response
  Insights        []Insight
  PendingInput    *PendingInput
  UpdatedAboutMe  *AboutMe
}

func (bw *BatchWriter) WriteBatch(ctx context.Context, req BatchWriteRequest) error
```

**Implementation:**
- Single transaction with multiple inserts/updates
- All-or-nothing: if any fails, rollback all
- Returns error clearly identifying which write failed
- Logs transaction ID for tracing

---

## PHASE 2: BUILD PIPELINE STAGES (8 hours)

### Step 2.1: Stage 1 - Validation & Context Loading (`orchestration/context_loader.go`)

**Input:** HTTP request with token, conversation_id, message  
**Output:** (userID, conversation, userContext, message)  
**Errors:** validation error, context load error, DB error

**Pseudocode:**
```go
func (p *Pipeline) Stage1_ValidateAndLoad(req HTTPRequest) (*PipelineState, error) {
  // 1. Extract & validate token → get userID
  userID, err := validateToken(req.AuthHeader)
  
  // 2. Validate JSON body
  message := req.Message
  if message == "" { return error }
  
  // 3. Load/create conversation
  conversation, err := getOrCreateConversation(userID, 30 * time.Day)
  
  // 4. Load complete user context (1 call)
  context, err := LoadUserContext(db, userID, conversation.ID)
  
  // 5. Parse message
  parsedMessage := Message{
    ConversationID: conversation.ID,
    Role: "user",
    Content: message,
    CreatedAt: time.Now(),
  }
  
  return &PipelineState{
    UserID: userID,
    Conversation: conversation,
    Context: context,
    Message: parsedMessage,
  }, nil
}
```

**Database calls:** 3-4 parallel queries (conversation + about_me + recent messages + pending inputs)  
**Error handling:** Early return on validation/DB failures

### Step 2.2: Stage 2 - Handle Pending Input (`orchestration/pending_input_handler.go`)

**Input:** state from Stage 1  
**Output:** updated state (with resolved pending input marked), OR early return if no match  
**Logic:** Check if user message answers any pending question

**Pseudocode:**
```go
func (p *Pipeline) Stage2_HandlePendingInput(state *PipelineState) (*PipelineState, error) {
  pending := state.Context.PendingInputs
  
  if len(pending) == 0 {
    return state, nil // No pending input, continue
  }
  
  firstPending := pending[0]
  
  // Try to parse answer from user message
  answer := parseAnswerFromMessage(state.Message.Content, firstPending)
  
  if answer == "" {
    // User didn't answer the pending question
    // Continue processing message anyway (unusual but valid)
    return state, nil
  }
  
  // User answered! Process resolution
  resolution := applyResolution(firstPending, answer)
  
  // Update database
  err := repo.Resolve(firstPending.ID, resolution)
  if err != nil { return error }
  
  err = repo.MarkApplied(firstPending.ID)
  if err != nil { return error }
  
  // Remove from context so Stage 3 doesn't re-detect
  state.ResolvedPendingID = firstPending.ID
  state.Context.PendingInputs = state.Context.PendingInputs[1:]
  
  // Continue with normal flow (message might also have new content)
  return state, nil
}

func parseAnswerFromMessage(msg string, pending PendingInput) string {
  switch pending.Type {
  case "conflict":
    // Check if msg contains markers: "both", "different situations", "preference changed"
    if contains(msg, ["both", "different"]) { return "merge" }
    if contains(msg, ["preference", "changed"]) { return "update" }
  case "clarification":
    // Message itself IS the answer
    return msg
  case "approval":
    // Check for yes/no
    if contains(msg, ["yes", "approve", "true"]) { return "approve" }
    if contains(msg, ["no", "reject", "false"]) { return "reject" }
  }
  return ""
}

func applyResolution(pending PendingInput, resolution string) string {
  // Parse context JSON, update user model, return formatted resolution
  var ctx map[string]interface{}
  json.Unmarshal(pending.Context, &ctx)
  
  // Call appropriate resolver:
  // - Conflict: update about_me with merged values
  // - Clarification: update about_me with new fact
  // - Approval: update reflection to approved/rejected
  
  return resolution
}
```

**Database calls:** 2 (Resolve + MarkApplied in transaction)  
**Key safeguard:** Removes resolved pending from state before Stage 3 to prevent re-detection

### Step 2.3: Stage 3 - Generate Response (`generation/response_orchestrator.go`)

**Input:** state from Stage 2  
**Output:** response object {text, type, metadata, pending_input?}

**Decision tree:**
```
if state.Context.PendingInputs still has items:
  → Generate conflict question (user didn't answer yet)
  → Add to response.PendingInput
else if clarification needed:
  → Generate clarification question
  → Add to response.PendingInput
else:
  → Generate Socratic response
  → Extract insights from message
  → No pending input to add
```

**Implementation detail - LLM calls (3 total):**

**Call 1: Extract context from message** (temperature 0.3)
```
Extract from message: contact, communication_style, intention, goals
Also identify: emotional_tone, topic, any value conflicts
Return: structured JSON
```

**Call 2: Generate response** (temperature 0.7)
```
You are Moly. User context: [about_me, recent messages, emotional state]
If conflicts pending: ask about them
Else if needs clarification: ask
Else: respond with validation + Socratic question
Generate natural response.
```

**Call 3: Extract insights** (temperature 0.5)
```
What did we learn from this message about the user?
Extract: new characteristics, goals, values, communication patterns, relationship insights
Return: structured insights for storage
```

**Safety/Ethics:** Pattern matching (no LLM call) for speed

**Pseudocode:**
```go
func (p *Pipeline) Stage3_GenerateResponse(state *PipelineState) (*PipelineState, error) {
  // Extract context from message (LLM Call 1)
  extraction, err := extractContext(state.Message.Content, state.Context.AboutMe)
  
  // Detect conflicts
  conflicts := detectConflicts(extraction, state.Context.AboutMe)
  
  var response *Response
  var pendingInput *PendingInput
  
  if len(conflicts) > 0 {
    // Generate conflict question
    question := generateConflictQuestion(conflicts[0])
    response = &Response{Text: question, Type: "question"}
    pendingInput = &PendingInput{
      Type: "conflict",
      Question: question,
      Context: marshalConflict(conflicts[0]),
    }
  } else if needsClarification(extraction, state.Context.AboutMe) {
    // Generate clarification question
    question := generateClarificationQuestion(extraction)
    response = &Response{Text: question, Type: "question"}
    pendingInput = &PendingInput{
      Type: "clarification",
      Question: question,
      Context: marshalClarification(extraction),
    }
  } else {
    // Generate full response (LLM Call 2)
    response, err = generateResponse(state.Message.Content, state.Context)
    if err != nil { return error }
  }
  
  // Extract insights (LLM Call 3)
  insights, err := extractInsights(state.Message.Content, state.Context)
  if err != nil { return error }
  
  // Run ethical gate (pattern matching, not LLM)
  ethical := checkEthics(response.Text)
  if ethical.Severity == "BLOCK" {
    response.Text = "I can't help with that."
    response.Metadata["ethicalIntervention"] = "blocked"
  } else if ethical.Severity == "WARN" {
    response.Metadata["ethicalIntervention"] = "warned"
    response.Metadata["ethicalReason"] = ethical.Reason
  }
  
  state.Response = response
  state.PendingInput = pendingInput
  state.Insights = insights
  
  return state, nil
}
```

**Database calls:** 0 (all read from state.Context loaded in Stage 1)

### Step 2.4: Stage 4 - Save & Return (`storage/batch_writer.go`)

**Input:** state from Stage 3  
**Output:** HTTP response  
**Action:** Save everything in single transaction

**Pseudocode:**
```go
func (p *Pipeline) Stage4_SaveAndRespond(state *PipelineState) (HTTPResponse, error) {
  batchReq := BatchWriteRequest{
    Message: &state.Message,
    Response: &state.Response,
    Insights: state.Insights,
    PendingInput: state.PendingInput,
    UpdatedAboutMe: state.UpdatedAboutMe, // from Stage 2 resolution if any
  }
  
  err := p.storage.WriteBatch(batchReq)
  if err != nil { return error }
  
  return HTTPResponse{
    Response: state.Response.Text,
    Metadata: state.Response.Metadata,
    PendingInput: state.PendingInput,
  }, nil
}
```

**Database calls:** 1 transaction (5 inserts/updates)

---

## PHASE 3: IMPLEMENT EXTRACTION TOOLS (4 hours)

### Step 3.1: Message Extractor (`extraction/message_extractor.go`)

**Replaces:**
- agents/conversation_agent.go's extraction logic

**Interface:**
```go
type MessageExtractor struct {
  llm LLMClient
}

type ExtractedContext struct {
  Contact string
  Style string
  Intention string
  Goals []string
  EmotionalTone string
  Topic string
}

func (e *MessageExtractor) Extract(msg string, userAboutMe AboutMe) (*ExtractedContext, error)
```

**LLM Prompt (temperature 0.3):**
```
Extract from the user's message:
1. Contact: who are they messaging? (or "unknown" if not specified)
2. Style: formal or casual? friendly or serious?
3. Intention: what do they want to accomplish?
4. Goals: what goals does this relate to?
5. Emotional tone: very_negative, negative, neutral, positive, very_positive
6. Topic: work, relationships, family, mental_health, general

Also identify: any conflicts with stored values

Return JSON: {contact, style, intention, goals[], emotional_tone, topic, conflicts[]}
```

### Step 3.2: Conflict Detector (`extraction/conflict_detector.go`)

**Interface:**
```go
type ConflictDetector struct {}

type Conflict struct {
  Type string // "style_conflict" | "intention_conflict" | "value_conflict"
  Field string
  StoredValue string
  ExtractedValue string
  Context string
}

func (cd *ConflictDetector) Detect(extraction ExtractedContext, userAboutMe AboutMe) []Conflict
```

**Logic:**
```go
func (cd *ConflictDetector) Detect(...) []Conflict {
  var conflicts []Conflict
  
  // Check style conflict
  if extraction.Style != userAboutMe.Style && extraction.Style != "" {
    conflicts = append(conflicts, Conflict{
      Type: "style_conflict",
      Field: "style",
      StoredValue: userAboutMe.Style,
      ExtractedValue: extraction.Style,
    })
  }
  
  // Check intention conflict
  if extraction.Intention != "" && userAboutMe.PastIntention != "" {
    if extraction.Intention != userAboutMe.PastIntention {
      // Only if intentionally different, not just new
      conflicts = append(conflicts, Conflict{
        Type: "intention_conflict",
        Field: "intention",
        StoredValue: userAboutMe.PastIntention,
        ExtractedValue: extraction.Intention,
      })
    }
  }
  
  // Skip "intention_conflict" false positives (different message intents)
  conflicts = filterIntentionFalsePositives(conflicts)
  
  return conflicts
}
```

### Step 3.3: Response Generator (`generation/response_orchestrator.go` - partial)

**Methods needed:**
```go
func (rg *ResponseGenerator) GenerateConflictQuestion(conflict Conflict) string
func (rg *ResponseGenerator) GenerateClarificationQuestion(missing ExtractedContext) string
func (rg *ResponseGenerator) GenerateResponse(msg string, context UserContext) (*Response, error)
```

**Conflict question template:**
```
You mentioned "{extracted_value}" but earlier you said "{stored_value}".
Are both true for different situations, or has your preference changed?
```

**Clarification question template:**
```
You mentioned {topic}, but I'm not clear on {missing_detail}.
Could you clarify: {specific_question}?
```

**Response generation (LLM):**
```
You are Moly, a thoughtful conversational AI.
User communication style: {style}
User emotional state: {emotional_tone}
Context: {recent_messages, about_me}

Generate a response that:
1. Validates their feelings
2. Asks a Socratic question to deepen thinking
3. Respects their preferred communication style
4. Addresses the actual concern, not generic template responses
```

---

## PHASE 4: REFACTOR DATABASE ACCESS (4 hours)

### Step 4.1: Create Unified PendingInputRepository

**File:** `database/pending_input_repository.go`

**Replace:** TemporaryFactStore + ContextConflictRepository + parts of ReflectionRepository

**Methods:**
```go
// Create pending input (clarification, conflict, or approval)
func (r *PendingInputRepository) Create(ctx context.Context, pi *PendingInput) (int64, error)

// Get all unresolved for user (all types mixed)
func (r *PendingInputRepository) GetUnresolved(ctx context.Context, userID int64) ([]PendingInput, error)

// Get specific pending item
func (r *PendingInputRepository) GetByID(ctx context.Context, id int64) (*PendingInput, error)

// Get all pending for conversation
func (r *PendingInputRepository) GetByConversation(ctx context.Context, convID int64) ([]PendingInput, error)

// Mark as resolved with user's answer
func (r *PendingInputRepository) Resolve(ctx context.Context, id int64, resolution string) error

// Mark as applied to user model
func (r *PendingInputRepository) MarkApplied(ctx context.Context, id int64) error

// Get by type for specific handling
func (r *PendingInputRepository) GetByType(ctx context.Context, userID int64, type string) ([]PendingInput, error)
```

**Schema:**
```sql
CREATE TABLE pending_input (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  conversation_id BIGINT NOT NULL,
  type ENUM('clarification', 'conflict', 'approval') NOT NULL,
  subtype VARCHAR(255),
  question TEXT NOT NULL,
  context JSON NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  resolved_at TIMESTAMP NULL,
  resolution TEXT NULL,
  applied BOOLEAN DEFAULT FALSE,
  metadata JSON,
  
  INDEX idx_user_pending (user_id, resolved_at),
  INDEX idx_conversation_type (conversation_id, type),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);
```

### Step 4.2: Enhance ReflectionRepository

**File:** `database/reflection_repository.go`

**Changes:**
1. Fix GetPendingApprovals() to include ID and created_at in SELECT
2. Add Reject() method (mirror of Approve())
3. Ensure all queries use correct column names
4. Add prepared statements for performance

```go
func (r *ReflectionRepository) GetPendingApprovals(ctx context.Context, userID int64) ([]Reflection, error) {
  query := `
    SELECT id, user_id, conversation_id, type, value, confidence, 
           source_message_id, status, created_at, approved_at, rejected_at
    FROM reflections
    WHERE user_id = ? AND status = 'pending_approval'
    ORDER BY created_at DESC
  `
  // ... scan results correctly
}

func (r *ReflectionRepository) Reject(ctx context.Context, id int64) error {
  query := `
    UPDATE reflections
    SET status = 'rejected', rejected_at = NOW()
    WHERE id = ?
  `
  // ... execute
}
```

### Step 4.3: Context Loading Function (`models/context_loader.go`)

**Purpose:** Single function to load all user context

**Implementation:**
```go
func LoadUserContext(db *sql.DB, userID int64, conversationID int64) (*UserContext, error) {
  // Execute 3-4 queries in parallel (or serial if cleaner):
  
  // Query 1: AboutMe
  aboutMe, _ := getAboutMe(userID)
  
  // Query 2: Recent messages (last 10)
  messages, _ := getRecentMessages(conversationID, 10)
  
  // Query 3: Pending inputs (all unresolved)
  pending, _ := getPendingInputs(userID)
  
  // Query 4: Recent insights (last 5)
  insights, _ := getRecentInsights(conversationID, 5)
  
  return &UserContext{
    AboutMe: aboutMe,
    RecentMessages: messages,
    PendingInputs: pending,
    RecentInsights: insights,
    CreatedAt: time.Now(),
  }, nil
}
```

---

## PHASE 5: INTEGRATION & VERIFICATION (2 hours per stage)

### Step 5.1: Route Integration in main.go

**Replace:**
```go
// OLD (delete):
r.POST("/api/v2/message-processor", MessageProcessorHandler)
r.POST("/api/v2/clarification-response", ClarificationResponseHandler)
r.POST("/api/v2/conflict-resolve", ConflictResolveHandler)

// NEW (add):
r.POST("/api/v2/message-processor", NewPipeline(db, llm).ServeHTTP)
```

**Single entry point:** POST /api/v2/message-processor handles all three cases (clarification, conflict, normal)

### Step 5.2: Dependency Injection

**Build pipeline once on startup:**
```go
func init() {
  pipeline := &orchestration.MessagePipeline{
    loader: &extraction.ContextLoader{db: database},
    extractor: &extraction.MessageExtractor{llm: client},
    conflictDetector: &extraction.ConflictDetector{},
    generator: &generation.ResponseOrchestrator{llm: client},
    storage: &storage.BatchWriter{db: database},
  }
  
  // Use pipeline for all requests
}
```

### Step 5.3: Verification Checklist

**Stage 1 Verification:**
```
[ ] Token validation works
[ ] Conversation created if missing
[ ] UserContext loads without errors
[ ] No N+1 query issues
[ ] Verify exact queries: 4 (conversation + aboutme + messages + pending)
```

**Stage 2 Verification:**
```
[ ] Conflict answers parsed correctly
[ ] Clarification answers parsed correctly
[ ] Resolution applied to database
[ ] Pending marked as resolved
[ ] Resolved item removed from state before Stage 3
```

**Stage 3 Verification:**
```
[ ] Message extraction returns correct structure
[ ] Conflict detection catches real conflicts only
[ ] Response generated for non-conflict cases
[ ] LLM calls limited to 3 (extraction + response + insights)
[ ] Ethical gate prevents harmful responses
```

**Stage 4 Verification:**
```
[ ] All inserts execute in transaction
[ ] Rollback works if any insert fails
[ ] Message saved with correct fields
[ ] Response saved with metadata
[ ] Pending input saved if generated
[ ] Return JSON matches expected format
```

---

## PHASE 6: CLEANUP & DELETION (1 hour)

### Step 6.1: Delete Old Code

**Files to delete entirely:**
```
tools/inline_conflict_resolver.go (moved to orchestration/)
tools/clarification_asker.go (logic moved to generation/)
```

**Methods to delete from main.go:**
```
- ClarificationResponseHandler (logic moved to Stage 2)
- ConflictResolveHandler (logic moved to Stage 2)
- Stage 2-8 code in MessageProcessorHandler
- ConversationAgent.Run() is now ResponseGenerator
```

**Methods to delete from agents/conversation_agent.go:**
```
- ConversationAgent struct and methods (all moved)
- Safety checking (moved to generation/ethical_gate.go)
- All extraction methods (moved to extraction/)
```

### Step 6.2: Archive Deprecated Tables

**Option A: Keep as archive** (safe, no data loss risk)
```sql
-- Rename for archival
ALTER TABLE temporary_fact_store RENAME TO _archive_temporary_fact_store;
ALTER TABLE context_conflicts RENAME TO _archive_context_conflicts;
```

**Option B: Clean delete** (after verified everything works)
```sql
DROP TABLE temporary_fact_store;
DROP TABLE context_conflicts;
```

### Step 6.3: Verify No Dead Code

**Check:**
```bash
# Find unused functions
go tool vet ./...

# Search for calls to deleted functions
grep -r "ClarificationResponseHandler" . --include="*.go"
grep -r "ConflictResolveHandler" . --include="*.go"
grep -r "ConversationAgent{}" . --include="*.go"

# Should return 0 results
```

---

## PHASE 7: FINAL VERIFICATION (2 hours)

### Step 7.1: Build Verification
```bash
cd moly-go
go build ./...
# Must be zero errors
```

### Step 7.2: End-to-End Test Scenarios

**Scenario 1: Normal message (no conflicts)**
```
Input: "I'm feeling really stressed about work today"
Expected:
  - Message saved
  - Response generated (not hardcoded)
  - Insights extracted
  - No pending input created
Verify: data.response contains actual LLM response, not "Got it..."
```

**Scenario 2: Conflict detected**
```
Input: "Today I want to tell my boss I'm upset" 
         (after previously saying: "I want to keep work professional")
Expected:
  - Message saved
  - Conflict detected (professional vs upset)
  - Conflict question generated
  - PendingInput created in database
  - No response sent (just question)
Verify: data.pending_input.question contains conflict question
```

**Scenario 3: User answers conflict**
```
Input: "Both are true - in private I'm upset, but in the meeting I'm professional"
Expected:
  - Stage 2 detects this answers pending conflict
  - Resolution parsed ("merge")
  - AboutMe updated
  - Conflict marked as resolved and applied
  - Normal response generated
Verify: Previous conflict no longer appears, normal response sent
```

**Scenario 4: Ethical gate blocks**
```
Input: Harmful request
Expected:
  - Response blocked
  - "I can't help with that" returned
  - Metadata includes ethicalIntervention: "blocked"
Verify: Original harmful response NEVER appears in response field
```

### Step 7.3: Database Verification

```sql
-- Verify PendingInput table working:
SELECT COUNT(*) FROM pending_input WHERE resolved_at IS NULL;
-- Should show only actual pending items, none duplicated

-- Verify no orphaned data:
SELECT COUNT(*) FROM _archive_context_conflicts;
-- Should be 0 if fully migrated, or match archived total

-- Verify MessageProcessor calls:
SELECT COUNT(*) FROM chat_messages WHERE created_at > NOW() - INTERVAL 1 hour;
-- Should increase with test messages
```

### Step 7.4: Performance Verification

```
Target metrics:
- Database queries per message: 4-10 (down from 20-25)
- LLM calls per message: 3 (down from 7)
- Handler code: ~400-500 lines (down from 1400+)
- Response latency: 300-500ms (Stage 1 I/O + Stage 3 LLM)
```

Measure:
```bash
# Use time tool or logging
curl -X POST http://localhost:8080/api/v2/message-processor \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"message":"test"}' \
  -w '\nTime: %{time_total}s\n'

# Expected: < 1s for normal flow, < 2s if LLM is slow
```

---

## CRITICAL SAFEGUARDS (Prevent Dead Code & Mismatches)

### Safeguard 1: Function Mapping Audit

After all phases complete, verify each function:
```bash
# Every function in "to be deleted" list should have zero references:
for func in \
  "ClarificationResponseHandler" \
  "ConflictResolveHandler" \
  "ConversationAgent.Run" \
  "TemporaryFactStore" \
  "ContextConflictRepository"; do
  
  echo "Checking $func..."
  count=$(grep -r "$func" ./moly-go --include="*.go" | wc -l)
  
  if [ $count -gt 0 ]; then
    echo "❌ FOUND $count references to $func - DEAD CODE RISK"
    grep -r "$func" ./moly-go --include="*.go"
  else
    echo "✅ SAFE: $func has no references"
  fi
done
```

### Safeguard 2: Route Coverage

Verify every HTTP route works:
```bash
# Before: 3 routes
POST /api/v2/message-processor
POST /api/v2/clarification-response
POST /api/v2/conflict-resolve

# After: 1 route (handles all 3 cases)
POST /api/v2/message-processor

# Test all 3 cases with same route:
- Normal message
- Then clarification response
- Then conflict response
- Then conflict + new message
```

### Safeguard 3: Database Contract Verification

```bash
# Verify old tables are truly unused:
SELECT * FROM information_schema.TABLE_CONSTRAINTS 
WHERE TABLE_NAME IN ('temporary_fact_store', 'context_conflicts');

# Should be empty or only archive references

# Verify new table has correct indexes:
SHOW INDEXES FROM pending_input;
# Should show idx_user_pending and idx_conversation_type
```

### Safeguard 4: Type Safety Check

```bash
# Verify no type mismatches in new code:
go test ./... -race
# Should pass with no data races or panics
```

---

## SUCCESS CRITERIA

✅ **All phases complete:**
- [ ] No build errors: `go build ./...`
- [ ] No dead code: all old functions removed or verified unused
- [ ] No mismatches: all routes wired correctly
- [ ] Database: pending_input table working, old tables archived
- [ ] Tests: all 4 scenarios pass end-to-end
- [ ] Performance: 4-10 DB queries, 3 LLM calls, <500ms latency

✅ **No broken functionality:**
- [ ] Normal messages work
- [ ] Conflicts detected and resolved
- [ ] Clarifications work
- [ ] Ethical gate blocks harmful content
- [ ] Reflections approve/reject work
- [ ] Frontend receives real response (not hardcoded)

✅ **Code quality:**
- [ ] No unused imports
- [ ] No commented-out code
- [ ] All error paths tested
- [ ] Logging shows clear flow

---

## EXECUTION ORDER

1. Phase 0: Preparation & Verification (0.5 hours)
2. Phase 1: Create Infrastructure (4 hours)
3. Phase 2: Build Pipeline Stages (8 hours)
4. Phase 3: Extraction Tools (4 hours)
5. Phase 4: Database Access (4 hours)
6. Phase 5: Integration (2 hours)
7. Phase 6: Cleanup (1 hour)
8. Phase 7: Final Verification (2 hours)

**Total: ~25.5 hours**

---

## ROLLBACK PLAN

If critical bugs found during verification:

1. Keep old code in separate branch: `git checkout -b refactor-backup`
2. If must rollback: `git revert <first-refactor-commit>`
3. Keep database migration reversible: `ALTER TABLE` to rename back
4. No data loss: everything in pending_input can be exported to old tables

---

## Next: Begin Phase 1

Ready to start building infrastructure? Confirm or specify any adjustments needed.
