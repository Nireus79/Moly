package agents

import (
	"fmt"
	"log"
	"strings"

	"moly/database"
)

// ConflictDetection represents a detected conflict
type ConflictDetection struct {
	ID              int64       `json:"id"`
	HasConflict     bool        `json:"hasConflict"`
	ConflictType    string      `json:"conflictType"` // "attribute_contradiction", "name_change", "relationship_change"
	Subject         string      `json:"subject"`      // who the conflict is about
	Context         string      `json:"context"`      // work, social, general
	SavedValue      interface{} `json:"savedValue"`
	ExtractedValue  interface{} `json:"extractedValue"`
	Evidence        string      `json:"evidence"`   // quote from message
	Severity        string      `json:"severity"`   // "low", "medium", "high"
	Description     string      `json:"description"`
	Recommendation  string      `json:"recommendation"`
}

// ConflictDetector identifies conflicts between saved context and newly extracted facts
type ConflictDetector struct {
	attrRepo  *database.ContextAttributeRepository
	contactRepo *database.ContactRepository
}

// NewConflictDetector creates a new detector
func NewConflictDetector(
	attrRepo *database.ContextAttributeRepository,
	contactRepo *database.ContactRepository,
) *ConflictDetector {
	return &ConflictDetector{
		attrRepo: attrRepo,
		contactRepo: contactRepo,
	}
}

// DetectConflict checks if newly extracted attribute conflicts with saved context
// Only returns conflict if:
// - Same subject
// - Same context
// - Value contradicts what was saved
// Different contexts (work vs social) are NOT conflicts
// Different subjects (user vs contact) are NOT conflicts
func (d *ConflictDetector) DetectConflict(
	userID string,
	newAttribute *database.ContextAttribute,
) (*ConflictDetection, error) {
	log.Printf("[V2] ConflictDetector: checking for conflicts on %s.%s", newAttribute.AttributedTo, newAttribute.FactType)

	// Get all existing attributes for this subject and type
	existing, err := d.attrRepo.GetByType(userID, newAttribute.AttributedTo, newAttribute.FactType)
	if err != nil {
		log.Printf("[V2] ConflictDetector ERROR: %v", err)
		return nil, err
	}

	if len(existing) == 0 {
		log.Printf("[V2] ConflictDetector: no existing attributes found, no conflict")
		return &ConflictDetection{HasConflict: false}, nil
	}

	// Check for contradictions in same context only
	for _, existingAttr := range existing {
		// Only check for conflict if contexts match (or either is general/unspecified)
		contextMatches := existingAttr.Context == newAttribute.Context ||
			existingAttr.Context == "general" || existingAttr.Context == "" ||
			newAttribute.Context == "general" || newAttribute.Context == ""

		log.Printf("[V2] ConflictDetector:   checking existing: type=%s value=%s context=%s (new context=%s)", existingAttr.FactType, existingAttr.FactValue, existingAttr.Context, newAttribute.Context)
		log.Printf("[V2] ConflictDetector:   context match=%v (saved=%s vs new=%s)", contextMatches, existingAttr.Context, newAttribute.Context)

		if contextMatches {
			// Check if values contradict (not just different, but contradictory)
			isContra := d.isContradictory(existingAttr.FactValue, newAttribute.FactValue)
			log.Printf("[V2] ConflictDetector:   contradictory=%v (saved=%s vs new=%s)", isContra, existingAttr.FactValue, newAttribute.FactValue)

			if isContra {
				log.Printf("[V2] ConflictDetector: ✗ CONFLICT DETECTED - %s vs %s (context: %s severity will be assigned)", existingAttr.FactValue, newAttribute.FactValue, newAttribute.Context)

				severity := "medium"
				if newAttribute.FactType == "name" || newAttribute.FactType == "relationship" {
					severity = "high"
				}

				log.Printf("[V2] ConflictDetector:   severity=%s type=%s subject=%s", severity, newAttribute.FactType, newAttribute.AttributedTo)

				return &ConflictDetection{
					HasConflict:    true,
					ConflictType:   "attribute_contradiction",
					Subject:        newAttribute.AttributedTo,
					Context:        newAttribute.Context,
					SavedValue:     existingAttr.FactValue,
					ExtractedValue: newAttribute.FactValue,
					Evidence:       newAttribute.Evidence,
					Severity:       severity,
					Description:    fmt.Sprintf("System previously learned: '%s' is %s in %s context. Now extracted: '%s'. Different context may resolve this.", existingAttr.FactValue, newAttribute.FactType, existingAttr.Context, newAttribute.FactValue),
					Recommendation: "If this is a different context (work vs social), save both. If contradiction in same context, ask user which is accurate.",
				}, nil
			}
		} else {
			log.Printf("[V2] ConflictDetector:   context mismatch: saved=%s new=%s (OK - different contexts allowed)", existingAttr.Context, newAttribute.Context)
		}
	}

	// No contradictions found
	log.Printf("[V2] ConflictDetector: ✓ NO CONFLICT - checked %d existing attributes, no contradictions found", len(existing))
	return &ConflictDetection{HasConflict: false}, nil
}

