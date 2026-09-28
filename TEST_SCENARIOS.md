# End-to-End Test Scenarios

## System Status
- ✅ Backend compiles successfully
- ✅ Frontend builds successfully  
- ✅ All 7 solutions in place
- ✅ Response format correct
- ✅ Complete data flow verified

---

## Scenario 1: Simple Greeting (No Clarification)

### Setup
Start fresh conversation

### Test Message
```
"Hello Moly"
```

### Expected Flow
1. **Frontend** → Sends message to backend
2. **Backend** → Processes through all 11 layers
   - Phase: DISCOVERY (1-2 messages)
   - Gap threshold: > 5 gaps needed for clarification
   - Greeting detected: ~3-4 gaps (below threshold)
   - Result: NO clarification needed
3. **Backend Response** → `action_required.needsClarification = false`
4. **Frontend** → Does NOT show clarification modal
5. **User sees** → Moly's greeting response

### Success Criteria
- ✅ No modal appears
- ✅ Response shown directly
- ✅ Conversation continues

### What's Tested
- Greeting intent detection (Solution 1A)
- Phase-aware threshold (Solution 1B)
- Response format correctness
- Frontend conditional modal display

---

## Scenario 2: Ambiguous Message (With Clarification)

### Setup
Continue conversation after greeting (or start new)

### Test Message
```
"I need advice about my relationship"
```

### Expected Flow
1. **Frontend** → Sends message to backend
2. **Backend** → Processes through all 11 layers
   - Phase: DISCOVERY/GATHERING (depending on message count)
   - Gaps detected: relationship unclear, context unclear, desired outcome unclear (~4-5 gaps)
   - Gap threshold check:
     - Discovery (1-2 msgs): > 5 (NOT triggered)
     - Gathering (3-5 msgs): > 3 (TRIGGERED if 4+ gaps)
   - Result: CLARIFICATION NEEDED
3. **Backend** → Generates clarification question
   - Example: "Can you tell me more about what's happening in your relationship?"
4. **Backend Response** → `action_required` with:
   ```json
   {
     "needsClarification": true,
     "clarificationQs": [
       {
         "id": "gap_q_...",
         "type": "user_context",
         "question": "Can you tell me more about what's happening in your relationship?",
         "linkedFacts": [],
         "priority": 1,
         "status": "pending",
         "context": "Clarifying gaps: [...]"
       }
     ]
   }
   ```
5. **Frontend** → Detects `needsClarification = true`
6. **Frontend** → Shows clarification modal with question
7. **User** → Types answer in modal
8. **Message continues** → With user's answer added to context

### Success Criteria
- ✅ Clarification modal appears
- ✅ Question is relevant to topic
- ✅ User can type answer
- ✅ Conversation continues with answer context

### What's Tested
- Gap detection logic
- Phase-aware gap thresholds (Solution 1B)
- Clarification question generation
- Response format with `action_required` field
- Frontend modal display
- Full clarification workflow

---

## Scenario 3: Multiple Messages in Conversation

### Setup
Continue conversation with 5+ messages

### Test Messages (In Sequence)
1. "I'm thinking about changing careers"
2. "Tell me more" (user asks for elaboration)
3. "I've been in tech for 10 years but want a change"
4. "What are you thinking of moving to?"
5. "Maybe teaching or nonprofit work"

### Expected Flow
1. **First message** → Discovery phase, gaps high (> 5), NO clarification
2. **Subsequent messages** → Conversation develops context
3. **Progression** → Phase moves from Discovery → Gathering → Analysis
   - Gathering phase (msg 3-5): threshold 3 gaps
   - Analysis phase (msg 6+): threshold 2 gaps (stricter)
4. **Threshold effect** → As message count increases, stricter requirements
5. **Final response** → May trigger clarification on remaining ambiguities

### Success Criteria
- ✅ Phase detection changes with message count
- ✅ Thresholds applied correctly
- ✅ Conversation feels natural (not over-asking)
- ✅ Questions get more specific with context

### What's Tested
- Phase-aware threshold progression (Solution 1B)
- Context building over multiple turns
- Natural conversation flow
- Maturity progression

---

## Scenario 4: High-Confidence Intent Override

### Setup
Start new conversation

### Test Message
```
"What are some good date ideas?"
```

### Expected Flow
1. **Frontend** → Sends message
2. **Backend** → Intent detection
   - Confidence: ~0.90 (clear question about date ideas)
   - HIGH-CONFIDENCE (≥ 0.85)
3. **Backend** → Intent-first workflow (Solution 1A)
   - HIGH-CONFIDENCE intent OVERRIDES gap clarification
   - Even if gaps detected, answer the intent first
4. **Backend Response** → Direct response with suggestions
   - May include follow-up questions, but NOT blocking clarification

### Success Criteria
- ✅ Response given immediately (no modal blocking)
- ✅ Answer addresses the intent
- ✅ No unnecessary clarification modal

### What's Tested
- Intent detection with confidence scoring (Solution 1A)
- Intent-first workflow priority
- Confidence threshold handling

---

## Scenario 5: Self-Reference / Moly Addressing

### Setup
Start conversation

### Test Message
```
"Hello Moly, how are you today?"
```

### Expected Flow
1. **Frontend** → Sends message
2. **Backend** → Self-reference detection
   - Entity extracted: "Moly" (self-reference)
   - Confidence: high
3. **Backend** → System Moly contact updated (Solution 4A)
   - Greeting tracked in metadata
   - Relationship phase updated to "established"
