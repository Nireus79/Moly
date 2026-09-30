# REDESIGN: Detailed File Mapping

## Phase 1A: Extraction Lock + Principle Metadata (2 weeks)

### FILES TO MODIFY

#### `agents/intent_detector.go` (MODIFY: 200 LOC)
```
CURRENT:
- extractEntitiesWithLLM() - extracts entities
- Returns ExtractedEntity[]

REDESIGNED:
- Add LLM prompt section for principle extraction
- Extract: "What principles does this relate to?"
- Extract: "What would contradict this?"
- Add error handling for principle extraction failures
- Return enhanced ExtractedEntity with Principles, Antonyms

LINES CHANGED:
  ~850: Update LLM prompt to include principle extraction
  ~900: Parse principle metadata from response
  ~920: Parse antonym list from response
  ~950: Validation: ensure antonyms exist for key characteristics

REUSE:
  ✅ Core extraction logic (just enhance it)
  ✅ LLM parsing (adapt for new fields)
  ✅ Error handling (keep existing)
```

#### `models/agent_types.go` (MODIFY: 50 LOC)
```
CURRENT:
type ExtractedEntity struct {
    Value      string
    Type       string
    Subject    string
    Confidence float64
    Evidence   string
}

REDESIGNED:
type ExtractedEntity struct {
    // Existing
    Value      string
    Type       string
    Subject    string
    Confidence float64
    Evidence   string
    
    // NEW
    Principles     []string  // ["Harm Prevention", "User Autonomy"]
    Antonyms       []string  // ["submissive", "passive"]
    AntonymSource  string    // "llm_reasoning"
    Bidirectional  bool      // true if both directions contradict
}

LINES CHANGED:
  All in agent_types.go - just add 4 fields

REUSE:
  ✅ All existing fields kept
  ✅ All existing logic unchanged
```

#### `models/extraction_artifact.go` (MODIFY: 100 LOC)
```
CURRENT:
type ExtractionArtifact struct {
    Entities    []ExtractedEntity
    Source      string
    Duration    float64
}

REDESIGNED:
type ExtractionArtifact struct {
    // Existing
    Entities    []ExtractedEntity
    Source      string
    Duration    float64
    
    // NEW
    IsImmutable bool
    LockedAt    time.Time
    
    // NEW METHODS
    func (ea *ExtractionArtifact) Lock() error {
        if ea.IsImmutable return ErrAlreadyLocked
        ea.IsImmutable = true
        ea.LockedAt = time.Now()
        return nil
    }
    
    func (ea *ExtractionArtifact) ValidateEntities() []error {
        // Check: all entities have principle info
        // Check: antonyms exist for key characteristics
    }
}

LINES CHANGED:
  Add 3 fields, 2 methods (~30 LOC)

REUSE:
  ✅ All existing fields kept
  ✅ All existing logic unchanged
```

### FILES TO CREATE

#### `database/migrations/023_add_extraction_principles.sql` (NEW: 50 LOC)
```sql
-- Add principle metadata storage
ALTER TABLE extracted_entities ADD COLUMN (
    principles TEXT[] DEFAULT '{}',
    antonyms TEXT[] DEFAULT '{}',
    antonym_source TEXT DEFAULT 'llm_reasoning',
    bidirectional BOOLEAN DEFAULT false
);

CREATE INDEX idx_extracted_entities_principles 
ON extracted_entities USING GIN(principles);
```

#### `models/principle_metadata.go` (NEW: 100 LOC)
```go
package models

type PrincipleMetadata struct {
    Principle      string
    Description    string
    Category       string
    RelatedPrinciples []string
}

var KnownPrinciples = []PrincipleMetadata{
    {
        Principle: "Harm Prevention",
        Description: "Avoid causing harm to self or others",
        Category: "Safety",
    },
    // ... more principles
}

func (pm *PrincipleMetadata) FindAntonyms(value string) []string {
    // Find contradictory values for a given principle
}
```

