# Phase 1.2 Schema Redesign: Notes-Based Model

## Core Principle
**No raw chat storage.** Only structured insights extracted from conversations. User-controlled, privacy-respecting, token-efficient.

## Data Model Philosophy

```
Conversation (ephemeral, 24h TTL)
    ↓ [ConversationAnalyzer extracts]
    ↓
Structured Notes (permanent, always in context)
    - AboutMe Profile
    - Contact Map + Patterns  
    - Communication Patterns
    - Active Goals
    - Reflection Journal
```

## Permanent Tables (User's Knowledge Base)

### 1. **users** (unchanged)
```sql
id TEXT PRIMARY KEY
created_at INTEGER
last_active INTEGER
context_level TEXT -- "minimal", "partial", "comprehensive"
safety_tier TEXT
```

### 2. **about_me_profile** (replaces raw about_me, structured for incremental updates)
```sql
id INTEGER PRIMARY KEY
user_id TEXT NOT NULL UNIQUE

-- Communication style (discrete, categorical)
communication_style TEXT -- "direct", "gentle", "thoughtful", null
tone_preference TEXT -- "warm", "professional", "casual", "formal"
pace_preference TEXT -- "slow & deliberate", "quick & efficient", null

-- Values (JSON array, user-controlled)
core_values TEXT -- ["honesty", "growth", "respect"]

-- Preferences (JSON, user-controlled)
preferences TEXT -- {
                 --   "dislikes_small_talk": true,
                 --   "prefers_async": true,
                 --   "needs_explicit_praise": false
                 -- }

-- Notes about their communication (auto-updated)
communication_notes TEXT -- "Tends to over-explain when defensive"

-- Metadata
extracted_from_count INTEGER -- how many conversations contributed
confidence REAL -- 0.0-1.0, how sure are we?
last_updated INTEGER
version INTEGER DEFAULT 1
```

### 3. **contacts** (person reference, no private details)
```sql
id INTEGER PRIMARY KEY
user_id TEXT NOT NULL

name TEXT NOT NULL -- "Sarah", "Mom", "Alex"
relationship_type TEXT -- "professional", "family", "romantic", "friendship"
context TEXT -- "My boss", "Best friend from college" (user-provided or inferred)

-- Metadata
first_mentioned INTEGER -- timestamp
times_mentioned INTEGER -- count
last_mentioned INTEGER

FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
UNIQUE(user_id, name)
```

### 4. **contact_communication_patterns** (HOW user relates to this person, not WHAT they discussed)
```sql
id INTEGER PRIMARY KEY
contact_id INTEGER NOT NULL

-- Frequency & mode
frequency TEXT -- "daily", "weekly", "monthly", "rare"
preferred_medium TEXT -- "in_person", "phone", "text", "email"

-- Pattern observations (high-level only, no specifics)
patterns TEXT -- JSON
-- {
--   "initiates_rarely": true,
--   "responds_quickly": true,
--   "tends_to_be_formal": true,
--   "discusses_emotions_rarely": true
-- }

-- Tone with this contact
tone_observed TEXT -- "warm", "formal", "tense", "relaxed"

-- Growth tracking
conversation_topic TEXT -- "feedback", "boundaries", "conflict_resolution"
recent_outcome TEXT -- "went well", "difficult", "unresolved" (user input)
notes TEXT -- User's own reflection on this relationship

last_updated INTEGER

FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
```

### 5. **communication_patterns** (General patterns, not tied to specific people)
```sql
id INTEGER PRIMARY KEY
user_id TEXT NOT NULL

-- Pattern observed
pattern TEXT NOT NULL -- "avoids_conflict_then_over_explains"
                      -- "gravitates_toward_assertiveness_topics"
                      -- "practices_direct_feedback"
                      -- "prefers_written_communication"

-- Confidence & metadata
confidence REAL -- how sure we are (0-1)
first_observed INTEGER
last_observed INTEGER
observation_count INTEGER -- how many times we've seen this pattern

-- Status
is_active BOOLEAN -- still relevant?
is_growth_area BOOLEAN -- is user working on this?

notes TEXT -- Context about this pattern

FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
```

### 6. **communication_goals** (What user is actively working on)
```sql
id INTEGER PRIMARY KEY
user_id TEXT NOT NULL

goal TEXT NOT NULL -- "say no without apologizing"
                   -- "give feedback more directly"
                   -- "listen more, talk less"

status TEXT DEFAULT 'active' -- "active", "achieved", "paused"
category TEXT -- "assertiveness", "listening", "vulnerability", "boundaries"

-- Progress tracking
started_at INTEGER
target_date INTEGER (optional)
progress_notes TEXT -- User's notes on how it's going

-- Confidence
confidence REAL -- user's own confidence in achieving this

FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
```

