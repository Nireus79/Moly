package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"moly/database"
	"moly/models"
)

// testDB is shared across database integration tests
var testDB *database.Database

// initTestDB initializes the shared test database
func initTestDB(t *testing.T) *database.Database {
	if testDB != nil {
		return testDB
	}
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.Init(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}
	testDB = db
	return db
}

// TestAboutMeRepository tests complete CRUD cycle
func TestAboutMeRepository(t *testing.T) {
	db := initTestDB(t)

	repo := database.NewAboutMeRepository(db)
	userID := "test_user_001"

	t.Run("SaveAboutMe", func(t *testing.T) {
		aboutMe := &models.AboutMe{
			UserID:            userID,
			CommunicationStyle: "warm",
			Values:            []string{"authenticity", "empathy", "growth"},
			PreferredTone:     "friendly",
			Notes:             "I value genuine conversations",
		}

		err := repo.Save(userID, aboutMe)
		if err != nil {
			t.Errorf("Failed to save AboutMe: %v", err)
		}
	})

	t.Run("GetAboutMe", func(t *testing.T) {
		aboutMe, err := repo.Get(userID)
		if err != nil {
			t.Errorf("Failed to get AboutMe: %v", err)
		}

		if aboutMe == nil {
			t.Error("AboutMe should exist after save")
		}

		if aboutMe.CommunicationStyle != "warm" {
			t.Errorf("Expected 'warm', got %s", aboutMe.CommunicationStyle)
		}

		if len(aboutMe.Values) != 3 {
			t.Errorf("Expected 3 values, got %d", len(aboutMe.Values))
		}
	})

	t.Run("UpdateAboutMe", func(t *testing.T) {
		updated := &models.AboutMe{
			UserID:            userID,
			CommunicationStyle: "direct",
			Values:            []string{"clarity", "honesty"},
			PreferredTone:     "professional",
			Notes:             "Prefer clear communication",
		}

		err := repo.Save(userID, updated)
		if err != nil {
			t.Errorf("Failed to update AboutMe: %v", err)
		}

		retrieved, _ := repo.Get(userID)
		if retrieved.CommunicationStyle != "direct" {
			t.Errorf("Update failed: expected 'direct', got %s", retrieved.CommunicationStyle)
		}
	})

	t.Run("GetNonexistent", func(t *testing.T) {
		result, err := repo.Get("nonexistent_user")
		if err != nil {
			t.Errorf("Should not error for missing user: %v", err)
		}

		if result != nil {
			t.Error("Should return nil for nonexistent user")
		}
	})
}

// TestContactRepository tests contact CRUD
func TestContactRepository(t *testing.T) {
	db := initTestDB(t)

	userID := "test_user_002"

	t.Run("SaveContact", func(t *testing.T) {
		repo := database.NewContactRepository(db)
		contact := &models.Contact{
			UserID:        userID,
			Name:          "Sarah",
			Relationship: "colleague",
			Characteristics: []string{"thoughtful", "creative", "empathetic"},
		}

		err := repo.Save(userID, contact)
		if err != nil {
			t.Errorf("Failed to save contact: %v", err)
		}
	})

	t.Run("GetContactByName", func(t *testing.T) {
		repo := database.NewContactRepository(db)
		contact, err := repo.GetByName(userID, "Sarah")
		if err != nil {
			t.Errorf("Failed to get contact: %v", err)
		}

		if contact == nil {
			t.Error("Contact should exist")
		}

		if contact.Relationship != "colleague" {
			t.Errorf("Expected 'colleague', got %s", contact.Relationship)
		}
	})

	t.Run("GetAllContacts", func(t *testing.T) {
		repo := database.NewContactRepository(db)
		// Add another contact
		contact2 := &models.Contact{
			UserID:        userID,
			Name:          "Mike",
			Relationship: "friend",
			Characteristics: []string{"funny", "loyal"},
		}
		repo.Save(userID, contact2)

		contacts, err := repo.GetAll(userID)
		if err != nil {
			t.Errorf("Failed to get all contacts: %v", err)
		}

		if len(contacts) != 2 {
			t.Errorf("Expected 2 contacts, got %d", len(contacts))
		}
	})

	t.Run("UpdateContact", func(t *testing.T) {
		repo := database.NewContactRepository(db)
		updated := &models.Contact{
			UserID:        userID,
			Name:          "Sarah",
			Relationship: "friend",
			Characteristics: []string{"thoughtful", "creative"},
		}

		err := repo.Save(userID, updated)
		if err != nil {
			t.Errorf("Failed to update contact: %v", err)
		}

		retrieved, _ := repo.GetByName(userID, "Sarah")
		if retrieved.Relationship != "friend" {
			t.Errorf("Update failed: expected 'friend', got %s", retrieved.Relationship)
		}
	})
}

