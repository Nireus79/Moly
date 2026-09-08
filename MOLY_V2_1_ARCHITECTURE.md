# Moly V2.1 Architecture

**Status**: Architectural Design (not yet implemented)  
**Version**: 2.1.0  
**Date**: September 8, 2026  
**Based on**: V2.0 (functional backend)  
**Timeline**: 3-4 weeks to implement

---

## Executive Summary

V2.1 is a fundamental pivot from **suggestion-based tool** to **personal communication coach**. Instead of users selecting text and getting suggestions, users chat with Moly—a personalized AI assistant that:
- Learns about users through natural conversation
- Helps with any communication challenge
- Suggests saving contacts when people are mentioned
- Improves over time through implicit learning

**Key Changes**:
- Multi-user with authentication (login codes)
- Chat interface (not suggestion picker)
- Implicit context gathering (conversation, not forms)
- Direct Moly chat (like ChatGPT, but personalized)

---

## Vision

### V2.0 Vision (Current)
> "Help users write better messages by providing AI-generated suggestions based on context"

### V2.1 Vision (New)
> "Be a trusted communication coach who knows the user personally, understands their relationships, and helps them navigate any communication challenge through natural conversation"

**Shift**: Tool → Personal Assistant

---

## Core Principles (V2.1)

1. **Chat First** - Natural conversation is better than forms
2. **Implicit Learning** - Gather context naturally, not through surveys
3. **Personal Relationship** - Moly knows user as a person, not just data
4. **Multi-User Ready** - Multiple people can use same installation
5. **Socratic at Heart** - Ask better questions, not just give answers
6. **No Friction** - Skip onboarding, start chatting immediately
7. **Privacy First** - All data stored locally, user owns their context

---

## Architecture Overview

### System Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                   Browser Extension                         │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 1. Login Screen                                     │   │
│  │    - Show generated code                            │   │
│  │    - Copy button                                    │   │
│  │    - Remember session                              │   │
│  └─────────────────────────────────────────────────────┘   │
│                        ↓                                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 2. Chat Interface                                   │   │
│  │    - Message list (not suggestions)                │   │
│  │    - Text input                                     │   │
│  │    - Contact mention cards                         │   │
│  │    - Socratic question suggestions                 │   │
│  │    - Learning indicators                           │   │
│  └─────────────────────────────────────────────────────┘   │
│                        ↓                                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 3. Content Script                                   │   │
│  │    - Inject "Ask Moly" button near textareas       │   │
│  │    - Capture selected text                         │   │
│  │    - Quick access to context                       │   │
│  └─────────────────────────────────────────────────────┘   │
│                        ↓                                    │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ 4. Background Worker                                │   │
│  │    - Message routing                                │   │
│  │    - Session management                             │   │
│  │    - Storage management                             │   │
│  └─────────────────────────────────────────────────────┘   │
│                        ↓                                    │
└────────────────────────────────────────────────────────────┬┘
                         │ HTTP
                         ↓
        ┌────────────────────────────────────────┐
        │  Moly V2.1 Backend (Go)                │
        │                                        │
        │  ┌──────────────────────────────────┐  │
        │  │ Auth System (NEW)                │  │
        │  │ - Generate codes                 │  │
        │  │ - Session management             │  │
        │  │ - Multi-user support             │  │
        │  └──────────────────────────────────┘  │
        │            ↓                            │
        │  ┌──────────────────────────────────┐  │
        │  │ Chat API (NEW)                   │  │
        │  │ - POST /api/v2.1/chat            │  │
        │  │ - Multi-turn conversation        │  │
        │  │ - Contact mention detection      │  │
        │  │ - Implicit context learning      │  │
        │  └──────────────────────────────────┘  │
        │            ↓                            │
        │  ┌──────────────────────────────────┐  │
        │  │ Chat Agent (REFACTORED)          │  │
        │  │ - Return chat responses          │  │
        │  │ - Embed Socratic questions       │  │
        │  │ - Detect contact mentions        │  │
        │  │ - Implicit AboutMe/Contact learn │  │
        │  └──────────────────────────────────┘  │
        │            ↓                            │
        │  ┌──────────────────────────────────┐  │
        │  │ LLM Integration (REUSE)          │  │
        │  │ - Ollama (local)                 │  │
        │  │ - Claude API                     │  │
        │  │ - Heuristic fallback             │  │
        │  └──────────────────────────────────┘  │
        │            ↓                            │
        │  ┌──────────────────────────────────┐  │
        │  │ Learning System (REUSE)          │  │
        │  │ - Extract AboutMe                │  │
        │  │ - Extract Contacts               │  │
        │  │ - Implicit pattern detection     │  │
        │  │ - Safety monitoring              │  │
        │  └──────────────────────────────────┘  │
        │            ↓                            │
        │  ┌──────────────────────────────────┐  │
        │  │ Database (REUSE)                 │  │
        │  │ - AboutMe, Contacts              │  │
        │  │ - Chat messages (NEW)            │  │
        │  │ - Sessions (NEW)                 │  │
        │  │ - Conversation history (NEW)     │  │
        │  └──────────────────────────────────┘  │
        └────────────────────────────────────────┘
