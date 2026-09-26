-- Migration 010: Fix schema column name mismatches
-- Date: September 26, 2026
-- Purpose: Add aliases for columns that the codebase uses vs. what the schema defines
-- The code uses: core_values, tone_preference, user_contacts table
-- The schema defines: values, preferred_tone, contacts table

-- Add core_values column to about_me (alias for values)
ALTER TABLE about_me ADD COLUMN core_values JSONB DEFAULT '[]';

-- Add tone_preference column to about_me (alias for preferred_tone)
ALTER TABLE about_me ADD COLUMN tone_preference TEXT DEFAULT '';

-- Create user_contacts view as alias for contacts table for backward compatibility
CREATE VIEW IF NOT EXISTS user_contacts AS
SELECT id, user_id, name, relationship, characteristics, notes, created_at, updated_at
FROM contacts;

-- Create triggers to keep core_values and values in sync
CREATE TRIGGER IF NOT EXISTS about_me_sync_values_on_insert
AFTER INSERT ON about_me
BEGIN
  UPDATE about_me SET core_values = values WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS about_me_sync_values_on_update
AFTER UPDATE ON about_me
BEGIN
  UPDATE about_me SET core_values = values WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS about_me_sync_tone_on_insert
AFTER INSERT ON about_me
BEGIN
  UPDATE about_me SET tone_preference = preferred_tone WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS about_me_sync_tone_on_update
AFTER UPDATE ON about_me
BEGIN
  UPDATE about_me SET tone_preference = preferred_tone WHERE id = NEW.id;
END;
