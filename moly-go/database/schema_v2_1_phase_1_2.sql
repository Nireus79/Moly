-- Moly V2.1 Phase 1.2 Schema: Notes-Based Model
-- Privacy-first, token-efficient, user-controlled
-- Replaces raw chat history with structured insights

-- ============================================================================
-- PERMANENT TABLES: User's Knowledge Base (Stays Forever)
-- ============================================================================

-- Enhanced AboutMe: Structured profile of user's communication
CREATE TABLE IF NOT EXISTS about_me_profile (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL UNIQUE,

    -- Communication style (discrete, categorical)
    communication_style TEXT, -- "direct", "gentle", "thoughtful", NULL
    tone_preference TEXT, -- "warm", "professional", "casual", "formal"
    pace_preference TEXT, -- "slow_deliberate", "quick_efficient", NULL

    -- Values (JSON array, user-controlled)
    core_values TEXT, -- JSON: ["honesty", "growth", "respect"]

    -- Preferences (JSON map)
    preferences TEXT, -- JSON: {"dislikes_small_talk": true, "prefers_async": true}

    -- Auto-updated notes about their communication
    communication_notes TEXT,

    -- Metadata about extraction
    extracted_from_count INTEGER DEFAULT 0, -- how many conversations contributed
    confidence REAL DEFAULT 0.0, -- 0.0-1.0, system confidence in profile
    last_updated INTEGER,
    version INTEGER DEFAULT 1,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Contact reference: People in user's life (no private details)
CREATE TABLE IF NOT EXISTS contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,

    name TEXT NOT NULL, -- "Sarah", "Mom", "Boss", etc.
    relationship_type TEXT, -- "professional", "family", "romantic", "friendship"
    context TEXT, -- "My boss", "Best friend from college" (user-provided)

    -- Metadata
    first_mentioned INTEGER, -- timestamp when first mentioned
    times_mentioned INTEGER DEFAULT 1, -- how many times referenced
    last_mentioned INTEGER, -- timestamp of most recent mention

    created_at INTEGER,
    updated_at INTEGER,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, name)
);

-- How user relates to each contact (patterns, not content)
CREATE TABLE IF NOT EXISTS contact_communication_patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    contact_id INTEGER NOT NULL,

    -- Frequency and mode
    frequency TEXT, -- "daily", "weekly", "monthly", "rare"
    preferred_medium TEXT, -- "in_person", "phone", "text", "email"

    -- Observed patterns (JSON)
    patterns TEXT, -- JSON: {"initiates_rarely": true, "responds_quickly": true}

    -- Tone observed with this contact
    tone_observed TEXT, -- "warm", "formal", "tense", "relaxed"

    -- Communication focus
    main_topics TEXT, -- JSON: ["feedback", "boundaries", "conflict_resolution"]

    -- Recent outcome from user's perspective
    recent_outcome TEXT, -- "went well", "difficult", "unresolved"

    -- User's notes about this relationship
    user_notes TEXT,

    -- Metadata
    last_updated INTEGER,
    confidence REAL DEFAULT 0.5, -- how sure system is about these patterns

    FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
);

-- General communication patterns observed
CREATE TABLE IF NOT EXISTS communication_patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,

    -- The pattern itself
    pattern TEXT NOT NULL, -- "avoids_conflict_then_over_explains"
    pattern_category TEXT, -- "avoidance", "assertiveness", "clarity", "listening"

    -- Metadata
    confidence REAL DEFAULT 0.5, -- 0-1, how confident system is
    first_observed INTEGER,
    last_observed INTEGER,
    observation_count INTEGER DEFAULT 1, -- reinforcement count

    -- Status
    is_active BOOLEAN DEFAULT true, -- still relevant?
    is_growth_area BOOLEAN DEFAULT false, -- user working on this?

    -- Context
    context_notes TEXT, -- why we think this is a pattern

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Active communication goals user is working on
CREATE TABLE IF NOT EXISTS communication_goals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,

    -- The goal
    goal TEXT NOT NULL, -- "say no without apologizing"
    category TEXT, -- "assertiveness", "listening", "vulnerability", "boundaries"

    -- Status
    status TEXT DEFAULT 'active', -- "active", "achieved", "paused", "abandoned"

    -- Timeline
    started_at INTEGER,
    target_date INTEGER, -- optional deadline
    achieved_at INTEGER, -- when completed

    -- User's progress notes
    progress_notes TEXT, -- user writes this
    confidence REAL DEFAULT 0.5, -- user's confidence in achieving it

    -- Metadata
    created_at INTEGER,
    last_updated INTEGER,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Optional: User's own reflection journal
CREATE TABLE IF NOT EXISTS reflection_journal (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,

    -- Entry (user writes this, optional)
    content TEXT NOT NULL,
    tags TEXT, -- JSON: ["mom", "boundaries", "growth"]

    -- Metadata
    created_at INTEGER,
    about_contact_id INTEGER, -- optional: if about a specific person
    entry_type TEXT, -- "reflection", "goal_update", "pattern_notice", "free_form"

    -- Privacy
    is_private BOOLEAN DEFAULT true, -- user controls visibility

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (about_contact_id) REFERENCES contacts(id) ON DELETE SET NULL
);