// TestInteractionRepository tests interaction logging
func TestInteractionRepository(t *testing.T) {
	db := initTestDB(t)
	

	userID := "test_user_003"
	conversationID := "conv_001"

	t.Run("SaveInteraction", func(t *testing.T) {
		repo := database.NewInteractionRepository(db)
		metadata := map[string]interface{}{
			"model":    "mistral",
			"latency":  150,
			"provider": "ollama",
		}

		err := repo.Save(userID, conversationID, "How should I talk to Sarah?", "user", metadata)
		if err != nil {
			t.Errorf("Failed to save interaction: %v", err)
		}
	})

	t.Run("GetConversation", func(t *testing.T) {
		repo := database.NewInteractionRepository(db)
		// Add more interactions
		repo.Save(userID, conversationID, "You could be thoughtful and honest", "agent", nil)
		repo.Save(userID, conversationID, "That sounds good", "user", nil)

		messages, err := repo.GetConversation(conversationID, 10)
		if err != nil {
			t.Errorf("Failed to get conversation: %v", err)
		}

		if len(messages) != 3 {
			t.Errorf("Expected 3 messages, got %d", len(messages))
		}
	})

	t.Run("GetConversationWithLimit", func(t *testing.T) {
		repo := database.NewInteractionRepository(db)
		messages, err := repo.GetConversation(conversationID, 2)
		if err != nil {
			t.Errorf("Failed to get conversation with limit: %v", err)
		}

		if len(messages) != 2 {
			t.Errorf("Expected 2 messages (limit), got %d", len(messages))
		}
	})

	t.Run("GetEmptyConversation", func(t *testing.T) {
		repo := database.NewInteractionRepository(db)
		messages, err := repo.GetConversation("nonexistent_conv", 10)
		if err != nil {
			t.Errorf("Should not error: %v", err)
		}

		if len(messages) != 0 {
			t.Errorf("Expected 0 messages, got %d", len(messages))
		}
	})
}

// TestBehaviorPatternRepository tests behavior tracking
func TestBehaviorPatternRepository(t *testing.T) {
	db := initTestDB(t)
	

	userID := "test_user_004"

	t.Run("SaveBehaviorPattern", func(t *testing.T) {
		repo := database.NewBehaviorPatternRepository(db)
		profile := &models.UserBehavioralProfile{
			Confidence: 0.85,
			CommunicationProfile: map[string]interface{}{
				"directness": 0.7,
				"warmth":     0.8,
			},
			CommunicationGoals: map[string]int{
				"clarity": 8,
				"warmth":  7,
			},
		}

		err := repo.Save(userID, profile)
		if err != nil {
			t.Errorf("Failed to save behavior pattern: %v", err)
		}
	})

	t.Run("GetBehaviorPattern", func(t *testing.T) {
		repo := database.NewBehaviorPatternRepository(db)
		profile, err := repo.Get(userID)
		if err != nil {
			t.Errorf("Failed to get behavior pattern: %v", err)
		}

		if profile == nil {
			t.Error("Profile should exist")
		}

		if profile.Confidence != 0.85 {
			t.Errorf("Expected confidence 0.85, got %v", profile.Confidence)
		}
	})
}

