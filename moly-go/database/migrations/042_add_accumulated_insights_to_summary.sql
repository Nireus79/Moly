-- Migration 042: Add accumulated insights tracking to conversation_summaries
-- Purpose: Enable maturity calculation to use all accumulated context, not just current message
-- Date: October 8, 2026
-- Impact: Fixes maturity recalculation bug where maturity appears flat across messages

-- Add columns to track accumulated insights across entire conversation
ALTER TABLE conversation_summaries ADD COLUMN accumulated_entity_count INTEGER DEFAULT 0;
ALTER TABLE conversation_summaries ADD COLUMN accumulated_contact_count INTEGER DEFAULT 0;
ALTER TABLE conversation_summaries ADD COLUMN accumulated_values TEXT; -- JSON array
ALTER TABLE conversation_summaries ADD COLUMN accumulated_characteristics TEXT; -- JSON array
ALTER TABLE conversation_summaries ADD COLUMN conflicts_resolved INTEGER DEFAULT 0;
ALTER TABLE conversation_summaries ADD COLUMN clarity_progression TEXT; -- JSON array [0.30, 0.45, 0.62]

-- Create indexes for efficient querying of accumulated data
CREATE INDEX IF NOT EXISTS idx_conversation_summaries_accumulated_entity_count
  ON conversation_summaries(user_id, conversation_id, accumulated_entity_count);

CREATE INDEX IF NOT EXISTS idx_conversation_summaries_clarity_progression
  ON conversation_summaries(user_id, conversation_id);

-- Migration notes:
-- accumulated_entity_count: Sum of all unique entities extracted across all messages
-- accumulated_contact_count: Count of unique contacts mentioned
-- accumulated_values: JSON array of all values mentioned (["authentic", "direct"])
-- accumulated_characteristics: JSON array of all characteristics (["smart", "thoughtful"])
-- conflicts_resolved: Count of contradictions that have been clarified
-- clarity_progression: JSON array of maturity scores per message [0.30, 0.45, 0.62, 0.80]
--   This shows the maturity journey and is used to detect if maturity is actually improving
