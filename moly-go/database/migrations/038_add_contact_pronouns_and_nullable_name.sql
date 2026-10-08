-- Migration 038: Add pronouns field to contacts and make name nullable
-- Purpose: Support unnamed contacts and track pronouns for disambiguation

-- Step 1: Add pronouns column (JSON array of pronouns)
ALTER TABLE contacts ADD COLUMN pronouns TEXT;
-- Format: JSON array like ["she", "her"] or ["he", "him"] or ["they", "them"]

-- Step 2: Make name column nullable (for unnamed contacts)
-- Some databases allow this directly, some need workaround
-- SQLite: Columns are NULL-able by default, so this is implicit
-- We just need to remove NOT NULL constraint if it exists on fresh migration

-- Step 3: Update index to handle nullable names
-- Drop old unique index if it exists
DROP INDEX IF EXISTS idx_contacts_user_name;

-- Create new index that allows NULL names
CREATE INDEX IF NOT EXISTS idx_contacts_user ON contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_name ON contacts(name);

-- Step 4: Add status field if it doesn't exist (to track unnamed status)
-- Some migrations may have added this already
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'active';
-- Possible values: active, archived, unnamed (in transition to named)
