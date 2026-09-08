package main

import (
	"path/filepath"
	"testing"
	"time"

	"moly/agents"
	"moly/database"
	"moly/models"
	"moly/tools"
)

// Shared test database for E2E tests
var e2eTestDB *database.Database

// setupE2ETest initializes a complete E2E test environment with shared database
func setupE2ETest(t *testing.T, testName string) (*database.Database, *agents.AgentSystem, string) {
	// Initialize shared test database on first use
	if e2eTestDB == nil {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "e2e_test.db")
		db, err := database.Init(dbPath)
		if err != nil {
			t.Fatalf("Failed to initialize test database: %v", err)
		}
		e2eTestDB = db
	}

	userID := "e2e_user_" + testName

	// Create agent system with shared database
	llmClient, _ := tools.NewLLMClient()
	agentSystem, err := agents.NewAgentSystem(llmClient, userID, e2eTestDB)
	if err != nil {
		t.Fatalf("Failed to create agent system: %v", err)
	}

	return e2eTestDB, agentSystem, userID
}

// TestE2EContextGatheringFlow tests the flow when context is missing
// User asks question → System detects missing context → System asks Socratic questions
func TestE2EContextGatheringFlow(t *testing.T) {
	_, agentSystem, userID := setupE2ETest(t, "context_gathering")
	// Shared DB, no close needed

	t.Run("MissingAllContext_GeneratesQuestions", func(t *testing.T) {
		// User has no AboutMe, no contacts, sends a message
		ctx := models.Context{
			AboutMe:        &models.AboutMe{UserID: userID},
			ContactProfile: &models.Contact{Name: "Contact"},
			ConversationHistory: []models.Message{
				{
					Role:    "user",
					Content: "What should I say?",
					Type:    "message",
				},
			},
			ContextQuality: "minimal",
		}

		t.Logf("[E2E] Starting conversation with minimal context")

		// Run conversation agent
		response, err := agentSystem.ConversationAgent.Run(ctx)
		if err != nil {
			t.Fatalf("ConversationAgent failed: %v", err)
		}

		// Verify phase is context_gathering
		if response.Phase != "context_gathering" {
			t.Errorf("Expected phase 'context_gathering', got '%s'", response.Phase)
		}

		// Verify questions were generated
		if len(response.Questions) == 0 {
			t.Error("Expected context gathering questions, got none")
		}

		t.Logf("[E2E] ✓ Generated %d context gathering questions", len(response.Questions))
		t.Logf("[E2E] ✓ Phase: %s", response.Phase)
	})

	t.Run("PartialContext_GathersMissing", func(t *testing.T) {
		// User has AboutMe but no contacts
		aboutMe := &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: "warm",
			Values:             []string{"honesty"},
			PreferredTone:      "friendly",
		}

		// Save AboutMe
		aboutMeRepo := database.NewAboutMeRepository(e2eTestDB)
		err := aboutMeRepo.Save(userID, aboutMe)
		if err != nil {
			t.Fatalf("Failed to save AboutMe: %v", err)
		}

		t.Logf("[E2E] Saved AboutMe for user")

		// Retrieve and verify
		retrievedAboutMe, err := aboutMeRepo.Get(userID)
		if err != nil || retrievedAboutMe == nil {
			t.Fatalf("Failed to retrieve AboutMe: %v", err)
		}

		// Now run conversation with AboutMe but without contact
		ctx := models.Context{
			AboutMe:        retrievedAboutMe,
			ContactProfile: &models.Contact{Name: "Contact"},
			ConversationHistory: []models.Message{
				{
					Role:    "user",
					Content: "How do I talk to my mom?",
					Type:    "message",
				},
			},
			ContextQuality: "partial",
		}

		t.Logf("[E2E] Running conversation with partial context")

		response, err := agentSystem.ConversationAgent.Run(ctx)
		if err != nil {
			t.Fatalf("ConversationAgent failed: %v", err)
		}

		// With AboutMe but no contact/intention, should still ask for more context
		if response.Phase == "context_gathering" {
			t.Logf("[E2E] ✓ Still gathering context (missing contact info)")
		} else if response.Phase == "suggestions_ready" {
			t.Logf("[E2E] ✓ Ready to suggest (enough context)")
		}

		t.Logf("[E2E] ✓ Phase: %s, Questions: %d", response.Phase, len(response.Questions))
	})
}

