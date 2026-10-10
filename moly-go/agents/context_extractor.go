package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// NOTE: ExtractedContext, ExtractedContact, ExtractedStyle types are now defined in models/agent_types.go
// This file uses the models.* versions for consistency

// ContextExtractor uses LLM-based semantic extraction via principle-based evaluation
type ContextExtractor struct {
	llmClient tools.LLMProvider
}

// NewContextExtractor creates a new context extractor with LLM-based semantic extraction
func NewContextExtractor(llmClient tools.LLMProvider) *ContextExtractor {
	return &ContextExtractor{
		llmClient: llmClient,
	}
}

// Extract analyzes a message and returns structured context with confidence scores
// Uses semantic extraction first, LLM fallback for low-confidence fields
func (ce *ContextExtractor) Extract(ctx context.Context, userMessage string) (*models.ExtractedContext, error) {
	if userMessage == "" {
		return &models.ExtractedContext{}, nil
	}

	log.Printf("[ContextExtractor] LLM-based semantic extraction: message_len=%d", len(userMessage))

	// STEP 1: LLM-based semantic extraction (principle-based evaluation, no pattern matching)
	// Uses buildExtractionPrompt() which asks LLM to extract in ONE call:
	// - Goal/intention (user's primary objective)
	// - Contacts (who they're discussing)
	// - Style/tone (how they communicate)
	// - Values (what matters to them)
	// - Characteristics (traits about user and contacts)
	// - Entities (all important concepts)

	prompt := ce.buildExtractionPrompt(userMessage)
	req := &tools.LLMRequest{
		UserPrompt:  prompt,
		Temperature: 0.3,
		MaxTokens:   2000,
		Retries:     2,
	}
	llmResponse, err := ce.llmClient.Call(ctx, req)
	if err != nil {
		log.Printf("[ContextExtractor] ⚠️ LLM extraction failed: %v, using fallback", err)
		return ce.basicExtraction(userMessage), nil
	}

	// Parse LLM JSON response
	extracted, parseErr := ce.parseLLMExtraction(llmResponse.Content)
	if parseErr != nil {
		log.Printf("[ContextExtractor] ⚠️ Failed to parse LLM response: %v, using fallback", parseErr)
		return ce.basicExtraction(userMessage), nil
	}

	applyFactConfidence(extracted, userMessage)

	log.Printf("[ContextExtractor] ✅ LLM semantic extraction complete - Goal conf: %.2f, Contact: %v, Characteristics: %d",
		extracted.IntentionConfidence, extracted.Contact != nil, len(extracted.UserCharacteristics))

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

A CONTACT is a THIRD PERSON the user wants advice/help with.
Extract ONLY contacts that are separate people (not Moly/Μώλυ, not self-references).

Message: "%s"

Extract and return JSON with:
- contact: {name, label, relationship (romantic|professional|family|friend|other), traits[], confidence (0-1), evidence, basis} [OMIT if user is addressing you or discussing themselves]
  name: the person's real name ONLY if the user gave one in this message (for example "her name is Anna" or "call her Girl from fet"). Otherwise name is an empty string.
  evidence: the exact words of the message that show this person.
  basis: "stated" if the user said it in so many words, "implied" if it follows from what they said, "guessed" if you are filling in something they did not say.
  label: the user's own words for the person, copied from this message. Never copy label into name. If this message does not mention a third person, omit contact entirely: never take a person from these instructions.
- intention: main goal/purpose (1-2 sentences capturing the FULL semantic goal. Include what they're trying to accomplish, who it involves, and what constraints matter. Leave it EMPTY (empty string) when the message is only a greeting, small talk, or a question about Moly itself with no goal of the user's. Also leave it EMPTY when the message only comments on, corrects or asks to adjust something Moly just wrote (for example "a bit shorter" or "too formal"): that continues the current goal and is not a new one. Also leave it EMPTY when the message only agrees or disagrees (for example "yes" or "no") or only answers a question that was just asked, for example giving a name or a detail: an answer is not a new goal, and you must not invent one such as identifying a person.)
- intentionBasis: "stated", "implied" or "guessed", for the intention, as for the contact basis
- intentionEvidence: the exact words of the message that state the goal, copied from it (a few words are enough). Empty when the intention is empty. If you cannot point to words of the message that state a goal, the intention must be empty.
- intentionPrinciples: constitutional principles engaged by this intention (select from: transparency, autonomy, empathy, fairness, growth, stakeholder)
  * transparency: communicating honestly/openly with others
  * autonomy: making own choices, standing up for self, independence
  * empathy: understanding/considering others' perspectives and needs
  * fairness: equity, just treatment, reciprocity in relationships
  * growth: learning, self-improvement, developing capabilities
  * stakeholder: considering impact on others, multiple perspectives
- userCharacteristics: [traits about the USER/person writing] - tag subject explicitly (FIX #5) - keep each trait 1-3 words
- contactCharacteristics: {[contact_name]: [traits about this contact]} - tag with actual contact name (FIX #5) - keep each trait 1-3 words
- systemFeedback: {isFeedback, feedbackType (positive|negative|directive|perception), feedback[], directives[], perceptions[], style, confidence (0-1), evidence}
  When user addresses Moly (says "you", "Moly", gives feedback about the system):
  * feedback: "too verbose", "helpful", "confusing", "clear"
  * directives: "be more concise", "ask questions", "stop asking about family"
  * perceptions: "good at analysis", "lacks empathy", "can't do legal"
  * style: ONLY if user gives directive about interaction style: "direct", "socratic", "collaborative"
  [OMIT if message is not about the system]
- entities: [{name, type (topic|goal_component|concern|context), confidence (0-1), evidence}] - COMBINED EXTRACTION (FIX #6)
  Extract ALL important entities/concepts from the message, not just contact info.
  This includes: topics discussed, goals mentioned, concerns raised, key concepts.

CRITICAL - SUBJECT TAGGING FOR ALL CHARACTERISTICS
When extracting any characteristic, preference, or value, ALWAYS tag the subject:
- If about USER (the person writing): tag as "USER|trait|confidence"
- If about CONTACT: tag as "CONTACT_[actual_name]|trait|confidence"

NEVER create undefined contacts:
- Only extract contact characteristics if clearly linked to a contact name
- Always reference the actual contact name from extracted contact object
- If no contact name known, don't extract contact characteristics

Only include fields that are clearly evident. If field is not mentioned, omit it.
Confidence should reflect how certain you are based on explicit mentions.
Evidence should be a quote or reference from the message.

Return ONLY valid JSON, no other text.`, userMessage)
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

// parseLLMExtraction parses the JSON response from LLM extraction into ExtractedContext
func (ce *ContextExtractor) parseLLMExtraction(llmJSON string) (*models.ExtractedContext, error) {
	// Decode JSON response from LLM
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(llmJSON), &response); err != nil {
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w", err)
	}

	extracted := &models.ExtractedContext{}

	// Extract intention
	if intention, ok := response["intention"].(string); ok && intention != "" {
		extracted.Intention = intention
		if basis, ok := response["intentionBasis"].(string); ok {
			extracted.IntentionBasis = strings.ToLower(strings.TrimSpace(basis))
		}
		if evidence, ok := response["intentionEvidence"].(string); ok {
			extracted.IntentionEvidence = strings.TrimSpace(evidence)
		}
	}

	// Extract intentionPrinciples
	if principles, ok := response["intentionPrinciples"].([]interface{}); ok {
		for _, p := range principles {
			if prin, ok := p.(string); ok {
				extracted.IntentionPrinciples = append(extracted.IntentionPrinciples, prin)
			}
		}
	}

	// Extract goals
	if goals, ok := response["goals"].([]interface{}); ok {
		for _, g := range goals {
			if goal, ok := g.(string); ok {
				extracted.Goals = append(extracted.Goals, goal)
			}
		}
	}

	// Extract userCharacteristics
	if chars, ok := response["userCharacteristics"].([]interface{}); ok {
		for _, c := range chars {
			if char, ok := c.(string); ok {
				extracted.UserCharacteristics = append(extracted.UserCharacteristics, char)
			}
		}
	}

	// Extract contact
	if contactObj, ok := response["contact"].(map[string]interface{}); ok {
		contact := &models.ExtractedContact{}
		if name, ok := contactObj["name"].(string); ok {
			contact.Name = strings.TrimSpace(name)
		}
		// PHASE 5: a contact without a given name keeps the user's own words as a label, and is not treated as named
		if label, ok := contactObj["label"].(string); ok && strings.TrimSpace(label) != "" {
			contact.Label = strings.TrimSpace(label)
		}
		contact.NameKnown = contact.Name != ""
		if contact.Name == "" && contact.Label != "" {
			contact.Name = contact.Label // stored under the label until the user gives a name
		}
		if rel, ok := contactObj["relationship"].(string); ok {
			contact.Relationship = rel
		}
		// The model's own number, or 0 when it gave none. The confidence the rest of Moly uses is worked out per fact
		// in applyFactConfidence; a missing number is never read as high.
		if conf, ok := contactObj["confidence"].(float64); ok {
			contact.Confidence = conf
		}
		if ev, ok := contactObj["evidence"].(string); ok {
			contact.Evidence = strings.TrimSpace(ev)
		}
		if basis, ok := contactObj["basis"].(string); ok {
			contact.Basis = strings.ToLower(strings.TrimSpace(basis))
		}
		if traits, ok := contactObj["traits"].([]interface{}); ok {
			for _, t := range traits {
				if trait, ok := t.(string); ok {
					contact.Traits = append(contact.Traits, trait)
				}
			}
		}
		if contact.Name != "" {
			extracted.Contact = contact
		}
	}

	// DISCONNECTED: Style and UserValues extraction
	// These are communication preferences (config), not extracted data
	// Intentionally omitted from extraction to avoid false inference

	// Extract contactCharacteristics
	if contactChars, ok := response["contactCharacteristics"].(map[string]interface{}); ok {
		extracted.ContactCharacteristics = make(map[string][]string)
		for contactName, chars := range contactChars {
			if charSlice, ok := chars.([]interface{}); ok {
				for _, c := range charSlice {
					if char, ok := c.(string); ok {
						extracted.ContactCharacteristics[contactName] = append(extracted.ContactCharacteristics[contactName], char)
					}
				}
			}
		}
	}

	// Extract systemFeedback (when user gives feedback about Moly)
	if sysFeedback, ok := response["systemFeedback"].(map[string]interface{}); ok {
		feedback := &models.SystemFeedbackInfo{}
		if isFb, ok := sysFeedback["isFeedback"].(bool); ok {
			feedback.IsFeedback = isFb
		}
		if fbType, ok := sysFeedback["feedbackType"].(string); ok {
			feedback.FeedbackType = fbType
		}
		if feedbackArr, ok := sysFeedback["feedback"].([]interface{}); ok {
			for _, f := range feedbackArr {
				if fb, ok := f.(string); ok {
					feedback.Feedback = append(feedback.Feedback, fb)
				}
			}
		}
		if directivesArr, ok := sysFeedback["directives"].([]interface{}); ok {
			for _, d := range directivesArr {
				if dir, ok := d.(string); ok {
					feedback.Directives = append(feedback.Directives, dir)
				}
			}
		}
		if perceptionsArr, ok := sysFeedback["perceptions"].([]interface{}); ok {
			for _, p := range perceptionsArr {
				if perc, ok := p.(string); ok {
					feedback.Perceptions = append(feedback.Perceptions, perc)
				}
			}
		}
		if style, ok := sysFeedback["style"].(string); ok {
			feedback.Style = style
		}
		if conf, ok := sysFeedback["confidence"].(float64); ok {
			feedback.Confidence = conf
		}
		if evid, ok := sysFeedback["evidence"].(string); ok {
			feedback.Evidence = evid
		}

		if feedback.IsFeedback || len(feedback.Feedback) > 0 || len(feedback.Directives) > 0 || len(feedback.Perceptions) > 0 {
			extracted.SystemFeedback = feedback
		}
	}

	// Ensure all arrays are initialized (not nil)
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

	return extracted, nil
}

// applyFactConfidence works out the confidence of each saved fact (see fact_confidence.go). It replaces the old
// constants (an intention was always 0.90, a contact without a number was 0.85).
func applyFactConfidence(ec *models.ExtractedContext, message string) {
	if ec == nil {
		return
	}
	if ec.Intention != "" {
		// A goal is only as sure as its quote: the exact words that state it must be in the message. Without them it is
		// capped as a guess, held and confirmed, never locked (a goal was invented from messages that state none).
		grounded := QuoteInMessage(message, ec.IntentionEvidence)
		ec.IntentionConfidence = FactConfidence(1, ec.IntentionBasis, grounded)
		log.Printf("[FactConfidence] goal: basis=%q grounded=%v -> %.2f (in doubt: %v)", ec.IntentionBasis, grounded, ec.IntentionConfidence, FactInDoubt(ec.IntentionConfidence))
	} else {
		ec.IntentionConfidence = 0
	}
	if c := ec.Contact; c != nil {
		grounded := QuoteInMessage(message, c.Evidence) || QuoteInMessage(message, c.Label) || (c.NameKnown && QuoteInMessage(message, c.Name))
		model := c.Confidence
		c.Confidence = FactConfidence(model, c.Basis, grounded)
		log.Printf("[FactConfidence] person: model=%.2f basis=%q grounded=%v -> %.2f (in doubt: %v)", model, c.Basis, grounded, c.Confidence, FactInDoubt(c.Confidence))
	}
}