// TestReflectionRepository tests reflection storage
func TestReflectionRepository(t *testing.T) {
	db := initTestDB(t)
	

	userID := "test_user_005"

	t.Run("SaveReflection", func(t *testing.T) {
		repo := database.NewReflectionRepository(db)
		reflection := &models.Reflection{
			ConversationID:  "conv_refl_001",
			Characteristics: []string{"empathetic", "thoughtful"},
			Intentions:      []string{"help others", "grow"},
		}

		err := repo.Save(userID, reflection)
		if err != nil {
			t.Errorf("Failed to save reflection: %v", err)
		}
	})

	t.Run("GetPendingApprovals", func(t *testing.T) {
		repo := database.NewReflectionRepository(db)
		// Add another pending reflection
		reflection2 := &models.Reflection{
			ConversationID:  "conv_refl_002",
			Characteristics: []string{"curious"},
			Intentions:      []string{"learn"},
		}
		repo.Save(userID, reflection2)

		reflections, err := repo.GetPendingApprovals(userID)
		if err != nil {
			t.Errorf("Failed to get pending approvals: %v", err)
		}

		if len(reflections) != 2 {
			t.Errorf("Expected 2 pending reflections, got %d", len(reflections))
		}
	})

	t.Run("ApproveReflection", func(t *testing.T) {
		repo := database.NewReflectionRepository(db)
		reflections, _ := repo.GetPendingApprovals(userID)
		if len(reflections) == 0 {
			t.Fatal("No reflections to approve")
		}

		// Approve first reflection (ID would be 1)
		err := repo.Approve(1)
		if err != nil {
			t.Errorf("Failed to approve reflection: %v", err)
		}
	})
}

// TestSuggestionChoiceRepository tests suggestion tracking
func TestSuggestionChoiceRepository(t *testing.T) {
	db := initTestDB(t)
	

	userID := "test_user_006"

	t.Run("RecordSuggestionChoice", func(t *testing.T) {
		repo := database.NewSuggestionChoiceRepository(db)
		err := repo.Record(userID, "sug_001", "Be more direct", "I'll try to be clearer")
		if err != nil {
			t.Errorf("Failed to record choice: %v", err)
		}
	})

	t.Run("GetUserChoices", func(t *testing.T) {
		repo := database.NewSuggestionChoiceRepository(db)
		// Add another choice
		repo.Record(userID, "sug_002", "Ask more questions", "")

		choices, err := repo.GetUserChoices(userID, 10)
		if err != nil {
			t.Errorf("Failed to get choices: %v", err)
		}

		if len(choices) != 2 {
			t.Errorf("Expected 2 choices, got %d", len(choices))
		}

		// Verify structure
		choice := choices[0]
		if choice["suggestion_id"] == "" {
			t.Error("Missing suggestion_id in choice")
		}
	})

	t.Run("GetUserChoicesWithLimit", func(t *testing.T) {
		repo := database.NewSuggestionChoiceRepository(db)
		// Add more choices
		for i := 3; i <= 5; i++ {
			repo.Record(userID, "sug_00"+string(rune(48+i)), "Suggestion "+string(rune(48+i)), "")
		}

		choices, err := repo.GetUserChoices(userID, 2)
		if err != nil {
			t.Errorf("Failed to get choices with limit: %v", err)
		}

		if len(choices) != 2 {
			t.Errorf("Expected 2 choices (limit), got %d", len(choices))
		}
	})
}

