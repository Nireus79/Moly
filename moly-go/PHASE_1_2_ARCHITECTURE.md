# Phase 1.2 Architecture: Notes-Based Coaching System

## Overview

**Phase 1** ✅ Built encryption, auth, ephemeral conversations.

**Phase 1.2** → Build notes extraction pipeline + UI for users to see/control their profile.

Core idea: Conversations flow in → ConversationAnalyzer extracts insights → Notes update → Conversation deleted.

```
User conversation
    ↓
TEMPORARY (24h): conversation_ephemeral, messages stored
    ↓
ConversationAnalyzer runs:
    - Extract AboutMe insights
    - Detect patterns
    - Link to contacts
    - Note goal progress
    ↓
PERMANENT: Update all notes tables
    ↓
Auto-delete conversation (24h later)
```

---

## New Components to Build

### 1. ConversationAnalyzer (Core Extraction Engine)

**File**: `agents/conversation_analyzer.go`

**Responsibility**: After each conversation, extract structured insights.

```go
type ConversationAnalyzer struct {
    llmClient tools.LLMProvider
    db *database.Database
}

// AnalyzeConversation: Extract insights from a completed conversation
func (ca *ConversationAnalyzer) AnalyzeConversation(
    ctx context.Context,
    userID string,
    conversationID string,
    messages []*models.Message, // {role, content, timestamp}
) (*ExtractionResult, error)

// Returns: what was extracted and with what confidence
type ExtractionResult struct {
    AboutMeUpdates []AboutMeUpdate
    PatternDetections []PatternDetection
    ContactMentions []ContactMention
    GoalProgress []GoalProgressUpdate
    ConfidenceScore float64
}
```

**Key methods**:
- `extractCommunicationStyle()` - detect direct/gentle/thoughtful
- `extractValues()` - identify stated values
- `extractPatterns()` - find recurring behaviors
- `extractContactInteractions()` - who did they mention, how did they relate?
- `extractGoalProgress()` - are active goals being worked on?

**LLM Prompt** (structured extraction):
```
Analyze this conversation. Extract ONLY high-confidence insights:

1. Communication style observed: [direct|gentle|thoughtful|mixed]
2. Values mentioned or demonstrated: [list]
3. Patterns observed:
   - [pattern]: confidence [0-1]
4. Contacts mentioned:
   - [name]: relationship type, frequency, tone observed
5. Communication goals:
   - Are they working on any of: [assertiveness|listening|vulnerability|boundaries]?

Return JSON only, no explanation.
```

### 2. ProfileUpdater (Database Mutation)

**File**: `services/profile_updater.go`

**Responsibility**: Apply extraction results to permanent tables.

```go
type ProfileUpdater struct {
    db *database.Database
}

// UpdateProfile: Apply all extracted insights to notes
func (pu *ProfileUpdater) UpdateProfile(
    ctx context.Context,
    userID string,
    result *ExtractionResult,
) error {
    // For each extraction:
    // - Check confidence threshold (default 0.6)
    // - Merge with existing data (don't overwrite, update incrementally)
    // - Update timestamps and reinforcement counts
}

// Specific update functions:
func (pu *ProfileUpdater) UpdateAboutMe(userID, key, value string, confidence float64)
func (pu *ProfileUpdater) AddOrUpdatePattern(userID, pattern string, confidence float64)
func (pu *ProfileUpdater) UpdateContactPattern(contactID string, updates map[string]interface{})
func (pu *ProfileUpdater) UpdateGoalProgress(goalID string, update string)
```

**Rules**:
- Only update if confidence > threshold
- Always merge, never overwrite (append to notes, update timestamps)
- Track reinforcement (if we see same insight twice, increase confidence)
- Allow user to reject updates (mark `is_rejected = true`)

### 3. EphemeralConversationManager (Lifecycle)

**File**: `services/ephemeral_manager.go`

**Responsibility**: Store conversation temporarily, trigger analysis, clean up.

```go
type EphemeralConversationManager struct {
    db *database.Database
    analyzer *ConversationAnalyzer
    updater *ProfileUpdater
}

// SaveConversation: Store conversation for 24h
func (ecm *EphemeralConversationManager) SaveConversation(
    ctx context.Context,
    userID string,
    conversationID string,
    messages []*models.Message,
) error {
    // Insert into conversation_ephemeral
    // Set expires_at = now + 24h
    // Queue for extraction
}

// ProcessQueue: Run extraction on pending conversations
func (ecm *EphemeralConversationManager) ProcessQueue(ctx context.Context) error {
    // Get all extraction_queue entries with status='pending'
    // For each:
    //   - Fetch conversation from conversation_ephemeral
    //   - Run ConversationAnalyzer
    //   - Save result to extraction_queue
    //   - Call ProfileUpdater to update permanent tables
    //   - Set status='completed'
}

// Cleanup: Delete expired conversations (background job)
func (ecm *EphemeralConversationManager) CleanupExpired(ctx context.Context) error {
    // DELETE FROM conversation_ephemeral WHERE expires_at < now()
    // Log what was deleted
}
```