// isContradictory determines if two values actually contradict
// Not all different values are contradictions
func (d *ConflictDetector) isContradictory(existing string, extracted string) bool {
	existing = strings.ToLower(strings.TrimSpace(existing))
	extracted = strings.ToLower(strings.TrimSpace(extracted))

	if existing == extracted {
		return false // Same value, no contradiction
	}

	// Define true contradictions
	contraryPairs := map[string]string{
		"formal":         "casual",
		"casual":         "formal",
		"direct":         "indirect",
		"indirect":       "direct",
		"detailed":       "brief",
		"brief":          "detailed",
		"professional":   "casual",
		"extrovert":      "introvert",
		"introvert":      "extrovert",
		"optimistic":     "pessimistic",
		"pessimistic":    "optimistic",
		"risk-taker":     "risk-averse",
		"risk-averse":    "risk-taker",
	}

	if opposite, exists := contraryPairs[existing]; exists && opposite == extracted {
		return true
	}

	// For other attributes, same type + different value might just be additional info
	// Not automatically a contradiction
	// E.g., "detail-oriented" and "thorough" are similar, not contradictory
	return false
}

// GetConflictsBySubject retrieves all conflicts for a subject
func (d *ConflictDetector) GetConflictsBySubject(userID string, subject string) ([]*ConflictDetection, error) {
	// This would pull from context_conflicts table if we persisted them
	// For now, returns empty (conflicts are detected in-flow)
	return []*ConflictDetection{}, nil
}

// ContextDependentAttributes checks if attribute should be context-specific
// Returns true if this type of attribute varies by context
func (d *ConflictDetector) ContextDependentAttributes(factType string) bool {
	contextDependentTypes := map[string]bool{
		"style":          true,  // communication style varies
		"formality":      true,  // formal at work, casual at home
		"tone":           true,  // professional in meetings, relaxed with friends
		"energy_level":   true,  // busy at work, relaxed on weekend
		"communication":  true,  // detailed reports at work, quick texts with family
		"trait":          false, // traits are consistent across contexts
		"value":          false, // values are mostly consistent
		"goal":           false, // goals don't typically vary by context
		"name":           false, // name doesn't change by context
		"relationship":   false, // relationship doesn't change by context
	}

	if isDependentDefaultValue, exists := contextDependentTypes[factType]; exists {
		return isDependentDefaultValue
	}

	return false // Default to non-dependent
}

// SuggestContextMerge provides guidance on how to handle context-dependent attributes
func (d *ConflictDetector) SuggestContextMerge(conflict *ConflictDetection) string {
	if !d.ContextDependentAttributes(conflict.ConflictType) {
		return "These values contradict in a context-independent attribute. One must be incorrect."
	}

	return fmt.Sprintf("This attribute may vary by context. You described '%s' in %s context previously. Is '%s' from a different context?",
		conflict.SavedValue, conflict.Context, conflict.ExtractedValue)
}
