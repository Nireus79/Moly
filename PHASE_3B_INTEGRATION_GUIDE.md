# Phase 3b: Message Processor Integration Guide

**Status:** Ready to implement  
**Effort:** 4-5 hours  
**Date:** October 8, 2026

---

## PHASE 3B OVERVIEW

Phase 3b integrates the clarification response handling into the main message processor. This is where:
1. User responds to clarification question
2. System detects the response
3. Contacts are resolved and updated
4. Original message is re-processed with resolved contacts
5. Final response is generated

---

## KEY INTEGRATION POINTS

### 1. Message Processor (main.go)

**Location:** `/home/nireus79/vs_projects/Moly/Moly/moly-go/main.go` around line 2076

**Current flow:**
```
messageProcessor:
  ├─ Build AnalysisContext
  ├─ Call orchestrator.ProcessMessage()
  └─ Generate response
```

**NEW flow (Phase 3b):**
```
messageProcessor:
  ├─ CHECK: Is this a clarification response?
  │  └─ If yes:
  │     ├─ Load clarification from DB
  │     ├─ Process user answer with ClarificationHandler
  │     ├─ Update contacts
  │     ├─ Re-invoke orchestrator with SAME original message
  │     └─ Continue to response generation
  │
  ├─ Build AnalysisContext
  ├─ Call orchestrator.ProcessMessage()
  └─ Generate response
```

**What to look for:**
- Check if `req.ClarificationResponse` or similar exists
- Add clarification detection before normal processing
- Load original message from clarification record
- Process answer with `ClarificationHandler`
- Update `analysisCtx` with resolved contacts
- Re-invoke orchestrator with updated context

---

### 2. ClarificationResponse Detection

**Pattern to detect:**
```go
// In message processor
if hasClarificationMetadata(req) {
    // This is a clarification response
    clarificationID := extractClarificationID(req)
    answer := extractAnswer(userMessage)
    
    // Process with handler
    handler := agents.NewClarificationHandler(db)
    updatedContacts, err := handler.ProcessResponse(
        userID,
        conversationID,
        clarificationID,
        userMessage,
        previousContacts, // from loading clarification
    )
    
    // Update analysisCtx with resolved contacts
    analysisCtx.ResolvedContacts = updatedContacts
}
```

**Implementation checklist:**
- [ ] Add `ClarificationResponse` field to ConversationRequest struct
- [ ] Detect if request contains clarification metadata
- [ ] Load clarification from database
- [ ] Extract user's answer
- [ ] Call ClarificationHandler.ProcessResponse()
- [ ] Update contacts in database
- [ ] Build updated AnalysisContext
- [ ] Re-invoke orchestrator
- [ ] Generate response

---

### 3. Contact Database Updates

**What to save after clarification:**
```go
// Update each contact's metadata
for _, contact := range updatedContacts {
    contact.Status = "confirmed"
    contact.Confidence = 0.99
    contact.LastMentionedAt = time.Now().Unix()
    
    err := contactRepo.Update(contact)
    if err != nil {
        log.Printf("ERROR updating contact: %v", err)
    }
}

// Also update clarification record
clarification.UserAnswer = answer
clarification.Status = "answered"
clarification.AnsweredAt = time.Now().Unix()
clarificationRepo.Update(clarification)
```

---

### 4. Re-orchestrator Invocation

**Key difference from normal flow:**
```go
// Load original message from clarification
originalMessage := clarification.OriginalMessage

// Re-invoke orchestrator with:
// - Same original message
// - Updated contacts in analysisCtx
// - Fresh message ID (optional, for tracking)

layerCtx, err := orchestrator.ProcessMessage(
    context.Background(),
    originalMessage,  // SAME message, not clarification response
    userID,
    conversationID,
    newMessageID,
    analysisCtxWithResolvedContacts,  // UPDATED contacts
    maturityContext,
)
```

**Critical:** Use ORIGINAL message, not the clarification response answer!

---

## INTEGRATION WORKFLOW

### Step 1: Detect Clarification Response

```go
func isClarificationResponse(req *ConversationRequest) bool {
    return req.ClarificationResponse != nil &&
           req.ClarificationResponse.ClarificationID != ""
}
```

### Step 2: Load Clarification Context

```go
clarificationRepo := database.NewClarificationQuestionRepository(db)
clarification, err := clarificationRepo.GetQuestion(clarificationID)

if err != nil || clarification == nil {
    log.Printf("Clarification not found: %s", clarificationID)
    // Handle gracefully
}

// Get original message
originalMessage := clarification.OriginalMessage
```

### Step 3: Process Response

```go
handler := agents.NewClarificationHandler(db)
updatedContacts, err := handler.ProcessResponse(
    userID,
    conversationID,
    clarificationID,
    userMessage,  // The clarification answer
    clarification.Options, // Available contact options
)

if err != nil {
    log.Printf("ERROR processing clarification: %v", err)
    // Return error response
}
```

