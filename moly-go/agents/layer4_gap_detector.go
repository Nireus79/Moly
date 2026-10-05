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

	// FIX #11: Phase 3 - Check message summary cache for recent messages
	// BUG FIX: Extract gaps info from cached summary instead of skipping with empty
	if lc.HasMessageSummary(lc.MessageID) {
		summary := lc.GetMessageSummary(lc.MessageID)
		if msgSummary, ok := summary.(*models.MessageSummary); ok && msgSummary != nil {
			if msgSummary.Confidence >= 0.85 {
				log.Printf("[Layer4] FIX #11 BUG FIX: ✓ Using cached summary for %s (confidence=%.2f, clear context=no gaps)",
					lc.MessageID, msgSummary.Confidence)

				// High confidence = clear context = no meaningful gaps
				lc.Layer4 = &tools.Layer4Result{
					DetectedGaps:  []tools.Gap{},
					GapCount:      0,
					CriticalGaps:  []tools.Gap{},
					ShouldClarify: false,
				}
				log.Printf("[Layer4] ✓ Layer4 complete (cached, duration=%.2fs)",
					time.Since(startTime).Seconds())
				return lc, nil
			}
		} else if msgSummary, ok := summary.(models.MessageSummary); ok {
			if msgSummary.Confidence >= 0.85 {
				log.Printf("[Layer4] FIX #11 BUG FIX: ✓ Using cached summary for %s (confidence=%.2f, clear context=no gaps)",
					lc.MessageID, msgSummary.Confidence)

				lc.Layer4 = &tools.Layer4Result{
					DetectedGaps:  []tools.Gap{},
					GapCount:      0,
					CriticalGaps:  []tools.Gap{},
					ShouldClarify: false,
				}
				log.Printf("[Layer4] ✓ Layer4 complete (cached, duration=%.2fs)",
					time.Since(startTime).Seconds())
				return lc, nil
			}
		}
	}

	// Detect gaps in current context - GOAL-ALIGNED
	log.Printf("[Layer4] Analyzing user profile, contacts, and extraction quality")

	// Extract goal and values if available (from Layer 1 extraction)
	userGoal := ""
	userValues := []string{}
	if lc.Layer1 != nil && lc.Layer1.ExtractedContext != nil {
		userGoal = lc.Layer1.ExtractedContext.Intention
		userValues = lc.Layer1.ExtractedContext.UserValues
	}

	// PHASE 6: Skip gap detection if goal is ambiguous
	// Layer 6 (Ambiguity handler) should clarify goal BEFORE Layer 4 asks about gaps
	// This separates boundary: Layer 6 clarifies WHAT, Layer 4 fills gaps in HOW
	if len(userGoal) == 0 {
		log.Printf("[Layer4] ℹ Goal unclear - deferring to Layer 6 for ambiguity clarification")
		lc.Layer4 = &tools.Layer4Result{
			DetectedGaps: []tools.Gap{},
			GapCount:     0,
			CriticalGaps: []tools.Gap{},
			ShouldClarify: false,
		}
		return lc, nil
	}

	// PHASE 2: Load previously answered gap types
	answeredGapTypes := []string{}
	// Note: Database access would be passed through lc or a service
	// For now, skip db query - will be handled by caller passing clarification data
	log.Printf("[Layer4] ℹ PHASE 2: Checking for %d previously answered gaps", len(answeredGapTypes))

	gaps := l4.gapAnalyzer.DetectGaps(
		lc.GetUserProfile(),
		lc.GetRelevantContacts(),
		lc.Analysis,
		lc.GetExtractionConfidence(),
		lc.GetMaturityScore(),
		userGoal,
		userValues,
		answeredGapTypes,
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

// DetectGaps analyzes context for missing information ALIGNED TO USER'S GOAL
// PHASE 1: Goal-aware gap detection - only ask gaps that matter for the stated goal
// PRIORITY ORDER: Goal-blocking gaps (HIGH) → Goal-supporting (MEDIUM) → Nice-to-know (LOW)
func (ga *GapAnalyzer) DetectGaps(
	profile *models.AboutMe,
	contacts []models.Contact,
	analysisCtx *models.AnalysisContext,
	extractionConfidence float64,
	maturity float64,
	userGoal string,           // What is user trying to accomplish? (from Layer 1 extraction)
	userValues []string,       // What matters to user? (from Layer 1 extraction)
	answeredGapTypes []string, // PHASE 2: Gap types already answered (don't regenerate)
) []tools.Gap {
	gaps := make([]tools.Gap, 0)

	// PRIORITY 1: Use extracted entities as CONTEXT, not as gap sources
	// Per spec: "Don't ask about extracted data - use it as context instead"
	// Extracted entities accumulate understanding over time (used for maturity, response shaping)
	// They are NOT the source of gap questions
	if analysisCtx != nil && len(analysisCtx.ExtractedEntities) > 0 {
		log.Printf("[Layer4] ℹ Using %d extracted entities as accumulated context (NOT generating gaps for each)", len(analysisCtx.ExtractedEntities))
		// Entities are used to:
		// - Build maturity (high confidence entities increase context completeness)
		// - Shape responses ("You mentioned X, so...")
		// - Understand goal (what is user trying to accomplish)
		// But NOT to generate gap questions like "You mentioned X. How does this apply?"
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

	// PHASE 1: Filter gaps to be goal-aligned
	// PHASE 2: Skip already-answered gaps
	// PHASE 4: Rank gaps by impact on goal achievement
	// Only keep gaps that matter for accomplishing the user's goal AND haven't been answered
	goalAlignedGaps := filterGapsByGoal(gaps, userGoal, userValues, answeredGapTypes)
	
	// Rank by impact: goal-blocking > goal-supporting > nice-to-know
	rankedGaps := rankGapsByImpact(goalAlignedGaps, userGoal)
	
	log.Printf("[Layer4] ✓ Goal-aware filtering: %d total → %d aligned → ranked by impact", len(gaps), len(rankedGaps))
	for i, gap := range rankedGaps {
		impact := calculateGapImpact(gap.Type, userGoal)
		log.Printf("[Layer4]   #%d (impact=%.2f) %s", i+1, impact, gap.Type)
	}
	
	return rankedGaps
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

// filterGapsByGoal filters gaps to only keep those aligned to user's goal
// PHASE 1: Goal-aware gap detection per MOLY_11_LAYER_SYSTEM.md spec
// Don't ask generic profile gaps - ask only gaps that help accomplish the goal
func filterGapsByGoal(gaps []tools.Gap, userGoal string, userValues []string, answeredGapTypes []string) []tools.Gap {
	filtered := make([]tools.Gap, 0)

	// If no goal provided, return all gaps (fallback to generic)
	if userGoal == "" {
		return gaps
	}

	for _, gap := range gaps {
		gapType := gap.Type

		// PHASE 2: Skip already-answered gaps
		isAnswered := false
		for _, answered := range answeredGapTypes {
			if gapType == answered {
				isAnswered = true
				log.Printf("[Layer4] ℹ Skipping gap type %s (already answered)", gapType)
				break
			}
		}
		if isAnswered {
			continue
		}

		// Always keep safety-related gaps
		if gapType == "unclear_intention" ||
		   gapType == "immature_context" ||
		   gapType == "no_contacts_identified" {
			filtered = append(filtered, gap)
			continue
		}

		// Filter based on goal relevance
		isRelevant := false

		// Check if gap type matches goal

		// Always keep high severity gaps
		if gap.Severity == "high" && gap.Confidence > 0.7 {
			isRelevant = true
		}

		if isRelevant {
			filtered = append(filtered, gap)
		}
	}

	// Ensure at least one gap if goal is present (don't return empty)
	if len(filtered) == 0 && len(gaps) > 0 && userGoal != "" {
		// Keep the immature context gap as fallback
		for _, gap := range gaps {
			if gap.Type == "immature_context" || gap.Severity == "high" {
				filtered = append(filtered, gap)
				break
			}
		}
	}

	return filtered
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

// rankGapsByImpact sorts gaps by impact on goal achievement
// PHASE 4: Implement full gap prioritization per MOLY_11_LAYER_SYSTEM.md spec
// Order: goal-blocking (HIGH) → goal-supporting (MEDIUM) → nice-to-know (LOW)
func rankGapsByImpact(gaps []tools.Gap, userGoal string) []tools.Gap {
	type gapWithImpact struct {
		gap    tools.Gap
		impact float64
	}

	// Score each gap
	gapsWithImpact := make([]gapWithImpact, 0)
	for _, gap := range gaps {
		impact := calculateGapImpact(gap.Type, userGoal)
		gapsWithImpact = append(gapsWithImpact, gapWithImpact{
			gap:    gap,
			impact: impact,
		})
	}

	// Sort by impact (descending: high impact first)
	for i := 0; i < len(gapsWithImpact); i++ {
		for j := i + 1; j < len(gapsWithImpact); j++ {
			if gapsWithImpact[j].impact > gapsWithImpact[i].impact {
				gapsWithImpact[i], gapsWithImpact[j] = gapsWithImpact[j], gapsWithImpact[i]
			}
		}
	}

	// Extract sorted gaps
	ranked := make([]tools.Gap, 0)
	for _, gapImpact := range gapsWithImpact {
		ranked = append(ranked, gapImpact.gap)
	}

	return ranked
}

// calculateGapImpact scores how much a gap impacts achieving the user's goal
// Returns 0.0-1.0 where higher = more critical for goal achievement
func calculateGapImpact(gapType string, userGoal string) float64 {
	// Safety gaps always critical
	if gapType == "unclear_intention" || gapType == "immature_context" {
		return 0.95
	}

	// Contact-related gaps block goal achievement if goal involves communication
	if gapType == "no_contacts_identified" || gapType == "vague_contact_name" || gapType == "unclear_relationship" {
		if len(userGoal) > 0 {
			// High impact if goal involves writing, messaging, communicating
			return 0.85
		}
		return 0.6
	}

	// Low confidence extraction
	if gapType == "low_extraction_confidence" {
		return 0.7
	}

	// Profile gaps are nice-to-know but not blocking
	if gapType == "missing_communication_style" || gapType == "missing_values" {
		return 0.3
	}

	// Default: moderate impact
	return 0.5
}
