-- Phase 0 Extension: Add missing Conversation fields
-- Date: September 13, 2026
-- Purpose: Extend conversations table to include purpose, members, and settings
--          so conversation metadata syncs across sessions

-- Add purpose column
ALTER TABLE conversations ADD COLUMN purpose TEXT DEFAULT NULL;

-- Add members column (JSON array of ConversationMember objects)
ALTER TABLE conversations ADD COLUMN members TEXT DEFAULT NULL;

-- Add settings column (JSON object with mode, context, llmProvider)
ALTER TABLE conversations ADD COLUMN settings TEXT DEFAULT NULL;

-- Add notes column for additional metadata
ALTER TABLE conversations ADD COLUMN notes TEXT DEFAULT NULL;

-- Create indexes for consistent query performance
CREATE INDEX IF NOT EXISTS idx_conversations_type ON conversations(type);
CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at);
