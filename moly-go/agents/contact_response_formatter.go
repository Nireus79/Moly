package agents

import (
	"fmt"
	"log"
	"moly/database"
	"moly/models"
	"moly/tools"
	"strings"
	"time"
)

// ContactResponseFormatter handles response formatting based on contact resolution status
type ContactResponseFormatter struct {
	db *database.Database
}

// NewContactResponseFormatter creates a new formatter
func NewContactResponseFormatter(db *database.Database) *ContactResponseFormatter {
	return &ContactResponseFormatter{
		db: db,
	}
}

// ContactClarificationResponse represents a clarification request to user
type ContactClarificationResponse struct {
	ID             string                 `json:"id"`
	Question       string                 `json:"question"`
	Options        []string               `json:"options"`
	Context        map[string]interface{} `json:"context"`
	DetectedContacts []*models.Contact    `json:"detectedContacts,omitempty"`
	Confidence     float64                `json:"confidence"`
}

// FormatResponse formats the final response based on contact status and orchestrator results
func (crf *ContactResponseFormatter) FormatResponse(
	layerCtx *tools.LayerContext,
	activeContacts []*models.Contact,
	agentResponse string,
	metadata map[string]interface{},
) map[string]interface{} {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Check if clarification is needed (ambiguous contacts)
	needsClarification := crf.CheckIfClarificationNeeded(activeContacts)

	if needsClarification {
		log.Printf("[ContactResponseFormatter] Contacts are ambiguous, generating clarification response")
		return crf.FormatClarificationResponse(activeContacts, metadata)
	}

	// Contacts are resolved - format normal response with contact context
	log.Printf("[ContactResponseFormatter] Contacts resolved, formatting normal response with context")
	return crf.FormatNormalResponse(agentResponse, activeContacts, metadata)
}

// CheckIfClarificationNeeded determines if we need to ask for clarification
func (crf *ContactResponseFormatter) CheckIfClarificationNeeded(contacts []*models.Contact) bool {
	if len(contacts) == 0 {
		return false
	}

	// Check if any contact has low confidence
	for _, c := range contacts {
		if c.Confidence < 0.50 {
			log.Printf("[ContactResponseFormatter] Contact %s has low confidence (%.2f)", c.Name, c.Confidence)
			return true
		}
	}

	// Check if we have multiple unnamed contacts (ambiguous)
	unnamedCount := 0
	for _, c := range contacts {
		if c.Name == "" || c.Status == "unnamed" {
			unnamedCount++
		}
	}

	if unnamedCount > 1 {
		log.Printf("[ContactResponseFormatter] Multiple unnamed contacts (%d) - ambiguous", unnamedCount)
		return true
	}

	return false
}

// FormatClarificationResponse creates a clarification request
func (crf *ContactResponseFormatter) FormatClarificationResponse(
	contacts []*models.Contact,
	metadata map[string]interface{},
) map[string]interface{} {
	// Generate clarification ID
	clarificationID := fmt.Sprintf("clr_%d", time.Now().UnixNano())

	// Build the clarification question
	question := crf.BuildClarificationQuestion(contacts)

	// Build options (A/B/C format)
	options := crf.BuildClarificationOptions(contacts)

	// Create clarification response
	clarification := map[string]interface{}{
		"id":                 clarificationID,
		"question":           question,
		"options":            options,
		"detectedContacts":   contacts,
		"confidence":         crf.CalculateAverageConfidence(contacts),
		"requiresResolution": true,
	}

	response := map[string]interface{}{
		"status":               "clarification_needed",
		"phase":                "context_gathering",
		"clarification":        clarification,
		"action_required":      true,
		"metadata":             metadata,
	}

	log.Printf("[ContactResponseFormatter] Formatted clarification response (ID: %s, options: %d)", clarificationID, len(options))
	return response
}

// FormatNormalResponse formats a regular response with contact context
func (crf *ContactResponseFormatter) FormatNormalResponse(
	agentResponse string,
	contacts []*models.Contact,
	metadata map[string]interface{},
) map[string]interface{} {
	// Add contact context to the response
	contactContext := crf.BuildContactContext(contacts)

	// Enhance response with contact awareness if needed
	enhancedResponse := crf.EnhanceResponseWithContactContext(agentResponse, contacts)

	response := map[string]interface{}{
		"status":          "success",
		"phase":           "responding",
		"response":        enhancedResponse,
		"contactContext":  contactContext,
		"action_required": false,
		"metadata":        metadata,
	}

	log.Printf("[ContactResponseFormatter] Formatted normal response with %d contacts", len(contacts))
	return response
}

