package monitoring

import (
	"sync"
	"time"
)

// LayerTiming tracks per-layer performance
type LayerTiming struct {
	LayerName     string
	TimeMs        []int64
	TotalCalls    int64
	AverageTimeMs float64
}

// Metrics tracks performance metrics for all phases
type Metrics struct {
	// Phase 1: Extraction Lock
	ExtractionLockEnabled      bool
	ExtractionTimeMs           []int64 // Latency samples
	ExtractionLockFailures     int64
	MultiPassCallsPerMessage   map[int]int64 // Distribution
	SubjectAttributionAccuracy float64
	ExtractionsLocked          int64
	ExtractionsUnlocked        int64

	// Phase 2: Layer 5 Conflict Channeling
	ConflictsDetected              int64
	ConflictsResultingInQuestions  int64
	QuestionDeduplicationPrevented int64
	ConflictQuestionAskRate        float64
	UserResponseTimeToConflictQ    []int64 // Latency samples

	// Phase 3: Constrained Response Generation
	ConstraintsBuilt             int64
	ConstraintCacheHitRate       float64
	ResponseValidationViolations int64
	RoleReversalBugOccurrences   int64
	FalsePositiveBlocks          int64
	ResponseLatencyMs            []int64 // Latency samples

	// Phase 4: Clean Schema
	MigrationStartTime  time.Time
	MigrationEndTime    time.Time
	MigrationDataLoss   int64
	MigrationErrorCount int64

	// PHASE 2.1: Layer-Level Timing (NEW)
	LayerTimings map[string]*LayerTiming // Per-layer performance

	mu sync.RWMutex
}

// Global metrics instance
var globalMetrics *Metrics
var metricsMutex sync.Once

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	metricsMutex.Do(func() {
		globalMetrics = &Metrics{
			MultiPassCallsPerMessage: make(map[int]int64),
			LayerTimings:             make(map[string]*LayerTiming), // PHASE 2.1: Layer timing
		}
	})
	return globalMetrics
}



// RecordExtractionTime records extraction latency (Phase 1)
func (m *Metrics) RecordExtractionTime(ms int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExtractionTimeMs = append(m.ExtractionTimeMs, ms)
	if len(m.ExtractionTimeMs) > 10000 {
		// Keep only recent 10k samples to avoid memory bloat
		m.ExtractionTimeMs = m.ExtractionTimeMs[1:]
	}
}

// RecordExtractionLockSuccess records successful lock
func (m *Metrics) RecordExtractionLockSuccess() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExtractionsLocked++
}

// RecordExtractionLockFailure records failed lock attempt
func (m *Metrics) RecordExtractionLockFailure() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExtractionLockFailures++
	m.ExtractionsUnlocked++
}














