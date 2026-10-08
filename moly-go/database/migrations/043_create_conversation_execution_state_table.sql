-- Migration 043: Create conversation_execution_state table
-- Purpose: Track execution phase and progress of conversation workflow
-- Date: October 8, 2026
-- Issue: Table was referenced in code but never created - CRITICAL FIX

-- Create the table that tracks where we are in the conversation workflow
-- This is used by ExecutionStateManager to coordinate multi-phase processing
CREATE TABLE IF NOT EXISTS conversation_execution_state (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    phase TEXT NOT NULL,                    -- "initial", "gathering_context", "processing", "complete"
    covered_categories TEXT,                -- JSON object tracking what we've asked about
    current_message_seq INTEGER DEFAULT 0,  -- Which message in conversation
    started_at INTEGER NOT NULL,            -- Unix timestamp when conversation started
    updated_at INTEGER NOT NULL,            -- Unix timestamp of last update
    version INTEGER DEFAULT 1,              -- For optimistic locking on concurrent updates

    UNIQUE(user_id, conversation_id),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_conversation_execution_state_user_conversation
  ON conversation_execution_state(user_id, conversation_id);

CREATE INDEX IF NOT EXISTS idx_conversation_execution_state_phase
  ON conversation_execution_state(user_id, phase);

CREATE INDEX IF NOT EXISTS idx_conversation_execution_state_updated
  ON conversation_execution_state(user_id, updated_at);

-- Migration notes:
-- phase: Tracks workflow state (initial → gathering_context → processing → complete)
-- covered_categories: JSON object {category: true/false} for tracking what was covered
-- current_message_seq: Message sequence number (0-based) for progress tracking
-- version: Used for optimistic locking to detect concurrent updates
-- Used by ExecutionStateManager in agents/execution_state.go
