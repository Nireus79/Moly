# MOLY: IMPLEMENTATION PLAN (Philosophy-First, Clean Schema)

**Goal**: Implement the 11-layer clarification philosophy with clean architecture  
**Status**: Ready to implement  
**Date**: Sept 30, 2026  
**Effort**: ~8 weeks (1 person) or ~5 weeks (2 people)

---

## OVERVIEW: PHILOSOPHY INTACT, IMPLEMENTATION CLEAN

### Moly's Core (Unchanging)
- ✅ 11-layer orchestration (extract → clarify → deepen → respond)
- ✅ Clarification before blocking philosophy
- ✅ Assume good intent by default
- ✅ Context maturity gating
- ✅ Principle-based reasoning (no hardcoded keywords)
- ✅ Socratic deepening once clear
- ✅ Privacy-first (local processing)

### Implementation Improvements (4 Phases)
1. **Phase 1 - Extraction Lock**: Extract once, lock it, never re-parse
2. **Phase 2 - Conflict Channeling**: Every detected conflict → clarification question
3. **Phase 3 - Constrained Generation**: Validate facts during generation, not after
4. **Phase 4 - Clean Schema**: Database redesigned for new architecture (not migration)

---

## PHASE 1: EXTRACTION LOCK (2 WEEKS)

### Purpose
Stop multi-pass parsing that corrupts subject attribution. Extract once, lock it, all downstream use same extraction.

### Strategy
- **Feature flag**: `useExtractionLock` (default false)
  - Start at 10% users, expand to 100%
  - Easy rollback if issues
  
- **Performance**: No caching needed (extraction is single-pass, already fast)
- **Benchmarking**: Track extraction time before/after lock enforcement
- **Monitoring**: Track accuracy on locked vs current extraction

### What Changes

#### File 1: `models/extraction_artifact.go`
**Current State**: ExtractionArtifact can be modified after creation

**Changes**:
```go
type ExtractionArtifact struct {
    ID                string
    Entities          []ExtractedEntity
    ConflictDetected  []ConflictResult
    PrincipleMetadata map[string]PrincipleInfo  // NEW: Principle info per entity
    
    // NEW: Immutability enforcement
    IsLocked          bool
    LockedAt          time.Time
    LockReason        string
}

// NEW: Lock the extraction (permanent)
func (ea *ExtractionArtifact) Lock(reason string) error {
    if ea.IsLocked {
        return fmt.Errorf("already locked: %s", ea.LockReason)
    }
    ea.IsLocked = true
    ea.LockedAt = time.Now()
    ea.LockReason = reason
    return nil
}

// NEW: Prevent modifications after lock
func (ea *ExtractionArtifact) AddEntity(entity ExtractedEntity) error {
    if ea.IsLocked {
        return fmt.Errorf("cannot add entity to locked extraction")
    }
    ea.Entities = append(ea.Entities, entity)
    return nil
}
```

**Effort**: 1 day | **Tests**: 3 unit tests

#### File 2: `agents/intent_detector.go`
**Current State**: Extracts entities, returns them (can be re-used)

**Changes**:
```go
// Add principle extraction to LLM prompt
func (id *LLMIntentDetector) extractEntitiesWithPrinciples(ctx context.Context, message string) ([]ExtractedEntity, error) {
    prompt := fmt.Sprintf(`Extract entities and their principle metadata:
    
User message: "%s"

For each extracted entity, provide:
1. Type (User, Contact, Property, Value, Goal, Concern)
2. Value (exact text or summary)
3. Subject (who is this about? "user", "contact_name", etc)
4. Principles involved (which of these apply: Harm Prevention, User Autonomy, Consent & Respect, Stakeholder Consideration, Transparency, Growth & Learning)
5. Antonyms/Contradictions (what would be opposite of this value?)

Return as JSON array with fields: type, value, subject, principles[], antonyms[], confidence.`, message)

    // Call LLM with this enhanced prompt
    resp, err := id.llmClient.Call(ctx, prompt)
    // Parse and return with principle metadata
}

// NEW: Lock extraction after creation
func (id *LLMIntentDetector) ExtractAndLock(ctx context.Context, message string) (*ExtractionArtifact, error) {
    artifact := &ExtractionArtifact{
        ID: generateID(),
    }
    
    // Extract with principle metadata
    entities, err := id.extractEntitiesWithPrinciples(ctx, message)
    if err != nil {
        return nil, err
    }
    
    // Detect conflicts
    conflicts, err := id.detectConflicts(entities)
    if err != nil {
        log.Printf("[IntentDetector] Warning: Conflict detection failed: %v", err)
        // Continue - conflicts optional
    }
    
    artifact.Entities = entities
    artifact.ConflictDetected = conflicts
    
    // LOCK IT (cannot be modified after this)
    if err := artifact.Lock("extraction complete"); err != nil {
        return nil, fmt.Errorf("failed to lock extraction: %v", err)
    }
    
    return artifact, nil
}
```

**Effort**: 2 days | **Tests**: 10 unit tests (principle extraction)

#### File 3: `tools/extraction_store.go` (NEW)
**Purpose**: Repository for locked extractions, prevents unlocking

**Implementation**:
```go
package tools

type ExtractionRepository struct {
    db *database.Database
}

// GetLocked retrieves and verifies extraction is locked
func (er *ExtractionRepository) GetLocked(extractionID string) (*ExtractionArtifact, error) {
    artifact, err := er.db.GetExtractionArtifact(extractionID)
    if err != nil {
        return nil, err
    }
    if !artifact.IsLocked {
        return nil, fmt.Errorf("extraction %s is not locked", extractionID)
    }
    return artifact, nil
}

// SaveLocked saves a locked extraction (cannot be modified later)
func (er *ExtractionRepository) SaveLocked(artifact *ExtractionArtifact) error {
    if !artifact.IsLocked {
        return fmt.Errorf("can only save locked extractions")
    }
    return er.db.SaveExtractionArtifact(artifact)
}

// TryModify returns error if extraction is locked
func (er *ExtractionRepository) TryModify(extractionID string) error {
    artifact, err := er.db.GetExtractionArtifact(extractionID)
    if err != nil {
        return err
    }
    if artifact.IsLocked {
        return fmt.Errorf("extraction is locked and cannot be modified")
    }
    return nil
}
```

**Effort**: 1 day | **Tests**: 5 unit tests

#### File 4: `database/migrations/026_add_extraction_principles.sql`
**Changes**:
```sql
-- Store principle metadata
CREATE TABLE extraction_principle_metadata (
    id UUID PRIMARY KEY,
    extraction_id UUID NOT NULL REFERENCES extractions(id),
    entity_value TEXT,
    principles TEXT[],
    antonyms TEXT[],
    antonym_source TEXT DEFAULT 'llm_reasoning',
    bidirectional BOOLEAN DEFAULT false,
    confidence FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Track extraction lock status
ALTER TABLE extractions ADD COLUMN (
    is_locked BOOLEAN DEFAULT false,
    locked_at TIMESTAMP,
    lock_reason TEXT
);

CREATE INDEX idx_extractions_locked ON extractions(is_locked);
```

**Effort**: 0.5 day

#### File 5: `main.go` (Lines 687-800: Remove re-parsing)
**Current State**: 
- Line 696: Call ExtractionPhase
- Line 715-850: Multiple fallback paths if extraction fails
- Line 1564+: Re-parse with LinguisticParser if needed

