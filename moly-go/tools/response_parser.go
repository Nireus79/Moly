package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"moly/models"
)

// ResponseParserInput - Input for parsing user responses
type ResponseParserInput struct {
	UserMessage string
	Context     string // What we were asking about
	UserID      string
}

// ResponseParserOutput - Parsed information from user response
type ResponseParserOutput struct {
	ExtractedAboutMe   *models.AboutMe
	ExtractedContact   *models.Contact
	ExtractedIntention string
	Confidence         float64
	RawContent         string
	ParsedSuccessfully bool
}

// ResponseParser - Parses user responses to extract context
type ResponseParser struct {
	llm LLMProvider
}

// NewResponseParser - Create new response parser
func NewResponseParser(llm LLMProvider) *ResponseParser {
	return &ResponseParser{
		llm: llm,
	}
}

// Parse - Parse user response and extract information
func (rp *ResponseParser) Parse(ctx context.Context, input *ResponseParserInput) (*ResponseParserOutput, error) {
	if input == nil {
		return nil, errors.New("input cannot be nil")
	}

	if input.UserMessage == "" {
		return nil, errors.New("user message cannot be empty")
	}

	// If no LLM, use heuristic parsing
	if rp.llm == nil {
		return rp.parseHeuristic(input), nil
	}

	// Use LLM for intelligent parsing
	return rp.parseLLM(ctx, input)
}

// parseHeuristic - Basic heuristic-based response parsing
func (rp *ResponseParser) parseHeuristic(input *ResponseParserInput) *ResponseParserOutput {
	output := &ResponseParserOutput{
		RawContent:         input.UserMessage,
		ParsedSuccessfully: false,
		Confidence:         0.3,
	}

	message := strings.ToLower(input.UserMessage)

	// Parse based on context (what we were asking about)
	contextLower := strings.ToLower(input.Context)

	// If we asked about communication style
	if strings.Contains(contextLower, "communication") || strings.Contains(contextLower, "style") {
		output.ExtractedAboutMe = rp.extractAboutMeFromStyle(input.UserMessage)
		if output.ExtractedAboutMe != nil {
			output.Confidence = 0.6
			output.ParsedSuccessfully = true
		}
	}

	// If we asked about a contact
	if strings.Contains(contextLower, "contact") || strings.Contains(contextLower, "person") || strings.Contains(contextLower, "who") {
		output.ExtractedContact = rp.extractContactInfo(input.UserMessage)
		if output.ExtractedContact != nil {
			output.Confidence = 0.7
			output.ParsedSuccessfully = true
		}
	}

	// Extract intention from message
	if strings.Contains(message, "congratulat") || strings.Contains(message, "celebrate") {
		output.ExtractedIntention = "celebrate"
		output.ParsedSuccessfully = true
	} else if strings.Contains(message, "sorry") || strings.Contains(message, "apologi") {
		output.ExtractedIntention = "apologize"
		output.ParsedSuccessfully = true
	} else if strings.Contains(message, "help") || strings.Contains(message, "need") || strings.Contains(message, "stuck") {
		output.ExtractedIntention = "seek_help"
		output.ParsedSuccessfully = true
	} else if strings.Contains(message, "hi") || strings.Contains(message, "hello") || strings.Contains(message, "hey") {
		output.ExtractedIntention = "greet"
		output.ParsedSuccessfully = true
	}

	return output
}

// parseLLM - Use LLM for intelligent response parsing
func (rp *ResponseParser) parseLLM(ctx context.Context, input *ResponseParserInput) (*ResponseParserOutput, error) {
	output := &ResponseParserOutput{
		RawContent:         input.UserMessage,
		ParsedSuccessfully: false,
		Confidence:         0.0,
	}

	req := &LLMRequest{
		SystemPrompt: `You are an expert at extracting user information from natural language responses.
Parse the user's message and extract:
1. Communication style/preferences (formal, casual, playful, etc)
2. Contact information (name, relationship, personality traits)
3. User's intention (celebrate, apologize, seek_help, greet, general)

Respond with JSON:
{
  "aboutMe": {"communicationStyle":"...", "values":[], "preferredTone":"..."},
  "contact": {"name":"...", "relationship":"...", "characteristics":[]},
  "intention":"...",
  "confidence":0.0-1.0
}

If you can't extract something, omit it from JSON.`,
		UserPrompt:  fmt.Sprintf("Parse this user response: \"%s\"\n\nContext: We were asking about %s", input.UserMessage, input.Context),
		MaxTokens:   500,
		Temperature: 0.3,
		Retries:     1,
	}

	resp, err := rp.llm.Call(ctx, req)
	if err != nil {
		// Fall back to heuristic
		return rp.parseHeuristic(input), nil
	}

	// Parse JSON response
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Content), &parsed); err != nil {
		// Fall back to heuristic if JSON parsing fails
		return rp.parseHeuristic(input), nil
	}

	// Extract AboutMe
	if aboutMeData, ok := parsed["aboutMe"].(map[string]interface{}); ok {
		output.ExtractedAboutMe = rp.mapToAboutMe(aboutMeData)
	}

	// Extract Contact
	if contactData, ok := parsed["contact"].(map[string]interface{}); ok {
		output.ExtractedContact = rp.mapToContact(contactData, input.UserID)
	}

	// Extract Intention
	if intention, ok := parsed["intention"].(string); ok {
		output.ExtractedIntention = intention
	}

	// Extract Confidence
	if conf, ok := parsed["confidence"].(float64); ok {
		output.Confidence = conf
	}

	output.ParsedSuccessfully = output.ExtractedAboutMe != nil || output.ExtractedContact != nil || output.ExtractedIntention != ""

	return output, nil
}

