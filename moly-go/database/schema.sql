-- Moly V2 Database Schema
-- SQLite with optimized indexes for performance

-- about_me: User's communication profile and preferences
CREATE TABLE IF NOT EXISTS about_me (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL UNIQUE,
    communication_style TEXT,
    "values" TEXT, -- JSON array (quoted because it's a reserved word)
    preferred_tone TEXT,
    notes TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    version INTEGER DEFAULT 1,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- contacts: People the user is communicating about/with
CREATE TABLE IF NOT EXISTS contacts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    relationship TEXT,
    age TEXT,
    characteristics TEXT, -- JSON array
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, name)
);

-- interactions: Message history and conversation records
CREATE TABLE IF NOT EXISTS interactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    content TEXT NOT NULL,
    type TEXT, -- "user", "agent", "suggestion", "question"
    intention TEXT,
    intention_confidence REAL,
    emotional_tone TEXT,
    timestamp INTEGER NOT NULL,
    metadata TEXT, -- JSON for additional context
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- behavior_patterns: Learned patterns about user behavior
CREATE TABLE IF NOT EXISTS behavior_patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL UNIQUE,
    total_interactions INTEGER DEFAULT 0,
    modification_rate REAL DEFAULT 0.0,
    preferred_tones TEXT, -- JSON map
    communication_style TEXT,
    growth_trend TEXT,
    last_analyzed INTEGER,
    confidence REAL DEFAULT 0.5,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- reflections: System insights about user personality and patterns
CREATE TABLE IF NOT EXISTS reflections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    characteristics TEXT, -- JSON array
    interests TEXT, -- JSON array
    intentions TEXT, -- JSON array
    status TEXT DEFAULT 'pending_approval', -- "pending_approval", "approved", "rejected"
    created_at INTEGER NOT NULL,
    approved_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- suggestion_choices: Track which suggestions user selects/modifies
CREATE TABLE IF NOT EXISTS suggestion_choices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    suggestion_id TEXT NOT NULL,
    suggested_text TEXT NOT NULL,
    user_modification TEXT,
    chosen_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- safety_incidents: Track safety-relevant events
CREATE TABLE IF NOT EXISTS safety_incidents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    severity TEXT, -- "low", "medium", "high", "critical"
    detected_at INTEGER NOT NULL,
    content TEXT,
    detected_by TEXT, -- "heuristic", "llm", "manual"
    response_provided TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- users: User master records
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL,
    last_active INTEGER NOT NULL,
    context_level TEXT DEFAULT 'minimal', -- "minimal", "partial", "comprehensive"
    safety_tier TEXT DEFAULT 'standard' -- "standard", "elevated", "managed"
);

-- sessions: V2.1 session management (24-hour expiry)
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    code TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    last_active INTEGER NOT NULL,
    device_name TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, code)
);

-- chat_messages: V2.1 encrypted chat history
CREATE TABLE IF NOT EXISTS chat_messages (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    role TEXT NOT NULL, -- "user" or "assistant"
    content TEXT NOT NULL,
    context_extracted TEXT, -- JSON
    contact_mention TEXT, -- JSON
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- INDEXES for performance
CREATE INDEX IF NOT EXISTS idx_about_me_user_id ON about_me(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_interactions_user_id ON interactions(user_id);
CREATE INDEX IF NOT EXISTS idx_interactions_conversation_id ON interactions(conversation_id);
CREATE INDEX IF NOT EXISTS idx_interactions_timestamp ON interactions(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_behavior_patterns_user_id ON behavior_patterns(user_id);
CREATE INDEX IF NOT EXISTS idx_reflections_user_id ON reflections(user_id);
CREATE INDEX IF NOT EXISTS idx_suggestion_choices_user_id ON suggestion_choices(user_id);
CREATE INDEX IF NOT EXISTS idx_safety_incidents_user_id ON safety_incidents(user_id);
CREATE INDEX IF NOT EXISTS idx_safety_incidents_severity ON safety_incidents(severity);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_messages_user_id ON chat_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_id ON chat_messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_created_at ON chat_messages(created_at DESC);
