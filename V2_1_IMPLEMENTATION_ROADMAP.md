# V2.1 Implementation Roadmap

**Status**: Ready to start  
**Target Start**: Immediately after this approval  
**Timeline**: 3-4 weeks (35-40 hours)  
**Approach**: Incremental with daily testing

---

## Phase 1: Core Chat (Weeks 1-2)
**Milestone**: Users can log in with code and chat with Moly (encrypted)

### Week 1: Foundation (Auth + Encryption)

**Day 1-2: Database Encryption Setup**
- [ ] Add SQLCipher to go.mod: `github.com/mutecomm/go-sqlcipher v4.4.0`
- [ ] Create `database/encryption.go`:
  - `DeriveKey(userID string) [32]byte` - SHA256(userID + hardcoded salt)
  - `OpenEncrypted(path, userID string) (*sql.DB, error)` - SQLCipher wrapper
- [ ] Update `database.Init()` to use encrypted connection
- [ ] Verify: `go test ./database -v -run "TestEncryption"`
  - Create db, write data, close, reopen with wrong key → should fail
  - Reopen with right key → data readable

**Day 3: Auth Code System**
- [ ] Create `auth/codes.go`:
  - `GenerateCode() string` - format: "moly-XXXXX-YYYYY" (5+5 hex)
  - `ValidateCode(code string) bool`
  - Code expires in 1 hour
- [ ] Create `auth/sessions.go`:
  - `Session` struct: id, userId, code, createdAt, expiresAt, lastActive
  - `SessionRepository` in database layer
  - `CreateSession(code string) (*Session, error)`
  - `ValidateSession(sessionId string) (*Session, error)`
  - `RefreshSession(sessionId string) error`

**Day 4: Auth Handlers**
- [ ] Create `v2_1_handlers.go` (new file):
  - `POST /api/v2.1/auth/generate` → returns code
  - `POST /api/v2.1/auth/login` → validates code, returns sessionId
  - `GET /api/v2.1/auth/verify` → checks sessionId validity
  - `POST /api/v2.1/auth/logout` → invalidates sessionId
- [ ] Add session middleware: check `Authorization: Bearer {sessionId}` header
- [ ] Verify: Integration tests for all 4 endpoints

**Day 5: Database Schema for V2.1**
- [ ] Create migration: Add tables:
  - `sessions` (id, user_id, code, created_at, expires_at, last_active)
  - `chat_messages` (id, user_id, conversation_id, role, content, created_at, encrypted)
  - `implicit_learning` (id, user_id, message_id, field, value, confidence, created_at, approved)
- [ ] Add columns to existing tables:
  - `about_me`: extracted_count, total_confidence, last_learned_at
  - `contacts`: tenure, extraction_confidence, last_mentioned, mention_count
- [ ] Verify: `sqlite3 moly.db ".schema"` shows all tables

**Day 5 EOD: Week 1 Gate**
- [ ] `go test ./... -v -timeout 30s` (all pass)
- [ ] No new TODOs in code
- [ ] Database encrypted (verify with file inspection tool)
- [ ] 4 auth endpoints working with mock client

---

### Week 2: Chat Implementation

**Day 6: Chat Agent Refactor**
- [ ] Rename `ConversationAgent.Run()` → `ConversationAgent.RunSuggestions()`
- [ ] Create `ConversationAgent.RunChat()`:
  - Takes: `ChatRequest {message, conversationId, sessionId, userId}`
  - Returns: `ChatResponse {response, contactMention, aboutMeGaps, contextLearned}`
  - Uses existing LLM client (Ollama/Claude/Heuristic)
  - Calls `LearningAgent` to extract AboutMe/Contact from message
  - Calls `RiskMonitor` for safety checking
  - Returns natural language response (not suggestions)
