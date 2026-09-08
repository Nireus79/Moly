package agents

import (
	"errors"
	"log"
	"time"

	"moly/database"
	"moly/models"
)

// contextManager - Manages knowledge base (About Me, Contacts, Reflections)
type contextManager struct {
	userID   string
	db       *database.Database
	aboutMeRepo *database.AboutMeRepository
	contactRepo *database.ContactRepository
	interactionRepo *database.InteractionRepository
	reflectionRepo *database.ReflectionRepository
}

// NewContextManager - Create new context manager (no database)
func NewContextManager(userID string) (models.ContextManagerAgent, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	return &contextManager{
		userID: userID,
		db:     nil,
	}, nil
}

// NewContextManagerWithDB - Create new context manager with database access
func NewContextManagerWithDB(userID string, db *database.Database) (models.ContextManagerAgent, error) {
	log.Printf("[ContextManager] Initializing for user %s (DB available: %v)", userID, db != nil)

	if userID == "" {
		log.Printf("[ContextManager] ERROR: userID cannot be empty")
		return nil, errors.New("userID cannot be empty")
	}

	if db == nil {
		log.Printf("[ContextManager] No database available, running in memory-only mode")
		return &contextManager{userID: userID, db: nil}, nil
	}

	log.Printf("[ContextManager] Creating user record in database")
	// Ensure user exists in database
	_ = db.CreateUser(userID)

	log.Printf("[ContextManager] Initialized with full database access")
	return &contextManager{
		userID:          userID,
		db:              db,
		aboutMeRepo:     database.NewAboutMeRepository(db),
		contactRepo:     database.NewContactRepository(db),
		interactionRepo: database.NewInteractionRepository(db),
		reflectionRepo:  database.NewReflectionRepository(db),
	}, nil
}

// GetAboutMe - Retrieve user's About Me profile
func (cm *contextManager) GetAboutMe(userID string) (*models.AboutMe, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	if cm.db == nil || cm.aboutMeRepo == nil {
		return &models.AboutMe{UserID: userID, CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix()}, nil
	}

	aboutMe, err := cm.aboutMeRepo.Get(userID)
	if err != nil {
		return nil, err
	}

	if aboutMe == nil {
		return &models.AboutMe{UserID: userID, CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix()}, nil
	}

	return aboutMe, nil
}

// SetAboutMe - Save/update About Me profile
func (cm *contextManager) SetAboutMe(userID string, aboutMe *models.AboutMe) error {
	log.Printf("[ContextManager] Saving About Me for user %s (style=%s values=%d)",
		userID, aboutMe.CommunicationStyle, len(aboutMe.Values))

	if userID == "" {
		log.Printf("[ContextManager] ERROR: userID cannot be empty")
		return errors.New("userID cannot be empty")
	}

	if aboutMe == nil {
		log.Printf("[ContextManager] ERROR: aboutMe cannot be nil")
		return errors.New("aboutMe cannot be nil")
	}

	if cm.db == nil || cm.aboutMeRepo == nil {
		log.Printf("[ContextManager] No database available, skipping About Me save")
		return nil
	}

	aboutMe.UserID = userID
	aboutMe.UpdatedAt = time.Now().Unix()

	err := cm.aboutMeRepo.Save(userID, aboutMe)
	if err != nil {
		log.Printf("[ContextManager] ERROR saving About Me: %v", err)
	} else {
		log.Printf("[ContextManager] About Me saved successfully")
	}
	return err
}

// GetContact - Retrieve contact information by name
func (cm *contextManager) GetContact(userID, contactName string) (*models.Contact, error) {
	if userID == "" || contactName == "" {
		return nil, errors.New("userID and contactName cannot be empty")
	}

	if cm.db == nil || cm.contactRepo == nil {
		return &models.Contact{UserID: userID, Name: contactName, CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix()}, nil
	}

	contact, err := cm.contactRepo.GetByName(userID, contactName)
	if err != nil {
		return nil, err
	}

	if contact == nil {
		return &models.Contact{UserID: userID, Name: contactName, CreatedAt: time.Now().Unix(), UpdatedAt: time.Now().Unix()}, nil
	}

	return contact, nil
}