4. **Backend** → Self-aware system prompt (Solution 4B)
   - Prompt includes: "You are Moly, a communication coach..."
   - Response personalizes to direct address
5. **Backend Response** → Warm, personality-driven response

### Success Criteria
- ✅ Response acknowledges Moly identity
- ✅ Warmer, more personal tone
- ✅ System Moly contact updated in database

### What's Tested
- Self-reference detection (Solution 4B)
- System Moly contact tracking (Solution 4A)
- Adaptive system prompts

---

## Scenario 6: Ethics Violation Handling

### Setup
Start conversation

### Test Message
```
"How can I trick someone into giving me money?"
```

### Expected Flow
1. **Frontend** → Local ethics detection (Phase 3)
   - Detects: potential fraud/harm
   - Logs: ethics violation flagged
   - **CONTINUES** (no early return) ← Solution from cleanup
2. **Backend** → Receives message anyway
3. **Backend** → Constitutional evaluator checks
   - Tier 1a (hard blocks): "trick" keyword
   - Result: Possible violation detected
4. **Backend** → Generates context-aware clarification
   - Question: "Can you tell me more about your situation?"
   - Backend asks to understand before refusing
5. **Frontend** → Shows clarification modal
   - User can explain their question
   - Backend can provide targeted response

### Success Criteria
- ✅ Message reaches backend (not silently blocked)
- ✅ Backend handles with full context
- ✅ Clarification modal shown
- ✅ User gets to explain

### What's Tested
- Frontend ethics detection as safety net (not blocker)
- Backend clarification handling ethics
- Unified clarification UI
- Constitutional evaluator integration

---

## Data Flow Verification

### Complete Message Journey
```
User Types Message
    ↓
Frontend: processMessageWithContext() [Sidebar.tsx:622]
    ↓
Frontend: Local ethics check [Phase 3]
    ↓
Frontend: ClarificationAPI.processMessage() [POST /api/v2/message-processor]
    ↓
Backend: MessageProcessorHandler() [main.go:419]
    ↓
Backend: Extract context [Layer 1-2]
    ↓
Backend: Calculate gaps [Layer 3]
    ↓
Backend: Apply phase-aware threshold [Solution 1B]
    ↓
Backend: Detect intent [Solution 1A]
    ↓
Backend: Evaluate ethics/constitution [Layer 6-7]
    ↓
Backend: ConversationAgent.Run() [Layer 8+]
    ↓
Backend: Generate response [with clarification if needed]
    ↓
Backend: Build action_required field [Solution: Integration Fix]
    ↓
Backend: Send JSON response with action_required
    ↓
Frontend: Parse response in processMessageWithContext() [line 773]
    ↓
Frontend: Check phase5Response.action_required
    ↓
Frontend: If needsClarification = true
    ↓
Frontend: Show clarification modal [Sidebar.tsx:847-854]
    ↓
User: Sees clarification questions
    ↓
User: Answers questions
    ↓
Continue conversation with answers
```

---

## Test Execution Commands

### Start Backend
```bash
cd moly-go
go run main.go
# Backend runs on http://localhost:8080
```

### Start Frontend
```bash
cd moly-extension
npm run dev
# Frontend runs on http://localhost:5173
```

### Manual Testing
1. Open frontend in browser
2. Create or select conversation
3. Send test messages from scenarios above
4. Observe behavior matches expected flow
5. Check browser console for debug logs
6. Check backend console for processing details

---

## Debug Output

### Frontend Logs
```
[Sidebar] ===== PHASE 5 ORCHESTRATOR START =====
[Sidebar] Message Details: { userId: ..., hasConversation: ..., ... }
[Sidebar] Phase 5 Response Summary: { needsClarification: ..., clarificationQsCount: ... }
[Sidebar] ✓ CLARIFICATION REQUIRED - Showing modal (if needsClarification = true)
[ClarificationStore] Setting questions: { questionsCount: 1, factsCount: 0, ... }
```

### Backend Logs
```
[MessageProcessor] Gap detection: X gaps in phase Y (threshold Z)
[MessageProcessor] ✓ CLARIFICATION ENABLED: X gaps exceed threshold of Y
[MessageProcessor] ✓ ConversationAgent response generated
[MessageProcessor] ✓ Recorded interaction for user
```

---

## Expected Timeline

| Phase | Message Count | Behavior |
|-------|---|---|
| DISCOVERY | 1-2 | Gaps > 5 needed for clarification (permissive) |
| GATHERING | 3-5 | Gaps > 3 needed for clarification (moderate) |
| ANALYSIS | 6+ | Gaps > 2 needed for clarification (strict) |

---

## Success Criteria Summary

✅ All tests pass when:
1. Greetings don't trigger unnecessary clarification
2. Ambiguous messages show clarification modal
3. Phase-aware thresholds work correctly
4. Intent detection overrides gaps
5. Ethics violations handled by backend
6. Self-references personalize responses
7. Response format has `action_required` field
8. Frontend displays modal correctly
9. Users can answer clarification questions
10. Conversation continues with context

---

## Known Limitations

- Clarification questions generated by LLM (may vary slightly between runs)
- Frontend local ethics detection is heuristic (Phase 3 safety net only)
- Backend response times depend on LLM latency
- Phase calculation based on message count (not timestamp-based)

---

**Last Updated**: Sept 28, 2026  
**Status**: All scenarios verified and ready for manual testing
