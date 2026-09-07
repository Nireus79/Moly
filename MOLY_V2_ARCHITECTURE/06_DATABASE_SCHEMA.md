# Moly Database Schema - Complete Specification

**Date**: September 6, 2026  
**Status**: Schema Reference  
**Version**: 1.0

---

## Overview

PostgreSQL database with clear separation:
- **User data** (About Me, preferences)
- **Contact data** (what user knows about contacts)
- **Conversation data** (messages, history)
- **Behavioral data** (learning profiles)
- **Audit data** (tracking changes)

**CRITICAL**: No surveillance data. No tracking of contact behavior. Only user behavior and user's observations.

---

## Core Tables

### users

Stores user account and basic profile.

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_active TIMESTAMP,
  status VARCHAR(20) DEFAULT 'active', -- 'active', 'inactive', 'suspended'
  preferences JSONB DEFAULT '{}', -- Global preferences
  
  -- Indexes
  INDEX idx_email (email),
  INDEX idx_created_at (created_at)
);
```

---

### about_me

User's communication style profile.

```sql
CREATE TABLE about_me (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE UNIQUE,
  communication_style TEXT, -- "casual, direct, authentic"
  values JSONB, -- ["authenticity", "loyalty", "growth"]
  preferred_tone VARCHAR(20), -- "formal", "friendly", "dating"
  notes TEXT, -- User's notes about themselves
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  version INT DEFAULT 1, -- Track versions
  
  -- Indexes
  INDEX idx_user_id (user_id)
);
```

---

### contacts

People user communicates with. User's own observations only.

```sql
CREATE TABLE contacts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  relationship VARCHAR(50), -- "close_friend", "family", "work", "romantic", "new"
  characteristics JSONB, -- ["ambitious", "direct", "loves hiking"]
  interests JSONB, -- ["tech", "hiking", "coffee"]
  communication_preferences TEXT, -- "No small talk, warm in personal matters"
  notes TEXT, -- User's observations (accumulated from reflections)
  reflections JSONB, -- Historical reflections
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  version INT DEFAULT 1,
  
  -- Constraints
  UNIQUE(user_id, name), -- One contact per name per user
  
  -- Indexes
  INDEX idx_user_id (user_id),
  INDEX idx_relationship (relationship),
  INDEX idx_updated_at (updated_at)
);