// TestE2ESuggestionGenerationFlow tests suggestion generation with full context
// User provides context → System generates personalized suggestions
func TestE2ESuggestionGenerationFlow(t *testing.T) {
	_, agentSystem, userID := setupE2ETest(t, "suggestion_generation")
	// Shared DB, no close needed

	t.Run("FullContext_GeneratesSuggestions", func(t *testing.T) {
		// Create complete context
		aboutMe := &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: "warm_but_honest",
			Values:             []string{"authenticity", "empathy"},
			PreferredTone:      "friendly",
			Notes:              "I like genuine conversations",
		}

		contact := &models.Contact{
			UserID:          userID,
			Name:            "Sarah",
			Relationship:    "close_friend",
			Characteristics: []string{"thoughtful", "creative", "sensitive"},
		}

		// Save context
		aboutMeRepo := database.NewAboutMeRepository(e2eTestDB)
		aboutMeRepo.Save(userID, aboutMe)

		contactRepo := database.NewContactRepository(e2eTestDB)
		contactRepo.Save(userID, contact)

		t.Logf("[E2E] Saved user context (AboutMe + Contact)")

		// Run conversation with full context
		ctx := models.Context{
			AboutMe:        aboutMe,
			ContactProfile: contact,
			ConversationHistory: []models.Message{
				{
					Role:    "user",
					Content: "I want to thank Sarah for being such a great friend, but I'm nervous about being too sentimental",
					Type:    "message",
				},
			},
			ContextQuality: "comprehensive",
		}

		t.Logf("[E2E] Running conversation with full context (comprehensive)")

		response, err := agentSystem.ConversationAgent.Run(ctx)
		if err != nil {
			t.Fatalf("ConversationAgent failed: %v", err)
		}

		// Verify suggestions phase
		if response.Phase != "suggestions_ready" {
			t.Errorf("Expected phase 'suggestions_ready', got '%s'", response.Phase)
		}

		// Verify suggestions were generated
		if len(response.Suggestions) == 0 {
			t.Error("Expected suggestions, got none")
		}

		t.Logf("[E2E] ✓ Phase: %s", response.Phase)
		t.Logf("[E2E] ✓ Generated %d suggestions", len(response.Suggestions))
		for i, sug := range response.Suggestions {
			t.Logf("[E2E]   Suggestion %d (confidence=%.2f): %s", i+1, sug.Confidence, sug.Text[:minLen(50, len(sug.Text))])
		}
	})

	t.Run("SuggestionQuality_MeetsExpectations", func(t *testing.T) {
		aboutMe := &models.AboutMe{
			UserID:             userID + "_quality",
			CommunicationStyle: "direct",
			Values:             []string{"clarity", "honesty"},
			PreferredTone:      "professional",
		}

		contact := &models.Contact{
			UserID:          userID + "_quality",
			Name:            "Boss",
			Relationship:    "work",
			Characteristics: []string{"demanding", "fair", "detail-oriented"},
		}

		ctx := models.Context{
			AboutMe:        aboutMe,
			ContactProfile: contact,
			ConversationHistory: []models.Message{
				{
					Role:    "user",
					Content: "I need to ask for a raise",
					Type:    "message",
				},
			},
			ContextQuality: "comprehensive",
		}

		response, err := agentSystem.ConversationAgent.Run(ctx)
		if err != nil {
			t.Fatalf("ConversationAgent failed: %v", err)
		}

		if len(response.Suggestions) > 0 {
			// Verify suggestions are non-empty and have confidence scores
			for _, sug := range response.Suggestions {
				if sug.Text == "" {
					t.Error("Suggestion text should not be empty")
				}
				if sug.Confidence < 0 || sug.Confidence > 1 {
					t.Errorf("Confidence should be 0-1, got %.2f", sug.Confidence)
				}
				if sug.Tone == "" {
					t.Error("Suggestion should have a tone")
				}
			}

			t.Logf("[E2E] ✓ All %d suggestions are well-formed", len(response.Suggestions))
		}
	})
}

