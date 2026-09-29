-- Migration 020: Add Subject Attribution Support
-- Purpose: Enable tracking WHO (subject) has WHAT (property) in multi-person messages
-- Dependencies: All prior migrations must be applied
-- Date: Sept 29, 2026

-- Table: subject_attributions
-- Stores subject identification for each extracted fact
-- Enables tracking which person/contact has which characteristic
CREATE TABLE IF NOT EXISTS subject_attributions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    -- Subject identification (WHO)
    subject_type TEXT NOT NULL, -- 'user', 'contact', 'pronoun', 'name'
    subject_value TEXT NOT NULL, -- 'user', 'she', 'christine', 'emma'
    subject_confidence REAL DEFAULT 0.75, -- 0.0-1.0

    -- Linked fact (WHAT)
    fact_id INTEGER, -- FK to context_attributes if applicable
    fact_type TEXT, -- Type of fact ('characteristic', 'preference', 'interest', 'location', etc.)
    fact_value TEXT, -- The extracted value

    -- Source tracking
    extraction_source TEXT, -- 'linguistic_parser', 'llm', 'profile_parser', 'clarification'
    message_id TEXT, -- Which message this was extracted from

    -- Metadata
    is_negated BOOLEAN DEFAULT 0, -- Whether this is a negation (NOT X)
    confidence REAL DEFAULT 0.75, -- Overall extraction confidence
    evidence_text TEXT, -- Quote from message supporting this extraction

    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now')),

    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    UNIQUE(user_id, conversation_id, subject_value, fact_type)
);

-- Index for fast lookup by user and conversation
CREATE INDEX IF NOT EXISTS idx_subject_attributions_user_conv
    ON subject_attributions(user_id, conversation_id);

-- Index for subject lookup
CREATE INDEX IF NOT EXISTS idx_subject_attributions_subject
    ON subject_attributions(user_id, subject_value);

-- Index for fact lookup
CREATE INDEX IF NOT EXISTS idx_subject_attributions_fact
    ON subject_attributions(user_id, fact_type);

-- Table: profile_extractions
-- Stores structured profile data extracted from messages
-- Enables storing parsed gender, role, interests, etc. with source tracking
CREATE TABLE IF NOT EXISTS profile_extractions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    -- Subject (whose profile)
    subject_type TEXT NOT NULL, -- 'user', 'contact'
    subject_value TEXT NOT NULL, -- 'user' or contact name

    -- Profile attributes
    attribute_key TEXT NOT NULL, -- 'gender', 'role', 'interests', 'age', 'location'
    attribute_value TEXT NOT NULL, -- The actual value
    is_list BOOLEAN DEFAULT 0, -- Whether this is a comma-separated list
    list_values TEXT, -- JSON array if is_list=1: ["Bondage", "Aftercare"]

    -- Extraction metadata
    extraction_format TEXT, -- 'fetlife', 'keyvalue', 'unstructured'
    confidence REAL DEFAULT 0.75, -- 0.0-1.0
    raw_match TEXT, -- Original text that was matched

    -- Source tracking
    message_id TEXT, -- Which message this came from
    extraction_source TEXT, -- 'profile_parser', 'llm', 'user_input'

    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now')),

    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

-- Index for fast profile lookup
CREATE INDEX IF NOT EXISTS idx_profile_extractions_subject
    ON profile_extractions(user_id, subject_value);

-- Index for attribute lookup
CREATE INDEX IF NOT EXISTS idx_profile_extractions_attribute
    ON profile_extractions(user_id, attribute_key);

-- Alter context_attributes table to include subject attribution
-- Add column to track which subject this attribute belongs to
ALTER TABLE context_attributes ADD COLUMN attributed_to TEXT;
ALTER TABLE context_attributes ADD COLUMN attribution_source TEXT;
ALTER TABLE context_attributes ADD COLUMN attribution_confidence REAL DEFAULT 0.75;

-- Index for subject-attributed queries
CREATE INDEX IF NOT EXISTS idx_context_attributes_attributed_to
    ON context_attributes(user_id, attributed_to);

