# Inline/Synchronous Clarification Flow - Natural Conversation Design

**Architecture:** Clarification + Answer in same turn, then resume processing  
**Philosophy:** Mirror natural human conversation (interrupt → clarify → resume)

---

## THE NATURAL CONVERSATION PATTERN

```
User: "She's different from him. We both like BDSM..."

Moly (interrupts): "Wait, I need to clarify - you mentioned:
                   - A girlfriend (unnamed)
                   - A colleague (unnamed)
                   
                   When you say 'she's different', do you mean:
                   A) Your girlfriend
                   B) Your colleague
                   C) Someone else?"

User (responds): "A) My girlfriend"

Moly (resumes): "Got it, so your girlfriend is different from your colleague...
                [continues analysis with resolved contacts]"
```

---

## TECHNICAL FLOW

### **Turn 1: User sends message with ambiguous contacts**

```
REQUEST:
{
  "message": "She's different from him. We both like BDSM...",
  "conversationId": "conv123",
  "userId": "user456"
}

PROCESSING:
1. main.go: Build AnalysisContext
2. unifiedOrchestrator.ProcessMessage():
   ├─ Create LayerContext
   ├─ Contact Workflow:
   │  ├─ Detect: 2 unnamed contacts (female, male)
   │  ├─ "she" matches female contact (0.95 confidence)
   │  ├─ "him" matches male contact (0.95 confidence)
   │  └─ Both high confidence → OK
   ├─ BUT WAIT: Message mentions "both" (plural)
   │  ├─ "We both like BDSM"
   │  ├─ Does "we" = user + girlfriend + colleague?
   │  ├─ Or user + girlfriend?
   │  ├─ Or just these two people?
   │  └─ Ambiguity detected (0.40 confidence)
   │
   ├─ STOP: Mark as needs_clarification=true
   ├─ Set clarification_type="contact_group_membership"
   ├─ Return early from ProcessMessage()
   └─ Return LayerContext with flag

RESPONSE:
{
  "status": "clarification_needed",
  "clarification": {
    "type": "contact_group_membership",
    "confidence": 0.40,
    "question": "When you say 'we both like BDSM', do you mean:",
    "options": [
      "A) Just you and your girlfriend",
      "B) You, your girlfriend, and your colleague",
      "C) Just you and your colleague",
      "D) All three of you"
    ],
    "context": {
      "detected_contacts": [
        {"id": "contact_1", "type": "female", "status": "unnamed"},
        {"id": "contact_2", "type": "male", "status": "unnamed"}
      ],
      "clarification_id": "clr_xyz123"
    }
  }
}
```

**UI Behavior:**
- Show question in conversation
- Display A/B/C/D options as buttons or selection
- Wait for user to select

---

### **Turn 2: User responds with clarification answer**

```
REQUEST:
{
  "message": "A) Just you and your girlfriend",  ← Answer
  "conversationId": "conv123",
  "userId": "user456",
  "clarification_response": {
    "clarification_id": "clr_xyz123",
    "answer": "A"
  }
}

PROCESSING:
1. main.go: MessageProcessor
   ├─ Detect: clarification_response flag present
   ├─ Extract: clarification_id = "clr_xyz123"
   ├─ Load: Clarification metadata from DB
   │  └─ Contains: original message, detected ambiguities
   │
   ├─ Process clarification answer:
   │  ├─ Answer = "A" (just user + girlfriend)
   │  ├─ Update contact_2 to "inactive" (not relevant)
   │  ├─ Set contact_1 group_membership = "user + contact_1"
   │  └─ Save to database
   │
   ├─ Load original message from clarification:
   │  └─ "She's different from him. We both like BDSM..."
   │
   ├─ Re-invoke orchestrator with:
   │  ├─ Same message (original)
   │  ├─ Updated contacts in analysisCtx
   │  ├─ Skip clarification detection (already done)
   │  └─ Run full orchestrator with clean data
   │
   └─ Generate response based on full analysis

RESPONSE:
{
  "status": "success",
  "message": "Got it, so your girlfriend is different from your colleague...
             [Full response about actual gaps/advice/etc]",
  "metadata": {
    "clarifications_resolved": 1,
    "contacts_identified": {
      "contact_1": {
        "name": null,
        "type": "romantic",
        "traits": ["different", "likes BDSM"],
        "group_membership": "user + contact_1"
      }
    }
  }
}
```

**UI Behavior:**
- Show clarification answer confirmation
- Display full Moly response (now with resolved context)
- Continue conversation

---

