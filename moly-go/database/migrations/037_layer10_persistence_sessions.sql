-- FIX #73: Layer 10 Persistent Questioning Session State
-- Tracks user responses to persistent questions across messages
-- Enables progressive questioning: question 1, 2, 3, 4 on different messages
-- Each session tracks: question count, previous answers, acknowledgment of harm

CREATE TABLE IF NOT EXISTS persistence_sessions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    conversation_id VARCHAR(255) NOT NULL,

    -- Tracking state
    question_count INT NOT NULL DEFAULT 0,       -- How many probes asked (0-4)
    has_acknowledged_harm BOOLEAN NOT NULL DEFAULT FALSE, -- Did user acknowledge?

    -- Question history
    previous_answers TEXT,                        -- JSON array of answers to each probe
    questions_asked TEXT,                         -- JSON array of questions asked

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_question_asked_at TIMESTAMP,             -- When was last question asked?

    -- Indexes
    KEY idx_user_conversation (user_id, conversation_id),
    KEY idx_updated_at (updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Layer 10: Persistent questioning session state for multi-turn harm evaluation';

-- FIX #73: History table for auditing (why did we ask these questions?)
CREATE TABLE IF NOT EXISTS persistence_session_history (
    id INT AUTO_INCREMENT PRIMARY KEY,
    session_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(255) NOT NULL,

    -- What happened
    action VARCHAR(50) NOT NULL,                  -- 'question_asked', 'answer_recorded', 'harm_acknowledged'
    question_text TEXT,                           -- The question asked
    answer_text TEXT,                             -- User's answer
    question_number INT,                          -- Which probe (1-4)

    -- Context
    violation_detected VARCHAR(100),              -- What principle was violated? (harm_prevention, consent_and_respect, etc)
    confidence DECIMAL(3,2),                      -- How confident in violation (0.0-1.0)

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Indexes
    KEY idx_session_id (session_id),
    KEY idx_user_id (user_id),
    KEY idx_action (action),
    FOREIGN KEY (session_id) REFERENCES persistence_sessions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
COMMENT='Layer 10: Audit trail of persistent questioning decisions and responses';
