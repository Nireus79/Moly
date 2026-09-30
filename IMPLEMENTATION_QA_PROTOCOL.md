# MOLY IMPLEMENTATION: Quality & Integration Protocol

**Goal**: Ensure no unwired code, no mismatches, no half-implemented features  
**Strategy**: Continuous integration + compiler discipline + incremental wiring  
**Enforcement**: Automated checks + manual code review + build gates

---

## PROBLEM: HOW IMPLEMENTATIONS FAIL

### Common Failures (We Must Prevent)

**Problem 1: Dead Code**
```go
// Written but never called
func (ca *conversationAgent) layer5Handler() { ... }
// But in Run() method, never called:
// if ca.layer5Handler != nil { ca.layer5Handler.Process() }  // MISSING
```

**Problem 2: Name Mismatches**
```go
// Defined as:
func (ch *ClarificationHandler) GenerateQuestion() string { ... }

// But called as:
q := handler.GetQuestion()  // ERROR: method not found
```

**Problem 3: Unwired Components**
```go
// New struct created:
type Layer5ConflictHandler struct { ... }

// But in ConversationAgent:
ca.layer5Handler = nil  // Never initialized
// Then in code:
if ca.layer5Handler != nil { ca.layer5Handler.Process() }  // Always nil!
```

**Problem 4: Type Mismatches**
```go
// Returns:
func GetConflicts() []ConflictResult { ... }

// But called expecting different type:
conflicts := handler.GetConflicts()
conflicts[0].UserCharacteristic  // ERROR: field doesn't exist
```

**Problem 5: Incomplete Refactoring**
```go
// Changed function signature:
func (ca *conversationAgent) Run(ctx Context, extraction *LockedExtraction) error

// But old callsites still pass old signature:
err := agent.Run(ctx)  // Missing extraction parameter
```

---

## SOLUTION: MULTI-LAYER QUALITY PROTOCOL

### Layer 1: Compiler Enforcement (ZERO TOLERANCE)

**Rule**: Code must compile with ZERO warnings

```bash
# Before every commit:
go build ./...           # Must succeed, zero errors
go build -race ./...     # Race detector
go vet ./...             # Vet checks
```

**Unused Code Detection**:
```bash
# Find dead code
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run ./...  # Reports unused functions, variables

# Find unused imports
go mod tidy              # Must pass
```

**Enforce in CI**:
```yaml
# .github/workflows/build.yml
- name: Build
  run: go build ./...
  
- name: Vet
  run: go vet ./...
  
- name: Lint
  run: golangci-lint run ./moly-go/...
  
- name: No dead code
  run: |
    if grep -r "TODO.*implement\|FIXME\|placeholder" moly-go/*.go; then
      echo "Found TODO/FIXME - must implement"
      exit 1
    fi
```

---

### Layer 2: Interface Contracts (BEFORE Implementation)

**Strategy**: Define interfaces FIRST, then implement

**Example - Phase 2 (Layer 5 Conflict Channeling)**:

```go
// Step 1: Define the interface (before writing implementation)
package agents

// Layer5ConflictHandler must implement this interface
type ConflictHandler interface {
    // Process conflicts and return clarification question
    ProcessConflicts(ctx context.Context, conflicts []ConflictResult) (*ClarificationQuestion, error)
    
    // Generate question from conflict
    GenerateConflictQuestion(ctx context.Context, conflict ConflictResult) (*ClarificationQuestion, error)
    
    // Track which conflicts were already asked
    WasRecentlyAsked(userID string, conflictID string) (bool, error)
}
```

**Then implement**:
```go
// Step 2: Implement the interface
type Layer5ConflictHandler struct { ... }

func (lch *Layer5ConflictHandler) ProcessConflicts(...) (*ClarificationQuestion, error) {
    // Implementation
}

// Compiler will ERROR if we miss a method!
// "Layer5ConflictHandler does not implement ConflictHandler"
```

**Benefit**: Compiler catches incomplete implementations immediately

---

### Layer 3: Incremental Wiring (ONE COMPONENT AT A TIME)

**Strategy**: Wire each component as implemented, test immediately

**Do NOT**:
- Implement all 4 phases before testing any
- Create stubs that "will be filled in later"
- Have components that aren't called

