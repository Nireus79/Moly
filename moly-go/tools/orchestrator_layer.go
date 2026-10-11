package tools

import (
	"context"
)

// Layer is the interface that all 11-layer handlers implement
type Layer interface {
	// Name returns the layer's identifier (e.g., "Layer1", "Layer5")
	Name() string

	// Process executes the layer's logic and returns updated context
	// Returns error only for critical failures; normal gating/blocking is handled via ShouldStop flag
	Process(ctx context.Context, lc *LayerContext) (*LayerContext, error)

	// CanSkip returns true if this layer can be skipped for this context
	// Used to optimize processing by skipping layers with no work to do
	CanSkip(lc *LayerContext) bool

	// Priority returns the relative priority (0-100) for error handling
	// Higher priority = more critical
	Priority() int
}

// OrchestratorMetrics tracks performance data
type OrchestratorMetrics struct {
	MessagesProcessed int64            // Total messages
	AvgProcessingTime int64            // ms
	LayerTiming       map[string]int64 // Layer name -> time in ms
	LLMCalls          int64            // Total LLM calls
	CacheHits         int64            // Extraction cache hits
	LayerSkips        map[string]int64 // Layer name -> skip count
	Errors            int64            // Total errors
}

// NewOrchestratorMetrics creates metrics tracker
func NewOrchestratorMetrics() *OrchestratorMetrics {
	return &OrchestratorMetrics{
		LayerTiming: make(map[string]int64),
		LayerSkips:  make(map[string]int64),
	}
}

// RecordLayerTime records execution time for a layer
func (om *OrchestratorMetrics) RecordLayerTime(layerName string, milliseconds int64) {
	om.LayerTiming[layerName] = milliseconds
}

// RecordLayerSkip increments skip count for a layer
func (om *OrchestratorMetrics) RecordLayerSkip(layerName string) {
	om.LayerSkips[layerName]++
}



// RecordError increments error counter
func (om *OrchestratorMetrics) RecordError() {
	om.Errors++
}
