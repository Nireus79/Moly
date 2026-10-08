-- Migration 045: Drop orphaned/unused tables from schema
-- Date: October 8, 2026
-- Purpose: Clean up schema by removing tables that are defined but never used
-- Impact: Reduces schema complexity, improves maintainability

-- Tables being dropped (0 references in codebase):
-- - user_contacts: Never referenced, duplicate functionality exists in contacts table

-- Step 1: Fix reflections table foreign key (was pointing to unused user_contacts)
-- Change contact_id from foreign key reference to simple TEXT field (contact IDs stored as strings)
-- SQLite doesn't support dropping foreign keys easily, so we recreate the table

BEGIN TRANSACTION;

-- Backup existing data
CREATE TABLE reflections_backup AS SELECT * FROM reflections;

-- Drop the old table with bad foreign key
DROP TABLE IF EXISTS reflections;

-- Recreate without the problematic foreign key to user_contacts
CREATE TABLE IF NOT EXISTS reflections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT,
    contact_id TEXT,
    message_id TEXT,
    characteristics TEXT,
    interests TEXT,
    intentions TEXT,
    communication_preferences TEXT,
    user_quotes TEXT,
    user_edits TEXT,
    extracted_style TEXT,
    extracted_intention TEXT,
    status TEXT DEFAULT 'pending_approval',
    created_at INTEGER NOT NULL,
    approved_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (message_id) REFERENCES chat_messages(id) ON DELETE SET NULL
);

-- Restore data
INSERT INTO reflections SELECT * FROM reflections_backup;
DROP TABLE reflections_backup;

COMMIT;

-- Step 2: Drop orphaned table
DROP TABLE IF EXISTS user_contacts;

-- Note: The following tables are minimal-use and may be useful for future features:
-- - audit_log, behavior_patterns, clarification_answers, response_templates,
--   structured_context, suggestion_choices, user_interactions
-- They are kept for now as they have some references or may be used in future phases.
-- Full cleanup can be done in Phase 4 after confirming they're not needed.
