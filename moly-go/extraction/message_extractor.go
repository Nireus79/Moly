package extraction

import (
	"encoding/json"
	"log"
	"strings"

	"moly/models"
)

// ExtractedContext - Structured extraction from user message
type ExtractedContext struct {
	Contact        string   `json:"contact"`
	Style          string   `json:"style"`
	Intention      string   `json:"intention"`
	Goals          []string `json:"goals"`
	EmotionalTone  string   `json:"emotional_tone"`
	Topic          string   `json:"topic"`
	HasConflicts   bool     `json:"has_conflicts"`
	ConflictFields []string `json:"conflict_fields"`
}

// MessageExtractor - Extracts structured context from messages
type MessageExtractor struct {
}

// NewMessageExtractor - Create new extractor
func NewMessageExtractor() *MessageExtractor {
	return &MessageExtractor{}
}

// Extract - Extract context from user message
// This is a placeholder that uses heuristics
// Real implementation will use LLM (Stage 3 Phase 2)
func (me *MessageExtractor) Extract(msg string, userAboutMe *models.AboutMe) *ExtractedContext {
	log.Printf("[MessageExtractor] Extracting context from message: %d chars", len(msg))

	ctx := &ExtractedContext{
		Contact:       "unknown",
		Style:         detectStyle(msg),
		Intention:     detectIntention(msg),
		EmotionalTone: detectEmotionalTone(msg),
		Topic:         detectTopic(msg),
	}

	log.Printf("[MessageExtractor] Extracted: style=%s emotion=%s topic=%s", ctx.Style, ctx.EmotionalTone, ctx.Topic)
	return ctx
}

// DetectStyle - Heuristically detect communication style
func detectStyle(msg string) string {
	lower := strings.ToLower(msg)

	// Formal indicators
	if containsWord(lower, "formally", "professionally", "business", "formal", "professional") {
		return "formal"
	}

	// Casual indicators
	if containsWord(lower, "casual", "chill", "relaxed", "laid-back", "easy", "fun") {
		return "casual"
	}

	// Playful indicators
	if containsWord(lower, "playful", "joke", "laugh", "funny", "silly") {
		return "playful"
	}

	return "neutral"
}

// DetectIntention - Heuristically detect user's intention
func detectIntention(msg string) string {
	lower := strings.ToLower(msg)

	if containsWord(lower, "help", "how", "advice", "tips", "suggest") {
		return "seek_advice"
	}

	if containsWord(lower, "tell", "message", "send", "say", "respond", "write") {
		return "draft_message"
	}

	if containsWord(lower, "understand", "explain", "why", "reason", "meaning") {
		return "seek_understanding"
	}

	if containsWord(lower, "worried", "anxious", "stressed", "concerned", "scared") {
		return "express_concern"
	}

	if containsWord(lower, "happy", "excited", "great", "wonderful", "good") {
		return "express_joy"
	}

	return "general_conversation"
}

// DetectEmotionalTone - Detect user's emotional state
func detectEmotionalTone(msg string) string {
	lower := strings.ToLower(msg)
	score := 0

	// Very negative indicators
	veryNegWords := []string{"hate", "awful", "terrible", "disaster", "worst", "impossible"}
	if countWords(lower, veryNegWords) >= 2 {
		return "very_negative"
	}

	// Negative indicators
	negWords := []string{"frustrated", "angry", "upset", "sad", "depressed", "hurt", "problem", "issue", "hard", "difficult"}
	negCount := countWords(lower, negWords)
	if negCount >= 2 {
		score -= 2
	} else if negCount >= 1 {
		score -= 1
	}

	// Very positive indicators
	veryPosWords := []string{"love", "amazing", "wonderful", "fantastic", "perfect", "excellent"}
	if countWords(lower, veryPosWords) >= 2 {
		return "very_positive"
	}

	// Positive indicators
	posWords := []string{"good", "great", "nice", "happy", "excited", "looking forward", "appreciate"}
	posCount := countWords(lower, posWords)
	if posCount >= 2 {
		score += 2
	} else if posCount >= 1 {
		score += 1
	}

	// Neutral is default
	if score >= 2 {
		return "positive"
	} else if score <= -2 {
		return "negative"
	}

	return "neutral"
}

// DetectTopic - Detect conversation topic
func detectTopic(msg string) string {
	lower := strings.ToLower(msg)

	if containsWord(lower, "work", "job", "boss", "colleague", "team", "project", "deadline") {
		return "work"
	}

	if containsWord(lower, "relationship", "partner", "date", "romantic", "love", "boyfriend", "girlfriend") {
		return "relationships"
	}

	if containsWord(lower, "family", "parent", "mom", "dad", "sibling", "brother", "sister") {
		return "family"
	}

	if containsWord(lower, "mental", "health", "therapy", "counselor", "anxiety", "depression", "stressed") {
		return "mental_health"
	}

	if containsWord(lower, "money", "financial", "budget", "salary", "expenses", "bills") {
		return "finances"
	}

	return "general"
}

// Helper functions

// containsWord - Check if text contains any of the words
func containsWord(text string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

// countWords - Count how many words are in text
func countWords(text string, words []string) int {
	count := 0
	for _, word := range words {
		if strings.Contains(text, word) {
			count++
		}
	}
	return count
}

// MarshalJSON - Marshal ExtractedContext to JSON
func (ec *ExtractedContext) MarshalJSON() ([]byte, error) {
	type Alias ExtractedContext
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(ec),
	})
}
