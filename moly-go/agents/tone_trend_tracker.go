package agents

import (
	"fmt"
	"strings"
)

// ToneTrendTracker tracks tone changes for users and contacts across conversations
type ToneTrendTracker struct{}

// ToneObservation represents a single tone observation
type ToneObservation struct {
	Who       string  `json:"who"`       // "user" or contact name
	Tone      string  `json:"tone"`      // "excited", "warm", "dismissive", etc.
	Timestamp int64   `json:"timestamp"`
	Confidence float64 `json:"confidence"`
}

// ToneTrend represents the tone trajectory for a person
type ToneTrend struct {
	Who              string                `json:"who"`
	ToneHistory      []ToneObservation     `json:"toneHistory"`
	CurrentTone      string                `json:"currentTone"`
	PreviousTone     string                `json:"previousTone"`
	ToneShiftDetected bool                 `json:"toneShiftDetected"`
	ShiftType        string                `json:"shiftType"`        // "warming", "cooling", "becoming_formal", "becoming_casual"
	Significance     string                `json:"significance"`     // "minor", "moderate", "significant"
	Insight          string                `json:"insight"`          // User-facing explanation
	Recommendation   string                `json:"recommendation"`   // Optional advice
}

// NewToneTrendTracker creates a new tracker
func NewToneTrendTracker() *ToneTrendTracker {
	return &ToneTrendTracker{}
}

// ObserveTone records a new tone observation
func (t *ToneTrendTracker) ObserveTone(who, tone string, timestamp int64, confidence float64) *ToneObservation {
	return &ToneObservation{
		Who:        who,
		Tone:       strings.ToLower(strings.TrimSpace(tone)),
		Timestamp:  timestamp,
		Confidence: confidence,
	}
}

// AnalyzeTrend examines tone history for a person and detects shifts
func (t *ToneTrendTracker) AnalyzeTrend(who string, observations []ToneObservation) *ToneTrend {
	if len(observations) == 0 {
		return nil
	}

	trend := &ToneTrend{
		Who:         who,
		ToneHistory: observations,
		CurrentTone: observations[len(observations)-1].Tone,
	}

	// Get previous tone if history exists
	if len(observations) > 1 {
		trend.PreviousTone = observations[len(observations)-2].Tone
	}

	// Detect if there's a shift
	if trend.PreviousTone != "" && trend.PreviousTone != trend.CurrentTone {
		trend.ToneShiftDetected = true
		trend.ShiftType = classifyShift(trend.PreviousTone, trend.CurrentTone)
		trend.Significance = assessShiftSignificance(trend.PreviousTone, trend.CurrentTone)

		if trend.Significance != "none" {
			trend.Insight = generateTrendInsight(who, trend.PreviousTone, trend.CurrentTone, trend.ShiftType)
			trend.Recommendation = generateTrendRecommendation(who, trend.PreviousTone, trend.CurrentTone, trend.ShiftType)
		}
	}

	return trend
}

// classifyShift determines what kind of tone change occurred
func classifyShift(fromTone, toTone string) string {
	// Warming shifts
	if isNegative(fromTone) && !isNegative(toTone) {
		return "warming"
	}
	if isFormal(fromTone) && !isFormal(toTone) {
		return "becoming_casual"
	}
	if isReserved(fromTone) && !isReserved(toTone) {
		return "becoming_more_open"
	}

	// Cooling shifts
	if !isNegative(fromTone) && isNegative(toTone) {
		return "cooling"
	}
	if !isFormal(fromTone) && isFormal(toTone) {
		return "becoming_formal"
	}
	if !isReserved(fromTone) && isReserved(toTone) {
		return "becoming_distant"
	}

	return "tone_shift"
}

// assessShiftSignificance determines how significant the shift is
func assessShiftSignificance(fromTone, toTone string) string {
	// Opposite shifts = significant
	if areOppositeTones(fromTone, toTone) {
		return "significant"
	}

	// Related but different = moderate
	if areRelatedTones(fromTone, toTone) {
		return "moderate"
	}

	return "minor"
}

// Tone classification helpers

func isNegative(tone string) bool {
	negativeTones := []string{"dismissive", "cold", "distant", "frustrated", "angry", "upset", "harsh"}
	return toneInSlice(tone, negativeTones)
}

func isPositive(tone string) bool {
	positiveTones := []string{"warm", "friendly", "enthusiastic", "excited", "happy", "supportive", "encouraging"}
	return toneInSlice(tone, positiveTones)
}

func isFormal(tone string) bool {
	formalTones := []string{"formal", "professional", "businesslike", "official", "serious"}
	return toneInSlice(tone, formalTones)
}

func isReserved(tone string) bool {
	reservedTones := []string{"reserved", "cautious", "guarded", "hesitant", "neutral", "matter-of-fact"}
	return toneInSlice(tone, reservedTones)
}

func isEngaged(tone string) bool {
	engagedTones := []string{"engaged", "interested", "invested", "focused", "attentive"}
	return toneInSlice(tone, engagedTones)
}

