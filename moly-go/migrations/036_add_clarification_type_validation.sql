-- FIX #36: Add CHECK constraint for valid clarification_type values

CREATE TABLE IF NOT EXISTS clarification_questions_new (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    clarification_type TEXT NOT NULL CHECK (clarification_type IN (
        'gap_clarification', 'context_gap', 'style_conflict', 'contact_conflict',
        'intention_conflict', 'relationship_conflict', 'characteristic_conflict',
        'safety_evaluation', 'ambiguity_check', 'clarity_check'
    )),
    question_text TEXT NOT NULL,
    context_notes TEXT,
    options TEXT,
    priority INTEGER DEFAULT 2,
    status TEXT DEFAULT 'pending',
    linked_facts TEXT,
    created_at INTEGER NOT NULL,
    answered_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

INSERT INTO clarification_questions_new
SELECT * FROM clarification_questions
WHERE clarification_type IN ('gap_clarification', 'context_gap', 'style_conflict', 'contact_conflict', 'intention_conflict', 'relationship_conflict', 'characteristic_conflict', 'safety_evaluation', 'ambiguity_check', 'clarity_check');

DROP TABLE clarification_questions;
ALTER TABLE clarification_questions_new RENAME TO clarification_questions;

CREATE INDEX IF NOT EXISTS idx_clarification_questions_user_id ON clarification_questions(user_id);
CREATE INDEX IF NOT EXISTS idx_clarification_questions_conversation_id ON clarification_questions(conversation_id);
CREATE INDEX IF NOT EXISTS idx_clarification_questions_status ON clarification_questions(status);
CREATE INDEX IF NOT EXISTS idx_clarification_questions_type ON clarification_questions(clarification_type);