```

---

## User Flows

### Flow 1: First Time User

```
1. Install Extension
   ↓
2. Extension generates unique code: "moly-12345-abcde"
   ↓
3. Display code with "Copy to clipboard" button
   ↓
4. User clicks extension icon
   ↓
5. Extension UI shows login screen with code
   ↓
6. User presses "Log In"
   ↓
7. POST /api/v2.1/auth/login {code: "moly-12345-abcde"}
   ↓
8. Backend validates code, creates session, returns sessionId
   ↓
9. Extension saves sessionId to storage
   ↓
10. Chat interface opens with Moly greeting:
    "Hi! I'm Moly, your communication coach. What's your name?"
    ↓
11. User types: "Sarah"
    ↓
12. Moly: "Nice to meet you, Sarah! To help you better, I'd like to 
    understand how you communicate. How would you describe your 
    communication style?"
    ↓
13. User can:
    a) Answer (context gathering begins)
    b) Skip ("I'll tell you later") → Chat about anything
    ↓
14. User starts chatting about communication challenge
    ↓
15. Moly learns, suggests, helps
```

### Flow 2: Chat with Contact Mention

```
User: "I need to tell my boss about the mistake I made"
  ↓
Backend detects: "boss" is a person reference
  ↓
Moly response: "That takes courage. Before we work on that, 
have you told me about your boss? I can remember details about 
your important relationships."
  ↓
UI shows contact card: "Save Sarah (boss) to contacts?"
  ↓
User clicks: "Yes, tell me more"
  ↓
Moly: "How long have you worked with your boss, and how would 
you describe your working relationship?"
  ↓
User: "2 years, she's fair but very results-focused"
  ↓
Backend stores: Contact {name: "Sarah", relationship: "boss", 
characteristics: ["fair", "results-focused"], tenure: "2 years"}
  ↓
Moly: "Got it. Knowing that Sarah is results-focused, how should 
you frame your mistake—as a learning opportunity or a problem 
already solved?"
  ↓
Conversation continues with personalization
```

### Flow 3: Implicit AboutMe Learning

```
User: "I hate small talk"
  ↓
Moly: "I hear you. So you prefer getting straight to the point?"
  ↓
User: "Yes, exactly"
  ↓
Backend learns: AboutMe {preferences: [..., "direct", "no_small_talk"]}
  ↓
Next message Moly sends: "Got it, I'll be direct with you too."
  ↓