**Changes**:
```go
// Remove all re-parsing paths
// OLD CODE (~80 lines): "if extraction failed, try LinguisticParser fallback"
// DELETE THESE BLOCKS

// NEW: Use locked extraction only
extractionPhase := agents.NewExtractionPhase(intentDetector, extractionStore, conflictDetector, db)
epOutput, err := extractionPhase.Run(ctx)
if err != nil {
    log.Printf("[MessageProcessor] FATAL: Extraction failed and cannot proceed (no fallback)")
    schema.RespondError(w, http.StatusInternalServerError, "Extraction failed")
    return
}

// Verify extraction is locked
lockedArtifact, err := extractionStore.GetLocked(epOutput.Extraction.ID)
if err != nil {
    log.Printf("[MessageProcessor] FATAL: Extraction not locked")
    schema.RespondError(w, http.StatusInternalServerError, "Extraction not locked")
    return
}

// Use locked artifact for all downstream processing
```

**Effort**: 2 days (searching and removing re-parsing paths) | **Tests**: Integration tests (no regressions)

#### File 6: `database/clarification_capture.go` (Remove multi-pass logic)
**Current State**: 
- Lines 150-250: LinguisticParser fallback
- Lines 300-400: Subject attribution retry logic

**Changes**:
```go
// REMOVE: All LinguisticParser usage
// REMOVE: All fallback re-parsing logic

// NEW: Use locked extraction only
func (cc *ClarificationCapture) ProcessWithLockedExtraction(
    userID string,
    extractedArtifact *ExtractionArtifact,
) (*ClarificationResult, error) {
    // Use extracted entities directly (never re-parse)
    entities := extractedArtifact.Entities
    
    // Subject attribution from extraction (trusted)
    for _, entity := range entities {
        if entity.Subject == "" {
            // If subject unclear in extraction, this is a gap
            // Ask clarification: "Who is this about?"
            return &ClarificationResult{
                NeedsQuestion: true,
                Question: "You mentioned " + entity.Value + ". Who is this about?",
            }, nil
        }
    }
    
    // No re-parsing, just use what was extracted
    return &ClarificationResult{NeedsQuestion: false}, nil
}
```

**Effort**: 1.5 days | **Tests**: 5 integration tests

### Success Criteria (Phase 1 Go/No-Go)
- ✅ Zero multi-pass parsing (verify in logs)
- ✅ Extraction accuracy maintained (compare before/after on 50 test messages)
- ✅ Subject attribution correct (no corruption)
- ✅ No regressions in existing functionality
- ✅ All 50+ tests passing
- ✅ Extraction latency maintained or improved
- ✅ 100% of extractions locked (verify in database)

### Performance Benchmarks (Phase 1)
```
Metric                    Baseline    Target      Threshold
Extraction time           <200ms      <200ms      < 250ms (110% baseline)
Subject attribution acc.  95%         95%+        < 93% (rollback)
Multi-pass calls          7/msg       0/msg       > 2/msg (rollback)
```

### Deployment Strategy (Phase 1)
- **100% rollout immediately** (low risk, no behavior change)
- Feature flag allows instant rollback if needed
- No staged rollout needed - extraction lock is transparent

### Monitoring (Phase 1)
- Track extraction_lock_enabled flag in logs
- Monitor extraction time distribution (p50, p95, p99)
- Alert if extraction time increases > 20%
- Monitor subject attribution accuracy on locked extractions

### Rollback Plan
- Disable feature flag: `useExtractionLock = false`
- Restore old re-parsing paths in main.go
- Effort: < 1 hour (feature flag flip)

---

## PHASE 2: LAYER 5 CONFLICT CHANNELING (3 WEEKS)

### Purpose
Every detected conflict becomes a clarification question in Layer 5.

### Architecture

#### Understanding Layer 5's Role
**Current flow**:
- Extraction: detects conflicts ✓
- Logging: logs conflicts ✓
- Question generation: maybe asks about them ❌

**Desired flow**:
- Extraction: detects conflicts ✓
- Layer 5: receives conflicts, generates questions ✓
- ConversationAgent: asks questions ✓

### What Changes

#### File 1: `agents/layer5_conflict_handler.go` (NEW)
**Purpose**: Dedicated Layer 5 handler for conflict-based clarifications

```go
package agents

type Layer5ConflictHandler struct {
    llmClient         LLMProvider
    db                *database.Database
    questionHistory   *ClarificationHistory
}

// ProcessConflicts examines detected conflicts and generates clarification questions
func (lch *Layer5ConflictHandler) ProcessConflicts(
    ctx context.Context,
    extractedConflicts []ConflictResult,
    userID string,
    conversationID string,
) (ClarificationQuestion, bool) {
    
    if len(extractedConflicts) == 0 {
        return ClarificationQuestion{}, false // No conflicts
    }
    
    // Process highest-severity conflict first
    conflict := extractedConflicts[0]
    
    // Check if we already asked about this conflict recently
    if lch.questionHistory.WasRecentlyAsked(userID, conflict.ID) {
        log.Printf("[Layer5] Conflict %s already asked recently, skipping", conflict.ID)
        return ClarificationQuestion{}, false
    }
    
    // Generate clarification question for this conflict
    question := lch.generateConflictQuestion(ctx, conflict)
    
    // Save to database
    if err := lch.saveQuestion(userID, conversationID, question); err != nil {
        log.Printf("[Layer5] Warning: Failed to save question: %v", err)
    }
    
    return question, true
}

// generateConflictQuestion creates a Socratic question about the conflict
func (lch *Layer5ConflictHandler) generateConflictQuestion(
    ctx context.Context,
    conflict ConflictResult,
) ClarificationQuestion {
    
    prompt := fmt.Sprintf(`User said previously: "%s"
User just said: "%s"

These seem contradictory. Ask a clarifying question (Socratic, not accusatory) to understand what changed.
Keep it conversational, not technical. Example: "You mentioned X before, but now you're saying Y. What changed?"

Format: Just the question, no explanation.`, conflict.PreviousValue, conflict.CurrentValue)
    
    resp, _ := lch.llmClient.Call(ctx, prompt)
    
    return ClarificationQuestion{
        ID:         fmt.Sprintf("layer5_conflict_%d", time.Now().UnixNano()),
        Type:       "conflict_resolution",
        Question:   resp,
        Priority:   2, // medium-high
        Reason:     fmt.Sprintf("Conflict detected: %s vs %s", conflict.PreviousValue, conflict.CurrentValue),
        PrincipleRelated: conflict.RelatedPrinciples,
    }
}

// saveQuestion persists the question
func (lch *Layer5ConflictHandler) saveQuestion(
    userID string,
    conversationID string,
    question ClarificationQuestion,
) error {
    clariRepo := lch.db.GetClarificationQuestionRepository()
    dbQuestion := &database.ClarificationQuestion{
        ID:                question.ID,
        UserID:            userID,
        ConversationID:    conversationID,
        ClarificationType: "conflict_resolution",
        QuestionText:      question.Question,
        ContextNotes:      question.Reason,
        Priority:          question.Priority,
        Status:            "pending",
        CreatedAt:         time.Now().Unix(),
    }
    return clariRepo.SaveQuestion(dbQuestion)
}
```

**Effort**: 2 days | **Tests**: 10 unit tests

#### File 2: `agents/clarification_history.go` (NEW)
**Purpose**: Track which clarifications already asked (deduplication)

