# main.go Integration Guide - Phases 1-4 Wiring

**Status**: Implementation guide for wiring feature flags and monitoring into main.go  
**File**: `moly-go/main.go` (5,101 lines)  
**Changes**: ~400 LOC of additions + removals

---

## Overview

This guide shows EXACTLY where to add feature flag checks, monitoring calls, and remove re-parsing logic in main.go.

---

## Section 1: Initialization (Around line 88)

### Location: NewV2APIServer function

**Current code** (line 173-174):
```go
extractionPhase := agents.NewExtractionPhase(intentDetector, extractionStore, conflictDetector, db)
```

**ADD AFTER** (new lines):
```go
// NEW: Initialize Phase Orchestrator for feature flags & monitoring
phaseOrchestrator := agents.NewPhaseOrchestrator(
    extractionPhase,
    nil, // layer5Handler will be set later
    nil, // responseValidator will be set later
    llm,
    responseGenerator,
    db,
)
log.Printf("[V2APIServer] ✓ Phase orchestrator initialized")

// NEW: Store orchestrator in server for use in handlers
srv.phaseOrchestrator = phaseOrchestrator
```

**Add to V2APIServer struct** (around line 85):
```go
type V2APIServer struct {
    // ... existing fields ...
    
    // NEW: Phase orchestration
    phaseOrchestrator *agents.PhaseOrchestrator
}
```

---

## Section 2: Message Processing - Extraction (Line 696)

### Location: `/api/v2/conversations/:id/messages` POST handler

**Current code** (lines 696-783):
```go
if processedMessage != "" {
    // Call ExtractionPhase (Layer 0)
    epInput := &agents.ExtractionPhaseInput{...}
    epOutput, err := srv.extractionPhase.Run(context.Background(), epInput)
    if err != nil {
        log.Printf("[MessageProcessor] ⚠ Extraction phase failed: %v - continuing without extraction", err)
    } else if epOutput != nil && epOutput.Artifact != nil {
        // ... handle extraction ...
    }
}
```

**MODIFY** (ADD monitoring calls):
```go
if processedMessage != "" {
    // PHASE 1: EXTRACTION LOCK
    flags := config.GetFeatureFlags()
    metrics := monitoring.GetMetrics()
    
    extractStartTime := time.Now()
    
    // Call ExtractionPhase (Layer 0)
    epInput := &agents.ExtractionPhaseInput{...}
    
    if flags.UseExtractionLock {
        log.Printf("[MessageProcessor] [Phase 1] Extraction lock ENABLED")
    } else {
        log.Printf("[MessageProcessor] [Phase 1] Extraction lock DISABLED")
    }
    
    epOutput, err := srv.extractionPhase.Run(context.Background(), epInput)
    
    // NEW: Record extraction metrics
    extractMs := time.Since(extractStartTime).Milliseconds()
    metrics.RecordExtractionTime(extractMs)
    
    if err != nil {
        log.Printf("[MessageProcessor] ⚠ Extraction phase failed: %v - NO FALLBACK", err)
        // REMOVE: Old fallback code that tried LinguisticParser
        // Just return error now that we have extraction lock
        if flags.UseExtractionLock {
            schema.RespondError(w, http.StatusInternalServerError, "Extraction required but failed")
            return
        }
    } else if epOutput != nil && epOutput.Artifact != nil {
        extractionArtifact = epOutput.Artifact
        extractedEntities = epOutput.Artifact.Entities
        extractionConflicts = epOutput.Conflicts
        
        // NEW: Verify artifact is locked (Phase 1)
        if flags.UseExtractionLock && !extractionArtifact.IsLocked {
            log.Printf("[MessageProcessor] ERROR: Artifact not locked!")
            metrics.RecordExtractionLockFailure()
        } else if flags.UseExtractionLock {
            log.Printf("[MessageProcessor] ✓ Artifact locked: %s", extractionArtifact.ID)
            metrics.RecordExtractionLockSuccess()
        }
        
        // ... rest of extraction handling ...
    }
}
```

---

## Section 3: Layer 5 Wiring (Around line 800-900)

### Location: ConversationAgent initialization

