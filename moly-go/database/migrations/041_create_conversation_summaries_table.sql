-- Migration 041: Create conversation_summaries table
-- Purpose: Central hub for accumulated conversation context across all messages
-- Date: October 8, 2026
-- Issue: Table was referenced in 042 but never created - CRITICAL FIX

-- Create the table that tracks accumulated insights for each conversation
-- This table is central to FIX #6 and the maturity calculation system
CREATE TABLE IF NOT EXISTS conversation_summaries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    arc TEXT,                          -- Narrative summary of conversation flow
    key_topics TEXT,                   -- JSON array of topics discussed
    user_patterns TEXT,                -- JSON array of communication patterns
    confirmed_choices TEXT,            -- JSON array of confirmed preferences
    open_questions TEXT,               -- JSON array of unresolved questions
    message_count INTEGER DEFAULT 0,
    messages_since_update INTEGER DEFAULT 0,
    summary_version INTEGER DEFAULT 0,
    confidence REAL DEFAULT 0.0,       -- 0-1: accuracy score
    last_updated INTEGER,              -- Unix timestamp
    created_at INTEGER,
    updated_at INTEGER,

    -- FIX #6: Accumulated insights (Oct 8, 2026)
    accumulated_entity_count INTEGER DEFAULT 0,
    accumulated_contact_count INTEGER DEFAULT 0,
    accumulated_values TEXT,           -- JSON array of values mentioned
    accumulated_characteristics TEXT,  -- JSON array of characteristics
    conflicts_resolved INTEGER DEFAULT 0,
    clarity_progression TEXT,          -- JSON array of maturity scores per message

    UNIQUE(user_id, conversation_id),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY(conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_conversation_summaries_user_conversation
  ON conversation_summaries(user_id, conversation_id);

CREATE INDEX IF NOT EXISTS idx_conversation_summaries_accumulated_entity_count
  ON conversation_summaries(user_id, accumulated_entity_count);

CREATE INDEX IF NOT EXISTS idx_conversation_summaries_clarity_progression
  ON conversation_summaries(user_id, conversation_id);

CREATE INDEX IF NOT EXISTS idx_conversation_summaries_updated
  ON conversation_summaries(user_id, last_updated);

-- Migration notes:
-- This table must exist BEFORE migration 042 runs (which alters it)
-- Previously missing and only referenced in ALTER, causing immediate failures
-- All columns needed by ConversationSummaryRepository.go are included
