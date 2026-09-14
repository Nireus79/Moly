package agents

import (
	"fmt"
	"log"

	"moly/database"
)

// ContextAttributeManager stores extracted facts with WHO attribution
type ContextAttributeManager struct {
	attrRepo *database.ContextAttributeRepository
}

// NewContextAttributeManager creates a new manager
func NewContextAttributeManager(repo *database.ContextAttributeRepository) *ContextAttributeManager {
	return &ContextAttributeManager{
		attrRepo: repo,
	}
}

// SaveAttribute stores an extracted fact with attribution
// fact: what was learned (e.g., "casual")
// factType: type of fact (e.g., "style")
// attributedTo: who it's about (e.g., "user" or "contact_manager_sarah")
// context: context where applicable (e.g., "work", "general")
// evidence: quote from message
// confidence: 0-1 score
func (m *ContextAttributeManager) SaveAttribute(
	userID string,
	conversationID string,
	factType string,
	factValue string,
	attributedTo string,
	context string,
	evidence string,
	confidence float64,
) (*database.ContextAttribute, error) {
	log.Printf("[V2] ContextAttributeManager: saving %s=%s attributed to %s", factType, factValue, attributedTo)

	if userID == "" || factValue == "" || attributedTo == "" {
		return nil, fmt.Errorf("userId, factValue, and attributedTo required")
	}

	attr := &database.ContextAttribute{
		UserID:         userID,
		ConversationID: conversationID,
		FactType:       factType,
		FactValue:      factValue,
		AttributedTo:   attributedTo,
		Context:        context,
		Evidence:       evidence,
		Confidence:     confidence,
		Source:         "explicit",
	}

	if err := m.attrRepo.Save(attr); err != nil {
		log.Printf("[V2] ContextAttributeManager ERROR: %v", err)
		return nil, err
	}

	log.Printf("[V2] ContextAttributeManager: saved attribute (id=%d)", attr.ID)
	return attr, nil
}

// GetContextForSubject retrieves all known facts about a subject
func (m *ContextAttributeManager) GetContextForSubject(userID string, subject string) (map[string][]string, error) {
	log.Printf("[V2] ContextAttributeManager: getting context for %s", subject)

	attributes, err := m.attrRepo.GetForSubject(userID, subject)
	if err != nil {
		return nil, err
	}

	// Group by fact type
	grouped := make(map[string][]string)
	for _, attr := range attributes {
		grouped[attr.FactType] = append(grouped[attr.FactType], attr.FactValue)
	}

	log.Printf("[V2] ContextAttributeManager: found %d attributes for %s", len(attributes), subject)
	return grouped, nil
}

// GetAllUserContext retrieves full context for a user (all subjects)
func (m *ContextAttributeManager) GetAllUserContext(userID string) (map[string]interface{}, error) {
	log.Printf("[V2] ContextAttributeManager: getting all context for user %s", userID)

	attributes, err := m.attrRepo.GetUserAttributes(userID)
	if err != nil {
		return nil, err
	}

	// Group by subject, then by fact type
	result := make(map[string]interface{})
	bySubject := make(map[string][]*database.ContextAttribute)

	for _, attr := range attributes {
		bySubject[attr.AttributedTo] = append(bySubject[attr.AttributedTo], attr)
	}

	// Transform for easier use
	for subject, attrs := range bySubject {
		grouped := make(map[string][]string)
		for _, attr := range attrs {
			grouped[attr.FactType] = append(grouped[attr.FactType], attr.FactValue)
		}
		result[subject] = grouped
	}

	log.Printf("[V2] ContextAttributeManager: found context for %d subjects", len(bySubject))
	return result, nil
}

// GetStyleForSubject retrieves communication style for a subject
func (m *ContextAttributeManager) GetStyleForSubject(userID string, subject string) ([]string, error) {
	attributes, err := m.attrRepo.GetByType(userID, subject, "style")
	if err != nil {
		return nil, err
	}

	styles := []string{}
	for _, attr := range attributes {
		styles = append(styles, attr.FactValue)
	}

	return styles, nil
}

// GetTraitsForSubject retrieves traits for a subject
func (m *ContextAttributeManager) GetTraitsForSubject(userID string, subject string) ([]string, error) {
	attributes, err := m.attrRepo.GetByType(userID, subject, "trait")
	if err != nil {
		return nil, err
	}

	traits := []string{}
	for _, attr := range attributes {
		traits = append(traits, attr.FactValue)
	}

	return traits, nil
}

// GetValuesForSubject retrieves values for a subject
func (m *ContextAttributeManager) GetValuesForSubject(userID string, subject string) ([]string, error) {
	attributes, err := m.attrRepo.GetByType(userID, subject, "value")
	if err != nil {
		return nil, err
	}

	values := []string{}
	for _, attr := range attributes {
		values = append(values, attr.FactValue)
	}

	return values, nil
}

// HasAttribute checks if a subject already has a specific attribute
func (m *ContextAttributeManager) HasAttribute(userID string, subject string, factType string, factValue string) (bool, error) {
	attributes, err := m.attrRepo.GetByType(userID, subject, factType)
	if err != nil {
		return false, err
	}

	for _, attr := range attributes {
		if attr.FactValue == factValue {
			return true, nil
		}
	}

	return false, nil
}

// GetConversationContext retrieves context learned in a specific conversation
func (m *ContextAttributeManager) GetConversationContext(conversationID string) ([]*database.ContextAttribute, error) {
	return m.attrRepo.GetForConversation(conversationID)
}
