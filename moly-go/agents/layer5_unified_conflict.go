package agents

import (
	"context"
	"log"
	"time"

	"moly/database"
	"moly/tools"
)

// Layer5UnifiedConflictDetection consolidates conflict detection from extraction and validation
// Unifies conflicts from multiple sources into a single clear view
type Layer5UnifiedConflictDetection struct {
	detector *ConflictDetector
	handler  *Layer5ConflictHandler
}

// NewLayer5UnifiedConflictDetection creates unified conflict detection layer
func NewLayer5UnifiedConflictDetection(
	detector *ConflictDetector,
	handler *Layer5ConflictHandler,
) *Layer5UnifiedConflictDetection {
	return &Layer5UnifiedConflictDetection{
		detector: detector,
		handler:  handler,
	}
}

// Name returns the layer identifier
func (l5 *Layer5UnifiedConflictDetection) Name() string {
	return "Layer5-UnifiedConflictDetection"
}

// Priority returns layer priority
func (l5 *Layer5UnifiedConflictDetection) Priority() int {
	return 60 // Medium priority
}

// CanSkip returns true if no gaps (nothing to conflict with)
func (l5 *Layer5UnifiedConflictDetection) CanSkip(lc *tools.LayerContext) bool {
	// Skip if no gaps and low ambiguity
	if lc.Layer4 != nil && lc.Layer4.GapCount == 0 {
		if lc.Layer6 != nil && !lc.Layer6.IsAmbiguous {
			return true
		}
	}

	// Skip if immature context (insufficient data for conflicts)
	if lc.Layer3 != nil && lc.Layer3.MaturityScore < 0.2 {
		return true
	}

	return false
}

// Process executes unified conflict detection
func (l5 *Layer5UnifiedConflictDetection) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()

	// Detect conflicts from extraction
	extractionConflicts := make([]tools.Conflict, 0)
	if l5.detector != nil && lc.Analysis != nil {
		// Convert ConflictDetectorResult to tools.Conflict
		detectorResults, err := l5.detector.DetectConflicts(
			ctx,
			lc.UserID,
			lc.ConversationID,
			lc.Analysis.ExtractedEntities,
		)

		if err == nil {
			for _, result := range detectorResults {
				extractionConflicts = append(extractionConflicts, tools.Conflict{
					Type:        result.Type,
					Severity:    result.Severity,
					Confidence:  result.Confidence,
					Description: result.Description,
					Resolution:  result.Resolution,
				})
			}
		} else {
			log.Printf("[Layer5] ⚠️ Conflict detection error: %v", err)
		}
	}

	// Identify critical conflicts (those that need resolution)
	criticalConflicts := filterCriticalConflicts(extractionConflicts)

	// Generate clarification questions for conflicts if needed
	var clarificationQuestions []*database.ClarificationQuestion
	if len(criticalConflicts) > 0 && l5.handler != nil {
		// Handler generates questions for conflicts
		// Note: Simplified for now - actual implementation would pass conflicts to handler
		clarificationQuestions = make([]*database.ClarificationQuestion, 0)
	}

	// Store results
	lc.Layer5 = &tools.Layer5Result{
		DetectedConflicts:      extractionConflicts,
		ConflictCount:          len(extractionConflicts),
		CriticalConflicts:      criticalConflicts,
		ClarificationQuestions: clarificationQuestions,
	}

	log.Printf("[Layer5] ✓ Conflict detection complete (total=%d, critical=%d, duration=%.2fs)",
		len(extractionConflicts), len(criticalConflicts), time.Since(startTime).Seconds())

	return lc, nil
}

// Helper: Filter conflicts that are critical (prevent Layer 5+)
func filterCriticalConflicts(conflicts []tools.Conflict) []tools.Conflict {
	critical := make([]tools.Conflict, 0)

	for _, conflict := range conflicts {
		if conflict.Severity == "critical" || (conflict.Severity == "high" && conflict.Confidence > 0.8) {
			critical = append(critical, conflict)
		}
	}

	return critical
}
