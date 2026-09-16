package agents

import (
	"log"
	"strings"

	"moly/models"
)

// SocraticQuestionSelector is responsible for intelligently selecting which
// Socratic question to ask based on context and what's been explored so far
type SocraticQuestionSelector struct {
	library      *models.QuestionLibrary
	constitution *models.Constitution
}

// NewSocraticQuestionSelector creates a new selector with loaded configuration
func NewSocraticQuestionSelector(lib *models.QuestionLibrary, constitution *models.Constitution) *SocraticQuestionSelector {
	return &SocraticQuestionSelector{
		library:      lib,
		constitution: constitution,
	}
}

// SelectNextQuestion determines the best question to ask next
// It analyzes context to identify ambiguity, maps to principles, selects approach
func (s *SocraticQuestionSelector) SelectNextQuestion(
	ctx *models.Context,
	userMessage string,
	previousQuestions []models.SocraticQuestion,
) *models.SocraticQuestion {

	// Step 1: Identify what's ambiguous/unknown
	ambiguities := s.identifyAmbiguity(ctx, userMessage)
	if len(ambiguities) == 0 {
		log.Printf("[Selector] No ambiguities detected, skipping question")
		return nil
	}

	log.Printf("[Selector] Identified ambiguities: %v", ambiguities)

	// Step 2: Map ambiguities to principles
	affectedPrinciples := s.mapAmbiguityToPrinciples(ambiguities)
	if len(affectedPrinciples) == 0 {
		log.Printf("[Selector] No principles affected, skipping question")
		return nil
	}

	log.Printf("[Selector] Affected principles: %v", affectedPrinciples)

	// Step 3: Find uncovered question categories
	uncoveredCategories := s.getUncoveredCategories(affectedPrinciples, previousQuestions)
	if len(uncoveredCategories) == 0 {
		log.Printf("[Selector] All categories covered, skipping question")
		return nil
	}

	log.Printf("[Selector] Uncovered categories: %v", uncoveredCategories)

	// Step 4: Select appropriate Socratic approach
	approach := s.selectSocraticApproach(ambiguities[0], uncoveredCategories[0])
	log.Printf("[Selector] Selected approach: %s", approach)

	// Step 5: Get best question for this approach+category
	question := s.library.FindByApproachAndCategory(approach, uncoveredCategories[0])

	if question == nil {
		log.Printf("[Selector] No question found for approach=%s, category=%s", approach, uncoveredCategories[0])
		return nil
	}

	log.Printf("[Selector] Selected question: %s (%s)", question.ID, question.Text)
	return question
}

// identifyAmbiguity analyzes context to return list of ambiguity types
// Returns: stakeholder, consequence, principle, assumption, or alternative ambiguity
func (s *SocraticQuestionSelector) identifyAmbiguity(ctx *models.Context, userMessage string) []string {
	ambiguities := make([]string, 0)

	// Check if we know who the contact is
	if ctx.ExtractedContext == nil || ctx.ExtractedContext.Contact == nil || ctx.ExtractedContext.Contact.Name == "" {
		log.Printf("[Selector] Contact unknown - ambiguity: stakeholder")
		ambiguities = append(ambiguities, "stakeholder")
	}

	// Check if intention is clear
	if ctx.ExtractedContext == nil || ctx.ExtractedContext.Intention == "" {
		log.Printf("[Selector] Intention unclear - ambiguity: consequence")
		ambiguities = append(ambiguities, "consequence")
	}

	// Check if we know user's values/principles
	if ctx.AboutMe == nil || len(ctx.AboutMe.Values) == 0 {
		log.Printf("[Selector] Values unclear - ambiguity: principle")
		ambiguities = append(ambiguities, "principle")
	}

	// Check for assumptions (always present if conversation is early)
	messageCount := len(ctx.ConversationHistory)
	if messageCount < 3 {
		log.Printf("[Selector] Early in conversation - ambiguity: assumption")
		ambiguities = append(ambiguities, "assumption")
	}

	// Check if alternatives have been explored
	if messageCount < 5 {
		log.Printf("[Selector] Early in conversation - ambiguity: alternative")
		ambiguities = append(ambiguities, "alternative")
	}

	return ambiguities
}

