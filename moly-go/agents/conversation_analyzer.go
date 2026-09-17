package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/database"
	"moly/tools"
)

// ConversationAnalyzer extracts structured insights from conversations
// Runs once per conversation to update user's profile/notes
type ConversationAnalyzer struct {
	llmClient tools.LLMProvider
	db        *database.Database
}

// ExtractionResult contains all insights extracted from a conversation
type ExtractionResult struct {
	AboutMeUpdates      []AboutMeUpdate      `json:"aboutMeUpdates"`
	PatternDetections   []PatternDetection   `json:"patternDetections"`
	ContactMentions     []ContactMention     `json:"contactMentions"`
	GoalProgressUpdates []GoalProgressUpdate `json:"goalProgressUpdates"`
	ConfidenceScore     float64              `json:"confidence"`
	ExtractedAt         int64                `json:"extractedAt"`
	ErrorMessage        string               `json:"error,omitempty"`
}

// AboutMeUpdate represents a single insight about user's communication
type AboutMeUpdate struct {
	Key        string  `json:"key"` // "communication_style", "value", "preference", "tone_preference"
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"` // 0-1
	Source     string  `json:"source"`     // what led to this extraction
	Reasoning  string  `json:"reasoning"`  // why system thinks this
}

// PatternDetection represents an observed communication pattern
type PatternDetection struct {
	Pattern      string  `json:"pattern"`  // "avoids_conflict_then_over_explains"
	Category     string  `json:"category"` // "avoidance", "assertiveness", "clarity"
	Confidence   float64 `json:"confidence"`
	Evidence     string  `json:"evidence"`     // specific example from conversation
	IsGrowthArea bool    `json:"isGrowthArea"` // is user working on this?
}

// ContactMention represents a person mentioned in the conversation
type ContactMention struct {
	Name             string   `json:"name"`
	RelationshipType string   `json:"relationshipType"` // "professional", "family", "romantic", "friendship"
	ToneObserved     string   `json:"toneObserved"`     // "warm", "formal", "tense", "fearful", etc.
	Context          string   `json:"context"`          // "My boss", "My mother", etc.
	MainTopics       []string `json:"mainTopics"`       // what they discuss
	Frequency        string   `json:"frequency"`        // "daily", "weekly", "monthly", "rare"
	Confidence       float64  `json:"confidence"`
}

// GoalProgressUpdate represents progress on active communication goals
type GoalProgressUpdate struct {
	GoalDescription string  `json:"goalDescription"` // what goal is being worked on
	Progress        string  `json:"progress"`        // "in_progress", "made_progress", "struggling"
	Evidence        string  `json:"evidence"`        // specific example
	Confidence      float64 `json:"confidence"`
}

