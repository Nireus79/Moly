# Phase 1.2 Background Jobs Setup

**Status**: Complete & Ready  
**Components**: JobScheduler + HTTP Handlers  
**Tests**: 7/7 passing

---

## What Background Jobs Do

### 1. Extraction Job (Hourly)
Runs every 1 hour and:
- Fetches pending conversations from `extraction_queue`
- Runs `ConversationAnalyzer` on each one (LLM extraction)
- Updates user profile via `ProfileUpdater`
- Marks items as completed/failed
- Logs metrics (processed, failed, time)

**Purpose**: Convert conversations to insights automatically  
**Frequency**: Every 1 hour  
**Max per run**: 10 conversations  
**On failure**: Retries up to 3 times

### 2. Cleanup Job (Daily)
Runs every 24 hours and:
- Deletes conversations older than 24 hours from `conversation_ephemeral`
- Keeps extraction results (permanent tables)
- Frees up disk space
- Logs deletion count

**Purpose**: Clean up raw conversation data per privacy policy  
**Frequency**: Every 24 hours  
**On failure**: Logs error, doesn't block other jobs

---

## Integration (5 minutes)

### Step 1: Add JobScheduler to main.go

```go
import (
    "moly/services"
    "moly/handlers"
)

func main() {
    // ... existing setup ...

    // Initialize database
    db, err := database.Init("moly.db", "your-encryption-key")
    if err != nil {
        log.Fatal(err)
    }

    // Initialize Phase 1.2 components
    analyzer := agents.NewConversationAnalyzer(llmClient, db)
    updater := services.NewProfileUpdater(db)
    manager := services.NewEphemeralConversationManager(db, analyzer, updater)

    // Create job scheduler
    scheduler := services.NewJobScheduler(db, manager)

    // Start background jobs
    jobConfig := services.JobConfig{
        ExtractionInterval:  1 * time.Hour,
        CleanupInterval:     24 * time.Hour,
        MaxExtractionsPerRun: 10,
        EnableExtraction:   true,
        EnableCleanup:      true,
    }

    if err := scheduler.Start(jobConfig); err != nil {
        log.Fatal("Failed to start job scheduler:", err)
    }

    // ... rest of server setup ...

    // Register HTTP handlers for job management
    jobsHandlers := handlers.NewJobsHandlers(scheduler)
    jobsHandlers.RegisterRoutes(mux)

    // Start server
    server := &http.Server{Handler: mux, Addr: ":8080"}
    
    // Graceful shutdown
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, os.Interrupt)
        <-sigChan
        
        scheduler.Stop()  // Stop jobs first
        server.Shutdown(context.Background())
    }()

    log.Fatal(server.ListenAndServe())
}
```

---

## Job Management Endpoints

### Check Job Status
```bash
GET /api/v2.1/jobs/status

Response:
{
  "data": {
    "running": true,
    "last_extraction_run": "2026-09-08T12:30:00Z",
    "last_cleanup_run": "2026-09-07T12:00:00Z",
    "extraction": {
      "jobs_run": 24,
      "successful": 23,
      "failed": 1,
      "items_processed": 156,
      "total_time_ms": 45000
    },
    "cleanup": {
      "jobs_run": 1,
      "successful": 1,
      "failed": 0,
      "conversations_deleted": 42,
      "total_time_ms": 2300
    }
  },
  "message": "Job scheduler status"
}
```

### Get Detailed Metrics
```bash
GET /api/v2.1/jobs/metrics

Response:
{
  "data": {
    "extraction": {
      "jobs_run": 24,
      "successful": 23,
      "failed": 1,
      "items_processed": 156,
      "total_time_ms": 45000
    },
    "cleanup": {
      "jobs_run": 1,
      "successful": 1,
      "failed": 0,
      "conversations_deleted": 42,
      "total_time_ms": 2300
    }
  },
  "message": "Job scheduler metrics"
}
```

### Trigger Extraction Job Manually
```bash
POST /api/v2.1/jobs/extraction/force

Response:
{
  "data": {"triggered": true},
  "message": "Extraction job triggered"
}
```

### Trigger Cleanup Job Manually
```bash
POST /api/v2.1/jobs/cleanup/force

Response:
{
  "data": {"triggered": true},
  "message": "Cleanup job triggered"
}
```

### Reset Metrics
```bash
POST /api/v2.1/jobs/metrics/reset

Response:
{
  "data": {"reset": true},
  "message": "Metrics reset"
}
```

---

## Configuration Options

### JobConfig

```go
type JobConfig struct {
    ExtractionInterval     time.Duration  // Default: 1 hour
    CleanupInterval        time.Duration  // Default: 24 hours
    MaxExtractionsPerRun   int            // Default: 10
    EnableExtraction      bool            // Default: true
    EnableCleanup         bool            // Default: true
}
```

### Example: Custom Configuration

```go
// Run extraction every 30 minutes, cleanup every 12 hours
config := services.JobConfig{
    ExtractionInterval:  30 * time.Minute,
    CleanupInterval:     12 * time.Hour,
    MaxExtractionsPerRun: 20,
    EnableExtraction:   true,
    EnableCleanup:      true,
}

scheduler.Start(config)
```

### Example: Disable Extraction (keep only cleanup)

```go
config := services.JobConfig{
    ExtractionInterval:  1 * time.Hour,
    CleanupInterval:     24 * time.Hour,
    EnableExtraction:   false,  // Disabled
    EnableCleanup:      true,
}

scheduler.Start(config)
```