### TESTS TO ADD

#### `agents/intent_detector_test.go` (ADD: 300 LOC)
```go
TestExtractPrinciples
├─ Test: "I'm dominant" → extracts principle "power dynamics"
├─ Test: "I want to manipulate" → extracts principle "autonomy violation"
├─ Test: Low confidence → marks with uncertainty flag

TestAntonymExtraction
├─ Test: "dominant" → antonyms include "submissive"
├─ Test: "assertive" → antonyms include "hesitant"
├─ Test: LLM failure → graceful fallback

TestLockMechanism
├─ Test: Can lock extraction
├─ Test: Cannot unlock
├─ Test: Cannot modify after lock
```

### TOTAL PHASE 1A
```
Files Modified:    3 (agents, models, models)
Files Created:     2 (migration, models)
Tests Added:       1 (~300 LOC)
Total New LOC:     ~350
Total Changed LOC: ~250
Risk Level:        LOW
```

---

## Phase 1B: Lock Enforcement (2 weeks)

### FILES TO MODIFY

#### `database/clarification_capture.go` (MODIFY: 400 LOC)
```
CURRENT:
- Multi-pass extraction with LinguisticParser fallback
- Lines 100-200: "If phase 1 fails, re-parse with LinguisticParser"
- Lines 250-350: Multiple extraction passes
- Lines 400-500: Subject attribution logic (uses original + re-parsed)

REDESIGNED:
- REMOVE all re-parsing logic (lines 100-200)
- REMOVE LinguisticParser dependency
- Keep: Use only locked extraction
- New validation: If extraction confidence too low, ask clarification (don't re-parse)
- Add: explicit needsClarification flag to extraction

LINES REMOVED: ~150
LINES ADDED:   ~100
LINES CHANGED: ~150

REUSE:
  ✅ Subject tracking logic (just use locked extraction)
  ✅ Clarification question generation
  ✅ Database operations
```

#### `database/contact_deduplicator.go` (MODIFY: 200 LOC)
```
CURRENT:
- resolveFromConversationHistory() tries to use extracted subject
- Falls back to regex pattern matching if extraction unclear
- Creates duplicates if subject resolution fails

REDESIGNED:
- Only use locked extraction subject
- If subject unclear in extraction, ask clarification (don't create duplicate)
- Remove fallback to regex/heuristics

LINES REMOVED: ~80
LINES ADDED:   ~50
LINES CHANGED: ~100

REUSE:
  ✅ Deduplication logic
  ✅ Contact merging
```

#### `main.go` (MODIFY: 300 LOC)
```
CURRENT:
- Line 2300-2400: "Load extraction, then re-parse if needed"
- Line 2450-2550: "Use extracted or re-parsed entities"
- Line 2600+: Multiple fallback paths

REDESIGNED:
- Line 2300-2400: "Load extraction and verify it's locked"
- Remove all "if extraction failed, re-parse" paths
- Error instead of fallback: "Extraction must be locked, contact support if not"

LINES REMOVED: ~150
LINES ADDED:   ~50
LINES CHANGED: ~200

REUSE:
  ✅ Most of main.go unchanged
  ✅ Database operations
  ✅ LLM client usage
```

#### `database/extraction_store.go` (NEW: 200 LOC)
```go
package database

type ExtractionRepository struct {
    db *Database
}

// Get locked extraction or error
func (er *ExtractionRepository) GetLockedExtraction(id string) (*models.LockedExtraction, error) {
    extraction := er.db.GetExtraction(id)
    if !extraction.IsImmutable {
        return nil, ErrExtractionNotLocked
    }
    return extraction, nil
}

// Prevent unlocking
func (er *ExtractionRepository) TryModify(id string) error {
    extraction := er.db.GetExtraction(id)
    if extraction.IsImmutable {
        return ErrCannotModifyLockedExtraction
    }
    return nil
}

// Lock an extraction
func (er *ExtractionRepository) Lock(id string) error {
    extraction := er.db.GetExtraction(id)
    return extraction.Lock()
}
```

