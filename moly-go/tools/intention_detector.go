package tools

import (
	"context"
	"fmt"
	"strings"
)

// IntentionType - Types of user intentions
type IntentionType string

const (
	IntentionCelebrate IntentionType = "celebrate"
	IntentionApologize IntentionType = "apologize"
	IntentionSeekHelp  IntentionType = "seek_help"
	IntentionGreet     IntentionType = "greet"
	IntentionAdvice    IntentionType = "ask_advice"
	IntentionConfess   IntentionType = "confess"
	IntentionReassure  IntentionType = "reassure"
	IntentionBoundary  IntentionType = "set_boundary"
	IntentionConfront  IntentionType = "confront"
	IntentionHeal      IntentionType = "heal_relationship"
	IntentionGeneral   IntentionType = "general_support"
)

// IntentionDetectorInput - Input for intention detection
type IntentionDetectorInput struct {
	Message                string
	ConversationHistory    []string
	UserCommunicationStyle string
	ContactRelationship    string
}

// IntentionDetectorOutput - Detected intention with confidence
type IntentionDetectorOutput struct {
	PrimaryIntention   IntentionType
	SecondaryIntention IntentionType
	Confidence         float64 // 0.0-1.0
	Indicators         []string
	Tone               string
	EmotionalContent   string
}

// IntentionDetector - Detects user intention from messages
type IntentionDetector struct {
	llm LLMProvider
}

// NewIntentionDetector - Create new intention detector
func NewIntentionDetector(llm LLMProvider) *IntentionDetector {
	return &IntentionDetector{
		llm: llm,
	}
}

// Detect - Detect intention from message
func (id *IntentionDetector) Detect(ctx context.Context, input *IntentionDetectorInput) (*IntentionDetectorOutput, error) {
	if input == nil || input.Message == "" {
		return nil, fmt.Errorf("input and message required")
	}

	// Try LLM-based detection first if available
	if id.llm != nil {
		output, err := id.detectLLM(ctx, input)
		if err == nil && output.Confidence > 0.7 {
			return output, nil
		}
	}

	// Fallback to heuristic detection
	return id.detectHeuristic(input), nil
}

// detectLLM - Use LLM for intelligent intention detection
func (id *IntentionDetector) detectLLM(ctx context.Context, input *IntentionDetectorInput) (*IntentionDetectorOutput, error) {
	systemPrompt := `You are an expert at understanding user intentions in conversations.
Analyze the message and determine:
1. Primary intention (what they mainly want to achieve)
2. Secondary intention (if there's a secondary goal)
3. Emotional tone (angry, sad, happy, neutral, etc.)
4. Confidence (0.0-1.0) in your detection

Possible intentions: celebrate, apologize, seek_help, greet, ask_advice, confess, reassure, set_boundary, confront, heal_relationship, general_support

Respond with: intention: <type>, secondary: <type>, confidence: <number>, tone: <emotion>, indicators: [...]`

	userPrompt := fmt.Sprintf(`Analyze this message for intention:
"%s"

Context:
- Communication style: %s
- Relationship: %s
- Previous messages: %v`,
		input.Message,
		input.UserCommunicationStyle,
		input.ContactRelationship,
		input.ConversationHistory)

	req := &LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		MaxTokens:    300,
		Temperature:  0.4,
		Retries:      1,
	}

	resp, err := id.llm.Call(ctx, req)
	if err != nil {
		return nil, err
	}

	return parseIntentionResponse(resp.Content), nil
}