## CODE ARCHITECTURE

### **Layer Context Changes**

```go
type LayerContext struct {
    // Existing fields...
    
    // NEW: Clarification state
    ClarificationNeeded    bool
    ClarificationFlag      string // "contact_ambiguity", "group_membership", etc.
    ClarificationID        string // Unique ID for this clarification
    ClarificationQuestion  string
    ClarificationOptions   []string
    ClarificationConfidence float64
    
    // Contact data (populated by contact workflow)
    ActiveContacts         []Contact
    PronounResolutions     map[string]PronounResolution
    ContactContext         string
}
```

### **Message Processor Changes**

```go
// In main.go MessageProcessor

// Check if this is a clarification response
if req.ClarificationResponse != nil {
    // Load original message + context
    origMsg := LoadClarificationContext(req.ClarificationResponse.ClarificationID)
    
    // Process the answer
    UpdateContactsFromClarification(
        req.ClarificationResponse.Answer,
        origMsg.DetectedAmbiguities,
    )
    
    // Re-run orchestrator with SAME original message
    // but with updated contacts
    layerCtx, _ := srv.unifiedOrchestrator.ProcessMessage(
        ctx,
        origMsg.OriginalMessage,  // Same message
        userID,
        conversationID,
        newMessageID,
        analysisCtxWithUpdatedContacts,  // Updated!
        maturityContext,
    )
    
    // Continue to response generation
} else {
    // Normal message processing
    layerCtx, _ := srv.unifiedOrchestrator.ProcessMessage(...)
}

// Check if orchestrator flagged clarification needed
if layerCtx.ClarificationNeeded {
    // Save clarification for next turn
    SaveClarification(
        conversationID,
        layerCtx.ClarificationID,
        layerCtx.ClarificationQuestion,
        layerCtx.ClarificationOptions,
        userMessage,  // Store original message
    )
    
    // Return clarification response (don't generate full response)
    return ClarificationResponse{
        Status: "clarification_needed",
        Clarification: layerCtx.ClarificationData,
    }
}

// Otherwise generate normal response
response := GenerateResponse(layerCtx)
return NormalResponse{response}
```

### **Orchestrator Changes**

```go
// In unified_orchestrator.go ProcessMessage

func (uo *UnifiedOrchestrator) ProcessMessage(...) (*tools.LayerContext, error) {
    
    // ... existing code ...
    
    // NEW: Contact Workflow (before layer execution)
    contactDetector := NewContactDetector()
    detectedContacts := contactDetector.DetectInMessage(
        message,
        analysisCtx,
    )
    
    pronounResolver := tools.NewPronounResolver(uo.db)
    activeContacts := []Contact{}
    
    for _, contact := range detectedContacts {
        resolutions := pronounResolver.ResolveAntecedent(
            contact.Pronoun,
            detectedContacts,
        )
        
        confidence := calculateConfidence(resolutions)
        
        // If confidence < 0.50, STOP
        if confidence < 0.50 {
            lc.ClarificationNeeded = true
            lc.ClarificationID = generateID()
            lc.ClarificationQuestion = buildQuestion(contact, resolutions)
            lc.ClarificationOptions = buildOptions(resolutions)
            lc.ClarificationConfidence = confidence
            
            // Return early - don't run layers
            return lc, nil
        }
        
        activeContacts = append(activeContacts, contact)
    }
    
    // Wire resolved contacts to LayerContext
    lc.ActiveContacts = activeContacts
    lc.ContactContext = buildContactContextString(activeContacts)
    
    // Now run all layers (with clean contact data)
    for i, layer := range uo.layers {
        // ... existing layer execution ...
    }
    
    return lc, nil
}
```

---

## DATABASE SCHEMA

### **New Table: Clarifications**

```sql
CREATE TABLE clarifications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    
    -- Original context
    original_message TEXT NOT NULL,
    message_sequence INT,
    
    -- Clarification data
    clarification_type TEXT,  -- "contact_ambiguity", "group_membership", etc.
    question TEXT NOT NULL,
    options TEXT,  -- JSON array: ["A", "B", "C", ...]
    
    -- Ambiguities detected
    detected_ambiguities TEXT,  -- JSON: what was ambiguous
    confidence REAL,
    
    -- Response
    user_answer TEXT,  -- "A", "B", etc.
    answer_message TEXT,  -- "A) Just you and your girlfriend"
    
    status TEXT,  -- "pending", "answered", "expired"
    
    created_at INTEGER,
    answered_at INTEGER,
    expires_at INTEGER,
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);
```