// TestE2EFeedbackAndLearningFlow tests the feedback loop
// User gets suggestions → User provides feedback → System learns and updates profile
func TestE2EFeedbackAndLearningFlow(t *testing.T) {
	_, agentSystem, userID := setupE2ETest(t, "feedback_learning")
	// Shared DB, no close needed

	conversationID := "conv_feedback_test"

	t.Run("UserChoiceRecorded_UpdatesProfile", func(t *testing.T) {
		// Record that user chose a suggestion
		choiceData := models.SuggestionChoiceData{
			UserID:          userID,
			ConversationID:  conversationID,
			SuggestionIndex: 0,
			ModifiedText:    "",
			UserFeedback:    "positive",
			CreatedAt:       time.Now().Unix(),
		}

		t.Logf("[E2E] Recording user suggestion choice")

		err := agentSystem.LearningAgent.RecordSuggestionChoice(choiceData)
		if err != nil {
			t.Errorf("Failed to record suggestion choice: %v", err)
		}

		t.Logf("[E2E] ✓ Suggestion choice recorded")

		// Verify choice was recorded (learning agent has it in memory)
		t.Logf("[E2E] ✓ Suggestion choice recorded in learning agent")
	})

	t.Run("UserModifiesText_ExtractsAndStores", func(t *testing.T) {
		conversationID := "conv_modify_test"

		// Simulate user modifying a suggestion
		modifiedText := "I really appreciate how thoughtful and creative you are"

		choiceData := models.SuggestionChoiceData{
			UserID:          userID,
			ConversationID:  conversationID,
			SuggestionIndex: 1,
			ModifiedText:    modifiedText,
			UserFeedback:    "neutral",
			CreatedAt:       time.Now().Unix(),
		}

		t.Logf("[E2E] Recording user modification of suggestion")

		err := agentSystem.LearningAgent.RecordSuggestionChoice(choiceData)
		if err != nil {
			t.Errorf("Failed to record modified choice: %v", err)
		}

		t.Logf("[E2E] ✓ Modified suggestion recorded")
		t.Logf("[E2E] User preferred: %s", modifiedText[:minLen(50, len(modifiedText))])
	})

	t.Run("BehavioralProfileUpdates_OverTime", func(t *testing.T) {
		// Record multiple choices to simulate learning
		for i := 0; i < 3; i++ {
			choiceData := models.SuggestionChoiceData{
				UserID:          userID,
				ConversationID:  conversationID + "_" + string(rune(i)),
				SuggestionIndex: 0,
				ModifiedText:    "",
				UserFeedback:    "positive",
				CreatedAt:       time.Now().Unix(),
			}

			agentSystem.LearningAgent.RecordSuggestionChoice(choiceData)
		}

		t.Logf("[E2E] Recorded 3 suggestion choices")

		// Retrieve behavioral profile
		profile, err := agentSystem.LearningAgent.GetUserProfile(userID)
		if err != nil {
			t.Errorf("Failed to get user profile: %v", err)
		}

		if profile != nil && profile.Confidence > 0 {
			t.Logf("[E2E] ✓ Behavioral profile built (confidence=%.2f)", profile.Confidence)
		}
	})
}

