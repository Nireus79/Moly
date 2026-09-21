package agents

import (
	"log"
	"strings"

	"moly/models"
)

// Intent - What is the user doing with this message
type Intent string

const (
	IntentAsk     Intent = "asking"     // User asks Moly a question
	IntentShare   Intent = "sharing"    // User shares information/context
	IntentReact   Intent = "reacting"   // User reacts to something Moly said
	IntentVent    Intent = "venting"    // User expresses emotion
	IntentConfirm Intent = "confirming" // User confirms/corrects understanding
	IntentUnknown Intent = "unknown"    // No clear intent
)

// IntentAnalysis - Result of intent detection
type IntentAnalysis struct {
	Intent           Intent
	Confidence       float64   // 0-1
	QuestionAsked    string    // if Intent=="asking"
	InfoShared       string    // if Intent=="sharing"
	Emotional        bool      // true if high emotional content
	ReactionTarget   string    // what they're reacting to (if Intent=="reacting")
	ConfirmedStatement string  // what they're confirming (if Intent=="confirming")
}

// DetectIntent analyzes what the user is doing in this message
// Priority: Ask > React > Share > Vent > Confirm
func DetectIntent(userMessage string, conversationHistory []models.Message) IntentAnalysis {
	log.Printf("[IntentDetector] Analyzing message intent")

	msg := strings.TrimSpace(userMessage)
	if msg == "" {
		return IntentAnalysis{Intent: IntentUnknown, Confidence: 0}
	}

	analysis := IntentAnalysis{Confidence: 0}

	// Check 1: Is this a question? (highest priority)
	if isQuestion(msg) {
		analysis.Intent = IntentAsk
		analysis.Confidence = 0.9
		analysis.QuestionAsked = msg
		log.Printf("[IntentDetector] Detected ASKING (confidence=%.2f)", analysis.Confidence)
		return analysis
	}

	// Check 2: Is this a reaction to something Moly said?
	if isReaction(msg, conversationHistory) {
		analysis.Intent = IntentReact
		analysis.Confidence = 0.85
		analysis.ReactionTarget = extractReactionTarget(msg)
		log.Printf("[IntentDetector] Detected REACTING to '%s' (confidence=%.2f)",
			analysis.ReactionTarget, analysis.Confidence)
		return analysis
	}

	// Check 3: Is this sharing information?
	if isSharing(msg) {
		analysis.Intent = IntentShare
		analysis.Confidence = 0.8
		analysis.InfoShared = msg
		log.Printf("[IntentDetector] Detected SHARING (confidence=%.2f)", analysis.Confidence)
		return analysis
	}

	// Check 4: Is this venting (expressing emotion)?
	if isVenting(msg) {
		analysis.Intent = IntentVent
		analysis.Confidence = 0.75
		analysis.Emotional = true
		log.Printf("[IntentDetector] Detected VENTING (confidence=%.2f)", analysis.Confidence)
		return analysis
	}

	// Check 5: Is this confirming something?
	if isConfirming(msg) {
		analysis.Intent = IntentConfirm
		analysis.Confidence = 0.7
		analysis.ConfirmedStatement = msg
		log.Printf("[IntentDetector] Detected CONFIRMING (confidence=%.2f)", analysis.Confidence)
		return analysis
	}

	// No clear intent
	analysis.Intent = IntentUnknown
	analysis.Confidence = 0
	log.Printf("[IntentDetector] No clear intent detected")
	return analysis
}

// isQuestion checks for question markers and structure
func isQuestion(msg string) bool {
	msg = strings.ToLower(strings.TrimSpace(msg))

	// End with question mark
	if strings.HasSuffix(msg, "?") {
		return true
	}

	// Start with question words
	questionWords := []string{
		"how ", "why ", "what ", "when ", "where ", "who ",
		"should ", "could ", "can ", "will ", "would ", "do ", "does ",
		"is ", "are ", "did ", "have ", "has ", "should i", "do i",
	}

	for _, qw := range questionWords {
		if strings.HasPrefix(msg, qw) {
			return true
		}
	}

	return false
}

