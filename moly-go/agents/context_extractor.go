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

// ContextExtractor uses LLM to intelligently extract structured context from messages
type ContextExtractor struct {
	llmClient tools.LLMProvider
}

// NewContextExtractor creates a new context extractor
func NewContextExtractor(llmClient tools.LLMProvider) *ContextExtractor {
	return &ContextExtractor{
		llmClient: llmClient,
	}
}

// Extract analyzes a message and returns structured context with confidence scores
func (ce *ContextExtractor) Extract(ctx context.Context, userMessage string) (*models.ExtractedContext, error) {
	if userMessage == "" {
		return &models.ExtractedContext{}, nil
	}

	log.Printf("[ContextExtractor] Extracting context from message: %.100s...", userMessage)

	// Build prompt for Claude
	prompt := ce.buildExtractionPrompt(userMessage)

	// Call LLM
	req := &tools.LLMRequest{
		SystemPrompt: `You are an expert at understanding user intent and extracting structured information from natural language messages.
Extract contact information, communication style, intentions, and goals from messages.
Respond with valid JSON only, no additional text.`,
		UserPrompt:  prompt,
		MaxTokens:   500,
		Temperature: 0.3,
		Retries:     1,
	}

	resp, err := ce.llmClient.Call(context.Background(), req)
	if err != nil {
		log.Printf("[ContextExtractor] LLM call failed: %v", err)
		// Fallback to basic extraction if LLM fails
		return ce.basicExtraction(userMessage), nil
	}

	log.Printf("[ContextExtractor] LLM response received: %d chars", len(resp.Content))

	// Parse JSON response
	extracted := &models.ExtractedContext{}
	if err := json.Unmarshal([]byte(resp.Content), extracted); err != nil {
		log.Printf("[ContextExtractor] Failed to parse LLM response as JSON: %v. Response: %s", err, resp.Content)
		// Fallback to basic extraction
		return ce.basicExtraction(userMessage), nil
	}

	// BUG FIX #25: Validate intention field is concise (2-5 words, not full message)
	if extracted.Intention != "" {
		intentionWords := len(strings.Fields(strings.TrimSpace(extracted.Intention)))
		if intentionWords > 10 {
			// Intention is way too long - LLM returned full message instead of summary
			log.Printf("[ContextExtractor] BUG FIX #25: Intention too long (%d words, expected 2-5). LLM returned full message. Truncating to first 5 words.", intentionWords)
			words := strings.Fields(extracted.Intention)
			if len(words) > 5 {
				extracted.Intention = strings.Join(words[:5], " ")
			}
		}
	}

	log.Printf("[ContextExtractor] Successfully extracted context (intention=%q)", extracted.Intention)
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
