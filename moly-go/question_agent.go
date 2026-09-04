package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type QuestionAgent struct {
	db       *Database
	config   *Config
	provider string
}

type ContextAnalysis struct {
	UnderstandingLevel int      `json:"understanding_level"`
	KnownFacts         []string `json:"known_facts"`
	Questions          []string `json:"questions"`
	Reasoning          string   `json:"reasoning"`
}

type InsightExtraction struct {
	ToneDetected      string   `json:"tone_detected"`
	Topics            []string `json:"topics"`
	RelationshipHints string   `json:"relationship_hints"`
	CommunicationStyle string  `json:"communication_style"`
}

func NewQuestionAgent(db *Database, config *Config, provider string) *QuestionAgent {
	return &QuestionAgent{
		db:       db,
		config:   config,
		provider: provider,
	}
}

func (qa *QuestionAgent) AnalyzeContext(contactID int, platform, messageStart string) (*ContextAnalysis, error) {
	// Get contact info
	contact, err := qa.db.getContact(contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %v", err)
	}

	// Get recent interactions
	interactions, err := qa.db.getRecentInteractions(contactID, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get interactions: %v", err)
	}

	// Get behavior patterns
	patterns, err := qa.db.getBehaviorPattern()
	if err != nil {
		return nil, fmt.Errorf("failed to get behavior patterns: %v", err)
	}

	knownFacts := qa.extractKnownFacts(contact, patterns, interactions)

	// Determine if this is a new contact (no interactions)
	isNewContact := len(interactions) == 0 && contact.InteractionCount == 0

	questionCount := 3
	if isNewContact {
		questionCount = 5 // More questions for new contacts
	}

	// Ask LLM what questions we need
	questionContext := ""
	if isNewContact {
		questionContext = `This is a NEW contact with no conversation history. The "Notes" field contains information ABOUT the contact (who they are, their preferences, etc).

Ask clarifying questions about the USER (not the contact):
- Who are YOU (the user)? Your age, interests, preferences?
- What is YOUR intention? Why are you reaching out to this person?
- What do YOU want to communicate or accomplish?
- What's YOUR relationship dynamic with them?
- How well do YOU know this person?
- What response or outcome are you hoping for?`
	}

	prompt := fmt.Sprintf(`You are analyzing a conversation context to determine what clarifying questions would help you give better advice.

CONTACT INFORMATION (about %s):
- Name: %s
- Relationship: %s
- Platform: %s
- About them: %s

USER'S CONTEXT:
- Starting message: "%s"

%s

IMPORTANT: The "About them" section is information ABOUT the contact, not about the user.
Ask clarifying questions to better understand the USER's perspective, needs, and intention - not about the contact.

Based on what we know, what clarifying questions (up to %d) would help you understand the USER's perspective better?
Return a JSON object with:
{
  "understanding_level": 0-100 (how much we understand the USER's perspective),
  "questions": ["question1", "question2", "question3"],
  "reasoning": "brief explanation"
}

Only return valid JSON, no other text.`,
		contact.Name, contact.Name, contact.Relationship, contact.Platform, contact.Notes,
		messageStart, questionContext, questionCount)

	response, err := qa.callLLM(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze context: %v", err)
	}

	// Parse response
	analysis := &ContextAnalysis{}
	if err := json.Unmarshal([]byte(response), analysis); err != nil {
		return nil, fmt.Errorf("failed to parse analysis: %v", err)
	}

	analysis.KnownFacts = knownFacts
	return analysis, nil
}

func (qa *QuestionAgent) ExtractInsights(contactID int, userMessage, molyResponse string) (*InsightExtraction, error) {
	contact, err := qa.db.getContact(contactID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %v", err)
	}

	prompt := fmt.Sprintf(`Analyze this interaction and extract insights to update our knowledge about this person.

Contact: %s
What they said: "%s"
Response we gave: "%s"

Extract insights as JSON:
{
  "tone_detected": "friendly|neutral|professional|casual|enthusiastic|concerned|angry|other",
  "topics": ["topic1", "topic2"],
  "relationship_hints": "what we learned about the relationship (or null)",
  "communication_style": "brief description of how they communicate"
}

Only return valid JSON, no other text.`,
		contact.Name, userMessage, molyResponse)

	response, err := qa.callLLM(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to extract insights: %v", err)
	}

	insights := &InsightExtraction{}
	if err := json.Unmarshal([]byte(response), insights); err != nil {
		return nil, fmt.Errorf("failed to parse insights: %v", err)
	}

	return insights, nil
}

func (qa *QuestionAgent) extractKnownFacts(contact *Contact, patterns *BehaviorPattern, interactions []Interaction) []string {
	var facts []string

	if contact.Relationship != "" {
		facts = append(facts, fmt.Sprintf("Relationship: %s", contact.Relationship))
	}
	if contact.Notes != "" {
		facts = append(facts, fmt.Sprintf("Notes: %s", contact.Notes))
	}
	if contact.InteractionCount > 0 {
		facts = append(facts, fmt.Sprintf("Talked %d times before", contact.InteractionCount))
	}
	if contact.CommunicationStyle != "" {
		facts = append(facts, fmt.Sprintf("Communication style: %s", contact.CommunicationStyle))
	}
	if len(interactions) > 0 {
		topics := qa.topicsSummary(interactions)
		if topics != "" {
			facts = append(facts, fmt.Sprintf("Previous topics: %s", topics))
		}
	}
	if patterns != nil && patterns.PreferredTone != "" {
		facts = append(facts, fmt.Sprintf("Preferred tone: %s", patterns.PreferredTone))
	}

	if len(facts) == 0 {
		facts = append(facts, "No prior information available")
	}

	return facts
}

func (qa *QuestionAgent) topicsSummary(interactions []Interaction) string {
	var topics []string
	seen := make(map[string]bool)

	for _, interaction := range interactions {
		if interaction.Topic != "" && !seen[interaction.Topic] {
			topics = append(topics, interaction.Topic)
			seen[interaction.Topic] = true
			if len(topics) >= 3 {
				break
			}
		}
	}

	return strings.Join(topics, ", ")
}

func (qa *QuestionAgent) callLLM(prompt string) (string, error) {
	config := loadConfig()

	switch config.Provider {
	case "local":
		return chatWithOllama(prompt, config.Model, "direct")
	case "claude":
		return chatWithClaude(prompt, config.Model, "direct")
	case "openai":
		return chatWithOpenAI(prompt, config.Model, "direct")
	default:
		return "", fmt.Errorf("provider not configured: %s", config.Provider)
	}
}
