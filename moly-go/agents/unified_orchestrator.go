package agents

import (
	"context"
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/models"
	"moly/storage"
	"moly/tools"
)

// UnifiedOrchestrator coordinates all 11 layers of message processing
// It takes the fragmented components and wires them into a unified pipeline
type UnifiedOrchestrator struct {
	// Layers (initialized in order)
	layers []tools.Layer

	// Infrastructure
	cache             *tools.ExtractionCache
	metrics           *tools.OrchestratorMetrics
	debugMode         bool

	// Dependencies
	contextExtractor      *ContextExtractor
	constitutionalEval    *tools.ConstitutionalEvaluator
	maturityService       *storage.MaturityService
	conflictDetector      *ConflictDetector
	layer5ConflictHandler *Layer5ConflictHandler

	// Database
	db *database.Database
}

// NewUnifiedOrchestrator creates a new orchestrator with all dependencies
func NewUnifiedOrchestrator(
	contextExtractor *ContextExtractor,
	constitutionalEval *tools.ConstitutionalEvaluator,
	maturityService *storage.MaturityService,
	conflictDetector *ConflictDetector,
	layer5ConflictHandler *Layer5ConflictHandler,
	db *database.Database,
) *UnifiedOrchestrator {
	orch := &UnifiedOrchestrator{
		layers:                make([]tools.Layer, 0),
		cache:                 tools.NewExtractionCache(),
		metrics:               tools.NewOrchestratorMetrics(),
		debugMode:             false,
		contextExtractor:      contextExtractor,
		constitutionalEval:    constitutionalEval,
		maturityService:       maturityService,
		conflictDetector:      conflictDetector,
		layer5ConflictHandler: layer5ConflictHandler,
		db:                    db,
	}

	// Initialize layers in order
	orch.initializeLayers()

	return orch
}

// initializeLayers registers all layers in the correct order
func (uo *UnifiedOrchestrator) initializeLayers() {
	log.Printf("[UnifiedOrchestrator] Initializing 11-layer pipeline")

	// Phase 0-1: Extraction (FIXED: properly initialized with dependencies)
	layer1Adapter := NewLayer1ContextExtractionAdapter(
		uo.contextExtractor,
		uo.cache,
	)
	uo.addLayer(layer1Adapter)

	// Phase 1-2: Principle Checking
	uo.addLayer(NewLayer2PrincipleCheckAdapter(uo.constitutionalEval))

	// Phase 1-3: Maturity Assessment
	uo.addLayer(NewLayer3MaturityAssessmentAdapter(uo.maturityService))

	// Layer 4: Gap Detection
	uo.addLayer(NewLayer4GapDetector())

	// Layer 5: Conflict Detection
	uo.addLayer(NewLayer5UnifiedConflictDetection(
		uo.conflictDetector,
		uo.layer5ConflictHandler,
	))

	// Layer 6: Ambiguous Request Handling
	uo.addLayer(NewLayer6AmbiguousRequestHandler())

	// Layer 7: Principle Violation Clarification
	uo.addLayer(NewLayer7PrincipleViolationClarification())

	// Layer 8: Socratic Deepening
	uo.addLayer(NewLayer8SocraticDeepening())

	// Layer 9: Topic Shift Detection
	uo.addLayer(NewLayer9TopicShiftDetection())

	// Layer 10: Persistent Questioning
	uo.addLayer(NewLayer10PersistentQuestioning())

	// Layer 11: Denial Protocol
	uo.addLayer(NewLayer11DenialProtocol())

	log.Printf("[UnifiedOrchestrator] ✓ Pipeline initialized with %d layers", len(uo.layers))
}

// addLayer registers a layer (internal)
func (uo *UnifiedOrchestrator) addLayer(layer tools.Layer) {
	uo.layers = append(uo.layers, layer)
}

