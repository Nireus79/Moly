# CODE ARCHAEOLOGY PROTOCOL

**Goal**: Never rewrite existing code, never create duplicates, never assume how code works  
**Strategy**: Audit first, understand existing code, refactor in place, not rewrite  
**Enforcement**: Mandatory pre-implementation audit + code reference documentation

---

## THE DANGER: Duplicate & Assumption Bugs

### Problem 1: Duplicate Methods
```go
// Existing (written years ago):
func (ca *conversationAgent) askClarificationQuestion(q string) error {
    // Some implementation
}

// New (written without checking):
func (ca *conversationAgent) AskClarification(question string) error {
    // Nearly identical implementation
}

// Result: Two methods doing same thing, both called, inconsistent behavior
```

### Problem 2: Assumption Wiring
```go
// Developer assumes ExtractedEntity has a "Principles" field
// Writes code:
func ProcessEntity(e ExtractedEntity) {
    for _, p := range e.Principles {  // ASSUMES this exists
        // ...
    }
}

// But actual ExtractedEntity definition:
type ExtractedEntity struct {
    Value string
    Type string
    // NO "Principles" field!
}

// Result: Code compiles, runs, crashes at runtime
```

### Problem 3: Incomplete Understanding
```go
// Developer sees method name:
func (db *Database) SaveExtraction(artifact *ExtractionArtifact) error

// Assumes: "This saves the extraction artifact to database"
// Writes code that calls it expecting data to be persisted

// But actual implementation:
func (db *Database) SaveExtraction(...) error {
    return db.queueForAsync()  // Only queues! Doesn't save immediately!
}

// Result: Code calls SaveExtraction, assumes it's done, data lost when crash happens
```

### Problem 4: Over-Wrapping
```go
// Existing method:
func (h *Handler) ProcessConflict(c Conflict) error { ... }

// Developer doesn't use it, creates wrapper:
func (h *Handler) HandleConflict(c Conflict) error {
    return h.ProcessConflict(c)  // Just calls the original!
}

// Result: Two methods, one is dead code, maintenance nightmare
```

---

## SOLUTION: CODE ARCHAEOLOGY PROTOCOL

### Phase 0: PRE-IMPLEMENTATION AUDIT (MANDATORY)

**BEFORE WRITING ANY CODE FOR A PHASE**, do this audit:

#### Step 1: Map All Existing Code

**For Phase 2 (Conflict Channeling), search for:**

```bash
# Find all conflict-related code
grep -r "conflict\|Conflict" moly-go --include="*.go" | grep -E "^[^:]+:(func|type|var|const)" | cut -d: -f1-3

# Find all clarification-related code
grep -r "clarif\|Clarif" moly-go --include="*.go" | grep -E "^[^:]+:(func|type)" | cut -d: -f1-3

# Find all question-related code
grep -r "question\|Question" moly-go --include="*.go" | grep -E "^[^:]+:(func|type)" | cut -d: -f1-3

# Result: Document of all existing code
```

**Create a document: `PHASE_2_CODE_AUDIT.md`**

```markdown
# Phase 2 Conflict Channeling - Code Audit

## Existing Conflict-Related Code

### Structures/Types
- [ ] ConflictResult (models/conflict_result.go:42-58)
  - Fields: Type, PreviousValue, CurrentValue, Severity
  - Usage: Returned by ConflictDetector

- [ ] ConflictDetector (agents/conflict_detector.go:15-30)
  - Methods: Detect(), GetConflicts()
  - Called from: ExtractionPhase.Run()

### Methods/Functions
- [ ] ConflictDetector.Detect() (agents/conflict_detector.go:32-85)
  - Input: ExtractionArtifact
  - Output: []ConflictResult
  - Side effects: Logs conflicts
  - WHO CALLS THIS? ExtractionPhase

- [ ] ExtractionPhase.Run() (agents/extraction_phase.go:40-120)
  - Calls: ConflictDetector.Detect() at line 95
  - Returns: conflicts in epOutput.Conflicts
  - WHO USES OUTPUT? MessageProcessor.processMessage() at line 717

### Question-Related Code
- [ ] ClarificationQuestion (models/clarification_question.go:5-20)
  - Fields: ID, QuestionText, Status, UserID
  - Database table: clarification_questions

- [ ] ClarificationQuestionRepository (database/clarification_repository.go:10-50)
  - Methods: SaveQuestion(), GetByConversation()
  - Used by: ConversationAgent

### Related Functionality
- [ ] Layer 4 gap detection (agents/conversation_agent.go:615-655)
  - Asks questions about missing context
  - Saves questions via ClarificationQuestionRepository
  - COULD BE REFACTORED to extract common question-asking logic

## What We DON'T Need to Write
- Questions storage (already have ClarificationQuestionRepository)
- Question retrieval (already implemented)
- Question types (already have ClarificationType field)

## What We NEED to Write
1. Layer5ConflictHandler (NEW - doesn't exist)
2. Link conflicts to clarification questions (not currently done)
3. Deduplication logic (NEW - doesn't exist)
4. Question reason tracking (NEW - needs new schema fields)

## What We Should REFACTOR (Don't rewrite!)
- Question saving: Extract common logic from Layer 4, 6, 7, etc
- Question generation: Abstract LLM prompting logic
```

