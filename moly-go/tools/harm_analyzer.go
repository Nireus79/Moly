package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"moly/models"
)

// HarmAnalysis represents the result of ethical harm analysis
type HarmAnalysis struct {
	Intent             string   `json:"intent"`              // What is being advised
	HarmType           string   `json:"harm_type"`           // Type of potential harm
	Severity           string   `json:"severity"`            // "critical", "moderate", "minor", "none"
	AffectedParties    []string `json:"affected_parties"`    // Who could be hurt
	Reasoning          string   `json:"reasoning"`           // Why this is harmful
	Intervention       string   `json:"intervention"`        // "BLOCK", "MODIFY", "WARN", "PROCEED"
	SuggestedAlternative string `json:"suggested_alternative"` // What to say instead (if modifying)
	ModifiedResponse   string   `json:"modified_response"`   // Softened version (if modifying)
	Explanation        string   `json:"explanation"`         // Why intervention happening (for user)
	ViolatedPrinciples []string `json:"violated_principles"` // Constitutional principles violated
}

// UserVulnerability represents vulnerability factors we've learned about user
type UserVulnerability struct {
	TraumaHistory    bool
	MentalHealthIssues []string // depression, anxiety, PTSD, etc
	Patterns         []string  // conflict-avoidant, people-pleaser, etc
	Confidence       string    // "high", "medium", "low"
	CommunicationStyle string
}

// ContactTraits represents what we've learned about a contact
type ContactTraits struct {
	Relationship string   // romantic, friend, professional, family
	Sensitivity  string   // high, medium, low
	Traits       []string // quiet, sensitive, volatile, etc
	Stability    string   // stable, unstable
}

// HarmAnalyzer uses LLM to reason about ethical harm
type HarmAnalyzer struct {
	llmClient     LLMProvider
	constitution  *models.Constitution // Optional: for principle-based checking
}

// NewHarmAnalyzer creates a new harm analyzer
func NewHarmAnalyzer(llmClient LLMProvider) *HarmAnalyzer {
	return &HarmAnalyzer{
		llmClient:    llmClient,
		constitution: nil, // Optional - set via SetConstitution if available
	}
}

// SetConstitution injects the constitution for principle-based checking (optional)
func (ha *HarmAnalyzer) SetConstitution(constitution *models.Constitution) {
	if ha != nil {
		ha.constitution = constitution
		log.Printf("[HarmAnalyzer] Constitution loaded for principle-based checking")
	}
}

// CheckPrinciples checks if response violates any constitutional principles
// Returns list of violated principles (empty if none violated or constitution not available)
func (ha *HarmAnalyzer) CheckPrinciples(response string) []*models.Principle {
	if ha.constitution == nil {
		return nil // Constitution not loaded
	}

	if response == "" {
		return nil
	}

	violated := make([]*models.Principle, 0)
	lowerResponse := strings.ToLower(response)

	// Check each principle's keywords
	for i := range ha.constitution.SupremePrinciples {
		principle := &ha.constitution.SupremePrinciples[i]

		// Check if any of the principle's check_keywords appear in response
		for _, keyword := range principle.CheckKeywords {
			if strings.Contains(lowerResponse, strings.ToLower(keyword)) {
				log.Printf("[HarmAnalyzer] Principle violation detected: %s (keyword: %s)", principle.Name, keyword)
				violated = append(violated, principle)
				break // Only add once per principle
			}
		}
	}

	return violated
}

