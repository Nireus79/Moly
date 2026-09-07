package tools

import (
	"context"
	"errors"
	"fmt"
)

// ContextExtractorInput - Input for context extraction
type ContextExtractorInput struct {
	Message                    string
	ConversationHistory        []string
	ExistingCharacteristics    []string
	ExistingInterests          []string
	ExistingCommunicationPrefs string
}

// ContextExtractorOutput - Output from context extraction
type ContextExtractorOutput struct {
	NewCharacteristics         []string
	NewInterests              []string
	UpdatedCommunicationPrefs string
	UserQuotes                []string
	Intentions                []string
	RelationshipPhase         string
	Confidence                float64 // 0-1
	ExtractedAt               int64
}

// InsightType - Type of extracted insight
type InsightType string

const (
	InsightTypeCharacteristic InsightType = "characteristic"
	InsightTypeInterest       InsightType = "interest"
	InsightTypePref           InsightType = "preference"
	InsightTypeIntention      InsightType = "intention"
	InsightTypePhase          InsightType = "phase"
)

// ContextExtractor - Extracts insights from conversations
type ContextExtractor struct {
	llm *LLMClient
}

// NewContextExtractor - Create new context extractor
func NewContextExtractor(llm *LLMClient) *ContextExtractor {
	return &ContextExtractor{
		llm: llm,
	}
}

// Extract - Extract insights from conversation
func (ce *ContextExtractor) Extract(ctx context.Context, input *ContextExtractorInput) (*ContextExtractorOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.Message == "" {
		return nil, errors.New("message cannot be empty")
	}

	output := &ContextExtractorOutput{
		NewCharacteristics:         []string{},
		NewInterests:              []string{},
		UpdatedCommunicationPrefs: input.ExistingCommunicationPrefs,
		UserQuotes:                []string{},
		Intentions:                []string{},
		Confidence:                0.0,
	}

	// Use LLM for nuanced extraction
	systemPrompt := ce.buildSystemPrompt()
	userPrompt := ce.buildUserPrompt(input)

	req := &LLMRequest{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		MaxTokens:           1000,
		Temperature:         0.5,
		UseExtendedThinking: true,
		Retries:             2,
	}

	resp, err := ce.llm.Call(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("context extraction failed: %w", err)
	}

	// Parse LLM response
	// TODO: Implement parsing logic when actual LLM integration is done
	if resp.Content != "" {
		output.UserQuotes = []string{resp.Content}
	}

	output.Confidence = 0.75

	return output, nil
}

// buildSystemPrompt - Build system prompt for extraction
func (ce *ContextExtractor) buildSystemPrompt() string {
	return `You are an expert at extracting insights from conversation.
Your job is to identify what the user reveals about the contact they're talking to.

IMPORTANT RULES:
1. Only extract what was explicitly stated
2. Don't infer or guess beyond the words
3. Keep the user's exact language where possible
4. Look for:
   - Characteristics (personality traits, behaviors)
   - Interests (things they're passionate about)
   - Communication preferences (how they like to be talked to)
   - User's intentions (what they want to achieve)
   - Relationship phase (beginning, deepening, etc.)

5. Flag what you're confident about vs. uncertain
6. Always include relevant user quotes

Format your response clearly with sections for each category.`
}

// buildUserPrompt - Build user prompt for extraction
func (ce *ContextExtractor) buildUserPrompt(input *ContextExtractorInput) string {
	history := ""
	if len(input.ConversationHistory) > 0 {
		history = "\nPrevious exchanges:\n"
		for i, msg := range input.ConversationHistory {
			if i >= 3 {
				break
			}
			history += fmt.Sprintf("- %s\n", msg)
		}
	}

	existing := ""
	if len(input.ExistingCharacteristics) > 0 {
		existing = fmt.Sprintf("\nAlready known: %v\n", input.ExistingCharacteristics)
	}

	return fmt.Sprintf(`Extract insights from this message:
"%s"%s%s
What does this reveal about who they are and what they want?
Which quotes best capture these insights?
How confident are you in these insights?`,
		input.Message, history, existing)
}

// ExtractDirectly - Direct extraction without LLM (fast path)
func (ce *ContextExtractor) ExtractDirectly(input *ContextExtractorInput) *ContextExtractorOutput {
	output := &ContextExtractorOutput{
		NewCharacteristics:         []string{},
		NewInterests:              []string{},
		UpdatedCommunicationPrefs: input.ExistingCommunicationPrefs,
		UserQuotes:                []string{},
		Intentions:                []string{},
		RelationshipPhase:         "developing",
		Confidence:                0.5,
	}

	// Simple keyword-based extraction
	if len(input.Message) > 10 {
		output.UserQuotes = []string{input.Message}
	}

	return output
}