#### Step 2: Read Actual Code (Don't Assume!)

**For each existing method**, read it completely:

```go
// BEFORE writing Phase 2 code, read this:
// File: agents/conflict_detector.go

func (cd *ConflictDetector) Detect(artifact *ExtractionArtifact) []ConflictResult {
    conflicts := []ConflictResult{}
    
    // This is what it ACTUALLY does:
    // 1. Gets extracted entities from artifact
    entities := artifact.GetEntities()
    
    // 2. Compares against saved facts
    for _, entity := range entities {
        saved, err := cd.db.GetFactByValue(entity.Value)
        if err != nil {
            continue  // Silently skips on error!
        }
        
        // 3. If mismatch, adds to conflicts
        if entity.Value != saved.Value {
            conflicts = append(conflicts, ConflictResult{
                Type: "value_mismatch",
                PreviousValue: saved.Value,
                CurrentValue: entity.Value,
            })
        }
    }
    
    // 4. Returns only value mismatches
    // Does NOT return characteristic mismatches!
    return conflicts
}

// KEY INSIGHT: Only detects value mismatches, not type conflicts
// Developer assumption would be: "Detect all conflicts"
// Reality: "Only detect value mismatches"
// If you call this for type conflicts, you'll miss them!
```

**Create a document: `EXISTING_CODE_REFERENCE.md`**

```markdown
# Existing Code Reference (For Implementation)

## ConflictDetector.Detect()

**Location**: agents/conflict_detector.go:32-65

**Actual behavior** (from reading code):
- Iterates extracted entities
- Compares each against saved facts in database
- Only detects VALUE conflicts (entity.Value != saved.Value)
- Silently skips database errors (continue, no error)
- Returns []ConflictResult with Type="value_mismatch"

**Does NOT do**:
- Detect type conflicts (e.g., "user" vs "contact")
- Detect principle conflicts
- Return characteristic conflicts

**Side effects**:
- Calls database (may be slow)
- Logs nothing

**Error handling**:
- Silently skips on db.GetFact() error
- Never returns error (signature is []ConflictResult, no error return)

**If you want to extend it**:
- DON'T create a new detector
- REFACTOR this one to handle more conflict types
- Add parameter: conflictTypes []string
```

#### Step 3: Map Dependencies & Call Sites

**For each existing method, find who calls it:**

```bash
# Find all callers of ConflictDetector.Detect
grep -r "\.Detect(" moly-go --include="*.go" -A 2 -B 2

# Output should show:
# agents/extraction_phase.go:95: epOutput.Conflicts = cd.Detect(artifact)

# Find all users of ClarificationQuestionRepository
grep -r "ClarificationQuestionRepository\|clariRepo\|clarification_repo" moly-go --include="*.go"
```

**Create a diagram:**

