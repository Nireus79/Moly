package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"moly/database"
)

// JobScheduler manages background jobs for Phase 1.2
type JobScheduler struct {
	db                *database.Database
	manager           *EphemeralConversationManager
	ticker            *time.Ticker
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	isRunning         bool
	mutex             sync.RWMutex
	lastExtractionRun time.Time
	lastCleanupRun    time.Time
	metrics           *JobMetrics
}

// JobMetrics tracks job execution metrics
type JobMetrics struct {
	ExtractionJobsRun        int64
	ExtractionSuccessful     int64
	ExtractionFailed         int64
	ExtractionItemsProcessed int64
	ExtractionTotalTimeMs    int64

	CleanupJobsRun              int64
	CleanupSuccessful           int64
	CleanupFailed               int64
	CleanupConversationsDeleted int64
	CleanupTotalTimeMs          int64
}

// RetryConfig holds configurable retry behavior
type RetryConfig struct {
	MaxRetries        int           // Maximum number of retry attempts (default: 3)
	InitialBackoff    time.Duration // Initial backoff delay (default: 100ms)
	MaxBackoff        time.Duration // Maximum backoff delay (default: 30s)
	BackoffMultiplier float64       // Exponential backoff multiplier (default: 2.0)
}

// JobConfig holds job configuration
type JobConfig struct {
	ExtractionInterval   time.Duration // How often to run extraction (default: 1 hour)
	CleanupInterval      time.Duration // How often to run cleanup (default: 24 hours)
	MaxExtractionsPerRun int           // Max items to process per run (default: 10)
	EnableExtraction     bool          // Enable extraction job
	EnableCleanup        bool          // Enable cleanup job
	RetryConfig          *RetryConfig  // Retry configuration for extraction errors
}

// DefaultJobConfig returns default configuration
func DefaultJobConfig() JobConfig {
	return JobConfig{
		ExtractionInterval:   1 * time.Hour,
		CleanupInterval:      24 * time.Hour,
		MaxExtractionsPerRun: 10,
		EnableExtraction:     true,
		EnableCleanup:        true,
		RetryConfig: &RetryConfig{
			MaxRetries:        3, // Up to 3 attempts
			InitialBackoff:    100 * time.Millisecond,
			MaxBackoff:        30 * time.Second,
			BackoffMultiplier: 2.0, // Double the backoff each time
		},
	}
}

// NewJobScheduler creates a new job scheduler
func NewJobScheduler(
	db *database.Database,
	manager *EphemeralConversationManager,
) *JobScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &JobScheduler{
		db:      db,
		manager: manager,
		ctx:     ctx,
		cancel:  cancel,
		metrics: &JobMetrics{},
	}
}

// Start begins the job scheduler
func (js *JobScheduler) Start(config JobConfig) error {
	js.mutex.Lock()
	if js.isRunning {
		js.mutex.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	js.isRunning = true
	js.mutex.Unlock()

	log.Println("[JobScheduler] Starting background job scheduler...")
	log.Printf("[JobScheduler] Extraction interval: %v\n", config.ExtractionInterval)
	log.Printf("[JobScheduler] Cleanup interval: %v\n", config.CleanupInterval)

	// Start extraction job
	if config.EnableExtraction {
		js.wg.Add(1)
		go js.extractionJobLoop(config.ExtractionInterval)
		log.Println("[JobScheduler] Extraction job started")
	}

	// Start cleanup job
	if config.EnableCleanup {
		js.wg.Add(1)
		go js.cleanupJobLoop(config.CleanupInterval)
		log.Println("[JobScheduler] Cleanup job started")
	}

	return nil
}

// Stop stops the job scheduler
func (js *JobScheduler) Stop() error {
	js.mutex.Lock()
	if !js.isRunning {
		js.mutex.Unlock()
		return fmt.Errorf("scheduler not running")
	}
	js.isRunning = false
	js.mutex.Unlock()

	log.Println("[JobScheduler] Stopping background job scheduler...")
	js.cancel()

	// Wait for goroutines to finish (with timeout)
	done := make(chan struct{})
	go func() {
		js.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("[JobScheduler] Scheduler stopped gracefully")
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("scheduler shutdown timeout")
	}
}

// IsRunning returns whether the scheduler is running
func (js *JobScheduler) IsRunning() bool {
	js.mutex.RLock()
	defer js.mutex.RUnlock()
	return js.isRunning
}

// ============================================================================
// Extraction Job (Hourly)
// ============================================================================

func (js *JobScheduler) extractionJobLoop(interval time.Duration) {
	defer js.wg.Done()

	log.Println("[ExtractionJob] Extraction job loop started")

	// Run immediately on start
	js.runExtractionJob()

	// Then run on schedule
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-js.ctx.Done():
			log.Println("[ExtractionJob] Extraction job stopping")
			return
		case <-ticker.C:
			js.runExtractionJob()
		}
	}
}