### Step 4: Update Database

```go
// Save updated contacts
for _, contact := range updatedContacts {
    contact.Status = "confirmed"
    contact.Confidence = 0.99
    err := contactRepo.Update(contact)
    if err != nil {
        log.Printf("WARNING: Failed to update contact: %v", err)
    }
}

// Update clarification record
clarification.UserAnswer = extractAnswer(userMessage)
clarification.Status = "answered"
clarificationRepo.MarkAnswered(clarificationID)
```

### Step 5: Re-invoke Orchestrator

```go
// Rebuild analysisCtx with resolved contacts
analysisCtx.RelevantContacts = updatedContacts

// Re-invoke orchestrator
layerCtx, err := orchestrator.ProcessMessage(
    context.Background(),
    originalMessage,  // NOT the clarification response
    userID,
    conversationID,
    newMessageID,
    analysisCtx,  // Updated with resolved contacts
    maturityContext,
)
```

### Step 6: Generate Response

```go
// Normal response generation continues
response := GenerateResponse(layerCtx)
response.Message = "Got it, so your girlfriend is different... " +
                   "[continue with normal analysis]"

return response
```

---

## CODE LOCATIONS

| Component | File | Lines | Status |
|-----------|------|-------|--------|
| Contact detection | agents/contact_detector.go | All | ✅ Done |
| Confidence scoring | agents/contact_confidence.go | All | ✅ Done |
| Clarification handler | agents/clarification_handler.go | All | ✅ Done |
| Message processor | main.go | ~2050-2150 | 🔲 TODO |
| Orchestrator integration | agents/unified_orchestrator.go | ~313-350 | ✅ Done |
| LayerContext | tools/layer_context.go | ~110-125 | ✅ Done |
| Contact repository | database/contact_repository.go | All | ✅ Done |
| Clarification repository | database/clarification_repository.go | All | ✅ Done |

---

## TESTING PHASE 3B

After implementing, test with:

```
Turn 1: User sends ambiguous message
  Message: "She likes BDSM. She's different from him."
  Expected: Clarification modal
    "You mentioned girlfriend and colleague.
     When you say 'she's different', do you mean:
     A) Your girlfriend
     B) Your colleague?"

Turn 2: User responds
  Message: "A) My girlfriend"
  Expected: Full response analyzed with resolved contacts
    "Got it, so your girlfriend is different from your colleague..."
```

---

## REMAINING PHASES

### Phase 4: Progressive Naming (2-3 hours)
- Detect "name is X" patterns
- Update unnamed contacts
- Link pronouns to new names
- Database updates

### Phase 5: Response Generation (4-6 hours)
- Format clarification responses
- Handle normal responses with contact context
- Update extraction prompt to use contact context

### Phase 6: Testing & Integration (4-6 hours)
- End-to-end tests
- Real conversation patterns
- Edge case handling
- Performance validation

---

## CRITICAL NOTES

1. **Use ORIGINAL message, not clarification answer**
   - The clarification response is metadata only
   - Re-analyze the original user message with resolved contacts

2. **Save contact updates BEFORE re-orchestration**
   - Database state must be current
   - Orchestrator uses updated contact list

3. **Handle missing clarifications gracefully**
   - Clarifications may expire
   - Fall back to normal processing

4. **Preserve conversation context**
   - Don't lose previous messages/context
   - Build on accumulated understanding

5. **Confidence must be high after clarification**
   - Set to 0.99 after user confirms
   - Prevents re-asking same question

---

## EXPECTED OUTCOME

After Phase 3b completion:

✅ Users can answer clarification questions inline
✅ Contacts are resolved and confirmed
✅ Original messages processed with resolved context
✅ No false contradictions from ambiguous contacts
✅ Natural conversation flow maintained

---

## IMPLEMENTATION TIPS

1. **Start with detection logic** - Make sure clarification responses are reliably detected
2. **Test database updates** - Verify contacts save correctly after response
3. **Verify re-orchestration** - Original message should process cleanly
4. **Check confidence scores** - Should prevent re-asking same clarification
5. **Handle edge cases** - Expired clarifications, invalid answers, etc.

---

## WHAT'S WIRED AND READY

✅ Pre-Layer-1 contact workflow (Phase 2)  
✅ Clarification detection in orchestrator (Phase 2)  
✅ Clarification handler foundation (Phase 3a)  
✅ Contact repository operations (Phase 1)  
✅ LayerContext with clarification fields (Phase 2)  
✅ Confidence scoring system (Phase 2)  

**All pieces are in place for Phase 3b integration.**

---

## SUMMARY

Phase 3b is the glue that ties everything together. It detects when users answer clarification questions, processes their answers, updates the contact database, and re-runs the analysis with resolved contacts.

The implementation is straightforward:
1. Detect clarification response
2. Load saved clarification
3. Process answer with handler
4. Update database
5. Re-invoke orchestrator
6. Generate response

All supporting components are built and tested. Ready to integrate!

