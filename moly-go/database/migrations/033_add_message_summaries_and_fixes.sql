-- Migration 033: Add message_summaries table and fix missing schema columns
-- Phase: Session 30 - Production bug fixes

-- message_summaries: Lightweight per-message summary for Phase 3 cache optimization
CREATE TABLE IF NOT EXISTS message_summaries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    message_id TEXT NOT NULL UNIQUE,
    entities_count INTEGER,
    confidence REAL,
    extraction_source TEXT, -- 'llm', 'heuristic', 'cache'
    summary_text TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Create indexes for message_summaries
CREATE INDEX IF NOT EXISTS idx_message_summaries_user_id ON message_summaries(user_id);
CREATE INDEX IF NOT EXISTS idx_message_summaries_conversation_id ON message_summaries(conversation_id);
CREATE INDEX IF NOT EXISTS idx_message_summaries_message_id ON message_summaries(message_id);
CREATE INDEX IF NOT EXISTS idx_message_summaries_created_at ON message_summaries(created_at);

-- Fix: Ensure contacts table has relationship_type column (from migration 030)
-- If it doesn't exist, add it
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS relationship_type TEXT DEFAULT 'other';

-- conversation_maturity alias for maturity_states (for backward compatibility)
-- The actual table is maturity_states from migration 015, but code might reference conversation_maturity
-- This is just documentation - the maturity_states table already exists
