-- Moly database schema: the single source of truth.
-- Applied once to an empty database. The schema version is stored in
-- PRAGMA user_version; bump SchemaVersion in database/db.go when this changes.
PRAGMA user_version = 1;


CREATE TABLE about_me (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL UNIQUE,
    communication_style TEXT,
    core_values TEXT, -- JSON array (was "values" - renamed for clarity)
    tone_preference TEXT, -- was preferred_tone
    preferences TEXT, -- JSON or text for additional preferences
    goals TEXT, -- JSON array of user goals
    patterns TEXT, -- JSON array of communication patterns
    notes TEXT, -- User's notes about themselves
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    version INTEGER DEFAULT 1, interests TEXT, needs TEXT, latest_intention TEXT, latest_intention_at INTEGER, user_instructions TEXT DEFAULT NULL,
    principles TEXT,
    characteristics TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE audit_log (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    details TEXT,
    timestamp INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE behavior_patterns (
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

CREATE TABLE chat_messages (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    role TEXT NOT NULL, -- "user" or "assistant"
    content TEXT NOT NULL,
    context_extracted TEXT, -- JSON
    contact_mention TEXT, -- JSON
    metadata TEXT, -- JSON for agent response metadata
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE clarification_answers (
    id TEXT PRIMARY KEY,
    clarification_question_id TEXT NOT NULL UNIQUE,
    user_answer TEXT NOT NULL,
    answered_at INTEGER NOT NULL,
    FOREIGN KEY (clarification_question_id) REFERENCES clarification_questions(id) ON DELETE CASCADE
);

CREATE TABLE "clarification_questions" (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    clarification_type TEXT NOT NULL,
    question_text TEXT NOT NULL,
    context_notes TEXT,
    options TEXT,
    priority INTEGER DEFAULT 2,
    status TEXT DEFAULT 'pending',
    linked_facts TEXT,
    created_at INTEGER NOT NULL,
    answered_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE TABLE clarification_responses (
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

CREATE TABLE contacts (
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
    updated_at INTEGER NOT NULL, confidence REAL DEFAULT 0.5, extraction_count INTEGER DEFAULT 1, last_mentioned_at INTEGER, pronouns TEXT, contact_role TEXT DEFAULT NULL, involved_intentions TEXT DEFAULT NULL, past_successes TEXT DEFAULT NULL, dependencies TEXT DEFAULT NULL,
    notes TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id, name)
);

CREATE TABLE "context_attributes" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    fact_type TEXT NOT NULL,
    fact_value TEXT NOT NULL,
    attributed_to TEXT NOT NULL,
    context TEXT DEFAULT 'general',
    confidence REAL DEFAULT 0.8,
    source TEXT DEFAULT 'explicit',
    evidence TEXT,
    version INTEGER DEFAULT 1,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE TABLE context_conflicts (
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

CREATE TABLE conversation_execution_state (
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

CREATE TABLE conversation_maturity (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    current_phase TEXT NOT NULL DEFAULT 'initial',
    maturity REAL NOT NULL DEFAULT 0.0,
    accomplishments_completed INTEGER NOT NULL DEFAULT 0,
    accomplishments_total INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    UNIQUE(user_id, conversation_id)
);

CREATE TABLE conversation_summaries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    -- Narrative summary (LLM-generated)
    arc TEXT,                           -- "User exploring romantic interests with multiple contacts..."

    -- Structured extraction (semantic tags for querying)
    key_topics TEXT,                    -- JSON array: ["communication", "boundaries", "interests"]
    user_patterns TEXT,                 -- JSON array: ["prefers_directness", "values_consent"]
    confirmed_choices TEXT,             -- JSON array: From Layer 3 clarifications
    open_questions TEXT,                -- JSON array: ["timing?", "comfort level?"]

    -- Update tracking
    message_count INTEGER DEFAULT 0,    -- Total messages in conversation
    messages_since_update INTEGER DEFAULT 0, -- How many new messages since last summary

    -- Metadata
    summary_version INTEGER DEFAULT 1,  -- Track summary iterations
    confidence REAL DEFAULT 0.0,        -- 0-1: how complete/accurate
    last_updated INTEGER NOT NULL,      -- Unix timestamp of last update

    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL, accumulated_entity_count INTEGER DEFAULT 0, accumulated_contact_count INTEGER DEFAULT 0, accumulated_values TEXT, accumulated_characteristics TEXT, conflicts_resolved INTEGER DEFAULT 0, clarity_progression TEXT,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    UNIQUE(user_id, conversation_id)
);

CREATE TABLE conversations (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT,
    description TEXT,
    purpose TEXT, -- JSON description of conversation purpose
    members TEXT, -- JSON array of ConversationMember objects
    settings TEXT, -- JSON object with mode, context, llmProvider
    notes TEXT, -- Additional metadata
    browser_session_id TEXT, -- Browser session ID when conversation was created (for detecting new sessions)
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE extracted_clarifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    interaction_id INTEGER, -- Link to interactions table
    gap_type TEXT NOT NULL, -- "communicationStyle", "coreValues", etc.
    question TEXT NOT NULL, -- The clarification question asked
    priority INTEGER DEFAULT 3, -- 1=critical, 2=important, 3=helpful
    asked_at INTEGER NOT NULL,
    response_received BOOLEAN DEFAULT 0,
    user_response TEXT, -- User's answer to the question
    responded_at INTEGER, -- When user answered
    was_helpful BOOLEAN, -- User feedback on usefulness
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (interaction_id) REFERENCES interactions(id) ON DELETE SET NULL
);

CREATE TABLE extraction_sentence_linking (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- The extracted fact
    context_attribute_id INTEGER NOT NULL,  -- FK to context_attributes

    -- What sentence it came from
    sentence_analysis_id INTEGER NOT NULL,  -- FK to sentence_analyses

    -- What subject it was attributed to (after resolution)
    subject_resolved_to TEXT NOT NULL,      -- "user", "Christine", "group_1", etc.
    subject_type TEXT,                      -- "user", "contact", "group"

    -- How certain are we about this attribution?
    attribution_confidence REAL DEFAULT 0.75,

    -- Metadata
    created_at INTEGER DEFAULT (strftime('%s', 'now')),

    FOREIGN KEY (context_attribute_id) REFERENCES context_attributes(id),
    FOREIGN KEY (sentence_analysis_id) REFERENCES sentence_analyses(id),

    UNIQUE(context_attribute_id)  -- Each fact linked to exactly one sentence
);

CREATE TABLE group_references (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    -- The group reference
    reference_pronoun TEXT NOT NULL,      -- "they", "them", "we", "us", "both", "all of us"
    reference_type TEXT NOT NULL,         -- "dual" (2), "plural" (3+), "collection"

    -- Members of the group
    member_ids TEXT NOT NULL,             -- JSON array: [1, 2, 3] or ["user_id", contact_id]
    member_names TEXT,                    -- JSON array: ["Christine", "Rigger"]

    -- Group properties
    is_user_in_group BOOLEAN DEFAULT 0,   -- Is the user part of this "we"?
    group_context TEXT,                   -- "couple", "trio", "group", "everyone involved"

    -- When this group reference was established
    message_id TEXT NOT NULL,
    established_at INTEGER DEFAULT (strftime('%s', 'now')),

    -- Confidence & evidence
    confidence REAL DEFAULT 0.75,         -- How certain are we about membership?
    evidence_text TEXT,                   -- Supporting text

    -- Scope & validity
    is_active BOOLEAN DEFAULT 1,          -- Current valid reference
    scope_end_message_id TEXT,            -- When this reference becomes invalid
    scope_end_seq INTEGER,

    created_at INTEGER DEFAULT (strftime('%s', 'now')),

    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE TABLE interactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    content TEXT NOT NULL,
    type TEXT, -- "user", "agent", "suggestion", "question"
    intention TEXT,
    intention_confidence REAL,
    emotional_tone TEXT,
    timestamp INTEGER NOT NULL,
    metadata TEXT, context_quality REAL, clarification_requested BOOLEAN DEFAULT 0, response_id TEXT, verdict_details TEXT, phase_results TEXT, maturity_score_at_message REAL, confidence_score_at_message REAL, maturity_phase_at_message TEXT, -- JSON for additional context
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE message_processing_state (
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

CREATE TABLE message_summaries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message_id TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    message_index INTEGER DEFAULT 0,
    role TEXT,
    message_length INTEGER,
    extracted_entities TEXT,
    entity_types TEXT,
    intention TEXT,
    key_phrases TEXT,
    tone TEXT,
    communication_style TEXT,
    confidence REAL,
    extraction_source TEXT,
    topic_shift BOOLEAN DEFAULT 0,
    has_clarification BOOLEAN DEFAULT 0,
    processed_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE TABLE messages (
  id TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL,
  role TEXT NOT NULL, -- 'user' or 'assistant'
  content TEXT NOT NULL,
  type TEXT,
  metadata TEXT,
  timestamp BIGINT NOT NULL,
  FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE TABLE pending_clarifications (
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

CREATE TABLE pending_input (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    type TEXT NOT NULL, -- "clarification" | "conflict" | "approval"
    subtype TEXT, -- "style_conflict" | "intention_conflict" | "reflection_approval" | etc
    question TEXT NOT NULL,
    context TEXT NOT NULL, -- JSON with full details (old_value, new_value, etc)
    created_at INTEGER NOT NULL,
    resolved_at INTEGER, -- NULL until user answers
    resolution TEXT, -- how user answered
    applied BOOLEAN DEFAULT 0, -- whether decision was applied to user model
    metadata TEXT, -- JSON for additional data

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE TABLE persistence_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    -- Tracking state
    question_count INTEGER NOT NULL DEFAULT 0,
    has_acknowledged_harm INTEGER NOT NULL DEFAULT 0,

    -- Question history
    previous_answers TEXT,
    questions_asked TEXT,

    -- Timestamps
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_question_asked_at DATETIME
);

CREATE TABLE pronoun_resolutions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    pronoun TEXT NOT NULL, -- "she", "he", "they", "it", etc.
    pronoun_type TEXT, -- "personal", "demonstrative", "relative", "possessive"
    antecedent_type TEXT, -- "name", "contact", "group", "concept", "unknown"
    antecedent_value TEXT NOT NULL, -- "Christine", "my boss", "both of them"
    antecedent_id INTEGER, -- FK to contacts if applicable
    message_id TEXT, -- Which message established this resolution
    sentence_position INTEGER, -- Position in that message
    confidence REAL DEFAULT 0.0, -- 0-1: how confident in this mapping
    evidence_text TEXT, -- Text that supports this resolution
    resolution_method TEXT, -- "linguistic_match", "llm_reasoning", "user_clarification", "context"
    scope_start_seq INTEGER, -- Message sequence number when valid from
    scope_end_seq INTEGER, -- Message sequence when invalid (NULL = ongoing)
    is_active BOOLEAN DEFAULT 1, -- Is this resolution still valid?
    created_at INTEGER NOT NULL,
    
    scope_start_message_id TEXT,
    scope_end_message_id TEXT,
    updated_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE TABLE question_history (
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

CREATE TABLE reflections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT, -- Link to which conversation this reflection is about
    contact_id TEXT, -- Link to which contact this reflection is about
    message_id TEXT, -- Link to which message triggered this reflection
    characteristics TEXT, -- JSON array
    interests TEXT, -- JSON array
    intentions TEXT, -- JSON array
    communication_preferences TEXT, -- User's communication preferences
    user_quotes TEXT, -- JSON array of quotes from user
    user_edits TEXT, -- JSON map of edits user made
    extracted_style TEXT, -- Communication style used in this context
    extracted_intention TEXT, -- User's intention in this exchange
    status TEXT DEFAULT 'pending_approval', -- "pending_approval", "approved", "rejected"
    created_at INTEGER NOT NULL,
    approved_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (message_id) REFERENCES chat_messages(id) ON DELETE SET NULL
);

CREATE TABLE response_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    context TEXT NOT NULL, -- "new_user_greeting", "first_message", "clarification_needed", "encouragement", etc.
    category TEXT NOT NULL, -- "greeting", "clarification", "suggestion", "warmup", etc.
    template_text TEXT NOT NULL, -- The actual response template (can include placeholders like {{name}}, {{intention}})
    priority INTEGER DEFAULT 1, -- Higher priority = preferred selection
    enabled BOOLEAN DEFAULT 1, -- Allow disabling without deleting
    version INTEGER DEFAULT 1, -- Track template versions
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    notes TEXT -- Admin notes about when/why to use this template
);

CREATE TABLE response_validations (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    message_id TEXT,
    response_text TEXT NOT NULL,
    constraints_applied INT DEFAULT 0,
    constraint_details TEXT,  -- JSON array of constraint details
    violations_detected INT DEFAULT 0,
    passed_validation BOOLEAN DEFAULT false,
    fallback_used BOOLEAN DEFAULT false,
    validation_time_ms INT,
    validated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE safety_incidents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    severity TEXT, -- "low", "medium", "high", "critical"
    detected_at INTEGER NOT NULL,
    content TEXT,
    detected_by TEXT, -- "heuristic", "llm", "manual"
    response_provided TEXT, verdict_reasoning TEXT, violated_principles TEXT, evidence_snippets TEXT, response_text TEXT, context_maturity REAL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE sentence_analyses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    -- Sentence structure
    sentence_text TEXT NOT NULL,           -- The actual sentence
    sentence_number INTEGER NOT NULL,     -- Which sentence in message (1-based)
    sentence_start_pos INTEGER,           -- Character position in message
    sentence_end_pos INTEGER,             -- Character position in message

    -- Subject-Verb-Object (SVO) Analysis
    subject TEXT,                         -- "I", "she", "Christine", "they"
    subject_type TEXT,                    -- "pronoun", "name", "group"
    subject_resolved_to TEXT,             -- After pronoun resolution: "user", "Christine", "group_1"

    verb TEXT,                            -- "am", "is", "like", "want"
    verb_type TEXT,                       -- "copula", "transitive", "intransitive", "phrasal"
    verb_negated BOOLEAN DEFAULT 0,       -- True if verb is negated (don't, won't, etc.)

    object TEXT,                          -- "dominant", "submissive", "communication"
    object_type TEXT,                     -- "adjective", "noun", "noun_phrase", "clause"

    -- Modifiers and additional context
    modifiers TEXT,                       -- JSON: {"adverbs": ["very"], "prepositional_phrases": ["in BDSM"]}
    negation BOOLEAN DEFAULT 0,           -- Is entire sentence negated?
    negation_scope TEXT,                  -- What's being negated

    -- Semantic relationships extracted from this sentence
    relationships TEXT,                   -- JSON: ["dominance:user->Christine", "preference:user->communication"]

    -- Extraction metadata
    confidence REAL DEFAULT 0.75,         -- SVO confidence (0.0-1.0)
    parsing_method TEXT,                  -- "regex", "llm", "hybrid"

    -- Tracking
    created_at INTEGER DEFAULT (strftime('%s', 'now')),

    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(user_id, message_id, sentence_number)
);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    device_id TEXT,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    last_used INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE structured_context (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,

    situation TEXT,                    -- "workplace conflict", "romantic uncertainty", etc.
    topic TEXT,                        -- Main topic of conversation

    people_involved TEXT,              -- JSON array of {name, relationship, role}
    goals TEXT,                        -- JSON array of strings
    "values" TEXT,                     -- JSON array of strings (quoted: reserved keyword)
    constraints TEXT,                  -- JSON array of strings

    past_attempts TEXT,                -- JSON array of strings ("what's been tried")
    current_blocker TEXT,              -- Current obstacle ("I don't know what to do next")

    emotional_tone TEXT,               -- "anxious", "frustrated", "hopeful", etc.

    remaining_gaps TEXT,               -- JSON array of open questions
    explored_topics TEXT,              -- JSON array of topics already covered

    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL, conversation_focus TEXT, focused_person TEXT,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    UNIQUE(user_id, conversation_id)
);

CREATE TABLE suggestion_choices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    suggestion_id TEXT NOT NULL,
    suggested_text TEXT NOT NULL,
    user_modification TEXT,
    chosen_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE system_context (
    user_id TEXT PRIMARY KEY,

    -- User's feedback about Moly's performance and behavior
    user_feedback TEXT DEFAULT NULL,  -- JSON array: ["too verbose", "helpful"]

    -- User's directives for how system should behave
    user_directives TEXT DEFAULT NULL,  -- JSON array: ["be concise", "ask more"]

    -- User's perception of system capabilities/limitations
    system_perceptions TEXT DEFAULT NULL,  -- JSON array: ["can do legal?", "lacks empathy"]

    -- User's preferred interaction style
    preferred_interaction_style TEXT DEFAULT NULL,  -- "direct", "socratic", "collaborative"

    -- Implicit ratings from conversation
    helpfulness_rating REAL DEFAULT 0.0,  -- 0-1
    clarity_rating REAL DEFAULT 0.0,      -- 0-1

    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    version INTEGER DEFAULT 1, system_instructions TEXT DEFAULT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE user_intentions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT,
    intention TEXT NOT NULL, -- The stated goal/intention
    confidence REAL, -- LLM confidence in extraction
    extracted_from TEXT, -- Message ID that revealed this intention
    confirmed BOOLEAN DEFAULT 0, -- Did user explicitly confirm?
    confirmed_at INTEGER, -- When user confirmed
    status TEXT DEFAULT 'active', -- "active", "achieved", "abandoned", "clarified"
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE user_interactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    conversation_id TEXT NOT NULL,
    user_message TEXT,
    suggestions_generated INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE users (
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

CREATE TABLE login_codes (
    code TEXT PRIMARY KEY,
    user_id TEXT,
    expires_at INTEGER NOT NULL,
    device_id TEXT,
    used BOOLEAN DEFAULT 0,
    created_at INTEGER NOT NULL
);

CREATE TABLE principle_violations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    principle_name TEXT NOT NULL,
    severity TEXT,
    details TEXT,
    detected_at INTEGER NOT NULL,
    resolved BOOLEAN DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE question_effectiveness (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    question_id TEXT,
    socratic_approach TEXT,
    question_text TEXT,
    user_response TEXT,
    reduced_ambiguity BOOLEAN DEFAULT 0,
    insight_gained BOOLEAN DEFAULT 0,
    depth_level_advanced BOOLEAN DEFAULT 0,
    principle_clarified BOOLEAN DEFAULT 0,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_about_me_latest_intention ON about_me(latest_intention);

CREATE INDEX idx_about_me_user_id ON about_me(user_id);

CREATE INDEX idx_about_me_user_instructions ON about_me(user_id)
WHERE user_instructions IS NOT NULL;

CREATE INDEX idx_behavior_patterns_user_id ON behavior_patterns(user_id);

CREATE INDEX idx_chat_messages_conversation_id ON chat_messages(conversation_id);

CREATE INDEX idx_chat_messages_created_at ON chat_messages(created_at DESC);

CREATE INDEX idx_chat_messages_user_id ON chat_messages(user_id);

CREATE INDEX idx_clarification_answers_question_id ON clarification_answers(clarification_question_id);

CREATE INDEX idx_clarification_questions_conversation_id ON clarification_questions(conversation_id);

CREATE INDEX idx_clarification_questions_status ON clarification_questions(status);

CREATE INDEX idx_clarification_questions_type ON clarification_questions(clarification_type);

CREATE INDEX idx_clarification_questions_user_id ON clarification_questions(user_id);

CREATE INDEX idx_clarification_responses_question_id ON clarification_responses(question_id);

CREATE INDEX idx_clarification_responses_user_id ON clarification_responses(user_id);

CREATE INDEX idx_clarifications_conversation ON extracted_clarifications(conversation_id);

CREATE INDEX idx_clarifications_gap_type ON extracted_clarifications(gap_type);

CREATE INDEX idx_clarifications_responded ON extracted_clarifications(response_received);

CREATE INDEX idx_contacts_contact_role ON contacts(user_id, contact_role)
WHERE contact_role IS NOT NULL;

CREATE INDEX idx_contacts_involved_intentions ON contacts(user_id)
WHERE involved_intentions IS NOT NULL;

CREATE INDEX idx_contacts_name ON contacts(name);

CREATE INDEX idx_contacts_user ON contacts(user_id);

CREATE INDEX idx_contacts_user_id ON contacts(user_id);

CREATE INDEX idx_contacts_user_mentioned ON contacts(user_id, last_mentioned_at);

CREATE INDEX idx_context_attributes_source ON context_attributes(source);

CREATE INDEX idx_context_attributes_user_conversation ON context_attributes(user_id, conversation_id);

CREATE INDEX idx_context_conflicts_conversation
    ON context_conflicts(user_id, conversation_id, status);

CREATE INDEX idx_context_conflicts_conversation_id ON context_conflicts(conversation_id);

CREATE INDEX idx_context_conflicts_status ON context_conflicts(status);

CREATE INDEX idx_context_conflicts_type ON context_conflicts(conflict_type);

CREATE INDEX idx_context_conflicts_unresolved
    ON context_conflicts(user_id, status)
    WHERE status = 'unresolved';

CREATE INDEX idx_context_conflicts_user_id ON context_conflicts(user_id);

CREATE INDEX idx_conversation_execution_state_phase
  ON conversation_execution_state(user_id, phase);

CREATE INDEX idx_conversation_execution_state_updated
  ON conversation_execution_state(user_id, updated_at);

CREATE INDEX idx_conversation_execution_state_user_conv
    ON conversation_execution_state(user_id, conversation_id);

CREATE INDEX idx_conversation_execution_state_user_conversation
  ON conversation_execution_state(user_id, conversation_id);

CREATE INDEX idx_conversation_execution_state_user_phase
    ON conversation_execution_state(user_id, phase);

CREATE INDEX idx_conversation_maturity_conversation ON conversation_maturity(conversation_id);

CREATE INDEX idx_conversation_maturity_user ON conversation_maturity(user_id);

CREATE INDEX idx_conversation_summaries_accumulated_entity_count
  ON conversation_summaries(user_id, conversation_id, accumulated_entity_count);

CREATE INDEX idx_conversation_summaries_clarity_progression
  ON conversation_summaries(user_id, conversation_id);

CREATE INDEX idx_conversation_summaries_last_updated
    ON conversation_summaries(last_updated);

CREATE INDEX idx_conversation_summaries_updated_at
    ON conversation_summaries(updated_at);

CREATE INDEX idx_conversation_summaries_user_conv
    ON conversation_summaries(user_id, conversation_id);

CREATE INDEX idx_conversation_summaries_user_conversation
  ON conversation_summaries(user_id, conversation_id);

CREATE INDEX idx_conversations_updated_at ON conversations(updated_at DESC);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);

CREATE INDEX idx_execution_state_conversation_id ON conversation_execution_state(conversation_id);

CREATE INDEX idx_execution_state_user_id ON conversation_execution_state(user_id);

CREATE INDEX idx_extraction_sentence_linking_fact
    ON extraction_sentence_linking(context_attribute_id);

CREATE INDEX idx_extraction_sentence_linking_sentence
    ON extraction_sentence_linking(sentence_analysis_id);

CREATE INDEX idx_extraction_sentence_linking_subject
    ON extraction_sentence_linking(subject_resolved_to);

CREATE INDEX idx_group_references_active
    ON group_references(user_id, is_active);

CREATE INDEX idx_group_references_pronoun
    ON group_references(user_id, reference_pronoun, is_active);

CREATE INDEX idx_group_references_user_conv
    ON group_references(user_id, conversation_id);

CREATE INDEX idx_interactions_clarification ON interactions(clarification_requested);

CREATE INDEX idx_interactions_context_quality ON interactions(context_quality);

CREATE INDEX idx_interactions_conversation
ON interactions(conversation_id);

CREATE INDEX idx_interactions_conversation_id ON interactions(conversation_id);

CREATE INDEX idx_interactions_response_id ON interactions(response_id);

CREATE INDEX idx_interactions_timestamp ON interactions(timestamp DESC);

CREATE INDEX idx_interactions_user_id ON interactions(user_id);

CREATE INDEX idx_message_processing_state_conversation_id ON message_processing_state(conversation_id);

CREATE INDEX idx_message_processing_state_lookup ON message_processing_state(user_id, conversation_id);

CREATE INDEX idx_message_processing_state_message_id ON message_processing_state(message_id);

CREATE INDEX idx_message_processing_state_user_id ON message_processing_state(user_id);

CREATE INDEX idx_message_summaries_conversation_id ON message_summaries(conversation_id);

CREATE INDEX idx_message_summaries_created_at ON message_summaries(created_at);

CREATE INDEX idx_message_summaries_message_id ON message_summaries(message_id);

CREATE INDEX idx_message_summaries_message_index ON message_summaries(message_index);

CREATE INDEX idx_message_summaries_user_id ON message_summaries(user_id);

CREATE INDEX idx_pending_clarifications_conversation_id ON pending_clarifications(conversation_id);

CREATE INDEX idx_pending_clarifications_expires_at ON pending_clarifications(expires_at);

CREATE INDEX idx_pending_clarifications_status ON pending_clarifications(status);

CREATE INDEX idx_pending_clarifications_user_id ON pending_clarifications(user_id);

CREATE INDEX idx_pending_fact ON pending_clarifications(fact_id);

CREATE INDEX idx_pending_input_conversation_type ON pending_input(conversation_id, type);

CREATE INDEX idx_pending_input_user_pending ON pending_input(user_id, resolved_at);

CREATE INDEX idx_pending_status ON pending_clarifications(status);

CREATE INDEX idx_pending_user ON pending_clarifications(user_id);

CREATE INDEX idx_persistence_updated_at ON persistence_sessions(updated_at);

CREATE INDEX idx_persistence_user_conversation ON persistence_sessions(user_id, conversation_id);

CREATE INDEX idx_pronoun_resolutions_active
    ON pronoun_resolutions(is_active, created_at DESC);

CREATE INDEX idx_pronoun_resolutions_antecedent
    ON pronoun_resolutions(user_id, antecedent_value);

CREATE INDEX idx_pronoun_resolutions_conversation
    ON pronoun_resolutions(conversation_id, created_at DESC);

CREATE INDEX idx_pronoun_resolutions_pronoun
    ON pronoun_resolutions(user_id, pronoun, is_active);

CREATE INDEX idx_pronoun_resolutions_scope
    ON pronoun_resolutions(user_id, scope_start_seq, scope_end_seq);

CREATE INDEX idx_pronoun_resolutions_user_conv
    ON pronoun_resolutions(user_id, conversation_id);

CREATE INDEX idx_pronoun_resolutions_user_pronoun
    ON pronoun_resolutions(user_id, pronoun, is_active);

CREATE INDEX idx_question_history_conversation_id ON question_history(conversation_id);

CREATE INDEX idx_question_history_question_id ON question_history(question_id);

CREATE INDEX idx_question_history_user_id ON question_history(user_id);

CREATE INDEX idx_reflections_contact_id ON reflections(contact_id);

CREATE INDEX idx_reflections_message_id ON reflections(message_id);

CREATE INDEX idx_reflections_user_id ON reflections(user_id);

CREATE INDEX idx_response_templates_context_category
    ON response_templates(context, category, priority DESC, enabled);

CREATE INDEX idx_response_validations_conversation ON response_validations(conversation_id);

CREATE INDEX idx_response_validations_passed ON response_validations(passed_validation);

CREATE INDEX idx_response_validations_user ON response_validations(user_id);

CREATE INDEX idx_response_validations_violations ON response_validations(violations_detected);

CREATE INDEX idx_safety_incidents_detected_at
ON safety_incidents(detected_at DESC);

CREATE INDEX idx_safety_incidents_severity ON safety_incidents(severity);

CREATE INDEX idx_safety_incidents_user_id ON safety_incidents(user_id);

CREATE INDEX idx_safety_incidents_verdict ON safety_incidents(verdict_reasoning);

CREATE INDEX idx_sentence_analyses_message
    ON sentence_analyses(user_id, message_id);

CREATE INDEX idx_sentence_analyses_subject
    ON sentence_analyses(user_id, subject);

CREATE INDEX idx_sentence_analyses_user_conv
    ON sentence_analyses(user_id, conversation_id);

CREATE INDEX idx_sentence_analyses_verb
    ON sentence_analyses(user_id, verb);

CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);

CREATE INDEX idx_structured_context_focus ON structured_context(conversation_focus);

CREATE INDEX idx_structured_context_updated_at
    ON structured_context(updated_at);

CREATE INDEX idx_structured_context_user_conv
    ON structured_context(user_id, conversation_id);

CREATE INDEX idx_suggestion_choices_user_id ON suggestion_choices(user_id);

CREATE INDEX idx_system_context_system_instructions ON system_context(user_id)
WHERE system_instructions IS NOT NULL;

CREATE INDEX idx_system_context_user_id ON system_context(user_id);

CREATE INDEX idx_user_intentions_status ON user_intentions(status);

CREATE INDEX idx_user_intentions_user_id ON user_intentions(user_id);

CREATE INDEX idx_users_created_at ON users(created_at DESC);
