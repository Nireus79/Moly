-- PHASE 4: CLEAN DATABASE SCHEMA REDESIGN
-- Complete schema redesign from scratch for Phase 1-3 architecture
-- Date: Sept 30, 2026
-- Strategy: Export/import from old schema, no dual-write complexity

-- ========================================
-- AUTHENTICATION & USER PROFILE
-- ========================================

-- Users (core identity)
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Sessions (authentication tokens)
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_token ON sessions(token);

-- User Profile (simplified, clean)
CREATE TABLE user_profile (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    communication_style TEXT,
    values TEXT,  -- JSON array
    goals TEXT,   -- JSON array
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ========================================
-- RELATIONSHIPS & CONTACTS
-- ========================================

-- Contacts (relationships)
CREATE TABLE contacts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    relationship_type TEXT,
    characteristics TEXT,  -- JSON: {dominant: true, sensitive: false}
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, name)
);

CREATE INDEX idx_contacts_user ON contacts(user_id);
CREATE INDEX idx_contacts_name ON contacts(name);

-- ========================================
-- CONVERSATIONS & MESSAGES
-- ========================================

-- Conversations (chat sessions)
CREATE TABLE conversations (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    archived_at TIMESTAMP  -- NULL = active, set when archived
);

CREATE INDEX idx_conversations_user ON conversations(user_id);
CREATE INDEX idx_conversations_archived ON conversations(archived_at);

-- Messages (user and Moly messages)
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender TEXT NOT NULL,  -- "user" or "moly"
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_conversation ON messages(conversation_id);
CREATE INDEX idx_messages_sender ON messages(sender);
CREATE UNIQUE INDEX idx_messages_unique ON messages(conversation_id, sender, created_at);

-- ========================================
-- EXTRACTION (PHASE 1)
-- Locked extraction artifacts with principle metadata
-- ========================================

-- Locked extraction artifacts (Phase 1)
CREATE TABLE extractions (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    is_locked BOOLEAN DEFAULT false,
    locked_at TIMESTAMP,
    lock_reason TEXT,
    extraction_confidence FLOAT,
    extraction_source TEXT DEFAULT 'llm',  -- "llm", "user_corrected", "clarification"
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(conversation_id, message_id)
);

CREATE INDEX idx_extractions_locked ON extractions(is_locked);
CREATE INDEX idx_extractions_confidence ON extractions(extraction_confidence);

-- Extracted entities with principle metadata (Phase 1)
CREATE TABLE extracted_entities (
    id TEXT PRIMARY KEY,
    extraction_id TEXT NOT NULL REFERENCES extractions(id) ON DELETE CASCADE,
    type TEXT NOT NULL,           -- "User", "Contact", "Property", "Value", "Goal", "Concern"
    value TEXT NOT NULL,
    subject TEXT NOT NULL,        -- "user", contact_name, etc
    principles TEXT,              -- JSON: ["Harm Prevention", "User Autonomy"]
    antonyms TEXT,                -- JSON: ["submissive", "passive"]
    antonym_source TEXT,          -- "llm_reasoning", "system_map"
    bidirectional BOOLEAN DEFAULT false,
    confidence FLOAT NOT NULL,
    evidence TEXT,                -- Exact quote from message
    source TEXT DEFAULT 'llm',    -- "llm", "user_corrected", "clarification"
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(extraction_id, value, subject)
);

CREATE INDEX idx_entities_extraction ON extracted_entities(extraction_id);
CREATE INDEX idx_entities_type ON extracted_entities(type);
CREATE INDEX idx_entities_subject ON extracted_entities(subject);

-- Detected conflicts (Phase 2)
CREATE TABLE detected_conflicts (
    id TEXT PRIMARY KEY,
    extraction_id TEXT NOT NULL REFERENCES extractions(id) ON DELETE CASCADE,
    entity_a_id TEXT NOT NULL REFERENCES extracted_entities(id) ON DELETE CASCADE,
    entity_b_id TEXT NOT NULL REFERENCES extracted_entities(id) ON DELETE CASCADE,
    type TEXT NOT NULL,           -- "characteristic_conflict", "value_conflict"
    severity TEXT DEFAULT 'medium',  -- "low", "medium", "high"
    description TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    resolution TEXT
);

CREATE INDEX idx_conflicts_extraction ON detected_conflicts(extraction_id);
CREATE INDEX idx_conflicts_severity ON detected_conflicts(severity);
CREATE INDEX idx_conflicts_resolved ON detected_conflicts(resolved_at);

