-- Migration 007: Convert clarifications to proper relational schema
-- Date: September 14, 2026
-- Purpose: Enable cross-session clarification resumption

-- Main table: Track facts awaiting clarification
CREATE TABLE IF NOT EXISTS pending_clarifications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT,
    fact_id TEXT NOT NULL,
    fact_type TEXT NOT NULL,
    fact_value TEXT NOT NULL,
    attributed_to TEXT NOT NULL,
    evidence TEXT,
    confidence REAL DEFAULT 0.8,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'partially_answered', 'complete', 'abandoned')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,

    UNIQUE(fact_id),
    FOREIGN KEY(user_id) REFERENCES users(id),
    FOREIGN KEY(conversation_id) REFERENCES conversations(id)
);

-- Questions linked to facts
CREATE TABLE IF NOT EXISTS clarification_questions (
    id TEXT PRIMARY KEY,
    pending_clarification_id TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    question_text TEXT NOT NULL,
    question_type TEXT NOT NULL,  -- "subject", "contact_context", "ambiguous", etc
    context TEXT,
    created_at INTEGER NOT NULL,

    FOREIGN KEY(pending_clarification_id) REFERENCES pending_clarifications(id) ON DELETE CASCADE
);

-- User responses
CREATE TABLE IF NOT EXISTS clarification_answers (
    id TEXT PRIMARY KEY,
    clarification_question_id TEXT NOT NULL UNIQUE,
    user_answer TEXT NOT NULL,
    answered_at INTEGER NOT NULL,

    FOREIGN KEY(clarification_question_id) REFERENCES clarification_questions(id) ON DELETE CASCADE
);

-- Indexes for fast queries
CREATE INDEX IF NOT EXISTS idx_pending_user ON pending_clarifications(user_id);
CREATE INDEX IF NOT EXISTS idx_pending_status ON pending_clarifications(status);
CREATE INDEX IF NOT EXISTS idx_pending_fact ON pending_clarifications(fact_id);
CREATE INDEX IF NOT EXISTS idx_questions_pending ON clarification_questions(pending_clarification_id);
CREATE INDEX IF NOT EXISTS idx_questions_type ON clarification_questions(question_type);
CREATE INDEX IF NOT EXISTS idx_answers_question ON clarification_answers(clarification_question_id);