```go
package agents

type ClarificationHistory struct {
    db *database.Database
}

// WasRecentlyAsked checks if similar clarification was asked in last N messages
func (ch *ClarificationHistory) WasRecentlyAsked(userID string, conflictID string) bool {
    repo := ch.db.GetClarificationQuestionRepository()
    
    // Get last 10 questions for this user
    recent, err := repo.GetRecentQuestions(userID, 10)
    if err != nil {
        return false // Allow retry if lookup fails
    }
    
    // Check if this conflictID was asked before
    for _, q := range recent {
        if strings.Contains(q.ContextNotes, conflictID) {
            return true
        }
    }
    
    return false
}

// IsSimilarQuestion checks if question is semantically similar to recent ones
func (ch *ClarificationHistory) IsSimilarQuestion(userID string, newQuestion string) bool {
    repo := ch.db.GetClarificationQuestionRepository()
    recent, err := repo.GetRecentQuestions(userID, 5)
    if err != nil {
        return false
    }
    
    // LLM-based similarity check (not just string match)
    for _, q := range recent {
        if ch.areSimilar(newQuestion, q.QuestionText) {
            return true
        }
    }
    return false
}

// areSimilar uses semantic similarity (LLM-based)
func (ch *ClarificationHistory) areSimilar(q1, q2 string) bool {
    // Implementation: embed both questions, check cosine similarity
    // For now: simple heuristic
    return strings.EqualFold(q1, q2) || // Exact same
           (len(q1) > 20 && len(q2) > 20 && 
            levenshteinSimilarity(q1, q2) > 0.8) // 80%+ similar
}
```

**Effort**: 1.5 days | **Tests**: 8 unit tests

#### File 3: `agents/conversation_agent.go` (Lines 432-700: Add Layer 5 gate)
**Current State**: Has clarity gate, gap gate, principle gate, but NOT explicit conflict gate

**Changes**:
```go
// Add AFTER clarity gate, BEFORE gap gate (in Run method)

// ⭐ [Layer 5] CONFLICT DETECTION & RESOLUTION
// Highest priority after clarity: resolve fact inconsistencies
if len(epOutput.Conflicts) > 0 && ca.layer5Handler != nil {
    log.Printf("[ConversationAgent] [Layer 5] %d conflicts detected", len(epOutput.Conflicts))
    
    conflictQ, hasQuestion := ca.layer5Handler.ProcessConflicts(
        ctx,
        epOutput.Conflicts,
        ctx.AboutMe.UserID,
        ctx.ConversationID,
    )
    
    if hasQuestion {
        response.Response = conflictQ.Question
        response.Metadata["layer5Conflict"] = true
        response.Metadata["conflictResolution"] = conflictQ.Reason
        response.Metadata["gate"] = "conflict_resolution"
        
        log.Printf("[ConversationAgent] [✓] Layer 5: Conflict question: %s", conflictQ.Question)
        response.ProcessingTimeMs = int(time.Since(startTime).Milliseconds())
        return response, nil
    }
}
```

**Effort**: 1 day | **Tests**: 5 integration tests

#### File 4: `models/conflict_result.go` (Enhance)
**Current State**: Has Type, Description, but missing relationship to clarification

**Changes**:
```go
type ConflictResult struct {
    ID                 string
    Type               string   // "characteristic_conflict", "value_conflict", etc
    PreviousValue      string   // What user said before
    CurrentValue       string   // What user says now
    Description        string   // Plain English explanation
    Severity           string   // "low", "medium", "high"
    RelatedPrinciples  []string // Which principles involved
    DetectedAt         time.Time
    
    // NEW: Link to clarification question
    ClarificationID    string   // Links to the question we asked
    Resolved           bool     // Did user answer the question?
    Resolution         string   // User's explanation
}
```

**Effort**: 0.5 day

#### File 5: `database/migrations/027_add_layer5_tracking.sql`
**Changes**:
```sql
-- Track Layer 5 conflict questions
ALTER TABLE clarification_questions ADD COLUMN (
    conflict_id TEXT,
    conflict_previous_value TEXT,
    conflict_current_value TEXT,
    layer TEXT DEFAULT '4' -- Which layer asked this
);

-- Track when conflicts are resolved
CREATE TABLE conflict_resolutions (
    id UUID PRIMARY KEY,
    conflict_id TEXT,
    user_id TEXT NOT NULL,
    conversation_id TEXT,
    user_explanation TEXT,
    resolved_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_clarifications_conflict ON clarification_questions(conflict_id);
CREATE INDEX idx_clarifications_layer ON clarification_questions(layer);
```

**Effort**: 0.5 day

### Integration: Wire Layer 5 Into ConversationAgent
```go
// In NewConversationAgent or SetDatabase:
if ca.db != nil && ca.llmClient != nil {
    ca.layer5Handler = agents.NewLayer5ConflictHandler(ca.llmClient, ca.db)
    ca.questionHistory = agents.NewClarificationHistory(ca.db)
    log.Printf("[ConversationAgent] Layer 5 conflict handler initialized")
}
```

**Effort**: 0.5 day

### Success Criteria (Phase 2 Go/No-Go)
- ✅ All detected conflicts generate clarification questions (100%)
- ✅ Questions are Socratic, not accusatory (manual review: 20 questions)
- ✅ Deduplication prevents repeated questions (< 1% repeats)
- ✅ Questions appear in Layer 5 position (before gap questions)
- ✅ Integration tests show conflicts → questions flow
- ✅ User engagement metrics stable (no increase in abandonment)
- ✅ No regressions in other layers

### Performance Benchmarks (Phase 2)
```
Metric                      Baseline    Target      Threshold
Conflict detection time     <100ms      <150ms      < 200ms
Question generation time    <500ms      <600ms      < 750ms
Dedup check time            N/A         <50ms       < 100ms
Response latency (p95)      2-3s        2-3s        < 3.5s (rollback)
```

### Deployment Strategy (Phase 2)
- **Week 1**: 10% users (48 hour monitoring)
- **Week 2**: 50% users (1 week monitoring)
- **Week 3**: 100% users
- Feature flag: `useLayer5ConflictGate` (default false)
- Parallel: Keep old layer system as fallback

### Monitoring (Phase 2)
- Track conflict_gate_triggered count
- Track question_generated_from_conflict rate
- Track deduplication_prevented count
- Monitor user response time to conflict questions
- Alert if: question abandonment > 20% or latency > 3.5s

### Rollback Plan
- Disable feature flag: `useLayer5ConflictGate = false`
- Route to old layer system
- Effort: < 1 hour (feature flag flip)

---

## PHASE 3: CONSTRAINED RESPONSE GENERATION (2 WEEKS)

### Purpose
Generate responses with constraints about user facts, validate BEFORE returning.

### Architecture

#### Understanding the Problem
**Current flow**:
1. Generate response (ignoring facts)
2. Validate response against facts
3. If violated: Too late, might return bad response

**New flow**:
1. Build constraints from facts ("User is dominant, don't suggest submission")
2. Pass constraints to LLM in system prompt
3. LLM generates WITH constraints in mind
4. Validate BEFORE returning
5. If violated: Ask clarification instead

### What Changes

#### File 1: `tools/constrained_response_generator.go` (NEW)
**Purpose**: Generate responses that respect user facts

