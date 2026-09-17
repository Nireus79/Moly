package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// SuggestionGeneratorInput - Input for suggestion generation
type SuggestionGeneratorInput struct {
	UserMessage               string
	UserCommunicationStyle    string
	UserValues                []string
	ContactCharacteristics    []string
	ContactInterests          []string
	ContactRelationship       string
	RecentConversationHistory []string
	UserIntention             string
	Mode                      string // "socratic", "direct"
	Tone                      string // "formal", "friendly", "dating"
}

// SuggestionGeneratorOutput - Output from suggestion generator
type SuggestionGeneratorOutput struct {
	Suggestions []GeneratedSuggestion
	Reasoning   string
	Quality     float64 // 0-1
	GeneratedAt int64
}

// GeneratedSuggestion - Single generated suggestion
type GeneratedSuggestion struct {
	Index       int
	Text        string
	Tone        string
	Reasoning   string
	Confidence  float64 // 0-1
	Alternative string
}

// SuggestionGenerator - Generates personalized suggestions
type SuggestionGenerator struct {
	llm LLMProvider
}

// NewSuggestionGenerator - Create new suggestion generator
func NewSuggestionGenerator(llm LLMProvider) *SuggestionGenerator {
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
		Quality:     0.85,
		GeneratedAt: 0,
		Suggestions: parseSuggestions(resp.Content),
	}

	if len(output.Suggestions) == 0 {
		output.Quality = 0.5
	}

	return output, nil
}

// buildSystemPrompt - Build system prompt for suggestion generation
func (sg *SuggestionGenerator) buildSystemPrompt(input *SuggestionGeneratorInput) string {
	mode := "direct questions"
	if input.Mode == "socratic" {
		mode = "Socratic questions that help them think through the situation"
	}

	values := ""
	if len(input.UserValues) > 0 {
		values = fmt.Sprintf("User values: %s. ", strings.Join(input.UserValues, ", "))
	}

	return fmt.Sprintf(`You are a listener helping someone think through their situation.
Your role is to ask 3-5 clarifying questions that help you understand them better.

%sTheir communication style: %s
Question tone: %s
Mode: %s

Key principles:
- Ask questions that help you understand their needs
- Don't tell them what to do
- Help them think, not give answers
- Keep questions open-ended and genuine
- Each question should reveal something new about what they're trying to figure out

Generate questions to help you listen better and understand what they really need.`,
		values, input.UserCommunicationStyle, input.Tone, mode)
}

// buildUserPrompt - Build user prompt for suggestion generation
func (sg *SuggestionGenerator) buildUserPrompt(input *SuggestionGeneratorInput) string {
	contactInfo := fmt.Sprintf("They're talking about: %s (%s)", input.ContactRelationship, strings.Join(input.ContactCharacteristics, ", "))

	if len(input.ContactInterests) > 0 {
		contactInfo += fmt.Sprintf(" - interested in: %s", strings.Join(input.ContactInterests, ", "))
	}

	history := ""
	if len(input.RecentConversationHistory) > 0 {
		history = fmt.Sprintf("\nContext from our conversation: %s", strings.Join(input.RecentConversationHistory[:minInt(3, len(input.RecentConversationHistory))], " | "))
	}

	intention := ""
	if input.UserIntention != "" {
		intention = fmt.Sprintf("\nWhat they're trying to figure out: %s", input.UserIntention)
	}

	return fmt.Sprintf(`They just said:
"%s"

%s%s%s

What do I need to understand better about their situation?
Generate 3-5 clarifying questions to help me listen better.
Format each as: [Index] "Question?" (Why asking: X) Confidence: Y`,
		input.UserMessage, contactInfo, history, intention)
}

// parseSuggestions - Parse LLM response into structured suggestions
func parseSuggestions(content string) []GeneratedSuggestion {
	if content == "" {
		return []GeneratedSuggestion{}
	}

	var suggestions []GeneratedSuggestion
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		suggestion := parseSingleSuggestion(line)
		if suggestion != nil {
			suggestions = append(suggestions, *suggestion)
		}
	}

	return suggestions
}

// parseSingleSuggestion - Parse a single suggestion line
func parseSingleSuggestion(line string) *GeneratedSuggestion {
	if !strings.Contains(line, "\"") {
		return nil
	}

	suggestion := &GeneratedSuggestion{
		Confidence: 0.75,
	}

	start := strings.Index(line, "\"")
	end := strings.LastIndex(line, "\"")
	if start >= 0 && end > start {
		suggestion.Text = line[start+1 : end]
	}

	if strings.Contains(line, "Tone:") {
		parts := strings.Split(line, "Tone:")
		if len(parts) > 1 {
			tonePart := strings.TrimSpace(parts[1])
			tonePart = strings.Split(tonePart, " ")[0]
			tonePart = strings.TrimSuffix(tonePart, ")")
			suggestion.Tone = tonePart
		}
	}

	if strings.Contains(line, "Confidence:") {
		parts := strings.Split(line, "Confidence:")
		if len(parts) > 1 {
			confStr := strings.TrimSpace(parts[1])
			confStr = strings.Split(confStr, " ")[0]
			var conf float64
			fmt.Sscanf(confStr, "%f", &conf)
			if conf > 0 && conf <= 1 {
				suggestion.Confidence = conf
			}
		}
	}

	if strings.Contains(line, "Reasoning:") {
		parts := strings.Split(line, "Reasoning:")
		if len(parts) > 1 {
			suggestion.Reasoning = strings.TrimSpace(parts[1])
		}
	}

	if suggestion.Text == "" {
		return nil
	}

	return suggestion
}

// minInt - Helper to get minimum of two ints
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