**DO**:
1. Implement Phase 1 (Extraction Lock)
2. Wire it into main.go
3. Test it (unit + integration)
4. Commit it
5. Then start Phase 2

**Example - Phase 1 Wiring**:

```go
// Step 1: Create handler
extractionLock := tools.NewExtractionLock()

// Step 2: Wire into main processing
func (m *MessageProcessor) processMessage(...) {
    // Use extraction lock
    lockedArtifact, err := extractionLock.LockExtraction(artifact)
    if err != nil {
        return nil, err
    }
    
    // Verify lock worked
    if !lockedArtifact.IsLocked {
        return nil, fmt.Errorf("extraction not locked - implementation incomplete")
    }
    
    // Only proceed if locked
    // This FORCES wiring before code works
}
```

**Verification at every step**:
```bash
# After Phase 1:
go test ./...                    # All tests pass
go build ./...                   # Compiles
./moly-go -test-extraction-lock  # Manual test
```

---

### Layer 4: Dependency Injection Verification

**Strategy**: Verify all dependencies are injected

**Pattern**:
```go
// Constructor that REQUIRES all dependencies
func NewConversationAgent(
    llm tools.LLMProvider,           // REQUIRED
    db *database.Database,           // REQUIRED
    layer5Handler *Layer5ConflictHandler,  // REQUIRED (not optional!)
) (*conversationAgent, error) {
    
    // Verify all dependencies
    if llm == nil {
        return nil, fmt.Errorf("llm is required")
    }
    if db == nil {
        return nil, fmt.Errorf("database is required")
    }
    if layer5Handler == nil {
        return nil, fmt.Errorf("layer5Handler is required - wiring incomplete")
    }
    
    agent := &conversationAgent{
        llmClient:         llm,
        db:                db,
        layer5Handler:     layer5Handler,  // Wired!
    }
    
    return agent, nil
}
```

**Benefit**: Constructor panics if anything isn't wired

---

### Layer 5: Integration Tests (MANDATORY)

**Strategy**: Every new component has integration test that proves it's wired

**Example - Phase 2 Integration Test**:

```go
func TestLayer5ConflictChanneling(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    llm := setupTestLLM(t)
    
    // Create Layer 5 handler
    handler := agents.NewLayer5ConflictHandler(llm, db)
    if handler == nil {
        t.Fatal("Layer5ConflictHandler not initialized - wiring failed")
    }
    
    // Create conversation agent WITH layer 5 handler
    agent, err := agents.NewConversationAgent(llm, db, handler)
    if err != nil {
        t.Fatalf("ConversationAgent initialization failed: %v", err)
    }
    
    // Test: conflict detection → question generation
    extractionWithConflict := &ExtractionArtifact{
        Conflicts: []ConflictResult{
            {
                ID: "conflict_1",
                PreviousValue: "I'm dominant",
                CurrentValue: "I'm submissive",
                Type: "characteristic_conflict",
            },
        },
    }
    
    // Run conversation
    ctx := models.Context{
        ConversationID: "test_conv",
        ExtractedContext: &models.ExtractedContext{
            Conflicts: extractionWithConflict.Conflicts,
        },
    }
    
    response, err := agent.Run(ctx)
    if err != nil {
        t.Fatalf("ConversationAgent.Run failed: %v", err)
    }
    
    // Verify: conflict resulted in question
    if response.Response == "" {
        t.Fatal("No clarification question generated from conflict")
    }
    
    if !strings.Contains(response.Response, "said") {
        t.Fatal("Question doesn't reference conflict")
    }
    
    // Verify: question was saved to database
    questions, err := db.GetClarificationQuestionRepository().GetByConversation("test_conv")
    if err != nil || len(questions) == 0 {
        t.Fatal("Conflict question not saved to database - wiring incomplete")
    }
    
    t.Logf("✓ Layer 5 conflict handling wired correctly: conflict → question")
}
```

**Benefit**: Test fails if component isn't wired

---

### Layer 6: Code Review Checklist (MANDATORY REVIEW)

**Before accepting any PR**:

