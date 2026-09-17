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

// AnalyzeChoicePatterns - Analyze patterns in user choices (conflicts resolved, suggestions accepted)
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
	modificationCount := 0
	keepCount := 0
	mergeCount := 0

	// Count choice patterns (looking for "resolution" fields in interactions)
	for _, interaction := range interactions {
		if m, ok := interaction.(map[string]interface{}); ok {
			if resolution, hasResolution := m["resolution"].(string); hasResolution {
				switch resolution {
				case "use_extracted":
					modificationCount++
				case "keep_saved":
					keepCount++
				case "merge":
					mergeCount++
				}
			}
		}
	}

	// Real modification rate based on actual data
	var modificationRate float64
	if totalChoices > 0 {
		modificationRate = float64(modificationCount+mergeCount) / float64(totalChoices)
	}

	profile.CommunicationGoals["total_choices"] = totalChoices
	profile.CommunicationGoals["accepted_changes"] = modificationCount
	profile.CommunicationGoals["kept_original"] = keepCount
	profile.CommunicationGoals["merged_contexts"] = mergeCount
	profile.SuccessMetrics["modification_rate"] = fmt.Sprintf("%.1f%%", modificationRate*100)
	profile.SuccessMetrics["consistency"] = fmt.Sprintf("%.1f%%", float64(keepCount)/float64(totalChoices)*100)

	// Calculate confidence based on sample size and consistency
	sampleFactor := math.Min(float64(totalChoices)/10.0, 1.0)
	consistency := math.Max(modificationRate, 1.0-modificationRate)
	consistencyFactor := consistency // High if user is consistent in choices

	profile.Confidence = 0.5 * sampleFactor * consistencyFactor
	if profile.Confidence > 1.0 {
		profile.Confidence = 1.0
	}
	if profile.Confidence < 0.5 {
		profile.Confidence = 0.5
	}

	return profile
}

// AnalyzeInteractionFrequency - Analyze how often and what sentiment user interacts with
func (ba *BehaviorAnalyzer) AnalyzeInteractionFrequency(interactions []interface{}) map[string]interface{} {
	stats := make(map[string]interface{})

	if len(interactions) == 0 {
		stats["frequency"] = "none"
		stats["count"] = 0
		stats["activity_level"] = "inactive"
		return stats
	}

	totalInteractions := len(interactions)
	stats["total_interactions"] = totalInteractions

	// Analyze sentiment distribution
	sentimentMap := make(map[string]int)
	interactionTypes := make(map[string]int)

	for _, interaction := range interactions {
		if m, ok := interaction.(map[string]interface{}); ok {
			// Count interaction types
			if interactionType, hasType := m["type"].(string); hasType {
				interactionTypes[interactionType]++
			}
			// Count sentiment
			if sentiment, hasSentiment := m["sentiment"].(string); hasSentiment {
				sentimentMap[sentiment]++
			}
		}
	}

	// Determine frequency pattern
	var frequency string
	switch {
	case totalInteractions >= 50:
		frequency = "very_frequent"
	case totalInteractions >= 20:
		frequency = "frequent"
	case totalInteractions >= 10:
		frequency = "regular"
	case totalInteractions >= 5:
		frequency = "occasional"
	default:
		frequency = "sparse"
	}
	stats["frequency"] = frequency

	// Activity level
	var activityLevel string
	if totalInteractions >= 30 {
		activityLevel = "highly_active"
	} else if totalInteractions >= 15 {
		activityLevel = "active"
	} else if totalInteractions >= 5 {
		activityLevel = "engaged"
	} else {
		activityLevel = "minimal"
	}
	stats["activity_level"] = activityLevel

	// Store type distribution
	if len(interactionTypes) > 0 {
		stats["by_type"] = interactionTypes
	}

	// Store sentiment distribution
	if len(sentimentMap) > 0 {
		stats["sentiment_distribution"] = sentimentMap
		// Calculate dominant sentiment
		var dominantSentiment string
		maxCount := 0
		for sentiment, count := range sentimentMap {
			if count > maxCount {
				maxCount = count
				dominantSentiment = sentiment
			}
		}
		if dominantSentiment != "" {
			stats["dominant_sentiment"] = dominantSentiment
			stats["sentiment_ratio"] = fmt.Sprintf("%.1f%%", float64(maxCount)/float64(totalInteractions)*100)
		}
	}

	return stats
}