// isReaction checks if message is responding to something Moly said
func isReaction(msg string, history []models.Message) bool {
	msg = strings.ToLower(strings.TrimSpace(msg))

	// Check for reaction words
	reactionWords := []string{
		"but ", "actually ", "no, ", "yes, ", "well, ", "exactly ", "right, ",
		"that's ", "that is ", "i didn't ", "i don't think ", "not really ",
	}

	for _, rw := range reactionWords {
		if strings.HasPrefix(msg, rw) {
			return true
		}
	}

	// If there's a recent assistant message and this starts with contradiction/agreement
	// After prepending, history is [current_msg, previous_msg, older_msg, ...]
	// Iterate forward from index 1 to find most recent assistant message
	if len(history) > 1 {
		for i := 1; i < len(history); i++ {
			if history[i].Role == "assistant" {
				// Found most recent Moly message
				if strings.HasPrefix(msg, "but") || strings.HasPrefix(msg, "actually") ||
					strings.HasPrefix(msg, "no") || strings.HasPrefix(msg, "yes") {
					return true
				}
				break
			}
		}
	}

	return false
}

// isSharing checks for narrative/sharing language (past tense, story structure)
func isSharing(msg string) bool {
	msg = strings.ToLower(strings.TrimSpace(msg))

	// Past tense indicators
	pastIndicators := []string{
		"i said ", "she said ", "he said ", "they said ", "it was ",
		"i tried ", "we talked ", "he told me ", "she told me ",
		"i asked ", "i mentioned ", "i told ",
		"yesterday ", "last week ", "this morning ", "earlier ",
	}

	for _, indicator := range pastIndicators {
		if strings.Contains(msg, indicator) {
			return true
		}
	}

	// Story structure: multiple clauses, semicolon, connects ideas
	if strings.Contains(msg, ";") || (strings.Count(msg, ",") >= 2 && len(msg) > 50) {
		return true
	}

	return false
}

// isVenting checks for emotional expression
func isVenting(msg string) bool {
	msg = strings.ToLower(strings.TrimSpace(msg))

	// Emotional words
	emotionalWords := []string{
		"frustrated", "angry", "upset", "worried", "anxious", "scared",
		"sad", "depressed", "devastated", "heartbroken", "confused",
		"exhausted", "tired", "burnt out", "overwhelmed", "stressed",
		"hate ", "can't stand ", "disgusted", "annoyed", "irritated",
		"!!", "!!!", "...", "????",
	}

	for _, ew := range emotionalWords {
		if strings.Contains(msg, ew) {
			return true
		}
	}

	// Multiple exclamation marks
	if strings.Count(msg, "!") >= 2 {
		return true
	}

	return false
}

// isConfirming checks if user is confirming or correcting understanding
func isConfirming(msg string) bool {
	msg = strings.ToLower(strings.TrimSpace(msg))

	confirmWords := []string{
		"yes ", "exactly ", "right ", "that's correct", "that's it",
		"you got it", "that's what i meant", "no that's wrong",
		"not quite ", "not really ", "kind of ", "sort of ",
		"i mean ", "what i meant ", "basically ",
	}

	for _, cw := range confirmWords {
		if strings.HasPrefix(msg, cw) || strings.Contains(" "+msg, " "+cw) {
			return true
		}
	}

	return false
}

// extractReactionTarget tries to identify what they're reacting to
func extractReactionTarget(msg string) string {
	msg = strings.ToLower(strings.TrimSpace(msg))

	// Look for "that" or "this" references
	if strings.HasPrefix(msg, "that") {
		return "previous_statement"
	}
	if strings.HasPrefix(msg, "this") {
		return "previous_statement"
	}

	// Look for subject matter in reaction
	if strings.Contains(msg, "you said") {
		return "moly_statement"
	}
	if strings.Contains(msg, "that's") {
		return "moly_suggestion"
	}

	return "previous_message"
}

// ResponseType - The type of response to generate
type ResponseType string

const (
	ResponseDirectAnswer    ResponseType = "direct_answer"    // Answer their question directly
	ResponseAcknowledgement ResponseType = "acknowledgement"   // Acknowledge what they shared
	ResponseDeepeningQ      ResponseType = "deepening_q"      // Acknowledgement + Socratic question
	ResponseClarification   ResponseType = "clarification"    // Clarify what they meant
	ResponseValidation      ResponseType = "validation"       // Validate their feelings
	ResponseConfirmation    ResponseType = "confirmation"     // Confirm understanding
)

// RouteResponse determines what type of response to generate
// Based on intent and context
func RouteResponse(intent Intent, shouldDeepen bool) ResponseType {
	switch intent {
	case IntentAsk:
		return ResponseDirectAnswer
	case IntentShare:
		if shouldDeepen {
			return ResponseDeepeningQ
		}
		return ResponseAcknowledgement
	case IntentReact:
		return ResponseClarification
	case IntentVent:
		return ResponseValidation
	case IntentConfirm:
		return ResponseConfirmation
	default:
		return ResponseAcknowledgement
	}
}
