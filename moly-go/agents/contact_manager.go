package agents

import (
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/models"
)

// ContactManager handles contact lifecycle: create, update, retrieve
type ContactManager struct {
	contactRepo *database.ContactRepository
}

// NewContactManager creates a new contact manager
func NewContactManager(repo *database.ContactRepository) *ContactManager {
	return &ContactManager{
		contactRepo: repo,
	}
}

// CreateContact creates a new contact with user permission
// Typically called after user confirms "Should I save this person?"
func (m *ContactManager) CreateContact(userID string, name string, relationship string, traits []string) (*models.Contact, error) {
	log.Printf("[Moly] ContactManager: CREATE START - name=%s relationship=%s traits=%d user=%s", name, relationship, len(traits), userID)

	if userID == "" || name == "" || relationship == "" {
		log.Printf("[Moly] ContactManager: CREATE FAILED - validation error (missing userId, name, or relationship)")
		return nil, fmt.Errorf("userId, name, and relationship required")
	}

	// Check if contact already exists
	log.Printf("[Moly] ContactManager: checking existence - user=%s name=%s", userID, name)
	existing, err := m.contactRepo.GetByName(userID, name)
	if err != nil {
		log.Printf("[Moly] ContactManager: CREATE FAILED - existence check error: %v", err)
		return nil, err
	}

	if existing != nil {
		log.Printf("[Moly] ContactManager: ✓ contact %s already exists (id=%d status=%s)", name, existing.ID, existing.Status)
		return existing, nil
	}
	log.Printf("[Moly] ContactManager: ✓ contact %s does not exist, creating new", name)

	// Create new contact
	contact := &models.Contact{
		UserID:           userID,
		Name:             name,
		Relationship:     relationship,
		Characteristics:  traits,
		FirstMentionedAt: time.Now().Unix(),
		CreatedVia:       "conversation",
		Status:           "active",
		Version:          1,
		CreatedAt:        time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
	}

	log.Printf("[Moly] ContactManager: preparing to save contact (name=%s rel=%s traits=%v status=active version=1)", contact.Name, contact.Relationship, contact.Characteristics)
	if err := m.contactRepo.Save(contact); err != nil {
		log.Printf("[Moly] ContactManager: ✗ SAVE FAILED - %v (name=%s relationship=%s traits=%v)", err, name, relationship, traits)
		return nil, err
	}
	log.Printf("[Moly] ContactManager: ✓ CREATED contact id=%d name=%s relationship=%s traits=%v user=%s status=active version=%d", contact.ID, contact.Name, contact.Relationship, contact.Characteristics, contact.UserID, contact.Version)
	return contact, nil
}

// GetContact retrieves a contact by name
func (m *ContactManager) GetContact(userID string, name string) (*models.Contact, error) {
	return m.contactRepo.GetByName(userID, name)
}
