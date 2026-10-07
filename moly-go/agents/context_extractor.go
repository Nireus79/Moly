package agents

import (
	"context"
	"fmt"
	"log"
	"time"

	"moly/models"
	"moly/tools"
)

// NOTE: ExtractedContext, ExtractedContact, ExtractedStyle types are now defined in models/agent_types.go
// This file uses the models.* versions for consistency

// ContextExtractor uses semantic extraction first, LLM as fallback for low-confidence fields
type ContextExtractor struct {
	llmClient           tools.LLMProvider
	semanticExtractor   *SemanticExtractor
}

// NewContextExtractor creates a new context extractor with semantic framework
func NewContextExtractor(llmClient tools.LLMProvider) *ContextExtractor {
	return &ContextExtractor{
		llmClient:         llmClient,
		semanticExtractor: NewSemanticExtractor(),
	}
}

// Extract analyzes a message and returns structured context with confidence scores
// Uses semantic extraction first, LLM fallback for low-confidence fields
func (ce *ContextExtractor) Extract(ctx context.Context, userMessage string) (*models.ExtractedContext, error) {
	if userMessage == "" {
		return &models.ExtractedContext{}, nil
	}

	log.Printf("[ContextExtractor] FIX #3 Semantic Extraction (Session 34): Extracting from: %.100s...", userMessage)

	// STEP 1: Semantic extraction (linguistic parsing for all entity types)
	semanticResult := ce.semanticExtractor.Extract(userMessage)
	log.Printf("[ContextExtractor] Semantic extraction complete - Goal conf: %.2f, Contacts: %d, Values: %d",
		semanticResult.IntentionConfidence, len(semanticResult.ExtractedContacts), len(semanticResult.UserValues))

	// STEP 2: Confidence-driven fallback - use LLM for low-confidence fields
	// High confidence (>= 0.80): use semantic result
	// Low confidence (< 0.80): ask clarification question
	// For now, return semantic result with confidence scores for downstream layers

	// TODO: Integrate clarification question generation for low-confidence fields
	// This will be FIX #4: clarification engine using confidence scores

	log.Printf("[ContextExtractor] ✅ Semantic extraction framework active - confidence-driven approach")

	// Semantic extraction is now the primary method
	// LLM fallback for low-confidence fields will be implemented in FIX #4
	extracted := semanticResult

	// FIX #26 & #27: Validate all ExtractedContext fields against specification
	if extracted != nil {
		// Sanitize intention (FIX #25: prevent full message echo)
		extracted.Intention = tools.SanitizeIntention(extracted.Intention)

		// Validate IntentionConfidence (0-1 range)
		if extracted.IntentionConfidence < 0 || extracted.IntentionConfidence > 1 {
			log.Printf("[ContextExtractor] FIX #27: IntentionConfidence out of range (%.2f), clamping to [0,1]", extracted.IntentionConfidence)
			if extracted.IntentionConfidence < 0 {
				extracted.IntentionConfidence = 0
			} else if extracted.IntentionConfidence > 1 {
				extracted.IntentionConfidence = 1
			}
		}

		// Validate Goals array (should have 1-5 items if present)
		if len(extracted.Goals) > 5 {
			log.Printf("[ContextExtractor] FIX #27: Goals array too large (%d items), keeping first 5", len(extracted.Goals))
			extracted.Goals = extracted.Goals[:5]
		}

		// Validate each goal is reasonable length (1-50 chars)
		for i, goal := range extracted.Goals {
			if len(goal) > 50 {
				log.Printf("[ContextExtractor] FIX #27: Goal %d too long (%d chars), truncating", i, len(goal))
				extracted.Goals[i] = goal[:50]
			}
		}

		// Ensure arrays aren't nil (FIX #27: prevent nil dereference)
		if extracted.Goals == nil {
			extracted.Goals = []string{}
		}
		if extracted.UserValues == nil {
			extracted.UserValues = []string{}
		}
		if extracted.UserCharacteristics == nil {
			extracted.UserCharacteristics = []string{}
		}
		if extracted.IntentionPrinciples == nil {
			extracted.IntentionPrinciples = []string{}
		}
		if extracted.ContactCharacteristics == nil {
			extracted.ContactCharacteristics = make(map[string][]string)
		}
	}

	log.Printf("[ContextExtractor] Successfully extracted context (intention=%q, goals=%d, confidence=%.2f)",
		extracted.Intention, len(extracted.Goals), extracted.IntentionConfidence)
	return extracted, nil
}

