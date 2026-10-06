-- Phase 0 Extension: Add missing AboutMe fields
-- Date: September 13, 2026
-- Purpose: Extend about_me table to include goals and patterns
--          so user data syncs across devices

-- Add goals and patterns columns to about_me table
ALTER TABLE about_me ADD COLUMN goals TEXT DEFAULT NULL;
ALTER TABLE about_me ADD COLUMN patterns TEXT DEFAULT NULL;

-- Create indexes for consistency
CREATE INDEX IF NOT EXISTS idx_about_me_user_id ON about_me(user_id);
