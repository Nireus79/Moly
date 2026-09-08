# Phase 1.2: Notes-Based Coaching System

## Problem We're Solving

**User's Concern**: 
- Storing full chat history is expensive (tokens scale with history)
- Users feel monitored if system keeps everything they said
- Privacy-first philosophy violated by transcript storage

**Solution**: 
- Keep conversations only 24h (for review)
- Auto-extract insights to permanent notes
- User controls what system "knows"
- Token cost stays constant

## What You'll Get

| Aspect | Benefit |
|--------|---------|
| **Token Efficiency** | 99.7% reduction: 200k → 550 tokens constant |
| **Privacy** | User feels coached, not monitored |
| **Transparency** | User sees what system learned (with confidence) |
| **Control** | User can confirm/reject learnings |
| **Scalability** | Storage and costs don't grow with history |

---

## Design Documents (Read in Order)

1. **PHASE_1_2_SCHEMA_REDESIGN.md** 
   - Detailed explanation of all tables
   - Data model philosophy
   - Why each table exists
   - **Read this to understand the data model**

2. **PHASE_1_2_ARCHITECTURE.md**
   - Component breakdown
   - Data flow end-to-end
   - API endpoints
   - Background jobs
   - Implementation order
   - **Read this to understand how to build it**

3. **PHASE_1_2_DESIGN_SUMMARY.md**
   - High-level comparison (before/after)
   - Privacy & UX considerations
   - Risk mitigation
   - Example user journey
   - **Read this for context & rationale**

---

## Database Schema

### New Tables (in `database/schema_v2_1_phase_1_2.sql`)

**Permanent (Forever)**:
- `about_me_profile` - User's communication style, values, preferences
- `contacts` - People in user's life
- `contact_communication_patterns` - How user relates to each person
- `communication_patterns` - General patterns (conflict avoidance, etc.)
- `communication_goals` - What user is working on
- `reflection_journal` - User's own notes (optional)
- `implicit_learning` - What system learned (with confidence)

**Ephemeral (24h)**:
- `conversation_ephemeral` - Raw messages, auto-deleted
- `extraction_queue` - Processing pipeline

### Key Philosophy

```
User conversation (24h)
    ↓
ConversationAnalyzer extracts insights
    ↓
ProfileUpdater applies to permanent notes
    ↓
Conversation auto-deleted, notes kept
```

---

## New Components to Build

### 1. ConversationAnalyzer (`agents/conversation_analyzer.go`)
**Job**: Extract structured insights from conversations

**What it does**:
- Takes raw messages
- Runs LLM extraction (once per conversation)
- Returns JSON: AboutMe, patterns, contacts, goals, confidence scores

**Example**:
```go
result := analyzer.AnalyzeConversation(ctx, userID, conversationID, messages)
// Returns: 
// {
//   AboutMeUpdates: [{key: "communication_style", value: "direct", confidence: 0.7}],
//   PatternDetections: [{pattern: "avoids_conflict", confidence: 0.8}],
//   ContactMentions: [{name: "boss", tone: "fearful"}],
//   GoalProgress: [{goal: "assertiveness", status: "progress"}]
// }
```

### 2. ProfileUpdater (`services/profile_updater.go`)
**Job**: Apply extraction results to permanent tables

