-- Migration: 003_add_socratic_tracking.sql
-- Purpose: Add columns to track Socratic questions and principle violations
-- Date: 2026-09-15
-- Description: Support for Socratic question framework integration

-- ============================================================================
-- Extend clarification_questions table with Socratic metadata
-- ============================================================================

ALTER TABLE clarification_questions ADD COLUMN (
    socratic_approach VARCHAR(50),       -- e.g., "identifying_stakeholders"
    targets_principle VARCHAR(100),      -- e.g., "user_autonomy"
    targets_framework VARCHAR(50),       -- e.g., "rights_based"
    expected_insights TEXT,              -- JSON array of expected insights
    depth_level INT,                     -- 1-5 progression
    follow_up_questions TEXT             -- JSON array of follow-up question IDs
);

-- Create index for faster lookups by approach
CREATE INDEX idx_clarification_socratic_approach 
ON clarification_questions(socratic_approach);

CREATE INDEX idx_clarification_targets_principle 
ON clarification_questions(targets_principle);

-- ============================================================================
-- Track principle violations in responses
-- ============================================================================

CREATE TABLE IF NOT EXISTS principle_violations (
    id VARCHAR(50) PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL,
    message_id VARCHAR(50),
    conversation_id VARCHAR(50),
    principle_name VARCHAR(100) NOT NULL,           -- e.g., "user_autonomy"
    severity VARCHAR(50) NOT NULL,                  -- critical, high, medium
    violation_type VARCHAR(100),                    -- e.g., "pressuring_user"
    description TEXT,                               -- Why this violates the principle
    response_snippet TEXT,                          -- The problematic response text
    created_at BIGINT NOT NULL,
    resolved BOOLEAN DEFAULT FALSE,                 -- Has this been addressed?
    resolution_notes TEXT
);

CREATE INDEX idx_principle_violations_user 
ON principle_violations(user_id);

CREATE INDEX idx_principle_violations_principle 
ON principle_violations(principle_name);

CREATE INDEX idx_principle_violations_severity 
ON principle_violations(severity);

-- ============================================================================
-- Track question effectiveness
-- ============================================================================

CREATE TABLE IF NOT EXISTS question_effectiveness (
    id VARCHAR(50) PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL,
    question_id VARCHAR(50) NOT NULL,              -- ID of the question asked
    socratic_approach VARCHAR(50),                  -- Approach used
    question_text TEXT,                             -- The question that was asked
    user_response TEXT,                             -- User's answer
    
    -- Effectiveness metrics
    reduced_ambiguity BOOLEAN,                      -- Did this help clarify?
    insight_gained TEXT,                            -- What was revealed?
    depth_level_advanced BOOLEAN,                   -- Should we go deeper?
    principle_clarified VARCHAR(100),               -- Which principle became clearer?
    
    -- Metadata
    created_at BIGINT NOT NULL,
    response_quality ENUM('unhelpful', 'somewhat_helpful', 'very_helpful'),
    follow_up_used VARCHAR(50)                      -- Which follow-up was used?
);

CREATE INDEX idx_question_effectiveness_user 
ON question_effectiveness(user_id);

CREATE INDEX idx_question_effectiveness_question 
ON question_effectiveness(question_id);

CREATE INDEX idx_question_effectiveness_approach 
ON question_effectiveness(socratic_approach);

-- ============================================================================
-- Rollback instructions (if needed)
-- ============================================================================

-- To rollback this migration:
-- DROP TABLE IF EXISTS question_effectiveness;
-- DROP TABLE IF EXISTS principle_violations;
-- ALTER TABLE clarification_questions DROP COLUMN socratic_approach;
-- ALTER TABLE clarification_questions DROP COLUMN targets_principle;
-- ALTER TABLE clarification_questions DROP COLUMN targets_framework;
-- ALTER TABLE clarification_questions DROP COLUMN expected_insights;
-- ALTER TABLE clarification_questions DROP COLUMN depth_level;
-- ALTER TABLE clarification_questions DROP COLUMN follow_up_questions;