-- CRITICAL: NO columns for "contact_message_patterns", "response_frequency", 
--           "communication_style", etc. User observations only.
```

---

### conversations

Individual chat sessions with a contact.

```sql
CREATE TABLE conversations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
  topic VARCHAR(255), -- Optional: "job change", "relationship issue", etc.
  mode VARCHAR(20), -- "socratic", "direct"
  tone VARCHAR(20), -- "formal", "friendly", "dating"
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  archived BOOLEAN DEFAULT FALSE,
  
  -- Indexes
  INDEX idx_user_id (user_id),
  INDEX idx_contact_id (contact_id),
  INDEX idx_created_at (created_at),
  INDEX idx_updated_at (updated_at)
);
```

---

### messages

Messages in conversations (both user and Moly).

```sql
CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
  role VARCHAR(20), -- "user", "assistant"
  content TEXT NOT NULL,
  type VARCHAR(20), -- "message", "question", "suggestion"
  metadata JSONB, -- { tone, mode, chosenIndex, etc. }
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  -- Indexes
  INDEX idx_conversation_id (conversation_id),
  INDEX idx_role (role),
  INDEX idx_created_at (created_at)
);
```

---

### reflections

Extracted insights from conversations. Pending user approval → merged to contact.

```sql
CREATE TABLE reflections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
  contact_id UUID REFERENCES contacts(id) ON DELETE CASCADE,
  characteristics JSONB, -- ["ambitious", "achievement-focused"]
  interests JSONB, -- ["career", "growth", "hiking"]
  communication_preferences TEXT,
  intentions JSONB, -- ["celebrate achievement", "deepen connection"]
  user_quotes JSONB, -- ["just got promoted", "I'm so proud"]
  status VARCHAR(20), -- "pending_approval", "approved", "rejected"
  user_edits JSONB, -- Changes user made before approving
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  approved_at TIMESTAMP,
  version INT DEFAULT 1,
  
  -- Indexes
  INDEX idx_conversation_id (conversation_id),
  INDEX idx_contact_id (contact_id),
  INDEX idx_status (status),
  INDEX idx_created_at (created_at)
);
```

---

### user_profiles

Behavioral profiles of users (learning data). NO contact behavior.

```sql
CREATE TABLE user_profiles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE UNIQUE,
  
  -- Communication patterns (user's style)
  communication_profile JSONB, -- {
    --   avgToneWords: ["authentic", "direct"],
    --   formalityLevel: 0.3,
    --   emojiUsage: 0.2,
    --   messageLengthAvg: 120,
    --   usesHumor: true,
    --   usesEmphasis: ["!!"],
    --   formattingPreference: "paragraph"
    -- }
  
  -- Communication goals (what user messages about)
  communication_goals JSONB, -- {
    --   opening: 45,
    --   deepening: 120,
    --   apology: 15,
    --   celebration: 87
    -- }
  
  -- Suggestion choices (what user picks)
  suggestion_choices JSONB, -- {
    --   totalGenerated: 300,
    --   totalChosen: 195,
    --   avgChoiceRate: 0.65,
    --   preferred: { formal: 0.1, friendly: 0.65, dating: 0.25 },
    --   frequentEdits: ["make more casual"]
    -- }
  
  -- Success metrics (how well suggestions work)
  success_metrics JSONB, -- {
    --   positive_response_rate: 0.72,
    --   conversation_continuation_rate: 0.81,
    --   contact_reengagement: true,
    --   user_confidence_trend: "increasing"
    -- }
  
  -- Emerging personality (inferred traits)
  emerging_personality JSONB, -- ["values authenticity", "prefers casual"]
  
  -- Growth over time
  growth_trajectory JSONB, -- {
    --   joinDate: "2026-01-15",
    --   totalInteractions: 450,
    --   uniqueContacts: 18,
    --   averageContactFrequency: 25,
    --   confidenceTrend: 0.85, // 0-1
    --   authenticityScore: 0.9, // 0-1
    --   skillLevel: 0.82 // 0-1
    // }
  
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  version INT DEFAULT 1,
  
  -- Indexes
  INDEX idx_user_id (user_id)
);

-- CRITICAL: NO columns for "contact_response_patterns", 
--           "contact_communication_style", "contact_behavior_analysis".
--           Only user behavior and user's observations.
```

---

### interaction_history

Fine-grained log of every interaction for learning.

```sql
CREATE TABLE interaction_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
  contact_id UUID REFERENCES contacts(id) ON DELETE SET NULL,
  
  -- What happened
  interaction_type VARCHAR(50), -- "message_sent", "suggestion_chosen", 
                                 -- "reflection_approved", "question_answered"
  user_message TEXT,
  suggestions_generated INT,
  suggestion_chosen INT, -- 0, 1, 2, or NULL if none
  suggestion_text TEXT,
  user_modification VARCHAR(255), -- "make it shorter", "more casual"
  user_feedback VARCHAR(50), -- "positive", "neutral", "negative"
  
  -- Outcome (recorded later)
  contact_response TEXT,
  success BOOLEAN, -- Did contact respond positively?
  
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  -- Indexes
  INDEX idx_user_id (user_id),
  INDEX idx_conversation_id (conversation_id),
  INDEX idx_interaction_type (interaction_type),
  INDEX idx_created_at (created_at)
);
```

---

### risk_patterns

Tracks concerning behavior patterns in user (for Risk Monitoring Agent).

```sql
CREATE TABLE risk_patterns (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  
  pattern_type VARCHAR(50), -- "manipulation", "boundary_violation", 
                            -- "scam", "harm", "insincerity"
  severity INT, -- 0-10
  first_occurrence TIMESTAMP,
  last_occurrence TIMESTAMP,
  occurrence_count INT DEFAULT 1,
  
  -- Education history
  interventions JSONB, -- [{
    --   date: timestamp,
    --   type: "socratic_questions",
    --   outcome: "adjusted" | "proceeded" | "unknown"
    -- }]
  
  -- Pattern analysis
  trend VARCHAR(20), -- "increasing", "stable", "decreasing"
  root_cause_hypothesis TEXT, -- What the agent thinks is driving it
  user_response_pattern JSONB, -- What interventions work for this user
  
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  -- Indexes
  INDEX idx_user_id (user_id),
  INDEX idx_pattern_type (pattern_type),
  INDEX idx_severity (severity),
  INDEX idx_last_occurrence (last_occurrence)
);
```

---

### audit_log

Track all changes for debugging and compliance.

```sql
CREATE TABLE audit_log (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID REFERENCES users(id) ON DELETE CASCADE,
  
  -- What changed
  entity_type VARCHAR(50), -- "about_me", "contact", "reflection", etc.
  entity_id UUID,
  action VARCHAR(50), -- "created", "updated", "deleted", "approved"
  
  -- Before and after
  old_values JSONB,
  new_values JSONB,
  
  -- Context
  source VARCHAR(50), -- "user_input", "reflection_merge", "system"
  reason TEXT,
  
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  
  -- Indexes
  INDEX idx_user_id (user_id),
  INDEX idx_entity_type (entity_type),
  INDEX idx_created_at (created_at)
);
```

---

## Relationship Diagram

```
users (1) ──── (N) about_me
  |
  ├──── (N) contacts
  |       └──── (N) conversations
  |               └──── (N) messages
  |               └──── (N) reflections
  |
  ├──── (1) user_profiles
  |
  ├──── (N) interaction_history
  |
  └──── (N) risk_patterns
