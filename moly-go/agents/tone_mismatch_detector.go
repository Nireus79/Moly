package agents

import (
	"fmt"
	"strings"
)

// ToneMismatchDetector detects when user tone differs from connection tone
type ToneMismatchDetector struct{}

// ToneMismatch represents a detected tone difference
type ToneMismatch struct {
	UserTone       string  `json:"userTone"`
	ConnectionTone string  `json:"connectionTone"`
	Significance   string  `json:"significance"`   // "minor", "moderate", "significant"
	Type           string  `json:"type"`           // "enthusiasm", "warmth", "formality", "engagement"
	Insight        string  `json:"insight"`        // Human-readable explanation
	Recommendation string  `json:"recommendation"` // Optional: how to adjust
	Confidence     float64 `json:"confidence"`     // 0-1 confidence in detection
}

// NewToneMismatchDetector creates a new detector
func NewToneMismatchDetector() *ToneMismatchDetector {
	return &ToneMismatchDetector{}
}

// Detect analyzes user and connection tones for mismatches
func (t *ToneMismatchDetector) Detect(userTone, connectionTone string) *ToneMismatch {
	if userTone == "" || connectionTone == "" {
		return nil
	}

	userTone = strings.ToLower(strings.TrimSpace(userTone))
	connectionTone = strings.ToLower(strings.TrimSpace(connectionTone))

	if userTone == connectionTone {
		return nil // No mismatch
	}

	// Classify the mismatch
	mismatchType := classifyMismatchType(userTone, connectionTone)
	significance := assessSignificance(userTone, connectionTone)

	if significance == "none" {
		return nil
	}

	insight := generateInsight(userTone, connectionTone, mismatchType)
	recommendation := generateRecommendation(userTone, connectionTone, mismatchType)

	return &ToneMismatch{
		UserTone:       userTone,
		ConnectionTone: connectionTone,
		Significance:   significance,
		Type:           mismatchType,
		Insight:        insight,
		Recommendation: recommendation,
		Confidence:     0.75, // Default confidence
	}
}

// classifyMismatchType determines what type of mismatch it is
func classifyMismatchType(userTone, connectionTone string) string {
	// Enthusiasm mismatch
	if (isEnthusiastic(userTone) && !isEnthusiastic(connectionTone)) ||
		(!isEnthusiastic(userTone) && isEnthusiastic(connectionTone)) {
		return "enthusiasm"
	}

	// Warmth mismatch
	if (isWarm(userTone) && !isWarm(connectionTone)) ||
		(!isWarm(userTone) && isWarm(connectionTone)) {
		return "warmth"
	}

	// Formality mismatch
	if (isFormal(userTone) && !isFormal(connectionTone)) ||
		(!isFormal(userTone) && isFormal(connectionTone)) {
		return "formality"
	}

	// Engagement mismatch
	if (isEngaged(userTone) && !isEngaged(connectionTone)) ||
		(!isEngaged(userTone) && isEngaged(connectionTone)) {
		return "engagement"
	}

	// Default
	return "general"
}

// assessSignificance determines how significant the mismatch is
func assessSignificance(userTone, connectionTone string) string {
	// Opposite emotions = significant
	if areOpposite(userTone, connectionTone) {
		return "significant"
	}

	// Related but different = moderate
	if areRelated(userTone, connectionTone) {
		return "moderate"
	}

	// Very different
	return "minor"
}

// areOpposite checks if tones are opposite
func areOpposite(t1, t2 string) bool {
	opposites := map[string][]string{
		"enthusiastic":    {"neutral", "reserved", "dismissive"},
		"excited":         {"neutral", "reserved", "dismissive", "hesitant"},
		"energetic":       {"tired", "lethargic", "neutral"},
		"warm":            {"formal", "professional", "cold"},
		"friendly":        {"formal", "professional", "dismissive"},
		"affectionate":    {"formal", "professional", "distant"},
		"engaged":         {"disengaged", "dismissive", "indifferent"},
		"interested":      {"uninterested", "bored", "dismissive"},
		"casual":          {"formal", "professional", "serious"},
		"relaxed":         {"tense", "stressed", "urgent"},
		"supportive":      {"critical", "dismissive", "harsh"},
		"optimistic":      {"pessimistic", "gloomy", "disappointed"},
		"hopeful":         {"hopeless", "despondent"},
	}

	if oppositeList, exists := opposites[t1]; exists {
		for _, opp := range oppositeList {
			if opp == t2 {
				return true
			}
		}
	}

	// Reverse check
	if oppositeList, exists := opposites[t2]; exists {
		for _, opp := range oppositeList {
			if opp == t1 {
				return true
			}
		}
	}

	return false
}

