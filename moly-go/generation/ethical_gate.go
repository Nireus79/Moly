package generation

import (
	"log"
	"strings"

	"moly/models"
)

// EthicalCheck - Result of ethical safety check
type EthicalCheck struct {
	Severity       string // "BLOCK" | "WARN" | "NONE"
	Category       string // harm category
	Reason         string // general category of concern (not specifics)
	ShouldRespond  bool
	SafeResponse   string // safe alternative if BLOCK
}

// EthicalGate - Safety and ethical checking for responses
type EthicalGate struct {
}

// NewEthicalGate - Create ethical gate checker
func NewEthicalGate() *EthicalGate {
	return &EthicalGate{}
}

// CheckResponse - Check if response violates safety guidelines
func (eg *EthicalGate) CheckResponse(userMessage, response string) *EthicalCheck {
	log.Printf("[EthicalGate] Checking response safety")

	// Check 1: Is user asking for harmful content?
	if eg.isHarmfulRequest(userMessage) {
		return &EthicalCheck{
			Severity:      "BLOCK",
			Category:      "harmful_request",
			Reason:        "This request could cause harm",
			ShouldRespond: false,
			SafeResponse:  "I can't help with that request. Could we discuss something else I can help with?",
		}
	}

	// Check 2: Does response contain harmful content?
	if eg.containsHarmfulContent(response) {
		return &EthicalCheck{
			Severity:      "BLOCK",
			Category:      "harmful_content",
			Reason:        "The response contains harmful content",
			ShouldRespond: false,
			SafeResponse:  "Let me reconsider my response to make sure it's helpful and safe.",
		}
	}

	// Check 3: Does response contain sensitive personal data?
	if eg.revealsPersonalData(response) {
		return &EthicalCheck{
			Severity:      "WARN",
			Category:      "personal_data",
			Reason:        "Response may contain personal information that shouldn't be recorded",
			ShouldRespond: true,
		}
	}

	// No issues found
	return &EthicalCheck{
		Severity:      "NONE",
		ShouldRespond: true,
	}
}

// isHarmfulRequest - Heuristically detect if user is asking for harmful content
func (eg *EthicalGate) isHarmfulRequest(msg string) bool {
	lower := strings.ToLower(msg)

	// Dangerous patterns (high confidence)
	dangerousWords := []string{
		"how to hurt", "how to harm", "how to kill",
		"bomb", "explosive", "weapon",
		"cocaine", "heroin", "illegal drug",
		"hack", "steal", "fraud", "scam",
		"self-harm", "suicide",
	}

	for _, word := range dangerousWords {
		if strings.Contains(lower, word) {
			return true
		}
	}

	return false
}

// containsHarmfulContent - Check if response contains harmful content
func (eg *EthicalGate) containsHarmfulContent(response string) bool {
	lower := strings.ToLower(response)

	// Harmful content patterns
	harmfulPatterns := []string{
		"how to hurt", "instructions for", "step-by-step",
		"bomb", "explosive", "weapon",
	}

	for _, pattern := range harmfulPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// revealsPersonalData - Check if response reveals personal data
func (eg *EthicalGate) revealsPersonalData(response string) bool {
	// This is a basic check - real implementation would be more comprehensive
	lower := strings.ToLower(response)

	// Patterns that suggest personal data revelation
	patterns := []string{
		"email address", "phone number", "social security",
		"credit card", "password", "ssn",
	}

	for _, pattern := range patterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// ApplyGate - Apply ethical gate to response
func (eg *EthicalGate) ApplyGate(resp *models.ConversationResponse, userMessage string) *models.ConversationResponse {
	log.Printf("[EthicalGate] Applying ethical gate")

	check := eg.CheckResponse(userMessage, resp.Response)

	if check.Severity == "BLOCK" {
		log.Printf("[EthicalGate] BLOCKED: %s", check.Reason)
		resp.Response = check.SafeResponse
		resp.Metadata["ethicalIntervention"] = "blocked"
		resp.Metadata["ethicalReason"] = check.Reason
		resp.Metadata["ethicalCategory"] = check.Category
		return resp
	}

	if check.Severity == "WARN" {
		log.Printf("[EthicalGate] WARNING: %s", check.Reason)
		resp.Metadata["ethicalIntervention"] = "warned"
		resp.Metadata["ethicalReason"] = check.Reason
		resp.Metadata["ethicalCategory"] = check.Category
		return resp
	}

	// NONE - response passes
	log.Printf("[EthicalGate] ✓ Response passes ethical check")
	return resp
}
