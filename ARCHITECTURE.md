# MOLY ARCHITECTURE

## System Overview

Moly is built on three layers: **Agents** (thinking), **Database** (memory), and **API** (interface).

```
┌─────────────────────────────────────────────────────────────┐
│ FRONTEND (Chrome Extension)                                 │
│ ├─ LoginScreen: Email/password auth                         │
│ ├─ ChatInterface: Natural conversation                       │
│ └─ UI state management (Zustand stores)                      │
└────────────────────────────┬────────────────────────────────┘
                             │ HTTP
                             ↓
┌─────────────────────────────────────────────────────────────┐
│ API LAYER (Go handlers)                                     │
│ ├─ Authentication endpoints (/api/auth/*)                  │
│ ├─ Message processing (/api/v2/message-processor)          │
│ ├─ Clarification handling (/api/v2/clarification/*)        │
│ ├─ Context management (/api/v2/about-me, /contacts, etc)   │
│ └─ Token validation & user isolation                        │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ↓
┌─────────────────────────────────────────────────────────────┐
│ AGENT LAYER (Backend logic)                                 │
│ ├─ ContextExtractor: LLM-based semantic understanding      │
│ │  ├─ Extracts: contact, style, intention, goals           │
│ │  ├─ Returns confidence scores (0-1)                      │
│ │  └─ Fallback: basic heuristics if LLM unavailable        │
│ ├─ ExecutionStateManager: Workflow phase tracking          │
│ │  ├─ Tracks: phase (initial→gathering→processing→complete)│
│ │  ├─ Tracks: covered categories (what's been asked)       │
│ │  └─ Optimistic locking with version tracking             │
│ ├─ SafetyChecker: Crisis & illegal pattern detection       │
│ │  ├─ Regex patterns for immediate threats                 │
│ │  └─ Returns: SafetyAlert with resources & recommendations│
│ ├─ RiskMonitor: LLM-based contextual risk analysis         │
│ │  ├─ Educational framework (Kantian, utilitarian, etc)    │
│ │  └─ Returns: severity level with guidance                │
│ ├─ ConversationAgent: Context-aware message processing     │
│ │  ├─ Uses extracted context (not hardcoded questions)     │
│ │  ├─ Generates dynamic questions per conversation phase   │
│ │  └─ Returns: phase, suggestions, extracted_contact       │
│ └─ LearningAgent: Pattern extraction & accumulation        │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ↓
┌─────────────────────────────────────────────────────────────┐
│ DATABASE LAYER (SQLite)                                     │
│ ├─ users: Authentication & profile (Bcrypt hashed)         │
│ ├─ sessions: 24-hour token session management              │
│ ├─ about_me: User communication preferences (from extraction)│
│ ├─ user_contacts: User's relationships (persisted)         │
│ ├─ conversations: Chat history (30-day persistence window) │
│ ├─ conversation_execution_state: Workflow tracking (NEW)   │
│ │  └─ Tracks: phase, covered_categories, message_seq      │
│ ├─ pending_clarifications: Facts awaiting clarification   │
│ ├─ clarification_questions: Individual questions (typed)   │
│ ├─ clarification_answers: User responses (linked)          │
│ ├─ context_attributes: Extracted facts & context           │
│ ├─ safety_incidents: Safety alerts & crises               │
│ └─ chat_messages: Encrypted message history                │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow: Message → Response

### 1. User sends message

```
Frontend → POST /api/v2/message-processor
{
  "message": "I need to talk to my boss about the project delay",
  "conversationId": "",
  "aboutMe": { /* user's profile */ }
}
```

### 2. API validates token & extracts userId

```
Authorization: Bearer <session-token>
↓
extractAndValidateToken() → userID
↓
User isolation: all queries filtered by user_id
```

### 3. Multi-stage context extraction & processing

**Stage 1: Load Execution State**
```
ExecutionStateManager.GetOrCreateState(userID, conversationID)
├─ Current phase: initial → gathering_context → processing → complete
├─ Covered categories: {contact_clarification: true, style_gathering: false}
├─ Message sequence counter
└─ Version tracking for optimistic locking
```

**Stage 2: Safety Detection**
- SafetyChecker: Pattern-based crisis/illegal detection (immediate)
- If triggered: Return SafetyAlert with resources, STOP processing
- Log incident to safety_incidents table

**Stage 3: LLM-Based Context Extraction** (Socrates pattern)
```
ContextExtractor.Extract(message)
├─ Contact: name, relationship (romantic|professional|family|friend), traits, confidence
├─ Style: communication style, tone, values, confidence  
├─ Intention: main goal/purpose
└─ Goals: list of user objectives
```
- Returns structured data with confidence scores (0-1)
- Evidence-based with message quotes
- Semantic understanding via Claude (not regex)
- Fallback to heuristics if LLM unavailable

**Stage 4: Risk Assessment** (Socratic-morality pattern)
- RiskMonitor.AssessRisk(message)
- Multi-framework analysis (Kantian, utilitarian, virtue ethics, rights-based)
- If high risk: Return educational recommendations, continue processing

**Stage 5: ConversationAgent Processing**
- Receives context from extraction (preferred) or database
- Uses extracted contact if confidence > 0.6
- Falls back to database values otherwise
- Checks execution state for covered categories
- Generates questions ONLY for uncovered categories
- Returns phase, suggestions, extracted_contact

**Stage 6: Question Deduplication & State Update**
- Filter generated questions against covered_categories
- Save new questions to database with type & status='pending'
- Update execution phase based on ConversationAgent phase
- Mark categories as covered in execution state

### 4. Execution State Management (NEW - Socrates Pattern)

**conversation_execution_state table:**
```
user_id, conversation_id
├─ phase: "initial" → "gathering_context" → "processing" → "complete"
├─ covered_categories: comma-separated list of answered question types
├─ current_message_seq: which message number in conversation
├─ started_at: when conversation began
├─ version: for optimistic locking (prevent stale writes)
└─ updated_at: refreshed on each message for 30-day persistence window
```

**Why:** Prevents duplicate questions, tracks workflow progress, enables multi-session resumption

**Example flow:**
```
Message 1: "I want to talk about a girl"
  → ContextExtractor detects: {contact: "?", relationship: "romantic"}
  → ExecutionState: covered_categories = {}
  → Generate question: "Tell me about her" (type: "contact_clarification")
  → Save: clarification_questions with type="contact_clarification", status="pending"
  → Update state: covered_categories = {contact_clarification: true}