// TestSafetyIncidentRepository tests safety tracking
func TestSafetyIncidentRepository(t *testing.T) {
	db := initTestDB(t)
	

	userID := "test_user_007"

	t.Run("RecordIncident", func(t *testing.T) {
		repo := database.NewSafetyIncidentRepository(db)
		err := repo.Record(userID, "medium", "Concerning language detected", "llm")
		if err != nil {
			t.Errorf("Failed to record incident: %v", err)
		}
	})

	t.Run("GetRecentIncidents", func(t *testing.T) {
		repo := database.NewSafetyIncidentRepository(db)
		// Add another incident
		repo.Record(userID, "low", "Keyword match", "heuristic")
		repo.Record(userID, "high", "Crisis language", "llm")

		incidents, err := repo.GetRecentIncidents(userID, 10)
		if err != nil {
			t.Errorf("Failed to get incidents: %v", err)
		}

		if len(incidents) != 3 {
			t.Errorf("Expected 3 incidents, got %d", len(incidents))
		}

		// Verify structure
		incident := incidents[0]
		if incident["severity"] == "" {
			t.Error("Missing severity in incident")
		}
	})

	t.Run("GetRecentIncidentsWithLimit", func(t *testing.T) {
		repo := database.NewSafetyIncidentRepository(db)
		incidents, err := repo.GetRecentIncidents(userID, 2)
		if err != nil {
			t.Errorf("Failed to get incidents with limit: %v", err)
		}

		if len(incidents) != 2 {
			t.Errorf("Expected 2 incidents (limit), got %d", len(incidents))
		}
	})

	t.Run("VerifyIncidentStructure", func(t *testing.T) {
		repo := database.NewSafetyIncidentRepository(db)
		incidents, _ := repo.GetRecentIncidents(userID, 1)
		if len(incidents) == 0 {
			t.Fatal("No incidents to verify")
		}

		incident := incidents[0]
		requiredFields := []string{"severity", "content", "detected_at", "detected_by"}
		for _, field := range requiredFields {
			if _, exists := incident[field]; !exists {
				t.Errorf("Missing field: %s", field)
			}
		}
	})
}

// TestCrossRepositoryIntegration tests workflow across repositories
func TestCrossRepositoryIntegration(t *testing.T) {
	db := initTestDB(t)
	

	userID := "integration_user"

	t.Run("CompleteUserSetup", func(t *testing.T) {
		// 1. Save user profile
		aboutMeRepo := database.NewAboutMeRepository(db)
		aboutMe := &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: "authentic",
			Values:            []string{"honesty"},
			PreferredTone:     "warm",
		}
		aboutMeRepo.Save(userID, aboutMe)

		// 2. Add contacts
		contactRepo := database.NewContactRepository(db)
		contact := &models.Contact{
			UserID:        userID,
			Name:          "Mom",
			Relationship: "family",
		}
		contactRepo.Save(userID, contact)

		// 3. Log interaction
		interactionRepo := database.NewInteractionRepository(db)
		interactionRepo.Save(userID, "conv1", "How should I talk to mom?", "user", nil)

		// 4. Record behavior pattern
		behaviorRepo := database.NewBehaviorPatternRepository(db)
		profile := &models.UserBehavioralProfile{
			Confidence: 0.7,
		}
		behaviorRepo.Save(userID, profile)

		// 5. Track suggestion choice
		suggestionRepo := database.NewSuggestionChoiceRepository(db)
		suggestionRepo.Record(userID, "sug1", "Be honest", "I'll be honest")

		// Verify everything was saved
		retrieved, _ := aboutMeRepo.Get(userID)
		if retrieved == nil {
			t.Error("AboutMe not saved")
		}

		contacts, _ := contactRepo.GetAll(userID)
		if len(contacts) != 1 {
			t.Error("Contact not saved")
		}

		messages, _ := interactionRepo.GetConversation("conv1", 10)
		if len(messages) != 1 {
			t.Error("Interaction not saved")
		}

		choices, _ := suggestionRepo.GetUserChoices(userID, 10)
		if len(choices) != 1 {
			t.Error("Suggestion choice not saved")
		}
	})

	t.Run("ConversationFlow", func(t *testing.T) {
		interactionRepo := database.NewInteractionRepository(db)
		contactRepo := database.NewContactRepository(db)

		conversationID := "conv_flow_test"

		// User asks
		interactionRepo.Save(userID, conversationID, "I'm worried about the argument", "user", nil)

		// Get contact context
		contact, _ := contactRepo.GetByName(userID, "Mom")
		if contact == nil {
			t.Error("Should have retrieved contact context")
		}

		// Agent responds
		interactionRepo.Save(userID, conversationID, "Remember her values", "agent", nil)

		// Verify conversation thread
		messages, _ := interactionRepo.GetConversation(conversationID, 10)
		if len(messages) != 2 {
			t.Errorf("Expected 2 messages in thread, got %d", len(messages))
		}
	})
}