func (ce *ContextExtractor) buildExtractionPrompt(userMessage string) string {
	return fmt.Sprintf(`You are Μώλυ (also called Moly in English), an AI thinking partner. The user is talking TO you.

Your task: Extract structured information about OTHER PEOPLE the user wants to discuss or get advice about.

SELF-AWARENESS:
YOU are Μώλυ/Moly - both names (Greek and English) refer to YOU (the AI system).
References to "Moly", "Μώλυ", "you" (addressing you), "I"/"me" (the user), are about the conversation happening between you and the user - NOT about a contact to discuss.

A CONTACT is a THIRD PERSON the user wants advice/help with:
- "My girlfriend Sarah is..." → Sarah is the contact
- "I want to message my boss..." → The boss is the contact
- "I love my sister" → Sister is the contact
- "Hello Moly" → NO contact (greeting to you)
- "Tell Μώλυ something" → NO contact (addressing you)
- "I want to tell you something" → NO contact (direct address to you)

Extract ONLY contacts that are separate people (not Moly/Μώλυ, not self-references).

Message: "%s"

Extract and return JSON with:
- contact: {name, relationship (romantic|professional|family|friend|other), traits[], confidence (0-1), evidence} [OMIT if user is addressing you or discussing themselves]
- style: {style (casual|formal|playful|mix), tone, values[], confidence (0-1)}
- intention: main goal/purpose (KEEP TO 2-5 WORDS ONLY - this field is used in follow-up questions, so keep it concise. Example: "improve communication" not "I want to know how to improve communication with my girlfriend".)
- intentionPrinciples: constitutional principles engaged by this intention (select from: transparency, autonomy, empathy, fairness, growth, stakeholder)
  * transparency: communicating honestly/openly with others
  * autonomy: making own choices, standing up for self, independence
  * empathy: understanding/considering others' perspectives and needs
  * fairness: equity, just treatment, reciprocity in relationships
  * growth: learning, self-improvement, developing capabilities
  * stakeholder: considering impact on others, multiple perspectives
- goals: list of short objectives (2-4 words each, e.g., ["improve communication", "understand her better"], not full sentence descriptions)
- userCharacteristics: [traits about the USER/person writing] - tag subject explicitly (FIX #5) - keep each trait 1-3 words
- contactCharacteristics: {[contact_name]: [traits about this contact]} - tag with actual contact name (FIX #5) - keep each trait 1-3 words
- entities: [{name, type (topic|goal_component|value|concern|context), confidence (0-1), evidence}] - COMBINED EXTRACTION (FIX #6)
  Extract ALL important entities/concepts from the message, not just contact info.
  This includes: topics discussed, goals mentioned, values expressed, concerns raised, key concepts.

CRITICAL - FIX #5: SUBJECT TAGGING FOR ALL CHARACTERISTICS
When extracting any characteristic, preference, or value, ALWAYS tag the subject:
- If about USER (the person writing): tag as "USER|trait|confidence"
  Example: "I'm dominant" → Include in userCharacteristics: ["USER|dominant|0.9"]
- If about CONTACT: tag as "CONTACT_[actual_name]|trait|confidence"
  Example: "Sarah is reserved" → Include contactCharacteristics["Sarah"]: ["CONTACT_Sarah|reserved|0.85"]

NEVER create undefined contacts:
- "good girl" in isolation (with undefined contact) → Only extract if clearly linked to a contact name
- Always reference the actual contact name from extracted contact object
- If no contact name known, don't extract contact characteristics

Only include fields that are clearly evident. If field is not mentioned, omit it.
Confidence should reflect how certain you are based on explicit mentions.
Evidence should be a quote or reference from the message.

Return ONLY valid JSON, no other text.

Example format (MUST be valid JSON with ALL fields including entities):
{
  "contact": {
    "name": "Sarah",
    "relationship": "romantic",
    "traits": ["intelligent", "kind"],
    "confidence": 0.9,
    "evidence": "I want to talk with you about a girl I'm seeing"
  },
  "style": {
    "style": "casual",
    "tone": "friendly",
    "confidence": 0.7
  },
  "intention": "get advice on romantic relationship",
  "intentionPrinciples": ["empathy", "stakeholder"],
  "goals": ["improve communication", "understand her better"],
  "userCharacteristics": ["USER|caring|0.85", "USER|thoughtful|0.80"],
  "contactCharacteristics": {
    "Sarah": ["CONTACT_Sarah|intelligent|0.90", "CONTACT_Sarah|kind|0.85"]
  },
  "entities": [
    {"name": "romantic relationship", "type": "topic", "confidence": 0.95, "evidence": "I'm interested to..."},
    {"name": "communication", "type": "goal_component", "confidence": 0.90, "evidence": "improve communication"},
    {"name": "understanding", "type": "value", "confidence": 0.85, "evidence": "understand her better"}
  ]
}`, userMessage)
}