```
WIRING CHECKLIST:
- [ ] Component has constructor or factory
- [ ] All dependencies injected (none are nil)
- [ ] Component called from expected location
- [ ] Compiler finds all call sites (grep verification)
- [ ] Integration test proves wiring works
- [ ] No "TODO - implement later" comments
- [ ] Method signatures match all callers
- [ ] Return types match all callers
- [ ] Error handling consistent across all call sites

NAMING CHECKLIST:
- [ ] Method names are accurate (if it does X, it's called "DoX")
- [ ] Method names consistent (don't have Get* and Fetch* for same operation)
- [ ] Type names clear (ConflictHandler not Handler or Manager)
- [ ] Variable names don't shadow outer scope
- [ ] No mismatches between definition and usage

COMPLETENESS CHECKLIST:
- [ ] Function body is complete (no "..."  placeholders)
- [ ] All branches implemented (no "default: return nil")
- [ ] Error cases handled
- [ ] Happy path tested
- [ ] Error path tested

REFACTORING CHECKLIST:
- [ ] Changed signature? All callers updated
- [ ] Moved function? grep confirms no old location called
- [ ] Renamed variable? IDE refactor used (not manual)
- [ ] Deleted function? No callers remain
```

---

### Layer 7: Build Gate (CI/CD)

**Every commit must pass**:

```bash
# 1. Compilation (REQUIRED)
go build ./...                    # Exit 0 or fail

# 2. Tests (REQUIRED)
go test ./...                     # 100% pass rate

# 3. Lint (REQUIRED)
golangci-lint run ./...           # No issues

# 4. Dead code check (REQUIRED)
go install github.com/golangci/golangci-lint@latest
golangci-lint run --disable-all --enable deadcode,unused ./...

# 5. Coverage (REQUIRED for new code)
go test -cover ./...              # New functions > 80% coverage
```

**In CI/CD**:
```yaml
# .github/workflows/merge-gate.yml
on: [pull_request]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Build
        run: go build ./...
        
      - name: Test
        run: go test ./... -v
        
      - name: Vet
        run: go vet ./...
        
      - name: Lint
        run: golangci-lint run ./moly-go/...
        
      - name: Check dead code
        run: |
          golangci-lint run --disable-all --enable deadcode,unused ./moly-go/...
      
      - name: Coverage
        run: go test -cover ./... | grep -E "coverage: [0-9]+" | awk '{print $NF}'
        
  block-merge:
    if: failure()
    runs-on: ubuntu-latest
    steps:
      - name: Block merge
        run: |
          echo "PR blocked: Build, test, or lint failed"
          exit 1
```

**Benefit**: Cannot merge incomplete code

---

## IMPLEMENTATION DISCIPLINE PER PHASE

### Phase 1 (Extraction Lock)

**Deliverables**:
1. ✅ `models/extraction_artifact.go` - Lock() method implemented
2. ✅ `agents/intent_detector.go` - ExtractAndLock() implemented
3. ✅ `tools/extraction_store.go` - GetLocked() implemented
4. ✅ `main.go` - Uses locked extraction (old re-parsing removed)
5. ✅ `database/clarification_capture.go` - Refactored to use locked only
6. ✅ All tests pass (15 new unit tests + 10 integration tests)
7. ✅ Compiler: zero warnings
8. ✅ Coverage: new code > 80%
9. ✅ No dead code detected
10. ✅ All old re-parsing paths removed (grep confirms)

**Verification**:
```bash
# After Phase 1
go build ./...                          # Must succeed
go test ./... -v                        # All 50+ tests pass
golangci-lint run ./moly-go/...        # Zero issues
grep -r "LinguisticParser\|re-parse\|fallback" ./moly-go/ | wc -l  # Should be 0
grep -r "extractArtifact.Entities =" ./moly-go/ | wc -l  # Should find 0 assignments
```

### Phase 2 (Conflict Channeling)

**Deliverables**:
1. ✅ `agents/layer5_conflict_handler.go` - Implemented
2. ✅ `agents/clarification_history.go` - Implemented
3. ✅ `agents/conversation_agent.go` - Layer 5 gate added (calls handler)
4. ✅ `models/conflict_result.go` - Enhanced
5. ✅ Database schema updated
6. ✅ All tests pass (20 new unit tests + 10 integration tests)
7. ✅ Integration test proves: conflict → question
8. ✅ Compiler: zero warnings
9. ✅ Layer5ConflictHandler actually called in Run() method

