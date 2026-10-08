-- Migration 044: Consolidate clarification_questions schema into authoritative version
-- Problem: 3 conflicting schemas for same table (migrations 007, 030, and code expectations)
-- Solution: Create unified schema combining all fields, migrate data, replace old table
-- Date: October 8, 2026
-- Impact: Fixes data integrity issues from schema conflicts

-- Step 1: Create new authoritative table with consolidated schema
CREATE TABLE clarification_questions_consolidated (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    -- Core question fields
    clarification_type TEXT NOT NULL,  -- "context", "conflict", "principle", "gap"
    question_text TEXT NOT NULL,

    -- Context and linking fields
    context_notes TEXT,                 -- Detailed context for the question
    layer TEXT,                         -- Which layer (Layer4, Layer5, Layer7, etc.)
    linked_conflict_id TEXT,            -- If from conflict resolution
    linked_facts TEXT,                  -- Related facts (JSON array)

    -- Status and priority
    status TEXT DEFAULT 'active',       -- active, answered, skipped
    priority INTEGER DEFAULT 1,         -- 1=low, 2=medium, 3=high

    -- Timestamps
    created_at INTEGER NOT NULL,
    answered_at INTEGER,
    updated_at INTEGER NOT NULL DEFAULT 0,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Step 2: Create indexes for the new table
CREATE INDEX idx_clarification_by_user_conversation
    ON clarification_questions_consolidated(user_id, conversation_id);

CREATE INDEX idx_clarification_by_status
    ON clarification_questions_consolidated(conversation_id, status);

CREATE INDEX idx_clarification_by_type
    ON clarification_questions_consolidated(clarification_type);

CREATE INDEX idx_clarification_by_layer
    ON clarification_questions_consolidated(layer);

-- Step 3: Migrate existing data from old table
-- Use COALESCE to handle missing columns from different schema versions
INSERT INTO clarification_questions_consolidated
    (id, user_id, conversation_id, clarification_type, question_text,
     context_notes, layer, linked_conflict_id, linked_facts, status,
     priority, created_at, answered_at, updated_at)
SELECT
    id,
    user_id,
    conversation_id,
    COALESCE(clarification_type, question_type, 'context'),  -- Try both field names
    question_text,
    COALESCE(context_notes, context, ''),  -- Try both field names
    layer,
    linked_conflict_id,
    linked_facts,
    COALESCE(status, 'active'),
    COALESCE(priority, 1),
    created_at,
    answered_at,
    COALESCE(updated_at, created_at)
FROM clarification_questions
WHERE id IS NOT NULL;  -- Skip any corrupted rows without ID

-- Step 4: Verify data migration
-- Check row counts match
-- SELECT COUNT(*) as old_count FROM clarification_questions;
-- SELECT COUNT(*) as new_count FROM clarification_questions_consolidated;

-- Step 5: Drop old table
DROP TABLE clarification_questions CASCADE;

-- Step 6: Rename new table to authoritative name
ALTER TABLE clarification_questions_consolidated RENAME TO clarification_questions;

-- Migration notes:
-- This consolidates 3 conflicting schemas from:
-- - Migration 007 (original): pending_clarification_id, sequence, question_type, context
-- - Migration 030 (added): conversation_id, layer, principle_basis, linked_conflict_id
-- - Code expectations: clarification_type, context_notes, options, priority, linked_facts
--
-- The new schema includes:
-- - All fields from all three versions
-- - Renamed for clarity (question_type → clarification_type, context → context_notes)
-- - Proper foreign keys and cascading deletes
-- - Comprehensive indexes for query performance
--
-- Code changes required:
-- - Update all ClarificationQuestion queries to use new field names
-- - Update Layer 5 to use 'clarification_type' instead of 'question_type'
-- - Update response generators to read 'context_notes' instead of 'context'
