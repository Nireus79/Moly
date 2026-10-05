package agents

import (
	"context"
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/models"
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

	// FIX #11: Phase 3B - Check message summary cache for conflict detection
	// BUG FIX: High confidence means no conflicts (consistent extraction)
	if lc.HasMessageSummary(lc.MessageID) {
		summary := lc.GetMessageSummary(lc.MessageID)
		if msgSummary, ok := summary.(*models.MessageSummary); ok && msgSummary != nil {
			if msgSummary.Confidence >= 0.80 {
				log.Printf("[Layer5] FIX #11 BUG FIX: ✓ Using cached summary for %s (confidence=%.2f, clear intent=no conflicts)",
					lc.MessageID, msgSummary.Confidence)

				// High confidence extraction = clear intent = no conflicts
				lc.Layer5 = &tools.Layer5Result{
					DetectedConflicts:      []tools.Conflict{},
					ConflictCount:          0,
					CriticalConflicts:      []tools.Conflict{},
					ClarificationQuestions: []*database.ClarificationQuestion{},
				}
				log.Printf("[Layer5] ✓ Conflict detection complete (cached, duration=%.2fs)",
					time.Since(startTime).Seconds())
				return lc, nil
			}
		} else if msgSummary, ok := summary.(models.MessageSummary); ok {
			if msgSummary.Confidence >= 0.80 {
				log.Printf("[Layer5] FIX #11 BUG FIX: ✓ Using cached summary for %s (confidence=%.2f, clear intent=no conflicts)",
					lc.MessageID, msgSummary.Confidence)

				lc.Layer5 = &tools.Layer5Result{
					DetectedConflicts:      []tools.Conflict{},
					ConflictCount:          0,
					CriticalConflicts:      []tools.Conflict{},
					ClarificationQuestions: []*database.ClarificationQuestion{},
				}
				log.Printf("[Layer5] ✓ Conflict detection complete (cached, duration=%.2fs)",
					time.Since(startTime).Seconds())
				return lc, nil
			}
		}
	}

	// Detect conflicts from extraction
	extractionConflicts := make([]tools.Conflict, 0)
	var detectorResults []ConflictDetectorResult // Keep results for subject info
	if l5.detector != nil && lc.Analysis != nil {
		// Convert ConflictDetectorResult to tools.Conflict
		var err error
		// PHASE 5: Pass accumulated entities for cross-message contradiction detection
		detectorResults, err = l5.detector.DetectConflicts(
			ctx,
			lc.UserID,
			lc.ConversationID,
			lc.Analysis.ExtractedEntities,
			lc.AccumulatedExtractedEntities, // From previous messages
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

	// Generate clarification questions for conflicts if needed (FIX #4: USE SUBJECT FIELD)
	var clarificationQuestions []*database.ClarificationQuestion
	if len(criticalConflicts) > 0 && l5.handler != nil {
		// Generate clarification questions for each critical conflict
		for i, conflict := range criticalConflicts {
			var questionText string
			var subject string

			// Find the original detector result to get Subject field
			if i < len(detectorResults) {
				subject = detectorResults[i].Entity.Subject
			}

			// Add subject context to questions (FIX #4)
			subjectClause := ""
			if subject != "" && subject != "unknown" {
				if subject == "user" {
					subjectClause = " (about you)"
				} else {
					subjectClause = fmt.Sprintf(" (about %s)", subject)
				}
			}

			// Generate question text based on conflict type - NOW WITH SUBJECT CONTEXT
			switch conflict.Type {
			case "value_contradiction":
				questionText = fmt.Sprintf("I noticed you said %s earlier, but now you're saying %s%s. Can you help me understand the difference?",
					conflict.Description, conflict.Resolution, subjectClause)
			case "internal_contradiction":
				questionText = fmt.Sprintf("You mentioned that %s is both %s and %s%s - can you clarify which one is accurate?",
					subject, conflict.Description, conflict.Resolution, subjectClause)
			case "subject_mismatch":
				questionText = fmt.Sprintf("About %s - are we still talking about the same person? You mentioned %s before.%s",
					subject, conflict.Description, subjectClause)
			case "timeline_inconsistency":
				questionText = fmt.Sprintf("Help me understand the timing - you mentioned something different before%s about %s.",
					subjectClause, conflict.Description)
			case "characteristic_conflict":
				questionText = fmt.Sprintf("I want to make sure I understand%s: You said %s, but now you're saying %s. Which describes you better?",
					subjectClause, conflict.Description, conflict.Resolution)
			default:
				questionText = fmt.Sprintf("I want to make sure I understand correctly%s. Can you clarify: %s?",
					subjectClause, conflict.Description)
			}

			question := &database.ClarificationQuestion{
				ID:                fmt.Sprintf("conflict_clarif_%d_%d", time.Now().UnixNano(), i),
				ClarificationType: "conflict_resolution",
				Priority:          2,
				QuestionText:      questionText,
				ContextNotes:      fmt.Sprintf("Resolving %s: %s (subject=%s)", conflict.Type, conflict.Description, subject),
			}

			clarificationQuestions = append(clarificationQuestions, question)

			log.Printf("[Layer5] Generated clarification question for %s conflict%s: %s", conflict.Type, subjectClause, questionText)
		}
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