// ProcessMessage runs all layers in sequence for a message
func (uo *UnifiedOrchestrator) ProcessMessage(
	ctx context.Context,
	message string,
	userID string,
	conversationID string,
	messageID string,
	analysisCtx *models.AnalysisContext,
) (*tools.LayerContext, error) {
	startTime := time.Now()

	if message == "" {
		return nil, fmt.Errorf("message cannot be empty")
	}

	if analysisCtx == nil {
		return nil, fmt.Errorf("analysis context cannot be nil")
	}

	// LOOP PATTERN: Detect if this is a clarification response to a previous question
	// If so, skip Layers 1-3 and jump to Layer 4 with accumulated context
	isAnsweringClarification := analysisCtx.CurrentMessage != "" &&
		analysisCtx.ExtractedConfidence > 0 &&
		len(analysisCtx.ExtractedEntities) > 0

	startLayer := 0 // Default: start from Layer 1
	if isAnsweringClarification {
		log.Printf("[UnifiedOrchestrator] 🔄 LOOP PATTERN: Clarification detected - jumping to Layer 4")
		startLayer = 3 // Layer 4 is at index 3 (0-indexed: L1=0, L2=1, L3=2, L4=3)
	}

	// Create layer context
	lc := tools.NewLayerContext(analysisCtx, userID, messageID, conversationID)

	if uo.debugMode {
		log.Printf("[UnifiedOrchestrator] Starting message processing (user=%s, msgID=%s, startLayer=%d, isClarification=%v)",
			userID, messageID, startLayer+1, isAnsweringClarification)
	}

	// Run each layer in sequence
	var lastError error
	layersExecuted := 0
	layersSkipped := 0

	for i, layer := range uo.layers {
		// LOOP PATTERN: Skip Layers 1-3 if answering clarification
		if i < startLayer {
			if uo.debugMode {
				log.Printf("[UnifiedOrchestrator] ⊘ Loop pattern: Skipping %s (clarification mode, using accumulated context)", layer.Name())
			}
			uo.metrics.RecordLayerSkip(layer.Name())
			layersSkipped++
			continue
		}
		// Check if we should stop early
		if lc.ShouldStop {
			if uo.debugMode {
				log.Printf("[UnifiedOrchestrator] ⏹️ Stopping at layer %d (%s): %s", i+1, layer.Name(), lc.StopReason)
			}
			break
		}

		// Check if layer can be skipped
		if layer.CanSkip(lc) {
			if uo.debugMode {
				log.Printf("[UnifiedOrchestrator] ⊘ Skipping %s (no work)", layer.Name())
			}
			uo.metrics.RecordLayerSkip(layer.Name())
			layersSkipped++
			continue
		}

		// Execute layer
		layerStart := time.Now()

		result, err := layer.Process(ctx, lc)
		if err != nil {
			log.Printf("[UnifiedOrchestrator] ⚠️ Layer %s error: %v", layer.Name(), err)
			lastError = err
			uo.metrics.RecordError()

			// For critical layers, stop immediately
			if layer.Priority() > 80 {
				log.Printf("[UnifiedOrchestrator] Stopping due to critical layer error")
				lc.StopReason = fmt.Sprintf("layer_error: %s", layer.Name())
				lc.ShouldStop = true
				break
			}

			// For non-critical layers, continue (graceful degradation)
			continue
		}

		// Update context
		lc = result

		// ARCHITECTURAL FIX #2: Use phase-aware proportional gating (not hardcoded threshold)
		// After Layer 3 (Maturity): Get phase and apply proportional severity gate
		if i == 2 && lc.Layer3 != nil { // Layer 3 (index 2)
			maturity := lc.Layer3.MaturityScore

			// Use maturity calculator to get phase-aware gate
			mc := tools.NewMaturityCalculator()
			currentPhase := mc.EstimateCurrentPhase(maturity)
			severityGate := mc.GetEvaluationSeverityGate(maturity)

			log.Printf("[UnifiedOrchestrator] ✓ Maturity Analysis: score=%.2f, phase=%s, severity_gate=%.2f",
				maturity, currentPhase, severityGate)

			// Store for Layer 4 to use in gap filtering
			lc.MaturityPhase = currentPhase
			lc.MaturitySeverityGate = severityGate
		}

		// After Layer 4 (Gap Detection): Filter gaps by severity gate (proportional, not block)
		if i == 3 && lc.Layer4 != nil && lc.Layer4.ShouldClarify { // Layer 4 (index 3)
			severityGate := lc.MaturitySeverityGate

			// Filter gaps: only keep those with sufficient confidence for this phase
			// This uses the proportional gate, not a binary block
			filtered := []tools.Gap{}
			for _, gap := range lc.Layer4.DetectedGaps {
				if gap.Confidence >= severityGate {
					filtered = append(filtered, gap)
				}
			}

			log.Printf("[UnifiedOrchestrator] ✓ Gap Filtering: %d gaps → %d after severity gate (%.2f)",
				len(lc.Layer4.DetectedGaps), len(filtered), severityGate)

			// Replace gaps with filtered ones (removes low-confidence gaps for this phase)
			lc.Layer4.DetectedGaps = filtered

			// Continue processing - do NOT block the pipeline
			// Layer 6+ will use these filtered gaps for clarification
		}

		// Record metrics
		layerDuration := time.Since(layerStart).Milliseconds()
		uo.metrics.RecordLayerTime(layer.Name(), layerDuration)
		layersExecuted++

		if uo.debugMode {
			log.Printf("[UnifiedOrchestrator] ✓ %s complete (duration=%dms)", layer.Name(), layerDuration)
		}
	}

	totalTime := time.Since(startTime).Milliseconds()
	uo.metrics.MessagesProcessed++

	log.Printf("[UnifiedOrchestrator] ✓ Message processing complete (layers=%d/%d, skipped=%d, total_time=%dms, stopped=%v)",
		layersExecuted, len(uo.layers), layersSkipped, totalTime, lc.ShouldStop)

	// Return context and any error encountered
	return lc, lastError
}

