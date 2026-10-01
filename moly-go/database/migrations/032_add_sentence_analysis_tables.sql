-- Migration 032: Add Sentence-Level Linguistic Analysis Infrastructure
-- Purpose: Enable context-aware subject attribution through sentence-level SVO analysis,
--          pronoun resolution tracking, and group reference management
-- Date: October 1, 2026
-- Dependencies: All prior migrations must be applied

-- ============================================================================
-- Table 1: sentence_analyses
-- ============================================================================
-- Stores Subject-Verb-Object analysis for each sentence in messages
-- Enables tracking sentence structure, subjects, and relationships
-- Used for context-aware extraction and pronoun resolution

CREATE TABLE IF NOT EXISTS sentence_analyses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    -- Sentence structure
    sentence_text TEXT NOT NULL,           -- The actual sentence
    sentence_number INTEGER NOT NULL,     -- Which sentence in message (1-based)
    sentence_start_pos INTEGER,           -- Character position in message
    sentence_end_pos INTEGER,             -- Character position in message

    -- Subject-Verb-Object (SVO) Analysis
    subject TEXT,                         -- "I", "she", "Christine", "they"
    subject_type TEXT,                    -- "pronoun", "name", "group"
    subject_resolved_to TEXT,             -- After pronoun resolution: "user", "Christine", "group_1"

    verb TEXT,                            -- "am", "is", "like", "want"
    verb_type TEXT,                       -- "copula", "transitive", "intransitive", "phrasal"
    verb_negated BOOLEAN DEFAULT 0,       -- True if verb is negated (don't, won't, etc.)

    object TEXT,                          -- "dominant", "submissive", "communication"
    object_type TEXT,                     -- "adjective", "noun", "noun_phrase", "clause"

    -- Modifiers and additional context
    modifiers TEXT,                       -- JSON: {"adverbs": ["very"], "prepositional_phrases": ["in BDSM"]}
    negation BOOLEAN DEFAULT 0,           -- Is entire sentence negated?
    negation_scope TEXT,                  -- What's being negated

    -- Semantic relationships extracted from this sentence
    relationships TEXT,                   -- JSON: ["dominance:user->Christine", "preference:user->communication"]

    -- Extraction metadata
    confidence REAL DEFAULT 0.75,         -- SVO confidence (0.0-1.0)
    parsing_method TEXT,                  -- "regex", "llm", "hybrid"

    -- Tracking
    created_at INTEGER DEFAULT (strftime('%s', 'now')),

    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(user_id, message_id, sentence_number)
);

-- Indexes for sentence analysis queries
CREATE INDEX IF NOT EXISTS idx_sentence_analyses_user_conv
    ON sentence_analyses(user_id, conversation_id);

CREATE INDEX IF NOT EXISTS idx_sentence_analyses_message
    ON sentence_analyses(user_id, message_id);

CREATE INDEX IF NOT EXISTS idx_sentence_analyses_subject
    ON sentence_analyses(user_id, subject);

CREATE INDEX IF NOT EXISTS idx_sentence_analyses_verb
    ON sentence_analyses(user_id, verb);

-- ============================================================================
-- Table 2: pronoun_resolutions
-- ============================================================================
-- Maps pronouns/references to their actual targets across conversation
-- Tracks when each pronoun resolution is valid (scope management)
-- Enables resolving facts about "she" to "Christine" etc.

CREATE TABLE IF NOT EXISTS pronoun_resolutions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    -- The pronoun or reference
    pronoun TEXT NOT NULL,                -- "she", "he", "they", "it", "both", "them"
    pronoun_type TEXT,                    -- "personal", "demonstrative", "relative", "possessive"

    -- What it refers to (antecedent)
    antecedent_type TEXT NOT NULL,        -- "name", "contact", "group", "concept", "unknown"
    antecedent_value TEXT NOT NULL,       -- "Christine", "my boss", "both of them", "unknown"
    antecedent_id INTEGER,                -- FK to contacts if antecedent is a specific contact

    -- Where this resolution was established
    message_id TEXT NOT NULL,
    sentence_position INTEGER,            -- Which sentence in message

    -- Confidence & evidence
    confidence REAL DEFAULT 0.75,         -- 0.0-1.0 (0=unknown, 1=certain)
    evidence_text TEXT,                   -- The text supporting this resolution
    resolution_method TEXT,               -- "linguistic_match", "llm_reasoning", "user_clarification", "context"

    -- Scope management: When is this resolution valid?
    scope_start_message_id TEXT,          -- From which message onwards valid
    scope_start_seq INTEGER,              -- Message sequence number (for ordering)
    scope_end_message_id TEXT,            -- Until which message (NULL = ongoing)
    scope_end_seq INTEGER,                -- Message sequence number

    is_active BOOLEAN DEFAULT 1,          -- Can be deactivated when superseded

    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now')),

    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    FOREIGN KEY (antecedent_id) REFERENCES contacts(id)
);

