package tools

import (
	"log"
	"strings"
)

// ContextExtractorHelper identifies the context in which user is speaking (work, home, social, general)
type ContextExtractorHelper struct{}

// NewContextExtractorHelper creates a new helper
func NewContextExtractorHelper() *ContextExtractorHelper {
	return &ContextExtractorHelper{}
}

// ExtractContextFromMessage analyzes message to identify context (work, home, social, general)
// Returns the primary context detected in the message
// Context is important because user behavior varies by context:
// - "formal at work" is NOT a conflict with "playful at home"
// - They're the same user in different contexts
func (h *ContextExtractorHelper) ExtractContextFromMessage(message string) string {
	if message == "" {
		return "general"
	}

	lowerMsg := strings.ToLower(message)

	// Work/Professional context indicators
	workIndicators := []string{
		"work", "job", "office", "boss", "colleague", "meeting", "project",
		"deadline", "professional", "formal", "business", "presentation",
		"client", "report", "manager", "team", "workplace", "corporate",
	}

	// Home/Personal context indicators
	homeIndicators := []string{
		"home", "family", "parent", "sibling", "house", "casual", "relax",
		"weekend", "personal", "intimate", "domestic", "kitchen", "bedroom",
		"comfortable", "private",
	}

	// Social/Friend context indicators
	socialIndicators := []string{
		"friend", "party", "hangout", "social", "group", "together", "casual",
		"fun", "laugh", "playful", "date", "dating", "relationship", "partner",
		"romantic", "couple",
	}

	workScore := 0
	homeScore := 0
	socialScore := 0

	// Count indicators in message
	for _, indicator := range workIndicators {
		if strings.Contains(lowerMsg, indicator) {
			workScore++
			log.Printf("[ContextExtractor] Found work indicator: '%s'", indicator)
		}
	}

	for _, indicator := range homeIndicators {
		if strings.Contains(lowerMsg, indicator) {
			homeScore++
			log.Printf("[ContextExtractor] Found home indicator: '%s'", indicator)
		}
	}

	for _, indicator := range socialIndicators {
		if strings.Contains(lowerMsg, indicator) {
			socialScore++
			log.Printf("[ContextExtractor] Found social indicator: '%s'", indicator)
		}
	}

	// Determine primary context by score
	maxScore := workScore
	primaryContext := "work"

	if homeScore > maxScore {
		maxScore = homeScore
		primaryContext = "home"
	}

	if socialScore > maxScore {
		maxScore = socialScore
		primaryContext = "social"
	}

	// If no strong context indicators found, assume general
	if maxScore == 0 {
		primaryContext = "general"
		log.Printf("[ContextExtractor] No context indicators found, defaulting to 'general'")
	} else {
		log.Printf("[ContextExtractor] Detected context: '%s' (work=%d, home=%d, social=%d)", primaryContext, workScore, homeScore, socialScore)
	}

	return primaryContext
}

// GetContextDescription provides human-readable description of context
func (h *ContextExtractorHelper) GetContextDescription(context string) string {
	descriptions := map[string]string{
		"work":    "at work",
		"home":    "at home",
		"social":  "in social settings",
		"general": "generally",
	}

	if desc, exists := descriptions[context]; exists {
		return desc
	}

	return "in " + context + " context"
}

// AreContextsCompatible checks if two contexts can coexist (e.g., "work" and "home" are compatible, "work" and "work" are not)
func (h *ContextExtractorHelper) AreContextsCompatible(context1, context2 string) bool {
	// Same context = not compatible (it's a conflict)
	if context1 == context2 {
		return false
	}

	// If either is "general", they conflict with specific contexts (general means always)
	if context1 == "general" || context2 == "general" {
		return false
	}

	// Different specific contexts (work vs home) are compatible - can save both
	return true
}