```go
package tools

type ConstrainedResponseGenerator struct {
    llmClient         LLMProvider
    basedResponseGen  *ResponseGenerator
    db                *database.Database
    validator         *ResponseValidator
}

// Generate creates response constrained by user facts
func (crg *ConstrainedResponseGenerator) Generate(
    ctx context.Context,
    userMessage string,
    userProfile *models.AboutMe,
    contacts []models.Contact,
    extractedContext *models.ExtractedContext,
    extractionArtifact *ExtractionArtifact,
) (*models.ConversationResponse, error) {
    
    // STEP 1: Build constraints from facts
    constraints := crg.buildConstraints(userProfile, contacts, extractedContext, extractionArtifact)
    log.Printf("[ConstrainedGen] Built %d constraints", len(constraints))
    
    // STEP 2: Build LLM prompt with constraints visible
    systemPrompt := crg.buildConstrainedSystemPrompt(constraints)
    
    // STEP 3: Generate response WITH constraints in view
    response, err := crg.basedResponseGen.GenerateWithSystemPrompt(
        ctx,
        userMessage,
        userProfile,
        contacts,
        systemPrompt, // NEW: pass enhanced system prompt
    )
    if err != nil {
        return nil, err
    }
    
    // STEP 4: Validate response against constraints
    violations := crg.validateResponse(response.Response, constraints)
    if violations > 0 {
        log.Printf("[ConstrainedGen] ⚠ Response violates %d constraints, asking clarification instead", violations)
        
        // Don't return bad response - ask clarification
        return &models.ConversationResponse{
            Response: crg.generateClarificationInstead(extractionArtifact),
            Metadata: map[string]interface{}{
                "responseValidationFailed": true,
                "violationCount": violations,
                "fallbackToClarity": true,
            },
        }, nil
    }
    
    // STEP 5: Response is valid, return it
    return response, nil
}

// buildConstraints creates facts-based constraints
func (crg *ConstrainedResponseGenerator) buildConstraints(
    userProfile *models.AboutMe,
    contacts []models.Contact,
    extractedContext *models.ExtractedContext,
    extractionArtifact *ExtractionArtifact,
) []Constraint {
    
    constraints := []Constraint{}
    
    // Constraint 1: User's known characteristics
    if extractionArtifact != nil {
        userEntities := extractionArtifact.GetEntitiesBySubject("user")
        for _, entity := range userEntities {
            if entity.Type == "property" || entity.Type == "characteristic" {
                antonyms := entity.Antonyms
                for _, contrary := range antonyms {
                    constraints = append(constraints, Constraint{
                        Type:    "user_characteristic",
                        Fact:    fmt.Sprintf("User is %s", entity.Value),
                        Dont:    fmt.Sprintf("Don't suggest %s", contrary),
                        Severity: "high",
                    })
                }
            }
        }
    }
    
    // Constraint 2: User's communication style
    if userProfile != nil && userProfile.CommunicationStyle != "" {
        constraints = append(constraints, Constraint{
            Type:    "communication_style",
            Fact:    fmt.Sprintf("User prefers %s communication", userProfile.CommunicationStyle),
            Dont:    "Don't use opposite communication style",
            Severity: "medium",
        })
    }
    
    // Constraint 3: Contact vulnerabilities
    for _, contact := range contacts {
        if strings.Contains(contact.Characteristics, "sensitive") {
            constraints = append(constraints, Constraint{
                Type:    "contact_vulnerability",
                Fact:    fmt.Sprintf("%s is sensitive", contact.Name),
                Dont:    fmt.Sprintf("Don't suggest harsh approaches with %s", contact.Name),
                Severity: "high",
            })
        }
    }
    
    return constraints
}

// buildConstrainedSystemPrompt includes constraints explicitly
func (crg *ConstrainedResponseGenerator) buildConstrainedSystemPrompt(constraints []Constraint) string {
    prompt := `You are Moly, a thinking partner helping users reason through communication challenges.

IMPORTANT CONSTRAINTS - You MUST respect these facts about the user:
`
    for _, c := range constraints {
        prompt += fmt.Sprintf("\n- %s (%s severity)\n  ✓ DO: %s\n  ✗ DON'T: %s",
            c.Fact, c.Severity, c.Do, c.Dont)
    }
    
    prompt += `

Generate a response that:
1. Respects all constraints above
2. Helps user think through consequences
3. Is conversational and warm
4. Offers alternatives when appropriate

If you cannot respect a constraint, ask a clarifying question instead of giving advice.`
    
    return prompt
}

// validateResponse checks if response violates constraints
func (crg *ConstrainedResponseGenerator) validateResponse(response string, constraints []Constraint) int {
    violations := 0
    lowerResponse := strings.ToLower(response)
    
    for _, constraint := range constraints {
        // Check: Does response violate this constraint?
        if constraint.Severity == "high" {
            // For high-severity, check strictly
            for _, dontWord := range strings.Split(constraint.Dont, ",") {
                dontWord = strings.TrimSpace(dontWord)
                if crg.containsWord(lowerResponse, dontWord) {
                    log.Printf("[ConstrainedGen] Violation: Response contains '%s' but constraint is '%s'",
                        dontWord, constraint.Fact)
                    violations++
                }
            }
        }
    }
    
    return violations
}

// containsWord checks for whole-word match (not substring)
func (crg *ConstrainedResponseGenerator) containsWord(text, word string) bool {
    pattern := fmt.Sprintf("\\b%s\\b", regexp.QuoteMeta(word))
    matched, err := regexp.MatchString(pattern, text)
    if err != nil {
        log.Printf("[ConstrainedGen] Regex error: %v", err)
        return false
    }
    return matched
}

// generateClarificationInstead creates clarification instead of bad response
func (crg *ConstrainedResponseGenerator) generateClarificationInstead(
    extractionArtifact *ExtractionArtifact,
) string {
    return fmt.Sprintf(`I want to make sure I understand you correctly before giving advice. 
    
Could you help me understand more about what you're looking for?`)
}
```

**Effort**: 3 days | **Tests**: 15 unit tests (constraint building, validation)

#### File 2: `models/constraint.go` (NEW)
**Purpose**: Define what a constraint is

```go
package models

type Constraint struct {
    ID       string   // Unique ID
    Type     string   // "user_characteristic", "communication_style", "contact_vulnerability"
    Fact     string   // What we know: "User is dominant"
    Do       string   // What to do: "Respect their assertiveness"
    Dont     string   // What not to do: "Don't suggest submission"
    Severity string   // "low", "medium", "high"
    Source   string   // Where this came from: "extracted", "aboutMe", "learned"
}
```

**Effort**: 0.5 day

#### File 3: `tools/response_generator.go` (Enhance)
**Current State**: `GenerateContextualResponse()` exists

**Changes**:
```go
// NEW: Add method to accept custom system prompt
func (rg *ResponseGenerator) GenerateWithSystemPrompt(
    ctx context.Context,
    userMessage string,
    userProfile *models.AboutMe,
    contacts []models.Contact,
    systemPrompt string, // NEW: allow custom system prompt
) (*models.ConversationResponse, error) {
    
    prompt := rg.buildPrompt(userMessage, userProfile, contacts)
    
    // Call LLM with custom system prompt
    resp, err := rg.llmClient.CallWithSystemPrompt(ctx, systemPrompt, prompt)
    
    // ... rest of method unchanged
}
```

**Effort**: 1 day (minimal changes, just add parameter)

#### File 4: `agents/conversation_agent.go` (Lines 1800-1900: Use ConstrainedResponseGenerator)
**Current State**: Uses ResponseGenerator directly

**Changes**:
```go
// Create constrained generator
constrainedGen := tools.NewConstrainedResponseGenerator(
    ca.llmClient,
    ca.responseGenerator,
    ca.db,
)