Future conversations reference this preference naturally
```

---

## API Specification (V2.1)

### 1. Authentication Endpoints

#### Generate Code (Backend → Extension)
```
GET /api/v2.1/auth/generate
Response:
{
  "code": "moly-12345-abcde",
  "expiresIn": 3600,
  "instructionUrl": "https://moly.localhost/setup"
}
```

#### Login with Code
```
POST /api/v2.1/auth/login
{
  "code": "moly-12345-abcde"
}
Response:
{
  "success": true,
  "userId": "user_xyz123",
  "sessionId": "sess_abc123def456",
  "expiresIn": 86400,
  "isNewUser": true
}
```

#### Verify Session
```
GET /api/v2.1/auth/verify
Headers: Authorization: Bearer {sessionId}
Response:
{
  "valid": true,
  "userId": "user_xyz123",
  "isNewUser": false,
  "lastActive": 1694173200
}
```

#### Logout
```
POST /api/v2.1/auth/logout
Headers: Authorization: Bearer {sessionId}
Response:
{
  "success": true
}
```

### 2. Chat Endpoint (Core)

#### Send Message
```
POST /api/v2.1/chat
Headers: Authorization: Bearer {sessionId}
{
  "message": "I need to talk to my manager about a mistake",
  "conversationId": "conv_123",
  "context": {
    "isFirstMessage": false,
    "parentMessageId": "msg_456"
  }
}