```
USER MESSAGE
    ↓
[MessageProcessor.processMessage()]
    ↓
[ExtractionPhase.Run()]
    ├─ ConflictDetector.Detect() → conflicts
    └─ Returns: epOutput.Conflicts
    ↓
[MessageProcessor] stores conflicts
    ↓
[ConversationAgent.Run()]
    ├─ Receives ctx with conflicts
    ├─ Layer 4: Asks gap questions (saves via ClarificationQuestionRepository)
    ├─ Layer 6: Asks principle questions (saves via ClarificationQuestionRepository)
    └─ (NO Layer 5: Conflict questions not asked currently!)
    
[NEW Phase 2 should insert here]:
    ↓
[Layer5ConflictHandler.ProcessConflicts(conflicts)] (NEW)
    ├─ Takes conflicts from extraction
    ├─ Generates clarification question
    └─ Saves via ClarificationQuestionRepository (REUSE!)
```

---

### Step 1: Document Assumptions (BEFORE Coding)

**Create: `PHASE_2_IMPLEMENTATION_ASSUMPTIONS.md`**

```markdown
# Phase 2 Implementation - Assumptions vs Reality

## Assumption 1: How ConflictDetector works
**What I assume**: It detects all conflicts (types, values, principles)
**What I need to verify**: Read agents/conflict_detector.go line 32-65
**Reality**: Only detects VALUE conflicts, ignores types

**Impact on my code**: 
- I CANNOT assume Detect() will find type conflicts
- I need to extend Detect() to handle CharacteristicConflict
- OR create new method in detector for this

## Assumption 2: How ClarificationQuestionRepository works
**What I assume**: SaveQuestion() persists question immediately
**What I need to verify**: Read database/clarification_repository.go
**Reality**: [READ FIRST]

## Assumption 3: When conflicts are available
**What I assume**: Conflicts available in ConversationAgent.Run()
**What I need to verify**: Trace context flow from MessageProcessor
**Reality**: [READ FIRST]

## Assumption 4: What ExtractedEntity structure contains
**What I assume**: Has Principles, Antonyms fields
**What I need to verify**: Read models/agent_types.go
**Reality**: [READ FIRST - Don't assume!]
```

**RULE**: Do not write ANY code until you verify each assumption against actual code!

---

### Step 2: Code Refactoring Map (Not Rewrite!)

**Identify what can be REUSED vs what needs NEW code:**

```markdown
# Phase 2: Refactor vs New Code Decision

## REFACTOR (Don't rewrite - modify existing)

### ConflictDetector.Detect()
- Current: Only VALUE conflicts
- Extend: Add parameter for conflict types
- Change signature: Detect(artifact, conflictTypes []string) []ConflictResult
- Location: agents/conflict_detector.go:32

### ClarificationQuestionRepository.SaveQuestion()
- Current: Generic save
- Reuse: As-is, no changes needed
- Location: database/clarification_repository.go:15

### Question generation in ConversationAgent
- Current: Layers 4, 6, 7 all generate questions differently
- Refactor: Extract common "generateClarificationQuestion" helper
- Location: agents/conversation_agent.go (multiple methods)

## NEW CODE (Doesn't exist, must create)

### Layer5ConflictHandler
- Why new: No existing conflict-specific handler
- What it does: Takes conflicts, generates Layer-5-specific questions
- Location: agents/layer5_conflict_handler.go (NEW)
- Dependencies: Uses ConflictDetector, LLM, ClarificationQuestionRepository

### Clarification deduplication
- Why new: Not currently tracked
- What it does: Checks if similar question recently asked
- Location: agents/clarification_history.go (NEW)
- Dependencies: ClarificationQuestionRepository

### Conflict → Question linking
- Why new: Not currently tracked
- What it does: Links conflicts to their clarification questions
- Location: models/clarification_question.go (ADD FIELD) + database schema
- Dependencies: Existing tables
```

---

### Step 3: Explicit Code Interface (Contract First)

**BEFORE coding implementation, define what you're calling:**