// detectHeuristic - Fast heuristic-based intention detection
func (id *IntentionDetector) detectHeuristic(input *IntentionDetectorInput) *IntentionDetectorOutput {
	message := strings.ToLower(input.Message)
	output := &IntentionDetectorOutput{
		PrimaryIntention:   IntentionGeneral,
		SecondaryIntention: "",
		Confidence:         0.5,
		Indicators:         []string{},
		Tone:               "neutral",
		EmotionalContent:   "low",
	}

	// Check for celebration
	celebrateWords := []string{"congratulat", "congrats", "awesome", "amazing", "wonderful", "great", "excellent", "proud", "excited", "thrilled"}
	for _, word := range celebrateWords {
		if strings.Contains(message, word) {
			output.PrimaryIntention = IntentionCelebrate
			output.Confidence = 0.85
			output.Tone = "happy"
			output.EmotionalContent = "high_positive"
			output.Indicators = append(output.Indicators, word)
			return output
		}
	}

	// Check for apology
	apologyWords := []string{"sorry", "apologize", "apologetic", "regret", "regrettable", "my bad", "my fault", "i messed up"}
	for _, word := range apologyWords {
		if strings.Contains(message, word) {
			output.PrimaryIntention = IntentionApologize
			output.Confidence = 0.9
			output.Tone = "sad"
			output.EmotionalContent = "high_negative"
			output.Indicators = append(output.Indicators, word)
			return output
		}
	}

	// Check for help-seeking
	helpWords := []string{"help", "stuck", "need advice", "confused", "lost", "don't know", "struggling", "difficult", "problem", "issue"}
	for _, word := range helpWords {
		if strings.Contains(message, word) {
			output.PrimaryIntention = IntentionSeekHelp
			output.Confidence = 0.8
			output.Tone = "uncertain"
			output.EmotionalContent = "medium_negative"
			output.Indicators = append(output.Indicators, word)
			return output
		}
	}

	// Check for greeting
	greetWords := []string{"hi", "hello", "hey", "how are you", "what's up", "good morning", "good evening"}
	for _, word := range greetWords {
		if strings.Contains(message, word) {
			output.PrimaryIntention = IntentionGreet
			output.Confidence = 0.9
			output.Tone = "friendly"
			output.EmotionalContent = "low"
			output.Indicators = append(output.Indicators, word)
			return output
		}
	}

	// Check for advice-seeking
	adviceWords := []string{"advice", "recommend", "suggest", "opinion", "thoughts", "think", "what should", "what would you"}
	for _, word := range adviceWords {
		if strings.Contains(message, word) {
			output.PrimaryIntention = IntentionAdvice
			output.Confidence = 0.75
			output.Tone = "questioning"
			output.EmotionalContent = "low"
			output.Indicators = append(output.Indicators, word)
			return output
		}
	}

	// Check for confession
	confessWords := []string{"confess", "admit", "truth", "honestly", "real talk", "be honest", "i need to tell"}
	for _, word := range confessWords {
		if strings.Contains(message, word) {
			output.PrimaryIntention = IntentionConfess
			output.Confidence = 0.8
			output.Tone = "vulnerable"
			output.EmotionalContent = "high_negative"
			output.Indicators = append(output.Indicators, word)
			return output
		}
	}

	// Check for confrontation
	confrontWords := []string{"confront", "angry", "mad", "upset", "furious", "tired of", "fed up", "not okay"}
	for _, word := range confrontWords {
		if strings.Contains(message, word) {
			output.PrimaryIntention = IntentionConfront
			output.Confidence = 0.85
			output.Tone = "angry"
			output.EmotionalContent = "high_negative"
			output.Indicators = append(output.Indicators, word)
			return output
		}
	}

	// Check for boundary-setting
	boundaryWords := []string{"boundary", "no", "can't", "won't", "stop", "need to end", "need space", "not happening"}
	for _, word := range boundaryWords {
		if strings.Contains(message, word) {
			output.PrimaryIntention = IntentionBoundary
			output.Confidence = 0.8
			output.Tone = "firm"
			output.EmotionalContent = "medium_negative"
			output.Indicators = append(output.Indicators, word)
			return output
		}
	}

	// Default to general support
	output.Confidence = 0.4
	return output
}

// parseIntentionResponse - Parse LLM response into structured output
func parseIntentionResponse(content string) *IntentionDetectorOutput {
	output := &IntentionDetectorOutput{
		PrimaryIntention:   IntentionGeneral,
		SecondaryIntention: "",
		Confidence:         0.5,
		Tone:               "neutral",
		EmotionalContent:   "low",
		Indicators:         []string{},
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), "intention:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				intentStr := strings.TrimSpace(parts[1])
				intentStr = strings.Split(intentStr, ",")[0]
				output.PrimaryIntention = IntentionType(intentStr)
			}
		} else if strings.Contains(strings.ToLower(line), "secondary:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				intentStr := strings.TrimSpace(parts[1])
				intentStr = strings.Split(intentStr, ",")[0]
				output.SecondaryIntention = IntentionType(intentStr)
			}
		} else if strings.Contains(strings.ToLower(line), "confidence:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				confStr := strings.TrimSpace(parts[1])
				confStr = strings.Split(confStr, ",")[0]
				fmt.Sscanf(confStr, "%f", &output.Confidence)
			}
		} else if strings.Contains(strings.ToLower(line), "tone:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				toneStr := strings.TrimSpace(parts[1])
				toneStr = strings.Split(toneStr, ",")[0]
				output.Tone = toneStr
			}
		}
	}

	return output
}

// GetIntentionDescription - Get human-readable description of intention
func GetIntentionDescription(intention IntentionType) string {
	descriptions := map[IntentionType]string{
		IntentionCelebrate: "Celebrating an achievement or good news",
		IntentionApologize: "Expressing regret and seeking forgiveness",
		IntentionSeekHelp:  "Asking for help or advice",
		IntentionGreet:     "Starting a conversation",
		IntentionAdvice:    "Asking for someone's opinion or recommendation",
		IntentionConfess:   "Revealing something vulnerable",
		IntentionReassure:  "Offering comfort and support",
		IntentionBoundary:  "Setting or enforcing a boundary",
		IntentionConfront:  "Addressing a problem directly",
		IntentionHeal:      "Working to repair a relationship",
		IntentionGeneral:   "General support or communication",
	}

	if desc, ok := descriptions[intention]; ok {
		return desc
	}
	return "Unclear intention"
}

// SuggestedTone - Get suggested communication tone for intention
func SuggestedTone(intention IntentionType) string {
	tones := map[IntentionType]string{
		IntentionCelebrate: "enthusiastic",
		IntentionApologize: "sincere",
		IntentionSeekHelp:  "vulnerable",
		IntentionGreet:     "warm",
		IntentionAdvice:    "thoughtful",
		IntentionConfess:   "honest",
		IntentionReassure:  "compassionate",
		IntentionBoundary:  "firm",
		IntentionConfront:  "direct",
		IntentionHeal:      "understanding",
		IntentionGeneral:   "friendly",
	}

	if tone, ok := tones[intention]; ok {
		return tone
	}
	return "appropriate"
}
