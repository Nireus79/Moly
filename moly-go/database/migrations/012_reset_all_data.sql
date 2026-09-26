-- Migration 012: Reset Database
-- Date: September 26, 2026
-- Purpose: Remove all user data, keep schema fresh
-- Impact: Clears all users, contacts, interactions, reflections, etc.
-- WARNING: This is destructive - all data will be deleted

-- Disable foreign key constraints to allow deletion
PRAGMA foreign_keys = OFF;

-- Delete all data from all tables (in order to respect foreign keys if re-enabled)
DELETE FROM login_codes;
DELETE FROM suggestion_choices;
DELETE FROM safety_incidents;
DELETE FROM user_interactions;
DELETE FROM behavior_patterns;
DELETE FROM reflections;
DELETE FROM interactions;
DELETE FROM contacts;
DELETE FROM about_me;
DELETE FROM users;

-- Reset auto-increment counters
DELETE FROM sqlite_sequence;

-- Re-enable foreign key constraints
PRAGMA foreign_keys = ON;