**Usage**:
- After chat message → `SaveConversation()` (store for 24h review)
- Hourly job → `ProcessQueue()` (extract insights from completed conversations)
- Daily job → `CleanupExpired()` (delete expired conversations)

### 4. ProfileService (Read API)

**File**: `services/profile_service.go`

**Responsibility**: Expose user's notes for UI consumption.

```go
type ProfileService struct {
    db *database.Database
}

// GetUserProfile: Complete profile snapshot
func (ps *ProfileService) GetUserProfile(userID string) (*UserProfile, error)

type UserProfile struct {
    AboutMe *AboutMeProfile
    Contacts []ContactWithPatterns
    ActivePatterns []CommunicationPattern
    ActiveGoals []CommunicationGoal
    RecentReflections []ReflectionEntry
}

// GetAboutMe: Just the user's communication profile
func (ps *ProfileService) GetAboutMe(userID string) (*AboutMeProfile, error)

// GetContacts: List of people in user's life with patterns
func (ps *ProfileService) GetContacts(userID string) ([]ContactWithPatterns, error)

// GetGoals: Active communication goals
func (ps *ProfileService) GetGoals(userID string) ([]CommunicationGoal, error)

// GetPattern: Single pattern with context
func (ps *ProfileService) GetPattern(userID string, patternID int) (*CommunicationPattern, error)

// GetReflections: User's journal entries
func (ps *ProfileService) GetReflections(userID string, limit int) ([]ReflectionEntry, error)

// GetLearnings: What system learned, with confidence
func (ps *ProfileService) GetLearnings(userID string) ([]ImplicitLearning, error)
```

### 5. HTTP Handlers (API Endpoints)

**File**: `v2_1_profile_handlers.go`

**Endpoints**:

```
GET  /api/v2.1/profile            → GetUserProfile (full snapshot)
GET  /api/v2.1/profile/about-me   → GetAboutMe
GET  /api/v2.1/profile/contacts   → GetContacts
GET  /api/v2.1/profile/goals      → GetGoals
GET  /api/v2.1/profile/patterns   → GetPatterns
GET  /api/v2.1/profile/learnings  → GetLearnings (with confidence)

POST /api/v2.1/profile/reflections → AddReflection (user writes note)
GET  /api/v2.1/profile/reflections → GetReflections

POST /api/v2.1/goals               → CreateGoal (user sets goal)
PATCH /api/v2.1/goals/{id}         → UpdateGoal (update progress)

POST /api/v2.1/learnings/{id}/confirm  → ConfirmLearning
POST /api/v2.1/learnings/{id}/reject   → RejectLearning
```

**Example Response** (GET /api/v2.1/profile):
```json
{
  "about_me": {
    "communication_style": "direct",
    "tone_preference": "warm",
    "core_values": ["honesty", "growth"],
    "preferences": {
      "dislikes_small_talk": true,
      "prefers_async": true
    },
    "confidence": 0.85,
    "extracted_from_count": 12
  },
  "contacts": [
    {
      "name": "Sarah",
      "relationship_type": "professional",
      "frequency": "weekly",
      "tone_observed": "formal",
      "main_topics": ["feedback", "recognition"],
      "confidence": 0.7
    }
  ],
  "active_goals": [
    {
      "id": 1,
      "goal": "say no without apologizing",
      "status": "active",
      "confidence": 0.6,
      "progress_notes": "Getting better, but still explaining too much"
    }
  ],
  "active_patterns": [
    {
      "pattern": "avoids_conflict_then_over_explains",
      "confidence": 0.8,
      "is_growth_area": true,
      "observation_count": 4
    }
  ],
  "recent_reflections": [
    {
      "content": "Had a good talk with Mom about boundaries",
      "tags": ["mom", "boundaries"],
      "created_at": 1234567890,
      "entry_type": "reflection"
    }
  ]
}
```

---

## Data Flow: End-to-End

### Scenario: User has a chat conversation

1. **Chat Happens**
   ```
   User: "I need to tell my boss she's being unfair"
   Assistant: [helpful response]
   User: "But I'm scared she'll get mad"
   ```

2. **Conversation Saved** (EphemeralConversationManager.SaveConversation)
   ```sql
   INSERT INTO conversation_ephemeral (
     id, user_id, conversation_id, messages,
     extraction_status, expires_at
   ) VALUES (...)
   
   INSERT INTO extraction_queue (
     conversation_id, user_id, extraction_type, status
   ) VALUES (...)
   ```

3. **Extraction Runs** (ConversationAnalyzer.AnalyzeConversation)
   - LLM analyzes: "Communication style = direct, but shows fear. Extracting patterns about conflict avoidance."
   - Returns:
     ```
     {
       "AboutMeUpdates": [
         {"key": "communication_style", "value": "direct", "confidence": 0.7},
         {"key": "value", "value": "honesty", "confidence": 0.6}
       ],
       "PatternDetections": [
         {"pattern": "avoids_conflict_then_over_explains", "confidence": 0.8}
       ],
       "ContactMentions": [
         {"name": "boss", "relationship_type": "professional", "tone": "fearful"}
       ]
     }
     ```

