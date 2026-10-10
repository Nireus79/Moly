package agents

import (
	"context"
	"fmt"
	"log"

	"moly/tools"
)

// ClarificationAgent uses LLM to generate natural clarification questions
type ClarificationAgent struct {
	llmClient tools.LLMProvider
}

// NewClarificationAgent creates a new clarification agent
func NewClarificationAgent(llmClient tools.LLMProvider) *ClarificationAgent {
	return &ClarificationAgent{
		llmClient: llmClient,
	}
}

// ProcessAnswer extracts context from user's answer
func (ca *ClarificationAgent) ProcessAnswer(
	question string,
	userAnswer string,
	linkedFacts []ExtractedFact,
) (*AnswerProcessingResult, error) {
	log.Printf("[ClarificationAgent] Processing answer to: %s", question)

	// Ask LLM to extract context from answer
	prompt := fmt.Sprintf(`You are analyzing a user's response to understand their communication context.

Question asked: "%s"
User answered: "%s"

Extract the key information revealed by this answer. Identify:
1. What did they tell us? (concrete facts)
2. What does this imply about their communication style? (patterns)
3. What should we remember about their preferences? (context)

Format as JSON:
{
  "factsExtracted": ["fact1", "fact2"],
  "patterns": ["pattern1", "pattern2"],
  "contextToSave": "what to save about them",
  "needsMoreClarification": true/false,
  "followUpGap": "if needs more, what should we ask about?"
}`, question, userAnswer)

	resp, err := ca.llmClient.Call(context.Background(), &tools.LLMRequest{
		UserPrompt:  prompt,
		MaxTokens:   500,
		Temperature: 0.5,
	})
	if err != nil {
		log.Printf("[ClarificationAgent] Error processing answer: %v", err)
		return nil, err
	}

	response := resp.Content

	// Parse JSON response
	result := &AnswerProcessingResult{
		RawResponse: response,
	}

	log.Printf("[ClarificationAgent] ✓ Processed answer: %s", response)
	return result, nil
}

// ConfirmUnderstanding generates a natural acknowledgment of user's answer
func (ca *ClarificationAgent) ConfirmUnderstanding(
	question string,
	userAnswer string,
) (string, error) {
	log.Printf("[ClarificationAgent] Generating acknowledgment")

	prompt := fmt.Sprintf(`You are Moly, an AI communication coach.
A user answered your clarification question.

Question you asked: "%s"
User answered: "%s"

Generate a natural, brief acknowledgment that shows you understood them.
Requirements:
- Acknowledge what they said (1 sentence)
- Show you understand the implication (1 sentence)
- Be warm and natural
- Max 2 sentences total
- No questions or follow-ups

Generate just the acknowledgment, nothing else.`, question, userAnswer)

	resp, err := ca.llmClient.Call(context.Background(), &tools.LLMRequest{
		UserPrompt:  prompt,
		MaxTokens:   200,
		Temperature: 0.7,
	})
	if err != nil {
		log.Printf("[ClarificationAgent] Error generating reply: %v", err)
		return "", err
	}

	reply := resp.Content

	log.Printf("[ClarificationAgent] ✓ Generated reply: %s", reply)
	return reply, nil
}

// AnswerProcessingResult holds extracted data from a user's answer
type AnswerProcessingResult struct {
	RawResponse            string
	FactsExtracted         []string
	Patterns               []string
	ContextToSave          string
	NeedsMoreClarification bool
	FollowUpGap            string
}