func areOppositeTones(t1, t2 string) bool {
	opposites := map[string][]string{
		"warm":         {"cold", "distant", "dismissive", "formal"},
		"friendly":     {"hostile", "dismissive", "formal"},
		"enthusiastic": {"dismissive", "indifferent", "reserved"},
		"excited":      {"disappointed", "frustrated", "apathetic"},
		"happy":        {"upset", "frustrated", "angry", "depressed"},
		"supportive":   {"critical", "dismissive", "hostile"},
		"engaged":      {"disengaged", "dismissive", "apathetic"},
		"casual":       {"formal", "serious", "professional"},
		"open":         {"guarded", "reserved", "cautious"},
	}

	if oppositeList, exists := opposites[t1]; exists {
		for _, opp := range oppositeList {
			if opp == t2 {
				return true
			}
		}
	}

	if oppositeList, exists := opposites[t2]; exists {
		for _, opp := range oppositeList {
			if opp == t1 {
				return true
			}
		}
	}

	return false
}

func areRelatedTones(t1, t2 string) bool {
	related := map[string][]string{
		"enthusiastic": {"excited", "energetic", "engaged", "interested"},
		"warm":         {"friendly", "affectionate", "genuine", "supportive"},
		"formal":       {"professional", "businesslike", "serious"},
		"casual":       {"relaxed", "informal", "laid-back"},
		"reserved":     {"cautious", "guarded", "hesitant", "neutral"},
		"frustrated":   {"stressed", "upset", "annoyed"},
		"happy":        {"content", "satisfied", "pleased"},
	}

	if relatedList, exists := related[t1]; exists {
		for _, rel := range relatedList {
			if rel == t2 {
				return true
			}
		}
	}

	if relatedList, exists := related[t2]; exists {
		for _, rel := range relatedList {
			if rel == t1 {
				return true
			}
		}
	}

	return false
}

func toneInSlice(tone string, list []string) bool {
	for _, v := range list {
		if v == tone {
			return true
		}
	}
	return false
}

// generateTrendInsight creates a human-readable explanation of the tone shift
func generateTrendInsight(who string, fromTone, toTone, shiftType string) string {
	if who == "user" {
		switch shiftType {
		case "warming":
			return fmt.Sprintf("Your tone has shifted from %s to %s. You seem to be feeling better.", fromTone, toTone)
		case "cooling":
			return fmt.Sprintf("Your tone has shifted from %s to %s. Something seems to have changed.", fromTone, toTone)
		case "becoming_formal":
			return fmt.Sprintf("Your tone is becoming more formal (was %s, now %s).", fromTone, toTone)
		case "becoming_casual":
			return fmt.Sprintf("Your tone is becoming more casual (was %s, now %s).", fromTone, toTone)
		case "becoming_distant":
			return fmt.Sprintf("Your tone is becoming more distant (was %s, now %s).", fromTone, toTone)
		case "becoming_more_open":
			return fmt.Sprintf("Your tone is becoming more open (was %s, now %s).", fromTone, toTone)
		default:
			return fmt.Sprintf("Your tone shifted from %s to %s.", fromTone, toTone)
		}
	} else {
		// Contact tone shift
		switch shiftType {
		case "warming":
			return fmt.Sprintf("%s's tone has warmed up (was %s, now %s).", who, fromTone, toTone)
		case "cooling":
			return fmt.Sprintf("%s's tone has cooled (was %s, now %s). They might be upset or distracted.", who, fromTone, toTone)
		case "becoming_formal":
			return fmt.Sprintf("%s is becoming more formal (was %s, now %s). They might be in work mode or less comfortable.", who, fromTone, toTone)
		case "becoming_casual":
			return fmt.Sprintf("%s is becoming more casual (was %s, now %s). They're loosening up.", who, fromTone, toTone)
		case "becoming_distant":
			return fmt.Sprintf("%s is becoming more distant (was %s, now %s). Something might be bothering them.", who, fromTone, toTone)
		case "becoming_more_open":
			return fmt.Sprintf("%s is becoming more open (was %s, now %s). They're relaxing.", who, fromTone, toTone)
		default:
			return fmt.Sprintf("%s's tone shifted from %s to %s.", who, fromTone, toTone)
		}
	}
}

// generateTrendRecommendation suggests how to respond to the tone shift
func generateTrendRecommendation(who string, fromTone, toTone, shiftType string) string {
	if who == "user" {
		switch shiftType {
		case "warming":
			return "Your mood is improving. Keep building on this positive momentum."
		case "cooling":
			return "Your mood seems to be shifting. Check in with yourself about what's changed."
		case "becoming_formal":
			return "You're being more formal now. Make sure you're still being authentic."
		case "becoming_casual":
			return "You're being more casual now. The other person might match your energy."
		default:
			return "Notice and reflect on what's causing this shift in your tone."
		}
	} else {
		switch shiftType {
		case "warming":
			return "They're warming up to you. You might be making progress."
		case "cooling":
			return "They might be upset or losing interest. Consider checking in with them."
		case "becoming_formal":
			return "They're being more professional now. Match their formality level."
		case "becoming_casual":
			return "They're loosening up. You can probably be more relaxed too."
		case "becoming_distant":
			return "They're pulling back. Give them space or gently ask if something's wrong."
		default:
			return "Pay attention to how they're responding and adjust your approach."
		}
	}
}

// FormatForResponse returns a formatted string for including in response text
func (t *ToneTrend) FormatForResponse() string {
	if !t.ToneShiftDetected || t.Significance == "none" {
		return ""
	}

	if t.Significance == "significant" {
		return fmt.Sprintf("\n\n**Tone shift detected**: %s", t.Insight)
	}

	return ""
}