---

## Logs You'll See

### On Startup
```
[JobScheduler] Starting background job scheduler...
[JobScheduler] Extraction interval: 1h0m0s
[JobScheduler] Cleanup interval: 24h0m0s
[JobScheduler] Extraction job started
[JobScheduler] Cleanup job started
```

### During Extraction (hourly)
```
[ExtractionJob] Starting extraction job...
[ExtractionJob] Queue status: 5 pending, 42 completed, 1 failed
[ExtractionJob] Extraction complete: 5 processed, 0 failed in 2.345s
```

### During Cleanup (daily)
```
[CleanupJob] Starting cleanup job...
[CleanupJob] Active conversations before cleanup: 12
[CleanupJob] Cleanup complete: 8 deleted, 4 remaining in 0.823s
```

### On Shutdown
```
[ExtractionJob] Extraction job stopping
[CleanupJob] Cleanup job stopping
[JobScheduler] Scheduler stopped gracefully
```

---

## Monitoring & Debugging

### Check if jobs are running
```bash
curl http://localhost:8080/api/v2.1/jobs/status | jq .data.running
# Returns: true or false
```

### Check last run times
```bash
curl http://localhost:8080/api/v2.1/jobs/status | jq '.data | {last_extraction: .last_extraction_run, last_cleanup: .last_cleanup_run}'
```

### Monitor metrics (watch command)
```bash
watch -n 5 'curl -s http://localhost:8080/api/v2.1/jobs/metrics | jq .data'
```

### Manual extraction trigger (for testing)
```bash
curl -X POST http://localhost:8080/api/v2.1/jobs/extraction/force
# Check logs for [ExtractionJob] output
```

### Manual cleanup trigger (for testing)
```bash
curl -X POST http://localhost:8080/api/v2.1/jobs/cleanup/force
# Check logs for [CleanupJob] output
```

---

## Performance Tuning

### If extraction is slow

1. **Reduce max extractions per run**:
   ```go
   config.MaxExtractionsPerRun = 5  // Was 10
   ```

2. **Increase interval**:
   ```go
   config.ExtractionInterval = 2 * time.Hour  // Was 1 hour
   ```

3. **Check LLM performance**:
   - Monitor extraction logs for duration
   - `total_time_ms / items_processed` = avg time per item
   - If > 5 seconds per item, LLM is slow (network or model issue)

### If cleanup is causing locks

1. **Run more frequently in smaller batches** (SQLite limitation):
   ```go
   config.CleanupInterval = 12 * time.Hour  // More frequent
   ```

2. **Check active queries**:
   ```bash
   # Add debug logging to see if cleanup is blocking extraction
   ```

### If disk space is limited

1. **Reduce cleanup interval**:
   ```go
   config.CleanupInterval = 12 * time.Hour  // Was 24
   ```

2. **Increase extraction speed** to clean up faster:
   ```go
   config.ExtractionInterval = 30 * time.Minute
   config.MaxExtractionsPerRun = 20
   ```

---

## Error Handling

### Job Fails
- Error is logged
- Job failure count incremented
- Next run continues normally
- No cascading failures

### Extraction Queue Fills Up
- Check logs for extraction errors
- Run manual trigger to force processing
- Check LLM connectivity if slow
- Reduce max per run if overloaded

### Cleanup Doesn't Run
- Check logs for errors
- Verify disk space
- Manual trigger to test
- Check database locks

---

## Testing Jobs Locally

### Run extraction on demand
```bash
curl -X POST http://localhost:8080/api/v2.1/jobs/extraction/force
curl http://localhost:8080/api/v2.1/jobs/metrics
```

### Test job startup in code
```go
// In tests/job_scheduler_test.go
config := services.DefaultJobConfig()
config.ExtractionInterval = 100 * time.Millisecond
config.CleanupInterval = 200 * time.Millisecond

err := scheduler.Start(config)
time.Sleep(500 * time.Millisecond)  // Let jobs run

metrics := scheduler.GetMetrics()
if metrics.ExtractionJobsRun == 0 {
    t.Fatal("Extraction job didn't run")
}
```

---

## Production Checklist

- [ ] Schema deployed (`deploy-schema.sh` completed)
- [ ] Database backups configured
- [ ] LLM API configured (Claude, OpenAI, or Ollama)
- [ ] Job scheduler integrated in main.go
- [ ] Job handlers registered in mux
- [ ] Monitoring/metrics endpoint accessible
- [ ] Logs being collected (stdout → logging system)
- [ ] Graceful shutdown handling in place
- [ ] Tested manual job triggers
- [ ] Tested with real conversations
- [ ] Performance baseline established
- [ ] Alert configured for job failures

---

## Summary

| Component | Status | Lines | Tests |
|-----------|--------|-------|-------|
| JobScheduler | ✅ | 250 | 7/7 |
| JobsHandlers | ✅ | 100 | Included |
| Tests | ✅ | 200+ | 7/7 |
| Integration Guide | ✅ | This doc | N/A |

**Time to integrate**: 5 minutes  
**Time to verify**: 5 minutes  
**Total time**: ~10 minutes

---

## Next Steps

1. ✅ Schema deployed
2. ✅ Background jobs set up
3. → Configure real LLM API
4. → Wire up extension UI
5. → Run E2E tests

**Jobs are ready to go!** 🚀