// mapAmbiguityToPrinciples converts ambiguity types to principles they affect
func (s *SocraticQuestionSelector) mapAmbiguityToPrinciples(ambiguities []string) []string {
	principleMap := make(map[string]bool)

	for _, ambiguity := range ambiguities {
		switch ambiguity {
		case "stakeholder":
			// Stakeholder ambiguity affects these principles
			principleMap["stakeholder_consideration"] = true
			principleMap["consent_and_respect"] = true

		case "consequence":
			// Consequence ambiguity affects these principles
			principleMap["stakeholder_consideration"] = true
			principleMap["harm_prevention"] = true

		case "principle":
			// Principle ambiguity affects these principles
			principleMap["user_autonomy"] = true
			principleMap["growth_and_learning"] = true

		case "assumption":
			// Assumption ambiguity affects these principles
			principleMap["transparency"] = true
			principleMap["user_autonomy"] = true

		case "alternative":
			// Alternative ambiguity affects these principles
			principleMap["user_autonomy"] = true
			principleMap["growth_and_learning"] = true
		}
	}

	// Convert map to slice
	principles := make([]string, 0, len(principleMap))
	for principle := range principleMap {
		principles = append(principles, principle)
	}

	return principles
}

// getUncoveredCategories returns question categories we haven't used yet
func (s *SocraticQuestionSelector) getUncoveredCategories(
	affectedPrinciples []string,
	previousQuestions []models.SocraticQuestion,
) []string {

	// Track which categories have been asked
	usedCategories := make(map[string]bool)
	for _, q := range previousQuestions {
		usedCategories[q.Category] = true
	}

	// Map principles to their primary question categories
	categoryMap := map[string]string{
		"stakeholder_consideration": "stakeholder",
		"consent_and_respect":       "stakeholder",
		"harm_prevention":           "consequence",
		"user_autonomy":             "principle",
		"growth_and_learning":       "alternative",
		"transparency":              "assumption",
	}

	// Collect uncovered categories for affected principles
	uncovered := make(map[string]bool)
	for _, principle := range affectedPrinciples {
		category := categoryMap[principle]
		if !usedCategories[category] {
			uncovered[category] = true
		}
	}

	// Convert to slice, preserving priority order
	priorityOrder := []string{"stakeholder", "consequence", "principle", "assumption", "alternative"}
	result := make([]string, 0)
	for _, category := range priorityOrder {
		if uncovered[category] {
			result = append(result, category)
		}
	}

	return result
}

// selectSocraticApproach selects which of the 5 Socratic approaches to use
func (s *SocraticQuestionSelector) selectSocraticApproach(ambiguity, category string) string {
	// Map ambiguity types to their corresponding approaches
	approachMap := map[string]string{
		"stakeholder":  "identifying_stakeholders",
		"consequence":  "exploring_consequences",
		"principle":    "testing_universality",
		"assumption":   "revealing_assumptions",
		"alternative":  "exploring_alternatives",
	}

	if approach, exists := approachMap[ambiguity]; exists {
		return approach
	}

	// Fallback: map category to approach
	categoryMap := map[string]string{
		"stakeholder":  "identifying_stakeholders",
		"consequence":  "exploring_consequences",
		"principle":    "testing_universality",
		"assumption":   "revealing_assumptions",
		"alternative":  "exploring_alternatives",
	}

	if approach, exists := categoryMap[category]; exists {
		return approach
	}

	return "identifying_stakeholders" // ultimate fallback
}

// GetLibrary returns the question library
func (s *SocraticQuestionSelector) GetLibrary() *models.QuestionLibrary {
	return s.library
}

// GetConstitution returns the constitution
func (s *SocraticQuestionSelector) GetConstitution() *models.Constitution {
	return s.constitution
}

// SelectFollowUp selects a follow-up question based on a previous question's answer
func (s *SocraticQuestionSelector) SelectFollowUp(previousQuestion *models.SocraticQuestion, userResponse string) *models.SocraticQuestion {
	if previousQuestion == nil || len(previousQuestion.FollowUpQuestions) == 0 {
		return nil // No follow-ups available
	}

	// Select first available follow-up question from library
	for _, followUpID := range previousQuestion.FollowUpQuestions {
		followUp := s.library.FindByID(followUpID)
		if followUp != nil {
			log.Printf("[Selector] Selected follow-up: %s (from %s)", followUpID, previousQuestion.ID)
			return followUp
		}
	}

	return nil
}

// ShouldProgressDepth determines if we should ask deeper follow-up questions
// Returns true if user's response suggests they're ready to explore deeper
func (s *SocraticQuestionSelector) ShouldProgressDepth(userResponse string) bool {
	if userResponse == "" {
		return false
	}

	// Signals user is engaged and ready for depth
	depthSignals := []string{
		"i think", "i realize", "that makes sense", "i hadn't thought",
		"good point", "i understand", "explain more", "deeper", "more",
		"why", "how does", "what if", "let me think",
	}

	lowerResponse := strings.ToLower(userResponse)
	for _, signal := range depthSignals {
		if strings.Contains(lowerResponse, signal) {
			log.Printf("[Selector] Depth progression signal detected")
			return true
		}
	}

	return false
}

