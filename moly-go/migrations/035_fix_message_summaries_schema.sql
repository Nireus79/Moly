-- Migration 035: Complete message_summaries schema (Session 34, FIX #1)
-- Issue: Code writes 19 fields but table only had 9 columns, causing "no such column" errors
-- All 13 missing columns are actively used in code - verified through complete codebase audit

-- Add all 13 missing columns that code requires
ALTER TABLE message_summaries ADD COLUMN message_index INTEGER DEFAULT 0;
ALTER TABLE message_summaries ADD COLUMN role TEXT DEFAULT 'user';
ALTER TABLE message_summaries ADD COLUMN message_length INTEGER DEFAULT 0;
ALTER TABLE message_summaries ADD COLUMN extracted_entities TEXT; -- JSON array
ALTER TABLE message_summaries ADD COLUMN entity_types TEXT; -- JSON array
ALTER TABLE message_summaries ADD COLUMN intention TEXT DEFAULT '';
ALTER TABLE message_summaries ADD COLUMN key_phrases TEXT; -- JSON array
ALTER TABLE message_summaries ADD COLUMN tone TEXT DEFAULT '';
ALTER TABLE message_summaries ADD COLUMN communication_style TEXT DEFAULT '';
ALTER TABLE message_summaries ADD COLUMN topic_shift INTEGER DEFAULT 0;
ALTER TABLE message_summaries ADD COLUMN has_clarification INTEGER DEFAULT 0;
ALTER TABLE message_summaries ADD COLUMN processed_at INTEGER DEFAULT 0;
ALTER TABLE message_summaries ADD COLUMN updated_at INTEGER DEFAULT 0;

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_message_summaries_message_index ON message_summaries(conversation_id, message_index);
CREATE INDEX IF NOT EXISTS idx_message_summaries_processed_at ON message_summaries(conversation_id, processed_at);
