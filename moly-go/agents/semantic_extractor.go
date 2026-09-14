package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/tools"
)

// ExtractedFact represents a fact extracted from user message
type ExtractedFact struct {
	ID              string  `json:"id"`
	Value           string  `json:"value"`           // "casual", "detail-oriented"
	FactType        string  `json:"factType"`        // "trait", "style", "value", "preference", "goal"
	ProposedSubject string  `json:"proposedSubject"` // "user", "she", "boss", "they"
	Confidence      float64 `json:"confidence"`      // 0-1
	Evidence        string  `json:"evidence"`        // Exact quote from message
	CreatedAt       int64   `json:"createdAt"`
}

// SemanticExtractor uses LLM to extract facts from messages
type SemanticExtractor struct {
	llm tools.LLMProvider
}

// NewSemanticExtractor creates a new semantic extractor
func NewSemanticExtractor(llm tools.LLMProvider) *SemanticExtractor {
	return &SemanticExtractor{llm: llm}
}

// Extract analyzes message and returns extracted facts
func (e *SemanticExtractor) Extract(message string) ([]ExtractedFact, error) {
	if message == "" {
		return []ExtractedFact{}, nil
	}

	// Filter out pure greetings and very short messages without substance
	trimmed := strings.TrimSpace(message)
	if len(trimmed) < 10 || isGreeting(trimmed) {
		log.Printf("[V2] SemanticExtractor.Extract: skipping greeting/empty message: %.60s", trimmed)
		return []ExtractedFact{}, nil
	}

	log.Printf("[V2] SemanticExtractor.Extract: analyzing message (%.60s...)", message)

	// Use LLM to extract facts
	prompt := fmt.Sprintf(`Analyze this message and extract factual statements about people, their traits, styles, values, and goals.

For each fact, identify:
1. Type: one of "trait" (personal characteristic), "style" (communication/work style), "value" (principle), "preference" (like/dislike), or "goal" (wants/needs)
2. Value: what's being stated (keep concise, 1-3 words)
3. Subject: who/what it's about - will be one of: "user" (speaker), "she", "he", "they", "boss", "manager", "colleague", "friend", "partner", "family", or specific name
4. Quote: exact text from the message supporting this fact
5. Confidence: 0-1 score for how certain this is

Rules:
- Only extract EXPLICIT statements (no inference)
- If pronouns used (she, he, they), include them as subject
- Be conservative - better to miss a fact than invent one
- Ignore greetings and general chat

Message: "%s"

Return ONLY valid JSON array like:
[
  {"type":"style", "value":"casual", "subject":"user", "quote":"I'm casual", "confidence":0.95},
  {"type":"trait", "value":"detail-oriented", "subject":"she", "quote":"boss is very detail-oriented", "confidence":0.9}
]

If no facts found, return empty array: []
`, message)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	req := &tools.LLMRequest{
		SystemPrompt: "You are a semantic analyzer. Extract factual statements from messages. Return ONLY valid JSON, no markdown, no explanation.",
		UserPrompt:   prompt,
		MaxTokens:    1000,
		Temperature:  0.3, // Low temperature for consistency
	}

	resp, err := e.llm.Call(ctx, req)
	if err != nil {
		log.Printf("[V2] SemanticExtractor.Extract ERROR: %v", err)
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	log.Printf("[V2] SemanticExtractor.Extract: LLM response: %s", resp.Content)

	// Parse JSON response
	var rawFacts []map[string]interface{}
	err = json.Unmarshal([]byte(resp.Content), &rawFacts)
	if err != nil {
		log.Printf("[V2] SemanticExtractor.Extract WARNING: Failed to parse LLM response as JSON: %v. Response: %s", err, resp.Content)
		return []ExtractedFact{}, nil // Return empty list on parse error
	}

	// Convert to ExtractedFact structs
	facts := make([]ExtractedFact, len(rawFacts))
	for i, raw := range rawFacts {
		conf := 0.8
		if c, ok := raw["confidence"].(float64); ok {
			conf = c
		}

		facts[i] = ExtractedFact{
			ID:              fmt.Sprintf("fact_%d_%d", time.Now().Unix(), i),
			Value:           toString(raw["value"]),
			FactType:        toString(raw["type"]),
			ProposedSubject: toString(raw["subject"]),
			Confidence:      conf,
			Evidence:        toString(raw["quote"]),
			CreatedAt:       time.Now().Unix(),
		}
	}

	log.Printf("[V2] SemanticExtractor.Extract: extracted %d facts", len(facts))
	for i, f := range facts {
		log.Printf("[V2]   Fact %d: type=%s, value=%s, subject=%s, confidence=%.2f", i, f.FactType, f.Value, f.ProposedSubject, f.Confidence)
	}

	return facts, nil
}

// toString safely converts interface{} to string
func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// isGreeting checks if a message is just a greeting with no substance
func isGreeting(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	greetings := []string{
		"hello", "hi", "hey", "greetings", "good morning", "good afternoon", "good evening",
		"hello moly", "hi moly", "hey moly", "moly",
		"what's up", "how's it going", "how are you", "sup", "yo",
	}

	for _, g := range greetings {
		if strings.EqualFold(lower, g) {
			return true
		}
	}

	return false
}