// Message represents a single message in a conversation
type Message struct {
	Role      string `json:"role"` // "user" or "assistant"
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// NewConversationAnalyzer creates a new analyzer
func NewConversationAnalyzer(llmClient tools.LLMProvider, db *database.Database) *ConversationAnalyzer {
	return &ConversationAnalyzer{
		llmClient: llmClient,
		db:        db,
	}
}

// AnalyzeConversation extracts structured insights from a completed conversation
// This is the main entry point for the analyzer
func (ca *ConversationAnalyzer) AnalyzeConversation(
	ctx context.Context,
	userID string,
	conversationID string,
	messages []Message,
) (*ExtractionResult, error) {
	if len(messages) == 0 {
		return &ExtractionResult{
			ConfidenceScore: 0,
			ExtractedAt:     time.Now().Unix(),
			ErrorMessage:    "no messages to analyze",
		}, nil
	}

	log.Printf("[ConversationAnalyzer] Analyzing %d messages from conversation %s", len(messages), conversationID)

	// Build the conversation text for LLM analysis
	conversationText := ca.buildConversationText(messages)

	// Extract insights using LLM
	result, err := ca.extractInsights(ctx, userID, conversationText)
	if err != nil {
		log.Printf("[ConversationAnalyzer] Extraction failed: %v", err)
		return &ExtractionResult{
			ConfidenceScore: 0,
			ExtractedAt:     time.Now().Unix(),
			ErrorMessage:    fmt.Sprintf("extraction failed: %v", err),
		}, err
	}

	result.ExtractedAt = time.Now().Unix()

	log.Printf("[ConversationAnalyzer] Extraction complete: %d AboutMe, %d patterns, %d contacts, %d goal updates, confidence=%.2f",
		len(result.AboutMeUpdates),
		len(result.PatternDetections),
		len(result.ContactMentions),
		len(result.GoalProgressUpdates),
		result.ConfidenceScore)

	return result, nil
}

// buildConversationText formats messages into readable text for LLM
func (ca *ConversationAnalyzer) buildConversationText(messages []Message) string {
	var sb strings.Builder
	sb.WriteString("Conversation transcript:\n\n")

	for _, msg := range messages {
		role := "User"
		if msg.Role == "assistant" {
			role = "Assistant"
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}

	return sb.String()
}

// extractInsights calls LLM to extract structured insights
func (ca *ConversationAnalyzer) extractInsights(
	ctx context.Context,
	userID string,
	conversationText string,
) (*ExtractionResult, error) {
	// Build the extraction prompt
	prompt := ca.buildExtractionPrompt(conversationText)

	// Call LLM
	req := &tools.LLMRequest{
		SystemPrompt: systemPromptExtraction,
		UserPrompt:   prompt,
		Temperature:  0.3, // Lower temperature for structured extraction
		MaxTokens:    2000,
	}

	resp, err := ca.llmClient.Call(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse JSON response
	var result ExtractionResult
	err = json.Unmarshal([]byte(resp.Content), &result)
	if err != nil {
		log.Printf("[ConversationAnalyzer] Failed to parse LLM response: %v\nResponse: %s", err, resp.Content)
		return nil, fmt.Errorf("failed to parse extraction result: %w", err)
	}

	// Validate and clean results
	ca.validateResults(&result)

	return &result, nil
}

// buildExtractionPrompt creates the LLM prompt for extraction
func (ca *ConversationAnalyzer) buildExtractionPrompt(conversationText string) string {
	return fmt.Sprintf(`Extract what you learned about how this person communicates:

%s

Return ONLY this JSON (no explanation, empty arrays if nothing found):
{
  "aboutMeUpdates": [{"key":"communication_style|tone|value|preference", "value":"", "confidence":0.0, "source":"", "reasoning":""}],
  "patternDetections": [{"pattern":"", "category":"avoidance|assertiveness|clarity|listening|vulnerability|boundaries", "confidence":0.0, "evidence":"", "isGrowthArea":false}],
  "contactMentions": [{"name":"", "relationshipType":"professional|family|romantic|friendship", "toneObserved":"warm|formal|tense|fearful|relaxed|casual", "context":"", "mainTopics":[], "frequency":"daily|weekly|monthly|rare", "confidence":0.0}],
  "goalProgressUpdates": [{"goalDescription":"", "progress":"in_progress|made_progress|struggling", "evidence":"", "confidence":0.0}],
  "confidence":0.0
}`, conversationText)
}

// validateResults cleans and validates extraction results
func (ca *ConversationAnalyzer) validateResults(result *ExtractionResult) {
	// Remove low-confidence items
	result.AboutMeUpdates = ca.filterAboutMeUpdates(result.AboutMeUpdates)
	result.PatternDetections = ca.filterPatternDetections(result.PatternDetections)
	result.ContactMentions = ca.filterContactMentions(result.ContactMentions)
	result.GoalProgressUpdates = ca.filterGoalProgressUpdates(result.GoalProgressUpdates)

	// Normalize confidence score
	if result.ConfidenceScore < 0 {
		result.ConfidenceScore = 0
	}
	if result.ConfidenceScore > 1 {
		result.ConfidenceScore = 1
	}

	// If no extractions, lower overall confidence
	totalExtractions := len(result.AboutMeUpdates) +
		len(result.PatternDetections) +
		len(result.ContactMentions) +
		len(result.GoalProgressUpdates)

	if totalExtractions == 0 {
		result.ConfidenceScore = 0.3
	}
}

// filterAboutMeUpdates removes low-confidence AboutMe items
func (ca *ConversationAnalyzer) filterAboutMeUpdates(updates []AboutMeUpdate) []AboutMeUpdate {
	var filtered []AboutMeUpdate
	for _, u := range updates {
		if u.Confidence >= 0.5 {
			// Clamp confidence to 0-1
			if u.Confidence > 1 {
				u.Confidence = 1
			}
			filtered = append(filtered, u)
		}
	}
	return filtered
}

// filterPatternDetections removes low-confidence patterns
func (ca *ConversationAnalyzer) filterPatternDetections(patterns []PatternDetection) []PatternDetection {
	var filtered []PatternDetection
	for _, p := range patterns {
		if p.Confidence >= 0.5 {
			if p.Confidence > 1 {
				p.Confidence = 1
			}
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// filterContactMentions removes low-confidence contacts
func (ca *ConversationAnalyzer) filterContactMentions(contacts []ContactMention) []ContactMention {
	var filtered []ContactMention
	for _, c := range contacts {
		if c.Confidence >= 0.5 {
			if c.Confidence > 1 {
				c.Confidence = 1
			}
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// filterGoalProgressUpdates removes low-confidence goal updates
func (ca *ConversationAnalyzer) filterGoalProgressUpdates(goals []GoalProgressUpdate) []GoalProgressUpdate {
	var filtered []GoalProgressUpdate
	for _, g := range goals {
		if g.Confidence >= 0.5 {
			if g.Confidence > 1 {
				g.Confidence = 1
			}
			filtered = append(filtered, g)
		}
	}
	return filtered
}

// systemPromptExtraction is the system prompt for extraction
const systemPromptExtraction = `Extract meaningful insights about how this person communicates to help them reflect and grow.

What to extract:
1. Communication style - How they naturally speak (direct, gentle, thoughtful, etc.)
2. Core values - What matters to them (authenticity, honesty, growth, respect, etc.)
3. Patterns - Recurring behaviors and tendencies
   - Positive patterns they're building (becoming more direct, setting boundaries)
   - Patterns they're struggling with (avoiding conflict, over-explaining)
   - Growth areas they're working on
4. People they mention - Who they interact with and how (tone, relationship type, topics)
5. What they're working on - Goals and growth they're actively pursuing

Rules:
- Only include if confidence >= 0.5
- Quote exact words when possible
- Show the evidence from conversation
- No inferring beyond what they explicitly said
- Focus on communication patterns, not judging character
- Highlight growth and self-awareness patterns
- Return JSON only, no explanation

Example: "I'm trying to be more direct with my mom, but I usually just go along"
- Style: "tends toward accommodation" (confidence 0.8)
- Value: "honesty/directness" (confidence 0.9)
- Pattern (struggle): "avoids conflict, then complies" (confidence 0.85, isGrowthArea: true)
- Pattern (growth): "actively working on assertiveness" (confidence 0.9)
- Contact: mom, family, warm but tense tone (confidence 0.9)
- Goal: "be more direct with family" (confidence 0.9)`

// ToJSON converts result to JSON for storage
func (er *ExtractionResult) ToJSON() (string, error) {
	b, err := json.Marshal(er)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FromJSON parses JSON into ExtractionResult
func ExtractionResultFromJSON(jsonStr string) (*ExtractionResult, error) {
	var result ExtractionResult
	err := json.Unmarshal([]byte(jsonStr), &result)
	return &result, err
}
