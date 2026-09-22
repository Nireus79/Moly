package agents

import (
	"log"
	"strings"

	"moly/database"
	"moly/models"
)

// ClarityAssessment represents the diagnostic analysis of a message's clarity
type ClarityAssessment struct {
	ClarityScore           float64                    // 0-1: how clear is the message?
	CanProceed             bool                       // true if we have enough info to respond meaningfully
	AmbiguousSubjects      []string                   // Pronouns that need clarification ("she", "he", "they")
	DetectedTopicShifts    []SubjectShift             // Topic changes detected mid-message
	MultipleTopicsDetected []string                   // If user mentioned multiple concerns
	RequiredClarifications []ClarificationNeed        // Prioritized list of what to ask about
	MissingContextFields   []string                   // Which AboutMe fields are missing
	MessageQuality         string                     // "clear", "ambiguous", "vague", "complex"
	PrimaryTopics          []string                   // Main topics in the message
}

// ClarificationNeed represents a single thing we need to clarify
type ClarificationNeed struct {
	Priority    int    // 1=critical, 2=important, 3=helpful
	Type        string // "pronoun", "topic_shift", "multiple_topics", "missing_context"
	Description string // Human-readable description
	Question    string // What to ask
	Subject     string // What this is about (if pronoun: "she", "he", etc.)
}

// MessageClarityAnalyzer - Orchestrates diagnostic analysis of message clarity
type MessageClarityAnalyzer struct {
	db                   *database.Database
	subjectAnalyzer      *SubjectAnalyzer
	subjectShiftDetector *SubjectShiftDetector
	clarificationEngine  *ClarificationEngine
	socraticSelector     *SocraticQuestionSelector
}

// NewMessageClarityAnalyzer creates a new clarity analyzer
func NewMessageClarityAnalyzer(db *database.Database, socraticSelector *SocraticQuestionSelector) *MessageClarityAnalyzer {
	return &MessageClarityAnalyzer{
		db:                   db,
		subjectAnalyzer:      NewSubjectAnalyzer(),
		subjectShiftDetector: NewSubjectShiftDetectorWithLLM(nil), // Can add LLM later if needed
		clarificationEngine:  NewClarificationEngine(),
		socraticSelector:     socraticSelector,
	}
}

// Analyze - Comprehensive clarity diagnostic
// Returns ClarityAssessment with prioritized clarification needs
func (mca *MessageClarityAnalyzer) Analyze(userMessage string, conversationHistory []models.Message) *ClarityAssessment {
	log.Printf("[MessageClarityAnalyzer] Starting clarity analysis: msg_len=%d history_len=%d", len(userMessage), len(conversationHistory))

	assessment := &ClarityAssessment{
		ClarityScore:           0.7, // Start with neutral
		CanProceed:             true,
		AmbiguousSubjects:      []string{},
		DetectedTopicShifts:    []SubjectShift{},
		MultipleTopicsDetected: []string{},
		RequiredClarifications: []ClarificationNeed{},
		MissingContextFields:   []string{},
		MessageQuality:         "clear",
		PrimaryTopics:          []string{},
	}

	if userMessage == "" {
		log.Printf("[MessageClarityAnalyzer] Empty message, aborting analysis")
		assessment.CanProceed = false
		assessment.MessageQuality = "empty"
		return assessment
	}

	// Step 1: Analyze pronouns and subject clarity
	mca.analyzeSubjectClarity(userMessage, assessment)

	// Step 2: Detect topic shifts (if conversation history exists)
	if len(conversationHistory) > 1 {
		mca.detectTopicShifts(userMessage, conversationHistory, assessment)
	}

	// Step 3: Detect multiple topics in single message
	mca.detectMultipleTopics(userMessage, assessment)

	// Step 4: Detect missing critical context
	mca.identifyMissingContext(assessment)

	// Step 5: Calculate final clarity score and decision
	mca.calculateFinalClarityScore(assessment, userMessage)

	log.Printf("[MessageClarityAnalyzer] ✓ Analysis complete: clarity=%.2f can_proceed=%v gaps=%d",
		assessment.ClarityScore, assessment.CanProceed, len(assessment.RequiredClarifications))

	return assessment
}

