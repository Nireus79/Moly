-- Migration 002: Create behavioral tracking tables
-- Date: Sep 7, 2026
-- Description: Create tables for user behavioral analysis (NOT contact surveillance)
-- Key principle: We learn about the USER, never about contacts

-- User Behavioral Profiles: What Moly learns about the user
CREATE TABLE IF NOT EXISTS user_behavioral_profiles (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL UNIQUE,
  communication_profile JSONB,
  communication_goals JSONB,
  suggestion_choices JSONB,
  success_metrics JSONB,
  emerging_personality JSONB,
  growth_trajectory JSONB,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  version INTEGER DEFAULT 1,
  confidence REAL DEFAULT 0.5,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- User Interactions: Record of what user did
CREATE TABLE IF NOT EXISTS user_interactions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  conversation_id TEXT NOT NULL,
  user_message TEXT,
  suggestions_generated INTEGER,
  created_at BIGINT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

-- Suggestion Choices: Track which suggestions user picked
CREATE TABLE IF NOT EXISTS suggestion_choices (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  conversation_id TEXT NOT NULL,
  suggestion_index INTEGER,
  modified_text TEXT,
  modification TEXT,
  user_feedback TEXT,
  created_at BIGINT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

-- User Risk Profiles: Tracked risk patterns for user
CREATE TABLE IF NOT EXISTS user_risk_profiles (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL UNIQUE,
  risk_patterns JSONB,
  highest_risk TEXT,
  intervention_log JSONB,
  updated_at BIGINT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Risk Patterns: Detected concerning patterns
CREATE TABLE IF NOT EXISTS risk_patterns (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  pattern_type TEXT,
  severity INTEGER,
  first_occurrence BIGINT,
  last_occurrence BIGINT,
  occurrence_count INTEGER DEFAULT 1,
  interventions JSONB,
  trend TEXT,
  root_cause_hypothesis TEXT,
  user_response_pattern JSONB,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Audit Log: Track important events
CREATE TABLE IF NOT EXISTS audit_log (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  action TEXT NOT NULL,
  details JSONB,
  timestamp BIGINT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id)
);