**ADD** (in NewV2APIServer):
```go
// NEW: Initialize Layer 5 Conflict Handler (Phase 2)
layer5Handler := agents.NewLayer5ConflictHandler(llm, db)
clarificationHistory := agents.NewClarificationHistory(db)

// Wire into conversation agent
if conversationAgent != nil {
    conversationAgent.SetLayer5ConflictHandler(layer5Handler)
    log.Printf("[V2APIServer] ✓ Layer 5 conflict handler initialized")
}

// Update phase orchestrator
phaseOrchestrator.SetLayer5Handler(layer5Handler)
```

### Location: ConversationAgent.Run (internal method)

**ADD** (after Layer 4, before Layer 6):
```go
// ⭐ [Layer 5] CONFLICT DETECTION & RESOLUTION (Phase 2)
flags := config.GetFeatureFlags()
metrics := monitoring.GetMetrics()

if flags.UseLayer5ConflictGate && ca.layer5Handler != nil && len(epOutput.Conflicts) > 0 {
    log.Printf("[ConversationAgent] [Phase 2] Layer 5 ENABLED - Processing %d conflicts", len(epOutput.Conflicts))
    
    conflictQ, hasQuestion := ca.layer5Handler.ProcessConflicts(
        ctx,
        epOutput.Conflicts,
        ctx.AboutMe.UserID,
        ctx.ConversationID,
    )
    
    if hasQuestion {
        metrics.RecordConflictDetected()
        response.Response = conflictQ.Question
        response.Metadata["layer5Conflict"] = true
        response.Metadata["gate"] = "conflict_resolution"
        return response, nil
    }
} else if !flags.UseLayer5ConflictGate {
    log.Printf("[ConversationAgent] [Phase 2] Layer 5 DISABLED")
}
```

---

## Section 4: Response Generation (Around line 1850)

### Location: Response generation code

**Current code** (approximate):
```go
agentResp, err := responseGenerator.GenerateContextualResponse(
    ctx,
    userMessage,
    ctx.AboutMe,
    ctx.ContactProfiles,
)
```

**MODIFY** (add Phase 3 wiring):
```go
flags := config.GetFeatureFlags()
metrics := monitoring.GetMetrics()

// PHASE 3: CONSTRAINED RESPONSE GENERATION
if flags.UseConstrainedResponseGeneration && srv.constrainedResponseGen != nil {
    log.Printf("[MessageProcessor] [Phase 3] Constrained generation ENABLED")
    
    generationStartTime := time.Now()
    
    agentResp, err = srv.constrainedResponseGen.Generate(
        ctx,
        userMessage,
        ctx.AboutMe,
        ctx.ContactProfiles,
        ctx.ExtractedContext,
        extractionArtifact,
    )
    
    generationMs := time.Since(generationStartTime).Milliseconds()
    metrics.RecordResponseLatency(generationMs)
    
    // Check if validation was applied
    if agentResp != nil && agentResp.Metadata != nil {
        if fallback, ok := agentResp.Metadata["fallbackToClarity"].(bool); ok && fallback {
            log.Printf("[MessageProcessor] ✓ Response validation failed, using clarification instead")
            metrics.RecordResponseValidationViolation()
        }
    }
} else {
    log.Printf("[MessageProcessor] [Phase 3] Constrained generation DISABLED (using basic generator)")
    
    agentResp, err = responseGenerator.GenerateContextualResponse(
        ctx,
        userMessage,
        ctx.AboutMe,
        ctx.ContactProfiles,
    )
}
```

---

## Section 5: Remove Re-Parsing Logic

### Location: Anywhere LinguisticParser is used

**DELETE all instances of**:
```go
// OLD: Fallback re-parsing (REMOVE THIS)
if extraction failed:
    linguisticParser.Parse(message)  // DELETE
    
// OLD: Multi-pass subject attribution retry (REMOVE THIS)
if entity.Subject == "" {
    reparsedEntity := linguisticParser.Parse(message)  // DELETE
}
```

### Location: database/clarification_capture.go

**AROUND line 150-250**:

**DELETE**:
```go
// OLD: Multi-pass parsing fallback (DELETE THESE BLOCKS)
linguisticParser.Parse(message)
// OLD: Subject attribution retry logic (DELETE)
if entity.Subject == "" {
    // retry with LinguisticParser
}
```

---

## Section 6: Initialization of Constrained Generator

