-- Migration: Add context_conflicts table for Phase 1 extraction conflict tracking
-- Purpose: Persist conflicts detected between extracted data and database during Phase 0-1
-- Used by: ConflictDetector (agents/conflict_detector.go), ContextConflictRepository
-- Session 15: Phase 6 - Database schema updates for centralized extraction pipeline

-- Table: context_conflicts
-- Tracks conflicts when extracted context differs from saved values
-- Enables user-driven conflict resolution and audit trail
CREATE TABLE IF NOT EXISTS context_conflicts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    -- Conflict identification
    conflict_type TEXT NOT NULL, -- "subject_mismatch", "value_contradiction", "new_entity"
    severity TEXT NOT NULL,      -- "high", "medium", "low"

    -- Values involved
    saved_value TEXT,            -- JSON representation of database value
    extracted_value TEXT,        -- JSON representation of extracted value
    description TEXT,            -- Human-readable conflict description

    -- Resolution
    status TEXT DEFAULT 'unresolved', -- "unresolved", "resolved"
    resolution TEXT,                  -- "keep_saved", "use_extracted", "merge", "ask_user"
    resolution_details TEXT,          -- JSON with resolution details

    -- Timestamps
    created_at INTEGER NOT NULL,
    resolved_at INTEGER,

    -- Foreign keys
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Index for fast conflict lookup by conversation
CREATE INDEX IF NOT EXISTS idx_context_conflicts_conversation
    ON context_conflicts(user_id, conversation_id, status);

-- Index for finding unresolved conflicts
CREATE INDEX IF NOT EXISTS idx_context_conflicts_unresolved
    ON context_conflicts(user_id, status)
    WHERE status = 'unresolved';
