# PATCH 4: Phase 3 Constrained Response Generation
# Location: Response generation handler (around line 1800-1900)

## Section 4A: Replace basic response generation with constrained generation

FIND the response generation call (search for "GenerateContextualResponse"):
```go
agentResp, err := responseGenerator.GenerateContextualResponse(
    ctx,
    userMessage,
    ctx.AboutMe,
    ctx.ContactProfiles,
)
```

REPLACE with:
```go
	// NEW: PHASE 3 - CONSTRAINED RESPONSE GENERATION
	flags := config.GetFeatureFlags()
	metrics := monitoring.GetMetrics()

	var agentResp *models.ConversationResponse
	var err error

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
			if violations, ok := agentResp.Metadata["violationCount"].(int); ok && violations > 0 {
				log.Printf("[MessageProcessor] Constraint violations detected: %d", violations)
			}
		}
	} else {
		log.Printf("[MessageProcessor] [Phase 3] Constrained generation DISABLED (using basic generator)")

		generationStartTime := time.Now()

		agentResp, err = responseGenerator.GenerateContextualResponse(
			ctx,
			userMessage,
			ctx.AboutMe,
			ctx.ContactProfiles,
		)

		generationMs := time.Since(generationStartTime).Milliseconds()
		metrics.RecordResponseLatency(generationMs)
	}

	if err != nil {
		log.Printf("[MessageProcessor] Response generation failed: %v", err)
```

---

## Section 4B: Add Phase 3 metadata to response

FIND where response metadata is set (search for "response.Metadata"):
```go
response.Metadata = map[string]interface{}{
    // ... existing metadata ...
}
```

ADD Phase 3 fields:
```go
	if agentResp != nil && agentResp.Metadata != nil {
		// NEW: Phase 3 metadata
		response.Metadata["phase3_enabled"] = flags.UseConstrainedResponseGeneration
		response.Metadata["phase3_validation"] = agentResp.Metadata["responseValidated"]
		response.Metadata["phase3_constraints_applied"] = agentResp.Metadata["constraintsApplied"]
		
		if fallback, ok := agentResp.Metadata["fallbackToClarity"].(bool); ok {
			response.Metadata["phase3_fallback"] = fallback
		}
	}
```

---

## Section 4C: Add response validation to all response paths

FIND all places where a response is returned to the user (search for "schema.RespondJSON"):
```go
schema.RespondJSON(w, 200, response)
```

ADD monitoring call BEFORE each response:
```go
	// NEW: Record response metrics
	flags := config.GetFeatureFlags()
	metrics := monitoring.GetMetrics()
	
	if flags.EnableMetrics {
		log.Printf("[MessageProcessor] Response metadata: %+v", response.Metadata)
	}

	schema.RespondJSON(w, 200, response)
```

---

## Section 4D: Track constraint cache performance (optional)

IF you want to track cache hit rate, add this somewhere after response generation:
```go
	// NEW: Cache statistics
	if srv.constrainedResponseGen != nil {
		cacheStats := srv.constrainedResponseGen.constraintCache.GetStats()
		log.Printf("[MessageProcessor] Cache stats: %+v", cacheStats)
		
		if hitRate, ok := cacheStats["hit_rate"].(float64); ok {
			metrics.RecordCacheHitRate(hitRate)
		}
	}
```

---

## Verification for Patch 4

After applying, test:
```bash
# Build should pass
go build ./...

# Enable Phase 3 and test response generation
export MOLY_USE_CONSTRAINED_RESPONSE_GENERATION=true

# Send a message and check:
# - "Phase 3 Constrained generation ENABLED"
# - Response contains constraint validation metadata
# - Latency recorded in metrics
# - Cache stats shown in logs
```

✅ When complete, move to PATCH_05_MONITORING_RESPONSE.md