- [ ] Create `models/chat.go`:
  ```go
  type ChatRequest struct {
    Message        string `json:"message"`
    ConversationId string `json:"conversationId"`
  }
  type ChatResponse struct {
    MessageId      string                 `json:"messageId"`
    Response       string                 `json:"response"`
    ContactMention *ContactMentionDetected `json:"contactMention,omitempty"`
    AboutMeGaps    []string               `json:"aboutMeGaps,omitempty"`
    ContextLearned map[string]interface{} `json:"contextLearned,omitempty"`
  }
  ```

**Day 7: Chat Endpoint**
- [ ] Create `POST /api/v2.1/chat` handler:
  - Validate session
  - Call `ConversationAgent.RunChat()`
  - Store message in `chat_messages` table (user + assistant)
  - Return ChatResponse
- [ ] Verify: Integration test with mock LLM
  - Send message → get response → verify response stored

**Day 8: Chat Message Storage**
- [ ] Create `ChatMessageRepository`:
  - `Save(msg *ChatMessage) error`
  - `GetHistory(userId, conversationId string, limit int) ([]*ChatMessage, error)`
  - `Delete(messageId string) error`
- [ ] Add indexing for fast history retrieval: `CREATE INDEX idx_chat_user_conv ON chat_messages(user_id, conversation_id, created_at DESC);`

**Day 9: Extension UI Updates (Phase 1 subset)**
- [ ] Update `popup.html` to show:
  - Login screen (show code, "Log In" button) if not authenticated
  - Chat interface if authenticated (list of messages, input field, "Send" button)
- [ ] Update `popup.js`:
  - `showLoginScreen()` - display code, handle login click
  - `showChatScreen()` - display messages, handle send click
  - `sendMessage()` - POST to `/api/v2.1/chat`, show response
  - Store sessionId in chrome.storage after login
- [ ] Update `background.js`:
  - New handler: `handleChat()` - forward to backend
  - New handler: `handleAuthLogin()` - POST /api/v2.1/auth/login
  - New handler: `handleAuthGenerate()` - GET /api/v2.1/auth/generate

**Day 10: Integration Testing**
- [ ] E2E test: Login flow
  - Generate code → Login with code → Verify sessionId stored
- [ ] E2E test: Send message
  - Login → Send "Hello" → Receive response → Verify stored in DB
- [ ] E2E test: Session persistence
  - Login → Close browser → Reopen → Should still have sessionId
- [ ] Performance test
  - Send 5 concurrent messages from 2 users → All stored correctly
- [ ] Encryption test
  - Send message → Verify database is encrypted (not readable as plaintext)

**Day 10 EOD: Phase 1 Gate**
- [ ] `go test ./... -v -timeout 30s` (all pass, +15 new tests)
- [ ] Extension loads without errors
- [ ] Manual smoke test: Login → Chat → Receive response
- [ ] Database encrypted with AES-256
- [ ] All messages stored in DB
- [ ] No console errors in extension

---

## Phase 2: Smart Context (Week 3)
**Milestone**: Moly learns naturally, detects contact mentions, asks Socratic questions

### Implementation Order
1. Contact mention detection (keyword + LLM)
2. Implicit AboutMe extraction 
3. Confidence scoring
4. Socratic question embedding
5. Learning UI indicators

**Key Files to Create**:
- `agents/contact_detector.go` - Detect "boss", "colleague", "Sarah", etc.
- `agents/context_extractor.go` - Extract AboutMe fields from chat
- `models/learning.go` - Learning confidence models

---

## Phase 3: Polish (Week 4)
**Milestone**: Production-ready, comprehensive testing

### Implementation Order
1. AboutMe/Contact viewers
2. Learning indicators in UI
3. Chat history viewer
4. Error handling improvements
5. Full test coverage

---

## Testing Strategy

### Unit Tests (Per Day)
```bash
go test ./... -v -timeout 30s
```
Required: 85%+ coverage on new code

### Integration Tests (Per Phase)
```bash
go test ./... -run Integration -v -timeout 60s
```
Test: Database, LLM, API together