### TESTS TO ADD

#### `database/clarification_capture_test.go` (ADD: 200 LOC)
```go
TestNoReparse
├─ Test: Extraction used as-is (no re-parsing)
├─ Test: Low confidence → asks clarification (doesn't re-parse)
├─ Test: LinguisticParser NOT called

TestSubjectTracking
├─ Test: Subject from locked extraction respected
├─ Test: Multi-person message → proper subject attribution
├─ Test: Pronoun resolution uses extraction (not heuristics)
```

#### `database/contact_deduplicator_test.go` (ADD: 150 LOC)
```go
TestNoDuplicates
├─ Test: Pronoun "her" → resolved from extraction
├─ Test: No duplicate "her" created
├─ Test: If resolution unclear → asks clarification
```

### TOTAL PHASE 1B
```
Files Modified:    4 (clarification, deduplicator, main, extraction_store)
Tests Added:       2 (~350 LOC)
Total New LOC:     ~200
Total Changed LOC: ~600
Risk Level:        LOW-MEDIUM
Lines Removed:     ~380 (dead code elimination)
```

---

## Phase 2: Unified Clarification Orchestrator (3 weeks)

### FILES TO CREATE

#### `agents/orchestration/clarification_orchestrator.go` (NEW: 800 LOC)
```go
package orchestration

type ClarificationOrchestrator struct {
    facts            *models.FactSet
    contradictions   []Contradiction
    gaps             []ContextGap
    ambiguities      []Ambiguity
    principleIssues  []PrincipleIssue
    llmClient        LLMProvider
}

func (co *ClarificationOrchestrator) GetNextQuestion(ctx context.Context) (*Question, error) {
    // Priority-based selection
    // All question types coordinated here
}

func (co *ClarificationOrchestrator) CreateQuestion(reason string, principle string) (*Question, error) {
    // LLM-generated questions (not hardcoded)
}

func (co *ClarificationOrchestrator) ShouldReaskQuestion(q *Question) bool {
    // Deduplication logic
}
```

#### `agents/orchestration/question_history.go` (NEW: 300 LOC)
```go
package orchestration

type QuestionHistory struct {
    questions []*Question
}

func (qh *QuestionHistory) WasAskedRecently(q *Question) bool {
    // Semantic similarity check (not exact text match)
}

func (qh *QuestionHistory) GetSimilarQuestions(q *Question) []*Question {
    // Find similar questions in history
}
```

### FILES TO MODIFY

#### `agents/conversation_agent.go` (MODIFY: 500 LOC)
```
CURRENT:
- Layers 4-8 scattered across file
- Gap detection in one place
- Question generation in another
- Contradiction handling in third

REDESIGNED:
- Route all clarification logic through ClarificationOrchestrator
- Keep: response generation, context loading
- Remove: Layer-by-layer question logic
- New: single call to orchestrator.GetNextQuestion()

LINES REMOVED: ~400 (consolidating logic)
LINES ADDED:   ~100 (orchestrator integration)
LINES CHANGED: ~300

REUSE:
  ✅ Context loading
  ✅ Response generation
  ✅ Error handling
```

#### `database/clarification_repository.go` (MODIFY: 100 LOC)
```
CURRENT:
type ClarificationQuestion struct {
    Question  string
    Status    string
    UserID    string
}

REDESIGNED:
type ClarificationQuestion struct {
    // Existing
    Question  string
    Status    string
    UserID    string
    
    // NEW
    PrincipalBasis string  // "Harm Prevention"
    QuestionReason string  // "gap" | "contradiction"
    OrchestratorID string  // Links to orchestrator run
}

LINES ADDED: ~20
```

#### `database/migrations/024_add_question_tracking.sql` (NEW: 40 LOC)
```sql
ALTER TABLE clarification_questions ADD COLUMN (
    principle_basis TEXT,
    question_reason TEXT,
    orchestrator_run_id TEXT,
    similar_asked_count INT DEFAULT 0
);

CREATE INDEX idx_questions_principle ON clarification_questions(principle_basis);
```

