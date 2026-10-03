# PHASE 4: Main.go Wiring - Quick Start

**Status:** Ready to implement  
**Scope:** Wire phase-based maturity system into main message processing pipeline  
**Effort:** ~2-3 hours  
**Key Files:**
- `moly-go/main.go` - Message processor handler (line ~518+)
- `storage/maturity_service.go` - NEW: Maturity calculations
- `models/phase_*.go` - NEW: Phase definitions and accomplishments

---

## What Phase 4 Does

Replaces the old binary maturity system with Socratic accomplishment-based phases:

| OLD (Broken) | NEW (Phase 4) |
|---|---|
| 8-category binary (1.0/0.0) | 4 conversation phases (Initial → Gathering → Analysis → Help) |
| Never improved | Improves each message via accomplishments |
| Arbitrary gates (0.3 cap) | 7 phase-aware gates, no arbitrary caps |
| DB loading disabled | Full DB persistence + phase tracking |

---

## Quick Checklist

- [ ] **Gate 1:** Load maturityCalc at message start (line ~667)
- [ ] **Gate 2:** Track accomplishments before orchestrator (line ~1800+)
- [ ] **Gate 3:** Recalculate phase after orchestrator (line ~3100+)
- [ ] **Gate 4:** Save phase progression to DB (line ~3200+)
- [ ] **Gate 5:** Use phase in response generation (line ~3500+)
- [ ] **Integration Test:** Single conversation, verify maturity improves M1→M2→M3
- [ ] **Build Check:** `go build ./...` passes cleanly

---

## Key Integration Points

### 1. Load Maturity (Already Done ✓)
```go
maturityCalc, matErr := srv.maturityService.LoadOrCreateMaturityContext(userID, req.ConversationID)
if matErr != nil { ... }
initialContextMaturity = maturityCalc.CalculateOverallMaturity()
```

### 2. Track Accomplishments (TO DO)
After orchestrator runs but before response generation:
```go
accomplishments := &models.PhaseAccomplishments{
    ExtractedConfidence: extractedContext.Confidence,
    ClarificationsAnswered: len(clarificationsFilled),
    ConflictsResolved: len(conflictsResolved),
    // ...
}
maturityCalc.RecordAccomplishments(accomplishments)
```

### 3. Update Phase (TO DO)
```go
newPhase := maturityCalc.EstimatePhase()
if newPhase != currentPhase {
    // Save phase transition to DB
    // Include in response metadata
}
```

### 4. Include in Response (TO DO)
```json
{
  "response": "...",
  "metadata": {
    "maturity": 0.65,
    "phase": "gathering",
    "phaseProgression": "initial→gathering",
    "accomplishments": {...}
  }
}
```

---

## Success Criteria

✅ Single message improves maturity from M1 to M2  
✅ Phase transitions (initial → gathering) detected  
✅ Build passes: `go build ./... && go test ./...`  
✅ No nil dereferences or race conditions  
✅ End-to-end test: Message flow produces valid response  

---

## See Also

- `phase-4-integration-guide.md` - Detailed line-by-line wiring
- `ARCHITECTURE.md` - System data flow
- `storage/maturity_service.go` - Implementation reference

---

**Next:** Phase 5 (Safety Checks)