### Location: NewV2APIServer (around line 170)

**ADD**:
```go
// NEW: Initialize Constrained Response Generator (Phase 3)
responseValidator := agents.NewResponseValidator()
constrainedResponseGen := tools.NewConstrainedResponseGenerator(
    llm,
    srv.responseGenerator,
    db,
    responseValidator,
)
srv.constrainedResponseGen = constrainedResponseGen
log.Printf("[V2APIServer] ✓ Constrained response generator initialized")

// Update orchestrator
phaseOrchestrator.SetResponseValidator(responseValidator)
```

**Add to V2APIServer struct**:
```go
type V2APIServer struct {
    // ... existing fields ...
    
    // NEW: Phase 3 components
    responseValidator        *agents.ResponseValidator
    constrainedResponseGen   *tools.ConstrainedResponseGenerator
}
```

---

## Section 7: Monitoring Integration

### Location: Response return (end of handler)

**ADD** (before returning response to client):
```go
// MONITORING: Record endpoint metrics
flags := config.GetFeatureFlags()
metrics := monitoring.GetMetrics()

if flags.EnableMetrics {
    metrics.LogMetrics()
    
    // Record phase-specific metrics
    if flags.UseExtractionLock {
        log.Printf("[Metrics] Phase 1: Extraction lock active")
    }
    if flags.UseLayer5ConflictGate {
        log.Printf("[Metrics] Phase 2: Layer 5 conflict gating active")
    }
    if flags.UseConstrainedResponseGeneration {
        log.Printf("[Metrics] Phase 3: Constrained generation active")
    }
}

// Add metadata to response
if response.Metadata == nil {
    response.Metadata = make(map[string]interface{})
}
response.Metadata["phase1_enabled"] = flags.UseExtractionLock
response.Metadata["phase2_enabled"] = flags.UseLayer5ConflictGate
response.Metadata["phase3_enabled"] = flags.UseConstrainedResponseGeneration
response.Metadata["phase4_enabled"] = flags.UseCleanSchema
```

---

## Summary of Changes

### Files to Modify
1. **main.go** (~400 LOC changes)
   - Add phase orchestrator initialization (20 LOC)
   - Add feature flag checks at 3 key points (150 LOC)
   - Add monitoring calls (100 LOC)
   - Remove re-parsing fallback code (50 LOC deletion)

2. **database/clarification_capture.go** (~100 LOC changes)
   - Remove multi-pass parsing logic
   - Use locked extraction only

### New Field in V2APIServer
```go
type V2APIServer struct {
    // ... existing 85+ fields ...
    
    // Phase orchestration
    phaseOrchestrator *agents.PhaseOrchestrator
    
    // Phase 3 components
    responseValidator *agents.ResponseValidator
    constrainedResponseGen *tools.ConstrainedResponseGenerator
}
```

### Key Integration Points
1. **Line 88**: Initialize PhaseOrchestrator in NewV2APIServer
2. **Line 696**: Add feature flag checks and monitoring around extraction
3. **Line 800**: Wire Layer 5 handler in ConversationAgent
4. **Line 1850**: Add Phase 3 constrained generation checks
5. **Line 5050**: Return response with phase metadata

---

## Testing After Integration

```bash
# 1. Build
cd moly-go && go build ./...

# 2. Test feature flags work
go test ./config -v

# 3. Test monitoring
go test ./monitoring -v

# 4. Test integration
go test ./agents -run "Phase" -v

# 5. Manual verification
# - Check logs show "Phase X ENABLED/DISABLED"
# - Check metrics are collected
# - Try toggling flags with env vars
```

---

## Verification Checklist

After wiring:

- [ ] Build passes: `go build ./...` ✅
- [ ] All tests pass: `go test ./...` ✅
- [ ] Feature flag logs appear: "Phase 1 ENABLED"
- [ ] Monitoring calls work: metrics collected
- [ ] Extraction lock enforced: log shows "locked"
- [ ] No re-parsing fallback: old code removed
- [ ] Response validator called: metadata shows validation status
- [ ] Layer 5 questions generated: conflict questions appear
- [ ] Metrics available: call `GetMetrics()` endpoint

---

**This completes the main.go wiring guide.**  
**Next: Create deployment scripts and final verification.**
