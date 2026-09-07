# Moly Implementation Roadmap - Complete

## Current Status

### ✓ Completed
- Database schema redesigned for multi-contact conversations
- Safety/ethics gating logic in Sidebar (blocks suggestions for unsafe/unethical content)
- Backend analysis endpoints (safety check, constitution eval)
- Extension UI structure (Sidebar, message input, suggestions)
- Service worker with fallback LLM provider chain

### ✗ Missing (Critical Path to MVP)
- Conversation management system (create, select, display)
- Contact management UI (add, edit, view contact details)
- Context injection into analysis
- Conversation history and learning
- Multi-contact conversation selection

---

## Phase 1: Conversation Management UI

### 1.1 Replace ContactSelector with ConversationSelector
**File:** `moly-extension/src/sidebar/components/ConversationSelector.tsx` (NEW)

**What it does:**
- Shows list of existing conversations
- "New Conversation" button
- Selected conversation shows member list
- Shows conversation type and purpose

**Implementation:**
```typescript
interface ConversationSelectorProps {
  onSelectConversation: (conversation: Conversation) => void;
  onNewConversation: () => void;
  currentConversation: Conversation | null;
}

// Fetch from chrome.storage.local (conversations stored locally)
// Display with:
// - Conversation name + type
// - Member avatars/names
// - Last interaction date
// - Purpose (cover letter, advice, etc.)
```

**Data Structure (in chrome.storage.local):**
```json
{
  "conversations": [
    {
      "id": 1,
      "name": "Sarah - Dating",
      "type": "single",
      "purpose": "relationship",
      "members": [{ "id": 1, "name": "Sarah", "relationship": "romantic" }],
      "notes": "Recently moved to LA, likes hiking",
      "created_at": 1234567890,
      "updated_at": 1234567890
    }
  ]
}
```

---

### 1.2 New Conversation Flow
**File:** `moly-extension/src/sidebar/components/NewConversationModal.tsx` (NEW)

**Steps:**
1. User clicks "New Conversation"
2. Modal opens with:
   - Conversation name (auto-filled or custom)
   - Type selector: `single` | `group` | `generic`
   - Purpose selector: `relationship` | `cover_letter` | `advice` | `other`
   - Multi-select contact picker
   - Notes field
3. On submit:
   - Save conversation to chrome.storage.local
   - Send to Go backend (eventually)
   - Select it in ConversationSelector

**Implementation:**
```typescript
interface NewConversationFlow {
  name: string;
  type: 'single' | 'group' | 'generic';
  purpose: string;
  selectedContactIds: number[];
  notes: string;
}

// When submitting:
// 1. Call backend: POST /api/create-conversation
// 2. Get conversation ID
// 3. Store in chrome.storage.local
// 4. Auto-select it
```

---

### 1.3 Contact Management UI
**File:** `moly-extension/src/sidebar/components/ContactManager.tsx` (NEW)

**Features:**
- List all contacts
- Add new contact form
- Edit contact details
- View contact notes/history

**Contact Form Fields:**
- Name (required)
- Relationship (friend/romantic/work/family/other)
- Platform (text/email/dating-app/work-chat/other)
- Communication style notes
- Initial notes about person

**On Add Contact:**
- Save to Go backend: POST /api/create-contact
- Moly suggests: "Let's chat about [Name]. Tell me about them?"
- Store notes in backend for future context

---

## Phase 2: Context Building & Injection

### 2.1 Contact Context Endpoint
**File:** `moly-go/main.go` - NEW HANDLER

**Endpoint:** `POST /api/get-conversation-context`

**Request:**
```json
{
  "conversation_id": 1,
  "include_history": true
}
```

**Response:**
```json
{
  "conversation": {
    "id": 1,
    "name": "Sarah - Dating",
    "type": "single",
    "purpose": "relationship",
    "notes": "Recently moved to LA, likes hiking"
  },
  "members": [
    {
      "id": 1,
      "name": "Sarah",
      "relationship": "romantic",
      "platform": "text",
      "notes": "Communication style: direct, appreciates humor",
      "interaction_count": 15,
      "last_interaction": "2026-09-05"
    }
  ],
  "recent_interactions": [
    {
      "date": "2026-09-04",
      "topic": "work stress",
      "sentiment": "worried",
      "summary": "Sarah is dealing with project deadline pressure"
    }
  ],
  "context_summary": "Sarah is your romantic interest. She's direct and appreciates humor. Recently stressed about work but excited about outdoor activities."
}
```

### 2.2 Inject Context into Analysis
**File:** `moly-extension/src/sidebar/Sidebar.tsx` - MODIFY

**Current:** `await analyze(userMessage, contactName, context)`

**New:** 
```typescript
// Before analyzing:
const conversationContext = await backend.getConversationContext(selectedConversation.id);

// Pass to analyze:
const results = await analyze(
  userMessage,
  selectedConversation,
  conversationContext // NEW: full context object
);
```

**In MolyAgent:**
- Include context in safety check prompt
- Include context in constitution eval prompt
- Use context for personalized analysis

### 2.3 Backend Analysis with Context
**File:** `moly-go/main.go` - MODIFY handlers

**Update safety check:**
- Include conversation context in prompt to Go backend
- Better understanding of situation
- More nuanced risk assessment

**Update constitution eval:**
- Know relationship type (romantic vs work vs friend)
- Evaluate ethics in context
- E.g., "stealing from friend" differs by relationship

