-- FIX #35: Add missing message_index column to message_summaries
-- This column is required by message_summary_repository.go

ALTER TABLE message_summaries ADD COLUMN message_index INTEGER DEFAULT 0;

-- Create index for efficient queries
CREATE INDEX IF NOT EXISTS idx_message_summaries_message_index ON message_summaries(message_index);