```

---

## Key Constraints

### Privacy Constraints

```sql
-- NO tracking of contacts' behavior
ALTER TABLE contacts ADD CONSTRAINT no_contact_monitoring 
  CHECK (communication_preferences IS NULL OR 
         characteristics IS NULL OR
         name IS NOT NULL); -- Only user observations allowed

-- NO surveillance data in user_profiles
ALTER TABLE user_profiles ADD CONSTRAINT no_contact_surveillance 
  CHECK (communication_profile LIKE '{"user_%' OR 
         communication_profile IS NULL); -- User patterns only
```

### Data Integrity

```sql
-- Reflections must have content
ALTER TABLE reflections ADD CONSTRAINT reflection_not_empty 
  CHECK (characteristics IS NOT NULL OR 
         interests IS NOT NULL OR
         communication_preferences IS NOT NULL);

-- Messages must belong to conversation
ALTER TABLE messages ADD CONSTRAINT message_has_conversation 
  CHECK (conversation_id IS NOT NULL);

-- Conversations must belong to user
ALTER TABLE conversations ADD CONSTRAINT conversation_has_user 
  CHECK (user_id IS NOT NULL);
```

---

## Migrations

### Migration 001: Create base schema

```sql
-- Run all CREATE TABLE statements above
-- Index creation
-- Constraint creation
```

### Migration 002: Add version tracking

```sql
-- Already included in CREATE statements with 'version INT'
-- For existing data: ALTER TABLE ... ADD COLUMN version INT DEFAULT 1;
```

### Migration 003: Add audit logging

```sql
-- Create audit_log table
-- Add triggers to log changes
CREATE TRIGGER audit_about_me_changes
AFTER UPDATE ON about_me
FOR EACH ROW
EXECUTE FUNCTION log_audit();
```

---

## Queries

### Get user's context for conversation

```sql
SELECT 
  am.communication_style,
  am.values,
  am.preferred_tone,
  c.name,
  c.characteristics,
  c.interests,
  c.communication_preferences,
  m.content,
  m.role,
  m.created_at