```go
// DEFINE: What existing code I'm going to use
package agents

// I will call these existing functions:

// 1. ConflictDetector (existing, to be extended)
type ConflictDetector interface {
    // Extended signature (currently doesn't have conflictTypes param)
    Detect(artifact *ExtractionArtifact, conflictTypes []string) []ConflictResult
}

// 2. ClarificationQuestionRepository (existing, reuse as-is)
type ClarificationRepository interface {
    SaveQuestion(q *database.ClarificationQuestion) error
    GetByConversation(convID string) ([]database.ClarificationQuestion, error)
}

// 3. LLM client (existing, reuse as-is)
type LLMProvider interface {
    Call(ctx context.Context, prompt string) (string, error)
}

// DEFINE: What I'm creating new
type Layer5ConflictHandler struct {
    conflictDetector ConflictDetector           // Will use existing (extended)
    clarificationRepo ClarificationRepository    // Will use existing (as-is)
    llmClient        LLMProvider                 // Will use existing (as-is)
}

func (lch *Layer5ConflictHandler) ProcessConflicts(...) (*ClarificationQuestion, error) {
    // Will implement new logic here
    // But reuse existing components above
}
```

**Document the contract:**

```markdown
# Phase 2: Code Dependencies Map

## What We REUSE (existing code)

### ConflictDetector.Detect()
- Current file: agents/conflict_detector.go
- Current signature: Detect(artifact) []ConflictResult
- How we'll use it: Call with conflicts from ExtractionPhase
- Changes needed: Add conflictTypes parameter (REFACTOR)
- Risk: None (we're extending, not replacing)

### ClarificationQuestionRepository
- Current file: database/clarification_repository.go
- Current methods: SaveQuestion(), GetByConversation()
- How we'll use it: Save Layer 5 questions same way as Layer 4
- Changes needed: None
- Risk: Low (it already does what we need)

### LLM Client
- Current file: tools/llm_client.go
- Current method: Call(ctx, prompt)
- How we'll use it: Generate Layer 5 questions
- Changes needed: None
- Risk: Low (established pattern)

## What We CREATE (new code)

### Layer5ConflictHandler
- New file: agents/layer5_conflict_handler.go
- Depends on: ConflictDetector, ClarificationRepository, LLM
- Does: Processes conflicts, generates questions
- Risk: Medium (new component, needs wiring)

### ClarificationHistory (Deduplication)
- New file: agents/clarification_history.go
- Depends on: ClarificationRepository
- Does: Tracks which questions asked, prevents repeats
- Risk: Low (mostly queries existing database)
```

---

### Step 4: Before-Writing Checklist

**BEFORE writing code for Phase 2, verify:**

```
PHASE 2 PRE-IMPLEMENTATION CHECKLIST

Code Archaeology:
- [ ] Read agents/conflict_detector.go completely
- [ ] Read database/clarification_repository.go completely
- [ ] Read agents/conversation_agent.go (gaps, principles sections)
- [ ] Understand how questions are currently saved (Layer 4, 6, 7)
- [ ] Document what ConflictDetector actually does vs assumptions

Dependency Mapping:
- [ ] Map all conflict-related code (grep results documented)
- [ ] Map all clarification-related code (grep results documented)
- [ ] Trace: ExtractionPhase → MessageProcessor → ConversationAgent
- [ ] Find: All places ClarificationQuestionRepository is used

Interface Definition:
- [ ] Define what existing code we'll reuse
- [ ] Define what code we'll extend (refactor)
- [ ] Define what code is NEW
- [ ] Document any assumption changes to existing signatures

Verification:
- [ ] Can ConflictDetector handle our conflict types? If not, extend it
- [ ] Does ClarificationQuestionRepository meet our needs? If not, extend it
- [ ] Will Layer 5 questions work with existing Question schema? If not, extend schema
- [ ] Are there existing "question generation" patterns to follow? If yes, use them

Decision:
- [ ] Do we need NEW Layer5ConflictHandler? (Yes, doesn't exist)
- [ ] Do we need NEW deduplication? (Yes, doesn't exist)
- [ ] Do we need to REFACTOR ConflictDetector? (Yes, only does value conflicts)
- [ ] Do we need to REFACTOR question saving? (Maybe, extract common pattern)
- [ ] Do we reuse ClarificationQuestionRepository? (Yes, save Layer 5 questions there)
```

---

## IMPLEMENTATION: NO ASSUMPTIONS

### During Coding Phase 2:

**When you write Layer5ConflictHandler, reference the code:**

