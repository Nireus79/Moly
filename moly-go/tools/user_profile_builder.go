package tools

import (
	"fmt"

	"moly/models"
)

// UserProfileBuilder - Builds comprehensive user profiles
type UserProfileBuilder struct {
	behaviorAnalyzer *BehaviorAnalyzer
	intentionDetector *IntentionDetector
}

// NewUserProfileBuilder - Create new profile builder
func NewUserProfileBuilder(llm LLMProvider) *UserProfileBuilder {
	return &UserProfileBuilder{
		behaviorAnalyzer:  NewBehaviorAnalyzer(),
		intentionDetector: NewIntentionDetector(llm),
	}
}

// BuildProfile - Build comprehensive user profile
func (upb *UserProfileBuilder) BuildProfile(
	userID string,
	aboutMe *models.AboutMe,
	contacts []models.Contact,
	interactions []interface{},
) *models.UserBehavioralProfile {

	profile := &models.UserBehavioralProfile{
		UserID:                userID,
		CommunicationProfile: make(map[string]interface{}),
		CommunicationGoals:   make(map[string]int),
		SuggestionChoices:    make(map[string]interface{}),
		SuccessMetrics:       make(map[string]interface{}),
		EmergingPersonality:  []string{},
		GrowthTrajectory:     make(map[string]interface{}),
	}

	// Analyze behavioral patterns
	behaviorProfile := upb.behaviorAnalyzer.AnalyzeChoicePatterns(interactions)
	profile.CommunicationProfile = behaviorProfile.CommunicationProfile
	profile.CommunicationGoals = behaviorProfile.CommunicationGoals
	profile.SuccessMetrics = behaviorProfile.SuccessMetrics

	// Analyze communication style
	style := upb.behaviorAnalyzer.AnalyzeCommunicationStyle(interactions, aboutMe)
	profile.CommunicationProfile["detected_style"] = style

	// Analyze tone preferences
	tonePrefs := upb.behaviorAnalyzer.AnalyzeTonePreference(interactions, aboutMe)
	profile.CommunicationProfile["tone_preferences"] = tonePrefs

	// Analyze growth
	growth := upb.behaviorAnalyzer.AnalyzeGrowth(interactions)
	profile.GrowthTrajectory = growth

	// Get personality insights
	insights := upb.behaviorAnalyzer.GeneratePersonalityInsights(profile, aboutMe)
	profile.EmergingPersonality = insights

	// Calculate context completeness
	completeness := upb.behaviorAnalyzer.CalculateContextCompleteness(aboutMe, contacts, len(interactions))
	profile.SuccessMetrics["context_completeness"] = fmt.Sprintf("%.0f%%", completeness*100)

	// Add about me data
	if aboutMe != nil {
		profile.CommunicationProfile["communication_style"] = aboutMe.CommunicationStyle
		profile.CommunicationProfile["preferred_tone"] = aboutMe.PreferredTone
		profile.CommunicationProfile["values"] = aboutMe.Values
	}

	// Add contact insights
	profile.CommunicationProfile["known_contacts"] = len(contacts)
	if len(contacts) > 0 {
		contactNames := make([]string, len(contacts))
		for i, contact := range contacts {
			contactNames[i] = contact.Name
		}
		profile.CommunicationProfile["contact_names"] = contactNames
	}

	// Calculate overall confidence
	if len(interactions) >= 10 {
		profile.Confidence = 0.85
	} else if len(interactions) >= 5 {
		profile.Confidence = 0.75
	} else if len(interactions) > 0 {
		profile.Confidence = 0.6
	} else {
		profile.Confidence = 0.4
	}

	profile.SuccessMetrics["interaction_count"] = len(interactions)
	profile.SuccessMetrics["total_confidence"] = fmt.Sprintf("%.1f%%", profile.Confidence*100)

	return profile
}

