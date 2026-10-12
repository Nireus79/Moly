package agents

import (
	"context"
	"fmt"
	"log"
	"strings"

	"moly/models"
	"moly/tools"
)

// MetaInstruction represents an instruction about Moly's behavior or focus
type MetaInstruction struct {
	Type           string // "identity", "focus", "instruction", "scope", "constraint"
	Confidence     float64
	TargetTopic    string // what to focus on (if Type == "focus")
	TargetBehavior string // what behavior to adopt (if Type == "instruction")
	Contradictory  bool   // the message itself gives instructions about Moly that cannot both be followed
	RawInstruction string // the original instruction
	Reasoning      string // why this is a meta-instruction
	Source         string // always "llm"
}

// MetaInstructionDetector detects when the user gives an instruction about Moly itself ("don't give me advice", "let's focus on my
// sister's wedding"). The model decides; no grammar patterns or phrase lists are tried first (the earlier tiers could fire on a
// message that was not an instruction, and the model was never reached for the cases they matched).
type MetaInstructionDetector struct {
	llmClient tools.LLMProvider
}

// NewMetaInstructionDetector creates a new meta-instruction detector
func NewMetaInstructionDetector(llm tools.LLMProvider) *MetaInstructionDetector {
	return &MetaInstructionDetector{llmClient: llm}
}

// Detect analyzes if the message is a meta-instruction about Moly.
func (mid *MetaInstructionDetector) Detect(ctx context.Context, message string) *MetaInstruction {
	return mid.DetectWithExchange(ctx, message, "")
}

// DetectWithExchange is Detect with the recent exchange in view: a reply such as "yes" may answer a question about how Moly
// should behave, and means nothing alone. Without a model, or when the answer cannot be read, there is no meta-instruction.
func (mid *MetaInstructionDetector) DetectWithExchange(ctx context.Context, message, exchange string) *MetaInstruction {
	msg := strings.TrimSpace(message)
	if msg == "" || mid.llmClient == nil {
		return nil
	}
	return mid.detectByLLM(ctx, msg, exchange)
}

// detectByLLM asks the model whether the message is a meta-instruction
func (mid *MetaInstructionDetector) detectByLLM(ctx context.Context, msg, exchange string) *MetaInstruction {
	systemPrompt := `You are a meta-instruction detector. Analyze if this message is an instruction ABOUT Moly's behavior/focus, not TO Moly for advice.

Meta-instructions include:
- Self-reference: "You are Moly", "I'm talking to Moly"
- Focus directives: "Lace is my focus", "let's focus on X"
- Constraints: "remember to...", "keep in mind..."
- Scope: "my priority is X", "don't give me advice about Y"
- Identity: "act like...", "be my..."

RESPOND WITH ONLY JSON (no markdown):
{
  "isMetaInstruction": true/false,
  "type": "identity|focus|instruction|scope|constraint|none",
  "confidence": 0.0-1.0,
  "targetTopic": "extracted focus topic or null",
  "targetBehavior": "extracted behavior or null",
  "contradictory": true/false,
  "reasoning": "brief explanation"
}

"contradictory" is true ONLY when this one message gives two instructions about how Moly should behave that cannot both be followed (for example "give me advice" together with "don't give me advice", or "keep it very short" together with "explain everything in detail"). Styles that fit together ("be direct but kind") are not contradictory. It is false when there is a single instruction or none.`

	userPrompt := models.WithExchange(exchange) + fmt.Sprintf(`Message: "%s"

Is this a meta-instruction about Moly's behavior or focus?`, msg)

	req := &tools.LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Temperature:  0.3,
		MaxTokens:    200,
	}

	resp, err := mid.llmClient.Call(ctx, req)
	if err != nil {
		log.Printf("[MetaInstructionDetector] LLM call failed: %v", err)
		return nil
	}

	// Parse response
	result := mid.parseMetaInstructionResponse(resp.Content, msg)
	if result != nil {
		log.Printf("[MetaInstructionDetector] Detected: type=%s, confidence=%.2f, topic=%s", result.Type, result.Confidence, result.TargetTopic)
	}

	return result
}

// parseMetaInstructionResponse reads the model's JSON answer. It used to look for the exact text `"isMetaInstruction":true`, which a
// model that writes `"isMetaInstruction": true` (Claude haiku, always) never matched: every meta-instruction was missed (found by the
// live judge battery, 0 of 9).
func (mid *MetaInstructionDetector) parseMetaInstructionResponse(response, originalMsg string) *MetaInstruction {
	var out struct {
		IsMeta         tools.LenientBool `json:"isMetaInstruction"`
		Type           string            `json:"type"`
		Confidence     float64           `json:"confidence"`
		TargetTopic    string            `json:"targetTopic"`
		TargetBehavior string            `json:"targetBehavior"`
		Contradictory  tools.LenientBool `json:"contradictory"`
		Reasoning      string            `json:"reasoning"`
	}
	if err := tools.SafeJSONParse("MetaInstruction", []byte(response), &out); err != nil || !bool(out.IsMeta) {
		return nil
	}
	switch out.Type {
	case "identity", "focus", "instruction", "scope", "constraint":
	default:
		return nil
	}
	result := &MetaInstruction{
		Type:           out.Type,
		Confidence:     out.Confidence,
		TargetTopic:    out.TargetTopic,
		TargetBehavior: out.TargetBehavior,
		Contradictory:  bool(out.Contradictory),
		Reasoning:      out.Reasoning,
		RawInstruction: originalMsg,
		Source:         "llm",
	}
	if result.Confidence == 0 {
		result.Confidence = 0.7 // default when the model gave none
	}
	return result
}