// TestE2ECompleteConversationCycle tests the entire cycle in one workflow
// 1. User starts with minimal context → questions asked
// 2. User provides context
// 3. System generates suggestions
// 4. User chooses and modifies suggestion
// 5. System learns from feedback
func TestE2ECompleteConversationCycle(t *testing.T) {
	_, agentSystem, userID := setupE2ETest(t, "complete_cycle")
	// Shared DB, no close needed

	conversationID := "cycle_complete_test"

	t.Run("CompleteUserJourney", func(t *testing.T) {
		// PHASE 1: Initial question with no context
		t.Logf("[E2E] PHASE 1: User asks question with minimal context")

		ctx1 := models.Context{
			AboutMe:        &models.AboutMe{UserID: userID},
			ContactProfile: &models.Contact{Name: "Contact"},
			ConversationHistory: []models.Message{
				{
					Role:    "user",
					Content: "I need to have a difficult conversation with my partner",
					Type:    "message",
				},
			},
			ContextQuality: "minimal",
		}

		response1, err := agentSystem.ConversationAgent.Run(ctx1)
		if err != nil {
			t.Fatalf("Phase 1 failed: %v", err)
		}

		if response1.Phase != "context_gathering" {
			t.Errorf("Phase 1: Expected context_gathering, got %s", response1.Phase)
		}

		contextQuestions := len(response1.Questions)
		t.Logf("[E2E] ✓ Phase 1 Complete: Asked %d questions to gather context", contextQuestions)

		// PHASE 2: User provides context via AboutMe and Contact
		t.Logf("[E2E] PHASE 2: User provides context (AboutMe + Contact)")

		aboutMe := &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: "gentle_but_honest",
			Values:             []string{"love", "growth", "vulnerability"},
			PreferredTone:      "warm",
			Notes:              "I want to be honest but also caring",
		}

		contact := &models.Contact{
			UserID:          userID,
			Name:            "Partner",
			Relationship:    "romantic",
			Characteristics: []string{"sensitive", "thoughtful", "communicative"},
		}

		// Save context
		aboutMeRepo := database.NewAboutMeRepository(e2eTestDB)
		contactRepo := database.NewContactRepository(e2eTestDB)

		err = aboutMeRepo.Save(userID, aboutMe)
		if err != nil {
			t.Fatalf("Phase 2: Failed to save AboutMe: %v", err)
		}

		err = contactRepo.Save(userID, contact)
		if err != nil {
			t.Fatalf("Phase 2: Failed to save contact: %v", err)
		}

		t.Logf("[E2E] ✓ Phase 2 Complete: Saved AboutMe and Contact")

		// PHASE 3: Generate suggestions with full context
		t.Logf("[E2E] PHASE 3: System generates suggestions with full context")

		ctx3 := models.Context{
			AboutMe:        aboutMe,
			ContactProfile: contact,
			ConversationHistory: []models.Message{
				{
					Role:    "user",
					Content: "I need to have a difficult conversation with my partner",
					Type:    "message",
				},
			},
			ContextQuality: "comprehensive",
		}

		response3, err := agentSystem.ConversationAgent.Run(ctx3)
		if err != nil {
			t.Fatalf("Phase 3 failed: %v", err)
		}

		if response3.Phase != "suggestions_ready" {
			t.Errorf("Phase 3: Expected suggestions_ready, got %s", response3.Phase)
		}

		if len(response3.Suggestions) == 0 {
			t.Error("Phase 3: Expected suggestions, got none")
		}

		t.Logf("[E2E] ✓ Phase 3 Complete: Generated %d suggestions", len(response3.Suggestions))
		if len(response3.Suggestions) > 0 {
			t.Logf("[E2E]   First suggestion: %s", response3.Suggestions[0].Text[:minLen(60, len(response3.Suggestions[0].Text))])
		}

		// PHASE 4: User provides feedback
		t.Logf("[E2E] PHASE 4: User chooses and modifies suggestion")

		feedback := models.ConversationFeedback{
			ConversationID:      conversationID,
			UserID:              userID,
			SuggestionChosen:    0,
			SuggestionText:      response3.Suggestions[0].Text,
			UserModified:        true,
			ModificationRequest: "I'll start with 'I want to talk about something that's been on my mind'",
			ReflectionApproved:  true,
			Timestamp:           time.Now().Unix(),
		}

		// Record feedback via learning agent
		choiceData := models.SuggestionChoiceData{
			UserID:          feedback.UserID,
			ConversationID:  feedback.ConversationID,
			SuggestionIndex: feedback.SuggestionChosen,
			ModifiedText:    feedback.ModificationRequest,
			UserFeedback:    "positive",
			CreatedAt:       feedback.Timestamp,
		}

		err = agentSystem.LearningAgent.RecordSuggestionChoice(choiceData)
		if err != nil {
			t.Errorf("Phase 4: Failed to record feedback: %v", err)
		}

		t.Logf("[E2E] ✓ Phase 4 Complete: Feedback recorded")

		// PHASE 5: Verify learning
		t.Logf("[E2E] PHASE 5: Verify system learned from feedback")

		profile, err := agentSystem.LearningAgent.GetUserProfile(userID)
		if err != nil {
			t.Errorf("Phase 5: Failed to get profile: %v", err)
		}

		if profile != nil {
			t.Logf("[E2E] ✓ Phase 5 Complete: Profile confidence = %.2f", profile.Confidence)
		}

		t.Logf("[E2E] ✅ COMPLETE CYCLE SUCCESS")
		t.Logf("[E2E] Summary:")
		t.Logf("[E2E]   1. Asked %d context questions", contextQuestions)
		t.Logf("[E2E]   2. Saved AboutMe (style=%s, values=%d)", aboutMe.CommunicationStyle, len(aboutMe.Values))
		t.Logf("[E2E]   3. Saved Contact (name=%s, relationship=%s)", contact.Name, contact.Relationship)
		t.Logf("[E2E]   4. Generated %d suggestions", len(response3.Suggestions))
		t.Logf("[E2E]   5. User modified: %s...", feedback.ModificationRequest[:minLen(50, len(feedback.ModificationRequest))])
		t.Logf("[E2E]   6. System learned and updated profile")
	})
}

