package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// SuggestionGeneratorInput - Input for suggestion generation
type SuggestionGeneratorInput struct {
	UserMessage              string
	UserCommunicationStyle   string
	UserValues               []string
	ContactCharacteristics   []string
	ContactInterests         []string
	ContactRelationship      string
	RecentConversationHistory []string
	UserIntention           string
	Mode                    string // "socratic", "direct"
	Tone                    string // "formal", "friendly", "dating"
}

// SuggestionGeneratorOutput - Output from suggestion generator
type SuggestionGeneratorOutput struct {
	Suggestions  []GeneratedSuggestion
	Reasoning    string
	Quality      float64 // 0-1
	GeneratedAt  int64
}

// GeneratedSuggestion - Single generated suggestion
type GeneratedSuggestion struct {
	Index      int
	Text       string
	Tone       string
	Reasoning  string
	Confidence float64 // 0-1
	Alternative string
}

// SuggestionGenerator - Generates personalized suggestions
type SuggestionGenerator struct {
	llm *LLMClient
}

// NewSuggestionGenerator - Create new suggestion generator
func NewSuggestionGenerator(llm *LLMClient) *SuggestionGenerator {
	return &SuggestionGenerator{
		llm: llm,
	}
}

// Generate - Generate suggestions for user message
func (sg *SuggestionGenerator) Generate(ctx context.Context, input *SuggestionGeneratorInput) (*SuggestionGeneratorOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.UserMessage == "" {
		return nil, errors.New("user message cannot be empty")
	}

	if input.Tone == "" {
		input.Tone = "friendly"
	}

	if input.Mode == "" {
		input.Mode = "direct"
	}

	systemPrompt := sg.buildSystemPrompt(input)
	userPrompt := sg.buildUserPrompt(input)

	req := &LLMRequest{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		MaxTokens:           1500,
		Temperature:         0.7,
		UseExtendedThinking: true,
		Retries:             2,
	}

	resp, err := sg.llm.Call(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate suggestions: %w", err)
	}

	output := &SuggestionGeneratorOutput{
		Reasoning:   "Suggestions generated based on context and user preferences",
		Quality:     0.8,
		GeneratedAt: 0,
	}

	// Parse suggestions from response
	// TODO: Implement parsing logic when actual LLM integration is done
	if resp.Content != "" {
		output.Suggestions = []GeneratedSuggestion{
			{
				Index:      0,
				Text:       resp.Content,
				Tone:       input.Tone,
				Reasoning:  "Generated based on context and user communication style",
				Confidence: 0.85,
			},
		}
	}

	return output, nil
}

// buildSystemPrompt - Build system prompt for suggestion generation
func (sg *SuggestionGenerator) buildSystemPrompt(input *SuggestionGeneratorInput) string {
	mode := "direct communication"
	if input.Mode == "socratic" {
		mode = "Socratic questions that help the user reflect"
	}

	values := ""
	if len(input.UserValues) > 0 {
		values = fmt.Sprintf("User values: %s. ", strings.Join(input.UserValues, ", "))
	}

	return fmt.Sprintf(`You are an expert communication coach helping users craft messages.
Your role is to generate 3-5 personalized communication suggestions.

%sYour communication style: %s
Target tone: %s
Mode: %s

Key principles:
- Every suggestion should be fresh and personalized, not templated
- Reflect the user's authentic voice
- Consider the relationship and context
- Provide reasoning for each suggestion
- Include confidence score (0-1) for each

Generate suggestions that feel natural and authentic to the user.`,
		values, input.UserCommunicationStyle, input.Tone, mode)
}

// buildUserPrompt - Build user prompt for suggestion generation
func (sg *SuggestionGenerator) buildUserPrompt(input *SuggestionGeneratorInput) string {
	contactInfo := fmt.Sprintf("Contact: %s (%s)", input.ContactRelationship, strings.Join(input.ContactCharacteristics, ", "))

	if len(input.ContactInterests) > 0 {
		contactInfo += fmt.Sprintf("\nInterests: %s", strings.Join(input.ContactInterests, ", "))
	}

	history := ""
	if len(input.RecentConversationHistory) > 0 {
		history = fmt.Sprintf("\nRecent context: %s", strings.Join(input.RecentConversationHistory[:minInt(3, len(input.RecentConversationHistory))], " | "))
	}

	intention := ""
	if input.UserIntention != "" {
		intention = fmt.Sprintf("\nYour intention: %s", input.UserIntention)
	}

	return fmt.Sprintf(`Generate suggestions for this message:
"%s"

%s%s%s

Provide 3-5 suggestions with tone, reasoning, and confidence score.
Format each as: [Index] "Suggestion text" (Tone: X) Confidence: Y Reasoning: Z`,
		input.UserMessage, contactInfo, history, intention)
}

// minInt - Helper to get minimum of two ints
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
