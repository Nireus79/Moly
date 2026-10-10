package tools

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"moly/database"
	"moly/models"
)

// ConstrainedResponseGenerator validates responses against extracted user characteristics
// Phase 3: Ensures response doesn't contradict user facts before returning
type ConstrainedResponseGenerator struct {
	llmClient       LLMProvider
	baseResponseGen *ResponseGenerator
	db              *database.Database
	validator       interface{} // ResponseValidator from agents package
	constraintCache ConstraintCache
}

// Constraint defines a fact-based constraint on response generation
type Constraint struct {
	ID       string
	Type     string // "user_characteristic", "communication_style", "contact_vulnerability"
	Fact     string // "User is dominant"
	Do       string // "Respect their assertiveness"
	Dont     string // "Don't suggest submission"
	Severity string // "low", "medium", "high"
	Source   string // "extracted", "aboutMe", "learned"
}

// NewConstrainedResponseGenerator creates a new constrained generator
func NewConstrainedResponseGenerator(
	llmClient LLMProvider,
	baseGen *ResponseGenerator,
	db *database.Database,
	validator interface{}, // Accept interface{} to avoid circular import
) *ConstrainedResponseGenerator {
	return &ConstrainedResponseGenerator{
		llmClient:       llmClient,
		baseResponseGen: baseGen,
		db:              db,
		validator:       validator,
		constraintCache: NewConstraintCache(1 * time.Hour),
	}
}

// Generate creates a response constrained by user facts
func (crg *ConstrainedResponseGenerator) Generate(
	ctx context.Context,
	userMessage string,
	userProfile *models.AboutMe,
	contacts []models.Contact,
	extractedContext *models.ExtractedContext,
	extractionArtifact *models.ExtractionArtifact,
) (*models.ConversationResponse, error) {

	log.Printf("[ConstrainedGen] Starting constrained response generation")

	// STEP 1: Build constraints from facts
	constraints := crg.buildConstraints(userProfile, contacts, extractedContext, extractionArtifact)
	log.Printf("[ConstrainedGen] Built %d constraints", len(constraints))

	// STEP 2: Build LLM prompt with constraints visible
	systemPrompt := crg.buildConstrainedSystemPrompt(constraints)

	// STEP 3: Generate response WITH constraints in view
	startTime := time.Now()

	// Build LLM call with constraint-aware prompt
	constraintPrompt := crg.buildConstraintLLMPrompt(userMessage, systemPrompt)
	llmReq := &LLMRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   constraintPrompt,
		Temperature:  0,
	}
	llmResp, err := crg.llmClient.Call(ctx, llmReq)
	if err != nil {
		log.Printf("[ConstrainedGen] LLM call failed: %v", err)
		return nil, fmt.Errorf("constraint generation failed: %w", err)
	}
	generationMs := time.Since(startTime).Milliseconds()

	responseText := llmResp.Content
	log.Printf("[ConstrainedGen] Response generated in %dms", generationMs)

	// STEP 4: Validate response against constraints
	violations := crg.validateResponse(responseText, constraints)

	if violations > 0 {
		log.Printf("[ConstrainedGen] ⚠ Response violates %d constraints", violations)

		// Return response with violation metadata
		return &models.ConversationResponse{
			Response: responseText,
			Metadata: map[string]interface{}{
				"responseValidationFailed": true,
				"violationCount":           violations,
				"constraintsApplied":       len(constraints),
				"generationTimeMs":         generationMs,
			},
		}, nil
	}

	// STEP 5: Response is valid, return it
	log.Printf("[ConstrainedGen] ✓ Response validated successfully against %d constraints", len(constraints))

	return &models.ConversationResponse{
		Response: responseText,
		Metadata: map[string]interface{}{
			"constraintsApplied": len(constraints),
			"generationTimeMs":   generationMs,
			"responseValidated":  true,
		},
	}, nil
}

