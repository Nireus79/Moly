# PATCH 5: Final Monitoring & Response Metadata
# Location: End of message handler (before returning response)

## Section 5A: Add comprehensive monitoring before response return

FIND the final response return (around line 5050+):
```go
w.Header().Set("Content-Type", "application/json")
schema.RespondJSON(w, 200, response)
```

ADD monitoring BEFORE the response:
```go
	// NEW: MONITORING - Record final metrics
	flags := config.GetFeatureFlags()
	metrics := monitoring.GetMetrics()

	if flags.EnableMetrics {
		// Log summary of what happened in this request
		summary := map[string]interface{}{
			"phase1_enabled": flags.UseExtractionLock,
			"phase2_enabled": flags.UseLayer5ConflictGate,
			"phase3_enabled": flags.UseConstrainedResponseGeneration,
			"phase4_enabled": flags.UseCleanSchema,
			"extraction_count": len(extractedEntities),
			"conflicts_detected": len(extractionConflicts),
		}

		log.Printf("[MessageProcessor] Request summary: %+v", summary)

		// Add to response metadata
		if response.Metadata == nil {
			response.Metadata = make(map[string]interface{})
		}
		response.Metadata["phase_status"] = summary
		response.Metadata["timestamp"] = time.Now().Unix()
	}

	if flags.EnableDetailedLogging {
		log.Printf("[MessageProcessor] Full metrics: %+v", metrics.GetSummary())
	}

	w.Header().Set("Content-Type", "application/json")
	schema.RespondJSON(w, 200, response)
```

---

## Section 5B: Add metrics endpoint (NEW ENDPOINT)

ADD a new HTTP handler for /health/metrics (around line 5100):
```go
// NEW: Metrics endpoint for monitoring
func (srv *V2APIServer) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	flags := config.GetFeatureFlags()
	metrics := monitoring.GetMetrics()

	response := map[string]interface{}{
		"flags": flags.GetStatus(),
		"metrics": metrics.GetSummary(),
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
```

---

## Section 5C: Register metrics endpoint

FIND where HTTP routes are registered (search for "mux.HandleFunc"):
```go
mux.HandleFunc("/api/v2/conversations", ...)
mux.HandleFunc("/health", ...)
```

ADD:
```go
	// NEW: Monitoring endpoint
	mux.HandleFunc("/health/metrics", srv.handleGetMetrics)
	log.Printf("[V2APIServer] ✓ Metrics endpoint registered at /health/metrics")
```

---

## Section 5D: Add startup logging (in main())

FIND the main() function where server starts (around line 5080+):
```go
func main() {
    // ... initialization ...
    log.Printf("Starting Moly server...")
}
```

ADD AFTER initialization:
```go
	// NEW: Log phase configuration at startup
	flags := config.GetFeatureFlags()
	log.Printf("[Main] ========================================")
	log.Printf("[Main] MOLY PHASES CONFIGURATION")
	log.Printf("[Main] ========================================")
	log.Printf("[Main] Phase 1 (Extraction Lock):        %v", flags.UseExtractionLock)
	log.Printf("[Main] Phase 2 (Layer 5 Conflicts):      %v", flags.UseLayer5ConflictGate)
	log.Printf("[Main] Phase 3 (Constrained Gen):       %v", flags.UseConstrainedResponseGeneration)
	log.Printf("[Main] Phase 4 (Clean Schema):          %v", flags.UseCleanSchema)
	log.Printf("[Main] Metrics Enabled:                 %v", flags.EnableMetrics)
	log.Printf("[Main] Detailed Logging:                %v", flags.EnableDetailedLogging)
	log.Printf("[Main] Metrics Collection Rate:         %d%%", flags.MetricsCollectionRate)
	log.Printf("[Main] ========================================")
	log.Printf("[Main] Server ready - metrics at /health/metrics")
```

---

## Section 5E: Add graceful shutdown with metrics (optional)

ADD at end of main():
```go
	// NEW: Graceful shutdown with metrics
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Printf("[Main] ========================================")
		log.Printf("[Main] FINAL METRICS BEFORE SHUTDOWN")
		log.Printf("[Main] ========================================")
		metrics := monitoring.GetMetrics()
		summary := metrics.GetSummary()
		log.Printf("[Main] %+v", summary)
		log.Printf("[Main] ========================================")
		os.Exit(0)
	}()
```

---

## Verification for Patch 5

After applying all patches, comprehensive test:
```bash
# 1. Build
cd moly-go
go build ./...  # Should pass with ZERO warnings

# 2. Run all tests
go test ./... -v 2>&1 | grep -E "PASS|FAIL" | tail -20

# 3. Start server with all phases enabled
export MOLY_USE_EXTRACTION_LOCK=true
export MOLY_USE_LAYER5_CONFLICT_GATE=true
export MOLY_USE_CONSTRAINED_RESPONSE_GENERATION=true
export MOLY_ENABLE_METRICS=true
export MOLY_ENABLE_DETAILED_LOGGING=true

# 4. Check startup logs show all phases
# Should see:
# [Main] MOLY PHASES CONFIGURATION
# [Main] Phase 1 (Extraction Lock):        true
# [Main] Phase 2 (Layer 5 Conflicts):      true
# [Main] Phase 3 (Constrained Gen):        true

# 5. Test health/metrics endpoint
curl http://localhost:8080/health/metrics | jq .

# 6. Send test message
curl -X POST http://localhost:8080/api/v2/conversations/test/messages \
  -H "Content-Type: application/json" \
  -d '{"message": "Test message"}'

# 7. Check response has phase metadata
# Should include:
# "phase1_enabled": true
# "phase2_enabled": true
# "phase3_enabled": true
# "phase_status": {...}
```

---

## Testing Checklist After All Patches

- [ ] Build passes: `go build ./...` ✅
- [ ] Tests pass: `go test ./...` ✅ (71+ tests)
- [ ] Phase 1 logs: "Extraction lock ENABLED" ✅
- [ ] Phase 2 logs: "Layer 5 ENABLED" ✅
- [ ] Phase 3 logs: "Constrained generation ENABLED" ✅
- [ ] Metrics endpoint works: `/health/metrics` ✅
- [ ] Response includes phase metadata ✅
- [ ] Can toggle flags with env vars ✅
- [ ] Startup shows configuration ✅
- [ ] Shutdown shows final metrics ✅

---

## If Tests Fail

Common issues and fixes:

**Issue**: "undefined: Phase1Handler"  
**Fix**: Check imports at top of main.go

**Issue**: Build fails on monitoring  
**Fix**: Ensure monitoring/metrics.go exists

**Issue**: Metrics endpoint 404  
**Fix**: Check handleGetMetrics is registered

**Issue**: Response metadata missing  
**Fix**: Ensure response.Metadata is initialized

---

✅ **ALL PATCHES COMPLETE**

When all 5 patches are applied:
- Full Phase 1-4 integration done
- Feature flags working
- Monitoring active
- Response metadata included
- Ready for deployment

