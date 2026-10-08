package tools

import (
	"context"
	"fmt"
	"log"
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

// LayerOrchestrator coordinates execution of all 11 layers
type LayerOrchestrator struct {
	layers    []Layer
	cache     *ExtractionCache
	metrics   *OrchestratorMetrics
	debugMode bool
}

// NewLayerOrchestrator creates a new orchestrator
func NewLayerOrchestrator() *LayerOrchestrator {
	return &LayerOrchestrator{
		layers:    make([]Layer, 0),
		cache:     NewExtractionCache(),
		metrics:   NewOrchestratorMetrics(),
		debugMode: false,
	}
}

// AddLayer registers a layer with the orchestrator
func (lo *LayerOrchestrator) AddLayer(layer Layer) error {
	if layer == nil {
		return fmt.Errorf("cannot add nil layer")
	}
	lo.layers = append(lo.layers, layer)
	log.Printf("[LayerOrchestrator] ✓ Registered %s (priority=%d)", layer.Name(), layer.Priority())
	return nil
}

// ProcessMessage executes all layers in sequence
func (lo *LayerOrchestrator) ProcessMessage(
	ctx context.Context,
	lc *LayerContext,
) (*LayerContext, error) {
	if lc == nil {
		return nil, fmt.Errorf("context is nil")
	}

	if len(lo.layers) == 0 {
		return nil, fmt.Errorf("no layers registered")
	}

	log.Printf("[LayerOrchestrator] Starting message processing (%d layers)", len(lo.layers))

	// Process each layer in sequence
	var lastError error
	for i, layer := range lo.layers {
		// Check if we should stop
		if lc.ShouldStop {
			log.Printf("[LayerOrchestrator] ⏹️ Stopping at layer %d (%s): %s", i+1, layer.Name(), lc.StopReason)
			break
		}

		// Check if layer can be skipped
		if layer.CanSkip(lc) {
			if lo.debugMode {
				log.Printf("[LayerOrchestrator] ⊘ Skipping %s (no work to do)", layer.Name())
			}
			continue
		}

		// Process layer
		if lo.debugMode {
			log.Printf("[LayerOrchestrator] ▶ Processing %s...", layer.Name())
		}

		result, err := layer.Process(ctx, lc)
		if err != nil {
			log.Printf("[LayerOrchestrator] ⚠️ %s error: %v", layer.Name(), err)
			lastError = err

			// For high-priority layers, stop immediately on error
			if layer.Priority() > 80 {
				log.Printf("[LayerOrchestrator] Stopping due to high-priority layer error")
				lc.StopReason = fmt.Sprintf("layer_error: %v", err)
				lc.ShouldStop = true
				break
			}

			// For low-priority layers, continue (graceful degradation)
			continue
		}

		// Update context
		lc = result

		if lo.debugMode {
			log.Printf("[LayerOrchestrator] ✓ %s complete (shouldStop=%v)", layer.Name(), lc.ShouldStop)
		}
	}

	log.Printf("[LayerOrchestrator] ✓ Message processing complete (stopped=%v)", lc.ShouldStop)

	// If we had errors but still have a context, return it (graceful degradation)
	if lastError != nil && lc != nil {
		log.Printf("[LayerOrchestrator] Returning context despite error: %v", lastError)
	}

	return lc, lastError
}

// SetDebugMode enables/disables debug logging
func (lo *LayerOrchestrator) SetDebugMode(enabled bool) {
	lo.debugMode = enabled
}

// GetMetrics returns performance metrics
func (lo *LayerOrchestrator) GetMetrics() *OrchestratorMetrics {
	return lo.metrics
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

// RecordLLMCall increments LLM call counter
func (om *OrchestratorMetrics) RecordLLMCall() {
	om.LLMCalls++
}

// RecordCacheHit increments cache hit counter
func (om *OrchestratorMetrics) RecordCacheHit() {
	om.CacheHits++
}

// RecordError increments error counter
func (om *OrchestratorMetrics) RecordError() {
	om.Errors++
}
