-- Migration 001: Create base schema
-- Date: Sep 7, 2026
-- Description: Create users, about_me, contacts, conversations, and messages tables

-- Users table
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL
);

-- About Me: User's own communication profile
CREATE TABLE IF NOT EXISTS about_me (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL UNIQUE,
  communication_style TEXT,
  values JSONB,
  preferred_tone TEXT,
  notes TEXT,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  version INTEGER DEFAULT 1,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Contacts: User's observations of contacts (NOT surveillance)
CREATE TABLE IF NOT EXISTS contacts (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  name TEXT NOT NULL,
  relationship TEXT,
  characteristics JSONB,
  interests JSONB,
  communication_preferences TEXT,
  notes TEXT,
  created_at BIGINT NOT NULL,
  updated_at BIGINT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Conversations: Metadata about conversations
CREATE TABLE IF NOT EXISTS conversations (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  contact_id TEXT,
  contact_name TEXT,
  message_count INTEGER DEFAULT 0,
  last_message TEXT,
  last_message_time BIGINT,
  created_at BIGINT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (contact_id) REFERENCES contacts(id)
);

-- Messages: Individual messages in conversations
CREATE TABLE IF NOT EXISTS messages (
  id TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL,
  role TEXT NOT NULL, -- 'user' or 'assistant'
  content TEXT NOT NULL,
  type TEXT,
  metadata JSONB,
  timestamp BIGINT NOT NULL,
  FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

-- Reflections: Extracted insights pending approval
CREATE TABLE IF NOT EXISTS reflections (
  id TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL,
  contact_id TEXT,
  user_id TEXT NOT NULL,
  characteristics JSONB,
  interests JSONB,
  communication_preferences TEXT,
  intentions JSONB,
  user_quotes JSONB,
  status TEXT DEFAULT 'pending_approval', -- 'pending_approval', 'approved', 'rejected'
  user_edits JSONB,
  created_at BIGINT NOT NULL,
  approved_at BIGINT,
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (conversation_id) REFERENCES conversations(id),
  FOREIGN KEY (contact_id) REFERENCES contacts(id)
);