### E2E Tests (Per Week)
```bash
go test ./... -run E2E -v -timeout 180s
```
Test: Full workflow with real components

### Manual Tests (Before Gate)
- [ ] Login with code
- [ ] Send message
- [ ] Receive response
- [ ] Check database encrypted
- [ ] Close/reopen → still logged in
- [ ] Try wrong session → 401 error

---

## Blockers & Dependencies

### None for Phase 1
- Encryption: SQLCipher is stable
- Auth: No external deps
- LLM: Already working from V2.0
- Database: Schema prepared

### Phase 2 Dependencies
- Phase 1 must be complete (auth + chat working)
- LLM provider must be available (Ollama/Claude/Heuristic)

### Phase 3 Dependencies
- Phase 2 must be complete (learning working)
- Extension UI framework (already have popup.html structure)

---

## Success Metrics

### Phase 1 Complete When
- [ ] 15+ new integration tests passing
- [ ] Login → Chat → Response working E2E
- [ ] Database encrypted (SQLCipher verified)
- [ ] Extension populates with chat UI
- [ ] No crashes on extended chat session (10+ messages)
- [ ] Session persists across browser restart

### Phase 2 Complete When
- [ ] Contact mention detected in 80%+ of messages mentioning people
- [ ] AboutMe extracted with 70%+ confidence
- [ ] Socratic questions embedded naturally (not forced)
- [ ] "Save contact?" suggestion appears when appropriate
- [ ] User can approve/reject learned AboutMe

### Phase 3 Complete When
- [ ] 30+ new tests passing
- [ ] Chat history viewer works (search, sort, delete)
- [ ] Learning indicators visible ("I've learned about your boss", etc.)
- [ ] 100% test coverage on new code
- [ ] Zero console errors
- [ ] Works with Ollama + heuristic fallback
- [ ] Performance: <500ms for chat response (excluding LLM)

---

## File Checklist

### Phase 1 Files to Create
- [ ] `database/encryption.go` (100 lines)
- [ ] `auth/codes.go` (60 lines)
- [ ] `auth/sessions.go` (150 lines)
- [ ] `v2_1_handlers.go` (300 lines)
- [ ] `models/chat.go` (50 lines)
- [ ] `repositories/chat_message_repo.go` (100 lines)
- [ ] `database/migrations.sql` (50 lines)
- [ ] `e2e_chat_test.go` (400 lines)

### Phase 1 Files to Modify
- [ ] `database/database.go` - Add encryption init
- [ ] `database.Init()` - Call encrypted connection
- [ ] `agents/conversation_agent.go` - Add RunChat() method
- [ ] `moly-extension/popup.html` - Chat UI
- [ ] `moly-extension/popup.js` - Chat logic
- [ ] `moly-extension/background.js` - Auth handlers

### Phase 1 Estimated LOC
- New: ~1200 lines
- Modified: ~400 lines
- Tests: ~400 lines
- **Total**: ~2000 lines (manageable)

---

## How to Start

1. **Today**: Run through database encryption setup (Day 1-2)
   ```bash
   go get github.com/mutecomm/go-sqlcipher/v4
   # Create database/encryption.go
   # Test: go test ./database -v -run Encryption
   ```

2. **Tomorrow**: Build auth system (Day 3-4)
   ```bash
   # Create auth/ package
   # Test: go test ./auth -v
   ```

3. **This week**: Build chat handlers (Day 5-6)
   ```bash
   # Create v2_1_handlers.go
   # Test: go test ./... -v -run Chat
   ```

4. **Next week**: Finish integration (Day 7-10)
   ```bash
   # Update extension
   # Run full E2E
   ```

---

## Questions Before Starting?

- SQLCipher version? (suggest v4.4.0+)
- Key rotation needed? (no, static per user is fine for V2.1)
- Session timeout? (suggest 24 hours for local use)
- Backward compatibility with V2.0? (no, this is V2.1 only)

**Ready to start Phase 1, Day 1?**