// AnalyzeResponse checks if a response could cause harm
// Returns HarmAnalysis with severity level and recommended intervention
func (ha *HarmAnalyzer) AnalyzeResponse(
	ctx context.Context,
	response string,
	userVuln *UserVulnerability,
	contactTraits *ContactTraits,
) (*HarmAnalysis, error) {

	if response == "" {
		return &HarmAnalysis{Severity: "none", Intervention: "PROCEED"}, nil
	}

	// Build context about vulnerabilities
	userContext := ha.buildUserContext(userVuln)
	contactContext := ha.buildContactContext(contactTraits)

	// Build prompt for LLM to reason about harm
	prompt := ha.buildHarmAnalysisPrompt(response, userContext, contactContext)

	req := &LLMRequest{
		SystemPrompt: `You are an ethical reasoning system. Your job is to analyze if a communication
response could cause harm to the user or others they might communicate with.

You reason through:
1. What action/words is this response advising?
2. Who could be hurt? (user, contact, third parties)
3. What type of harm? (direct violence, psychological trauma, manipulation, degradation, relationship sabotage)
4. How likely and severe?
5. Does the user's vulnerabilities amplify harm?
6. Is the benefit worth the risk?

Be honest about harm. Don't rationalize away real risks.

CRITICAL SAFETY RULE: When describing "reasoning" or "explanation", focus ONLY on the CATEGORY of harm
(e.g. "this could involve violence," "this pressures someone without consent"). DO NOT restate or repeat
specific operational details, methods, quantities, steps, or technical procedures from the harmful request,
even when explaining why it's harmful. The explanation shown to the user must make clear what KIND of concern
Moly has, never what SPECIFICALLY was withheld.

Return a JSON analysis with your reasoning and recommended intervention.`,
		UserPrompt:  prompt,
		Temperature: 0.7,
		MaxTokens:   600,
	}

	resp, err := ha.llmClient.Call(ctx, req)
	if err != nil {
		log.Printf("[HarmAnalyzer] LLM call failed: %v", err)
		// Graceful degradation: assume no harm if analyzer fails
		return &HarmAnalysis{
			Severity: "none",
			Intervention: "PROCEED",
			Reasoning: "Unable to analyze (analyzer failed)",
		}, nil
	}

	if resp == nil || resp.Content == "" {
		return &HarmAnalysis{Severity: "none", Intervention: "PROCEED"}, nil
	}

	// Parse LLM response
	analysis := &HarmAnalysis{}
	if err := json.Unmarshal([]byte(resp.Content), analysis); err != nil {
		log.Printf("[HarmAnalyzer] Failed to parse analysis: %v. Response: %s", err, resp.Content)
		// Graceful degradation: assume no harm if parsing fails
		return &HarmAnalysis{
			Severity: "none",
			Intervention: "PROCEED",
			Reasoning: "Unable to analyze (parsing failed)",
		}, nil
	}

	// Normalize severity
	analysis.Severity = strings.ToLower(analysis.Severity)
	if !isValidSeverity(analysis.Severity) {
		analysis.Severity = "none"
	}

	// Check for constitutional principle violations (if constitution available)
	violatedPrinciples := ha.CheckPrinciples(response)
	if len(violatedPrinciples) > 0 {
		principleNames := make([]string, 0, len(violatedPrinciples))
		severities := make([]string, 0, len(violatedPrinciples))
		for _, p := range violatedPrinciples {
			principleNames = append(principleNames, p.Name)
			severities = append(severities, p.Severity)
			log.Printf("[HarmAnalyzer] ✓ Constitutional principle violated: %s (severity: %s, keywords: %v)", p.Name, p.Severity, p.CheckKeywords)
		}
		analysis.ViolatedPrinciples = principleNames
		log.Printf("[HarmAnalyzer] Principles violated: %v (severities: %v)", principleNames, severities)

		// If principles violated, might need to escalate intervention
		if len(violatedPrinciples) > 0 && analysis.Intervention == "PROCEED" {
			// Check if any violated principle is critical
			for _, p := range violatedPrinciples {
				if p.Severity == "critical" {
					analysis.Intervention = "WARN"
					log.Printf("[HarmAnalyzer] ⚠️ Escalating to WARN due to critical principle violation: %s", p.Name)
					break
				}
			}
		}
	} else {
		log.Printf("[HarmAnalyzer] ℹ️ No principle violations detected")
	}

	log.Printf("[HarmAnalyzer] ✓ COMPLETE: severity=%s intervention=%s principles_violated=%d affected_parties=%v",
		analysis.Severity, analysis.Intervention, len(violatedPrinciples), analysis.AffectedParties)

	return analysis, nil
}