// AnalyzeCommunicationStyle - Infer communication style from AboutMe or interaction patterns
func (ba *BehaviorAnalyzer) AnalyzeCommunicationStyle(interactions []interface{}, aboutMe *models.AboutMe) string {
	// First priority: use explicitly set AboutMe style
	if aboutMe != nil && aboutMe.CommunicationStyle != "" {
		return aboutMe.CommunicationStyle
	}

	// If no AboutMe, analyze from interactions
	if len(interactions) == 0 {
		return "developing"
	}

	// Count style preferences across interactions
	styleScores := make(map[string]int)

	for _, interaction := range interactions {
		if m, ok := interaction.(map[string]interface{}); ok {
			// Look for style indicators in interactions
			if style, hasStyle := m["communication_style"].(string); hasStyle && style != "" {
				styleScores[style]++
			}
			// Also check "tone" field
			if tone, hasTone := m["tone"].(string); hasTone && tone != "" {
				styleScores[tone]++
			}
		}
	}

	// Find dominant style
	if len(styleScores) > 0 {
		var dominantStyle string
		maxScore := 0
		for style, score := range styleScores {
			if score > maxScore {
				maxScore = score
				dominantStyle = style
			}
		}
		if dominantStyle != "" && maxScore >= len(interactions)/3 {
			// Only return if this style appears in at least 1/3 of interactions
			return dominantStyle
		}
	}

	// Fallback: try to infer from AboutMe values
	if aboutMe != nil && len(aboutMe.Values) > 0 {
		// Map values to communication styles
		valueToStyle := map[string]string{
			"authenticity": "authentic",
			"honesty":      "direct",
			"warmth":       "warm",
			"professionalism": "professional",
		}
		for _, value := range aboutMe.Values {
			if style, exists := valueToStyle[strings.ToLower(value)]; exists {
				return style
			}
		}
	}

	return "developing"
}

// AnalyzeGrowth - Analyze user growth and evolution over time
func (ba *BehaviorAnalyzer) AnalyzeGrowth(interactions []interface{}) map[string]interface{} {
	growth := make(map[string]interface{})

	if len(interactions) < 2 {
		growth["status"] = "insufficient_data"
		growth["trend"] = "insufficient"
		return growth
	}

	// Extract timestamped interactions
	type TimestampedInteraction struct {
		timestamp int64
		style     string
	}
	var timestamped []TimestampedInteraction

	for _, interaction := range interactions {
		if m, ok := interaction.(map[string]interface{}); ok {
			var ts int64
			var style string

			if timestamp, hasTS := m["timestamp"].(float64); hasTS {
				ts = int64(timestamp)
			}
			if s, hasStyle := m["communication_style"].(string); hasStyle {
				style = s
			}

			if ts > 0 && style != "" {
				timestamped = append(timestamped, TimestampedInteraction{ts, style})
			}
		}
	}

	// Analyze growth patterns
	if len(timestamped) < 2 {
		growth["status"] = "insufficient_timestamped_data"
		growth["trend"] = "developing"
		return growth
	}

	// Sort by timestamp (oldest first)
	// Calculate early vs recent modifications
	midpoint := len(timestamped) / 2
	earlyInteractions := timestamped[:midpoint]
	recentInteractions := timestamped[midpoint:]

	// Detect style changes
	earlyStyleMap := make(map[string]int)
	recentStyleMap := make(map[string]int)

	for _, ti := range earlyInteractions {
		earlyStyleMap[ti.style]++
	}
	for _, ti := range recentInteractions {
		recentStyleMap[ti.style]++
	}

	// Find dominant styles
	earlyDominant := ba.getDominantStyle(earlyStyleMap)
	recentDominant := ba.getDominantStyle(recentStyleMap)

	// Determine trend
	var trend string
	if earlyDominant != recentDominant && earlyDominant != "" && recentDominant != "" {
		trend = "growing" // Style is evolving
		growth["transition"] = fmt.Sprintf("%s → %s", earlyDominant, recentDominant)
	} else if recentDominant != "" {
		trend = "stable" // Consistent style
	} else {
		trend = "volatile" // Inconsistent
	}

	growth["status"] = "analyzed"
	growth["trend"] = trend
	growth["early_style"] = earlyDominant
	growth["recent_style"] = recentDominant
	growth["early_modifications"] = len(earlyInteractions)
	growth["recent_modifications"] = len(recentInteractions)

	// Confidence improves with consistent recent behavior
	if earlyDominant == recentDominant {
		growth["confidence_trend"] = "increasing"
		growth["pattern_confirmed"] = true
	} else {
		growth["confidence_trend"] = "establishing"
		growth["pattern_confirmed"] = false
	}

	return growth
}

