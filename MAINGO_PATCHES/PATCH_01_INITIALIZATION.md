# PATCH 1: Main.go Initialization Section
# Location: NewV2APIServer function (around line 88-180)

## Section 1A: Add imports (top of main.go, line ~1-30)

ADD these imports if not present:
```go
import (
    // ... existing imports ...
    
    "moly/config"         // NEW: Feature flags
    "moly/monitoring"     // NEW: Metrics
)
```

---

## Section 1B: Add fields to V2APIServer struct (around line 37-85)

FIND this in the struct:
```go
type V2APIServer struct {
    llmClient               tools.LLMProvider
    database                *database.Database
    // ... other 80+ fields ...
}
```

ADD these new fields AFTER existing fields:
```go
	// NEW: Phase orchestration (Phases 1-4)
	phaseOrchestrator      *agents.PhaseOrchestrator
	
	// NEW: Phase 3 components (Constrained Generation)
	responseValidator      *agents.ResponseValidator
	constrainedResponseGen *tools.ConstrainedResponseGenerator
```

---

## Section 1C: Initialize PhaseOrchestrator in NewV2APIServer (around line 170-180)

FIND this section:
```go
func NewV2APIServer(llm tools.LLMProvider, db *database.Database) (*V2APIServer, error) {
    if db == nil {
        return nil, fmt.Errorf("database cannot be nil")
    }
    
    // ... initialization code ...
    
    extractionPhase := agents.NewExtractionPhase(intentDetector, extractionStore, conflictDetector, db)
```

ADD these lines RIGHT AFTER the extractionPhase initialization:
```go
	// NEW: Initialize Phase Orchestrator for feature flags & monitoring
	phaseOrchestrator := agents.NewPhaseOrchestrator(
		extractionPhase,
		nil, // layer5Handler will be set later
		nil, // responseValidator will be set later
		llm,
		nil, // responseGenerator will be set later
		db,
	)
	log.Printf("[V2APIServer] ✓ Phase orchestrator initialized")
	
	// NEW: Initialize Phase 3 Response Validator
	responseValidator := agents.NewResponseValidator()
	log.Printf("[V2APIServer] ✓ Response validator initialized")
	
	// NEW: Initialize Constrained Response Generator
	constrainedResponseGen := tools.NewConstrainedResponseGenerator(
		llm,
		nil, // Will set responseGenerator reference later
		db,
		responseValidator,
	)
	log.Printf("[V2APIServer] ✓ Constrained response generator initialized")
```

---

## Section 1D: Initialize Layer 5 Conflict Handler (around line 190-210)

FIND where ConversationAgent is created:
```go
conversationAnalyzer := agents.NewConversationAnalyzer(...)
```

ADD BEFORE or AFTER it:
```go
	// NEW: Initialize Layer 5 Conflict Handler (Phase 2)
	layer5Handler := agents.NewLayer5ConflictHandler(llm, db)
	clarificationHistory := agents.NewClarificationHistory(db)
	log.Printf("[V2APIServer] ✓ Layer 5 conflict handler initialized")
```

---

## Section 1E: Wire all components together (at END of NewV2APIServer, before return statement)

FIND the return statement near the end of the function (around line 250+):
```go
return &V2APIServer{
    llmClient: llm,
    database: db,
    // ... other fields ...
}, nil
```

MODIFY it to include new fields:
```go
return &V2APIServer{
    llmClient:                llm,
    database:                 db,
    // ... existing fields ...
    
    // NEW: Phase orchestration
    phaseOrchestrator:        phaseOrchestrator,
    
    // NEW: Phase 3 components
    responseValidator:        responseValidator,
    constrainedResponseGen:   constrainedResponseGen,
}, nil
```

---

## Verification for Patch 1

After applying, verify:
```bash
# Build should still pass
go build ./...

# No compile errors about undefined fields
grep -n "phaseOrchestrator\|responseValidator\|constrainedResponseGen" main.go | head -20

# Should see 3+ occurrences
```

✅ When complete, move to PATCH_02_EXTRACTION_MONITORING.md

