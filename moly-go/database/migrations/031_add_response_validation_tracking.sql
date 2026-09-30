-- PHASE 3-4: Response Validation Tracking
-- Audit trail for response validation and constraint checking
-- Date: Sept 30, 2026

-- Response validation audit trail (Phase 3)
CREATE TABLE IF NOT EXISTS response_validations (
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

CREATE INDEX IF NOT EXISTS idx_response_validations_conversation ON response_validations(conversation_id);
CREATE INDEX IF NOT EXISTS idx_response_validations_user ON response_validations(user_id);
CREATE INDEX IF NOT EXISTS idx_response_validations_violations ON response_validations(violations_detected);
CREATE INDEX IF NOT EXISTS idx_response_validations_passed ON response_validations(passed_validation);

-- Feature flag state tracking (Phase 1-4)
CREATE TABLE IF NOT EXISTS feature_flag_state (
    id TEXT PRIMARY KEY,
    flag_name TEXT NOT NULL UNIQUE,
    enabled BOOLEAN DEFAULT false,
    rollout_percentage INT DEFAULT 0,  -- 0-100
    rollout_start_time TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default feature flags
INSERT OR IGNORE INTO feature_flag_state (id, flag_name, enabled, rollout_percentage)
VALUES
    ('ff_phase1', 'UseExtractionLock', false, 0),
    ('ff_phase2', 'UseLayer5ConflictGate', false, 0),
    ('ff_phase3', 'UseConstrainedResponseGeneration', false, 0),
    ('ff_phase4', 'UseCleanSchema', false, 0);

-- Constraint tracking (Phase 3)
CREATE TABLE IF NOT EXISTS constraints_applied (
    id TEXT PRIMARY KEY,
    validation_id TEXT NOT NULL,
    constraint_type TEXT,  -- "user_characteristic", "communication_style", "contact_vulnerability"
    fact TEXT,
    dont_text TEXT,
    severity TEXT,
    source TEXT,
    violated BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (validation_id) REFERENCES response_validations(id)
);

CREATE INDEX IF NOT EXISTS idx_constraints_validation ON constraints_applied(validation_id);
CREATE INDEX IF NOT EXISTS idx_constraints_violated ON constraints_applied(violated);

-- Extraction lock tracking (Phase 1)
CREATE TABLE IF NOT EXISTS extraction_lock_events (
    id TEXT PRIMARY KEY,
    extraction_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    conversation_id TEXT,
    is_locked BOOLEAN,
    locked_at TIMESTAMP,
    lock_reason TEXT,
    time_to_lock_ms INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX IF NOT EXISTS idx_extraction_locks_user ON extraction_lock_events(user_id);
CREATE INDEX IF NOT EXISTS idx_extraction_locks_conversation ON extraction_lock_events(conversation_id);
CREATE INDEX IF NOT EXISTS idx_extraction_locks_locked ON extraction_lock_events(is_locked);

-- Conflict detection events (Phase 2)
CREATE TABLE IF NOT EXISTS conflict_events (
    id TEXT PRIMARY KEY,
    extraction_id TEXT,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    conflict_type TEXT,
    severity TEXT,
    description TEXT,
    entities_involved TEXT,  -- JSON: [entity1, entity2]
    question_asked BOOLEAN DEFAULT false,
    question_id TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,

    FOREIGN KEY (conversation_id) REFERENCES conversations(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_conflicts_conversation ON conflict_events(conversation_id);
CREATE INDEX IF NOT EXISTS idx_conflicts_user ON conflict_events(user_id);
CREATE INDEX IF NOT EXISTS idx_conflicts_type ON conflict_events(conflict_type);
CREATE INDEX IF NOT EXISTS idx_conflicts_resolved ON conflict_events(resolved_at);

-- Schema migration events (Phase 4)
CREATE TABLE IF NOT EXISTS schema_migration_events (
    id TEXT PRIMARY KEY,
    migration_name TEXT NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    status TEXT,  -- "pending", "in_progress", "success", "rollback"
    records_exported INT,
    records_imported INT,
    data_loss INT DEFAULT 0,
    error_message TEXT,
    rollback_reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_migrations_status ON schema_migration_events(status);
CREATE INDEX IF NOT EXISTS idx_migrations_completed ON schema_migration_events(completed_at);

-- Performance metrics snapshot (Phase 1-4)
CREATE TABLE IF NOT EXISTS performance_metrics (
    id TEXT PRIMARY KEY,
    metric_name TEXT NOT NULL,
    metric_type TEXT,  -- "latency", "count", "rate", "accuracy"
    value REAL,
    user_id TEXT,
    conversation_id TEXT,
    phase INT,  -- 1, 2, 3, or 4
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX IF NOT EXISTS idx_metrics_name ON performance_metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_metrics_phase ON performance_metrics(phase);
CREATE INDEX IF NOT EXISTS idx_metrics_recorded ON performance_metrics(recorded_at);

-- Cleanup: Archive old validations (keep 30 days)
-- Run daily via cron:
-- DELETE FROM response_validations WHERE validated_at < datetime('now', '-30 days');
-- DELETE FROM extraction_lock_events WHERE created_at < datetime('now', '-30 days');
-- DELETE FROM conflict_events WHERE detected_at < datetime('now', '-30 days');

-- End of migration
