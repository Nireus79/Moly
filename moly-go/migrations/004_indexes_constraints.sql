-- Migration 003: Create indexes and constraints
-- Date: Sep 7, 2026
-- Description: Optimize query performance and enforce data integrity

-- Indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_about_me_user_id ON about_me(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_contact_id ON conversations(contact_id);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_reflections_conversation_id ON reflections(conversation_id);
CREATE INDEX IF NOT EXISTS idx_reflections_contact_id ON reflections(contact_id);
CREATE INDEX IF NOT EXISTS idx_reflections_user_id ON reflections(user_id);
CREATE INDEX IF NOT EXISTS idx_reflections_status ON reflections(status);

-- Behavioral tracking indexes
CREATE INDEX IF NOT EXISTS idx_user_behavioral_profiles_user_id ON user_behavioral_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_interactions_user_id ON user_interactions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_interactions_conversation_id ON user_interactions(conversation_id);
CREATE INDEX IF NOT EXISTS idx_suggestion_choices_user_id ON suggestion_choices(user_id);
CREATE INDEX IF NOT EXISTS idx_suggestion_choices_conversation_id ON suggestion_choices(conversation_id);
CREATE INDEX IF NOT EXISTS idx_user_risk_profiles_user_id ON user_risk_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_risk_patterns_user_id ON risk_patterns(user_id);
CREATE INDEX IF NOT EXISTS idx_risk_patterns_pattern_type ON risk_patterns(pattern_type);

-- Audit log indexes
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp);

-- Check constraints: Privacy enforcement
ALTER TABLE user_interactions ADD CONSTRAINT check_no_contact_data
  CHECK (user_message IS NOT NULL);

ALTER TABLE suggestion_choices ADD CONSTRAINT check_valid_feedback
  CHECK (user_feedback IN ('positive', 'neutral', 'negative', NULL));

ALTER TABLE risk_patterns ADD CONSTRAINT check_severity_range
  CHECK (severity >= 0 AND severity <= 10);

-- Ensure only user behavior is tracked, never contact behavior
ALTER TABLE user_interactions ADD CONSTRAINT check_user_message_exists
  CHECK (LENGTH(TRIM(user_message)) > 0);

-- Ensure reflection status is valid
ALTER TABLE reflections ADD CONSTRAINT check_reflection_status
  CHECK (status IN ('pending_approval', 'approved', 'rejected'));
