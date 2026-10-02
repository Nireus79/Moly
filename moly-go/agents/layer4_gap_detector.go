package agents

import (
	"context"
	"log"
	"time"

	"moly/models"
	"moly/tools"
)

// Layer4GapDetector detects missing context needed for deeper processing
// Gaps indicate what information we still need before Layer 5+ operations
type Layer4GapDetector struct {
	gapAnalyzer *GapAnalyzer
}

// GapAnalyzer detects gaps in context
type GapAnalyzer struct {
	minMaturityForL5Plus float64 // Minimum maturity to proceed to Layer 5+
}

// NewLayer4GapDetector creates a new gap detection layer
func NewLayer4GapDetector() *Layer4GapDetector {
	return &Layer4GapDetector{
		gapAnalyzer: &GapAnalyzer{
			minMaturityForL5Plus: 0.3, // 30% maturity needed for Layer 5+
		},
	}
}

// Name returns the layer identifier
func (l4 *Layer4GapDetector) Name() string {
	return "Layer4-GapDetection"
}

// Priority returns layer priority
func (l4 *Layer4GapDetector) Priority() int {
	return 70 // Medium priority - gaps don't block pipeline, they inform decisions
}

// CanSkip returns false - Layer 4 always analyzes
func (l4 *Layer4GapDetector) CanSkip(lc *tools.LayerContext) bool {
	return false
}

// Process executes gap detection
func (l4 *Layer4GapDetector) Process(ctx context.Context, lc *tools.LayerContext) (*tools.LayerContext, error) {
	startTime := time.Now()
	log.Printf("[Layer4] ▶ Starting gap detection (maturity=%.2f, extraction_confidence=%.2f)",
		lc.GetMaturityScore(), lc.GetExtractionConfidence())

	// Detect gaps in current context
	log.Printf("[Layer4] Analyzing user profile, contacts, and extraction quality")
	gaps := l4.gapAnalyzer.DetectGaps(
		lc.GetUserProfile(),
		lc.GetRelevantContacts(),
		lc.Analysis,
		lc.GetExtractionConfidence(),
		lc.GetMaturityScore(),
	)
	log.Printf("[Layer4] ✓ Detected %d total gaps", len(gaps))

	// Determine if gaps are critical (prevent Layer 5+)
	criticalGaps := filterCriticalGaps(gaps)
	log.Printf("[Layer4] Gap severity: %d critical, %d non-critical", len(criticalGaps), len(gaps)-len(criticalGaps))

	if len(criticalGaps) > 0 {
		log.Printf("[Layer4] ⚠ Critical gaps found:")
		for i, g := range criticalGaps {
			log.Printf("[Layer4]   %d. %s (severity=%s)", i+1, g.Description, g.Severity)
		}
	}

	// Store results
	lc.Layer4 = &tools.Layer4Result{
		DetectedGaps:   gaps,
		GapCount:       len(gaps),
		CriticalGaps:   criticalGaps,
		ShouldClarify:  len(criticalGaps) > 0,
	}

	duration := time.Since(startTime).Seconds()
	log.Printf("[Layer4] ✓ Layer4 complete (gaps=%d, critical=%d, should_clarify=%v, duration=%.2fs)",
		len(gaps), len(criticalGaps), len(criticalGaps) > 0, duration)

	return lc, nil
}

// analyzeExtractedEntitiesForGaps identifies what's unclear or needs application context
// in the entities extracted from the user's message
func analyzeExtractedEntitiesForGaps(entities []models.ExtractedEntity) []tools.Gap {
	gaps := make([]tools.Gap, 0)

	for _, entity := range entities {
		// For each extracted entity, determine if it needs clarification or application context
		switch entity.Type {
		case "preference":
			// User expressed a preference - does it need context application?
			gap := tools.Gap{
				Type:        "extracted_preference_needs_context",
				Description: "You mentioned preferring " + entity.Value + ". How does this apply to your specific situation?",
				Severity:    "high",
				Confidence:  entity.Confidence,
			}
			gaps = append(gaps, gap)

		case "characteristic":
			// User described themselves - does it need clarification?
			gap := tools.Gap{
				Type:        "extracted_characteristic_needs_context",
				Description: "You described yourself as " + entity.Value + ". How does this inform your approach here?",
				Severity:    "high",
				Confidence:  entity.Confidence,
			}
			gaps = append(gaps, gap)

		case "negation":
			// User said what they DON'T want - need to clarify what they DO want
			gap := tools.Gap{
				Type:        "extracted_negation_needs_clarification",
				Description: "You said you don't prefer " + entity.Value + ". What would you prefer instead?",
				Severity:    "medium",
				Confidence:  entity.Confidence,
			}
			gaps = append(gaps, gap)
		}
	}

	return gaps
}

