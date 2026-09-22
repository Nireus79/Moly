package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"moly/models"
	"moly/tools"
)

// LLMMessageExtractor - LLM-powered context extraction
type LLMMessageExtractor struct {
	llm *tools.LLMClient
}

// NewLLMMessageExtractor - Create LLM-based extractor
func NewLLMMessageExtractor(llm *tools.LLMClient) *LLMMessageExtractor {
	return &LLMMessageExtractor{llm: llm}
}

// Extract - Extract context using LLM
func (lme *LLMMessageExtractor) Extract(ctx context.Context, msg string, userAboutMe *models.AboutMe) (*ExtractedContext, error) {
	log.Printf("[LLMMessageExtractor] Extracting context from message: %d chars", len(msg))

	if lme.llm == nil {
		return nil, fmt.Errorf("LLM client is nil - cannot extract context without LLM provider")
	}

	// Build prompt that asks LLM to extract structured context
	systemPrompt := `You are an expert at understanding user communication patterns.
Extract the following information from the user's message and return ONLY valid JSON (no markdown, no code blocks):
{
  "contact": "who they're messaging (string)",
  "style": "formal|casual|playful|neutral",
  "intention": "seek_advice|draft_message|express_concern|express_joy|seek_understanding|general_conversation",
  "goals": ["list of goals mentioned"],
  "emotional_tone": "very_negative|negative|neutral|positive|very_positive",
  "topic": "work|relationships|family|mental_health|finances|general",
  "has_conflicts": false,
  "conflict_fields": []
}`

	userPrompt := fmt.Sprintf(`Extract context from this message:
"%s"

If user's previously stored style was "%s", note if this message suggests a different style.
Return only valid JSON.`, msg, userAboutMe.CommunicationStyle)

	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.3, // Low temperature for consistent extraction
		MaxTokens:    500,
	}

	resp, err := lme.llm.Call(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("LLM context extraction failed: %w", err)
	}

	// Parse LLM response as JSON
	var extracted ExtractedContext
	if err := json.Unmarshal([]byte(resp.Content), &extracted); err != nil {
		return nil, fmt.Errorf("failed to parse LLM extraction response as JSON: %w (response was: %s)", err, resp.Content)
	}

	log.Printf("[LLMMessageExtractor] ✓ Extracted: style=%s emotion=%s topic=%s", extracted.Style, extracted.EmotionalTone, extracted.Topic)
	return &extracted, nil
}
