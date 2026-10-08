-- Create system_context table for Moly's self-awareness
-- Tracks user feedback, directives, and preferences about the system
-- Date: October 8, 2026
-- Purpose: Persist system-directed feedback across messages (parallel to AboutMe for user feedback)

CREATE TABLE IF NOT EXISTS system_context (
    user_id TEXT PRIMARY KEY,

    -- User's feedback about Moly's performance and behavior
    user_feedback TEXT DEFAULT NULL,  -- JSON array: ["too verbose", "helpful"]

    -- User's directives for how system should behave
    user_directives TEXT DEFAULT NULL,  -- JSON array: ["be concise", "ask more"]

    -- User's perception of system capabilities/limitations
    system_perceptions TEXT DEFAULT NULL,  -- JSON array: ["can do legal?", "lacks empathy"]

    -- User's preferred interaction style
    preferred_interaction_style TEXT DEFAULT NULL,  -- "direct", "socratic", "collaborative"

    -- Implicit ratings from conversation
    helpfulness_rating REAL DEFAULT 0.0,  -- 0-1
    clarity_rating REAL DEFAULT 0.0,      -- 0-1

    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    version INTEGER DEFAULT 1,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Index for quick lookup by user
CREATE INDEX IF NOT EXISTS idx_system_context_user_id ON system_context(user_id);
