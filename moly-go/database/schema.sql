-- Moly V2 Database Schema
-- SQLite with optimized indexes for performance

-- users: Core user identity table (must be created first)
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE,
    name TEXT NOT NULL DEFAULT 'User',
    password_hash TEXT,
    created_at INTEGER NOT NULL,
    last_active INTEGER NOT NULL,
    updated_at INTEGER DEFAULT 0,
    context_level TEXT DEFAULT 'minimal', -- "minimal", "partial", "comprehensive"
    safety_tier TEXT DEFAULT 'standard' -- "standard", "elevated", "managed"
);

-- login_codes: Temporary login codes for authentication
CREATE TABLE IF NOT EXISTS login_codes (
    code TEXT PRIMARY KEY,
    user_id TEXT,
    expires_at INTEGER NOT NULL,
    device_id TEXT,
    used BOOLEAN NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- about_me: User's communication profile and preferences
CREATE TABLE IF NOT EXISTS about_me (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL UNIQUE,
    communication_style TEXT,
    core_values TEXT, -- JSON array (was "values" - renamed for clarity)
    tone_preference TEXT, -- was preferred_tone
    preferences TEXT, -- JSON or text for additional preferences
    goals TEXT, -- JSON array of user goals
    patterns TEXT, -- JSON array of communication patterns
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
    characteristics TEXT, -- JSON array (traits discovered)
    first_mentioned_at INTEGER, -- when contact was first mentioned
    created_via TEXT, -- "conversation", "manual", "import"
    "status" TEXT DEFAULT 'active', -- "active", "archived" (quoted because reserved word)
    version INTEGER DEFAULT 1, -- for optimistic locking
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

-- user_interactions: Record of user's message interactions for learning
CREATE TABLE IF NOT EXISTS user_interactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    user_message TEXT,
    suggestions_generated INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- reflections: System insights about user personality and patterns
CREATE TABLE IF NOT EXISTS reflections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    contact_id TEXT, -- Link to which contact this reflection is about
    message_id TEXT, -- Link to which message triggered this reflection
    characteristics TEXT, -- JSON array
    interests TEXT, -- JSON array
    intentions TEXT, -- JSON array
    extracted_style TEXT, -- Communication style used in this context
    extracted_intention TEXT, -- User's intention in this exchange
    status TEXT DEFAULT 'pending_approval', -- "pending_approval", "approved", "rejected"
    created_at INTEGER NOT NULL,
    approved_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (contact_id) REFERENCES user_contacts(id) ON DELETE SET NULL,
    FOREIGN KEY (message_id) REFERENCES chat_messages(id) ON DELETE SET NULL
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
-- sessions: V2.1 session management (30-day expiry)
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    device_id TEXT,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    last_used INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Session indexes for performance
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

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

-- context_conflicts: Conflicts when extracted context differs from saved
CREATE TABLE IF NOT EXISTS context_conflicts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    conflict_type TEXT NOT NULL, -- "aboutme_communication_style", "contact_name", etc.
    severity TEXT DEFAULT 'low', -- "low", "medium", "high"
    saved_value TEXT, -- JSON of what was previously saved
    extracted_value TEXT, -- JSON of what was just extracted
    description TEXT NOT NULL,
    status TEXT DEFAULT 'unresolved', -- "unresolved", "resolved"
    resolution TEXT, -- "keep_saved", "use_extracted", "merge"
    resolution_details TEXT, -- JSON with additional details
    created_at INTEGER NOT NULL,
    resolved_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Phase 4: Context Attributes (extracted facts with attribution)
CREATE TABLE IF NOT EXISTS context_attributes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    fact_type TEXT NOT NULL, -- "style", "trait", "value", "preference", "goal"
    fact_value TEXT NOT NULL,
    attributed_to TEXT NOT NULL, -- "user", "contact_manager_sarah", "group_team"
    context TEXT DEFAULT 'general', -- "work", "social", "family", "general"
    confidence REAL DEFAULT 0.8,
    source TEXT DEFAULT 'explicit', -- "explicit", "inferred", "stated_directly"
    evidence TEXT, -- Original quote from message
    version INTEGER DEFAULT 1,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES interactions(conversation_id) ON DELETE CASCADE
);

-- Phase 2: Clarification Questions and Responses
CREATE TABLE IF NOT EXISTS clarification_questions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    clarification_type TEXT NOT NULL, -- "subject_clarification", "contact_confirmation", "conflict_clarification", "context_clarification"
    question_text TEXT NOT NULL,
    context_notes TEXT,
    options TEXT, -- JSON array of multiple choice options
    priority INTEGER DEFAULT 2, -- 1=critical, 2=important, 3=nice-to-have
    status TEXT DEFAULT 'pending', -- "pending", "answered", "skipped"
    linked_facts TEXT, -- JSON array of fact IDs this question addresses
    created_at INTEGER NOT NULL,
    answered_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES interactions(conversation_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS clarification_responses (
    id TEXT PRIMARY KEY,
    question_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    response_text TEXT,
    selected_option TEXT,
    responded_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (question_id) REFERENCES clarification_questions(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- conversations: User conversations (topics they want to discuss with Moly)
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT,
    description TEXT,
    purpose TEXT, -- JSON description of conversation purpose
    members TEXT, -- JSON array of ConversationMember objects
    settings TEXT, -- JSON object with mode, context, llmProvider
    notes TEXT, -- Additional metadata
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- user_contacts: People the user is communicating about/with (consolidated from contacts table)
-- String IDs for API consistency (format: contact_<timestamp>_<nanoseconds>)
-- Includes all fields from both user_contacts and contacts for full consolidation
CREATE TABLE IF NOT EXISTS user_contacts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    relationship TEXT,
    age TEXT,
    characteristics TEXT, -- JSON array (traits discovered)
    first_mentioned_at INTEGER, -- when contact was first mentioned
    created_via TEXT, -- "conversation", "manual", "import"
    notes TEXT,
    status TEXT DEFAULT 'active', -- "active", "archived"
    version INTEGER DEFAULT 1, -- for optimistic locking
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, name)
);

-- audit_log: Audit trail of important system events
CREATE TABLE IF NOT EXISTS audit_log (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    details TEXT,
    timestamp INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- INDEXES for performance
CREATE INDEX IF NOT EXISTS idx_about_me_user_id ON about_me(user_id);
CREATE INDEX IF NOT EXISTS idx_contacts_user_id ON contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_contacts_user_id ON user_contacts(user_id);
CREATE INDEX IF NOT EXISTS idx_user_contacts_updated_at ON user_contacts(updated_at DESC);
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
CREATE INDEX IF NOT EXISTS idx_implicit_learning_user_id ON implicit_learning(user_id);
CREATE INDEX IF NOT EXISTS idx_implicit_learning_confirmed ON implicit_learning(is_confirmed);
CREATE INDEX IF NOT EXISTS idx_context_conflicts_user_id ON context_conflicts(user_id);
CREATE INDEX IF NOT EXISTS idx_context_conflicts_conversation_id ON context_conflicts(conversation_id);
CREATE INDEX IF NOT EXISTS idx_context_conflicts_status ON context_conflicts(status);
CREATE INDEX IF NOT EXISTS idx_context_conflicts_type ON context_conflicts(conflict_type);
CREATE INDEX IF NOT EXISTS idx_context_attributes_user_id ON context_attributes(user_id);
CREATE INDEX IF NOT EXISTS idx_context_attributes_conversation_id ON context_attributes(conversation_id);
CREATE INDEX IF NOT EXISTS idx_context_attributes_attributed_to ON context_attributes(attributed_to);
CREATE INDEX IF NOT EXISTS idx_context_attributes_fact_type ON context_attributes(fact_type);
CREATE INDEX IF NOT EXISTS idx_clarification_questions_user_id ON clarification_questions(user_id);
CREATE INDEX IF NOT EXISTS idx_clarification_questions_conversation_id ON clarification_questions(conversation_id);
CREATE INDEX IF NOT EXISTS idx_clarification_questions_status ON clarification_questions(status);
CREATE INDEX IF NOT EXISTS idx_clarification_questions_type ON clarification_questions(clarification_type);
CREATE INDEX IF NOT EXISTS idx_clarification_responses_user_id ON clarification_responses(user_id);
CREATE INDEX IF NOT EXISTS idx_clarification_responses_question_id ON clarification_responses(question_id);

-- Temporary facts awaiting clarification (for persistence across sessions)
CREATE TABLE IF NOT EXISTS pending_clarifications (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    fact_id TEXT NOT NULL UNIQUE,
    fact_type TEXT NOT NULL,
    fact_value TEXT NOT NULL,
    attributed_to TEXT NOT NULL,
    evidence TEXT,
    confidence REAL DEFAULT 0.8,
    status TEXT DEFAULT 'pending', -- "pending", "partially_answered", "complete", "abandoned"
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Answers to clarification questions
CREATE TABLE IF NOT EXISTS clarification_answers (
    id TEXT PRIMARY KEY,
    clarification_question_id TEXT NOT NULL UNIQUE,
    user_answer TEXT NOT NULL,
    answered_at INTEGER NOT NULL,
    FOREIGN KEY (clarification_question_id) REFERENCES clarification_questions(id) ON DELETE CASCADE
);

-- Conversation Execution State - tracks progress through conversation workflow
CREATE TABLE IF NOT EXISTS conversation_execution_state (
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    phase TEXT DEFAULT 'initial', -- "initial", "gathering_context", "processing", "complete"
    covered_categories TEXT, -- Comma-separated categories that have been covered
    current_message_seq INTEGER DEFAULT 0,
    started_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    version INTEGER DEFAULT 1,
    PRIMARY KEY (user_id, conversation_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Message Processing State - tracks pipeline stage completion for deduplication
-- Enables retries to skip already-completed stages, reducing token waste and API calls
CREATE TABLE IF NOT EXISTS message_processing_state (
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    completed_stages TEXT NOT NULL, -- JSON map: {"context_extraction": true, "risk_assessment": false, ...}
    stage_results TEXT NOT NULL, -- JSON map: {"context_extraction": {...}, "risk_assessment": {...}}
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    version INTEGER DEFAULT 1,
    PRIMARY KEY (user_id, conversation_id, message_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_pending_clarifications_user_id ON pending_clarifications(user_id);
CREATE INDEX IF NOT EXISTS idx_pending_clarifications_conversation_id ON pending_clarifications(conversation_id);
CREATE INDEX IF NOT EXISTS idx_pending_clarifications_status ON pending_clarifications(status);
CREATE INDEX IF NOT EXISTS idx_pending_clarifications_expires_at ON pending_clarifications(expires_at);
CREATE INDEX IF NOT EXISTS idx_clarification_answers_question_id ON clarification_answers(clarification_question_id);
CREATE INDEX IF NOT EXISTS idx_execution_state_user_id ON conversation_execution_state(user_id);
CREATE INDEX IF NOT EXISTS idx_execution_state_conversation_id ON conversation_execution_state(conversation_id);
CREATE INDEX IF NOT EXISTS idx_message_processing_state_user_id ON message_processing_state(user_id);
CREATE INDEX IF NOT EXISTS idx_message_processing_state_conversation_id ON message_processing_state(conversation_id);
CREATE INDEX IF NOT EXISTS idx_message_processing_state_message_id ON message_processing_state(message_id);
-- Socratic question history: tracks which questions have been asked in each conversation
CREATE TABLE IF NOT EXISTS question_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    question_id TEXT NOT NULL,
    question_text TEXT NOT NULL,
    asked_at INTEGER NOT NULL,

    -- Context when question was asked
    situation_type TEXT, -- e.g., "workplace_anxiety", "relationship_confusion"
    emotion_state TEXT, -- e.g., "negative", "very_negative", "positive"
    risk_level TEXT, -- e.g., "elevated", "clear"

    -- User's response
    user_response TEXT,
    response_length INTEGER, -- How much the user said in response

    -- Question metadata
    depth_level INTEGER, -- 1-5 progression
    socratic_approach TEXT, -- e.g., "identifying_stakeholders"
    category TEXT, -- e.g., "stakeholder", "consequence"

    -- Linking for follow-ups
    follow_up_question_id TEXT, -- ID of the follow-up question if one was asked

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

-- Create indexes for question history queries
CREATE INDEX IF NOT EXISTS idx_question_history_user_id ON question_history(user_id);
CREATE INDEX IF NOT EXISTS idx_question_history_conversation_id ON question_history(conversation_id);
CREATE INDEX IF NOT EXISTS idx_question_history_question_id ON question_history(question_id);

-- SCHEMA MIGRATIONS FOR EXISTING DATABASES
-- Note: SQLite doesn't support IF NOT EXISTS in ALTER TABLE
-- Column additions are handled in Go code with proper error handling

-- Commented out: these would need conditional logic in Go code
-- ALTER TABLE reflections ADD COLUMN contact_id TEXT;
-- ALTER TABLE reflections ADD COLUMN message_id TEXT;
-- ALTER TABLE reflections ADD COLUMN extracted_style TEXT;
-- ALTER TABLE reflections ADD COLUMN extracted_intention TEXT;
-- ALTER TABLE chat_messages ADD COLUMN metadata TEXT;

-- Create indexes for new foreign keys
CREATE INDEX IF NOT EXISTS idx_reflections_contact_id ON reflections(contact_id);
CREATE INDEX IF NOT EXISTS idx_reflections_message_id ON reflections(message_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_user_id ON chat_messages(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_id ON chat_messages(conversation_id);