// buildConstraints creates fact-based constraints
func (crg *ConstrainedResponseGenerator) buildConstraints(
	userProfile *models.AboutMe,
	contacts []models.Contact,
	extractedContext *models.ExtractedContext,
	extractionArtifact *models.ExtractionArtifact,
) []Constraint {

	constraints := []Constraint{}

	// Constraint 1: User's extracted characteristics
	if extractionArtifact != nil {
		// Get user entities (filter by subject="user")
		for _, entity := range extractionArtifact.Entities {
			if entity.Subject == "user" && (entity.Type == "property" || entity.Type == "characteristic") {
				// Create constraint from this entity and its antonyms
				if entity.Antonyms != nil && len(entity.Antonyms) > 0 {
					for _, antonym := range entity.Antonyms {
						constraint := Constraint{
							ID:       fmt.Sprintf("extracted_%s_%s", entity.Value, antonym),
							Type:     "user_characteristic",
							Fact:     fmt.Sprintf("User is %s", entity.Value),
							Do:       fmt.Sprintf("Respect their %s nature", entity.Value),
							Dont:     fmt.Sprintf("Don't suggest %s", antonym),
							Severity: "high",
							Source:   "extracted",
						}
						constraints = append(constraints, constraint)
						log.Printf("[ConstrainedGen] Constraint: User=%s, Don't suggest %s", entity.Value, antonym)
					}
				}
			}
		}
	}

	// Constraint 2: Communication style preference
	if userProfile != nil && userProfile.CommunicationStyle != "" {
		constraint := Constraint{
			ID:       fmt.Sprintf("comm_%s", userProfile.CommunicationStyle),
			Type:     "communication_style",
			Fact:     fmt.Sprintf("User prefers %s communication", userProfile.CommunicationStyle),
			Do:       fmt.Sprintf("Use %s communication style", userProfile.CommunicationStyle),
			Dont:     "Don't use opposite communication style",
			Severity: "medium",
			Source:   "aboutMe",
		}
		constraints = append(constraints, constraint)
		log.Printf("[ConstrainedGen] Constraint: Communication style=%s", userProfile.CommunicationStyle)
	}

	// Constraint 3: Contact vulnerabilities
	for _, contact := range contacts {
		// Check if "sensitive" is in characteristics (it's a []string)
		isSensitive := false
		for _, char := range contact.Characteristics {
			if strings.ToLower(char) == "sensitive" {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			constraint := Constraint{
				ID:       fmt.Sprintf("contact_%s_sensitive", contact.Name),
				Type:     "contact_vulnerability",
				Fact:     fmt.Sprintf("%s is sensitive", contact.Name),
				Do:       fmt.Sprintf("Use gentle, caring approach with %s", contact.Name),
				Dont:     fmt.Sprintf("Don't suggest harsh approaches with %s", contact.Name),
				Severity: "high",
				Source:   "aboutMe",
			}
			constraints = append(constraints, constraint)
			log.Printf("[ConstrainedGen] Constraint: Contact %s is sensitive", contact.Name)
		}
	}

	// Constraint 4: Extracted context values
	if extractedContext != nil {
		if extractedContext.UserValues != nil && len(extractedContext.UserValues) > 0 {
			for _, value := range extractedContext.UserValues {
				constraint := Constraint{
					ID:       fmt.Sprintf("value_%s", value),
					Type:     "user_value",
					Fact:     fmt.Sprintf("User values: %s", value),
					Do:       fmt.Sprintf("Align suggestions with %s", value),
					Dont:     fmt.Sprintf("Don't suggest anything conflicting with %s", value),
					Severity: "medium",
					Source:   "extracted",
				}
				constraints = append(constraints, constraint)
			}
		}
	}

	return constraints
}

// buildConstrainedSystemPrompt includes constraints in system prompt
func (crg *ConstrainedResponseGenerator) buildConstrainedSystemPrompt(constraints []Constraint) string {
	prompt := `You are Moly, a thinking partner helping users reason through communication challenges.

IMPORTANT CONSTRAINTS - You MUST respect these facts about the user:
`

	for _, c := range constraints {
		prompt += fmt.Sprintf("\n- %s [%s severity]\n  %s\n  Do not: %s",
			c.Fact, c.Severity, c.Do, c.Dont)
	}

	prompt += `

Generate a response that:
1. Respects all constraints above
2. Helps user think through consequences
3. Is conversational and warm
4. Offers alternatives when appropriate

If you cannot respect a constraint, ask a clarifying question instead of giving advice.
Never suggest something that violates the constraints above.`

	return prompt
}

// buildConstraintLLMPrompt combines user message with system prompt
func (crg *ConstrainedResponseGenerator) buildConstraintLLMPrompt(userMessage, systemPrompt string) string {
	return fmt.Sprintf("User message:\n%s\n\n%s", userMessage, systemPrompt)
}

// validateResponse checks if response violates constraints
func (crg *ConstrainedResponseGenerator) validateResponse(response string, constraints []Constraint) int {
	violations := 0
	lowerResponse := strings.ToLower(response)

	for _, constraint := range constraints {
		// For high-severity constraints, check strictly
		if constraint.Severity == "high" {
			// Check if response contains words that violate constraint
			for _, dontWord := range strings.Split(constraint.Dont, ",") {
				dontWord = strings.TrimSpace(dontWord)
				dontWord = strings.ToLower(dontWord)

				// Extract just the word to avoid (after "Don't suggest")
				parts := strings.Split(dontWord, " ")
				if len(parts) > 0 {
					checkWord := parts[len(parts)-1]

					if crg.containsWord(lowerResponse, checkWord) {
						log.Printf("[ConstrainedGen] ⚠ Violation: Response contains '%s' but constraint is '%s'",
							checkWord, constraint.Fact)
						violations++
						break
					}
				}
			}
		}
	}

	return violations
}

// containsWord checks for whole-word match (not substring)
func (crg *ConstrainedResponseGenerator) containsWord(text, word string) bool {
	pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(word))
	matched, err := regexp.MatchString(pattern, text)
	if err != nil {
		log.Printf("[ConstrainedGen] Regex error: %v", err)
		return false
	}
	return matched
}