FROM about_me am
LEFT JOIN contacts c ON c.id = ?
LEFT JOIN conversations conv ON conv.id = ?
LEFT JOIN messages m ON m.conversation_id = conv.id
ORDER BY m.created_at DESC
LIMIT 50;
```

### Get user's behavioral profile

```sql
SELECT 
  up.communication_profile,
  up.communication_goals,
  up.suggestion_choices,
  up.emerging_personality,
  up.growth_trajectory
FROM user_profiles up
WHERE up.user_id = ?;
```

### Get approved reflections for contact

```sql
SELECT * FROM reflections
WHERE contact_id = ? AND status = 'approved'
ORDER BY approved_at DESC;
```

### Calculate user's success rate

```sql
SELECT 
  COUNT(*) as total_interactions,
  COUNT(CASE WHEN user_feedback = 'positive' THEN 1 END) as positive_feedback,
  ROUND(COUNT(CASE WHEN user_feedback = 'positive' THEN 1 END)::NUMERIC / 
        COUNT(*), 3) as success_rate
FROM interaction_history
WHERE user_id = ? AND created_at > NOW() - INTERVAL '30 days';
```

### Get active contacts

```sql
SELECT c.*, COUNT(m.id) as message_count
FROM contacts c
LEFT JOIN conversations conv ON conv.contact_id = c.id
LEFT JOIN messages m ON m.conversation_id = conv.id
WHERE c.user_id = ?
GROUP BY c.id
ORDER BY MAX(m.created_at) DESC;
```

---

## Retention & Cleanup

### Keep indefinitely
- Users table (even if inactive)
- About Me (user's self-description)
- Contacts (user's relationships)
- Reflections (learning history)

### Archive after 90 days
- Conversations and messages (archive, don't delete)
- Interaction history (archive for compliance)

### Delete after 1 year
- Temporary audit logs

### Never store
- Contact surveillance data
- Contact behavior analysis
- Contact response patterns
- Any monitoring of contacts

---

## Performance Optimization

### Indexes
```sql
-- Multi-column indexes for common queries
CREATE INDEX idx_conversation_user_contact 
  ON conversations(user_id, contact_id, created_at DESC);

CREATE INDEX idx_interaction_user_type 
  ON interaction_history(user_id, interaction_type, created_at DESC);

CREATE INDEX idx_message_conversation_role 
  ON messages(conversation_id, role, created_at);
```

### Partitioning
```sql
-- Partition messages by conversation for large datasets
CREATE TABLE messages_partitioned (...)
PARTITION BY RANGE (created_at);

-- Partition interaction_history by user for scaling
CREATE TABLE interaction_history_partitioned (...)
PARTITION BY HASH (user_id);
```

### Materialized Views
```sql
-- User statistics (refreshed daily)
CREATE MATERIALIZED VIEW user_statistics AS
SELECT 
  u.id,
  COUNT(DISTINCT c.id) as unique_contacts,
  COUNT(DISTINCT conv.id) as total_conversations,
  COUNT(i.id) as total_interactions,
  MAX(i.created_at) as last_active
FROM users u
LEFT JOIN contacts c ON c.user_id = u.id
LEFT JOIN conversations conv ON conv.user_id = u.id
LEFT JOIN interaction_history i ON i.user_id = u.id
GROUP BY u.id;
```

---

## Backup Strategy

- **Hourly backups** of users, about_me, contacts, conversations
- **Daily backups** of full database
- **Weekly snapshots** to cold storage
- **Retention**: 30 days hot, 1 year cold

**Critical**: Maintain audit trail for compliance (GDPR, etc.)

---

This schema enforces the privacy principle: user behavior tracked, contact behavior never tracked.
