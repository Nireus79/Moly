-- Migration: Add conversation focus tracking for meta-instructions
-- Tracks when user directs Moly to focus on specific person/topic

ALTER TABLE structured_contexts ADD COLUMN conversation_focus TEXT;
ALTER TABLE structured_contexts ADD COLUMN focused_person TEXT;

-- Index for focus queries
CREATE INDEX IF NOT EXISTS idx_structured_contexts_focus ON structured_contexts(conversation_focus);
