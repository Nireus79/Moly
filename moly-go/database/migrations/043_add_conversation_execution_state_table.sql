-- Migration 043: Create conversation_execution_state table
-- Purpose: Track execution phase and state for each conversation
-- Date: October 8, 2026
-- Status: CRITICAL FIX - table was referenced in code but never created

-- Creates table for tracking conversation workflow state
-- Used by ExecutionStateManager in agents/execution_state.go
CREATE TABLE IF NOT EXISTS conversation_execution_state (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    phase TEXT NOT NULL DEFAULT 'initial', -- 'initial', 'gathering_context', 'processing', 'complete'
    covered_categories TEXT,               -- JSON or comma-separated list of covered categories
    current_message_seq INTEGER DEFAULT 0, -- Which message number we're on
    started_at INTEGER NOT NULL,           -- Unix timestamp when conversation started
    updated_at INTEGER NOT NULL,           -- Unix timestamp of last update
    version INTEGER DEFAULT 1,             -- Optimistic locking version

    UNIQUE(user_id, conversation_id),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Index for efficient lookups by user + conversation
CREATE INDEX IF NOT EXISTS idx_conversation_execution_state_user_conv
    ON conversation_execution_state(user_id, conversation_id);

-- Index for finding active conversations by user
CREATE INDEX IF NOT EXISTS idx_conversation_execution_state_user_phase
    ON conversation_execution_state(user_id, phase);

-- Migration notes:
-- This table was missing but referenced in agents/execution_state.go
-- ExecutionStateManager.GetOrCreateState() expects this table to exist
-- Main.go also references this table (line 2548)
-- Table tracks: which phase of conversation workflow, which categories answered, version tracking