// extractAboutMeFromStyle - Extract AboutMe from style description
func (rp *ResponseParser) extractAboutMeFromStyle(message string) *models.AboutMe {
	lower := strings.ToLower(message)

	style := "friendly"
	if strings.Contains(lower, "formal") {
		style = "formal"
	} else if strings.Contains(lower, "casual") {
		style = "casual"
	} else if strings.Contains(lower, "playful") || strings.Contains(lower, "fun") {
		style = "playful"
	} else if strings.Contains(lower, "direct") {
		style = "direct"
	} else if strings.Contains(lower, "warm") {
		style = "warm"
	}

	return &models.AboutMe{
		CommunicationStyle: style,
		PreferredTone:      style,
		Notes:              message,
	}
}

// extractContactInfo - Extract contact information from message
func (rp *ResponseParser) extractContactInfo(message string) *models.Contact {
	words := strings.Fields(message)
	if len(words) == 0 {
		return nil
	}

	// Simple heuristic: first capitalized word is likely a name
	var name string
	for _, word := range words {
		if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' {
			name = trimPunctuation(word)
			break
		}
	}

	if name == "" {
		return nil
	}

	// Guess relationship
	relationship := "friend"
	lower := strings.ToLower(message)
	if strings.Contains(lower, "boss") || strings.Contains(lower, "manager") {
		relationship = "professional"
	} else if strings.Contains(lower, "parent") || strings.Contains(lower, "mom") || strings.Contains(lower, "dad") {
		relationship = "family"
	} else if strings.Contains(lower, "partner") || strings.Contains(lower, "spouse") {
		relationship = "romantic"
	} else if strings.Contains(lower, "mentor") || strings.Contains(lower, "coach") {
		relationship = "mentor"
	}

	return &models.Contact{
		Name:         name,
		Relationship: relationship,
		Notes:        message,
	}
}

// mapToAboutMe - Convert map to AboutMe model
func (rp *ResponseParser) mapToAboutMe(data map[string]interface{}) *models.AboutMe {
	aboutMe := &models.AboutMe{}

	if style, ok := data["communicationStyle"].(string); ok {
		aboutMe.CommunicationStyle = style
	}

	if tone, ok := data["preferredTone"].(string); ok {
		aboutMe.PreferredTone = tone
	}

	if values, ok := data["values"].([]interface{}); ok {
		for _, v := range values {
			if str, ok := v.(string); ok {
				aboutMe.Values = append(aboutMe.Values, str)
			}
		}
	}

	if aboutMe.CommunicationStyle == "" && aboutMe.PreferredTone == "" && len(aboutMe.Values) == 0 {
		return nil
	}

	return aboutMe
}

// mapToContact - Convert map to Contact model
func (rp *ResponseParser) mapToContact(data map[string]interface{}, userID string) *models.Contact {
	contact := &models.Contact{UserID: userID}

	if name, ok := data["name"].(string); ok {
		contact.Name = name
	}

	if rel, ok := data["relationship"].(string); ok {
		contact.Relationship = rel
	}

	if chars, ok := data["characteristics"].([]interface{}); ok {
		for _, c := range chars {
			if str, ok := c.(string); ok {
				contact.Characteristics = append(contact.Characteristics, str)
			}
		}
	}

	if interest, ok := data["interests"].([]interface{}); ok {
		for _, i := range interest {
			if str, ok := i.(string); ok {
				contact.Interests = append(contact.Interests, str)
			}
		}
	}

	if contact.Name == "" {
		return nil
	}

	return contact
}

// ExtractQuickInfo - Quick extraction without full parsing
func (rp *ResponseParser) ExtractQuickInfo(message string) *ResponseParserOutput {
	return rp.parseHeuristic(&ResponseParserInput{
		UserMessage: message,
		Context:     "general",
	})
}

// trimPunctuation - Remove leading/trailing punctuation
func trimPunctuation(s string) string {
	if len(s) == 0 {
		return s
	}

	start := 0
	end := len(s)

	for start < end && isPunctuation(rune(s[start])) {
		start++
	}

	for end > start && isPunctuation(rune(s[end-1])) {
		end--
	}

	return s[start:end]
}

// isPunctuation - Check if rune is punctuation
func isPunctuation(r rune) bool {
	return (r >= '!' && r <= '/') || (r >= ':' && r <= '@') || (r >= '[' && r <= '`') || (r >= '{' && r <= '~')
}