// analyzeSubjectClarity - Uses SubjectAnalyzer to find unclear pronouns
func (mca *MessageClarityAnalyzer) analyzeSubjectClarity(userMessage string, assessment *ClarityAssessment) {
	log.Printf("[MessageClarityAnalyzer] Step 1: Analyzing subject clarity")

	// Extract pronouns from message (simple pattern matching)
	pronouns := []string{"she", "he", "they", "it", "that"}
	lowerMsg := strings.ToLower(userMessage)

	// Check if this pronoun was already clarified in previous messages
	// Simple heuristic: if a clarification was asked for "she", don't ask again
	// (This would be improved with actual context tracking in DB)
	alreadyClarified := mca.checkIfPronounAlreadyClarified(lowerMsg)
	if alreadyClarified {
		log.Printf("[MessageClarityAnalyzer] Pronoun was already clarified in earlier messages, skipping re-clarification")
		return
	}

	for _, pronoun := range pronouns {
		// Check for pronoun in various positions: start, middle, end
		// "She likes me", "tell me she", "about she", "She?", "she's", etc.
		found := false
		lowerPronoun := strings.ToLower(pronoun)

		// Method 1: Space before and after: " she "
		if strings.Contains(lowerMsg, " "+lowerPronoun+" ") {
			found = true
		}
		// Method 2: At start of message: "she " or "she,"/"she?"/"she."
		if strings.HasPrefix(lowerMsg, lowerPronoun+" ") ||
			strings.HasPrefix(lowerMsg, lowerPronoun+",") ||
			strings.HasPrefix(lowerMsg, lowerPronoun+".") ||
			strings.HasPrefix(lowerMsg, lowerPronoun+"?") ||
			strings.HasPrefix(lowerMsg, lowerPronoun+"!") ||
			strings.HasPrefix(lowerMsg, lowerPronoun+"'") { // "She's"
			found = true
		}
		// Method 3: At end of message: " she", " she.", " she?"
		if strings.HasSuffix(lowerMsg, " "+lowerPronoun) ||
			strings.HasSuffix(lowerMsg, " "+lowerPronoun+".") ||
			strings.HasSuffix(lowerMsg, " "+lowerPronoun+"?") ||
			strings.HasSuffix(lowerMsg, " "+lowerPronoun+"!") ||
			strings.HasSuffix(lowerMsg, " "+lowerPronoun+",") {
			found = true
		}
		// Method 4: Middle of sentence followed by punctuation: "she,"/"she."/"she?"
		if strings.Contains(lowerMsg, " "+lowerPronoun+",") ||
			strings.Contains(lowerMsg, " "+lowerPronoun+".") ||
			strings.Contains(lowerMsg, " "+lowerPronoun+"?") ||
			strings.Contains(lowerMsg, " "+lowerPronoun+"!") ||
			strings.Contains(lowerMsg, " "+lowerPronoun+"'") { // " she's"
			found = true
		}

		if found {
			// Found pronoun - create a fact and analyze it
			fact := ExtractedFact{
				Value:           pronoun,
				ProposedSubject: pronoun,
				Evidence:        userMessage[:minInt(100, len(userMessage))],
			}

			analysis := mca.subjectAnalyzer.Analyze(fact)

			if analysis.ClarityLevel == "ambiguous" {
				log.Printf("[MessageClarityAnalyzer]   Found ambiguous pronoun: %s", pronoun)
				assessment.AmbiguousSubjects = append(assessment.AmbiguousSubjects, pronoun)
				assessment.ClarityScore -= 0.15
				assessment.MessageQuality = "ambiguous"

				// Add to clarification needs
				assessment.RequiredClarifications = append(assessment.RequiredClarifications, ClarificationNeed{
					Priority:    1, // Critical - must know who we're talking about
					Type:        "pronoun",
					Description: "Unclear pronoun: " + pronoun,
					Question:    analysis.RequiredQuestion,
					Subject:     analysis.DetectedSubject,
				})
			}
		}
	}

	// Also check for vague references like "she" without prior context
	if len(assessment.AmbiguousSubjects) > 0 {
		assessment.CanProceed = false
		log.Printf("[MessageClarityAnalyzer] Ambiguous pronouns found, cannot proceed without clarification")
	}
}

