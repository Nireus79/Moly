package agents

import (
	"context"
	"fmt"
	"log"
	"strings"

	"moly/tools"
)

// MetaInstruction represents an instruction about Moly's behavior or focus
type MetaInstruction struct {
	Type           string // "identity", "focus", "instruction", "scope", "constraint"
	Confidence     float64
	TargetTopic    string // what to focus on (if Type == "focus")
	TargetBehavior string // what behavior to adopt (if Type == "instruction")
	RawInstruction string // the original instruction
	Reasoning      string // why this is a meta-instruction
}

// MetaInstructionDetector detects when user is giving instructions about Moly itself
type MetaInstructionDetector struct {
	llmClient tools.LLMProvider
}

// NewMetaInstructionDetector creates a new meta-instruction detector
func NewMetaInstructionDetector(llm tools.LLMProvider) *MetaInstructionDetector {
	return &MetaInstructionDetector{llmClient: llm}
}

// Detect analyzes if message is a meta-instruction about Moly
func (mid *MetaInstructionDetector) Detect(ctx context.Context, message string) *MetaInstruction {
	if message == "" {
		return nil
	}

	msg := strings.TrimSpace(message)

	// Fast path: keyword-based detection (deterministic, no LLM)
	if fastResult := mid.detectByKeywords(msg); fastResult != nil {
		return fastResult
	}

	// Slow path: LLM-based detection for nuanced cases
	if mid.llmClient != nil {
		return mid.detectByLLM(ctx, msg)
	}

	return nil
}

// detectByKeywords - Fast deterministic detection using patterns
func (mid *MetaInstructionDetector) detectByKeywords(msg string) *MetaInstruction {
	lower := strings.ToLower(msg)

	// Pattern: "You are Moly" or "You're Moly" (identity self-reference)
	if strings.Contains(lower, "you are moly") || strings.Contains(lower, "you're moly") {
		return &MetaInstruction{
			Type:           "identity",
			Confidence:     0.95,
			RawInstruction: msg,
			Reasoning:      "Direct self-reference to Moly",
		}
	}

	// Pattern: "X is my focus" or "focus on X" or "my focus is X"
	if strings.Contains(lower, "my focus") || strings.Contains(lower, "is my focus") {
		focus := extractFocusTarget(msg)
		if focus != "" {
			return &MetaInstruction{
				Type:           "focus",
				Confidence:     0.9,
				TargetTopic:    focus,
				RawInstruction: msg,
				Reasoning:      "Explicit focus/scope directive",
			}
		}
	}

	// Pattern: "focus on X" or "let's focus on X"
	if strings.Contains(lower, "focus on ") {
		focus := extractFocusTarget(msg)
		if focus != "" {
			return &MetaInstruction{
				Type:           "focus",
				Confidence:     0.85,
				TargetTopic:    focus,
				RawInstruction: msg,
				Reasoning:      "Explicit focus directive",
			}
		}
	}

	// Pattern: "remember X" or "don't forget X" (constraint/reminder)
	if strings.Contains(lower, "remember ") || strings.Contains(lower, "don't forget") || strings.Contains(lower, "keep in mind") {
		target := extractConstraintTarget(msg)
		if target != "" {
			return &MetaInstruction{
				Type:           "constraint",
				Confidence:     0.85,
				TargetBehavior: target,
				RawInstruction: msg,
				Reasoning:      "Constraint or reminder instruction",
			}
		}
	}

	// Pattern: "my priority is X" or "prioritize X"
	if strings.Contains(lower, "priority") || strings.Contains(lower, "prioritize") {
		focus := extractFocusTarget(msg)
		if focus != "" {
			return &MetaInstruction{
				Type:           "focus",
				Confidence:     0.85,
				TargetTopic:    focus,
				RawInstruction: msg,
				Reasoning:      "Priority/focus statement",
			}
		}
	}

	return nil
}