-- System's implicit learning (what we inferred without asking)
CREATE TABLE IF NOT EXISTS implicit_learning (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,

    -- What we learned
    learning_type TEXT, -- "communication_style", "value", "preference", "pattern"
    learning_key TEXT, -- "prefers_direct_language"
    learning_value TEXT, -- the actual value

    -- Confidence & source
    confidence REAL DEFAULT 0.5, -- 0-1
    source TEXT, -- "conversation_pattern", "explicit_statement", "repeated_behavior"

    -- Lifecycle
    first_extracted INTEGER,
    last_reinforced INTEGER,
    reinforcement_count INTEGER DEFAULT 1,

    -- User control (can explicitly confirm or reject)
    is_confirmed BOOLEAN DEFAULT false, -- user said "yes, that's me"
    is_rejected BOOLEAN DEFAULT false, -- user said "no, that's wrong"
    user_feedback TEXT, -- why they accepted/rejected

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ============================================================================
-- EPHEMERAL TABLES: Temporary, Auto-Delete (24h TTL)
-- ============================================================================

-- Conversation: Kept only 24h for review, then deleted
CREATE TABLE IF NOT EXISTS conversation_ephemeral (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,

    -- Conversation metadata
    conversation_id TEXT NOT NULL,
    started_at INTEGER,
    ended_at INTEGER,

    -- Messages (JSON array: [{role, content, timestamp}])
    -- Only kept for 24h review, then deleted
    messages TEXT,

    -- Extraction status
    extraction_status TEXT DEFAULT 'pending', -- "pending", "processing", "completed", "failed"
    extraction_attempted_at INTEGER,
    extraction_completed_at INTEGER,

    -- Lifecycle
    created_at INTEGER,
    expires_at INTEGER, -- 24h from creation, auto-delete trigger

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Extraction queue: Pipeline for processing conversations
CREATE TABLE IF NOT EXISTS extraction_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,

    -- What to extract
    extraction_type TEXT, -- "about_me_update", "pattern_detection", "goal_progress"
    status TEXT DEFAULT 'pending', -- "pending", "processing", "completed", "skipped", "failed"

    -- Result
    extraction_result TEXT, -- JSON of what was extracted
    confidence REAL DEFAULT 0.0,

    -- Processing
    queued_at INTEGER,
    attempted_at INTEGER,
    completed_at INTEGER,
    error_message TEXT,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ============================================================================
-- SESSION MANAGEMENT (From Phase 1, unchanged)
-- ============================================================================

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    code TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    last_active INTEGER NOT NULL,
    device_name TEXT,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, code)
);

-- ============================================================================
-- INDEXES
-- ============================================================================

-- AboutMe
CREATE INDEX IF NOT EXISTS idx_about_me_user_id ON about_me_profile(user_id);
CREATE INDEX IF NOT EXISTS idx_about_me_updated ON about_me_profile(last_updated DESC);

-- Contacts
CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_last_mentioned ON contacts(last_mentioned DESC);

-- Contact patterns
CREATE INDEX IF NOT EXISTS idx_contact_patterns_contact_id ON contact_communication_patterns(contact_id);
CREATE INDEX IF NOT EXISTS idx_contact_patterns_updated ON contact_communication_patterns(last_updated DESC);

-- Communication patterns
CREATE INDEX IF NOT EXISTS idx_patterns_user_id ON communication_patterns(user_id);
CREATE INDEX IF NOT EXISTS idx_patterns_active ON communication_patterns(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_patterns_growth ON communication_patterns(is_growth_area) WHERE is_growth_area = true;

-- Goals
CREATE INDEX IF NOT EXISTS idx_goals_user_id ON communication_goals(user_id);
CREATE INDEX IF NOT EXISTS idx_goals_status ON communication_goals(status);
CREATE INDEX IF NOT EXISTS idx_goals_active ON communication_goals(user_id, status) WHERE status = 'active';

-- Reflection journal
CREATE INDEX IF NOT EXISTS idx_journal_user_id ON reflection_journal(user_id);
CREATE INDEX IF NOT EXISTS idx_journal_contact_id ON reflection_journal(about_contact_id);
CREATE INDEX IF NOT EXISTS idx_journal_created ON reflection_journal(created_at DESC);

-- Implicit learning
CREATE INDEX IF NOT EXISTS idx_learning_user_id ON implicit_learning(user_id);
CREATE INDEX IF NOT EXISTS idx_learning_confirmed ON implicit_learning(is_confirmed) WHERE is_confirmed = true;
CREATE INDEX IF NOT EXISTS idx_learning_type ON implicit_learning(learning_type);

-- Ephemeral
CREATE INDEX IF NOT EXISTS idx_ephemeral_user_id ON conversation_ephemeral(user_id);
CREATE INDEX IF NOT EXISTS idx_ephemeral_expires ON conversation_ephemeral(expires_at);
CREATE INDEX IF NOT EXISTS idx_extraction_status ON extraction_queue(status);
CREATE INDEX IF NOT EXISTS idx_extraction_user ON extraction_queue(user_id);

-- Sessions
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at DESC);