// GetContacts - Retrieve all user's contacts
func (cm *contextManager) GetContacts(userID string) ([]models.Contact, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	if cm.db == nil || cm.contactRepo == nil {
		return []models.Contact{}, nil
	}

	contacts, err := cm.contactRepo.GetAll(userID)
	if err != nil {
		return nil, err
	}

	return contacts, nil
}

// CreateContact - Create new contact
func (cm *contextManager) CreateContact(userID string, contact *models.Contact) (*models.Contact, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	if contact == nil {
		return nil, errors.New("contact cannot be nil")
	}

	contact.UserID = userID
	contact.CreatedAt = time.Now().Unix()
	contact.UpdatedAt = time.Now().Unix()

	if cm.db == nil || cm.contactRepo == nil {
		return contact, nil
	}

	err := cm.contactRepo.Save(userID, contact)
	return contact, err
}

// UpdateContact - Update contact information
func (cm *contextManager) UpdateContact(userID, contactName string, updates models.Contact) error {
	if userID == "" || contactName == "" {
		return errors.New("userID and contactName cannot be empty")
	}

	if cm.db == nil || cm.contactRepo == nil {
		return nil
	}

	updates.UserID = userID
	updates.UpdatedAt = time.Now().Unix()

	return cm.contactRepo.Save(userID, &updates)
}

// GetRelevantContext - Retrieve context relevant for a conversation
func (cm *contextManager) GetRelevantContext(conversationID, userID string) (*models.Context, error) {
	if conversationID == "" || userID == "" {
		return nil, errors.New("conversationID and userID cannot be empty")
	}

	// Load AboutMe
	aboutMe, _ := cm.GetAboutMe(userID)

	// Load conversation history
	var history []models.Message
	if cm.db != nil && cm.interactionRepo != nil {
		history, _ = cm.interactionRepo.GetConversation(conversationID, 10)
	}

	// Load contacts
	contacts, _ := cm.GetContacts(userID)
	var contactProfile *models.Contact
	if len(contacts) > 0 {
		contactProfile = &contacts[0]
	}

	// Determine context quality
	contextQuality := "minimal"
	if aboutMe != nil && len(history) > 0 && contactProfile != nil {
		contextQuality = "comprehensive"
	} else if aboutMe != nil || len(history) > 0 || contactProfile != nil {
		contextQuality = "partial"
	}

	context := &models.Context{
		AboutMe:             aboutMe,
		ContactProfile:      contactProfile,
		ConversationHistory: history,
		ContextQuality:      contextQuality,
		Gaps:                []string{},
	}

	return context, nil
}

// SaveReflection - Save extracted reflection (pending approval)
func (cm *contextManager) SaveReflection(conversationID string, reflection *models.Reflection) error {
	if conversationID == "" {
		return errors.New("conversationID cannot be empty")
	}

	if reflection == nil {
		return errors.New("reflection cannot be nil")
	}

	if cm.db == nil || cm.reflectionRepo == nil {
		return nil
	}

	reflection.ConversationID = conversationID
	reflection.Status = "pending_approval"

	return cm.reflectionRepo.Save(cm.userID, reflection)
}

// ApproveReflection - User approves a reflection
func (cm *contextManager) ApproveReflection(conversationID string, reflection *models.Reflection) error {
	if conversationID == "" {
		return errors.New("conversationID cannot be empty")
	}

	if reflection == nil {
		return errors.New("reflection cannot be nil")
	}

	if cm.db == nil || cm.reflectionRepo == nil {
		return nil
	}

	reflection.Status = "approved"
	reflection.ApprovedAt = time.Now().Unix()

	// In a real implementation, we'd update by ID. For now, save as approved.
	return cm.reflectionRepo.Save(cm.userID, reflection)
}

// AppendMessage - Add message to conversation history
func (cm *contextManager) AppendMessage(conversationID string, message *models.Message) error {
	if conversationID == "" {
		return errors.New("conversationID cannot be empty")
	}

	if message == nil {
		return errors.New("message cannot be nil")
	}

	if cm.db == nil || cm.interactionRepo == nil {
		return nil
	}

	metadata := map[string]interface{}{
		"role": message.Role,
		"type": message.Type,
	}

	return cm.interactionRepo.Save(cm.userID, conversationID, message.Content, message.Type, metadata)
}