func (js *JobScheduler) runExtractionJob() {
	start := time.Now()
	js.metrics.ExtractionJobsRun++

	log.Printf("[ExtractionJob] STARTED: Job #%d (run_time: %s)", js.metrics.ExtractionJobsRun, time.Now().Format("15:04:05"))

	// Get queue status first
	status, err := js.manager.GetQueueStatus()
	if err != nil {
		log.Printf("[ExtractionJob] ERROR: Failed to get queue status: %v", err)
		js.metrics.ExtractionFailed++
		return
	}

	pendingCount := status["pending"]
	completedCount := status["completed"]
	failedCount := status["failed"]
	log.Printf("[ExtractionJob] QUEUE_STATUS: pending=%d, completed=%d, failed=%d",
		pendingCount, completedCount, failedCount)

	if pendingCount == 0 {
		elapsed := time.Since(start).Milliseconds()
		log.Printf("[ExtractionJob] SKIPPED: No pending items (elapsed: %dms)", elapsed)
		return
	}

	// Process queue
	log.Printf("[ExtractionJob] PROCESSING: Starting queue processing for %d items", pendingCount)
	stats, err := js.manager.ProcessQueue()
	if err != nil {
		elapsed := time.Since(start).Milliseconds()
		log.Printf("[ExtractionJob] ERROR: Queue processing failed after %dms: %v", elapsed, err)
		js.metrics.ExtractionFailed++
		return
	}

	elapsed := time.Since(start).Milliseconds()
	js.metrics.ExtractionSuccessful++
	js.metrics.ExtractionItemsProcessed += int64(stats.Processed)
	js.metrics.ExtractionTotalTimeMs += elapsed
	log.Printf("[ExtractionJob] COMPLETED: Processed %d items in %dms (success_rate: %d/%d)",
		stats.Processed, elapsed, js.metrics.ExtractionSuccessful, js.metrics.ExtractionJobsRun)

	log.Printf("[ExtractionJob] Extraction complete: %d processed, %d failed in %v\n",
		stats.Processed, stats.Failed, time.Since(start))

	js.mutex.Lock()
	js.lastExtractionRun = time.Now()
	js.mutex.Unlock()
}

// ============================================================================
// Cleanup Job (Daily)
// ============================================================================

func (js *JobScheduler) cleanupJobLoop(interval time.Duration) {
	defer js.wg.Done()

	log.Println("[CleanupJob] Cleanup job loop started")

	// Run immediately on start
	js.runCleanupJob()

	// Then run on schedule
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-js.ctx.Done():
			log.Println("[CleanupJob] Cleanup job stopping")
			return
		case <-ticker.C:
			js.runCleanupJob()
		}
	}
}

