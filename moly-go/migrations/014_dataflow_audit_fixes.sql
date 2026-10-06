-- Migration: Fix dataflow gaps identified in C-30k audit
-- Adds persistence for: contacts, intentions, interests, needs, clarifications, context quality, verdict details, response linking

-- 1. Add missing fields to about_me (Gap 3: interests, needs; Gap 2: latest_intention)
ALTER TABLE about_me ADD COLUMN interests TEXT; -- JSON array of user interests
ALTER TABLE about_me ADD COLUMN needs TEXT; -- JSON array of immediate needs
ALTER TABLE about_me ADD COLUMN latest_intention TEXT; -- Latest extracted user intention
ALTER TABLE about_me ADD COLUMN latest_intention_at INTEGER; -- When intention was last updated

-- 2. Enhance contacts table for better tracking (Gap 1: persist extracted contacts)
ALTER TABLE contacts ADD COLUMN confidence REAL DEFAULT 0.5; -- Confidence score of contact extraction
ALTER TABLE contacts ADD COLUMN extraction_count INTEGER DEFAULT 1; -- How many times mentioned
ALTER TABLE contacts ADD COLUMN last_mentioned_at INTEGER; -- Most recent mention

-- 3. Add fields to interactions table (Gap 5, 6, 7)
ALTER TABLE interactions ADD COLUMN context_quality REAL; -- % of context fields loaded (0.0-1.0)
ALTER TABLE interactions ADD COLUMN clarification_requested BOOLEAN DEFAULT 0; -- Whether clarification was asked
ALTER TABLE interactions ADD COLUMN response_id TEXT; -- Link to assistant response in chat_messages
ALTER TABLE interactions ADD COLUMN verdict_details TEXT; -- JSON: detailed safety verdict if any
ALTER TABLE interactions ADD COLUMN phase_results TEXT; -- JSON: which security layers triggered

-- 4. Enhance safety_incidents table (Gap 6: store verdict details)
ALTER TABLE safety_incidents ADD COLUMN verdict_reasoning TEXT; -- Detailed reasoning from evaluator
ALTER TABLE safety_incidents ADD COLUMN violated_principles TEXT; -- JSON: ["principle1", "principle2"]
ALTER TABLE safety_incidents ADD COLUMN evidence_snippets TEXT; -- JSON: ["snippet1", "snippet2"]
ALTER TABLE safety_incidents ADD COLUMN response_text TEXT; -- FIXED: actual response sent to user (was storing title)
ALTER TABLE safety_incidents ADD COLUMN context_maturity REAL; -- Context maturity at time of incident

-- 5. New table: user_intentions (Gap 2: carry intention forward)
-- Tracks evolution of user goals across conversations
CREATE TABLE IF NOT EXISTS user_intentions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT,
    intention TEXT NOT NULL, -- The stated goal/intention
    confidence REAL, -- LLM confidence in extraction
    extracted_from TEXT, -- Message ID that revealed this intention
    confirmed BOOLEAN DEFAULT 0, -- Did user explicitly confirm?
    confirmed_at INTEGER, -- When user confirmed
    status TEXT DEFAULT 'active', -- "active", "achieved", "abandoned", "clarified"
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_user_intentions_user_id ON user_intentions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_intentions_status ON user_intentions(status);

-- 6. New table: extracted_clarifications (Gap 4: save clarification questions)
-- Tracks clarification questions asked by system
CREATE TABLE IF NOT EXISTS extracted_clarifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    interaction_id INTEGER, -- Link to interactions table
    gap_type TEXT NOT NULL, -- "communicationStyle", "coreValues", etc.
    question TEXT NOT NULL, -- The clarification question asked
    priority INTEGER DEFAULT 3, -- 1=critical, 2=important, 3=helpful
    asked_at INTEGER NOT NULL,
    response_received BOOLEAN DEFAULT 0,
    user_response TEXT, -- User's answer to the question
    responded_at INTEGER, -- When user answered
    was_helpful BOOLEAN, -- User feedback on usefulness
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (interaction_id) REFERENCES interactions(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_clarifications_conversation ON extracted_clarifications(conversation_id);
CREATE INDEX IF NOT EXISTS idx_clarifications_gap_type ON extracted_clarifications(gap_type);
CREATE INDEX IF NOT EXISTS idx_clarifications_responded ON extracted_clarifications(response_received);

-- 7. New table: verdict_principles (Gap 6: track principle violations)
-- Normalized table for principle violations (denormalized in safety_incidents via JSON)
CREATE TABLE IF NOT EXISTS verdict_principles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    safety_incident_id INTEGER,
    principle_id TEXT NOT NULL, -- e.g., "consent", "autonomy", "safety"
    principle_name TEXT,
    severity TEXT, -- "low", "medium", "high", "critical"
    confidence REAL, -- LLM confidence in violation
    created_at INTEGER NOT NULL,
    FOREIGN KEY (safety_incident_id) REFERENCES safety_incidents(id) ON DELETE CASCADE
);

-- 8. Indexes for performance
CREATE INDEX IF NOT EXISTS idx_interactions_context_quality ON interactions(context_quality);
CREATE INDEX IF NOT EXISTS idx_interactions_response_id ON interactions(response_id);
CREATE INDEX IF NOT EXISTS idx_interactions_clarification ON interactions(clarification_requested);
CREATE INDEX IF NOT EXISTS idx_safety_incidents_verdict ON safety_incidents(verdict_reasoning);
CREATE INDEX IF NOT EXISTS idx_contacts_user_mentioned ON contacts(user_id, last_mentioned_at);
CREATE INDEX IF NOT EXISTS idx_about_me_latest_intention ON about_me(latest_intention);