// GetUserPersonality - Get formatted personality description
func (upb *UserProfileBuilder) GetUserPersonality(profile *models.UserBehavioralProfile) string {
	if profile == nil {
		return "Profile not yet available"
	}

	description := "User Profile:\n"

	// Add communication style
	if style, ok := profile.CommunicationProfile["communication_style"].(string); ok {
		description += fmt.Sprintf("- Communication Style: %s\n", style)
	}

	// Add tone preference
	if tone, ok := profile.CommunicationProfile["preferred_tone"].(string); ok {
		description += fmt.Sprintf("- Preferred Tone: %s\n", tone)
	}

	// Add known contacts
	if contactCount, ok := profile.CommunicationProfile["known_contacts"].(int); ok {
		description += fmt.Sprintf("- Known Contacts: %d\n", contactCount)
	}

	// Add insights
	if len(profile.EmergingPersonality) > 0 {
		description += "- Key Insights:\n"
		for _, insight := range profile.EmergingPersonality {
			description += fmt.Sprintf("  * %s\n", insight)
		}
	}

	// Add completeness
	if completeness, ok := profile.SuccessMetrics["context_completeness"].(string); ok {
		description += fmt.Sprintf("- Context Completeness: %s\n", completeness)
	}

	return description
}

// RecommendNextAction - Recommend what to do with the user next
func (upb *UserProfileBuilder) RecommendNextAction(profile *models.UserBehavioralProfile, missingContext []string) string {
	if profile == nil {
		return "Collect initial information about communication style"
	}

	if len(missingContext) > 0 {
		return fmt.Sprintf("Ask about: %v", missingContext)
	}

	// All context available - suggest next step
	interactionCount := 0
	if ic, ok := profile.SuccessMetrics["interaction_count"].(int); ok {
		interactionCount = ic
	}

	if interactionCount < 3 {
		return "Continue building context with more conversations"
	} else if interactionCount < 10 {
		return "Gather more interaction data to strengthen predictions"
	} else if interactionCount < 20 {
		return "Provide proactive suggestions based on detected patterns"
	} else {
		return "Deep personalization ready - anticipate user needs"
	}
}

// UpdateProfile - Update profile with new interaction
func (upb *UserProfileBuilder) UpdateProfile(
	profile *models.UserBehavioralProfile,
	newInteraction interface{},
) *models.UserBehavioralProfile {

	if profile == nil || newInteraction == nil {
		return profile
	}

	// Update interaction count
	if current, ok := profile.SuccessMetrics["interaction_count"].(int); ok {
		profile.SuccessMetrics["interaction_count"] = current + 1
	}

	// Recalculate confidence
	if ic, ok := profile.SuccessMetrics["interaction_count"].(int); ok {
		if ic >= 10 {
			profile.Confidence = 0.85
		} else if ic >= 5 {
			profile.Confidence = 0.75
		} else {
			profile.Confidence = 0.6
		}
		profile.SuccessMetrics["total_confidence"] = fmt.Sprintf("%.1f%%", profile.Confidence*100)
	}

	return profile
}

// GetProfileSummary - Get brief summary of profile
func (upb *UserProfileBuilder) GetProfileSummary(profile *models.UserBehavioralProfile) map[string]interface{} {
	summary := make(map[string]interface{})

	if profile == nil {
		summary["status"] = "no_profile"
		return summary
	}

	summary["status"] = "complete"

	// Style
	if style, ok := profile.CommunicationProfile["communication_style"].(string); ok {
		summary["style"] = style
	}

	// Contacts known
	if contacts, ok := profile.CommunicationProfile["known_contacts"].(int); ok {
		summary["contacts_known"] = contacts
	}

	// Interactions
	if interactions, ok := profile.SuccessMetrics["interaction_count"].(int); ok {
		summary["interactions"] = interactions
	}

	// Confidence
	summary["confidence"] = fmt.Sprintf("%.0f%%", profile.Confidence*100)

	// Top insight
	if len(profile.EmergingPersonality) > 0 {
		summary["top_insight"] = profile.EmergingPersonality[0]
	}

	return summary
}
