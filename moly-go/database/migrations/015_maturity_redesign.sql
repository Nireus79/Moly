-- Migration: Redesign maturity system to match PoC (phase-based, event-driven, confidence-weighted)
-- Adds: maturity state tracking, event history, phase progression

-- maturity_states: Current maturity state for each conversation
CREATE TABLE IF NOT EXISTS maturity_states (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    phase TEXT NOT NULL, -- "discovery", "analysis", "design", "implementation"
    overall_score REAL NOT NULL DEFAULT 0.0, -- 0.0-1.0
    category_scores TEXT, -- JSON: {"category_name": {current_score, target_score, confidence, spec_count}}
    missing_categories TEXT, -- JSON: ["category1", "category2"]
    strongest_categories TEXT, -- JSON: ["category1", "category2"]
    weakest_categories TEXT, -- JSON: ["category1", "category2"]
    is_ready_to_advance BOOLEAN DEFAULT 0, -- Can advance to next phase?
    warnings TEXT, -- JSON: ["warning1", "warning2"]
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE(user_id, conversation_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- maturity_events: Historical record of maturity changes (event-driven re-evaluation)
CREATE TABLE IF NOT EXISTS maturity_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    phase TEXT NOT NULL, -- Phase when event occurred
    score_before REAL NOT NULL, -- Previous maturity score
    score_after REAL NOT NULL, -- New maturity score after event
    delta REAL NOT NULL, -- score_after - score_before
    event_type TEXT NOT NULL, -- "clarification_answered", "context_extracted", "phase_advanced", "confidence_updated"
    details TEXT, -- JSON: event-specific details (category_name, confidence, etc.)
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_maturity_states_user_id ON maturity_states(user_id);
CREATE INDEX IF NOT EXISTS idx_maturity_states_conversation ON maturity_states(conversation_id);
CREATE INDEX IF NOT EXISTS idx_maturity_states_phase ON maturity_states(phase);
CREATE INDEX IF NOT EXISTS idx_maturity_events_user_conv ON maturity_events(user_id, conversation_id);
CREATE INDEX IF NOT EXISTS idx_maturity_events_event_type ON maturity_events(event_type);
CREATE INDEX IF NOT EXISTS idx_maturity_events_created_at ON maturity_events(created_at);

-- Add maturity tracking columns to interactions table (from C-30l fixes)
-- These were added in migration 014, but we extend them here for new system
ALTER TABLE interactions ADD COLUMN maturity_score_at_message REAL; -- Maturity when message was received
ALTER TABLE interactions ADD COLUMN confidence_score_at_message REAL; -- Confidence when message was received
ALTER TABLE interactions ADD COLUMN maturity_phase_at_message TEXT; -- What phase we were in
