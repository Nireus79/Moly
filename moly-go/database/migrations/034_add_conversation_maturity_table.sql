-- Add conversation_maturity table for Phase 6 maturity persistence
CREATE TABLE IF NOT EXISTS conversation_maturity (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    current_phase TEXT NOT NULL DEFAULT 'initial',
    maturity REAL NOT NULL DEFAULT 0.0,
    accomplishments_completed INTEGER NOT NULL DEFAULT 0,
    accomplishments_total INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    UNIQUE(user_id, conversation_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_maturity_user ON conversation_maturity(user_id);
CREATE INDEX IF NOT EXISTS idx_conversation_maturity_conversation ON conversation_maturity(conversation_id);
