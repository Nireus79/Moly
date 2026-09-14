package agents

import (
	"fmt"
	"log"
	"time"

	"moly/database"
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
func (m *ContactManager) CreateContact(userID string, name string, relationship string, traits []string) (*database.Contact, error) {
	log.Printf("[V2] ContactManager: CREATE START - name=%s relationship=%s traits=%d user=%s", name, relationship, len(traits), userID)

	if userID == "" || name == "" || relationship == "" {
		log.Printf("[V2] ContactManager: CREATE FAILED - validation error (missing userId, name, or relationship)")
		return nil, fmt.Errorf("userId, name, and relationship required")
	}

	// Check if contact already exists
	log.Printf("[V2] ContactManager: checking existence - user=%s name=%s", userID, name)
	existing, err := m.contactRepo.GetByName(userID, name)
	if err != nil {
		log.Printf("[V2] ContactManager: CREATE FAILED - existence check error: %v", err)
		return nil, err
	}

	if existing != nil {
		log.Printf("[V2] ContactManager: ✓ contact %s already exists (id=%d status=%s)", name, existing.ID, existing.Status)
		return existing, nil
	}
	log.Printf("[V2] ContactManager: ✓ contact %s does not exist, creating new", name)

	// Create new contact
	contact := &database.Contact{
		UserID:           userID,
		Name:             name,
		Relationship:     relationship,
		Traits:           traits,
		FirstMentionedAt: time.Now().Unix(),
		CreatedVia:       "conversation",
		Status:           "active",
		Version:          1,
		CreatedAt:        time.Now().Unix(),
		UpdatedAt:        time.Now().Unix(),
	}

	log.Printf("[V2] ContactManager: preparing to save contact (name=%s rel=%s traits=%v status=active version=1)", contact.Name, contact.Relationship, contact.Traits)
	if err := m.contactRepo.Save(contact); err != nil {
		log.Printf("[V2] ContactManager: ✗ SAVE FAILED - %v (name=%s relationship=%s traits=%v)", err, name, relationship, traits)
		return nil, err
	}
	log.Printf("[V2] ContactManager: ✓ CREATED contact id=%d name=%s relationship=%s traits=%v user=%s status=active version=%d", contact.ID, contact.Name, contact.Relationship, contact.Traits, contact.UserID, contact.Version)
	return contact, nil
}

// GetContact retrieves a contact by name
func (m *ContactManager) GetContact(userID string, name string) (*database.Contact, error) {
	return m.contactRepo.GetByName(userID, name)
}

// GetContactByID retrieves a contact by ID
func (m *ContactManager) GetContactByID(contactID int64) (*database.Contact, error) {
	return m.contactRepo.GetByID(contactID)
}

// GetUserContacts retrieves all contacts for a user
func (m *ContactManager) GetUserContacts(userID string) ([]*database.Contact, error) {
	return m.contactRepo.GetByUserID(userID)
}

// GetContactsByRelationship retrieves contacts of a specific relationship type
func (m *ContactManager) GetContactsByRelationship(userID string, relationship string) ([]*database.Contact, error) {
	return m.contactRepo.GetByRelationship(userID, relationship)
}

// AddTraitToContact adds a discovered trait to a contact
func (m *ContactManager) AddTraitToContact(contactID int64, trait string) error {
	log.Printf("[V2] ContactManager: adding trait '%s' to contact %d", trait, contactID)

	if trait == "" {
		return fmt.Errorf("trait cannot be empty")
	}

	return m.contactRepo.AddTrait(contactID, trait)
}

// UpdateContactRelationship updates a contact's relationship type
// Used when clarification reveals different relationship than initially thought
func (m *ContactManager) UpdateContactRelationship(contactID int64, newRelationship string) error {
	log.Printf("[V2] ContactManager: updating contact %d relationship to %s", contactID, newRelationship)

	contact, err := m.contactRepo.GetByID(contactID)
	if err != nil {
		return err
	}

	if contact == nil {
		return fmt.Errorf("contact not found")
	}

	if contact.Relationship == newRelationship {
		return nil // No change needed
	}

	oldRelationship := contact.Relationship
	contact.Relationship = newRelationship
	contact.UpdatedAt = time.Now().Unix()

	if err := m.contactRepo.Update(contact); err != nil {
		log.Printf("[V2] ContactManager ERROR: %v", err)
		return err
	}

	log.Printf("[V2] ContactManager: updated contact %s relationship %s → %s", contact.Name, oldRelationship, newRelationship)
	return nil
}

// ArchiveContact soft-deletes a contact
func (m *ContactManager) ArchiveContact(contactID int64) error {
	log.Printf("[V2] ContactManager: archiving contact %d", contactID)
	return m.contactRepo.Delete(contactID)
}

// ContactExists checks if a contact exists for a user
func (m *ContactManager) ContactExists(userID string, name string) (bool, error) {
	contact, err := m.contactRepo.GetByName(userID, name)
	return contact != nil, err
}

// GetContactDisplayName formats a contact for display
func (m *ContactManager) GetContactDisplayName(contact *database.Contact) string {
	if contact == nil {
		return "Unknown"
	}

	if contact.Relationship != "" {
		return fmt.Sprintf("%s (%s)", contact.Name, contact.Relationship)
	}

	return contact.Name
}
