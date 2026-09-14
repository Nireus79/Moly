-- Migration 008: Add name column to users table
-- Date: September 14, 2026
-- Purpose: Support proper Name/Email/Password authentication

ALTER TABLE users ADD COLUMN name TEXT NOT NULL DEFAULT 'User';

-- Update existing users with a default name based on email
UPDATE users SET name = SUBSTR(email, 1, INSTR(email, '@') - 1) WHERE name = 'User';

-- Make name unique per user (not globally)
CREATE INDEX IF NOT EXISTS idx_users_email_name ON users(email, name);
