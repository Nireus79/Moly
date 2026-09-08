package tools

import (
	"fmt"
	"math"
	"strings"

	"moly/models"
)

// BehaviorAnalyzer - Analyzes user behavior patterns
type BehaviorAnalyzer struct{}

// NewBehaviorAnalyzer - Create new behavior analyzer
func NewBehaviorAnalyzer() *BehaviorAnalyzer {
	return &BehaviorAnalyzer{}
}

// AnalyzeChoicePatterns - Analyze patterns in suggestion choices
func (ba *BehaviorAnalyzer) AnalyzeChoicePatterns(interactions []interface{}) *models.UserBehavioralProfile {
	profile := &models.UserBehavioralProfile{
		CommunicationProfile: make(map[string]interface{}),
		CommunicationGoals:   make(map[string]int),
		SuggestionChoices:    make(map[string]interface{}),
		SuccessMetrics:       make(map[string]interface{}),
		EmergingPersonality:  []string{},
		GrowthTrajectory:     make(map[string]interface{}),
		Confidence:           0.5,
	}

	if len(interactions) == 0 {
		return profile
	}

	totalChoices := len(interactions)

	if totalChoices > 0 {
		modificationRate := 0.3  // Simplified - would analyze in full version
		profile.CommunicationGoals["total_choices"] = totalChoices
		profile.CommunicationGoals["modifications"] = int(modificationRate * float64(totalChoices))
		profile.SuccessMetrics["modification_rate"] = fmt.Sprintf("%.1f%%", modificationRate*100)
	}

	// Update confidence based on data
	if totalChoices >= 10 {
		profile.Confidence = 0.8
	} else if totalChoices >= 5 {
		profile.Confidence = 0.7
	} else if totalChoices > 0 {
		profile.Confidence = 0.6
	}

	return profile
}

// AnalyzeInteractionFrequency - Analyze how often user interacts
func (ba *BehaviorAnalyzer) AnalyzeInteractionFrequency(interactions []interface{}) map[string]interface{} {
	stats := make(map[string]interface{})

	if len(interactions) == 0 {
		stats["frequency"] = "none"
		stats["count"] = 0
		return stats
	}

	stats["total_interactions"] = len(interactions)
	stats["by_type"] = make(map[string]int)

	// Simplified for now - would analyze interactions in full version
	stats["sentiment_distribution"] = make(map[string]int)

	return stats
}

// AnalyzeCommunicationStyle - Infer communication style from patterns
func (ba *BehaviorAnalyzer) AnalyzeCommunicationStyle(interactions []interface{}, aboutMe *models.AboutMe) string {
	if aboutMe != nil && aboutMe.CommunicationStyle != "" {
		return aboutMe.CommunicationStyle
	}

	// Analyze from interactions if no aboutMe
	if len(interactions) == 0 {
		return "developing"
	}

	// Simplified version - just return developing for now
	// Full version would parse interaction objects

	return "developing"
}

// AnalyzeGrowth - Analyze user growth over time
func (ba *BehaviorAnalyzer) AnalyzeGrowth(interactions []interface{}) map[string]interface{} {
	growth := make(map[string]interface{})

	if len(interactions) < 2 {
		growth["status"] = "insufficient_data"
		return growth
	}

	// Simplified growth analysis
	if len(interactions) >= 2 {
		growth["trend"] = "stable"
		growth["early_modifications"] = len(interactions) / 2
		growth["recent_modifications"] = len(interactions) / 2
	}

	return growth
}

// AnalyzeTonePreference - Analyze preferred communication tones
func (ba *BehaviorAnalyzer) AnalyzeTonePreference(interactions []interface{}, aboutMe *models.AboutMe) map[string]float64 {
	preferences := make(map[string]float64)

	if aboutMe != nil && aboutMe.PreferredTone != "" {
		preferences[aboutMe.PreferredTone] = 1.0
		return preferences
	}

	// Analyze from interactions
	tones := []string{"formal", "casual", "warm", "direct", "playful", "serious"}
	toneScores := make(map[string]float64)

	for _, tone := range tones {
		toneScores[tone] = 0.5 // Default neutral
	}

	// Normalize and return top scores
	maxScore := 0.0
	for _, score := range toneScores {
		if score > maxScore {
			maxScore = score
		}
	}

	if maxScore > 0 {
		for tone, score := range toneScores {
			preferences[tone] = math.Min(score/maxScore, 1.0)
		}
	}

	return preferences
}

