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
│ ├─ ConversationAgent: 5-phase message processing           │
│ │  ├─ Phase 1: Extract facts                               │
│ │  ├─ Phase 2: Generate clarifications                     │
│ │  ├─ Phase 3: Identify contacts                           │
│ │  ├─ Phase 4: Attribute context                           │
│ │  └─ Phase 5: Generate suggestions                        │
│ ├─ ClarificationEngine: Question generation                │
│ ├─ ContextManager: Profile & contact management            │
│ └─ LearningAgent: Pattern extraction                        │
└────────────────────────────┬────────────────────────────────┘
                             │
                             ↓
┌─────────────────────────────────────────────────────────────┐
│ DATABASE LAYER (SQLite)                                     │
│ ├─ users: Authentication & profile                         │
│ ├─ about_me: User communication preferences                │
│ ├─ contacts: User's relationships                          │
│ ├─ pending_clarifications: Facts awaiting clarification   │
│ ├─ clarification_questions: Individual questions           │
│ ├─ clarification_answers: User responses                   │
│ ├─ context_attributes: Extracted facts & context           │
│ └─ conversations: Chat history                             │
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

### 3. ConversationAgent processes message (5 phases)

**Phase 1: Extract Facts**
- Parses message for relationships, events, context
- Returns: `[]*ExtractedFact`

**Phase 2: Generate Clarifications**
- Identifies missing context
- Generates targeted questions
- Returns: `[]*ClarificationQuestion`

**Phase 3: Identify Contacts**
- Detects person mentions
- Matches against user's contacts
- Creates new contacts if needed

**Phase 4: Attribute Context**
- Links facts to user's profile
- Calculates confidence scores
- Stores in context_attributes table

**Phase 5: Generate Suggestions**
- Uses accumulated context
- Generates multiple response options
- Returns ranked by relevance

### 4. Database stores everything

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

## Extending Moly

To add a new capability:

1. **New question type?**
   - Add to ClarificationEngine
   - Returns `*schema.ClarificationQuestion`
   - Stored in database automatically

2. **New context to extract?**
   - Add to Phase 1 (ExtractedFact types)
   - Add to Phase 4 (ContextAttribute storage)
   - Phase 5 automatically uses it

3. **New API endpoint?**
   - Create handler that calls extractAndValidateToken()
   - Always filter queries by user_id
   - Return proper error for 401/403

4. **Frontend UI change?**
   - Chat happens in ChatInterface.tsx
   - Uses useAuth for session
   - useAboutMe for profile
   - Other hooks for data

## See Also

- MOLY_VISION.md: Why Moly exists, core principles
- DEVELOPMENT.md: How to build and run locally
- API.md: Endpoint contracts
- CONTRIBUTING.md: Code guidelines
