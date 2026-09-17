package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// HarmAnalysisInput - Input for harm analysis
type HarmAnalysisInput struct {
	Message string `json:"message"`
}

// HarmAnalysisOutput - Output from harm analyzer
type HarmAnalysisOutput struct {
	IsHarmful        bool   `json:"is_harmful"`
	Severity         string `json:"severity"` // "none", "warn", "block"
	Category         string `json:"category"` // "crisis", "illegal", "unsafe", "none"
	ReasoningCategory string `json:"reasoning_category"` // General category explanation only
	ShouldRespond    bool   `json:"should_respond"`
	SafeResponse     string `json:"safe_response"` // What to tell the user if blocked
}

// HarmAnalyzer - Analyzes messages for harmful content
type HarmAnalyzer struct {
	llm LLMProvider
}

// NewHarmAnalyzer - Create new harm analyzer
func NewHarmAnalyzer(llm LLMProvider) *HarmAnalyzer {
	return &HarmAnalyzer{llm: llm}
}

// Analyze - Analyze a message for harmful content
func (ha *HarmAnalyzer) Analyze(ctx context.Context, input *HarmAnalysisInput) (*HarmAnalysisOutput, error) {
	if ha.llm == nil {
		// Fallback if no LLM
		return &HarmAnalysisOutput{
			IsHarmful:    false,
			Severity:     "none",
			Category:     "none",
			ShouldRespond: true,
		}, nil
	}

	output := &HarmAnalysisOutput{
		IsHarmful:     false,
		Severity:      "none",
		Category:      "none",
		ShouldRespond: true,
	}

	systemPrompt := ha.buildSystemPrompt()
	userPrompt := ha.buildUserPrompt(input.Message)

	log.Printf("[HarmAnalyzer] Analyzing message for harm")

	req := &LLMRequest{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		Temperature:         0.1,
		MaxTokens:           500,
		UseExtendedThinking: false,
	}

	resp, err := ha.llm.Call(ctx, req)
	if err != nil {
		log.Printf("[HarmAnalyzer] Error: %v", err)
		// Fail open - if analysis fails, allow the message
		return output, nil
	}

	// Parse response
	err = json.Unmarshal([]byte(resp.Content), output)
	if err != nil {
		log.Printf("[HarmAnalyzer] Parse error: %v", err)
		// Fail open - if parsing fails, allow the message
		return output, nil
	}

	log.Printf("[HarmAnalyzer] Analysis result: severity=%s category=%s should_respond=%v",
		output.Severity, output.Category, output.ShouldRespond)

	return output, nil
}

func (ha *HarmAnalyzer) buildSystemPrompt() string {
	return `You are a safety expert. Analyze messages for harmful content.

SEVERITY LEVELS:
- "none": Safe message, respond normally
- "warn": Contains concerning content but user is seeking help or expressing emotion - respond with compassion
- "block": Explicit request for harmful instructions or imminent danger - do not provide requested content

CATEGORIES:
- "crisis": Explicit suicide/self-harm intent, active immediate danger
- "illegal": Explicit request for how to commit illegal activities (not discussion of ethics/legality)
- "unsafe": Pressure tactics, manipulation, harassment, or safety concerns
- "none": Safe

CRITICAL GUARDRAILS:
1. ONLY flag "crisis" if message explicitly says they WANT TO HARM themselves or others RIGHT NOW
   - Do NOT flag: sadness, depression, emotional distress, asking for help, suicidal thoughts without action intent
   - DO flag: "I'm going to kill myself", "I want to hurt myself", explicit plan

2. ONLY flag "illegal" if message ASKS HOW TO DO something illegal
   - Do NOT flag: discussion of laws, asking if something is legal, venting about frustration
   - DO flag: "How do I make explosives", "How do I hack into...", "Tell me how to steal"

3. For "reasoning_category", ONLY describe the GENERAL CATEGORY of concern
   - Do NOT include: specific methods, steps, quantities, or operational details from user's request
   - GOOD: "This involves dangerous substance preparation"
   - BAD: "This describes how to make explosives step-by-step"

4. When user is SEEKING HELP or EXPRESSING EMOTION, always set "warn" not "block" with compassionate response

5. Return ONLY valid JSON, no explanation.

RESPONSE FORMAT (JSON only, no text):
{
  "is_harmful": boolean,
  "severity": "none" | "warn" | "block",
  "category": "crisis" | "illegal" | "unsafe" | "none",
  "reasoning_category": "brief general category description ONLY",
  "should_respond": boolean,
  "safe_response": "compassionate message if blocked, empty string if responding normally"
}
`
}

func (ha *HarmAnalyzer) buildUserPrompt(message string) string {
	return fmt.Sprintf(`Analyze this message for harmful content:

"%s"

Return ONLY valid JSON (no explanation or code blocks).`, message)
}