**What it does**:
- Takes extraction result
- Checks confidence thresholds
- Merges into permanent notes (doesn't overwrite)
- Tracks reinforcement (if we see same pattern twice, confidence increases)

**Example**:
```go
updater.UpdateProfile(ctx, userID, extractionResult)
// This merges insights into all permanent tables
// Only updates if confidence > 0.6 (configurable)
```

### 3. EphemeralConversationManager (`services/ephemeral_manager.go`)
**Job**: Manage conversation lifecycle (save → extract → delete)

**What it does**:
- Save conversation to ephemeral table (24h TTL)
- Queue for extraction
- Run extraction pipeline (ProcessQueue)
- Auto-delete expired conversations

**Example**:
```go
// After chat conversation completes
manager.SaveConversation(ctx, userID, conversationID, messages)
// → Stores for 24h, queues extraction

// Hourly job
manager.ProcessQueue(ctx)
// → Extract pending, update profiles

// Daily job
manager.CleanupExpired(ctx)
// → Delete 24h-old conversations
```

### 4. ProfileService (`services/profile_service.go`)
**Job**: Expose user's profile for UI consumption

**What it does**:
- Get complete profile snapshot
- Get individual components (AboutMe, contacts, goals, etc.)
- Handle user confirmation/rejection of learnings

**Example**:
```go
profile := service.GetUserProfile(userID)
// Returns: AboutMe, Contacts, Goals, Patterns, Reflections

learnings := service.GetLearnings(userID)
// Returns: All implicit learnings with confidence scores
```

---

## API Endpoints

### Profile Read
```
GET  /api/v2.1/profile                    → Full profile snapshot
GET  /api/v2.1/profile/about-me           → Communication style, values, preferences
GET  /api/v2.1/profile/contacts           → People in user's life with patterns
GET  /api/v2.1/profile/goals              → Active communication goals
GET  /api/v2.1/profile/patterns           → Observed patterns
GET  /api/v2.1/profile/learnings          → What system learned (with confidence)
GET  /api/v2.1/profile/reflections        → User's journal entries
```

### Profile Write
```
POST /api/v2.1/goals                      → Create new goal
PATCH /api/v2.1/goals/{id}                → Update goal progress

POST /api/v2.1/reflections                → Add journal entry

POST /api/v2.1/learnings/{id}/confirm     → User says "yes, that's me"
POST /api/v2.1/learnings/{id}/reject      → User says "no, that's wrong"
```

---

## Example Response: GET /api/v2.1/profile

```json
{
  "about_me": {
    "communication_style": "direct",
    "tone_preference": "warm",
    "core_values": ["honesty", "growth"],
    "confidence": 0.85,
    "extracted_from_count": 12
  },
  "contacts": [
    {
      "name": "Sarah",
      "relationship_type": "professional",
      "frequency": "weekly",
      "tone_observed": "formal",
      "main_topics": ["feedback", "recognition"]
    }
  ],
  "active_goals": [
    {
      "goal": "say no without apologizing",
      "status": "active",
      "confidence": 0.6,
      "progress_notes": "Getting better, working on it"
    }
  ],
  "active_patterns": [
    {
      "pattern": "avoids_conflict_then_over_explains",
      "confidence": 0.8,
      "is_growth_area": true,
      "observation_count": 4
    }
  ]
}
```

---

## Background Jobs

### ProcessQueue (Hourly)
```
1. Get all extraction_queue entries with status='pending'
2. For each:
   a. Fetch conversation_ephemeral
   b. Run ConversationAnalyzer
   c. Run ProfileUpdater
   d. Update extraction_queue (status='completed')
```

### CleanupExpired (Daily)
```
1. DELETE FROM conversation_ephemeral WHERE expires_at < now()
2. Log what was deleted
```

---

## Implementation Roadmap

### Week 1: Foundation
- [ ] Apply schema_v2_1_phase_1_2.sql to database
- [ ] Build ConversationAnalyzer
- [ ] Build ProfileUpdater
- [ ] Write unit tests

### Week 2: Integration
- [ ] Build EphemeralConversationManager
- [ ] Integrate with chat handler (save conversations)
- [ ] Set up background jobs (extraction, cleanup)
- [ ] Write integration tests

### Week 3: API & UI
- [ ] Build ProfileService
- [ ] Build HTTP handlers for profile endpoints
- [ ] Browser extension UI to show profile
- [ ] User confirmation/rejection flow

### Week 4: Polish & Testing
- [ ] E2E testing (conversation → extraction → profile)
- [ ] Performance optimization
- [ ] Edge case handling
- [ ] Documentation

---

## Key Decisions Made

### ✅ Store Only Insights, Not Transcripts
- Permanent: AboutMe, patterns, goals, contacts, learnings
- Ephemeral (24h): Raw messages
- Result: Privacy + efficiency

### ✅ Confidence Scores on Everything
- User sees: "Communication style: direct (85% confidence)"
- User can confirm/reject: "Yes, that's me" or "No, that's wrong"
- Result: Transparency + control

### ✅ Incremental Updates, Not Overwrites
- If we learn "direct" in conversation 1, keep it
- If we learn "direct" again in conversation 5, bump confidence
- Result: Learning improves over time

### ✅ User-Controlled Reflection Journal
- User can write own notes (optional)
- "Had good talk with Mom about boundaries"
- System never writes user's journal
- Result: Agency + explicit learning

---

## Privacy Guarantees

### What User Should Never See
- ❌ Full transcripts of conversations
- ❌ "You said: ..."
- ❌ Verbatim quotes from their messages
- ❌ Who said what, when

### What User Should See
- ✅ "Communication style: direct"
- ✅ "Values: honesty, growth"
- ✅ "Pattern: sometimes avoids conflict (seen 3x)"
- ✅ "Contact: Boss (professional, weekly)"
- ✅ Confidence scores
- ✅ Source of each learning

### What Moly Should Never Do
- ❌ Analyze user's character/personality
- ❌ Judge the user
- ❌ Store private details about contacts
- ❌ Infer things not explicitly stated

---

## Risk & Mitigation

| Risk | Mitigation |
|------|------------|
| LLM extracts wrong insight | Confidence threshold (only update if > 0.6), user can reject |
| User feels still monitored | Be explicit about 24h deletion, show what system knows |
| Important context lost after 24h | Extract to notes before deleting, show in notes |
| Notes become inaccurate | Reinforcement tracking, explicit user confirmation |
| User confusion about learning | Show confidence scores, allow feedback loop |

---

## Questions to Answer During Implementation

1. **Confidence threshold**: 0.6 default? Should it vary per type?
2. **Extraction frequency**: Hourly jobs? Real-time? Batch?
3. **User education**: How do we explain the notes model?
4. **Contact inference**: Can we infer things about people, or just user's perception?
5. **Update conflicts**: If user said "I'm direct" but contradicts self, how merge?

---

## Success Criteria

- ✅ Conversations auto-delete after 24h
- ✅ Notes stay permanent
- ✅ Context size stays constant (~2-3k tokens)
- ✅ User can see what system learned
- ✅ User can confirm/reject learnings
- ✅ Extraction confidence > 0.6 for updates
- ✅ No raw transcripts in permanent storage
- ✅ E2E flow: conversation → extraction → profile works

---

## Related Documents

- Phase 1: Encryption + Auth + Chat (✅ Complete)
- Phase 1.2: Notes system (this document)
- Phase 1.3: Browser extension UI (planned)
- Phase 2: LLM integration for suggestions (planned)