Response:
{
  "messageId": "msg_789",
  "response": "That takes courage. Before we work on that...",
  "contactMention": {
    "detected": true,
    "person": "manager",
    "suggestion": "Would you like to save info about your manager?"
  },
  "aboutMeGaps": ["relationship_dynamic"],
  "suggestedFollowUp": "How long have you worked together?",
  "contextLearned": {
    "communication_challenge": "addressing mistake with authority",
    "emotional_state": "anxious"
  },
  "timestamp": 1694173200
}
```

### 3. Context Management (Implicit)

#### Get AboutMe (Auto-Extracted)
```
GET /api/v2.1/about-me
Headers: Authorization: Bearer {sessionId}
Response:
{
  "name": "Sarah",
  "communicationStyle": "direct but warm",
  "values": ["honesty", "clarity"],
  "preferences": ["no_small_talk"],
  "extractedFrom": [
    {
      "message": "I hate small talk",
      "learned": "preferences.no_small_talk",
      "confidence": 0.92
    }
  ],
  "confidence": 0.75
}
```

#### Save/Approve AboutMe
```
POST /api/v2.1/about-me/approve
Headers: Authorization: Bearer {sessionId}
{
  "field": "communicationStyle",
  "value": "direct but warm",
  "override": false
}
Response:
{
  "success": true,
  "updatedAt": 1694173200
}
```

#### Get Contacts
```
GET /api/v2.1/contacts
Headers: Authorization: Bearer {sessionId}
Response:
{
  "contacts": [
    {
      "id": "contact_123",
      "name": "Sarah",
      "relationship": "boss",
      "characteristics": ["fair", "results-focused"],
      "tenure": "2 years",
      "lastMentioned": 1694173200,
      "confidence": 0.95
    }
  ]
}
```

#### Save Contact
```
POST /api/v2.1/contacts
Headers: Authorization: Bearer {sessionId}
{
  "name": "Sarah",
  "relationship": "boss",
  "characteristics": ["fair", "results-focused"],
  "tenure": "2 years",
  "notes": "Can be direct with her"
}
Response:
{
  "id": "contact_123",
  "success": true
}
```

### 4. Conversation History

#### Get Chat History
```
GET /api/v2.1/conversations/{conversationId}/messages
Headers: Authorization: Bearer {sessionId}
Response:
{
  "messages": [
    {
      "id": "msg_1",
      "role": "assistant",
      "content": "Hi Sarah, what brings you to Moly today?",
      "timestamp": 1694173000
    },
    {
      "id": "msg_2",
      "role": "user",
      "content": "I need to talk to my boss...",
      "timestamp": 1694173100
    }
  ]
}
```

### 5. Health Check (Unchanged)

```
GET /api/v2.1/health
Response:
{
  "status": "ok",
  "database": "ok",
  "llm": "ok|none",
  "auth": "ok",
  "time": "2026-09-08T12:00:00Z"
}
```

---

## Data Models (V2.1)

### New Tables

#### sessions
```sql
CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  code TEXT UNIQUE,
  created_at INTEGER,
  expires_at INTEGER,
  last_active INTEGER,
  device_name TEXT
);
```

#### chat_messages (new)
```sql
CREATE TABLE chat_messages (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  conversation_id TEXT NOT NULL,
  role TEXT,  -- "user" or "assistant"
  content TEXT,
  context_extracted JSONB,
  contact_mention JSONB,
  created_at INTEGER,
  FOREIGN KEY (user_id) REFERENCES users(id)
);
```

#### implicit_learning (new)
```sql
CREATE TABLE implicit_learning (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  message_id TEXT,
  field TEXT,  -- "communication_style", "values", etc
  extracted_value TEXT,
  confidence REAL,
  created_at INTEGER,
  approved BOOLEAN,
  FOREIGN KEY (user_id) REFERENCES users(id)
);
```

### Modified Tables

#### about_me
```sql
-- Add columns:
ALTER TABLE about_me ADD COLUMN extracted_count INTEGER DEFAULT 0;
ALTER TABLE about_me ADD COLUMN total_confidence REAL DEFAULT 0.0;
ALTER TABLE about_me ADD COLUMN last_learned_at INTEGER;
```

#### contacts
```sql
-- Add columns:
ALTER TABLE contacts ADD COLUMN tenure TEXT;
ALTER TABLE contacts ADD COLUMN extraction_confidence REAL DEFAULT 0.0;
ALTER TABLE contacts ADD COLUMN last_mentioned INTEGER;
ALTER TABLE contacts ADD COLUMN mention_count INTEGER DEFAULT 0;
```

---

## Implementation Roadmap

### Phase 1: Core Chat (Weeks 1-2)
**Goal**: Basic chat working with authentication and encryption

- [ ] Add auth system (codes, sessions)
- [ ] Implement SQLCipher (AES-256 database encryption)
- [ ] Create chat endpoint
- [ ] Refactor ConversationAgent for chat mode
- [ ] Create chat message storage (encrypted by default)
- [ ] Update extension with chat UI
- [ ] Test login flow end-to-end
- [ ] Verify encryption working (database unreadable without key)

**Deliverable**: Users can log in and chat with Moly with data encrypted at rest

### Phase 2: Smart Context (Week 3)
**Goal**: Implicit learning and contact detection

- [ ] Implement contact mention detection (NLP)
- [ ] Add implicit AboutMe extraction
- [ ] Create "Save contact?" suggestion UI
- [ ] Build confidence scoring for learning
- [ ] Add Socratic question embedding
- [ ] Store learned context

**Deliverable**: Moly learns naturally through chat

### Phase 3: Polish (Week 4)
**Goal**: Production-ready experience

- [ ] AboutMe/Contact viewers
- [ ] Learning indicators ("I've learned...")
- [ ] Chat history export
- [ ] Conversation search
- [ ] UX polish (loading states, errors)
- [ ] Comprehensive testing

**Deliverable**: Polished, ready to ship

---

## Comparison: V2.0 vs V2.1

| Aspect | V2.0 | V2.1 |
|--------|------|------|
| **User Model** | Single user | Multi-user with auth |
| **Primary Interface** | Suggestion picker | Chat |
| **Context Source** | User-filled forms | Natural conversation |
| **AboutMe Gathering** | Form-based | Socratic in chat |
| **Contact Saving** | Form + manual | Auto-detect + approve |
| **Learning** | Feedback-based | Implicit + feedback |
| **Relationship with Moly** | Tool | Personal coach |
| **Entry Point** | "Select text" | "Open chat" |
| **Onboarding** | Skip or complete form | Start chatting immediately |
| **Chat with Moly** | No | Yes, main feature |

---

## V2.0 → V2.1 Migration

### What Stays
- ✅ Database schema (7 repositories work as-is)
- ✅ Agent architecture (refactored, not rebuilt)
- ✅ LLM integration (Ollama, Claude API, heuristics)
- ✅ Safety monitoring (reused)
- ✅ Learning system (reused for implicit learning)
- ✅ Risk assessment (unchanged)

### What Changes
- 🔄 ConversationAgent (suggestion → chat mode)
- 🔄 API endpoints (suggestion → chat endpoint)
- 🔄 Extension UI (popup suggestion → chat interface)
- 🔄 Authentication (none → code-based)

### What's New
- ✨ Auth system (codes, sessions)
- ✨ Chat endpoint
- ✨ Chat history storage
- ✨ Implicit learning system
- ✨ Contact mention detection
- ✨ Chat UI for extension

---

## Technical Decisions

### 1. Authentication Method
**Decision**: Code-based login (not OAuth)  
**Reasoning**: 
- Simple to implement
- No third-party dependencies
- Works offline
- Clear to users ("Your code is moly-12345")
- Codes can expire

### 2. Context Learning
**Decision**: Implicit extraction + explicit approval  
**Reasoning**:
- Natural conversation doesn't feel like a survey
- Users can override/approve learning
- Confidence scoring handles uncertainty
- No friction for users

### 3. Chat vs Suggestion API
**Decision**: Replace suggestion endpoint with chat endpoint  
**Reasoning**:
- Chat is more flexible (multi-turn, context-aware)
- Can embed suggestions within conversation
- More natural interaction
- Easier to scale (V2.1 foundation)

### 4. Contact Detection
**Decision**: Keyword matching + Socratic confirmation  
**Reasoning**:
- Simple to implement initially
- Can improve with ML later
- Socratic questions confirm intent
- No false positives hurting UX

### 5. Database Encryption
**Decision**: SQLCipher with AES-256, key derived from user ID  
**Reasoning**:
- SQLCipher is mature, drop-in SQLite replacement
- AES-256 is unbreakable with current tech
- Key derived from user ID (not stored, ephemeral in memory)
- If system compromised, database still unreadable
- No cloud sync needed—local encryption is sufficient
- Minimal performance overhead on modern systems

**Implementation**:
```go
// Key derivation (one per user)
key := SHA256(userID + salt)  // 32-byte key for AES-256

