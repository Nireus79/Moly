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
		UserPrompt: prompt,
		MaxTokens: 500,
		Temperature: 0.3,
		Retries: 1,
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

	log.Printf("[ContextExtractor] Successfully extracted context")
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
- intention: main goal/purpose
- involvesMessaging: true if user will send message/communicate directly to contact, false if seeking advice/thoughts only
- pronouns: pronouns used to describe the contact (he, she, they, other)
- goals: list of objectives

Only include fields that are clearly evident. If field is not mentioned, omit it.
Confidence should reflect how certain you are based on explicit mentions.
Evidence should be a quote or reference from the message.

Return ONLY valid JSON, no other text.

Example format:
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
  "involvesMessaging": false,
  "pronouns": "she",
  "goals": ["improve communication", "understand her better"]
}`, userMessage)
}


// basicExtraction provides fallback extraction without LLM
// TODO: REMOVE ALL HARDCODED KEYWORDS - Use safe defaults instead
// Problem: Keywords like "girl", "crush", "girlfriend" for romantic detection
//          "boss", "manager", "colleague" for professional detection
//          "mom", "dad", "parent" for family detection
//          These can be easily bypassed by rewording
// Solution: When LLM fails, return empty/unknown values instead of keyword matching:
//   - Set Contact = nil (not extracted)
//   - Set Style = nil (not extracted)
//   - Set InvolvesMessaging = false (conservative default)
//   - Set Pronouns = "other" (conservative default)
// Reasoning: No data is safer than wrong data from keyword matching
// User will get asked clarifying questions naturally in the conversation flow
func (ce *ContextExtractor) basicExtraction(userMessage string) *models.ExtractedContext {
	log.Printf("[ContextExtractor] Using safe fallback extraction (no keyword matching)")

	// REMOVED: All hardcoded keyword extraction for contact types
	// REMOVED: All hardcoded keyword extraction for style
	// Return safe defaults instead of guessing via keywords
	extracted := &models.ExtractedContext{}

	extracted.InvolvesMessaging = false
	extracted.Pronouns = "other"

	return extracted
}

