package services

import (
	"path/filepath"
	"testing"
	"time"

	"moly/agents"
	"moly/database"
	"moly/tools"
)

// setupJobSchedulerTest creates test environment
func setupJobSchedulerTest(t *testing.T) *JobScheduler {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_jobs.db")

	systemKey := "test-jobs-key"
	db, err := database.Init(dbPath, systemKey)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}

	analyzer := agents.NewConversationAnalyzer(tools.NewMockLLMClient(), db)
	updater := NewProfileUpdater(db)
	manager := NewEphemeralConversationManager(db, analyzer, updater)
	scheduler := NewJobScheduler(db, manager)

	return scheduler
}

// TestJobSchedulerStart tests starting the scheduler
func TestJobSchedulerStart(t *testing.T) {
	scheduler := setupJobSchedulerTest(t)

	config := JobConfig{
		ExtractionInterval:  10 * time.Second,
		CleanupInterval:     20 * time.Second,
		EnableExtraction:   true,
		EnableCleanup:      true,
	}

	err := scheduler.Start(config)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	if !scheduler.IsRunning() {
		t.Error("Scheduler should be running")
	}

	t.Log("✓ Scheduler started successfully")

	// Stop
	err = scheduler.Stop()
	if err != nil {
		t.Fatalf("Failed to stop scheduler: %v", err)
	}

	if scheduler.IsRunning() {
		t.Error("Scheduler should not be running")
	}

	t.Log("✓ Scheduler stopped successfully")
}

// TestJobSchedulerDoubleStart tests that double start fails
func TestJobSchedulerDoubleStart(t *testing.T) {
	scheduler := setupJobSchedulerTest(t)

	config := DefaultJobConfig()
	config.ExtractionInterval = 10 * time.Second
	config.CleanupInterval = 20 * time.Second

	err := scheduler.Start(config)
	if err != nil {
		t.Fatalf("First start failed: %v", err)
	}

	// Try to start again
	err = scheduler.Start(config)
	if err == nil {
		t.Error("Double start should fail")
	}

	scheduler.Stop()
	t.Log("✓ Double start correctly rejected")
}

// TestJobSchedulerMetrics tests metrics tracking
func TestJobSchedulerMetrics(t *testing.T) {
	scheduler := setupJobSchedulerTest(t)

	// Check initial metrics
	metrics := scheduler.GetMetrics()
	if metrics.ExtractionJobsRun != 0 {
		t.Error("Metrics should start at 0")
	}

	t.Log("✓ Initial metrics correct")

	// Reset metrics
	scheduler.ResetMetrics()
	metrics = scheduler.GetMetrics()
	if metrics.ExtractionJobsRun != 0 {
		t.Error("Metrics should be reset to 0")
	}

	t.Log("✓ Reset metrics works")
}

// TestJobSchedulerStatus tests status reporting
func TestJobSchedulerStatus(t *testing.T) {
	scheduler := setupJobSchedulerTest(t)

	config := DefaultJobConfig()
	err := scheduler.Start(config)
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	status := scheduler.GetStatus()

	if running, ok := status["running"].(bool); !ok || !running {
		t.Error("Status should show running=true")
	}

	if extraction, ok := status["extraction"].(map[string]interface{}); ok {
		if _, hasJobs := extraction["jobs_run"]; !hasJobs {
			t.Error("Status should have extraction metrics")
		}
	} else {
		t.Error("Status should have extraction field")
	}

	scheduler.Stop()
	t.Log("✓ Status reporting works")
}

// TestJobSchedulerForceExtraction tests manual extraction trigger
func TestJobSchedulerForceExtraction(t *testing.T) {
	scheduler := setupJobSchedulerTest(t)

	config := DefaultJobConfig()
	err := scheduler.Start(config)
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Force extraction
	err = scheduler.ForceExtractionNow()
	if err != nil {
		t.Fatalf("Force extraction failed: %v", err)
	}

	// Give it time to run
	time.Sleep(100 * time.Millisecond)

	metrics := scheduler.GetMetrics()
	if metrics.ExtractionJobsRun == 0 {
		t.Error("Should have run at least one extraction job")
	}

	scheduler.Stop()
	t.Log("✓ Force extraction works")
}

// TestJobSchedulerForceCleanup tests manual cleanup trigger
func TestJobSchedulerForceCleanup(t *testing.T) {
	scheduler := setupJobSchedulerTest(t)

	config := DefaultJobConfig()
	err := scheduler.Start(config)
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Force cleanup
	err = scheduler.ForceCleanupNow()
	if err != nil {
		t.Fatalf("Force cleanup failed: %v", err)
	}

	// Give it time to run
	time.Sleep(100 * time.Millisecond)

	metrics := scheduler.GetMetrics()
	if metrics.CleanupJobsRun == 0 {
		t.Error("Should have run at least one cleanup job")
	}

	scheduler.Stop()
	t.Log("✓ Force cleanup works")
}

// TestJobSchedulerDisableExtraction tests disabling extraction
func TestJobSchedulerDisableExtraction(t *testing.T) {
	scheduler := setupJobSchedulerTest(t)

	config := DefaultJobConfig()
	config.EnableExtraction = false
	config.CleanupInterval = 10 * time.Second

	err := scheduler.Start(config)
	if err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	metrics := scheduler.GetMetrics()
	if metrics.ExtractionJobsRun != 0 {
		t.Error("Extraction should be disabled")
	}

	scheduler.Stop()
	t.Log("✓ Disabling extraction works")
}

// TestJobSchedulerGate - Complete scheduler verification
func TestJobSchedulerGate(t *testing.T) {
	tests := []struct {
		name string
		test func(*testing.T)
	}{
		{"Start", TestJobSchedulerStart},
		{"DoubleStart", TestJobSchedulerDoubleStart},
		{"Metrics", TestJobSchedulerMetrics},
		{"Status", TestJobSchedulerStatus},
		{"ForceExtraction", TestJobSchedulerForceExtraction},
		{"ForceCleanup", TestJobSchedulerForceCleanup},
		{"DisableExtraction", TestJobSchedulerDisableExtraction},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}

	t.Log("\n✓ JobScheduler gate PASSED - Background jobs ready")
}