---

## FLOW SEQUENCE DIAGRAM

```
Turn 1: User Message
┌─────────────────────────────────────────────────┐
│ User: "She's different from him. We both..."    │
└──────────────┬──────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────┐
│ main.go: Build AnalysisContext                  │
│          Load conversation history              │
└──────────────┬──────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────┐
│ unifiedOrchestrator.ProcessMessage()            │
│ ├─ Contact Workflow:                            │
│ │  ├─ Detect: 2 unnamed contacts                │
│ │  ├─ Ambiguity: "we both" (0.40 confidence)    │
│ │  ├─ STOP ← ClarificationNeeded=true           │
│ │  └─ Return early                              │
│ └─ (Layers NOT executed)                        │
└──────────────┬──────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────┐
│ Response: Clarification Question                │
│ "When you say 'we both like BDSM':             │
│  A) Just you and girlfriend                     │
│  B) You, girlfriend, and colleague              │
│  C) Just you and colleague                      │
│  D) All three"                                  │
│                                                 │
│ [Save clarification to DB]                      │
└──────────────┬──────────────────────────────────┘
               │
        User selects "A"
               │
               ▼
Turn 2: Clarification Response
┌─────────────────────────────────────────────────┐
│ Request: clarification_response="A"             │
│          clarification_id="clr_xyz123"          │
└──────────────┬──────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────┐
│ main.go: MessageProcessor                       │
│ ├─ Load clarification context                   │
│ ├─ Update contacts from answer                  │
│ ├─ Save to database                             │
│ └─ Reload original message with updated DB     │
└──────────────┬──────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────┐
│ unifiedOrchestrator.ProcessMessage()            │
│ (same original message, but contacts resolved)  │
│ ├─ Contact Workflow: ✓ No ambiguity             │
│ ├─ Wire contacts to LayerContext                │
│ ├─ Run Layers 1-11 (with clean data)            │
│ └─ Return complete LayerContext                 │
└──────────────┬──────────────────────────────────┘
               │
               ▼
┌─────────────────────────────────────────────────┐
│ Response: Full Moly Response                    │
│ "Got it, so your girlfriend is different...    │
│  [Analysis with resolved contacts]              │
│  [Gaps/advice based on clean data]"             │
└─────────────────────────────────────────────────┘
```

---

## IMPLEMENTATION CHECKLIST

### **Phase 1: Data Layer**
- [ ] Create `clarifications` table
- [ ] Add methods to ClarificationRepository
- [ ] Update Contact model to support clarification links

### **Phase 2: Contact Workflow**
- [ ] Create ContactDetector
- [ ] Integrate into unifiedOrchestrator
- [ ] Implement confidence scoring
- [ ] Implement ambiguity detection
- [ ] Return LayerContext with ClarificationNeeded flag

### **Phase 3: Message Processor**
- [ ] Detect clarification_response in request
- [ ] Load clarification context from DB
- [ ] Process clarification answer
- [ ] Update contacts in DB
- [ ] Re-invoke orchestrator with same message

### **Phase 4: Response Generation**
- [ ] Handle ClarificationNeeded flag
- [ ] Format clarification question + options
- [ ] Save clarification to DB
- [ ] Generate normal response for answered clarifications

### **Phase 5: UI Integration**
- [ ] Display clarification questions
- [ ] Render options as selectable
- [ ] Submit clarification answer
- [ ] Show resolved response

---

## KEY DESIGN PRINCIPLES

1. **Natural Interruption Pattern**
   - Moly stops and asks, doesn't continue guessing
   - User answers inline (same turn)
   - Moly resumes with full context

2. **Database-Backed Clarifications**
   - Save original message + context
   - Clarifications expire if not answered
   - Auditable: track what was asked, when, what was answered

3. **Clean Data Through Layers**
   - Ambiguities resolved BEFORE orchestrator runs
   - All 11 layers get unambiguous contact data
   - No false contradictions in Layer 5

4. **Single-Turn Recovery**
   - User answers clarification in same turn
   - System immediately re-processes with resolved data
   - Natural conversation flow maintained

---

## EXPECTED OUTCOMES

✅ **Natural Conversation Flow** - Mirrors human interruption/clarification pattern  
✅ **Zero False Positives** - Contact ambiguity resolved before analysis  
✅ **Database Audit Trail** - Every clarification saved and tracked  
✅ **Single-Turn Resolution** - Answer + resolution in same turn  
✅ **Coherent Context** - All layers work with clean, unambiguous data  