**New suggestion system:**
- Pass full conversation context to LLM
- "For Sarah (romantic partner who likes hiking)..."
- Suggestions personalized to all participants

---

## Phase 3: Conversational Learning

### 3.1 Post-Conversation Reflection
**File:** `moly-extension/src/sidebar/components/ReflectionModal.tsx` (NEW)

**After LLM suggestions generated:**
- Ask: "Anything I should remember about this conversation?"
- User can add notes
- Store on conversation record
- Reference on future chats

**Flow:**
1. After suggestions shown
2. Optional reflection: "Save notes for next time?"
3. User adds summary/insights
4. Save to conversation.notes

### 3.2 Conversation Memory Store
**File:** `moly-extension/src/stores/conversationStore.ts` (NEW)

**Manages:**
- Current selected conversation
- Conversation list
- Load/save to chrome.storage.local
- Sync with backend

**Data:**
```typescript
interface ConversationStore {
  conversations: Conversation[];
  selectedConversation: Conversation | null;
  loading: boolean;
  error: string | null;
  
  // Methods
  loadConversations(): Promise<void>;
  selectConversation(id: number): Promise<void>;
  createConversation(conv: Conversation): Promise<void>;
  addNotes(id: number, notes: string): Promise<void>;
}
```

---

## Phase 4: Backend Integration

### 4.1 New Go Endpoints
**Implement in `moly-go/main.go`:**

```
POST /api/create-conversation
  - Create conversation in database
  - Add members
  - Return conversation ID

POST /api/create-contact
  - Create contact in database
  - Return contact ID

POST /api/get-conversation-context
  - Fetch conversation + members + recent interactions
  - Build context summary

POST /api/add-conversation-notes
  - Update conversation notes
  - Called after reflection

POST /api/get-all-conversations
  - List all conversations
  - For ConversationSelector

POST /api/analyze-with-context
  - Extended analyze endpoint
  - Accepts conversation context
  - Returns analysis with context awareness
```

### 4.2 Database Queries
**Implement in `moly-go/database.go`:**

```
- GetAllConversations()
- UpdateConversationNotes(conversationID, notes)
- GetConversationWithMembers(conversationID)
- RecordInteractionToConversation(conversationID, interaction)
```

---

## Phase 5: Gated Suggestions (Already Partially Done)

### 5.1 Current Implementation
✓ Crisis detection → show resources, don't suggest
✓ Ethics violations → ask clarifying questions

### 5.2 Still Needed
- Personalization based on relationship type
- Multi-participant context awareness
- Contextual question generation

---

## Implementation Order (Recommend)

**Sprint 1: Foundation**
1. Build ConversationSelector UI
2. Build NewConversationModal
3. Build ContactManager UI
4. Update Sidebar to use ConversationSelector

**Sprint 2: Backend Plumbing**
1. Implement conversation CRUD endpoints
2. Implement contact endpoints
3. Implement get-conversation-context endpoint
4. Wire extension to backend

**Sprint 3: Context Injection**
1. Modify analyze() to accept context
2. Update Go handlers to use context
3. Wire context from selector to analysis

**Sprint 4: Learning Loop**
1. Build ReflectionModal
2. Implement post-conversation notes
3. Create conversationStore
4. Hook into history display

---

## Key Integration Points

### Sidebar Flow (New)
```
User clicks icon
  ↓
ConversationSelector loads conversations
  ↓
User selects or creates conversation
  ↓
Get conversation context from backend
  ↓
User types message
  ↓
analyze(message, conversation, context) ← CONTEXT PASSED
  ↓
Safety/ethics gating with full context
  ↓
Generate suggestions (personalized to members)
  ↓
Display results
  ↓
Optional: Reflection modal
  ↓
Save notes to conversation
```

### Data Flow
```
Extension (chrome.storage.local)
  ↓
Go Backend (SQLite database)
  ↓ 
Analysis Endpoints (context-aware)
  ↓
LLM Suggestions (personalized)
```

---

## Critical Notes

### Do NOT implement without:
1. ✗ Don't add suggestions without conversation context
2. ✗ Don't use old ContactSelector approach
3. ✗ Don't skip reflection/learning step
4. ✗ Don't hardcode contact names in analysis

### Must have for MVP:
1. ✓ Single contact conversations
2. ✓ Multi-contact support
3. ✓ Generic purpose chats
4. ✓ Context injection to analysis
5. ✓ Gated suggestions (crisis/ethics)
6. ✓ Post-conversation learning

### Future (Post-MVP):
- Predictive suggestion refinement (learns over time)
- Relationship timeline tracking
- Platform-specific coaching
- Export conversation summaries

---

## Files to Create (Summary)

| File | Purpose |
|------|---------|
| ConversationSelector.tsx | UI for selecting/creating conversations |
| NewConversationModal.tsx | Flow for adding conversation |
| ContactManager.tsx | Contact CRUD UI |
| ReflectionModal.tsx | Post-conversation notes |
| conversationStore.ts | State management |
| conversation-api.ts | Bridge to Go backend |

## Files to Modify (Summary)

| File | Change |
|------|--------|
| Sidebar.tsx | Use ConversationSelector, pass context to analyze |
| molyAgent.ts | Accept context parameter |
| main.go | Add conversation endpoints |
| database.go | Add conversation queries |

---

## Status

**Current:** Database schema done, safety gating done  
**Next:** UI for conversation management (Sprint 1)  
**Timeline:** Realistic MVP = 4 sprints (4-6 weeks)

