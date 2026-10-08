-- FIX #73: Layer 10 Persistent Questioning Session State
-- SQLite version (not MySQL): Tracks user responses across messages
-- Enables progressive questioning: question 1, 2, 3, 4 on different messages
-- Each session tracks: question count, previous answers, acknowledgment of harm

CREATE TABLE IF NOT EXISTS persistence_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    -- Tracking state
    question_count INTEGER NOT NULL DEFAULT 0,
    has_acknowledged_harm INTEGER NOT NULL DEFAULT 0,

    -- Question history
    previous_answers TEXT,
    questions_asked TEXT,

    -- Timestamps
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_question_asked_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_persistence_user_conversation ON persistence_sessions(user_id, conversation_id);
CREATE INDEX IF NOT EXISTS idx_persistence_updated_at ON persistence_sessions(updated_at);

-- FIX #73: History table for auditing (why did we ask these questions?)
CREATE TABLE IF NOT EXISTS persistence_session_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    user_id TEXT NOT NULL,

    -- What happened
    action TEXT NOT NULL,
    question_text TEXT,
    answer_text TEXT,
    question_number INTEGER,

    -- Context
    violation_detected TEXT,
    confidence REAL,

    -- Timestamps
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (session_id) REFERENCES persistence_sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_history_session_id ON persistence_session_history(session_id);
CREATE INDEX IF NOT EXISTS idx_history_user_id ON persistence_session_history(user_id);
CREATE INDEX IF NOT EXISTS idx_history_action ON persistence_session_history(action);