-- Alter clarification_responses table to include subject tracking
ALTER TABLE clarification_responses ADD COLUMN extracted_subjects TEXT; -- JSON array of subjects
ALTER TABLE clarification_responses ADD COLUMN profile_data TEXT; -- JSON of parsed profile attributes
ALTER TABLE clarification_responses ADD COLUMN extraction_count INTEGER DEFAULT 0;

-- Index for clarifications with extractions
CREATE INDEX IF NOT EXISTS idx_clarification_responses_extracted
    ON clarification_responses(user_id, extraction_count);

-- Table: extraction_cache_metadata
-- Tracks LLM cache effectiveness for extractions
-- Enables monitoring cache hit rates and performance
CREATE TABLE IF NOT EXISTS extraction_cache_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,

    -- Cache entry info
    cache_key TEXT NOT NULL UNIQUE, -- MD5 hash of input
    cache_type TEXT NOT NULL, -- 'extraction', 'intent', 'profile', 'topic'
    input_size INTEGER, -- Size of input message

    -- Hit tracking
    total_hits INTEGER DEFAULT 0,
    last_hit_at INTEGER,
    first_cached_at INTEGER DEFAULT (strftime('%s', 'now')),

    -- Performance metrics
    extraction_time_ms INTEGER, -- Time to extract (LLM call)
    fallback_time_ms INTEGER, -- Time to extract via fallback
    used_fallback BOOLEAN DEFAULT 0,

    -- Metadata
    extraction_quality TEXT, -- 'high', 'medium', 'low'
    message_length INTEGER,

    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Index for cache statistics
CREATE INDEX IF NOT EXISTS idx_extraction_cache_metadata_user
    ON extraction_cache_metadata(user_id);

-- Index for cache type queries
CREATE INDEX IF NOT EXISTS idx_extraction_cache_metadata_type
    ON extraction_cache_metadata(cache_type);

-- Table: multi_person_tracking_log
-- Tracks success/failure of multi-person message processing
-- Enables debugging and monitoring of subject attribution
CREATE TABLE IF NOT EXISTS multi_person_tracking_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    message_id TEXT NOT NULL,
    message_text TEXT,
    message_length INTEGER,

    -- Processing results
    subjects_detected INTEGER DEFAULT 0, -- How many subjects found
    subjects TEXT, -- JSON array of detected subjects
    extractions_created INTEGER DEFAULT 0,

    -- Success/failure tracking
    status TEXT, -- 'success', 'partial', 'failed'
    error_message TEXT,

    -- Performance metrics
    processing_time_ms INTEGER,
    extraction_method TEXT, -- 'linguistic_parser', 'llm', 'fallback'
    cache_hit BOOLEAN DEFAULT 0,

    created_at INTEGER DEFAULT (strftime('%s', 'now')),

    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

-- Index for monitoring
CREATE INDEX IF NOT EXISTS idx_multi_person_tracking_log_user
    ON multi_person_tracking_log(user_id);

-- Index for failure analysis
CREATE INDEX IF NOT EXISTS idx_multi_person_tracking_log_status
    ON multi_person_tracking_log(status);

-- Table: deduplication_log
-- Tracks contact deduplication attempts and results
-- Enables monitoring for "Christine vs the girl" scenarios
CREATE TABLE IF NOT EXISTS deduplication_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,

    -- Candidates for deduplication
    contact_id_1 INTEGER,
    contact_name_1 TEXT,
    contact_id_2 INTEGER,
    contact_name_2 TEXT,

    -- Decision
    merged BOOLEAN DEFAULT 0, -- Whether they were merged
    confidence REAL, -- Confidence score for merge decision
    reason TEXT, -- Why merge was or wasn't done

    -- Metadata
    message_id TEXT, -- Which message triggered this check
    processing_stage TEXT, -- Which layer/component detected it

    created_at INTEGER DEFAULT (strftime('%s', 'now')),

    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (contact_id_1) REFERENCES user_contacts(id),
    FOREIGN KEY (contact_id_2) REFERENCES user_contacts(id)
);

-- Index for dedup analysis
CREATE INDEX IF NOT EXISTS idx_deduplication_log_user
    ON deduplication_log(user_id);

-- Index for merge tracking
CREATE INDEX IF NOT EXISTS idx_deduplication_log_merged
    ON deduplication_log(merged);
