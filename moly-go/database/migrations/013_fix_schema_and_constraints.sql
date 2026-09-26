-- Migration 013: Fix Schema Issues and Foreign Key Constraints
-- Date: September 26, 2026
-- Purpose: Fix foreign key references and schema mismatches
-- Issues Fixed:
--   1. clarification_questions foreign key referencing wrong table
--   2. context_attributes foreign key referencing wrong table
--   3. Support for confirmed preferences loading

-- STEP 1: Fix clarification_questions table foreign key
-- Old: FOREIGN KEY (conversation_id) REFERENCES interactions(conversation_id)
-- New: FOREIGN KEY (conversation_id) REFERENCES conversations(id)

-- Recreate clarification_questions with correct foreign keys
CREATE TABLE IF NOT EXISTS clarification_questions_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    clarification_type TEXT NOT NULL,
    question_text TEXT NOT NULL,
    context_notes TEXT,
    options TEXT,
    priority INTEGER DEFAULT 2,
    status TEXT DEFAULT 'pending',
    linked_facts TEXT,
    created_at INTEGER NOT NULL,
    answered_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Copy data from old table
INSERT INTO clarification_questions_new
SELECT * FROM clarification_questions;

-- Drop old table and rename new one
DROP TABLE IF EXISTS clarification_questions;
ALTER TABLE clarification_questions_new RENAME TO clarification_questions;

-- Recreate indexes for clarification_questions
CREATE INDEX IF NOT EXISTS idx_clarification_questions_user_id ON clarification_questions(user_id);
CREATE INDEX IF NOT EXISTS idx_clarification_questions_conversation_id ON clarification_questions(conversation_id);
CREATE INDEX IF NOT EXISTS idx_clarification_questions_status ON clarification_questions(status);
CREATE INDEX IF NOT EXISTS idx_clarification_questions_type ON clarification_questions(clarification_type);

-- STEP 2: Fix context_attributes table foreign key
-- Old: FOREIGN KEY (conversation_id) REFERENCES interactions(conversation_id)
-- New: FOREIGN KEY (conversation_id) REFERENCES conversations(id)

CREATE TABLE IF NOT EXISTS context_attributes_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    fact_type TEXT NOT NULL,
    fact_value TEXT NOT NULL,
    attributed_to TEXT NOT NULL,
    context TEXT DEFAULT 'general',
    confidence REAL DEFAULT 0.8,
    source TEXT DEFAULT 'explicit',
    evidence TEXT,
    version INTEGER DEFAULT 1,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Copy data from old table
INSERT INTO context_attributes_new
SELECT * FROM context_attributes;

-- Drop old table and rename new one
DROP TABLE IF EXISTS context_attributes;
ALTER TABLE context_attributes_new RENAME TO context_attributes;

-- Recreate indexes for context_attributes
CREATE INDEX IF NOT EXISTS idx_context_attributes_user_conversation ON context_attributes(user_id, conversation_id);
CREATE INDEX IF NOT EXISTS idx_context_attributes_source ON context_attributes(source);

-- STEP 3: Verify all constraints are now correct
-- Run these manually to verify:
-- SELECT COUNT(*) as orphaned_clarifications
-- FROM clarification_questions
-- WHERE conversation_id NOT IN (SELECT id FROM conversations);
--
-- SELECT COUNT(*) as orphaned_context_attributes
-- FROM context_attributes
-- WHERE conversation_id NOT IN (SELECT id FROM conversations);