4. **Profile Updates** (ProfileUpdater.UpdateProfile)
   ```sql
   UPDATE about_me_profile 
   SET communication_style = 'direct', 
       confidence = 0.7,
       extracted_from_count = extracted_from_count + 1
   WHERE user_id = ?
   
   INSERT INTO communication_patterns (user_id, pattern, confidence, ...)
   VALUES (?, 'avoids_conflict_then_over_explains', 0.8, ...)
   
   UPDATE contact_communication_patterns
   SET tone_observed = 'fearful',
       last_updated = now()
   WHERE contact_id = (SELECT id FROM contacts WHERE user_id = ? AND name = 'boss')
   ```

5. **Queue Updated**
   ```sql
   UPDATE extraction_queue
   SET status = 'completed',
       extraction_result = '...',
       completed_at = now()
   WHERE conversation_id = ?
   ```

6. **24h Later: Cleanup**
   ```sql
   DELETE FROM conversation_ephemeral
   WHERE expires_at < now()
   ```

7. **User Views Profile**
   - GET /api/v2.1/profile
   - Sees: "Communication style: Direct (85% confidence)"
   - Sees: "Working on: saying no to authority figures?"
   - Can confirm/reject: "Yes, that's me" or "No, that's wrong"

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    Chat Handler                         │
│           (from v2_1_chat_handlers.go)                  │
└────────────────────┬────────────────────────────────────┘
                     │
                     ↓
         ┌───────────────────────┐
         │ EphemeralManager      │
         │ .SaveConversation()   │
         └───────┬───────────────┘
                 │
    ┌────────────┴────────────┐
    ↓                         ↓
┌──────────────┐   ┌──────────────────────┐
│  Ephemeral   │   │  Extraction Queue    │
│  Conversation│   │  (pending status)    │
│  (24h TTL)   │   └──────────────────────┘
└──────────────┘
    
              [Hourly job]
                  ↓
    ┌─────────────────────────┐
    │ ConversationAnalyzer    │
    │ .AnalyzeConversation()  │
    └────────────┬────────────┘
                 │
          ┌──────┴──────────────────┐
          ↓                         ↓
    ┌──────────────┐        ┌──────────────┐
    │ ExtractionQ  │        │ ProfileUpdtr │
    │ (completed)  │        │ .UpdateProfile
    └──────────────┘        └───────┬──────┘
                                   │
         ┌─────────────────────────┤
         ↓         ↓         ↓      ↓
    ┌─────────┐ ┌──────┐ ┌────┐ ┌──────┐
    │ AboutMe │ │Cnctct│ │Ptrn│ │Goals │
    │ Profile │ │Ptrns │ │List│ │(PERM)│
    └─────────┘ └──────┘ └────┘ └──────┘
    
              [24h cleanup job]
                  ↓
    ┌─────────────────────────┐
    │ DELETE conversation_    │
    │ ephemeral WHERE < 24h   │
    └─────────────────────────┘
    
              [UI Layer]
                  ↓
    ┌─────────────────────────┐
    │  ProfileService GET APIs│
    │  /profile               │
    │  /profile/contacts      │
    │  /profile/goals         │
    └─────────────────────────┘
```

---

## Implementation Order (Phase 1.2)

1. **Database Schema**: Apply `schema_v2_1_phase_1_2.sql` ✅
2. **ConversationAnalyzer**: Extract insights from conversations
3. **ProfileUpdater**: Apply extractions to notes tables
4. **EphemeralConversationManager**: Lifecycle management
5. **ProfileService**: Read API for profile data
6. **HTTP Handlers**: Expose profile endpoints
7. **Background Jobs**: Extraction queue processor + cleanup
8. **UI**: Browser extension shows user's profile

---

## Testing Strategy

### Unit Tests
- ConversationAnalyzer: Extract correct insights from sample conversations
- ProfileUpdater: Merge logic, confidence handling
- EphemeralManager: TTL and cleanup

### Integration Tests
- Full flow: Conversation → Analysis → Profile update → Query
- Confidence thresholds: Low confidence updates not applied
- User rejection: Rejected learnings not reappear

### E2E Tests
- User has conversation
- Wait for extraction
- Query profile
- Confirm/reject learnings
- See profile update

---

## Open Design Questions

1. **Extraction Confidence Threshold**: When do we update notes?
   - Default: 0.6 (60%)
   - Tunable per extraction type?

2. **Incremental Updates vs Overwrite**: 
   - AboutMe.communication_style: overwrite if higher confidence?
   - Patterns: always append new observation, merge if similar?

3. **User Visibility into Extraction**:
   - Should user see: "System thinks you value honesty (confidence: 0.7)"?
   - Allow explicit confirm/reject?

4. **Reflection Journal Visibility**:
   - Private journal + shared notes?
   - System learns from journal, or user-controlled only?

5. **Contact Privacy**:
   - Should system infer things about contacts (e.g., "your boss is demanding")?
   - Or only store user's perception?