// Use it instead of basic generator (around line 1850)
agentResp, err := constrainedGen.Generate(
    ctx,
    userMessage,
    ctx.AboutMe,
    ctx.ContactProfiles,
    ctx.ExtractedContext,
    ctx.ExtractionArtifact, // NEW: pass the artifact with principle metadata
)
```

**Effort**: 1 day | **Tests**: 5 integration tests

#### File 5: `database/migrations/028_add_response_validation_log.sql`
**Changes**:
```sql
-- Track response validations
CREATE TABLE response_validations (
    id UUID PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    response_text TEXT,
    constraints_applied INT,
    constraint_details JSONB,
    violations_detected INT,
    passed_validation BOOLEAN,
    fallback_used BOOLEAN,
    validated_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX idx_response_validations_user ON response_validations(user_id);
```

**Effort**: 0.5 day

### Optimization: Caching & Async Validation (Phase 3)

**Problem**: Constraint building + validation adds latency.

**Solution**:
1. **Cache constraint lists** - Build once per user, reuse (TTL: 1 hour)
   - Key: (userID, conversationID)
   - Value: constraint list
   - Hit rate expected: 80%+

2. **Async validation** - Validate response while generating next message
   - Doesn't block response return
   - Logs violations for next interaction

3. **Confidence-based fallback** - If LLM confidence < 0.8, skip validation
   - Rare cases where LLM unsure
   - Fall back to asking clarification

**Implementation**:
```go
// Cache constraint lists (1 hour TTL)
constraintCache := NewLRUCache(1000, 1*time.Hour)

// Async validation (non-blocking)
go func() {
    violations := validateResponse(resp, constraints)
    if violations > 0 {
        logViolation(userID, violationCount)
    }
}()
```

**Expected Impact**:
- Constraint building: 80% from cache (< 10ms vs 200ms)
- Validation: Async (not in response path)
- Total latency increase: < 5% (vs 10% without optimization)

### Success Criteria (Phase 3 Go/No-Go)
- ✅ Constraints built from facts and principles
- ✅ Constraints visible in LLM system prompt
- ✅ Response validated before returning
- ✅ Zero role-reversal bugs (compare before/after on 100 test cases)
- ✅ Constraint violation rate < 5% (for edge cases)
- ✅ Performance: latency increase < 10%
  - p50: within 5% of baseline
  - p95: within 10% of baseline
  - p99: within 15% of baseline
- ✅ Cache hit rate > 70%
- ✅ No regressions in existing functionality

### Performance Benchmarks (Phase 3)
```
Metric                      Baseline    Target      Threshold
Constraint building         200ms       10ms (cached) < 400ms
Constraint validation       50ms        (async)     N/A
Response latency (p95)      2-3s        2.4-3.2s    < 3.5s (rollback)
Response latency (p99)      3-4s        3.2-4.2s    < 4.5s (rollback)
Role-reversal bug rate      0.1% (prev) 0% (target) > 0.05% (rollback)
False positive rate (wrong block) N/A    < 5%        > 8% (rollback)
```

### Deployment Strategy (Phase 3)
- **Week 1**: 10% users with feature flag (48 hour monitoring)
  - Collect constraint violation data
  - Monitor latency distribution
  - Manual review: 20 responses for quality
  
- **Week 2**: 50% users (1 week monitoring)
  - A/B test: old vs new response quality
  - Track user satisfaction metrics
  
- **Week 3**: 100% users
  - Full rollout
  - Monitor for 2 weeks

- Feature flag: `useConstrainedResponseGeneration` (default false)
- Fallback: Old ResponseGenerator if flag off

### Monitoring (Phase 3)
- Track constraint_violations count
- Track cache_hit_rate
- Track response_latency (p50, p95, p99)
- Alert if:
  - Latency p95 > 3.5s
  - Violation rate > 5%
  - Cache hit rate < 60% (might need larger cache)

### A/B Test (Phase 3)
- 50% users: new constrained generation
- 50% users: old generation (control)
- Measure:
  - Response quality (rating)
  - Role-reversal bug rate
  - User satisfaction

### Rollback Plan
- Disable feature flag: `useConstrainedResponseGeneration = false`
- Revert to basic ResponseGenerator
- Effort: < 1 hour (feature flag flip)
- Note: Role-reversal bug returns (accept tradeoff if performance critical)

---

## PHASE 4: CLEAN DATABASE REDESIGN (1-2 WEEKS)

### Purpose
Design database schema from scratch for new architecture. No migration complexity, clean implementation, built-in principle tracking.

### Why Clean Schema (Not Migration)

**Old approach (dual-write)**: Complex, 3 weeks
- Week 1: Add columns
- Week 2: Dual-write + backfill
- Week 3: Cutover + monitor

**New approach (clean build)**: Simple, 1 week
- Day 1: Design schema (no legacy baggage)
- Days 2-3: Export active data + migration job
- Day 4: Deploy + test (0-downtime or 1hr maintenance)
- Done

**Data we can safely lose**:
- Clarification questions older than 30 days (already expire)
- Old extractions (only need current state)
- Historical validation logs (only useful for debugging)

**Data we keep**:
- Current AboutMe (user profile)
- Current Contacts (relationship data)
- Recent conversations (30-day window)
- All active clarifications (current conversation)

### New Schema Design (From Scratch)

#### Core Philosophy
- **One table per concept**: extracting, clarifying, validating, learning
- **Principle metadata built-in**: Not added later
- **Immutability enforced**: Can't modify locked data
- **Reason logged always**: Why each question was asked
- **Clean indexes**: Only what's needed

#### Complete Schema

**File 1: `database/migrations/030_create_clean_schema.sql`**

```sql
-- ========================================
-- AUTHENTICATION & USER PROFILE
-- ========================================

-- Users (unchanged, existing data migrated)
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Sessions (unchanged)
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User Profile (simplified, clean)
CREATE TABLE user_profile (
    user_id TEXT PRIMARY KEY REFERENCES users(id),
    communication_style TEXT,
    values TEXT,  -- JSON array
    goals TEXT,   -- JSON array
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ========================================
-- RELATIONSHIPS & CONTACTS
-- ========================================

-- Contacts (simplified)
CREATE TABLE contacts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    relationship_type TEXT,
    characteristics TEXT,  -- JSON: {dominant: true, sensitive: false}
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, name)
);

-- ========================================
-- CONVERSATIONS & MESSAGES
-- ========================================

CREATE TABLE conversations (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    archived_at TIMESTAMP  -- NULL = active, set when archived
);

CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id),
    sender TEXT NOT NULL,  -- "user" or "moly"
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(conversation_id, sender, created_at)
);

-- ========================================
-- EXTRACTION (NEW - Core to Redesign)
-- ========================================

-- Locked extraction artifacts
CREATE TABLE extractions (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id),
    message_id TEXT NOT NULL REFERENCES messages(id),
    is_locked BOOLEAN DEFAULT false,
    locked_at TIMESTAMP,
    extraction_confidence FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(conversation_id, message_id)
);

-- Extracted entities with principle metadata
CREATE TABLE extracted_entities (
    id TEXT PRIMARY KEY,
    extraction_id TEXT NOT NULL REFERENCES extractions(id),
    type TEXT NOT NULL,           -- "User", "Contact", "Property", "Value", "Goal"
    value TEXT NOT NULL,
    subject TEXT NOT NULL,        -- "user", contact_name, etc
    principles TEXT,              -- JSON: ["Harm Prevention", "User Autonomy"]
    antonyms TEXT,                -- JSON: ["submissive", "passive"]
    antonym_source TEXT,          -- "llm_reasoning"
    bidirectional BOOLEAN DEFAULT false,
    confidence FLOAT NOT NULL,
    evidence TEXT,                -- Exact quote from message
    source TEXT DEFAULT 'llm',    -- "llm", "user_corrected", "clarification"
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(extraction_id, value, subject)
);