// BuildClarificationQuestion creates a natural language question about ambiguous contacts
func (crf *ContactResponseFormatter) BuildClarificationQuestion(contacts []*models.Contact) string {
	if len(contacts) == 0 {
		return "Who did you mean?"
	}

	if len(contacts) == 1 {
		c := contacts[0]
		if c.Name == "" {
			return fmt.Sprintf("When you mention your %s, who do you mean?", c.Relationship)
		}
		return fmt.Sprintf("Just to clarify, are you talking about %s?", c.Name)
	}

	// Multiple contacts
	names := make([]string, 0)
	for _, c := range contacts {
		if c.Name != "" {
			names = append(names, c.Name)
		} else {
			names = append(names, fmt.Sprintf("your %s", c.Relationship))
		}
	}

	if len(names) == 2 {
		return fmt.Sprintf("When you mention 'she' or 'they', are you talking about %s or %s?", names[0], names[1])
	}

	return fmt.Sprintf("Just to clarify, which one did you mean: %s?", strings.Join(names, ", "))
}

// BuildClarificationOptions creates A/B/C options for clarification
func (crf *ContactResponseFormatter) BuildClarificationOptions(contacts []*models.Contact) []string {
	options := make([]string, 0, len(contacts))

	for i, c := range contacts {
		letter := string(rune('A' + i))

		if c.Name != "" {
			options = append(options, fmt.Sprintf("%s) %s (%s)", letter, c.Name, c.Relationship))
		} else {
			options = append(options, fmt.Sprintf("%s) Your %s", letter, c.Relationship))
		}
	}

	// Add "someone else" option if there are only 2 options
	if len(options) <= 2 {
		letter := string(rune('A' + len(options)))
		options = append(options, fmt.Sprintf("%s) Someone else?", letter))
	}

	return options
}

// BuildContactContext creates a map of contact information for the response
func (crf *ContactResponseFormatter) BuildContactContext(contacts []*models.Contact) map[string]interface{} {
	context := map[string]interface{}{
		"count":    len(contacts),
		"resolved": crf.AreAllContactsNamed(contacts),
		"contacts": make([]map[string]interface{}, 0, len(contacts)),
	}

	for _, c := range contacts {
		contactInfo := map[string]interface{}{
			"id":            c.ID,
			"name":          c.Name,
			"relationship":  c.Relationship,
			"confidence":    c.Confidence,
			"status":        c.Status,
			"pronouns":      c.Pronouns,
		}

		contactSlice := context["contacts"].([]map[string]interface{})
		contactSlice = append(contactSlice, contactInfo)
		context["contacts"] = contactSlice
	}

	return context
}

// EnhanceResponseWithContactContext prefixes the response with contact clarification
func (crf *ContactResponseFormatter) EnhanceResponseWithContactContext(
	response string,
	contacts []*models.Contact,
) string {
	if len(contacts) == 0 || response == "" {
		return response
	}

	// Get named contacts
	namedContacts := make([]*models.Contact, 0)
	for _, c := range contacts {
		if c.Name != "" && c.Status == "named" {
			namedContacts = append(namedContacts, c)
		}
	}

	if len(namedContacts) == 0 {
		return response
	}

	// Build clarification prefix
	var prefix string
	if len(namedContacts) == 1 {
		c := namedContacts[0]
		prefix = fmt.Sprintf("Got it, so your %s %s ", c.Relationship, c.Name)
	} else {
		// Multiple named contacts
		names := make([]string, 0)
		for _, c := range namedContacts {
			names = append(names, fmt.Sprintf("your %s %s", c.Relationship, c.Name))
		}
		prefix = "Got it, so " + strings.Join(names, " and ") + " "
	}

	// Prefix the response
	return prefix + strings.ToLower(string(response[0])) + response[1:]
}

// AreAllContactsNamed checks if all contacts have been named
func (crf *ContactResponseFormatter) AreAllContactsNamed(contacts []*models.Contact) bool {
	if len(contacts) == 0 {
		return true
	}

	for _, c := range contacts {
		if c.Name == "" || c.Status == "unnamed" {
			return false
		}
	}

	return true
}

// CalculateAverageConfidence computes average confidence across contacts
func (crf *ContactResponseFormatter) CalculateAverageConfidence(contacts []*models.Contact) float64 {
	if len(contacts) == 0 {
		return 0.0
	}

	total := 0.0
	for _, c := range contacts {
		total += c.Confidence
	}

	return total / float64(len(contacts))
}

// GetFormattedContactSummary returns a human-readable summary of contacts
func (crf *ContactResponseFormatter) GetFormattedContactSummary(contacts []*models.Contact) string {
	if len(contacts) == 0 {
		return "No contacts"
	}

	summaries := make([]string, 0, len(contacts))
	for _, c := range contacts {
		if c.Name != "" {
			summaries = append(summaries, fmt.Sprintf("%s (%s, %.0f%%)", c.Name, c.Relationship, c.Confidence*100))
		} else {
			summaries = append(summaries, fmt.Sprintf("Unnamed %s (%.0f%%)", c.Relationship, c.Confidence*100))
		}
	}

	return strings.Join(summaries, ", ")
}
