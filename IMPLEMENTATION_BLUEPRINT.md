# Contact Workflow Implementation Blueprint

**Status:** Ready for Implementation  
**Architecture:** Inline/Synchronous, Natural Conversation Pattern  
**Effort:** 18-26 hours  
**Date:** October 8, 2026

---

## MASTER SUMMARY

**What we're building:** A contact management system that mirrors natural conversation flow.

**How it works:**
1. User mentions contacts (named or unnamed)
2. System detects and tracks them
3. System resolves pronouns intelligently
4. When ambiguous (< 0.50 confidence): **Interrupt and ask**
5. User answers clarification **inline**
6. System updates database and **resumes**
7. All layers get clean contact data

**Key difference from modals:** No separate UI. Ask → Answer → Resume, all in natural conversation flow.

---

## ARCHITECTURE AT A GLANCE

```
Turn 1: User sends ambiguous message
  ↓
Orchestrator detects contacts
  ↓
Contact Workflow runs (before Layer 1)
  ├─ Detect contacts
  ├─ Resolve pronouns
  └─ Check confidence
  ↓
If ambiguous (< 0.50):
  ├─ Return clarification question
  └─ STOP (don't run layers yet)
  
Response: Ask question with options
[Save to database]

---

Turn 2: User responds with answer
  ↓
Detect: This is clarification response
  ↓
Load original message from DB
  ↓
Update contacts with answer
  ↓
Re-invoke orchestrator with SAME message (now with resolved contacts)
  ├─ Contact Workflow: ✓ No ambiguity
  ├─ Wire to LayerContext
  └─ Run Layers 1-11 (CLEAN DATA)
  
Response: Full Moly response
(All layers have clean contact context)
```

---

## EXISTING INFRASTRUCTURE (80% Built)

### ✅ Already Have

**Contact Management:**
- Contact model (in models/conversation_types.go)
- ContactRepository (CRUD, mention tracking)
- Database schema with contacts table

**Contact Deduplication:**
- ContactDeduplicator (generic→specific naming)
- Merge logic (generic Kyle → specific Kyle)
- Similarity scoring

**Pronoun Resolution:**
- PronounResolver (detect + resolve)
- PronounReference struct
- PronounResolution struct
- Database schema (migration 032)
- Scope tracking (when valid from/to)

**Subject Tracking:**
- ExtractedEntity.Subject field
- ClarificationContext structure

**Orchestrator:**
- Unified 11-layer pipeline
- Existing clarification detection (line 147-170)
- Layer context wiring (line 211-286)

### ❌ Need to Add

1. **Unnamed Contact Support** (LOW: 2-3h)
   - Make Contact.Name nullable
   - Generate placeholder names

2. **Pronouns Field** (LOW: 1-2h)
   - Add Pronouns []string to Contact
   - Database migration

3. **Contact Workflow Layer** (MEDIUM: 4-6h)
   - New ContactDetector
   - Integrate into orchestrator
   - Implement confidence scoring

4. **Clarification Handling** (MEDIUM: 6-8h)
   - Detect clarification_response in request
   - Load context from database
   - Re-invoke orchestrator
   - Update contacts from answer

5. **Progressive Naming** (LOW-MEDIUM: 2-3h)
   - Detect "name is X" patterns
   - Rename contacts
   - Update pronouns

6. **Response Generation** (MEDIUM: 4-6h)
   - Format clarification responses
   - Handle ClarificationNeeded flag
   - Generate normal responses

7. **Testing** (MEDIUM: 4-6h)
   - Integration tests
   - Multi-contact scenarios
   - Edge cases

---

## IMPLEMENTATION PHASES

### Phase 1: Schema & Model Updates (2-3 hours)

**Files:** 
- `models/conversation_types.go`
- `database/contact_repository.go`
- `database/migrations/*.sql`

**Changes:**

1. Make Contact.Name nullable:
```go
type Contact struct {
    // ... existing fields ...
    Name *string  // ← Changed from string
    Pronouns []string  // ← NEW
}
```

2. Database migration:
```sql
ALTER TABLE contacts 
ADD COLUMN pronouns TEXT;  -- JSON array

ALTER TABLE contacts 
MODIFY COLUMN name TEXT;  -- Allow NULL
```

3. Helper for placeholder names:
```go
func GeneratePlaceholderName(contactType string) string {
    // "Unknown Female 1", "Unnamed Colleague", etc.
}
```

**Checklist:**
- [ ] Update Contact struct (Name nullable, add Pronouns)
- [ ] Create migration for pronouns column
- [ ] Create migration to allow NULL on name
- [ ] Update ContactRepository Save() to handle nullable Name
- [ ] Update ContactRepository Update() for Pronouns
- [ ] Add GeneratePlaceholderName() helper
- [ ] Test with NULL names in database