-- Conflicts detected during extraction
CREATE TABLE detected_conflicts (
    id TEXT PRIMARY KEY,
    extraction_id TEXT NOT NULL REFERENCES extractions(id),
    entity_a_id TEXT NOT NULL REFERENCES extracted_entities(id),
    entity_b_id TEXT NOT NULL REFERENCES extracted_entities(id),
    type TEXT NOT NULL,           -- "characteristic_conflict", "value_conflict"
    severity TEXT DEFAULT 'medium',  -- "low", "medium", "high"
    description TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    resolution TEXT
);

-- ========================================
-- CLARIFICATION FLOW (NEW - Redesigned)
-- ========================================

-- Clarification questions with principle tracking
CREATE TABLE clarification_questions (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    layer TEXT NOT NULL,          -- "4", "5", "6-7", "8", "9", "10"
    question_type TEXT NOT NULL,  -- "gap", "conflict", "principle", "socratic", "topic_shift"
    question_text TEXT NOT NULL,
    principle_basis TEXT,         -- "Harm Prevention", "User Autonomy", etc
    reason TEXT NOT NULL,         -- Why this question was asked
    
    -- Links to what triggered this question
    linked_conflict_id TEXT REFERENCES detected_conflicts(id),
    linked_gap TEXT,              -- Gap name if Layer 4
    linked_principle TEXT,        -- Principle name if Layer 6-7
    
    -- Status
    status TEXT DEFAULT 'pending', -- "pending", "answered", "clarified"
    answer_text TEXT,
    answered_at TIMESTAMP,
    
    -- Deduplication tracking
    similar_questions_recent INT DEFAULT 0,
    deduplicated BOOLEAN DEFAULT false,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_conversation (conversation_id),
    INDEX idx_layer (layer),
    INDEX idx_principle_basis (principle_basis),
    INDEX idx_question_type (question_type)
);

-- ========================================
-- RESPONSE VALIDATION (NEW)
-- ========================================

-- Response validation audit trail
CREATE TABLE response_validations (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    response_text TEXT NOT NULL,
    constraints_applied INT,
    constraint_details TEXT,  -- JSON array of constraints
    violations_detected INT DEFAULT 0,
    passed_validation BOOLEAN,
    fallback_used BOOLEAN DEFAULT false,
    validated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_conversation (conversation_id),
    INDEX idx_user (user_id),
    INDEX idx_violations (violations_detected)
);

-- ========================================
-- PRINCIPLE TRACKING (NEW)
-- ========================================

-- Principle violations and resolutions
CREATE TABLE principle_violations (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id),
    extracted_entity_id TEXT REFERENCES extracted_entities(id),
    principle TEXT NOT NULL,     -- "Harm Prevention", "User Autonomy", etc
    violation_type TEXT,         -- "self_harm", "autonomy", "consent"
    severity TEXT DEFAULT 'medium', -- "low", "medium", "high", "critical"
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    resolution TEXT,
    
    INDEX idx_conversation (conversation_id),
    INDEX idx_principle (principle),
    INDEX idx_severity (severity)
);

-- ========================================
-- LEARNING & FACTS (Optional, Future)
-- ========================================

-- Learned facts about user (extracted from conversations)
CREATE TABLE learned_facts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    fact_type TEXT NOT NULL,     -- "characteristic", "value", "goal", "pattern"
    value TEXT NOT NULL,
    confidence FLOAT,
    source TEXT,                 -- Where learned (extracted, user_input, inferred)
    learned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_user (user_id),
    INDEX idx_fact_type (fact_type)
);

-- ========================================
-- MONITORING & CLEANUP
-- ========================================

-- Clean up old conversations (30-day retention)
CREATE TRIGGER cleanup_old_conversations
AFTER INSERT ON conversations
BEGIN
    DELETE FROM conversations 
    WHERE archived_at IS NOT NULL 
    AND archived_at < datetime('now', '-30 days');
END;
```

**Effort**: 1 day (design + SQL)

#### File 2: `database/migration_job.go` (NEW - Data Export & Import)

```go
package database

// MigrationJob exports active data and imports to new schema
type MigrationJob struct {
    db *Database
}

// Run executes the migration
func (mj *MigrationJob) Run(ctx context.Context) error {
    log.Printf("[Migration] Starting clean schema migration")
    
    // Step 1: Export active data (from old schema)
    users, err := mj.exportUsers()
    if err != nil {
        return fmt.Errorf("export users failed: %w", err)
    }
    
    contacts, err := mj.exportContacts()
    if err != nil {
        return fmt.Errorf("export contacts failed: %w", err)
    }
    
    conversations, err := mj.exportRecentConversations() // Last 30 days only
    if err != nil {
        return fmt.Errorf("export conversations failed: %w", err)
    }
    
    log.Printf("[Migration] Exported: %d users, %d contacts, %d conversations",
        len(users), len(contacts), len(conversations))
    
    // Step 2: Import to new schema
    if err := mj.importUsers(users); err != nil {
        return fmt.Errorf("import users failed: %w", err)
    }
    
    if err := mj.importContacts(contacts); err != nil {
        return fmt.Errorf("import contacts failed: %w", err)
    }
    
    if err := mj.importConversations(conversations); err != nil {
        return fmt.Errorf("import conversations failed: %w", err)
    }
    
    log.Printf("[Migration] Migration complete - ready for cutover")
    return nil
}