```go
package agents

// Layer5ConflictHandler processes conflicts into clarification questions
// BASED ON ACTUAL CODE READING:
// - ConflictDetector returns []ConflictResult (see conflict_detector.go:32)
// - ClarificationQuestionRepository.SaveQuestion exists (see clarification_repository.go:15)
// - Questions generated via LLM (pattern from conversation_agent.go:670)
type Layer5ConflictHandler struct {
    db *database.Database
    llmClient tools.LLMProvider
    
    // NOT creating new "question saver" - will use existing ClarificationQuestionRepository
    // (verified from reading line database/clarification_repository.go:15)
}

func (lch *Layer5ConflictHandler) ProcessConflicts(
    ctx context.Context,
    conflicts []ConflictResult,
    userID string,
    conversationID string,
) (*ClarificationQuestion, error) {
    // REUSE: Get repository from db (same pattern as Layer 4)
    repo := lch.db.GetClarificationQuestionRepository()
    
    // DON'T assume SaveQuestion works - read it first
    // (verified from reading clarification_repository.go - it works as expected)
    
    // Generate question using LLM (same pattern as conversation_agent.go:670)
    question := lch.generateQuestion(ctx, conflicts[0])
    
    // Save using EXISTING repository (don't create new SaveConflictQuestion)
    dbQuestion := &database.ClarificationQuestion{
        // Fields verified from database/clarification_repository.go
        ID: question.ID,
        QuestionText: question.Question,
        Status: "pending",
    }
    
    return repo.SaveQuestion(dbQuestion)
}

// Document WHICH existing code you're calling
// comment above function explains why we call it this way
```

### Code Comments Showing References:

```go
// Every method that calls existing code should reference it:

// ProcessConflicts calls ConflictDetector.Detect (agents/conflict_detector.go:32)
// which returns only VALUE conflicts, so we extend it to handle CHARACTERISTIC conflicts
conflicts := lch.detectConflicts(ctx, artifact)  // Uses existing detector logic

// Save question using ClarificationQuestionRepository (database/clarification_repository.go:15)
// which already handles all question types, no duplication needed
repo.SaveQuestion(dbQuestion)  // Reuse existing save

// Generate question via LLM (same pattern as conversation_agent.go:670-695)
// Don't create new question generator, follow existing pattern
question := lch.generateQuestionWithLLM(ctx, conflict)
```

---

## REVIEW GATE: Verify No Duplication

**Code review checklist for Phase 2:**

```
REVIEW: Prevent Duplicates & Assumptions

Code Reuse Verification:
- [ ] grep -r "ProcessConflicts" moly-go | Only Layer5ConflictHandler
- [ ] grep -r "GenerateQuestion" moly-go | One pattern (from conversation_agent)
- [ ] ClarificationQuestionRepository.SaveQuestion used for Layer 5 (no new SaveQuestion)
- [ ] No new "conflict handler" created if one exists already

Existing Code Usage:
- [ ] ConflictDetector used (not reimplemented)
- [ ] ClarificationQuestionRepository used (not duplicated)
- [ ] LLM client used (same pattern as existing)
- [ ] Question generation follows existing pattern (conversation_agent.go)

Refactoring Verification:
- [ ] If ConflictDetector extended, old callers still work (backward compatible)
- [ ] No change to ClarificationQuestion schema (extends existing, not replaces)
- [ ] Database migrations are ADDITIVE (add fields, don't remove)

Assumption Verification:
- [ ] Every assumption about existing code has code reference
- [ ] Comments cite file:line of existing code being reused
- [ ] No "I assume X works this way" without verification
- [ ] If assumption was wrong, existing code was refactored, not duplicated
```

---

## SUMMARY: How to Prevent Duplicates & Assumptions

| Problem | Prevention | Enforcement |
|---------|-----------|---|
| Don't know existing code exists | Pre-implementation audit (grep all) | Code review: grep for duplicates |
| Assume how code works | Read it completely before coding | Code review: cite file:line references |
| Rewrite instead of refactor | Create refactor map before coding | Code review: verify refactor not rewrite |
| Create duplicate methods | Document "I will reuse X" before coding | Code review: grep for identical methods |
| Miss existing repo/utility | Map dependencies before coding | Tests fail if not using existing |
| Wrong assumptions cause bugs | Define contract before coding | Compiler catches interface mismatches |

**Golden Rule**: "Read the actual code first. Reference it in your code. Never assume. Extend, don't duplicate."