---

### Phase 2: Contact Workflow Integration (4-6 hours)

**Files:**
- New: `agents/contact_detector.go`
- Modified: `agents/unified_orchestrator.go`
- Modified: `tools/layer_context.go`

**Create ContactDetector:**

```go
type ContactDetector struct {
    db *database.Database
}

func (cd *ContactDetector) DetectInMessage(
    message string,
    analysisCtx *models.AnalysisContext,
) []models.Contact {
    // 1. Look for explicit mentions ("girl", "colleague", "boyfriend")
    // 2. Look for pronouns (she, he, they)
    // 3. Look for named references (Emily, Marcus)
    // 4. Return detected contacts
}
```

**Update UnifiedOrchestrator.ProcessMessage:**

After line 206 (after IsMessageOne setup), add:

```go
// === NEW: Contact Workflow ===
contactDetector := agents.NewContactDetector(uo.db)
detectedContacts := contactDetector.DetectInMessage(message, analysisCtx)

var activeContacts []models.Contact
pronounResolver := tools.NewPronounResolver(uo.db)

for _, contact := range detectedContacts {
    // Resolve pronouns using existing tool
    resolutions := pronounResolver.ResolveAntecedent(
        contact.Pronouns,
        detectedContacts,
    )
    
    // Calculate confidence
    confidence := CalculateConfidence(contact, resolutions)
    
    // If low confidence, stop and ask
    if confidence < 0.50 {
        lc.ClarificationNeeded = true
        lc.ClarificationID = generateID()
        lc.ClarificationQuestion = buildQuestion(contact, resolutions)
        lc.ClarificationOptions = buildOptions(resolutions)
        lc.ClarificationConfidence = confidence
        return lc, nil  // Return early
    }
    
    activeContacts = append(activeContacts, contact)
}

// Wire to LayerContext
lc.ActiveContacts = activeContacts
lc.ContactContext = buildContactContextString(activeContacts)
```

**Update LayerContext:**

```go
type LayerContext struct {
    // ... existing fields ...
    
    // NEW: Clarification state
    ClarificationNeeded bool
    ClarificationID string
    ClarificationQuestion string
    ClarificationOptions []string
    ClarificationConfidence float64
    
    // NEW: Contact data
    ActiveContacts []models.Contact
    PronounResolutions map[string]models.PronounResolution
    ContactContext string
}
```

**Implement Confidence Scoring:**

```go
func CalculateConfidence(contact models.Contact, resolutions []models.PronounResolution) float64 {
    // Base score
    score := 0.5
    
    // Adjust based on:
    // - Number of possible contacts (fewer = higher confidence)
    // - Pronoun-name agreement
    // - Recency of mention
    // - Explicit type matching
    
    // Return threshold: >= 0.80 (high), 0.50-0.80 (medium), < 0.50 (low)
    return score
}
```

**Checklist:**
- [ ] Create ContactDetector
- [ ] Implement DetectInMessage()
- [ ] Update LayerContext struct
- [ ] Add ClarificationNeeded fields to LayerContext
- [ ] Add ActiveContacts, ContactContext to LayerContext
- [ ] Integrate into UnifiedOrchestrator.ProcessMessage
- [ ] Implement CalculateConfidence()
- [ ] Test contact detection
- [ ] Test confidence calculation

---

### Phase 3: Clarification Request/Response Handling (6-8 hours)

**Files:**
- Modified: `main.go` (MessageProcessor)
- New: `database/clarification_repository.go`
- New: `database/migrations/*.sql`

**Create Clarification Table:**

```sql
CREATE TABLE clarifications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    original_message TEXT NOT NULL,
    message_sequence INT,
    clarification_type TEXT,
    question TEXT NOT NULL,
    options TEXT,  -- JSON
    detected_ambiguities TEXT,  -- JSON
    confidence REAL,
    user_answer TEXT,
    answer_message TEXT,
    status TEXT,  -- pending, answered, expired
    created_at INTEGER,
    answered_at INTEGER,
    expires_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);
```

**Update MessageProcessor (main.go ~line 2076):**