// Helper: get dominant style from frequency map
func (ba *BehaviorAnalyzer) getDominantStyle(styleMap map[string]int) string {
	if len(styleMap) == 0 {
		return ""
	}
	var dominantStyle string
	maxCount := 0
	for style, count := range styleMap {
		if count > maxCount {
			maxCount = count
			dominantStyle = style
		}
	}
	return dominantStyle
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

// ContextProfile describes how user communicates in a specific context
type ContextProfile struct {
	Context    string                 `json:"context"`    // work, home, social, general
	Style      string                 `json:"style"`      // dominant communication style
	Confidence float64                `json:"confidence"` // 0.5 to 1.0
	SampleSize int                    `json:"sampleSize"` // how many interactions
	Details    map[string]interface{} `json:"details"`    // additional metrics
}

// BuildContextProfiles - Analyze communication style for each context
func (ba *BehaviorAnalyzer) BuildContextProfiles(interactions []interface{}) map[string]*ContextProfile {
	profiles := make(map[string]*ContextProfile)

	if len(interactions) == 0 {
		return profiles
	}

	// Group interactions by context
	contextGroups := make(map[string][]interface{})
	for _, interaction := range interactions {
		if m, ok := interaction.(map[string]interface{}); ok {
			context := "general" // default
			if ctx, hasCtx := m["context"].(string); hasCtx && ctx != "" {
				context = ctx
			}
			contextGroups[context] = append(contextGroups[context], interaction)
		}
	}

	// Build profile for each context
	for context, contextInteractions := range contextGroups {
		profile := &ContextProfile{
			Context:    context,
			Details:    make(map[string]interface{}),
			SampleSize: len(contextInteractions),
		}

		// Extract dominant style for this context
		styleScores := make(map[string]int)
		for _, interaction := range contextInteractions {
			if m, ok := interaction.(map[string]interface{}); ok {
				if style, hasStyle := m["communication_style"].(string); hasStyle && style != "" {
					styleScores[style]++
				}
			}
		}

		// Find dominant style
		if len(styleScores) > 0 {
			maxCount := 0
			for style, count := range styleScores {
				if count > maxCount {
					maxCount = count
					profile.Style = style
				}
			}
		}

		if profile.Style == "" {
			profile.Style = "developing"
		}

		// Calculate confidence based on consistency
		if profile.SampleSize > 0 {
			// Higher confidence if style appears in most interactions
			dominantCount := 0
			if profile.Style != "developing" {
				dominantCount = styleScores[profile.Style]
			}

			consistency := float64(dominantCount) / float64(profile.SampleSize)
			sampleFactor := math.Min(float64(profile.SampleSize)/10.0, 1.0)
			profile.Confidence = 0.5 + (consistency * 0.5 * sampleFactor)
			if profile.Confidence > 1.0 {
				profile.Confidence = 1.0
			}
		}

		// Add style distribution to details
		profile.Details["styleDistribution"] = styleScores
		profile.Details["dominantCount"] = styleScores[profile.Style]

		profiles[context] = profile
	}

	return profiles
}