// areRelated checks if tones are related but not opposite
func areRelated(t1, t2 string) bool {
	related := map[string][]string{
		"enthusiastic": {"interested", "engaged"},
		"excited":      {"interested", "engaged", "optimistic"},
		"warm":         {"friendly", "affectionate", "supportive"},
		"formal":       {"professional", "businesslike"},
		"neutral":      {"matter-of-fact", "objective"},
		"reserved":     {"cautious", "guarded", "hesitant"},
		"casual":       {"relaxed", "informal"},
		"tense":        {"stressed", "frustrated", "upset"},
	}

	if relatedList, exists := related[t1]; exists {
		for _, rel := range relatedList {
			if rel == t2 {
				return true
			}
		}
	}

	// Reverse check
	if relatedList, exists := related[t2]; exists {
		for _, rel := range relatedList {
			if rel == t1 {
				return true
			}
		}
	}

	return false
}

// Helper functions to classify tones

func isEnthusiastic(tone string) bool {
	enthusiasmTones := []string{"enthusiastic", "excited", "energetic", "passionate", "eager"}
	return toneInList(tone, enthusiasmTones)
}

func isWarm(tone string) bool {
	warmTones := []string{"warm", "friendly", "affectionate", "genuine", "kind", "caring"}
	return toneInList(tone, warmTones)
}

func isFormal(tone string) bool {
	formalTones := []string{"formal", "professional", "businesslike", "official"}
	return toneInList(tone, formalTones)
}

func isEngaged(tone string) bool {
	engagedTones := []string{"engaged", "interested", "focused", "attentive", "invested"}
	return toneInList(tone, engagedTones)
}

func toneInList(item string, list []string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

// generateInsight creates a human-readable explanation of the mismatch
func generateInsight(userTone, connectionTone, mismatchType string) string {
	switch mismatchType {
	case "enthusiasm":
		if isEnthusiastic(userTone) {
			return fmt.Sprintf("You're %s, but they're being more %s. Don't be surprised if they're slower to commit.", userTone, connectionTone)
		}
		return fmt.Sprintf("They're %s, but you're being more %s. They might be more excited about this than you seem.", connectionTone, userTone)

	case "warmth":
		if isWarm(userTone) {
			return fmt.Sprintf("You're bringing warmth (%s), but they're being more %s. Keep the warmth but match their reserve.", userTone, connectionTone)
		}
		return fmt.Sprintf("They're %s, but you're being more %s. Let them set the temperature—don't force warmth if they're not ready.", connectionTone, userTone)

	case "formality":
		if isFormal(connectionTone) {
			return fmt.Sprintf("They're being %s, but you're being more %s. Respect their tone and match their formality.", connectionTone, userTone)
		}
		return fmt.Sprintf("You're being %s, but they're more %s. Consider being more casual and less formal.", userTone, connectionTone)

	case "engagement":
		if isEngaged(userTone) {
			return fmt.Sprintf("You're %s, but they seem %s. They might not be as invested in this right now.", userTone, connectionTone)
		}
		return fmt.Sprintf("They're %s, but you're being %s. They might have more stake in this than you're showing.", connectionTone, userTone)

	default:
		return fmt.Sprintf("You're being %s, they're being %s. Notice the difference in how you're approaching this.", userTone, connectionTone)
	}
}

// generateRecommendation suggests how to adjust
func generateRecommendation(userTone, connectionTone, mismatchType string) string {
	switch mismatchType {
	case "enthusiasm":
		if isEnthusiastic(userTone) {
			return "Keep your energy but be prepared for a more measured response. Don't over-invest expectations."
		}
		return "They're excited—consider matching their energy or at least showing interest."

	case "warmth":
		if isWarm(userTone) {
			return "Your warmth is good, but don't push it if they're being formal. Let them warm up naturally."
		}
		return "They're being formal. Respect that and keep responses professional. Warmth can come later."

	case "formality":
		if isFormal(connectionTone) {
			return "Keep it professional and clear. Avoid casual language or jokes."
		}
		return "Loosen up a bit. They're being casual, so overly formal responses might feel cold."

	case "engagement":
		if isEngaged(userTone) {
			return "Check in with them. They might be distracted or not as focused right now."
		}
		return "Show more interest. They clearly care about this more than you're showing."

	default:
		return "Consider adjusting your approach to match their tone."
	}
}

// FormatForResponse returns a formatted string for including in response text
func (t *ToneMismatch) FormatForResponse() string {
	if t.Significance != "significant" {
		return ""
	}

	return fmt.Sprintf("\n\n**Tone note**: %s", t.Insight)
}
