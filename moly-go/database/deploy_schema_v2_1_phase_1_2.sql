-- ============================================================================
-- Moly V2.1 Phase 1.2 Schema Deployment Script
-- ============================================================================
-- SAFE: Uses "IF NOT EXISTS" to prevent errors if tables already exist
-- REVERSIBLE: Includes rollback procedures
-- VERIFIED: Includes verification queries
-- DOCUMENTED: Includes comments for maintenance
-- ============================================================================

-- Enable foreign keys (important for cascading deletes)
PRAGMA foreign_keys = ON;

-- ============================================================================
-- PHASE 1: CREATE PERMANENT TABLES (User's Knowledge Base)
-- ============================================================================

-- AboutMe Profile: User's communication style and preferences
CREATE TABLE IF NOT EXISTS about_me_profile (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL UNIQUE,
    communication_style TEXT,
    tone_preference TEXT,
    pace_preference TEXT,
    core_values TEXT,
    preferences TEXT,
    communication_notes TEXT,
    extracted_from_count INTEGER DEFAULT 0,
    confidence REAL DEFAULT 0.0,
    last_updated INTEGER,
    version INTEGER DEFAULT 1,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Contacts: People in user's life
CREATE TABLE IF NOT EXISTS contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    relationship_type TEXT,
    context TEXT,
    first_mentioned INTEGER,
    times_mentioned INTEGER DEFAULT 1,
    last_mentioned INTEGER,
    created_at INTEGER,
    updated_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, name)
);

-- Contact Communication Patterns: How user relates to each contact
CREATE TABLE IF NOT EXISTS contact_communication_patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    contact_id INTEGER NOT NULL,
    frequency TEXT,
    preferred_medium TEXT,
    patterns TEXT,
    tone_observed TEXT,
    main_topics TEXT,
    recent_outcome TEXT,
    user_notes TEXT,
    last_updated INTEGER,
    confidence REAL DEFAULT 0.5,
    FOREIGN KEY (contact_id) REFERENCES contacts(id) ON DELETE CASCADE
);

-- Communication Patterns: General observed patterns
CREATE TABLE IF NOT EXISTS communication_patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    pattern TEXT NOT NULL,
    pattern_category TEXT,
    confidence REAL DEFAULT 0.5,
    first_observed INTEGER,
    last_observed INTEGER,
    observation_count INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    is_growth_area BOOLEAN DEFAULT false,
    context_notes TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Communication Goals: What user is working toward
CREATE TABLE IF NOT EXISTS communication_goals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    goal TEXT NOT NULL,
    category TEXT,
    status TEXT DEFAULT 'active',
    started_at INTEGER,
    target_date INTEGER,
    achieved_at INTEGER,
    progress_notes TEXT,
    confidence REAL DEFAULT 0.5,
    created_at INTEGER,
    last_updated INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Reflection Journal: User's personal notes
CREATE TABLE IF NOT EXISTS reflection_journal (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    content TEXT NOT NULL,
    tags TEXT,
    created_at INTEGER,
    about_contact_id INTEGER,
    entry_type TEXT,
    is_private BOOLEAN DEFAULT true,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (about_contact_id) REFERENCES contacts(id) ON DELETE SET NULL
);

