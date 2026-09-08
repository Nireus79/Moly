# Phase 1.2 Design Summary: From History to Notes

## The Shift

### Phase 1: Foundation (✅ Complete)
- Encryption: ✅ AES-256 per-user
- Auth: ✅ Code generation + sessions  
- Chat: ✅ Session-based message endpoints
- **What wasn't built**: Long-term storage model

### Phase 1.2: Intelligent Notes (New)
- **Don't store conversations**: Keep only 24h for review
- **Store insights**: AboutMe, patterns, goals, contacts
- **Auto-extract**: ConversationAnalyzer + ProfileUpdater
- **User control**: Confirm/reject learnings, write reflections

**Result**: Token costs stay constant. User never feels monitored. System learns intelligently.

---

## Key Differences: What Changed

| Aspect | Phase 1 | Phase 1.2 |
|--------|---------|----------|
| **Chat Storage** | Sent to chat_messages table | Stored 24h in conversation_ephemeral, then deleted |
| **History Keeping** | All messages kept forever | Only notes are permanent |
| **LLM Context** | Fetch full conversation from DB | Fetch structured notes (~2-3k tokens) |
| **Token Cost** | Scales with conversation length | Constant, regardless of history |
| **User Privacy Feel** | "System has transcript of everything" | "Coach is taking notes about me" |
| **Data on Disk** | Full chat transcripts | Structured insights only |
| **User Correction** | N/A (wasn't tracking learnings) | Can reject/confirm what system learns |

---

## What Survives from Phase 1

### Unchanged
- ✅ Encryption (AES-256 per-user)
- ✅ Session management (24h tokens)
- ✅ Chat endpoints (`POST /api/v2.1/chat`)
- ✅ Bearer token validation middleware

### Renamed/Reorganized
- `chat_messages` table → `conversation_ephemeral` (temporary)
- `about_me` table → `about_me_profile` (structured, updated incrementally)
- New: `conversation_ephemeral`, `extraction_queue`, `communication_patterns`, `communication_goals`, `reflection_journal`, `implicit_learning`, `contact_communication_patterns`

### Deleted
- Raw `chat_messages` table (we don't keep raw messages)
- `interactions` table (replaced by patterns)
- `behavior_patterns` table (replaced by structured `communication_patterns`)

---

## New Tables (Phase 1.2)

8 new tables, organized by lifecycle:

### Permanent (Forever)
1. **about_me_profile** - User's communication style, values, preferences
2. **contacts** - People in user's life
3. **contact_communication_patterns** - How user relates to each person
4. **communication_patterns** - General patterns observed
5. **communication_goals** - What user is working on
6. **reflection_journal** - Optional user notes
7. **implicit_learning** - What system learned (with confidence)

### Ephemeral (24h)
8. **conversation_ephemeral** - Raw messages, auto-deleted
9. **extraction_queue** - Processing pipeline

---

## The Extraction Pipeline

### 1. User Has Conversation
```
User → Chat Endpoint → Session validated → Message stored
```

### 2. Save to Ephemeral (EphemeralConversationManager)
```python
SaveConversation(userID, conversationID, messages):
  INSERT conversation_ephemeral (messages, expires_at=now+24h)
  INSERT extraction_queue (status='pending')
```

**What happens**: Conversation lives in DB for 24h so user can review, then auto-deletes.

### 3. Extract Insights (ConversationAnalyzer) - Hourly Job
```python
ProcessQueue():
  FOR each pending extraction:
    messages = fetch from conversation_ephemeral
    result = LLM_analyze(messages)  # Extract insights
    UPDATE extraction_queue (status='completed', result=...)
```

**LLM receives**: Raw messages (only this once, during extraction)
**LLM returns**: Structured insights (JSON)

**Example LLM prompt**:
```
Analyze this conversation and extract ONLY high-confidence insights:

1. Communication style: [direct|gentle|thoughtful]
2. Values demonstrated: [list]
3. Patterns: [{pattern, confidence 0-1}]
4. Contacts mentioned: [{name, relationship_type, tone}]
5. Goal progress: [what goals are they working on?]

Return JSON only.
```

### 4. Update Profile (ProfileUpdater)
```python
UpdateProfile(userID, extractionResult):
  FOR each insight:
    IF confidence > threshold (0.6):
      UPDATE permanent table (merge, don't overwrite)
      INCREMENT reinforcement_count
      TRACK source & timestamp
```

**Merge logic**:
- AboutMe: "Set communication_style if new or higher confidence"
- Patterns: "Add new pattern with observation_count=1, or increment count if exists"
- Contacts: "Merge tone_observed, frequency, topics"
- Goals: "Link to active goals if mentioned"

### 5. User Queries Profile (ProfileService)
```
GET /api/v2.1/profile
  → AboutMe profile (confidence 0.85)
  → Contacts with patterns (frequency, tone, topics)
  → Active goals (with progress)
  → Patterns observed (with reinforcement count)
  → Learnings (with confidence scores)
```

### 6. User Confirms/Rejects (ProfileService)
```
POST /api/v2.1/learnings/123/confirm
POST /api/v2.1/learnings/123/reject

→ Sets is_confirmed or is_rejected flag
→ System doesn't re-propose rejected learnings
→ Confirmed learnings boost confidence
```

### 7. Cleanup (EphemeralConversationManager) - Daily Job
```python
CleanupExpired():
  DELETE FROM conversation_ephemeral WHERE expires_at < now()
  # Raw messages gone, only notes remain
```

---

## Privacy & User Experience

### Before (Raw History)
- 😱 User thinks: "Everything I said is stored"
- 💸 Cost: Grows with every message
- 🤔 LLM sees: Full transcript every time (wasteful)
- 🚫 No user control: Can't correct system's understanding

### After (Notes Model)
- 😊 User thinks: "Coach takes notes about me"
- 💰 Cost: Constant (~2-3k tokens for context)
- 🎯 LLM sees: Only structured insights
- ✅ User control: Can confirm/reject what system learned

---

## Technical Design: Key Components

### ConversationAnalyzer
- Runs once per conversation (during extraction)
- LLM extracts from raw messages
- Returns confidence-scored insights
- **Handles**: Extraction, confidence thresholds, error handling

### ProfileUpdater
- Runs after extraction
- Applies insights to permanent tables
- Merges incrementally (doesn't overwrite)
- **Handles**: Threshold checking, conflict resolution, tracing

### EphemeralConversationManager
- Saves conversation → 24h window
- Triggers extraction
- Cleans up expired conversations
- **Handles**: Lifecycle, TTL, queue management

### ProfileService
- Read-only API for user's profile
- Exposes all notes tables
- Includes confidence/source metadata
- **Handles**: Access control, response formatting

### Background Jobs
1. **ProcessQueue** (hourly): Extract pending conversations
2. **CleanupExpired** (daily): Delete 24h-old conversations

---

## Why This Design Works

### ✅ Privacy-First
- No surveillance feeling (notes, not transcript)
- User can review/correct system's understanding
- Explicit user control over learnings

### ✅ Cost-Efficient
- Context = ~2-3k tokens, constant size
- No "re-reading history" cost
- LLM sees data once (during extraction), not repeatedly

### ✅ Scalable
- Database stores only insights, not raw text
- No query performance hit from large histories
- Storage costs capped at ~100k per user

### ✅ Transparent
- User sees what system learned (confidence scores)
- Can confirm "yes, that's me" or "no, that's wrong"
- Can write own reflections (journal)

### ✅ Coachable
- System learns patterns across conversations
- Tracks goal progress
- Knows what matters to user
- Never loses context (notes stay)

---

## Risk Mitigation

### Risk: LLM Extracts Wrong Insights
**Mitigations**:
- Confidence thresholds (only update if > 0.6)
- User can reject: "No, I don't value that"
- Manual review: Check what's being extracted

### Risk: User Feels System is Still Monitoring
**Mitigations**:
- Be explicit: "We keep notes, not transcripts"
- Show the notes: User can see what system knows
- Control: User can delete/reject learnings
- Journal: User writes their own notes

### Risk: Important Context Lost After 24h
**Mitigations**:
- Extract to notes (important stuff stays)
- 24h window for user to retrieve if needed
- Extraction happens before deletion

### Risk: Notes Become Inaccurate Over Time
**Mitigations**:
- Reinforcement tracking (see pattern multiple times)
- Explicit confirmation (user says "yes, that's me")
- Rejection mechanism (user says "no, wrong")
- Regular review flow (show what system learned)

---

## Example User Journey

### Day 1
```
User: "I need to tell my boss she's being unfair"
System: Saves conversation (24h)
        → Extracts: "Communication style = direct, values honesty, pattern = conflict avoidance"
        → Updates profile
        → Next day: Deletes conversation
```

### Day 2
```
User: Views profile
System: Shows:
  "Communication style: Direct (85% confidence)"
  "Values: Honesty, Growth"
  "Pattern: Sometimes avoids conflict (seen 2x)"
  "Contact: Boss (professional, weekly, tone = cautious)"
  "Goal: Say no without apologizing"

User: Clicks on pattern "avoids conflict"
System: Shows: "Based on 2 conversations, confidence 0.7"
        Offers: "Is this right?" [Confirm] [Reject]

User: [Confirm] "Yes, I do this"
System: Confidence → 0.9, marked is_confirmed=true
```

### Day 5
```
User: Has another conversation mentioning boss
System: Extracts again
        Pattern "avoids conflict" now seen 3x, confidence 0.85
        But tone_with_boss changes from "fearful" → "more assertive"
        
User: Checks profile
System: Shows: "You've been more direct with Sarah recently 👍"
        Goal "say no without apologizing": progress updated
```

---

## What Gets Deleted vs Kept

### ❌ Deleted After 24h (conversation_ephemeral)
- Raw message text
- Full transcripts
- "He said X" / "She said Y"
- Timing details within conversation

### ✅ Kept Forever (structured notes)
- "Communication style: direct"
- "Values: honesty, growth"
- "Pattern: avoids conflict then over-explains"
- "Boss: professional relationship, cautious tone"
- "Working on: assertiveness"
- User's own reflection notes

---

## Comparison: Raw History vs Notes

### Scenario: 100 Conversations Over Time

**Raw History Approach**:
```
Conversation 1: 2k tokens (stored)
Conversation 2: 2k tokens (stored)
...
Conversation 100: 2k tokens (stored)

Total storage: 200k tokens
Total cost when querying: 200k tokens per context
User feeling: "They have a transcript of everything"
```

**Notes Approach**:
```
All 100 conversations analyzed once (during extraction)
Insights merged into permanent notes:
  - AboutMe (50 tokens)
  - 5 contacts with patterns (100 tokens)
  - 8 patterns observed (150 tokens)
  - 3 active goals (50 tokens)
  - Journal entries (200 tokens)

Total storage: ~550 tokens
Total cost when querying: ~550 tokens per context (constant!)
User feeling: "They know me, but don't have a recording of everything"
```

**Savings**: 99.7% less context when querying. Same coaching value.

---

## Next Steps: Phase 1.2 Implementation

1. ✅ Design complete (this document)
2. → Apply database schema
3. → Build ConversationAnalyzer
4. → Build ProfileUpdater
5. → Build EphemeralConversationManager
6. → Build ProfileService + HTTP handlers
7. → Set up background jobs (extraction, cleanup)
8. → Browser extension UI to show profile
9. → Test end-to-end

---

## Questions to Answer Before Building

1. **Confidence threshold**: Should it be 0.6 for all, or vary per insight type?
2. **User visibility**: Should users see learnings with confidence scores by default?
3. **Reflection journal**: Is it totally user-written, or can system suggest entries?
4. **Contact inference**: Can system infer things about contacts (e.g., "demanding")?
5. **Update merging**: How do we handle contradictions? (was direct, now gentle?)