// Verify checks data consistency between old and new schema
func (mj *MigrationJob) Verify() error {
    oldCount, err := mj.countOldContacts()
    if err != nil {
        return err
    }
    
    newCount, err := mj.countNewContacts()
    if err != nil {
        return err
    }
    
    if oldCount != newCount {
        return fmt.Errorf("data mismatch: old=%d, new=%d", oldCount, newCount)
    }
    
    log.Printf("[Migration] Verification passed: %d records verified", oldCount)
    return nil
}
```

**Effort**: 1 day (implement export/import logic)

### Deployment Strategy (Phase 4 - Clean Schema)

**Timeline**: 1 week total (vs 3 weeks for dual-write)

**Week 1:**
- **Day 1-2**: Design schema (SQL), export/import job, testing
- **Day 3**: Test migration on database copy
- **Day 4**: Schedule downtime (1 hour) or use 0-downtime strategy
- **Day 5**: Run migration
  ```bash
  # Export active data (< 1 minute)
  ./moly-migrate export-active-data > backup.json
  
  # Create new schema (< 1 minute)
  psql < migrations/030_create_clean_schema.sql
  
  # Import data (< 30 seconds)
  ./moly-migrate import-active-data < backup.json
  
  # Verify (< 1 minute)
  ./moly-migrate verify
  
  # Switch to new schema (< 1 minute)
  ./moly-migrate cutover
  
  # Delete old schema (1 minute, keep backup 7 days)
  # (happens after 1-week verification period)
  ```
- **Day 6-7**: Monitor new schema, verify all queries work

**Zero-Downtime Alternative**:
- Run export/import in parallel
- Switch connection string
- No downtime needed

### Success Criteria (Phase 4 Go/No-Go)

- ✅ New schema designed and tested
- ✅ Migration job exports all active data correctly
- ✅ Data import verified (100% record count match)
- ✅ All indexes created and verified
- ✅ Query performance acceptable (p95 < 100ms)
- ✅ No data loss (backup retained 7 days)
- ✅ All layers can query new schema correctly
- ✅ Monitoring shows healthy queries

### Performance Benchmarks (Phase 4)

```
Metric                      Baseline    Target      Threshold
Query time (get questions)  50ms        50-55ms     < 100ms
Migration export time       N/A         < 1 min     < 5 min
Migration import time       N/A         < 1 min     < 5 min
Verification time           N/A         < 1 min     < 5 min
Index creation time         N/A         < 1 min     < 5 min
Data loss                   N/A         0 records   > 0 (rollback)
```

### Monitoring (Phase 4)

- Export/import process logs
- Row counts before/after
- Query performance on new schema
- No query errors in logs
- Alert if any migration step fails

### Rollback Plan (Phase 4)

- **If migration fails**: Revert connection string, use old schema
- **If queries slow**: Check indexes, rerun ANALYZE
- **If data mismatch**: Restore backup, investigate
- **Effort**: < 30 minutes (switch connection string back)
- **Data safety**: Backup retained 7 days, can restore if needed

---

## TIMELINE & EFFORT SUMMARY

| Phase | Duration | Effort | Risk | Rollback |
|-------|----------|--------|------|----------|
| 1: Extraction Lock | 2 weeks | 8 days | LOW | < 1 hour |
| 2: Conflict Channeling | 3 weeks | 10 days | MEDIUM | < 1 hour |
| 3: Constrained Generation | 2 weeks | 8 days | MEDIUM | < 1 hour |
| 4: Clean Schema | 1 week | 3 days | LOW | < 30 min |
| **TOTAL** | **~8 weeks** | **~29 days** | | |

**For 2 people (parallel)**: ~5 weeks
- Person A: Phases 1 & 3 (parallel)
- Person B: Phases 2 & 4 (parallel)
- Overlap: 1 week for integration testing

**Phase 4 Benefit**: Clean schema removes 2 weeks of dual-write/backfill complexity
- Old approach: 3 weeks (Week 10-12 in old plan)
- New approach: 1 week (simple export/import)
- Saved: 2 weeks, lower risk, cleaner code

---

## GO/NO-GO GATES

### After Phase 1 (Week 2)
**MEASURE**:
- ✅ Zero re-parsing in logs
- ✅ Extraction accuracy maintained on 100 test messages
- ✅ No regressions in existing tests
- ✅ All 50+ tests passing

**GO IF**: All metrics ✓  
**NO-GO IF**: Accuracy drops OR regressions found  
**ACTION IF NO-GO**: Rollback (1 day), investigate re-parsing detection logic

---

### After Phase 2 (Week 5)
**MEASURE**:
- ✅ 100% of detected conflicts generate questions
- ✅ Questions are Socratic (not accusatory)
- ✅ Deduplication prevents repeats
- ✅ No regressions

**GO IF**: All metrics ✓  
**NO-GO IF**: Conflicts not generating questions OR questions accusatory  
**ACTION IF NO-GO**: Rollback (1 day), refine question generation prompt

---

### After Phase 3 (Week 7)
**MEASURE**:
- ✅ Zero role-reversal bugs in test suite
- ✅ Constraint violation rate < 5%
- ✅ Response latency increase < 10% (p95 < 3s)
- ✅ No regressions

**GO IF**: All metrics ✓  
**NO-GO IF**: Latency degrades significantly OR false positives > 10%  
**ACTION IF NO-GO**: Rollback (1 day), optimize constraint checking

---

### After Phase 4 (Week 9)
**MEASURE**:
- ✅ 100% of questions have principle_basis filled
- ✅ Audit trails show clear reasoning
- ✅ No regressions
- ✅ Database schema stable

**GO IF**: All metrics ✓  
**NO-GO IF**: Data inconsistencies OR audit trails unclear  
**ACTION IF NO-GO**: No rollback needed (just cleanup), migrate data

---

## TESTING STRATEGY

### Unit Tests (Per Phase)
| Component | Tests |
|-----------|-------|
| Extraction lock/principle metadata | 15 |
| Layer 5 conflict handler | 12 |
| Clarification history/dedup | 10 |
| Constraint building | 12 |
| Response validation | 10 |
| Principle tracking | 8 |
| **TOTAL** | **67 new tests** |

### Integration Tests (Cross-phase)
- End-to-end: message → extraction → Layer 5 → response
- Conflict scenarios: 5 test cases
- Principle scenarios: 5 test cases
- Role-reversal prevention: 10 test cases
- Deduplication: 5 test cases
- **TOTAL**: 30 integration tests

### Regression Tests
- All existing 50+ tests must pass
- No latency regressions (p95 baseline)
- No accuracy regressions on extraction

---

## DEPLOYMENT STRATEGY

### Staged Rollout (Per Phase)
1. **Phase 1** (Extraction lock):
   - 100% rollout immediately (low risk, no behavior change)

2. **Phase 2** (Conflict channeling):
   - 10% users → 24 hours
   - 50% users → 3 days
   - 100% users → 1 week
   - Monitor: question ask rate, user satisfaction

3. **Phase 3** (Constrained generation):
   - 10% users → 48 hours
   - 50% users → 1 week
   - 100% users → 2 weeks
   - Monitor: role-reversal bugs, latency, constraint violations

4. **Phase 4** (Tracking):
   - 100% rollout (no behavior change, just tracking)

### Monitoring Per Phase
- Error rates
- Response latency (p50, p95, p99)
- User satisfaction (if available)
- Bug reports related to questions/responses
- Constraint violation rate

### Rollback Trigger
- Error rate > 2% above baseline → Rollback
- Latency p95 > baseline + 30% → Rollback
- More than 3 reported bugs per 1000 users → Investigate

---

## ARCHITECTURE DIAGRAM

```
USER MESSAGE
    ↓
[PHASE 1: Locked Extraction]
├─ Extract once (LLM + principle metadata)
├─ Detect conflicts
├─ LOCK (cannot re-parse)
└─ Send to downstream
    ↓
[PHASE 2: Layer 5 Channeling]
├─ Receive locked extraction + conflicts
├─ Layer 5 handler: conflict → question
├─ Deduplication check
└─ If conflict: ask clarification, STOP
    ↓
[Phase 1-4: Other gates]
├─ Layer 4: Gap-based questions
├─ Layer 6-7: Principle concerns
├─ Layer 9: Topic shifts
└─ Layer 8: Socratic (if all clear)
    ↓
[PHASE 3: Constrained Generation]
├─ Build constraints from facts + principles
├─ Generate response WITH constraints
├─ Validate response BEFORE returning
└─ If violates: ask clarification instead
    ↓
[PHASE 4: Tracking & Metadata]
├─ Log principle_basis for every question
├─ Log layer for every question
├─ Log reason for every question
└─ Enable audit trail
    ↓
