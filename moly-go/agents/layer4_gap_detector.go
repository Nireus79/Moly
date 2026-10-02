package agents

import (
	"context"
	"log"
	"strings"
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

// DetectGaps analyzes context for missing information
func (ga *GapAnalyzer) DetectGaps(
	profile *models.AboutMe,
	contacts []models.Contact,
	analysisCtx *models.AnalysisContext,
	extractionConfidence float64,
	maturity float64,
) []tools.Gap {
	gaps := make([]tools.Gap, 0)

	// Gap 1: Missing user profile information
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

	// NEW: Gap 6-8: Message-specific gaps (vague/ambiguous claims in the message)
	if analysisCtx != nil && analysisCtx.CurrentMessage != "" {
		msg := analysisCtx.CurrentMessage

		// Check for vague common ground claims
		if containsPhrase(msg, "things in common", "have in common", "similar") {
			if !containsPhrase(msg, "specifically", "like", "such as", "for example") {
				gaps = append(gaps, tools.Gap{
					Type:        "vague_common_ground",
					Description: "User mentions shared interests but doesn't specify what they are",
					Severity:    "high",
					Confidence:  0.85,
				})
			}
		}

		// Check for context-dependent tone requests
		if containsPhrase(msg, "playful", "smart", "casual", "formal") {
			if containsPhrase(msg, "dynamic", "dom", "sub", "bdsm") {
				gaps = append(gaps, tools.Gap{
					Type:        "tone_in_context",
					Description: "User requests specific tone/style but needs clarification on how it applies to this relationship dynamic",
					Severity:    "high",
					Confidence:  0.80,
				})
			}
		}

		// Check for relationship dynamic without clarification
		if containsPhrase(msg, "dominant", "submissive", "dom", "sub") {
			if !containsPhrase(msg, "approach", "how to", "way to", "style of") {
				gaps = append(gaps, tools.Gap{
					Type:        "dynamic_approach",
					Description: "User mentions D/s dynamic but hasn't clarified how this should influence the message approach",
					Severity:    "high",
					Confidence:  0.82,
				})
			}
		}
	}

	return gaps
}

// Helper: Check if message contains any of the phrases (case-insensitive)
func containsPhrase(msg string, phrases ...string) bool {
	msgLower := strings.ToLower(msg)
	for _, phrase := range phrases {
		if strings.Contains(msgLower, strings.ToLower(phrase)) {
			return true
		}
	}
	return false
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