// detectTopicShifts - Uses SubjectShiftDetector to find topic changes
func (mca *MessageClarityAnalyzer) detectTopicShifts(userMessage string, conversationHistory []models.Message, assessment *ClarityAssessment) {
	log.Printf("[MessageClarityAnalyzer] Step 2: Detecting topic shifts")

	if len(conversationHistory) < 2 {
		return
	}

	// Get the most recent previous user message
	var previousUserMessage string
	for i := 1; i < len(conversationHistory); i++ {
		if conversationHistory[i].Role == "user" {
			previousUserMessage = conversationHistory[i].Content
			break
		}
	}

	if previousUserMessage == "" {
		log.Printf("[MessageClarityAnalyzer]   No previous user message found")
		return
	}

	// Extract subject from previous message
	previousSubject := extractPrimarySubject(previousUserMessage)
	if previousSubject == "" {
		log.Printf("[MessageClarityAnalyzer]   Could not extract previous subject")
		return
	}

	// Detect if topic shifted
	shifts := mca.subjectShiftDetector.DetectShifts(userMessage, previousSubject)
	if len(shifts) > 0 {
		log.Printf("[MessageClarityAnalyzer]   Detected %d topic shift(s): %s → %s", len(shifts), shifts[0].From, shifts[0].To)
		assessment.DetectedTopicShifts = shifts
		assessment.ClarityScore -= 0.1

		// Add to clarification needs
		assessment.RequiredClarifications = append(assessment.RequiredClarifications, ClarificationNeed{
			Priority:    2, // Important but not critical
			Type:        "topic_shift",
			Description: "Topic changed from " + shifts[0].From + " to " + shifts[0].To,
			Question:    "Just to confirm - you were talking about " + shifts[0].From + ", now you're talking about " + shifts[0].To + ". Are both concerns, or are you moving on?",
		})
	}
}

// detectMultipleTopics - Check if user mentioned multiple unrelated concerns
func (mca *MessageClarityAnalyzer) detectMultipleTopics(userMessage string, assessment *ClarityAssessment) {
	log.Printf("[MessageClarityAnalyzer] Step 3: Detecting multiple topics")

	topics := extractTopicsFromMessage(userMessage)
	assessment.PrimaryTopics = topics

	if len(topics) > 1 {
		log.Printf("[MessageClarityAnalyzer]   Found %d topics: %v", len(topics), topics)
		assessment.MultipleTopicsDetected = topics
		assessment.ClarityScore -= 0.1

		if len(topics) > 3 {
			assessment.MessageQuality = "complex"
			assessment.RequiredClarifications = append(assessment.RequiredClarifications, ClarificationNeed{
				Priority:    2,
				Type:        "multiple_topics",
				Description: "Multiple concerns mentioned",
				Question:    "You mentioned " + strings.Join(topics, ", ") + ". Which is most urgent right now?",
			})
		} else {
			// 2-3 topics is manageable but worth noting
			assessment.RequiredClarifications = append(assessment.RequiredClarifications, ClarificationNeed{
				Priority:    3,
				Type:        "multiple_topics",
				Description: "Multiple concerns mentioned",
				Question:    "I see you're concerned about " + strings.Join(topics, " and ") + ". Should I focus on all of these, or one first?",
			})
		}
	}
}

// identifyMissingContext - Check what AboutMe fields are missing
func (mca *MessageClarityAnalyzer) identifyMissingContext(assessment *ClarityAssessment) {
	log.Printf("[MessageClarityAnalyzer] Step 4: Identifying missing context fields")

	// This will be populated by ConversationAgent with ctx.Gaps
	// We're adding placeholder logic here
	// The actual gaps come from the Context object passed to ConversationAgent
}