### 7. **reflection_journal** (Optional: user-written notes)
```sql
id INTEGER PRIMARY KEY
user_id TEXT NOT NULL

-- Entry (user writes this explicitly, optional)
content TEXT NOT NULL
tags TEXT -- ["mom", "boundaries", "growth"] (user-provided)

-- Metadata
created_at INTEGER
about_contact_id INTEGER (optional, if about a specific person)
is_private BOOLEAN DEFAULT true -- user controls

-- Reflection type
entry_type TEXT -- "reflection", "goal_update", "pattern_notice"

FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
FOREIGN KEY (about_contact_id) REFERENCES contacts(id) ON DELETE SET NULL
```

### 8. **implicit_learning** (Renamed: what system learned without asking)
```sql
id INTEGER PRIMARY KEY
user_id TEXT NOT NULL

-- What we learned
learning_type TEXT -- "communication_style", "value", "preference", "pattern"
learning_key TEXT -- "prefers_direct_language", "values_honesty"
learning_value TEXT -- the actual value learned

-- Confidence
confidence REAL -- 0.0-1.0
source TEXT -- "conversation_pattern", "explicit_statement", "repeated_behavior"

-- Lifecycle
first_extracted INTEGER
last_reinforced INTEGER
reinforcement_count INTEGER

-- User control
is_confirmed BOOLEAN -- user explicitly said "yes, that's me"
is_rejected BOOLEAN -- user explicitly said "no, that's wrong"

FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
```

---

## Ephemeral Tables (Temporary, Auto-Delete)

### 9. **conversation_ephemeral** (24-hour TTL, deleted after analysis)
```sql
id TEXT PRIMARY KEY
user_id TEXT NOT NULL

-- Conversation metadata
conversation_id TEXT NOT NULL
started_at INTEGER
ended_at INTEGER

-- Raw content (only kept 24h)
messages TEXT -- JSON array of {role, content, timestamp}
              -- Can be reviewed in this window, then deleted

-- Processing status
extraction_status TEXT -- "pending", "processing", "completed"
extraction_attempted_at INTEGER
extraction_completed_at INTEGER

-- TTL
created_at INTEGER
expires_at INTEGER -- 24h from creation, auto-delete

FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
```

### 10. **extraction_queue** (Processing pipeline)
```sql
id INTEGER PRIMARY KEY
conversation_id TEXT NOT NULL
user_id TEXT NOT NULL

-- What to extract
extraction_type TEXT -- "about_me_update", "pattern_detection", "goal_progress"
status TEXT -- "pending", "processing", "completed", "skipped"

-- Result
extraction_result TEXT -- JSON of what was extracted
confidence REAL

-- Metadata
queued_at INTEGER
attempted_at INTEGER
completed_at INTEGER
error_message TEXT (if failed)

FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
```

---

## Removed from Phase 1
- `chat_messages` - we keep only ephemeral conversations
- `interactions` - replaced by patterns tables
- `suggestion_choices` - user guides this via reflection journal
- `behavior_patterns` - replaced by communication_patterns table

---

## Sessions Table (Keep from Phase 1)
```sql
-- Same as Phase 1
id TEXT PRIMARY KEY
user_id TEXT NOT NULL
code TEXT NOT NULL
expires_at INTEGER
created_at INTEGER
last_active INTEGER
device_name TEXT

UNIQUE(user_id, code)
```

---

## Key Design Decisions

### Privacy & UX
- ✅ No transcript storage → user never feels "monitored"
- ✅ User controls reflection journal → agency over their notes
- ✅ Extraction confidence scores → system is transparent about what it "knows"
- ✅ Explicit rejection option → user can correct the system

### Token Efficiency  
- ✅ Context always fits in ~2-3k tokens
- ✅ AboutMe: ~50 tokens
- ✅ Active goals: ~100 tokens
- ✅ Top patterns: ~100 tokens
- ✅ Recent contact interactions: ~100 tokens
- ✅ Reflection journal (last 3 entries): ~200 tokens

### Incrementality
- ✅ No need to re-read history
- ✅ Each conversation updates notes once
- ✅ System learns continuously without scaling costs

### Data Lifecycle
```
Conversation → 24h ephemeral → Extracted to notes → Deleted
              (can review)    (permanent)
```

---

## Extraction Algorithm (ConversationAnalyzer)

After each conversation:

```
1. Parse conversation
2. For each insight:
   a. Does it update AboutMe? (communication style, values, preferences)
   b. Does it mention a contact? (Update contact_communication_patterns)
   c. Does it reveal a pattern? (Add to communication_patterns if new)
   d. Does it relate to active goals? (Update goal progress)
3. Store extraction record
4. Queue ephemeral conversation for deletion (24h later)
```

Extraction only happens if confidence > threshold (e.g., 0.6).

---

## Open Questions for Phase 1.2

1. **Extraction confidence**: How confident must system be to update notes?
2. **User correction flow**: How does user explicitly reject/confirm extractions?
3. **Reflection journal**: Simple text, or tagged/structured?
4. **Contact patterns**: Auto-infer or require user input?
5. **AboutMe evolution**: How do we show user how their profile is changing?

