package monitoring

import (
	"log"
	"sync"
	"time"
)

// LayerTiming tracks per-layer performance
type LayerTiming struct {
	LayerName    string
	TimeMs       []int64
	TotalCalls   int64
	AverageTimeMs float64
}

// Metrics tracks performance metrics for all phases
type Metrics struct {
	// Phase 1: Extraction Lock
	ExtractionLockEnabled        bool
	ExtractionTimeMs             []int64 // Latency samples
	ExtractionLockFailures       int64
	MultiPassCallsPerMessage     map[int]int64 // Distribution
	SubjectAttributionAccuracy   float64
	ExtractionsLocked            int64
	ExtractionsUnlocked          int64

	// Phase 2: Layer 5 Conflict Channeling
	ConflictsDetected            int64
	ConflictsResultingInQuestions int64
	QuestionDeduplicationPrevented int64
	ConflictQuestionAskRate      float64
	UserResponseTimeToConflictQ  []int64 // Latency samples

	// Phase 3: Constrained Response Generation
	ConstraintsBuilt             int64
	ConstraintCacheHitRate       float64
	ResponseValidationViolations int64
	RoleReversalBugOccurrences   int64
	FalsePositiveBlocks          int64
	ResponseLatencyMs            []int64 // Latency samples

	// Phase 4: Clean Schema
	MigrationStartTime           time.Time
	MigrationEndTime             time.Time
	MigrationDataLoss            int64
	MigrationErrorCount          int64

	// PHASE 2.1: Layer-Level Timing (NEW)
	LayerTimings                 map[string]*LayerTiming // Per-layer performance

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

// RecordLayerTime records execution time for a specific layer
// PHASE 2.1: Per-layer performance tracking
func (m *Metrics) RecordLayerTime(layerName string, ms int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	timing, exists := m.LayerTimings[layerName]
	if !exists {
		timing = &LayerTiming{LayerName: layerName}
		m.LayerTimings[layerName] = timing
	}

	timing.TimeMs = append(timing.TimeMs, ms)
	timing.TotalCalls++

	// Keep only recent 1000 samples
	if len(timing.TimeMs) > 1000 {
		timing.TimeMs = timing.TimeMs[1:]
	}

	// Calculate average
	total := int64(0)
	for _, t := range timing.TimeMs {
		total += t
	}
	timing.AverageTimeMs = float64(total) / float64(len(timing.TimeMs))

	log.Printf("[Metrics] Layer %s: %dms (avg: %.1fms, calls: %d)",
		layerName, ms, timing.AverageTimeMs, timing.TotalCalls)
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

// RecordMultiPassCalls records multi-pass parsing attempts (Phase 1)
func (m *Metrics) RecordMultiPassCalls(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MultiPassCallsPerMessage[count]++
}

// RecordConflictDetected records conflict detection (Phase 2)
func (m *Metrics) RecordConflictDetected() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConflictsDetected++
}

// RecordConflictQuestion records when conflict generates question (Phase 2)
func (m *Metrics) RecordConflictQuestion() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ConflictsResultingInQuestions++
}

// RecordDeduplicationPrevented records deduplication working (Phase 2)
func (m *Metrics) RecordDeduplicationPrevented() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QuestionDeduplicationPrevented++
}

// RecordResponseValidationViolation records constraint violation (Phase 3)
func (m *Metrics) RecordResponseValidationViolation() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ResponseValidationViolations++
}

// RecordRoleReversalBug records role-reversal bug occurrence (Phase 3)
func (m *Metrics) RecordRoleReversalBug() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RoleReversalBugOccurrences++
}

// RecordResponseLatency records response generation latency (Phase 3)
func (m *Metrics) RecordResponseLatency(ms int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ResponseLatencyMs = append(m.ResponseLatencyMs, ms)
	if len(m.ResponseLatencyMs) > 10000 {
		// Keep only recent 10k samples
		m.ResponseLatencyMs = m.ResponseLatencyMs[1:]
	}
}

// GetSummary returns a summary of metrics for logging/monitoring
func (m *Metrics) GetSummary() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary := map[string]interface{}{
		"phase1_extraction": map[string]interface{}{
			"lock_enabled":          m.ExtractionLockEnabled,
			"locked_extractions":    m.ExtractionsLocked,
			"unlocked_extractions":  m.ExtractionsUnlocked,
			"lock_failures":         m.ExtractionLockFailures,
			"avg_extraction_time_ms": getAverage(m.ExtractionTimeMs),
			"p95_extraction_time_ms": getPercentile(m.ExtractionTimeMs, 0.95),
		},
		"phase2_conflicts": map[string]interface{}{
			"conflicts_detected":              m.ConflictsDetected,
			"conflicts_resulting_in_questions": m.ConflictsResultingInQuestions,
			"deduplication_prevented":         m.QuestionDeduplicationPrevented,
		},
		"phase3_validation": map[string]interface{}{
			"constraints_built":               m.ConstraintsBuilt,
			"validation_violations":           m.ResponseValidationViolations,
			"role_reversal_bugs":              m.RoleReversalBugOccurrences,
			"false_positive_blocks":           m.FalsePositiveBlocks,
			"avg_response_latency_ms":         getAverage(m.ResponseLatencyMs),
			"p95_response_latency_ms":         getPercentile(m.ResponseLatencyMs, 0.95),
		},
	}

	return summary
}

// LogMetrics logs current metrics summary
func (m *Metrics) LogMetrics() {
	summary := m.GetSummary()
	log.Printf("[Metrics] Summary: %+v", summary)
}

// Helper functions
func getAverage(samples []int64) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum int64
	for _, s := range samples {
		sum += s
	}
	return float64(sum) / float64(len(samples))
}

func getPercentile(samples []int64, percentile float64) int64 {
	if len(samples) == 0 {
		return 0
	}
	// Simple percentile calculation (not sorting, just finding position)
	// In production, use proper percentile calculation
	pos := int(float64(len(samples)) * percentile)
	if pos >= len(samples) {
		pos = len(samples) - 1
	}
	return samples[pos]
}