SEND RESPONSE (validated, constrained, tracked)
```

---

## SUMMARY

**Keep**: The 11-layer clarification philosophy (it's brilliant)  
**Fix**: How extraction, conflicts, and validation are implemented  
**Result**: Same purpose (help user think), better way (extraction lock, conflict channeling, constrained generation)

**Timeline**: 9 weeks (1 person) or 5-6 weeks (2 people)  
**Risk**: LOW-MEDIUM (all phases have rollback plans)  
**Benefit**: Eliminates role-reversal bugs, ensures all conflicts asked, deduplicated questions

---

## COMPREHENSIVE MONITORING & METRICS

### Real-Time Dashboards (All Phases)

**Phase 1: Extraction Lock**
```
extraction_lock_enabled: boolean
extraction_time_ms: histogram [p50, p95, p99]
multi_pass_calls_per_message: gauge
subject_attribution_accuracy: percent
extraction_lock_failures: counter
```

**Phase 2: Conflict Channeling**
```
conflicts_detected_per_conversation: histogram
conflicts_resulting_in_questions: percent
question_deduplication_prevented: counter
conflict_question_ask_rate: percent
user_response_time_to_conflict_q: histogram
```

**Phase 3: Constrained Generation**
```
constraints_built: counter
constraint_cache_hit_rate: percent
response_validation_violations: counter
response_latency_ms: histogram [p50, p95, p99]
role_reversal_bug_occurrences: counter (TARGET: 0)
false_positive_blocks: counter
```

**Phase 4: Tracking**
```
questions_with_principle_basis: percent
questions_with_reason: percent
audit_trail_completeness: percent
backfill_progress: percent
query_latency_ms: histogram
```

### SLA Thresholds (Rollback Triggers)

| Metric | Phase | Threshold | Action |
|--------|-------|-----------|--------|
| Response latency p95 | 1-4 | > baseline + 20% | ALERT |
| Response latency p95 | 1-4 | > baseline + 30% | ROLLBACK |
| Error rate | 1-4 | > 2% | ROLLBACK |
| Role-reversal bugs | 3 | > 0 per 1000 users | INVESTIGATE |
| Constraint violations | 3 | > 8% | ROLLBACK |
| Question abandonment | 2 | > 25% | INVESTIGATE |
| Data corruption | 4 | > 0 | ROLLBACK |
| Cache hit rate | 3 | < 60% | ALERT |

### Weekly Reporting (During Rollout)

Each phase reports:
1. **Metrics Summary**
   - Key metrics vs targets
   - Any threshold breaches
   - Recommended action (continue/pause/rollback)

2. **User Impact**
   - Question ask rate
   - User satisfaction
   - Bug reports
   - Error rates

3. **Performance**
   - Latency distribution (p50, p95, p99)
   - Cache performance
   - Database query times

4. **Reliability**
   - Rollback readiness
   - Data consistency
   - Fallback activation rate

---

## SUCCESS METRICS PER PHASE

### Phase 1: Extraction Lock
**Measure**: Extraction quality + performance

| Metric | Baseline | Target | Success Criterion |
|--------|----------|--------|-------------------|
| Multi-pass parsing calls | 7/msg | 0/msg | ≤ 1/msg (test-only) |
| Subject attribution accuracy | 95% | 95%+ | ≥ 94% |
| Extraction time p95 | 200ms | 200ms | < 250ms |
| Test pass rate | 100% | 100% | ≥ 99.5% |

**Decision**: If all targets met → GO to Phase 2

---

### Phase 2: Conflict Channeling
**Measure**: Question quality + no regressions

| Metric | Baseline | Target | Success Criterion |
|--------|----------|--------|-------------------|
| Conflicts generating questions | 70% | 100% | ≥ 95% |
| Question quality (manual review) | N/A | Socratic | ≥ 18/20 rated good |
| Repeated questions | 5-10% | < 1% | < 2% |
| Response latency p95 | 2-3s | 2-3s | < 3.5s |
| User question abandonment | 10-15% | < 20% | < 25% |

**Decision**: If 100% conflicts ask, and latency ok → GO to Phase 3

---

### Phase 3: Constrained Generation
**Measure**: Bug elimination + performance maintained

| Metric | Baseline | Target | Success Criterion |
|--------|----------|--------|-------------------|
| Role-reversal bugs | 0.1% users | 0% | ZERO in test cohort |
| Constraint violations | N/A | < 5% | < 8% (false positives ok) |
| Response latency p95 | 2-3s | 2.4-3.2s | < 3.5s (alert), < 4s (rollback) |
| Cache hit rate | N/A | > 70% | ≥ 60% |
| False positive blocks | N/A | < 5% | < 10% |
| User satisfaction | N/A | Stable | No decrease in ratings |

**Decision**: If role-reversal = 0 and latency ok → GO to Phase 4

---

### Phase 4: Clean Schema
**Measure**: Correct data migration + performance

| Metric | Baseline | Target | Success Criterion |
|--------|----------|--------|-------------------|
| Data loss | N/A | 0 records | 0 |
| Migration export time | N/A | < 1 min | < 5 min |
| Migration import time | N/A | < 1 min | < 5 min |
| Data verification (count match) | N/A | 100% | ≥ 99% |
| Query latency p95 | 50ms | 50-55ms | < 100ms |
| Rollback safety | N/A | Backup retained 7 days | Confirmed |
| All queries work on new schema | N/A | 100% | ≥ 99% no errors |

**Decision**: If migration clean (0 data loss) and queries ok → COMPLETE

---

## RISK SUMMARY

| Risk | Phase | Probability | Impact | Mitigation |
|------|-------|-------------|--------|-----------|
| Accuracy regression | 1 | LOW | HIGH | Feature flag, benchmarks, rollback in 1 day |
| Latency spike | 2-3 | MEDIUM | HIGH | Caching, async, SLA monitoring, rollback < 1h |
| Performance degradation | 3 | MEDIUM | MEDIUM | Constraint caching, async validation |
| Data corruption | 4 | LOW | HIGH | Dual-write, backfill verification, 30-day archive |
| Question quality issues | 2 | LOW | MEDIUM | Manual review, LLM-driven, easy to fix |
| Constraint false positives | 3 | MEDIUM | LOW | User override, confidence threshold |
| Database query slowdown | 4 | LOW | MEDIUM | Index strategy, migration testing |

---

## PRE-IMPLEMENTATION CHECKLIST

**Code Quality**:
- [ ] All 50+ existing tests passing
- [ ] Code review of implementation plan
- [ ] Test plan reviewed (67 new unit tests, 30 integration tests)

**Infrastructure**:
- [ ] Production database backed up (Phase 4 especially)
- [ ] Monitoring dashboards created
- [ ] Feature flags configured (Phases 1-3)
- [ ] Alerting rules set up (SLA thresholds)

**Operations**:
- [ ] Rollback procedures tested (all phases)
- [ ] Team trained on monitoring
- [ ] On-call prepared for rollout
- [ ] Stakeholders briefed on 8-week timeline

**Deployment**:
- [ ] Gradual rollout plan confirmed (10% → 50% → 100%)
- [ ] SLA thresholds documented
- [ ] Go/no-go decision criteria clear
- [ ] Database migration job tested (Phase 4)

---

## WHAT'S IN THIS PLAN

✅ **4 Phases, ~8 weeks**:
1. Extraction lock (2w, 8d) - Fix multi-pass parsing
2. Conflict channeling (3w, 10d) - All conflicts become questions
3. Constrained generation (2w, 8d) - Validate before sending
4. Clean schema (1w, 3d) - New database from scratch

✅ **Operational Excellence**:
- Feature flags for rollback (< 1 hour)
- Gradual rollout (10% → 50% → 100%)
- Performance benchmarks (all phases)
- SLA monitoring with rollback triggers
- A/B testing (Phase 3)

✅ **Safety & Reliability**:
- 97+ new tests (unit + integration)
- Rollback plans per phase (< 30 min to 1 hour)
- Monitoring dashboards (real-time)
- Zero-downtime deployment (Phase 4)

✅ **Clean Architecture**:
- No dual-write complexity
- No migration backfill delays
- Database designed for new purpose (not adapted)
- Principle metadata built-in (not added later)

---

## QUICK START

1. **Read this plan** (this document)
2. **Review Phase 1** details (Extraction Lock)
3. **Set up monitoring** (dashboards, alerts, thresholds)
4. **Configure feature flags** (ready to toggle)
5. **Run Phase 1 tests** (67 total across all phases)
6. **Begin Phase 1** (Extraction Lock)

Expected timeline:
- Phases 1-3: Week 1-7 (in parallel: Persons A & B)
- Phase 4: Week 7-8 (during Phase 3 final testing)
- 1 week buffer: Testing + monitoring

---

**Status**: READY TO IMPLEMENT

**Next Action**: Begin Phase 1 (Extraction Lock)