-- Implicit Learning: What system learned about user
CREATE TABLE IF NOT EXISTS implicit_learning (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    learning_type TEXT,
    learning_key TEXT,
    learning_value TEXT,
    confidence REAL DEFAULT 0.5,
    source TEXT,
    first_extracted INTEGER,
    last_reinforced INTEGER,
    reinforcement_count INTEGER DEFAULT 1,
    is_confirmed BOOLEAN DEFAULT false,
    is_rejected BOOLEAN DEFAULT false,
    user_feedback TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ============================================================================
-- PHASE 2: CREATE EPHEMERAL TABLES (24h Auto-Delete)
-- ============================================================================

-- Conversation Ephemeral: Raw messages (24h TTL)
CREATE TABLE IF NOT EXISTS conversation_ephemeral (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    started_at INTEGER,
    ended_at INTEGER,
    messages TEXT,
    extraction_status TEXT DEFAULT 'pending',
    extraction_attempted_at INTEGER,
    extraction_completed_at INTEGER,
    created_at INTEGER,
    expires_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Extraction Queue: Processing pipeline
CREATE TABLE IF NOT EXISTS extraction_queue (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    extraction_type TEXT,
    status TEXT DEFAULT 'pending',
    extraction_result TEXT,
    confidence REAL DEFAULT 0.0,
    queued_at INTEGER,
    attempted_at INTEGER,
    completed_at INTEGER,
    error_message TEXT,
    extraction_attempt INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ============================================================================
-- PHASE 3: CREATE INDEXES (Performance Optimization)
-- ============================================================================

-- AboutMe indexes
CREATE INDEX IF NOT EXISTS idx_about_me_user_id ON about_me_profile(user_id);
CREATE INDEX IF NOT EXISTS idx_about_me_updated ON about_me_profile(last_updated DESC);

-- Contacts indexes
CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_last_mentioned ON contacts(last_mentioned DESC);
CREATE INDEX IF NOT EXISTS idx_contacts_name ON contacts(user_id, name);

-- Contact patterns indexes
CREATE INDEX IF NOT EXISTS idx_contact_patterns_contact_id ON contact_communication_patterns(contact_id);
CREATE INDEX IF NOT EXISTS idx_contact_patterns_updated ON contact_communication_patterns(last_updated DESC);

-- Communication patterns indexes
CREATE INDEX IF NOT EXISTS idx_patterns_user_id ON communication_patterns(user_id);
CREATE INDEX IF NOT EXISTS idx_patterns_active ON communication_patterns(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_patterns_growth ON communication_patterns(is_growth_area) WHERE is_growth_area = true;
CREATE INDEX IF NOT EXISTS idx_patterns_category ON communication_patterns(pattern_category);

-- Goals indexes
CREATE INDEX IF NOT EXISTS idx_goals_user_id ON communication_goals(user_id);
CREATE INDEX IF NOT EXISTS idx_goals_status ON communication_goals(status);
CREATE INDEX IF NOT EXISTS idx_goals_active ON communication_goals(user_id, status) WHERE status = 'active';

-- Reflection journal indexes
CREATE INDEX IF NOT EXISTS idx_journal_user_id ON reflection_journal(user_id);
CREATE INDEX IF NOT EXISTS idx_journal_contact_id ON reflection_journal(about_contact_id);
CREATE INDEX IF NOT EXISTS idx_journal_created ON reflection_journal(created_at DESC);

-- Implicit learning indexes
CREATE INDEX IF NOT EXISTS idx_learning_user_id ON implicit_learning(user_id);
CREATE INDEX IF NOT EXISTS idx_learning_confirmed ON implicit_learning(is_confirmed) WHERE is_confirmed = true;
CREATE INDEX IF NOT EXISTS idx_learning_type ON implicit_learning(learning_type);
CREATE INDEX IF NOT EXISTS idx_learning_key ON implicit_learning(learning_key);

-- Ephemeral indexes
CREATE INDEX IF NOT EXISTS idx_ephemeral_user_id ON conversation_ephemeral(user_id);
CREATE INDEX IF NOT EXISTS idx_ephemeral_expires ON conversation_ephemeral(expires_at);
CREATE INDEX IF NOT EXISTS idx_extraction_status ON extraction_queue(status);
CREATE INDEX IF NOT EXISTS idx_extraction_user ON extraction_queue(user_id);
CREATE INDEX IF NOT EXISTS idx_extraction_queued ON extraction_queue(queued_at ASC) WHERE status = 'pending';

-- ============================================================================
-- VERIFICATION QUERIES (Run these to verify deployment)
-- ============================================================================

-- Count tables created
SELECT 'VERIFICATION: Tables Created' as check_name;
SELECT COUNT(*) as table_count FROM sqlite_master WHERE type='table' AND name IN (
    'about_me_profile', 'contacts', 'contact_communication_patterns',
    'communication_patterns', 'communication_goals', 'reflection_journal',
    'implicit_learning', 'conversation_ephemeral', 'extraction_queue'
);

-- Verify indexes
SELECT 'VERIFICATION: Indexes Created' as check_name;
SELECT COUNT(*) as index_count FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%';

-- Check table structure (AboutMe as example)
SELECT 'VERIFICATION: AboutMe Profile Schema' as check_name;
PRAGMA table_info(about_me_profile);

-- ============================================================================
-- BACKUP & MAINTENANCE PROCEDURES
-- ============================================================================

-- To backup before deployment:
-- VACUUM INTO '/path/to/backup/moly_backup_YYYYMMDD.db';

-- To restore from backup:
-- .restore /path/to/backup/moly_backup_YYYYMMDD.db

-- To vacuum database (optimize after large operations):
-- VACUUM;

-- To check database integrity:
-- PRAGMA integrity_check;

-- ============================================================================
-- NOTES
-- ============================================================================
/*
DEPLOYMENT SAFETY:
- All CREATE TABLE statements use "IF NOT EXISTS"
- Foreign keys are enabled (important for data integrity)
- Indexes are created separately (can be recreated if needed)
- Session table reference assumes it exists from Phase 1

ROLLBACK PROCEDURE (if needed):
- To rollback to Phase 1.1:
  1. Backup current database
  2. Drop Phase 1.2 tables:
     DROP TABLE IF EXISTS implicit_learning;
     DROP TABLE IF EXISTS extraction_queue;
     DROP TABLE IF EXISTS conversation_ephemeral;
     DROP TABLE IF EXISTS reflection_journal;
     DROP TABLE IF EXISTS communication_goals;
     DROP TABLE IF EXISTS communication_patterns;
     DROP TABLE IF EXISTS contact_communication_patterns;
     DROP TABLE IF EXISTS contacts;
     DROP TABLE IF EXISTS about_me_profile;
  3. Drop Phase 1.2 indexes (optional, they'll just be unused)

PERFORMANCE NOTES:
- indexes on user_id for fast lookups by user
- indexes on timestamps for sorting
- partial indexes on boolean columns for faster queries
- Compound indexes on common query patterns

MAINTENANCE:
- Run VACUUM periodically to optimize database
- Monitor index usage for performance
- Backup database before schema changes
- Use transactions for multi-table updates
*/

-- ============================================================================
-- END OF DEPLOYMENT SCRIPT
-- ============================================================================