```go
// Check if this is a clarification response
if req.ClarificationResponse != nil {
    // Load clarification context
    clrCtx := clarificationRepo.GetByID(req.ClarificationResponse.ClarificationID)
    
    // Update contacts from answer
    UpdateContactsFromClarification(
        clrCtx,
        req.ClarificationResponse.Answer,
        analysisCtx,
    )
    
    // Save clarification answer
    clrCtx.UserAnswer = req.ClarificationResponse.Answer
    clrCtx.AnsweredAt = time.Now().Unix()
    clrCtx.Status = "answered"
    clarificationRepo.Save(clrCtx)
    
    // Re-invoke orchestrator with SAME message
    layerCtx, _ := orchestrator.ProcessMessage(
        ctx,
        clrCtx.OriginalMessage,  // Same message
        userID,
        conversationID,
        newMessageID,
        analysisCtxWithUpdatedContacts,  // Updated contacts
        maturityContext,
    )
} else {
    // Normal message processing
    layerCtx, _ = orchestrator.ProcessMessage(...)
}

// Check if orchestrator flagged clarification needed
if layerCtx != nil && layerCtx.ClarificationNeeded {
    // Save clarification for next turn
    clarification := &models.Clarification{
        ID: layerCtx.ClarificationID,
        UserID: userID,
        ConversationID: conversationID,
        OriginalMessage: userMessageForDB,
        MessageSequence: len(conversationHistory),
        ClarificationType: layerCtx.ClarificationFlag,
        Question: layerCtx.ClarificationQuestion,
        Options: layerCtx.ClarificationOptions,
        Confidence: layerCtx.ClarificationConfidence,
        Status: "pending",
        CreatedAt: time.Now().Unix(),
        ExpiresAt: time.Now().Unix() + (24 * 3600),  // 24 hour expiry
    }
    clarificationRepo.Save(clarification)
    
    // Return clarification response (not normal response)
    return &Response{
        Status: "clarification_needed",
        Clarification: layerCtx.ClarificationQuestion,
        Options: layerCtx.ClarificationOptions,
        ClarificationID: layerCtx.ClarificationID,
    }
}

// Normal response generation
response := GenerateResponse(layerCtx)
return &Response{
    Status: "success",
    Message: response,
}
```

**Checklist:**
- [ ] Create clarifications table (migration)
- [ ] Create ClarificationRepository
- [ ] Implement Save(), GetByID(), GetPending()
- [ ] Update MessageProcessor to detect clarification_response
- [ ] Implement UpdateContactsFromClarification()
- [ ] Update contacts in DB from clarification answer
- [ ] Re-invoke orchestrator with updated context
- [ ] Save clarification to database
- [ ] Return correct response format
- [ ] Test clarification flow

---

### Phase 4: Progressive Naming (2-3 hours)

**Files:**
- Modified: `agents/unified_orchestrator.go` (in contact workflow section)
- Modified: `database/contact_repository.go`

**Add to Contact Workflow:**

After resolving pronouns, check for naming patterns:

```go
// After contact resolution, check for name updates
nameUpdates := detectNamingPatterns(message, activeContacts)

for contactID, newName := range nameUpdates {
    // Find the contact
    contact := findContactByID(activeContacts, contactID)
    if contact != nil {
        // Update name
        contact.Name = &newName
        contact.Status = "named"
        
        // Save to database
        contactRepo.Update(contact)
        
        // Update all pronoun resolutions for this contact
        pronounResolver.UpdateResolutionsForContact(contactID, newName)
    }
}
```

**Implement Pattern Detection:**

```go
func detectNamingPatterns(message string, activeContacts []models.Contact) map[int64]string {
    // Look for patterns like:
    // - "Her name is X"
    // - "My girlfriend is Emily"
    // - "The colleague's name is Marcus"
    
    // Match pronouns/descriptions to active contacts
    // Return map of contact ID → new name
}
```

**Checklist:**
- [ ] Implement detectNamingPatterns()
- [ ] Detect "name is X" patterns
- [ ] Link pronouns to contacts
- [ ] Update contact names
- [ ] Update pronoun resolutions with names
- [ ] Preserve all characteristics during rename
- [ ] Test progressive naming with real messages

---

### Phase 5: Response Generation Updates (4-6 hours)

**Files:**
- Modified: Response generation logic (TBD based on current architecture)

**Handle Clarification Response:**

When `layerCtx.ClarificationNeeded == true`:

```
Format response as:
{
  "status": "clarification_needed",
  "clarification": {
    "id": "clr_xyz123",
    "question": "When you say 'she', do you mean:",
    "options": [
      "A) Your girlfriend",
      "B) Your colleague",
      "C) Someone else?"
    ],
    "context": {
      "detected_contacts": [...],
      "confidence": 0.40
    }
  }
}
```

When `layerCtx.ClarificationNeeded == false`:

```
Format as normal response with all layers' results
```

**Checklist:**
- [ ] Format clarification responses
- [ ] Include options with A/B/C format
- [ ] Include clarification_id for next turn
- [ ] Generate normal responses for resolved contacts
- [ ] Test response formatting

---