func (js *JobScheduler) runCleanupJob() {
	start := time.Now()
	js.metrics.CleanupJobsRun++

	log.Printf("[CleanupJob] STARTED: Job #%d (run_time: %s)", js.metrics.CleanupJobsRun, time.Now().Format("15:04:05"))

	// Get count before cleanup
	countBefore, err := js.manager.GetEphemeralConversationCount()
	if err != nil {
		log.Printf("[CleanupJob] ERROR: Failed to get count before cleanup: %v", err)
		js.metrics.CleanupFailed++
		return
	}

	log.Printf("[CleanupJob] SNAPSHOT_BEFORE: %d active conversations (24h TTL)", countBefore)

	// Run cleanup
	log.Printf("[CleanupJob] PROCESSING: Removing expired conversations")
	stats, err := js.manager.CleanupExpired()
	if err != nil {
		elapsed := time.Since(start).Milliseconds()
		log.Printf("[CleanupJob] ERROR: Cleanup failed after %dms: %v", elapsed, err)
		js.metrics.CleanupFailed++
		return
	}

	// Get count after cleanup
	countAfter, err := js.manager.GetEphemeralConversationCount()
	if err != nil {
		log.Printf("[CleanupJob] WARN: Failed to get count after cleanup: %v", err)
		// Don't mark as failed - cleanup actually succeeded
	} else {
		log.Printf("[CleanupJob] SNAPSHOT_AFTER: %d active conversations", countAfter)
	}

	elapsed := time.Since(start).Milliseconds()
	js.metrics.CleanupSuccessful++
	js.metrics.CleanupConversationsDeleted += int64(stats.DeletedCount)
	js.metrics.CleanupTotalTimeMs += elapsed
	log.Printf("[CleanupJob] COMPLETED: Deleted %d conversations in %dms (before=%d, after=%d, success_rate=%d/%d)",
		stats.DeletedCount, elapsed, countBefore, countAfter, js.metrics.CleanupSuccessful, js.metrics.CleanupJobsRun)

	log.Printf("[CleanupJob] Cleanup complete: %d deleted, %d remaining in %v\n",
		stats.DeletedCount, countAfter, time.Since(start))

	js.mutex.Lock()
	js.lastCleanupRun = time.Now()
	js.mutex.Unlock()
}

// ============================================================================
// Metrics and Status
// ============================================================================

// GetMetrics returns current job metrics
func (js *JobScheduler) GetMetrics() JobMetrics {
	js.mutex.RLock()
	defer js.mutex.RUnlock()
	return *js.metrics
}

// ResetMetrics resets all metrics
func (js *JobScheduler) ResetMetrics() {
	js.mutex.Lock()
	defer js.mutex.Unlock()
	js.metrics = &JobMetrics{}
}

// GetStatus returns current scheduler status
func (js *JobScheduler) GetStatus() map[string]interface{} {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	metrics := js.metrics
	return map[string]interface{}{
		"running":             js.isRunning,
		"last_extraction_run": js.lastExtractionRun.Format(time.RFC3339),
		"last_cleanup_run":    js.lastCleanupRun.Format(time.RFC3339),
		"extraction": map[string]interface{}{
			"jobs_run":        metrics.ExtractionJobsRun,
			"successful":      metrics.ExtractionSuccessful,
			"failed":          metrics.ExtractionFailed,
			"items_processed": metrics.ExtractionItemsProcessed,
			"total_time_ms":   metrics.ExtractionTotalTimeMs,
		},
		"cleanup": map[string]interface{}{
			"jobs_run":              metrics.CleanupJobsRun,
			"successful":            metrics.CleanupSuccessful,
			"failed":                metrics.CleanupFailed,
			"conversations_deleted": metrics.CleanupConversationsDeleted,
			"total_time_ms":         metrics.CleanupTotalTimeMs,
		},
	}
}

// ForceExtractionNow runs extraction immediately (for testing/manual triggers)
func (js *JobScheduler) ForceExtractionNow() error {
	if !js.IsRunning() {
		return fmt.Errorf("scheduler not running")
	}

	log.Println("[JobScheduler] Manual extraction triggered")
	go js.runExtractionJob()
	return nil
}

// ForceCleanupNow runs cleanup immediately (for testing/manual triggers)
func (js *JobScheduler) ForceCleanupNow() error {
	if !js.IsRunning() {
		return fmt.Errorf("scheduler not running")
	}

	log.Println("[JobScheduler] Manual cleanup triggered")
	go js.runCleanupJob()
	return nil
}