**Verification**:
```bash
# After Phase 2
grep -n "layer5Handler.ProcessConflicts" ./moly-go/agents/conversation_agent.go  # Must find call
go test ./agents/... -run TestLayer5 -v  # Integration test proves wiring
go vet ./...  # No issues
```

### Phase 3 (Constrained Generation)

**Deliverables**:
1. ✅ `tools/constrained_response_generator.go` - Implemented
2. ✅ `models/constraint.go` - Implemented
3. ✅ `tools/response_generator.go` - Enhanced with SystemPrompt variant
4. ✅ `agents/conversation_agent.go` - Uses constrained generator
5. ✅ Feature flag works (can toggle on/off)
6. ✅ All tests pass (15 unit + 20 integration)
7. ✅ Caching working (hit rate > 70%)
8. ✅ Performance benchmarks show < 10% latency increase

**Verification**:
```bash
# After Phase 3
grep -n "constrainedGen.Generate" ./moly-go/agents/conversation_agent.go  # Must find call
go test ./tools/... -run TestConstrained -v  # Integration test
go test -bench=Constraint ./...  # Performance benchmark
./moly-go -validate-cache-hit-rate  # Should show > 70%
```

### Phase 4 (Clean Schema)

**Deliverables**:
1. ✅ `database/migrations/030_create_clean_schema.sql` - Schema created
2. ✅ `database/migration_job.go` - Export/import implemented
3. ✅ All repositories updated to use new schema
4. ✅ Migration tested on database copy
5. ✅ Zero data loss (verified)
6. ✅ All queries work on new schema
7. ✅ No old schema references remain

**Verification**:
```bash
# After Phase 4
grep -r "old_clarification_questions\|v1_schema" ./moly-go/ | wc -l  # Should be 0
go test ./database/... -run Migration -v  # Test migration job
# Database check: SELECT COUNT(*) FROM clarification_questions; -- Should match old DB
```

---

## RUNTIME SAFETY CHECKS

### Panic on Incomplete Initialization

```go
// Every component checks itself on creation
func NewConversationAgent(...) (*conversationAgent, error) {
    agent := &conversationAgent{...}
    
    // Panic if anything not initialized (catches wiring bugs FAST)
    if err := agent.verify(); err != nil {
        panic(fmt.Sprintf("ConversationAgent not properly initialized: %v", err))
    }
    
    return agent, nil
}

func (ca *conversationAgent) verify() error {
    if ca.llmClient == nil {
        return fmt.Errorf("llmClient not initialized")
    }
    if ca.layer5Handler == nil {
        return fmt.Errorf("layer5Handler not initialized - wiring incomplete")
    }
    if ca.constrainedGen == nil {
        return fmt.Errorf("constrainedGen not initialized - wiring incomplete")
    }
    return nil
}
```

### Runtime Assertions

```go
// In critical paths, assert wiring is correct
func (ca *conversationAgent) Run(ctx Context) (*ConversationResponse, error) {
    // Panic if something that should be wired isn't
    if ca.layer5Handler == nil {
        panic("layer5Handler is nil - component not wired during initialization")
    }
    
    // Process...
}
```

---

## SUMMARY: HOW TO ENSURE QUALITY

| Prevention Layer | Mechanism | Catch Time | Strictness |
|---|---|---|---|
| 1. Compiler | Build fails on dead code | Compile time | ZERO TOLERANCE |
| 2. Interfaces | Type mismatch errors | Compile time | ZERO TOLERANCE |
| 3. Dependency Injection | Constructor requires all deps | Init time | PANIC |
| 4. Integration Tests | Test proves wiring | Test time | FAIL IF NOT WIRED |
| 5. Code Review | Checklist verification | Review time | BLOCK MERGE |
| 6. CI/CD Gate | Build, test, lint, dead-code | CI time | BLOCK COMMIT |
| 7. Runtime Checks | Panic on incomplete init | Runtime | IMMEDIATE CRASH |

**Result**: Impossible to have:
- ✗ Unwired code (integration tests fail)
- ✗ Name mismatches (compiler catches)
- ✗ Half-implemented (tests fail)
- ✗ Dead code (linter catches)
- ✗ Incomplete refactoring (grep + compiler)

**Discipline Rule**: "If it compiles, passes tests, and passes review - it's wired correctly."