// extractGoalLinguistically uses verb/subject/object parsing to extract goal without LLM
// FIX #73: Uses ExtractionOrchestrator to parse sentences and extract goal deterministically
// Returns empty string if linguistic parsing doesn't find a clear goal (will fall back to LLM)
func (ce *ContextExtractor) extractGoalLinguistically(userMessage string) string {
	orchestrator := tools.NewExtractionOrchestrator()

	// Analyze message using linguistic parsing
	analysis, err := orchestrator.AnalyzeMessageForExtraction(userMessage, []string{}, nil)
	if err != nil {
		log.Printf("[ContextExtractor] FIX #73: Linguistic analysis failed: %v", err)
		return ""
	}

	if analysis == nil || len(analysis.SentenceAnalyses) == 0 {
		return ""
	}

	// Extract goal from verb + object in sentences
	for _, sent := range analysis.SentenceAnalyses {
		// Look for action verbs: need, want, help, improve, understand, solve, write, etc.
		actionVerbs := map[string]bool{
			"need": true, "want": true, "help": true, "improve": true,
			"understand": true, "solve": true, "write": true, "get": true,
			"make": true, "create": true, "learn": true, "achieve": true,
		}

		verb := sent.Verb
		if verb == "" {
			continue
		}

		// Check if this is an action verb we care about
		if !actionVerbs[verb] {
			continue
		}

		// Construct goal from verb + object
		if sent.Object != "" {
			// "need" + "message" = "need message" (will be truncated to 5 words by SanitizeIntention)
			goal := verb + " " + sent.Object
			log.Printf("[ContextExtractor] FIX #73: Extracted linguistic goal: %q (verb=%s, object=%s, confidence=%.2f)",
				goal, verb, sent.Object, sent.Confidence)
			return goal
		}

		// If no object, use verb alone as last resort
		if sent.Confidence > 0.7 {
			log.Printf("[ContextExtractor] FIX #73: Using verb alone as goal: %q (confidence=%.2f)", verb, sent.Confidence)
			return verb
		}
	}

	log.Printf("[ContextExtractor] FIX #73: Linguistic parsing found no clear goal in %d sentences", len(analysis.SentenceAnalyses))
	return ""
}

// basicExtraction provides fallback extraction without LLM
// Returns safe defaults without any keyword matching
// Reasoning: No data is safer than wrong data from hardcoded keywords
// User will get asked clarifying questions naturally in conversation flow
func (ce *ContextExtractor) basicExtraction(userMessage string) *models.ExtractedContext {
	log.Printf("[ContextExtractor] Using safe fallback extraction (no keyword matching)")

	// Return safe defaults - all fields empty/nil (not extracted)
	// This forces clarification questions in the normal workflow
	extracted := &models.ExtractedContext{}
	extracted.IntentionPrinciples = []string{} // No principles detected

	return extracted
}
