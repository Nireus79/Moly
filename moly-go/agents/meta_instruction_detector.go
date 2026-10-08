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
	TargetTopic    string   // what to focus on (if Type == "focus")
	TargetBehavior string   // what behavior to adopt (if Type == "instruction")
	Subjects       []string // who the instruction is about (e.g., ["user", "she"])
	IsNegated      bool     // whether the instruction is negated
	RawInstruction string   // the original instruction
	Reasoning      string   // why this is a meta-instruction
	Source         string   // which tier detected this: "linguistic_parser", "keywords", "llm"
}

// MetaInstructionDetector detects when user is giving instructions about Moly itself
type MetaInstructionDetector struct {
	llmClient tools.LLMProvider
	parser    *tools.LinguisticParser // NEW: Tier 1 detection
}

// NewMetaInstructionDetector creates a new meta-instruction detector
func NewMetaInstructionDetector(llm tools.LLMProvider) *MetaInstructionDetector {
	return &MetaInstructionDetector{
		llmClient: llm,
		parser:    tools.NewLinguisticParser(),
	}
}

// Detect analyzes if message is a meta-instruction about Moly
// Uses 3-tier approach: Tier 1 (Linguistic Parser), Tier 2 (Keywords), Tier 3 (LLM)
func (mid *MetaInstructionDetector) Detect(ctx context.Context, message string) *MetaInstruction {
	if message == "" {
		return nil
	}

	msg := strings.TrimSpace(message)

	// Tier 1: LinguisticParser (NEW - deterministic, fast, accurate)
	if result := mid.extractMetaFromLinguistic(msg); result != nil {
		return result
	}

	// Tier 2: Keyword-based detection (deterministic, no LLM)
	if result := mid.detectByKeywords(msg); result != nil {
		return result
	}

	// Tier 3: LLM-based detection for nuanced cases (slow but accurate)
	if mid.llmClient != nil {
		return mid.detectByLLM(ctx, msg)
	}

	return nil
}

// extractMetaFromLinguistic converts LinguisticParser extractions to MetaInstruction
// This is Tier 1 detection: deterministic, <100ms, no LLM needed
func (mid *MetaInstructionDetector) extractMetaFromLinguistic(msg string) *MetaInstruction {
	extractions := mid.parser.Parse(msg)

	for _, ext := range extractions {
		// Map grammar extraction types to meta-instruction types
		switch ext.Type {
		case "focus":
			return &MetaInstruction{
				Type:           "focus",
				Confidence:     ext.Confidence,
				TargetTopic:    ext.Property,
				Subjects:       []string{ext.Subject},
				IsNegated:      ext.IsNegated,
				RawInstruction: msg,
				Reasoning:      fmt.Sprintf("Focus directive: %s", ext.RawMatch),
				Source:         "linguistic_parser",
			}

		case "constraint":
			return &MetaInstruction{
				Type:           "constraint",
				Confidence:     ext.Confidence,
				TargetBehavior: ext.Property,
				Subjects:       []string{ext.Subject},
				IsNegated:      ext.IsNegated,
				RawInstruction: msg,
				Reasoning:      fmt.Sprintf("Constraint: %s", ext.RawMatch),
				Source:         "linguistic_parser",
			}

		case "priority":
			return &MetaInstruction{
				Type:           "focus",
				Confidence:     ext.Confidence,
				TargetTopic:    ext.Property,
				Subjects:       []string{ext.Subject},
				IsNegated:      ext.IsNegated,
				RawInstruction: msg,
				Reasoning:      fmt.Sprintf("Priority: %s", ext.RawMatch),
				Source:         "linguistic_parser",
			}

		case "interest":
			// "not interested in X" or "don't want X"
			return &MetaInstruction{
				Type:           "scope",
				Confidence:     ext.Confidence,
				TargetTopic:    ext.Property,
				Subjects:       []string{ext.Subject},
				IsNegated:      ext.IsNegated,
				RawInstruction: msg,
				Reasoning:      fmt.Sprintf("Interest/preference: %s", ext.RawMatch),
				Source:         "linguistic_parser",
			}

		case "preference":
			// "don't want X"
			return &MetaInstruction{
				Type:           "scope",
				Confidence:     ext.Confidence,
				TargetTopic:    ext.Property,
				Subjects:       []string{ext.Subject},
				IsNegated:      ext.IsNegated,
				RawInstruction: msg,
				Reasoning:      fmt.Sprintf("Preference: %s", ext.RawMatch),
				Source:         "linguistic_parser",
			}
		}
	}

	return nil
}

// detectByKeywords - Simplified Tier 2 detection (only obvious identity cases)
// Most focus/constraint detection is now handled by Tier 1 (LinguisticParser)
func (mid *MetaInstructionDetector) detectByKeywords(msg string) *MetaInstruction {
	lower := strings.ToLower(msg)

	// Pattern: "You are Moly" or "You're Moly" (identity self-reference)
	// This is the only keyword pattern - everything else goes to Tier 1 (LinguisticParser) or Tier 3 (LLM)
	if strings.Contains(lower, "you are moly") || strings.Contains(lower, "you're moly") ||
		strings.Contains(lower, "you are moly") || strings.Contains(lower, "i am talking to moly") {
		return &MetaInstruction{
			Type:           "identity",
			Confidence:     0.95,
			RawInstruction: msg,
			Reasoning:      "Direct self-reference to Moly",
			Source:         "keywords",
		}
	}

	return nil
}

// detectByLLM - Tier 3: LLM-based detection for nuanced meta-instructions
func (mid *MetaInstructionDetector) detectByLLM(ctx context.Context, msg string) *MetaInstruction {
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
		Source:         "llm",
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
