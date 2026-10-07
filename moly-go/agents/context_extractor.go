package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"moly/models"
	"moly/tools"
)

// NOTE: ExtractedContext, ExtractedContact, ExtractedStyle types are now defined in models/agent_types.go
// This file uses the models.* versions for consistency

// ContextExtractor uses LLM-based semantic extraction via principle-based evaluation
type ContextExtractor struct {
	llmClient                    tools.LLMProvider
	confidenceBasedClarifications *ConfidenceBasedClarifications
}

// NewContextExtractor creates a new context extractor with LLM-based semantic extraction
func NewContextExtractor(llmClient tools.LLMProvider) *ContextExtractor {
	return &ContextExtractor{
		llmClient:                    llmClient,
		confidenceBasedClarifications: NewConfidenceBasedClarifications(),
	}
}

// Extract analyzes a message and returns structured context with confidence scores
// Uses semantic extraction first, LLM fallback for low-confidence fields
func (ce *ContextExtractor) Extract(ctx context.Context, userMessage string) (*models.ExtractedContext, error) {
	if userMessage == "" {
		return &models.ExtractedContext{}, nil
	}

	log.Printf("[ContextExtractor] LLM-based semantic extraction: Extracting from: %.100s...", userMessage)

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
- intention: main goal/purpose (1-2 sentences capturing the FULL semantic goal, not truncated. Example: "craft a smart, playful opening message to Christine_sub" not just "initiate conversation". Include what they're trying to accomplish, who it involves, and what constraints matter.)
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
		extracted.IntentionConfidence = 0.90 // LLM extraction has high confidence by default
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
			contact.Name = name
		}
		if rel, ok := contactObj["relationship"].(string); ok {
			contact.Relationship = rel
		}
		if conf, ok := contactObj["confidence"].(float64); ok {
			contact.Confidence = conf
		} else {
			contact.Confidence = 0.85
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

	// Extract style
	if styleObj, ok := response["style"].(map[string]interface{}); ok {
		style := &models.ExtractedStyle{}
		if s, ok := styleObj["style"].(string); ok {
			style.Style = s
		}
		if t, ok := styleObj["tone"].(string); ok {
			style.Tone = t
		}
		if conf, ok := styleObj["confidence"].(float64); ok {
			style.Confidence = conf
		} else {
			style.Confidence = 0.80
		}
		extracted.Style = style
	}

	// Extract userValues
	if values, ok := response["userValues"].([]interface{}); ok {
		for _, v := range values {
			if val, ok := v.(string); ok {
				extracted.UserValues = append(extracted.UserValues, val)
			}
		}
	}

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
