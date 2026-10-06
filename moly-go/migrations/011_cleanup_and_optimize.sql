-- Migration 011: Database Cleanup & Optimization
-- Date: September 26, 2026
-- Purpose: Remove orphaned data, add performance indexes, verify constraints
-- Impact: +Performance, +Data Integrity, -0 records with valid data

-- STEP 1: Remove expired login codes (older than 30 days)
-- These are one-time codes that have expired and are no longer needed
DELETE FROM login_codes
WHERE expires_at < strftime('%s', 'now', '-30 days');

-- STEP 2: Remove orphaned reflections (contact deleted but reflection remains)
-- Ensure referential integrity: reflections should only exist for contacts that exist
DELETE FROM reflections
WHERE contact_id NOT NULL
AND contact_id NOT IN (SELECT id FROM contacts);

-- STEP 3: Remove orphaned reflections (message deleted but reflection remains)
-- Ensure referential integrity: reflections should only exist for messages that exist
DELETE FROM reflections
WHERE message_id NOT NULL
AND message_id NOT IN (SELECT id FROM chat_messages);

-- STEP 4: Remove behavior patterns for deleted users
-- Orphaned patterns have no user to analyze
DELETE FROM behavior_patterns
WHERE user_id NOT IN (SELECT id FROM users);

-- STEP 5: Remove user interactions for deleted users
-- Orphaned interactions have no owner
DELETE FROM user_interactions
WHERE user_id NOT IN (SELECT id FROM users);

-- STEP 6: Add performance indexes for common query patterns

-- Index for time-based queries on safety incidents
CREATE INDEX IF NOT EXISTS idx_safety_incidents_detected_at
ON safety_incidents(detected_at DESC);

-- Index for conversation history queries (user's messages over time)
CREATE INDEX IF NOT EXISTS idx_interactions_timestamp
ON interactions(user_id, timestamp DESC);

-- Index for login expiry checks
CREATE INDEX IF NOT EXISTS idx_login_codes_expires
ON login_codes(expires_at);

-- Index for conversation message lookups
CREATE INDEX IF NOT EXISTS idx_interactions_conversation
ON interactions(conversation_id);

-- STEP 7: Verify constraint violations (for debugging, disabled by default)
-- Run these manually if integrity issues are suspected:
-- SELECT COUNT(*) as orphaned_login_codes
-- FROM login_codes
-- WHERE user_id NOT IN (SELECT id FROM users);
--
-- SELECT COUNT(*) as users_with_multiple_about_me
-- FROM about_me
-- GROUP BY user_id
-- HAVING COUNT(*) > 1;
--
-- SELECT COUNT(*) as duplicate_active_contacts
-- FROM contacts
-- WHERE status = 'active'
-- GROUP BY user_id, name
-- HAVING COUNT(*) > 1;