// GetMetrics returns performance metrics
func (uo *UnifiedOrchestrator) GetMetrics() *tools.OrchestratorMetrics {
	return uo.metrics
}

// SetDebugMode enables/disables debug logging
func (uo *UnifiedOrchestrator) SetDebugMode(enabled bool) {
	uo.debugMode = enabled
	log.Printf("[UnifiedOrchestrator] Debug mode: %v", enabled)
}

// GetCache returns the extraction cache
func (uo *UnifiedOrchestrator) GetCache() *tools.ExtractionCache {
	return uo.cache
}

// SetLayer1Adapter updates the Layer 1 adapter (for testing/customization)
func (uo *UnifiedOrchestrator) SetLayer1Adapter(adapter tools.Layer) error {
	if len(uo.layers) == 0 {
		return fmt.Errorf("no layers registered")
	}

	// Replace first layer (should be Layer 1)
	uo.layers[0] = adapter
	log.Printf("[UnifiedOrchestrator] ✓ Updated Layer 1 adapter")
	return nil
}

// AddLayer adds a new layer to the pipeline (for building out Layers 4-11)
func (uo *UnifiedOrchestrator) AddLayer(layer tools.Layer) error {
	if layer == nil {
		return fmt.Errorf("cannot add nil layer")
	}

	uo.layers = append(uo.layers, layer)
	log.Printf("[UnifiedOrchestrator] ✓ Added layer %s (total: %d)", layer.Name(), len(uo.layers))
	return nil
}

// GetLayerCount returns number of registered layers
func (uo *UnifiedOrchestrator) GetLayerCount() int {
	return len(uo.layers)
}

// ListLayers returns names of all registered layers
func (uo *UnifiedOrchestrator) ListLayers() []string {
	names := make([]string, 0, len(uo.layers))
	for _, layer := range uo.layers {
		names = append(names, layer.Name())
	}
	return names
}