-- Indexes for pronoun resolution queries
CREATE INDEX IF NOT EXISTS idx_pronoun_resolutions_user_conv
    ON pronoun_resolutions(user_id, conversation_id);

CREATE INDEX IF NOT EXISTS idx_pronoun_resolutions_pronoun
    ON pronoun_resolutions(user_id, pronoun, is_active);

CREATE INDEX IF NOT EXISTS idx_pronoun_resolutions_antecedent
    ON pronoun_resolutions(user_id, antecedent_value);

CREATE INDEX IF NOT EXISTS idx_pronoun_resolutions_scope
    ON pronoun_resolutions(user_id, scope_start_seq, scope_end_seq);

-- ============================================================================
-- Table 3: group_references
-- ============================================================================
-- Tracks group references like "they", "we", "both", "all of us"
-- Stores which people are in each group
-- Enables understanding "We like intensity" = specific people like intensity

CREATE TABLE IF NOT EXISTS group_references (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    -- The group reference
    reference_pronoun TEXT NOT NULL,      -- "they", "them", "we", "us", "both", "all of us"
    reference_type TEXT NOT NULL,         -- "dual" (2), "plural" (3+), "collection"

    -- Members of the group
    member_ids TEXT NOT NULL,             -- JSON array: [1, 2, 3] or ["user_id", contact_id]
    member_names TEXT,                    -- JSON array: ["Christine", "Rigger"]

    -- Group properties
    is_user_in_group BOOLEAN DEFAULT 0,   -- Is the user part of this "we"?
    group_context TEXT,                   -- "couple", "trio", "group", "everyone involved"

    -- When this group reference was established
    message_id TEXT NOT NULL,
    established_at INTEGER DEFAULT (strftime('%s', 'now')),

    -- Confidence & evidence
    confidence REAL DEFAULT 0.75,         -- How certain are we about membership?
    evidence_text TEXT,                   -- Supporting text

    -- Scope & validity
    is_active BOOLEAN DEFAULT 1,          -- Current valid reference
    scope_end_message_id TEXT,            -- When this reference becomes invalid
    scope_end_seq INTEGER,

    created_at INTEGER DEFAULT (strftime('%s', 'now')),

    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

-- Indexes for group reference queries
CREATE INDEX IF NOT EXISTS idx_group_references_user_conv
    ON group_references(user_id, conversation_id);

CREATE INDEX IF NOT EXISTS idx_group_references_pronoun
    ON group_references(user_id, reference_pronoun, is_active);

CREATE INDEX IF NOT EXISTS idx_group_references_active
    ON group_references(user_id, is_active);

-- ============================================================================
-- Table 4: extraction_sentence_linking
-- ============================================================================
-- Links extracted facts to the sentence they came from
-- Enables tracing: fact → sentence → SVO → pronoun resolution
-- Stores resolved subject after all pronoun resolution

CREATE TABLE IF NOT EXISTS extraction_sentence_linking (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- The extracted fact
    context_attribute_id INTEGER NOT NULL,  -- FK to context_attributes

    -- What sentence it came from
    sentence_analysis_id INTEGER NOT NULL,  -- FK to sentence_analyses

    -- What subject it was attributed to (after resolution)
    subject_resolved_to TEXT NOT NULL,      -- "user", "Christine", "group_1", etc.
    subject_type TEXT,                      -- "user", "contact", "group"

    -- How certain are we about this attribution?
    attribution_confidence REAL DEFAULT 0.75,

    -- Metadata
    created_at INTEGER DEFAULT (strftime('%s', 'now')),

    FOREIGN KEY (context_attribute_id) REFERENCES context_attributes(id),
    FOREIGN KEY (sentence_analysis_id) REFERENCES sentence_analyses(id),

    UNIQUE(context_attribute_id)  -- Each fact linked to exactly one sentence
);

-- Indexes for extraction linking queries
CREATE INDEX IF NOT EXISTS idx_extraction_sentence_linking_fact
    ON extraction_sentence_linking(context_attribute_id);

CREATE INDEX IF NOT EXISTS idx_extraction_sentence_linking_sentence
    ON extraction_sentence_linking(sentence_analysis_id);

CREATE INDEX IF NOT EXISTS idx_extraction_sentence_linking_subject
    ON extraction_sentence_linking(subject_resolved_to);

-- ============================================================================
-- Migration Complete
-- ============================================================================
-- These 4 tables enable:
-- ✓ Sentence-level SVO parsing
-- ✓ Pronoun resolution with scope tracking
-- ✓ Group membership tracking
-- ✓ Complete audit trail of fact extraction
-- ✓ Context-aware subject attribution
-- ✓ Query "What do we know about X?" with proper resolution