### Phase 6: Testing & Validation (4-6 hours)

**Test Scenarios:**

1. **Single Unnamed Contact**
   - M1: "I wanna talk about a girl"
   - M2: "She likes..." (should resolve to girl)
   - M3: "Her name is Sarah" (should rename)

2. **Multiple Unnamed Contacts with Ambiguity**
   - M1: "I'm dating this girl"
   - M2: "I work with a colleague"
   - M3: "She's dominant" (ambiguous → ask)
   - User: "A) The colleague"
   - Verify: Correct contact updated

3. **Named Contact from Start**
   - M1: "My girlfriend Emily"
   - M2: "She likes..." (should resolve to Emily)
   - No clarification needed

4. **Progressive Naming Reversal**
   - M1: "This woman I know"
   - M2: "She's dominant"
   - M3: "Her name is Sarah"
   - M4: "Sarah said..." (should work)

5. **Group References**
   - M1: "Girl and colleague"
   - M2: "Both of them..."
   - Detect plural, handle group

**Checklist:**
- [ ] Unit tests for ContactDetector
- [ ] Unit tests for CalculateConfidence
- [ ] Unit tests for detectNamingPatterns
- [ ] Integration test: Unnamed → Named flow
- [ ] Integration test: Clarification → Resolution flow
- [ ] Integration test: Multiple unnamed contacts
- [ ] Integration test: Progressive naming
- [ ] End-to-end test: Real conversations
- [ ] Verify no false contradictions in Layer 5

---

## CRITICAL IMPLEMENTATION NOTES

### 1. Database Consistency
- Always update contacts in DB **before** re-invoking orchestrator
- Clarification answers must be saved synchronously
- Don't lose intermediate state

### 2. Context Preservation
- Load full AnalysisContext after clarification
- Include updated contacts in reprocessing
- Don't lose accumulated entities from prior messages

### 3. Confidence Scoring
- < 0.50: MUST ask (don't guess)
- 0.50-0.80: Can assume (optional ask)
- >= 0.80: Assume (high confidence)

### 4. Error Handling
- Clarification expires after 24 hours
- If contact update fails, don't invoke orchestrator
- Log all clarifications for audit trail
- Graceful degradation if contact resolution fails

### 5. Performance
- Contact detection should be fast (regex patterns)
- Pronounce resolution uses existing tools (already optimized)
- No additional LLM calls in critical path
- Save clarification asynchronously if possible

---

## EXPECTED OUTCOMES

After implementation:

✅ **No false contradictions** - Contact ambiguity resolved before Layer 5  
✅ **Correct subject attribution** - "She" links to specific contact  
✅ **Progressive naming** - Names accepted anytime, not upfront  
✅ **Natural flow** - Mirrors human conversation (interrupt → clarify → resume)  
✅ **Unnamed contacts** - Tracked throughout conversation  
✅ **Pronoun resolution** - High confidence links to correct contacts  
✅ **Database audit trail** - All clarifications tracked  

---

## FILES TO MODIFY - QUICK REFERENCE

| File | Changes | Priority | Hours |
|------|---------|----------|-------|
| models/conversation_types.go | Name nullable, add Pronouns | High | 1 |
| database/contact_repository.go | Handle nullable Name, update Pronouns | High | 1 |
| database/migrations/*.sql | Schema changes | High | 1 |
| agents/contact_detector.go | NEW file | High | 2-3 |
| agents/unified_orchestrator.go | Add contact workflow | High | 3-4 |
| tools/layer_context.go | Add clarification fields | Medium | 1 |
| main.go | Add clarification handling | Medium | 3-4 |
| database/clarification_repository.go | NEW file | Medium | 2 |
| database/migrations/*.sql | Clarifications table | Medium | 1 |
| Response generation | Handle clarification flag | Medium | 2-3 |
| Tests | All integration tests | Medium | 4-6 |

**Total Effort: 18-26 hours**

---

## NEXT STEP

When ready to implement:

1. **Start with Phase 1** (Schema updates - lowest risk)
2. **Then Phase 2** (Contact workflow layer)
3. **Then Phase 3** (Clarification handling)
4. **Then Phase 4** (Progressive naming)
5. **Then Phases 5-6** (Response + testing)

Each phase is relatively independent and can be tested before moving to the next.

---

## RELATED DOCUMENTS

- **CONTACT_WORKFLOW_ARCHITECTURE.md** - Detailed workflow design
- **INFRASTRUCTURE_AUDIT.md** - What exists vs what's needed
- **WORKFLOW_INTEGRATION_ANALYSIS.md** - Why inline + blocking is best
- **INLINE_CLARIFICATION_FLOW.md** - Complete technical flow