// TestJSONSerialization tests that complex types serialize/deserialize correctly
func TestJSONSerialization(t *testing.T) {
	db := initTestDB(t)
	

	t.Run("AboutMeValues", func(t *testing.T) {
		repo := database.NewAboutMeRepository(db)
		userID := "json_test_user"

		values := []string{"authenticity", "growth", "connection", "learning"}
		aboutMe := &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: "warm",
			Values:            values,
			PreferredTone:     "friendly",
		}

		repo.Save(userID, aboutMe)
		retrieved, _ := repo.Get(userID)

		// Verify values deserialized correctly
		if len(retrieved.Values) != 4 {
			t.Errorf("Expected 4 values, got %d", len(retrieved.Values))
		}

		valuesJSON, _ := json.Marshal(values)
		retrievedJSON, _ := json.Marshal(retrieved.Values)

		if string(valuesJSON) != string(retrievedJSON) {
			t.Error("Values JSON not preserved correctly")
		}
	})

	t.Run("ContactCharacteristics", func(t *testing.T) {
		repo := database.NewContactRepository(db)
		userID := "json_test_user_2"

		characteristics := []string{"ambitious", "kind", "thoughtful", "funny"}
		contact := &models.Contact{
			UserID:        userID,
			Name:          "Friend",
			Relationship: "friend",
			Characteristics: characteristics,
		}

		repo.Save(userID, contact)
		retrieved, _ := repo.GetByName(userID, "Friend")

		if len(retrieved.Characteristics) != 4 {
			t.Errorf("Expected 4 characteristics, got %d", len(retrieved.Characteristics))
		}
	})

	t.Run("InteractionMetadata", func(t *testing.T) {
		repo := database.NewInteractionRepository(db)
		userID := "json_test_user_3"
		conversationID := "json_conv"

		metadata := map[string]interface{}{
			"model":    "mistral",
			"tokens":   250,
			"latency":  150,
			"provider": "ollama",
			"confidence": 0.92,
		}

		repo.Save(userID, conversationID, "Test message", "user", metadata)

		// Verify it was stored (we can't retrieve it with current API, but at least verify no error)
		messages, err := repo.GetConversation(conversationID, 10)
		if err != nil {
			t.Errorf("Failed to retrieve conversation: %v", err)
		}

		if len(messages) != 1 {
			t.Error("Message not stored correctly")
		}
	})
}

// TestDatabaseConcurrency tests that multiple operations work together
func TestDatabaseConcurrency(t *testing.T) {
	db := initTestDB(t)
	

	t.Run("MultipleRepositoriesSimultaneously", func(t *testing.T) {
		aboutMeRepo := database.NewAboutMeRepository(db)
		contactRepo := database.NewContactRepository(db)
		interactionRepo := database.NewInteractionRepository(db)

		userID := "concurrent_user"
		conversationID := "concurrent_conv_" + t.Name()

		// Save multiple things
		aboutMeRepo.Save(userID, &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: "direct",
		})

		contactRepo.Save(userID, &models.Contact{
			UserID:        userID,
			Name:          "Contact1",
			Relationship: "work",
		})

		interactionRepo.Save(userID, conversationID, "Message 1", "user", nil)

		// Retrieve all
		aboutMe, _ := aboutMeRepo.Get(userID)
		contacts, _ := contactRepo.GetAll(userID)
		messages, _ := interactionRepo.GetConversation(conversationID, 10)

		if aboutMe == nil {
			t.Error("AboutMe not retrieved")
		}

		if len(contacts) < 1 {
			t.Error("Contact not retrieved")
		}

		if len(messages) < 1 {
			t.Error("Interaction not retrieved")
		}
	})

	t.Run("ConsistentAcrossReads", func(t *testing.T) {
		repo := database.NewAboutMeRepository(db)
		userID := "consistent_user"

		aboutMe := &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: "balanced",
			Values:            []string{"honesty"},
		}

		repo.Save(userID, aboutMe)

		// Read multiple times
		for i := 0; i < 3; i++ {
			retrieved, _ := repo.Get(userID)
			if retrieved.CommunicationStyle != "balanced" {
				t.Errorf("Iteration %d: Expected 'balanced', got %s", i, retrieved.CommunicationStyle)
			}
		}
	})
}