### TESTS TO ADD

#### `agents/orchestration/clarification_orchestrator_test.go` (ADD: 500 LOC)
```go
TestPriorities
├─ Test: Self-harm questions first
├─ Test: Principle violations second
├─ Test: Contradictions third
├─ Test: Gaps fourth
├─ Test: Socratic deepening only when ready

TestQuestionDedup
├─ Test: Same question not re-asked
├─ Test: Similar questions not re-asked
├─ Test: Different questions all asked

TestLLMGeneration
├─ Test: Questions generated by LLM (not hardcoded)
├─ Test: Questions specific to user context
├─ Test: LLM generation failure → fallback
```

### FEATURE FLAG

Add to config:
```go
UseNewOrchestrator bool = false  // Start at 10%, increase to 100%
```

### TOTAL PHASE 2
```
Files Created:     2 (orchestrator, question_history)
Files Modified:    3 (conversation_agent, clarification_repository, migration)
Tests Added:       1 (~500 LOC)
Total New LOC:     ~1100
Total Changed LOC: ~400
Lines Removed:     ~400 (consolidating scattered logic)
Risk Level:        HIGH (requires feature flag + gradual rollout)
```

---

## Phase 3: Constrained Response Generation (2 weeks)

### FILES TO CREATE

#### `tools/constrained_response_generator.go` (NEW: 600 LOC)
```go
package tools

type ConstrainedResponseGenerator struct {
    facts             *models.FactSet
    extractedNow      []ExtractedEntity
    userMaturity      float64
    llmClient         LLMProvider
    validator         ResponseValidator
}

func (crg *ConstrainedResponseGenerator) Generate(ctx context.Context) (*Response, error) {
    // Step 1: Build constraints
    constraints := crg.buildConstraints()
    
    // Step 2: Generate with constraints in LLM prompt
    resp, err := crg.llm.Call(ctx, &LLMRequest{
        SystemPrompt: crg.buildConstrainedPrompt(constraints),
        UserPrompt:   crg.buildUserPrompt(),
    })
    
    // Step 3: Validate response
    violations := crg.validateResponse(resp, constraints)
    if violations > 0 {
        return nil, NewResponseContradictionError(violations)
    }
    
    // Step 4: Save learning
    crg.saveLearning(resp)
    return resp, nil
}

func (crg *ConstrainedResponseGenerator) buildConstraints() []Constraint {
    // Build from user characteristics, contact profiles, values
}

func (crg *ConstrainedResponseGenerator) validateResponse(resp *Response, constraints []Constraint) int {
    // Count violations
}
```

#### `tools/response_validator.go` (NEW: 400 LOC)
```go
package tools

type ResponseValidator struct {
    antonymGraph    *AntonymGraph
    principleChecker *PrincipleChecker
}

func (rv *ResponseValidator) ValidateAgainstFacts(response string, facts *FactSet) ValidationResult {
    // Check if response violates any facts
    // Returns: violations count, violation list
}

// Reuse/integrate existing ValidateResponseAgainstCharacteristics
```

### FILES TO MODIFY

#### `agents/conversation_agent.go` (MODIFY: 200 LOC)
```
CURRENT:
- Lines ~1850: Generate response

REDESIGNED:
- Replace ResponseGenerator with ConstrainedResponseGenerator
- Keep: context building, error handling
- New: fallback if constrained generation fails

LINES CHANGED: ~150
LINES ADDED:   ~50

REUSE:
  ✅ Context loading
  ✅ LLM client
  ✅ Prompt building (adapt for constraints)
```

#### `tools/response_generator.go` (MODIFY: 100 LOC)
```
CURRENT:
- Standalone response generation

REDESIGNED:
- Can be wrapped by ConstrainedResponseGenerator
- Keep: core logic
- Add: option to accept constraints in prompt

LINES CHANGED: ~80
LINES ADDED:   ~30

REUSE:
  ✅ All existing logic (just enhanced)
```

