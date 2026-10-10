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
	log.Printf("[Moly] ContextAttributeManager: saving %s=%s attributed to %s", factType, factValue, attributedTo)

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
		log.Printf("[Moly] ContextAttributeManager ERROR: %v", err)
		return nil, err
	}

	log.Printf("[Moly] ContextAttributeManager: saved attribute (id=%d)", attr.ID)
	return attr, nil
}