// calculateFinalClarityScore - Determine final clarity and whether to proceed
// NOTE: CanProceed reflects whether to skip clarification gates, NOT whether to continue processing
func (mca *MessageClarityAnalyzer) calculateFinalClarityScore(assessment *ClarityAssessment, userMessage string) {
	log.Printf("[MessageClarityAnalyzer] Step 5: Calculating final clarity score")

	// Critical issues: must clarify before proceeding
	if len(assessment.AmbiguousSubjects) > 0 {
		assessment.CanProceed = false
		assessment.ClarityScore = 0.3
		log.Printf("[MessageClarityAnalyzer]   Critical: Ambiguous pronouns, must clarify first")
		return
	}

	// Important issues: should clarify but don't completely block
	if len(assessment.DetectedTopicShifts) > 0 {
		assessment.CanProceed = false // We'll ask for clarification via early return
		assessment.ClarityScore = 0.5
		log.Printf("[MessageClarityAnalyzer]   Important: Topic shift detected, should confirm before proceeding")
		return
	}

	if len(assessment.MultipleTopicsDetected) > 3 {
		assessment.CanProceed = false // We'll ask for prioritization via early return
		assessment.ClarityScore = 0.6
		assessment.MessageQuality = "complex"
		log.Printf("[MessageClarityAnalyzer]   Complex: Multiple topics (4+), should prioritize before proceeding")
		return
	}

	// Message is clear enough to proceed
	assessment.CanProceed = true
	if assessment.ClarityScore < 0.7 {
		assessment.ClarityScore = 0.7
	}
	log.Printf("[MessageClarityAnalyzer]   Clear: Final score %.2f, can proceed", assessment.ClarityScore)
}

// checkIfPronounAlreadyClarified - Check conversation history for earlier clarifications
// Returns true if a pronoun was likely already clarified (heuristic)
func (mca *MessageClarityAnalyzer) checkIfPronounAlreadyClarified(currentLowerMsg string) bool {
	// Heuristic: if previous messages mention clarifying pronouns (e.g., "my girlfriend", "the boss"),
	// then the current pronoun likely refers to something already established
	// This is a simple approach - a full implementation would track clarifications in database

	// For now, just return false (don't skip clarification)
	// TODO: Improve with database tracking of which pronouns have been clarified
	return false
}

// Helper functions

// extractPrimarySubject - Get the main subject from a message
func extractPrimarySubject(message string) string {
	lowerMsg := strings.ToLower(message)

	// Check for common subjects
	subjects := []string{"boss", "girlfriend", "friend", "family", "work", "job", "partner", "spouse", "colleague"}
	for _, subject := range subjects {
		if strings.Contains(lowerMsg, subject) {
			return subject
		}
	}

	return ""
}

// extractTopicsFromMessage - Extract all topics mentioned
func extractTopicsFromMessage(userMessage string) []string {
	lowerMsg := strings.ToLower(userMessage)
	topicsMap := make(map[string]bool)

	topicKeywords := map[string]string{
		"work":          "work",
		"job":           "work",
		"career":        "work",
		"boss":          "work",
		"colleague":     "work",
		"relationship":  "relationships",
		"partner":       "relationships",
		"girlfriend":    "relationships",
		"boyfriend":     "relationships",
		"romantic":      "relationships",
		"family":        "family",
		"parent":        "family",
		"sibling":       "family",
		"anxiety":       "mental_health",
		"depression":    "mental_health",
		"stressed":      "mental_health",
		"health":        "health",
		"money":         "finances",
		"finance":       "finances",
		"friend":        "relationships",
	}

	for keyword, topic := range topicKeywords {
		if strings.Contains(lowerMsg, keyword) {
			topicsMap[topic] = true
		}
	}

	// Convert map to slice
	var topics []string
	for topic := range topicsMap {
		topics = append(topics, topic)
	}

	return topics
}

// minInt - Return minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