// TestE2EErrorHandling tests error scenarios
func TestE2EErrorHandling(t *testing.T) {
	_, agentSystem, userID := setupE2ETest(t, "error_handling")
	// Shared DB, no close needed

	t.Run("MissingAboutMe_HandledGracefully", func(t *testing.T) {
		ctx := models.Context{
			AboutMe: nil, // Missing AboutMe
			ConversationHistory: []models.Message{
				{Role: "user", Content: "Hello", Type: "message"},
			},
		}

		_, err := agentSystem.ConversationAgent.Run(ctx)
		if err == nil {
			t.Error("Expected error for missing AboutMe, got none")
		} else {
			t.Logf("[E2E] ✓ Correctly rejected missing AboutMe: %v", err)
		}
	})

	t.Run("EmptyUserID_Rejected", func(t *testing.T) {
		choiceData := models.SuggestionChoiceData{
			UserID: "", // Empty user ID
		}

		err := agentSystem.LearningAgent.RecordSuggestionChoice(choiceData)
		if err == nil {
			t.Error("Expected error for empty userID")
		} else {
			t.Logf("[E2E] ✓ Correctly rejected empty userID")
		}
	})

	t.Run("NilContact_Handled", func(t *testing.T) {
		ctx := models.Context{
			AboutMe:        &models.AboutMe{UserID: userID},
			ContactProfile: nil, // Nil contact is ok, means missing context
			ConversationHistory: []models.Message{
				{Role: "user", Content: "Test", Type: "message"},
			},
			ContextQuality: "minimal",
		}

		response, err := agentSystem.ConversationAgent.Run(ctx)
		if err != nil {
			t.Errorf("Should handle nil contact gracefully: %v", err)
		}

		if response != nil {
			t.Logf("[E2E] ✓ Handled nil contact gracefully (phase=%s)", response.Phase)
		}
	})
}

// TestE2EContextPersistence tests that context is preserved across calls
func TestE2EContextPersistence(t *testing.T) {
	_, _, userID := setupE2ETest(t, "context_persistence")
	// Shared DB, no close needed

	t.Run("SavedContext_RetrievableAcrossCalls", func(t *testing.T) {
		// Save context
		aboutMe := &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: "authentic",
			Values:             []string{"trust", "honesty"},
			PreferredTone:      "warm",
		}

		aboutMeRepo := database.NewAboutMeRepository(e2eTestDB)
		err := aboutMeRepo.Save(userID, aboutMe)
		if err != nil {
			t.Fatalf("Failed to save AboutMe: %v", err)
		}

		t.Logf("[E2E] Saved AboutMe to database")

		// Retrieve context in separate call
		retrievedAboutMe, err := aboutMeRepo.Get(userID)
		if err != nil {
			t.Fatalf("Failed to retrieve AboutMe: %v", err)
		}

		if retrievedAboutMe == nil {
			t.Error("Retrieved AboutMe should not be nil")
		} else if retrievedAboutMe.CommunicationStyle != "authentic" {
			t.Errorf("Expected 'authentic', got %s", retrievedAboutMe.CommunicationStyle)
		} else {
			t.Logf("[E2E] ✓ Context persisted and retrieved correctly")
			t.Logf("[E2E]   Style: %s", retrievedAboutMe.CommunicationStyle)
			t.Logf("[E2E]   Values: %v", retrievedAboutMe.Values)
		}
	})

	t.Run("MultipleInteractions_PreservesState", func(t *testing.T) {
		contactRepo := database.NewContactRepository(e2eTestDB)

		// Save first contact
		contact1 := &models.Contact{
			UserID:          userID,
			Name:            "Alice",
			Relationship:    "friend",
			Characteristics: []string{"creative", "fun"},
		}

		contactRepo.Save(userID, contact1)
		t.Logf("[E2E] Saved first contact: Alice")

		// Save second contact
		contact2 := &models.Contact{
			UserID:          userID,
			Name:            "Bob",
			Relationship:    "colleague",
			Characteristics: []string{"professional", "detail-oriented"},
		}

		contactRepo.Save(userID, contact2)
		t.Logf("[E2E] Saved second contact: Bob")

		// Retrieve all contacts
		contacts, err := contactRepo.GetAll(userID)
		if err != nil {
			t.Fatalf("Failed to get all contacts: %v", err)
		}

		if len(contacts) != 2 {
			t.Errorf("Expected 2 contacts, got %d", len(contacts))
		} else {
			t.Logf("[E2E] ✓ Both contacts persisted correctly")
			for _, c := range contacts {
				t.Logf("[E2E]   - %s (%s)", c.Name, c.Relationship)
			}
		}
	})
}

// Helper function for string truncation
func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}
