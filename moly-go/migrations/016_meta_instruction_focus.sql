-- Migration: Add conversation focus tracking for meta-instructions
-- Tracks when user directs Moly to focus on specific person/topic

ALTER TABLE structured_context ADD COLUMN conversation_focus TEXT;
ALTER TABLE structured_context ADD COLUMN focused_person TEXT;

-- Index for focus queries
CREATE INDEX IF NOT EXISTS idx_structured_context_focus ON structured_context(conversation_focus);