// extractFocusTarget extracts what user wants to focus on
func extractFocusTarget(msg string) string {
	// Pattern: "X is my focus"
	if idx := strings.Index(strings.ToLower(msg), "is my focus"); idx != -1 {
		// Get the part before "is my focus"
		before := msg[:idx]
		// Take the last word or name
		parts := strings.Fields(strings.TrimSpace(before))
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// Pattern: "focus on X"
	if idx := strings.Index(strings.ToLower(msg), "focus on "); idx != -1 {
		after := msg[idx+len("focus on "):]
		words := strings.Fields(strings.TrimSpace(after))
		if len(words) > 0 {
			// Take first one or two words (e.g., "Lace" or "my mom")
			if len(words) == 1 {
				return words[0]
			}
			if len(words) == 2 && (words[0] == "my" || words[0] == "the") {
				return words[1]
			}
			return words[0]
		}
	}

	// Pattern: "my priority is X"
	if idx := strings.Index(strings.ToLower(msg), "priority is "); idx != -1 {
		after := msg[idx+len("priority is "):]
		words := strings.Fields(strings.TrimSpace(after))
		if len(words) > 0 {
			return words[0]
		}
	}

	return ""
}

// extractConstraintTarget extracts what to remember/keep in mind
func extractConstraintTarget(msg string) string {
	lower := strings.ToLower(msg)
	var after string

	// "remember X"
	if idx := strings.Index(lower, "remember "); idx != -1 {
		after = msg[idx+len("remember "):]
	}
	// "don't forget X"
	if idx := strings.Index(lower, "don't forget "); idx != -1 {
		after = msg[idx+len("don't forget "):]
	}
	// "keep in mind X"
	if idx := strings.Index(lower, "keep in mind "); idx != -1 {
		after = msg[idx+len("keep in mind "):]
	}

	if after != "" {
		// Take up to next sentence boundary
		if endIdx := strings.IndexAny(after, ".!?"); endIdx != -1 {
			return strings.TrimSpace(after[:endIdx])
		}
		return strings.TrimSpace(after)
	}

	return ""
}

// detectByLLM - LLM-based detection for nuanced meta-instructions
func (mid *MetaInstructionDetector) detectByLLM(ctx context.Context, msg string) *MetaInstruction {
	systemPrompt := `You are a meta-instruction detector. Analyze if this message is an instruction ABOUT Moly's behavior/focus, not TO Moly for advice.

Meta-instructions include:
- Self-reference: "You are Moly", "I'm talking to Moly"
- Focus directives: "Lace is my focus", "let's focus on X", or IMPLICIT topic focus (e.g., "girl... her name is Lace" = Lace is topic to track)
- Constraints: "remember to...", "keep in mind..."
- Scope: "my priority is X", "don't give me advice about Y"
- Identity: "act like...", "be my..."

KEY: Detect IMPLICIT focus when user introduces a new named entity (person, topic) they want to discuss.
Example implicit focus:
- Input: "girl I found on fetlife, her profile is Lace"
- Output: {"isMetaInstruction": true, "type": "focus", "targetTopic": "Lace", "confidence": 0.8}

RESPOND WITH ONLY JSON (no markdown):
{
  "isMetaInstruction": true/false,
  "type": "identity|focus|instruction|scope|constraint|none",
  "confidence": 0.0-1.0,
  "targetTopic": "extracted focus topic or null",
  "targetBehavior": "extracted behavior or null",
  "reasoning": "brief explanation"
}`

	userPrompt := fmt.Sprintf(`Message: "%s"

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

// parseMetaInstructionResponse parses LLM response
func (mid *MetaInstructionDetector) parseMetaInstructionResponse(response, originalMsg string) *MetaInstruction {
	// Basic JSON parsing
	if !strings.Contains(response, "isMetaInstruction") {
		return nil
	}

	isMetaInstr := strings.Contains(response, `"isMetaInstruction":true`)
	if !isMetaInstr {
		return nil
	}

	// Extract fields from JSON response (simple string matching)
	result := &MetaInstruction{
		RawInstruction: originalMsg,
	}

	// Extract type
	if strings.Contains(response, `"type":"identity"`) {
		result.Type = "identity"
	} else if strings.Contains(response, `"type":"focus"`) {
		result.Type = "focus"
	} else if strings.Contains(response, `"type":"instruction"`) {
		result.Type = "instruction"
	} else if strings.Contains(response, `"type":"scope"`) {
		result.Type = "scope"
	} else if strings.Contains(response, `"type":"constraint"`) {
		result.Type = "constraint"
	} else {
		return nil
	}

	// Extract confidence (simple approach: look for number after confidence)
	if idx := strings.Index(response, `"confidence"`); idx != -1 {
		after := response[idx+len(`"confidence"`):]
		if idx2 := strings.Index(after, ":"); idx2 != -1 {
			numStr := after[idx2+1:]
			if idx3 := strings.IndexAny(numStr, ",}"); idx3 != -1 {
				numStr = numStr[:idx3]
			}
			fmt.Sscanf(strings.TrimSpace(numStr), "%f", &result.Confidence)
		}
	}

	if result.Confidence == 0 {
		result.Confidence = 0.7 // Default if parsing failed
	}

	// Extract targetTopic if present
	if idx := strings.Index(response, `"targetTopic"`); idx != -1 {
		after := response[idx+len(`"targetTopic"`):]
		if idx2 := strings.Index(after, ":"); idx2 != -1 {
			after = after[idx2+1:]
			if strings.Contains(after, `"`) {
				if idx3 := strings.Index(after, `"`); idx3 != -1 {
					after = after[idx3+1:]
					if idx4 := strings.Index(after, `"`); idx4 != -1 {
						result.TargetTopic = after[:idx4]
					}
				}
			}
		}
	}

	// Extract reasoning if present
	if idx := strings.Index(response, `"reasoning"`); idx != -1 {
		after := response[idx+len(`"reasoning"`):]
		if idx2 := strings.Index(after, ":"); idx2 != -1 {
			after = after[idx2+1:]
			if strings.Contains(after, `"`) {
				if idx3 := strings.Index(after, `"`); idx3 != -1 {
					after = after[idx3+1:]
					if idx4 := strings.Index(after, `"`); idx4 != -1 {
						result.Reasoning = after[:idx4]
					}
				}
			}
		}
	}

	return result
}