#### `database/migrations/025_add_response_validation_log.sql` (NEW: 50 LOC)
```sql
CREATE TABLE response_validations (
    id UUID PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    response_text TEXT,
    constraints TEXT[],
    violations INT,
    passed BOOLEAN,
    fallback_used BOOLEAN,
    validated_at TIMESTAMP,
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);
```

### TESTS TO ADD

#### `tools/constrained_response_generator_test.go` (ADD: 400 LOC)
```go
TestConstraintBuilding
├─ Test: "User is dominant" → constraint in prompt
├─ Test: "Contact is sensitive" → constraint in prompt
├─ Test: Multiple constraints combined

TestConstrainedGeneration
├─ Test: LLM respects constraints (low violation rate)
├─ Test: If violated → return error
├─ Test: Fallback if constraint generation fails

TestValidation
├─ Test: Response with contradiction → caught
├─ Test: Valid response → passes
├─ Test: Edge case response → validation clear
```

### FEATURE FLAG

Add to config:
```go
UseConstrainedGeneration bool = false  // Start at 10%, increase to 100%
```

### TOTAL PHASE 3
```
Files Created:     3 (constrained_generator, response_validator, migration)
Files Modified:    2 (conversation_agent, response_generator)
Tests Added:       1 (~400 LOC)
Total New LOC:     ~1000
Total Changed LOC: ~180
Risk Level:        MEDIUM (constrained LLM + fallback paths)
```

---

## Phase 4: Database Schema (3 weeks)

### FILES TO CREATE

#### Multiple migrations (NEW)
```sql
026_create_extraction_artifacts.sql     (100 LOC)
027_create_extraction_entities.sql      (80 LOC)
028_create_principle_violations.sql     (60 LOC)
```

### FILES TO MODIFY

#### `database/repositories.go` (MODIFY: 200 LOC)
```
CURRENT:
- ExtractionRepository uses old schema

REDESIGNED:
- Support both old + new schema (dual-write during transition)
- Add: New methods for locked extraction
- Add: Principle metadata queries

LINES ADDED: ~150
LINES CHANGED: ~100

REUSE:
  ✅ Most existing logic
```

### MIGRATION STRATEGY (DUAL-WRITE)

Week 1: Non-destructive migrations (add columns)
Week 2: Dual-write (write to both old + new)
Week 3: Cutover (read from new, keep old as backup)

### TOTAL PHASE 4
```
Files Created:     3 (migrations)
Files Modified:    1 (repositories)
Total New LOC:     ~240
Total Changed LOC: ~300
Risk Level:        HIGH (data migration)
Mitigation:        Dual-write, rollback migration, staging
```

---

## SUMMARY BY PHASE

| Phase | Duration | Files Changed | New LOC | Risk | Rollback Time |
|-------|----------|---|---|---|---|
| 1A | 2w | 5 | 350 | LOW | 1h |
| 1B | 2w | 4 | 200 | LOW-MED | 1h |
| 2 | 3w | 5 | 1100 | HIGH | < 1h (flag) |
| 3 | 2w | 5 | 1000 | MED | < 1h (flag) |
| 4 | 3w | 4 | 240 | HIGH | 2h |
| **TOTAL** | **9w** | **23** | **2890** | | |

## Files NOT Changing

- ✅ `moly-extension/` (TypeScript frontend)
- ✅ `moly-proxy/` (CORS proxy)
- ✅ `auth/` (authentication)
- ✅ `tools/llm_client.go` (LLM interface)
- ✅ `tools/learning_agent.go` (learning system)
- ✅ Most database layer (just extending)

## Files Completely Removed

- ❌ Multi-pass parsing logic (~380 LOC dead code)
- ❌ LinguisticParser dependency in clarification (no longer used)
- ❌ Layer-by-layer question logic (consolidated into orchestrator)

