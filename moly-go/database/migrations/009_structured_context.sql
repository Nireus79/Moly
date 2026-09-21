-- Migration 009: Add structured_context table
-- Date: September 21, 2026
-- Purpose: Track structured understanding of user situations (goals, values, people, blockers, etc.)

CREATE TABLE IF NOT EXISTS structured_context (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    situation TEXT,                    -- "workplace conflict", "romantic uncertainty", etc.
    topic TEXT,                        -- Main topic of conversation

    people_involved TEXT,              -- JSON array of {name, relationship, role}
    goals TEXT,                        -- JSON array of strings
    values TEXT,                       -- JSON array of strings
    constraints TEXT,                  -- JSON array of strings

    past_attempts TEXT,                -- JSON array of strings ("what's been tried")
    current_blocker TEXT,              -- Current obstacle ("I don't know what to do next")

    emotional_tone TEXT,               -- "anxious", "frustrated", "hopeful", etc.

    remaining_gaps TEXT,               -- JSON array of open questions
    explored_topics TEXT,              -- JSON array of topics already covered

    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    UNIQUE(user_id, conversation_id)
);

-- Index for lookups
CREATE INDEX IF NOT EXISTS idx_structured_context_user_conv
    ON structured_context(user_id, conversation_id);
CREATE INDEX IF NOT EXISTS idx_structured_context_updated_at
    ON structured_context(updated_at);
