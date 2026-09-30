# PATCH 2: Extraction Monitoring & Phase 1 Feature Flags
# Location: Message processing handler (around line 690-800)

## Section 2A: Add feature flag and metrics initialization

FIND this section (around line 695):
```go
var extractedEntitiesNeedClarification = false
var extractedEntitiesClarificationQ string
var extractionArtifact *models.ExtractionArtifact
var extractionConflicts []agents.ConflictDetectorResult

if processedMessage != "" {
```

MODIFY to ADD feature flags and metrics BEFORE processing:
```go
	var extractedEntitiesNeedClarification = false
	var extractedEntitiesClarificationQ string
	var extractionArtifact *models.ExtractionArtifact
	var extractionConflicts []agents.ConflictDetectorResult

	// NEW: Get feature flags and metrics
	flags := config.GetFeatureFlags()
	metrics := monitoring.GetMetrics()

	if processedMessage != "" {
```

---

## Section 2B: Add extraction time tracking and Phase 1 logging

FIND the extraction call (around line 696-710):
```go
if processedMessage != "" {
    // Call ExtractionPhase (Layer 0) to get extraction + conflict detection + analysis context
    epInput := &agents.ExtractionPhaseInput{
        UserID:         userID,
        ConversationID: req.ConversationID,
        MessageID:      userMessageID,
        Message:        processedMessage,
        MessageCount:   0,
        RecentMessages: []models.Message{},
        UserProfile:    nil,
        Cache:          srv.llmCache,
    }

    epOutput, err := srv.extractionPhase.Run(context.Background(), epInput)
```

MODIFY to ADD timing and Phase 1 logging:
```go
	if processedMessage != "" {
		// NEW: PHASE 1 - EXTRACTION LOCK
		if flags.UseExtractionLock {
			log.Printf("[MessageProcessor] [Phase 1] Extraction lock ENABLED")
		} else {
			log.Printf("[MessageProcessor] [Phase 1] Extraction lock DISABLED (fallback mode)")
		}

		extractStartTime := time.Now()

		// Call ExtractionPhase (Layer 0) to get extraction + conflict detection + analysis context
		epInput := &agents.ExtractionPhaseInput{
			UserID:         userID,
			ConversationID: req.ConversationID,
			MessageID:      userMessageID,
			Message:        processedMessage,
			MessageCount:   0,
			RecentMessages: []models.Message{},
			UserProfile:    nil,
			Cache:          srv.llmCache,
		}

		epOutput, err := srv.extractionPhase.Run(context.Background(), epInput)
		
		// NEW: Record extraction metrics
		extractMs := time.Since(extractStartTime).Milliseconds()
		metrics.RecordExtractionTime(extractMs)
		log.Printf("[MessageProcessor] [Phase 1] Extraction time: %dms", extractMs)
```

---

## Section 2C: Remove re-parsing fallback and add lock verification

FIND this section (around line 710-730):
```go
if err != nil {
    log.Printf("[MessageProcessor] ⚠ Extraction phase failed: %v - continuing without extraction", err)
} else if epOutput != nil && epOutput.Artifact != nil {
```

REPLACE with:
```go
		if err != nil {
			// NEW: PHASE 1 - No fallback with extraction lock
			if flags.UseExtractionLock {
				log.Printf("[MessageProcessor] FATAL: Extraction failed - cannot proceed (Phase 1 enabled)")
				metrics.RecordExtractionLockFailure()
				schema.RespondError(w, http.StatusInternalServerError, "Extraction required but failed")
				return
			} else {
				log.Printf("[MessageProcessor] ⚠ Extraction phase failed: %v - continuing without extraction", err)
			}
		} else if epOutput != nil && epOutput.Artifact != nil {
```

---

## Section 2D: Add lock verification after extraction succeeds

FIND this section (around line 712-720):
```go
} else if epOutput != nil && epOutput.Artifact != nil {
    extractionArtifact = epOutput.Artifact
    extractedEntities = epOutput.Artifact.Entities
    extractionConflicts = epOutput.Conflicts

    log.Printf("[MessageProcessor] ✓ Phase 0 extraction: %d entities, %d conflicts detected",
        len(extractedEntities), len(extractionConflicts))
```

ADD lock verification:
```go
		} else if epOutput != nil && epOutput.Artifact != nil {
			extractionArtifact = epOutput.Artifact
			extractedEntities = epOutput.Artifact.Entities
			extractionConflicts = epOutput.Conflicts

			// NEW: PHASE 1 - Verify artifact is locked
			if flags.UseExtractionLock {
				if !extractionArtifact.IsLocked {
					log.Printf("[MessageProcessor] ERROR: Artifact not locked! (Phase 1 failure)")
					metrics.RecordExtractionLockFailure()
				} else {
					log.Printf("[MessageProcessor] ✓ Artifact locked: %s (Phase 1)", extractionArtifact.ID)
					metrics.RecordExtractionLockSuccess()
				}
			}

			log.Printf("[MessageProcessor] ✓ Phase 0 extraction: %d entities, %d conflicts detected",
				len(extractedEntities), len(extractionConflicts))
```

---

## Section 2E: Log phase status (around line 780+)

FIND the end of extraction handling:
```go
log.Printf("[MessageProcessor] ✓ Extracted %d contacts with subject attribution: %v",
    len(contactsWithSubjects), contactsWithSubjects)
}
```

ADD AFTER this block:
```go
	}

	// NEW: Log phase status after extraction
	if flags.EnableMetrics && flags.EnableDetailedLogging {
		log.Printf("[MessageProcessor] Phase status: phase1=%v, phase2=%v, phase3=%v, phase4=%v",
			flags.UseExtractionLock,
			flags.UseLayer5ConflictGate,
			flags.UseConstrainedResponseGeneration,
			flags.UseCleanSchema,
		)
	}
```

---

## Verification for Patch 2

After applying, test:
```bash
# Build should pass
go build ./...

# Enable Phase 1 and test extraction
export MOLY_USE_EXTRACTION_LOCK=true
export MOLY_ENABLE_DETAILED_LOGGING=true

# Start server and send a message
# Check logs for:
# - "[Phase 1] Extraction lock ENABLED"
# - "[Phase 1] Extraction time: XXXms"
# - "[Phase 1] Artifact locked"
# - "Phase status: phase1=true"
```

✅ When complete, move to PATCH_03_LAYER5_WIRING.md