-- ========================================
-- CLARIFICATION FLOW (PHASE 2)
-- Questions with principle tracking and layer information
-- ========================================

-- Clarification questions with full metadata
CREATE TABLE clarification_questions (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    layer TEXT NOT NULL,          -- "4", "5", "6-7", "8", "9", "10"
    question_type TEXT NOT NULL,  -- "gap", "conflict", "principle", "socratic", "topic_shift"
    question_text TEXT NOT NULL,
    principle_basis TEXT,         -- "Harm Prevention", "User Autonomy", etc (NULL if not principle-based)
    reason TEXT NOT NULL,         -- Why this question was asked

    -- Links to what triggered this question
    linked_conflict_id TEXT REFERENCES detected_conflicts(id),
    linked_gap TEXT,              -- Gap name if Layer 4
    linked_principle TEXT,        -- Principle name if Layer 6-7

    -- Status tracking
    status TEXT DEFAULT 'pending', -- "pending", "answered", "clarified"
    answer_text TEXT,
    answered_at TIMESTAMP,

    -- Deduplication tracking
    similar_questions_recent INT DEFAULT 0,
    deduplicated BOOLEAN DEFAULT false,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_clarif_conversation (conversation_id),
    INDEX idx_clarif_layer (layer),
    INDEX idx_clarif_type (question_type),
    INDEX idx_clarif_principle (principle_basis),
    INDEX idx_clarif_status (status)
);

-- ========================================
-- RESPONSE VALIDATION (PHASE 3)
-- Audit trail for response validation
-- ========================================

-- Response validation audit trail
CREATE TABLE response_validations (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    response_text TEXT NOT NULL,
    constraints_applied INT,
    constraint_details TEXT,  -- JSON array of constraints
    violations_detected INT DEFAULT 0,
    passed_validation BOOLEAN,
    fallback_used BOOLEAN DEFAULT false,
    validated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_validation_conversation (conversation_id),
    INDEX idx_validation_user (user_id),
    INDEX idx_validation_violations (violations_detected)
);

-- ========================================
-- PRINCIPLE TRACKING
-- Principle violations and resolutions
-- ========================================

-- Principle violations
CREATE TABLE principle_violations (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    extracted_entity_id TEXT REFERENCES extracted_entities(id) ON DELETE SET NULL,
    principle TEXT NOT NULL,     -- "Harm Prevention", "User Autonomy", etc
    violation_type TEXT,         -- "self_harm", "autonomy", "consent"
    severity TEXT DEFAULT 'medium', -- "low", "medium", "high", "critical"
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    resolution TEXT,

    INDEX idx_violation_conversation (conversation_id),
    INDEX idx_violation_principle (principle),
    INDEX idx_violation_severity (severity)
);

-- ========================================
-- LEARNING & FACTS (Optional, Future)
-- Learned facts about user
-- ========================================

-- Learned facts about user
CREATE TABLE learned_facts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fact_type TEXT NOT NULL,     -- "characteristic", "value", "goal", "pattern"
    value TEXT NOT NULL,
    confidence FLOAT,
    source TEXT,                 -- "extracted", "user_input", "inferred"
    learned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_facts_user (user_id),
    INDEX idx_facts_type (fact_type)
);

-- ========================================
-- MAINTENANCE & CLEANUP
-- ========================================

-- Cleanup old conversations (30-day retention)
-- Run manually or with a cron job:
-- DELETE FROM conversations WHERE archived_at IS NOT NULL AND archived_at < datetime('now', '-30 days');
-- DELETE FROM messages WHERE conversation_id NOT IN (SELECT id FROM conversations WHERE archived_at IS NULL);

-- View: Active conversations only
CREATE VIEW active_conversations AS
SELECT * FROM conversations WHERE archived_at IS NULL;

-- View: Recent messages (last 10 per conversation)
CREATE VIEW recent_messages AS
SELECT * FROM messages
WHERE conversation_id IN (SELECT id FROM active_conversations)
ORDER BY created_at DESC
LIMIT 10;

-- ========================================
-- END OF CLEAN SCHEMA
-- ========================================
-- Schema Version: 1.0
-- Includes: Phase 1 (Extraction Lock), Phase 2 (Conflict Channeling), Phase 3 (Response Validation)
-- Ready for: Export/import from old schema