// PredictNextIntention - Predict what user might want to do next
func (ba *BehaviorAnalyzer) PredictNextIntention(interactions []interface{}, lastIntention IntentionType) IntentionType {
	if len(interactions) < 2 {
		return lastIntention
	}

	// Simplified prediction - return last intention
	// Real implementation would analyze patterns in interaction sequence
	return lastIntention
}

// GeneratePersonalityInsights - Generate insights about user personality
func (ba *BehaviorAnalyzer) GeneratePersonalityInsights(profile *models.UserBehavioralProfile, aboutMe *models.AboutMe) []string {
	insights := []string{}

	if profile == nil {
		return insights
	}

	// Based on modification rate
	if modRate, ok := profile.SuccessMetrics["modification_rate"].(string); ok {
		if strings.Contains(modRate, "90") || strings.Contains(modRate, "95") {
			insights = append(insights, "Highly customizes suggestions - values authenticity")
		} else if strings.Contains(modRate, "5") || strings.Contains(modRate, "10") {
			insights = append(insights, "Prefers suggestions as-is - trusts recommendations")
		}
	}

	// Based on confidence
	if profile.Confidence > 0.8 {
		insights = append(insights, "Well-established patterns in communication")
	} else if profile.Confidence < 0.5 {
		insights = append(insights, "Still developing communication patterns")
	}

	// Based on about me
	if aboutMe != nil {
		if aboutMe.CommunicationStyle != "" {
			style := aboutMe.CommunicationStyle
			switch style {
			case "formal":
				insights = append(insights, "Prefers professional, structured communication")
			case "casual":
				insights = append(insights, "Comfortable with informal, relaxed style")
			case "warm":
				insights = append(insights, "Prioritizes emotional connection and warmth")
			case "direct":
				insights = append(insights, "Values clarity and straightforward communication")
			case "playful":
				insights = append(insights, "Enjoys humor and light-hearted interactions")
			}
		}

		if len(aboutMe.Values) > 0 {
			insights = append(insights, fmt.Sprintf("Values: %s", strings.Join(aboutMe.Values, ", ")))
		}
	}

	return insights
}

// CalculateContextCompleteness - Calculate how complete user context is
func (ba *BehaviorAnalyzer) CalculateContextCompleteness(aboutMe *models.AboutMe, contacts []models.Contact, interactions int) float64 {
	score := 0.0

	// AboutMe completeness (0.33)
	if aboutMe != nil {
		aboutMeScore := 0.0
		if aboutMe.CommunicationStyle != "" {
			aboutMeScore += 0.11
		}
		if len(aboutMe.Values) > 0 {
			aboutMeScore += 0.11
		}
		if aboutMe.PreferredTone != "" {
			aboutMeScore += 0.11
		}
		score += aboutMeScore
	}

	// Contact knowledge (0.33)
	if len(contacts) > 0 {
		contactScore := 0.0
		if len(contacts) >= 1 {
			contactScore += 0.11
		}
		if len(contacts) >= 3 {
			contactScore += 0.11
		}
		if len(contacts) >= 5 {
			contactScore += 0.11
		}
		score += contactScore
	}

	// Interaction history (0.34)
	if interactions > 0 {
		interactionScore := 0.0
		if interactions >= 1 {
			interactionScore += 0.11
		}
		if interactions >= 5 {
			interactionScore += 0.11
		}
		if interactions >= 10 {
			interactionScore += 0.12
		}
		score += interactionScore
	}

	return math.Min(score, 1.0)
}