// DetectGaps analyzes context for missing information
// PRIORITY ORDER: Extracted data gaps (HIGH) → Profile gaps (MEDIUM) → Contact gaps (LOW)
func (ga *GapAnalyzer) DetectGaps(
	profile *models.AboutMe,
	contacts []models.Contact,
	analysisCtx *models.AnalysisContext,
	extractionConfidence float64,
	maturity float64,
) []tools.Gap {
	gaps := make([]tools.Gap, 0)

	// PRIORITY 1: Analyze extracted entities (what was just learned in THIS conversation)
	// These have HIGH priority because they're fresh, contextual data
	if analysisCtx != nil && len(analysisCtx.ExtractedEntities) > 0 {
		extractedGaps := analyzeExtractedEntitiesForGaps(analysisCtx.ExtractedEntities)
		gaps = append(gaps, extractedGaps...)
		if len(extractedGaps) > 0 {
			log.Printf("[Layer4] ✓ Detected %d gaps from extracted entities", len(extractedGaps))
		}
	}

	// PRIORITY 2: Gap 1: Missing user profile information (lower priority than extracted)
	if profile == nil || profile.UserID == "" {
		gaps = append(gaps, tools.Gap{
			Type:        "missing_user_profile",
			Description: "No user profile data (communication style, values, goals)",
			Severity:    "high",
			Confidence:  1.0,
		})
	} else {
		// Check specific profile gaps
		if profile.CommunicationStyle == "" {
			gaps = append(gaps, tools.Gap{
				Type:        "missing_communication_style",
				Description: "Communication style not defined (casual, formal, direct, etc.)",
				Severity:    "medium",
				Confidence:  0.9,
			})
		}
		if len(profile.Values) == 0 {
			gaps = append(gaps, tools.Gap{
				Type:        "missing_values",
				Description: "Personal values not identified",
				Severity:    "medium",
				Confidence:  0.8,
			})
		}
	}

	// Gap 2: Unclear contact information
	if len(contacts) == 0 {
		gaps = append(gaps, tools.Gap{
			Type:        "no_contacts_identified",
			Description: "No contacts identified in conversation",
			Severity:    "medium",
			Confidence:  0.7,
		})
	} else {
		// Check for vague contact names
		for _, contact := range contacts {
			if isVagueContactName(contact.Name) {
				gaps = append(gaps, tools.Gap{
					Type:        "vague_contact_name",
					Description: "Contact name is vague (e.g., 'the girl', 'my friend', not a real name)",
					Severity:    "low",
					Confidence:  0.6,
				})
			}
			if contact.Relationship == "" {
				gaps = append(gaps, tools.Gap{
					Type:        "unclear_relationship",
					Description: "Contact relationship type unclear",
					Severity:    "low",
					Confidence:  0.5,
				})
			}
		}
	}

	// Gap 3: Unclear intention
	if analysisCtx != nil && analysisCtx.CurrentMessage == "" {
		gaps = append(gaps, tools.Gap{
			Type:        "unclear_intention",
			Description: "User intention in this message unclear",
			Severity:    "high",
			Confidence:  0.8,
		})
	}

	// Gap 4: Low extraction confidence
	if extractionConfidence < 0.5 {
		gaps = append(gaps, tools.Gap{
			Type:        "low_extraction_confidence",
			Description: "Extracted information has low confidence score",
			Severity:    "medium",
			Confidence:  0.9,
		})
	}

	// Gap 5: Immature context
	if maturity < 0.3 {
		gaps = append(gaps, tools.Gap{
			Type:        "immature_context",
			Description: "Context is immature - insufficient information gathered",
			Severity:    "medium",
			Confidence:  1.0,
		})
	}

	return gaps
}

// Helper: Check if contact name is vague
func isVagueContactName(name string) bool {
	vaguePatterns := map[string]bool{
		"the girl":     true,
		"the guy":      true,
		"my friend":    true,
		"my ex":        true,
		"my boss":      true,
		"this person":  true,
		"someone":      true,
		"they":         true,
		"he":           true,
		"she":          true,
		"it":           true,
	}

	return vaguePatterns[name]
}

// Helper: Filter gaps that are critical (prevent Layer 5+)
func filterCriticalGaps(gaps []tools.Gap) []tools.Gap {
	critical := make([]tools.Gap, 0)

	for _, gap := range gaps {
		if gap.Severity == "high" || (gap.Severity == "medium" && gap.Confidence > 0.8) {
			critical = append(critical, gap)
		}
	}

	return critical
}