Message 2: "I think she likes me"  
  → ExecutionState.IsCategoryCovered("contact_clarification") → true
  → Skip generating contact question (already covered)
  → Only ask about intention or new categories
```

### 5. Database stores everything

**pending_clarifications table:**
- fact_id, fact_value, fact_type
- status: pending → partially_answered → complete
- expires_at: 7 days (auto-cleanup)

**clarification_questions table:**
- Linked to pending_clarifications
- Full question data: text, type, context, options
- Sequence for ordering

**clarification_answers table:**
- User's response to each question
- Timestamp for history

**context_attributes table:**
- All extracted facts
- Linked to conversations
- Confidence scores

### 5. API returns Phase5Response

```json
{
  "success": true,
  "phase1": {
    "facts": [/* extracted facts */],
    "shifts": [/* subject changes */]
  },
  "phase2": {
    "clarifications": [/* pending questions */]
  },
  "phase3": {
    "unknown_contacts": [],
    "created_contacts": []
  },
  "phase4": {
    "saved_attributes": [/* context */]
  },
  "action_required": {
    "needsClarification": true,
    "clarificationQs": [
      {
        "id": "q_123",
        "type": "subject",
        "question": "Is this about the current project or a different one?",
        "context": "You mentioned 'the project delay'",
        "options": ["Current project", "Different project"],
        "priority": 1,
        "linkedFacts": ["fact_1"]
      }
    ],
    "temporaryFacts": []
  }
}
```

### 6. Frontend displays & waits for response

- Shows clarification questions
- User answers
- Frontend posts to `/api/v2/clarification/respond`

### 7. If user logs out mid-question

- Questions stored in DB with full data
- User logs back in
- `RemainingQuestionsWithObjects()` retrieves them
- Same questions displayed (no re-entry needed)

## Multi-Session Resumption

```
Day 1: User sends message
  → ClarificationEngine generates questions
  → Stored in database with full data
  → User logs out (mid-question)
  
Day 3: User logs back in
  → GET pending_clarifications WHERE user_id = ? AND status != 'complete'
  → SELECT clarification_questions WHERE pending_clarification_id = ?
  → Display exact same questions
  → User answers
  → Context accumulates
  → Suggestions improve
```

## Authentication & Isolation

**Registration:**
```
Email + Username + Password → Bcrypt hash → Store in users table
```

**Login:**
```
Email OR Username + Password → Verify hash → Create session
Session: {sessionId (JWT-like), userId, expiresAt (24h)}
→ Store in sessions table
→ Return to frontend
```

**Session Validation:**
```
Every API request:
  Authorization: Bearer <token>
  → extractAndValidateToken()
  → Query sessions WHERE token = ? AND expires_at > now()
  → Extract user_id
  → Verify sessions.user_id matches request user
  → All data queries filtered by user_id
```

**On Logout:**
```
DELETE FROM sessions WHERE token = ?
→ All user data removed from frontend storage
→ Contexts cleared
```

## Key Design Decisions

### 1. Schema as Neutral Layer
- `schema.ClarificationQuestion` is single source of truth
- Agents import schema, models import schema
- Breaks circular dependency (agents ← models)

### 2. Database-Backed Clarifications
- Not in-memory (survives logout/restart)
- Full question data stored (not just text)
- Multi-question sequences supported
- Auto-expiry after 7 days

### 3. User Isolation Per Session
- Bearer token → sessions table → user_id
- Every query filtered by user_id
- No cross-user data access
- Logout deletes session immediately

### 4. Agentic Processing
- ConversationAgent drives logic
- Each phase independent
- Extensible for new capabilities
- LLM integration for reasoning

### 5. Natural Conversation Flow
- No setup forms required
- Context extracted from messages
- Questions asked progressively
- Learning accumulates over time

## Socrates Architectural Patterns (Sept 2026)

Moly evolved to inherit proven patterns from Socrates and Socratic-Morality projects:

### 1. LLM-Based Semantic Extraction (vs Regex)

**Pattern: Intelligent categorization**

Instead of hardcoded keyword matching:
```go
// Old (broken):
if contains(msg, "girl") { relationship = "romantic" }