// Database connection
sqlcipher.Open(dbPath + "?key=" + key)

// Result: plaintext queries, encrypted file at rest
```

---

## Risks & Mitigations

| Risk | Severity | Mitigation |
|------|----------|-----------|
| Session token exposure | Medium | Use secure storage, short expiry |
| Contact mention false positives | Low | Require user confirmation |
| Implicit learning errors | Low | Confidence scoring, user can override |
| Chat context explosion | Medium | Implement message limit, summarization |
| Multi-device sync | Low | Out of scope for V2.1 (Phase 2) |
| Data at rest exposure | High | SQLCipher AES-256 encryption (Phase 1) |

---

## Success Criteria

V2.1 is complete when:

- [ ] Users can log in with code
- [ ] Chat interface works end-to-end
- [ ] Moly responds naturally to chat
- [ ] Contact mentions detected automatically
- [ ] AboutMe gathered implicitly through chat
- [ ] "Save contact?" suggestions appear contextually
- [ ] Socratic questions embedded in conversation
- [ ] All 71 existing tests pass + 30 new chat tests
- [ ] Zero console errors in extension
- [ ] Works with Ollama + heuristic fallback
- [ ] Chat history persists across sessions
- [ ] Learning confidence scoring working
- [ ] Database encrypted with AES-256 (SQLCipher)
- [ ] Cannot read database without encryption key

---

## Future Possibilities (Phase 2+)

- Cloud sync of user profiles
- Multi-device support
- Conversation search & analytics
- Voice chat
- Contact relationship mapping
- Communication insights dashboard
- Team/family accounts
- Private calendar integration
- Email drafting assistance

---

## Summary

**V2.1 transforms Moly from a suggestion tool into a personal communication coach.**

- **Single shift**: Suggestion-picking → Natural chat
- **Major benefit**: More intuitive, faster to value, implicit learning
- **Effort**: 3-4 weeks (70% code reuse from V2.0)
- **Risk**: Moderate (significant refactor but architecturally sound)
- **Impact**: Makes Moly feel like a real coach, not a tool

**This is the right direction.** Users will immediately prefer chat to suggestion-picking.

