# PATCH 3: Layer 5 Conflict Channeling Wiring
# Location: ConversationAgent processing (around line 800-900)

## Section 3A: Wire Layer 5 handler into ConversationAgent

FIND where ConversationAgent is initialized (around line 100-120 in NewV2APIServer):
```go
// Initialize ConversationAgent or where it's currently created
conversationAgent := agents.NewConversationAgent(...)
```

ADD AFTER initialization:
```go
	// NEW: Wire Phase 2 components into ConversationAgent
	if conversationAgent != nil && layer5Handler != nil {
		conversationAgent.SetLayer5ConflictHandler(layer5Handler)
		log.Printf("[V2APIServer] ✓ Layer 5 handler wired into ConversationAgent")
	}
```

---

## Section 3B: Add Layer 5 processing in ConversationAgent.Run()

NOTE: This may be in `agents/conversation_agent.go` instead of main.go. Check line 400-500 in that file.

FIND the section after Layer 4 gap detection (should say "// LAYER 4" or similar):
```go
// Layer 4 processing
// ... gap detection code ...
```

ADD IMMEDIATELY AFTER Layer 4 (before Layer 6):
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
			response.Metadata["layer5_question"] = true

			log.Printf("[ConversationAgent] ✓ Layer 5: Conflict question: %s", conflictQ.Question)
			return response, nil
		}
	} else if !flags.UseLayer7ConflictGate {
		log.Printf("[ConversationAgent] [Phase 2] Layer 5 DISABLED")
	}
```

---

## Section 3C: Record conflict metrics in ConversationAgent

FIND where conflicts are first detected or logged (search for "conflict" in the file):
```go
if len(extractionConflicts) > 0 {
    // ... handle conflicts ...
}
```

ADD metrics recording:
```go
	if len(extractionConflicts) > 0 {
		metrics := monitoring.GetMetrics()
		for range extractionConflicts {
			metrics.RecordConflictDetected()
		}
		log.Printf("[ConversationAgent] Recorded %d conflict detections", len(extractionConflicts))
	}
```

---

## Section 3D: Add deduplication logging

FIND where ClarificationHistory is checked:
```go
if clarificationHistory.WasRecentlyAsked(...) {
    // Skip this question
}
```

ADD metrics call:
```go
	if clarificationHistory.WasRecentlyAsked(userID, conflictID) {
		// Skip this question
		metrics.RecordDeduplicationPrevented()
		log.Printf("[ConversationAgent] Deduplication: Skipped recently-asked question")
	}
```

---

## Verification for Patch 3

After applying, verify:
```bash
# Build should pass
go build ./...

# Enable Phase 2 and test conflict detection
export MOLY_USE_LAYER5_CONFLICT_GATE=true

# Send two contradictory messages to see Layer 5 kick in
# Check logs for:
# - "[Phase 2] Layer 5 ENABLED"
# - "Conflict question"
# - "Deduplication: Skipped"
```

✅ When complete, move to PATCH_04_PHASE3_RESPONSE.md

