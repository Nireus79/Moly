-- Migration: Add message_processing_state table
-- Date: 2026-09-29
-- Purpose: Track pipeline stage completion for each message

CREATE TABLE IF NOT EXISTS message_processing_state (
  user_id TEXT NOT NULL,
  conversation_id TEXT NOT NULL,
  message_id TEXT NOT NULL,
  completed_stages JSONB,
  stage_results JSONB,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  version INTEGER DEFAULT 1,
  PRIMARY KEY (user_id, conversation_id, message_id),
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

-- Index for efficient lookups
CREATE INDEX IF NOT EXISTS idx_message_processing_state_lookup ON message_processing_state(user_id, conversation_id);
