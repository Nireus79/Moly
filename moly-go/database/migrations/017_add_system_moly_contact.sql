-- Migration: Add reserved system contact for Moly self-awareness
-- Date: 2026-09-28
-- Purpose: Track user's relationship with Moly system itself

-- Ensure system_moly contact exists for each user
-- Run once per existing user, then auto-create on registration

-- This contact tracks the user's relationship with the system itself
-- Metadata: greeting frequency, relationship phase, average tone, etc.

-- For new users (created after this migration):
-- The AuthHandler will auto-create this contact during registration

-- For existing users:
-- This migration will create it. If it already exists, it's ignored.

INSERT OR IGNORE INTO contacts (
  id,
  user_id,
  name,
  relationship,
  characteristics,
  created_at,
  updated_at
)
SELECT
  'system_moly_' || users.id,
  users.id,
  'Moly',
  'system_coach',
  json('{"greeting_count": 0, "relationship_phase": "new", "avg_tone": "neutral", "last_greeted_at": null}'),
  strftime('%s', 'now'),
  strftime('%s', 'now')
FROM users
WHERE id NOT IN (
  SELECT user_id FROM contacts WHERE name = 'Moly' AND relationship = 'system_coach'
);

-- Index for fast lookup of system_moly contacts
CREATE INDEX IF NOT EXISTS idx_system_moly_contacts ON contacts(user_id, name)
WHERE relationship = 'system_coach' AND name = 'Moly';