// New (Socrates pattern):
extracted := contextExtractor.Extract(msg)
// Returns: {contact: {name?, relationship:"romantic", confidence:0.85}}
```

**Benefits:**
- Understands intent ("She's someone I like" → romantic contact)
- Confidence scores on all data (0-1)
- Evidence-based with message quotes
- Semantic understanding via LLM

**Implementation:** `agents/context_extractor.go`

### 2. Execution State Tracking (vs No State)

**Pattern: Workflow phase machine**

From Socrates' `WorkflowExecutionState`:
```go
ExecutionPhase: initial → gathering_context → processing → complete
CoveredCategories: {contact_clarification: true, style_gathering: false}
CurrentMessageSeq: which message in conversation
Version: optimistic locking
```

**Benefits:**
- Prevents asking same question twice
- Tracks where in conversation we are
- Enables multi-session resumption
- Atomic updates with version checking

**Implementation:** `agents/execution_state.go` + `conversation_execution_state` table

### 3. Dynamic Question Generation (vs Hardcoded)

**Pattern: Category-driven progressive disclosure**

Instead of fixed question sequence:
```go
// Old (Moly v1):
if !hasAboutMe { return "Tell me about yourself..." }
if !hasContact { return "Who are you messaging?" }

// New (Socrates pattern):
uncovered := execState.GetUncoveredCategories(allCategories)
questions := questionGenerator.GenerateForCategories(uncovered)
// Questions only for categories not yet answered
```

**Benefits:**
- Questions adapt to context
- No duplicate questions
- Progressive information gathering
- Better user experience

### 4. Multi-Framework Risk Analysis (vs Pattern Matching)

**Pattern: Socratic-Morality ethical framework**

Instead of only regex patterns:
```go
// SafetyChecker: Pattern-based (immediate threats)
crisis := safetyChecker.CheckMessage(msg)  // regex

// RiskMonitor: LLM-based (contextual risks)
risk := riskMonitor.AssessRisk(msg)  // Kantian, utilitarian, virtue ethics
```

**Benefits:**
- Immediate detection (SafetyChecker)
- Contextual understanding (RiskMonitor)
- Multi-framework reasoning
- Educational guidance instead of just blocking

**Implementation:** `safety/checker.go` + `agents/risk_monitor.go`

### 5. Confidence Scoring (vs Binary Decisions)

**Pattern: Uncertain data explicitly represented**

All extracted data includes confidence:
```json
{
  "contact": {
    "name": "?",
    "relationship": "romantic",
    "confidence": 0.85,
    "evidence": "mentioned someone special"
  }
}
```

**Benefits:**
- Transparent uncertainty
- Fallback strategies (prefer high-confidence extracted data)
- Merge multiple sources intelligently
- Improves gradually as confidence grows

## Extending Moly

To add a new capability (following Socrates patterns):

1. **New extraction type? (e.g., values, goals, preferences)**
   - Add field to `ExtractedContext` struct
   - Update `ContextExtractor.buildExtractionPrompt()` LLM prompt
   - Include confidence scoring
   - Add evidence from message

2. **New question category? (e.g., "goal_clarification")**
   - Define category name
   - Add to `QuestionGenerator.uncoveredCategories` logic
   - Generate only when category not in `execState.CoveredCategories`
   - Mark covered when user answers

3. **New execution phase? (e.g., "conflict_resolution")**
   - Add phase to `ExecutionPhase` enum
   - Update `ExecutionStateManager.UpdatePhase()`
   - Conditional question generation based on phase
   - Persist in `conversation_execution_state` table

4. **New risk framework? (e.g., feminist ethics)**
   - Add to `RiskMonitor.frameworks` array
   - Implement `_<framework>_analysis()` method
   - Return: allowed, confidence, concerns
   - Aggregate into final decision

5. **Integrate safety detection for new threat type?**
   - Add regex patterns to `SafetyChecker`
   - Add to `compileCrisisPatterns()` or `compileIllegalPatterns()`
   - Returns `SafetyAlert` with resources
   - Triggers immediate halt to processing

6. **Frontend UI change?**
   - Chat happens in ChatInterface.tsx
   - Uses useAuth for session
   - useAboutMe for profile
   - Other hooks for execution state
   - Questions now dynamic (not hardcoded list)

## See Also

- MOLY_VISION.md: Why Moly exists, core principles
- DEVELOPMENT.md: How to build and run locally
- API.md: Endpoint contracts
- CONTRIBUTING.md: Code guidelines