// buildUserContext creates description of user vulnerabilities for LLM
func (ha *HarmAnalyzer) buildUserContext(vuln *UserVulnerability) string {
	if vuln == nil {
		return "User: No known vulnerabilities"
	}

	parts := []string{"User context:"}

	if vuln.TraumaHistory {
		parts = append(parts, "- Has trauma history (past abuse or major adverse experience)")
	}

	if len(vuln.MentalHealthIssues) > 0 {
		parts = append(parts, fmt.Sprintf("- Mental health concerns: %s", strings.Join(vuln.MentalHealthIssues, ", ")))
	}

	if len(vuln.Patterns) > 0 {
		parts = append(parts, fmt.Sprintf("- Communication patterns: %s", strings.Join(vuln.Patterns, ", ")))
	}

	if vuln.Confidence != "" {
		parts = append(parts, fmt.Sprintf("- Confidence level: %s", vuln.Confidence))
	}

	if vuln.CommunicationStyle != "" {
		parts = append(parts, fmt.Sprintf("- Communication style: %s", vuln.CommunicationStyle))
	}

	return strings.Join(parts, "\n")
}

// buildContactContext creates description of contact for LLM
func (ha *HarmAnalyzer) buildContactContext(traits *ContactTraits) string {
	if traits == nil {
		return "Contact: No information"
	}

	parts := []string{"Contact context:"}

	if traits.Relationship != "" {
		parts = append(parts, fmt.Sprintf("- Relationship: %s", traits.Relationship))
	}

	if traits.Sensitivity != "" {
		parts = append(parts, fmt.Sprintf("- Sensitivity level: %s", traits.Sensitivity))
	}

	if len(traits.Traits) > 0 {
		parts = append(parts, fmt.Sprintf("- Known traits: %s", strings.Join(traits.Traits, ", ")))
	}

	if traits.Stability != "" {
		parts = append(parts, fmt.Sprintf("- Relationship stability: %s", traits.Stability))
	}

	return strings.Join(parts, "\n")
}

// buildHarmAnalysisPrompt creates the analysis prompt
func (ha *HarmAnalyzer) buildHarmAnalysisPrompt(response, userContext, contactContext string) string {
	return fmt.Sprintf(`Analyze if this response could cause harm:

MOLY'S PROPOSED RESPONSE:
"%s"

%s

%s

REASONING:
1. What is this response advising the user to do or say?
2. Who could be hurt by this? (user, contact, third parties)
3. What type of harm? (direct violence, psychological, manipulation, exploitation, relationship damage, other)
4. How likely and severe? (critical, moderate, minor, none)
5. Do the user's vulnerabilities amplify the harm?
6. Is the potential harm worth the benefit to the user?

DECISION:
Based on your analysis, what intervention is needed?
- BLOCK: Critical harm - don't send this response at all
- MODIFY: Moderate harm - soften the language and provide safer alternative
- WARN: Minor harm - send it but include a caution note
- PROCEED: No significant harm - send as-is

Return a JSON response with this exact structure:
{
  "intent": "what is being advised",
  "harm_type": "type of potential harm",
  "severity": "critical|moderate|minor|none",
  "affected_parties": ["who could be hurt"],
  "reasoning": "why this causes harm (describe category, never specific details withheld)",
  "intervention": "BLOCK|MODIFY|WARN|PROCEED",
  "suggested_alternative": "if modifying, what to say instead",
  "explanation": "explanation for the user if intervention needed (describe concern category, never what was withheld)"
}`, response, userContext, contactContext)
}

// isValidSeverity checks if severity is valid
func isValidSeverity(severity string) bool {
	validLevels := map[string]bool{
		"critical": true,
		"moderate": true,
		"minor": true,
		"none": true,
	}
	return validLevels[severity]
}

// ShouldBlock returns true if response should be blocked entirely
func (ha *HarmAnalysis) ShouldBlock() bool {
	return ha.Severity == "critical"
}

// ShouldModify returns true if response should be softened
func (ha *HarmAnalysis) ShouldModify() bool {
	return ha.Severity == "moderate"
}

// ShouldWarn returns true if warning should be added
func (ha *HarmAnalysis) ShouldWarn() bool {
	return ha.Severity == "minor"
}

// CanProceed returns true if no intervention needed
func (ha *HarmAnalysis) CanProceed() bool {
	return ha.Severity == "none"
}
