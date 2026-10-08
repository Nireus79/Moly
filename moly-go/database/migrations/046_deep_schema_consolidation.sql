-- Migration 046: Deep Schema Consolidation
-- Date: October 8, 2026
-- Phase: 5 - Final Optimization
-- Purpose: Remove all minimal-use tables, consolidate to production schema

-- Tables being removed (verified as not used in active orchestrator):
-- - audit_log (1 ref - isolated utility)
-- - behavior_patterns (1 ref - isolated utility)
-- - clarification_answers (2 refs - isolated utility, replaced by responses)
-- - response_templates (3 refs - unused manager)
-- - structured_context (2 refs - isolated utility)
-- - suggestion_choices (3 refs - isolated utility)
-- - user_interactions (1 ref - isolated utility)

BEGIN TRANSACTION;

-- Step 1: Drop minimal-use tables
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS behavior_patterns;
DROP TABLE IF EXISTS clarification_answers;
DROP TABLE IF EXISTS response_templates;
DROP TABLE IF EXISTS structured_context;
DROP TABLE IF EXISTS suggestion_choices;
DROP TABLE IF EXISTS user_interactions;

-- Step 2: Clean up indexes for dropped tables
-- SQLite automatically removes indexes when tables are dropped

-- Step 3: Remove foreign key constraint references
-- (No other tables reference these tables after audit)

-- Consolidated Production Schema:
-- Core tables (actively used):
-- - users, sessions, conversations, chat_messages
-- - contacts, about_me
-- - context_attributes, context_conflicts, clarification_questions
-- - pending_input, pending_clarifications
-- - conversation_summaries, message_processing_state
-- - conversation_execution_state
-- - reflections, pronoun_resolutions, group_references
-- - sentence_analyses, extraction_sentence_linking
-- - question_history, login_codes, interactions

COMMIT;

-- Impact: Removed 7 tables, ~40 lines of schema definition
-- Result: Leaner production schema with only actively-used entities
