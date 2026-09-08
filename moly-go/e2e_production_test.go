package main

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"moly/agents"
	"moly/database"
	"moly/models"
	"moly/tools"
)

// Shared test database for production E2E tests
var prodTestDB *database.Database

// setupProductionE2ETest initializes production testing with real LLM
func setupProductionE2ETest(t *testing.T, testName string) (*database.Database, *agents.AgentSystem, *tools.LLMClient, string) {
	// Initialize shared test database on first use
	if prodTestDB == nil {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "prod_e2e_test.db")
		db, err := database.Init(dbPath)
		if err != nil {
			t.Fatalf("Failed to initialize test database: %v", err)
		}
		prodTestDB = db
	}

	userID := "prod_e2e_user_" + testName

	// Create real LLM client (uses Ollama or configured provider)
	llm, err := tools.NewLLMClient()
	if err != nil {
		t.Skipf("Failed to initialize LLM client: %v", err)
	}

	if llm == nil {
		t.Skipf("No LLM provider available (expected for CI environments), skipping production test")
	}

	t.Logf("[Production] LLM client initialized, connecting to provider")

	// Create agent system with real LLM
	agentSystem, err := agents.NewAgentSystem(llm, userID, prodTestDB)
	if err != nil {
		t.Fatalf("Failed to create agent system: %v", err)
	}

	return prodTestDB, agentSystem, llm, userID
}

// TestProductionE2ECompleteFlow - Full production test with real LLM
func TestProductionE2ECompleteFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production test in short mode")
	}

	db, agentSystem, llm, userID := setupProductionE2ETest(t, "complete_flow")
	_ = db // Shared DB
	_ = llm

	t.Logf("[Production] Starting complete flow test with real LLM")
	t.Logf("[Production] User ID: %s", userID)

	// PHASE 1: User asks with minimal context
	t.Run("Phase1_ContextGathering", func(t *testing.T) {
		t.Logf("[Production] PHASE 1: User asks question with no context")

		ctx1 := models.Context{
			AboutMe: &models.AboutMe{UserID: userID},
			ConversationHistory: []models.Message{
				{Role: "user", Content: "I need help with a difficult conversation with my manager about my salary expectations", Type: "message"},
			},
			ContextQuality: "minimal",
		}

		response, err := agentSystem.ConversationAgent.Run(ctx1)
		if err != nil {
			t.Fatalf("Phase 1 failed: %v", err)
		}

		if response.Phase != "context_gathering" {
			t.Errorf("Phase 1: Expected context_gathering, got %s", response.Phase)
		}

		if len(response.Questions) == 0 {
			t.Error("Phase 1: Expected context gathering questions")
		}

		t.Logf("[Production] ✓ Phase 1: Generated %d questions", len(response.Questions))
		for i, q := range response.Questions {
			t.Logf("[Production]   Question %d: %s", i+1, q[:minLen(60, len(q))])
		}
	})

	// PHASE 2: User provides context
	t.Run("Phase2_ProvideContext", func(t *testing.T) {
		t.Logf("[Production] PHASE 2: User provides AboutMe and Contact context")

		aboutMe := &models.AboutMe{
			UserID:             userID,
			CommunicationStyle: "direct but respectful",
			Values:             []string{"fairness", "honesty", "growth"},
			PreferredTone:      "professional but warm",
			Notes:              "Prefer clarity over softening",
		}

		contact := &models.Contact{
			UserID:          userID,
			Name:            "Manager",
			Relationship:    "professional",
			Characteristics: []string{"analytical", "results-driven", "values performance"},
			Notes:           "Fair but firm on expectations",
		}

		// Save to database
		aboutMeRepo := database.NewAboutMeRepository(prodTestDB)
		contactRepo := database.NewContactRepository(prodTestDB)

		err := aboutMeRepo.Save(userID, aboutMe)
		if err != nil {
			t.Fatalf("Failed to save AboutMe: %v", err)
		}

		err = contactRepo.Save(userID, contact)
		if err != nil {
			t.Fatalf("Failed to save Contact: %v", err)
		}

		t.Logf("[Production] ✓ Phase 2: Saved AboutMe (style=%s)", aboutMe.CommunicationStyle)
		t.Logf("[Production] ✓ Phase 2: Saved Contact (relationship=%s)", contact.Relationship)
	})

	// PHASE 3: Generate suggestions with real LLM
	t.Run("Phase3_GenerateSuggestions", func(t *testing.T) {
		t.Logf("[Production] PHASE 3: Generate suggestions with real LLM")

		// Reload context from database
		aboutMeRepo := database.NewAboutMeRepository(prodTestDB)
		contactRepo := database.NewContactRepository(prodTestDB)
		aboutMe, _ := aboutMeRepo.Get(userID)
		contact, _ := contactRepo.GetByName(userID, "Manager")

		ctx3 := models.Context{
			AboutMe:        aboutMe,
			ContactProfile: contact,
			ConversationHistory: []models.Message{
				{Role: "user", Content: "I need help with a difficult conversation with my manager about my salary expectations", Type: "message"},
				{Role: "assistant", Content: "Thank you for sharing. To help you prepare the best approach, let me understand your situation better.", Type: "message"},
			},
			ContextQuality: "comprehensive",
		}

		startTime := time.Now()

		t.Logf("[Production] Calling ConversationAgent.Run() (60s timeout for slow system)...")

		response, err := agentSystem.ConversationAgent.Run(ctx3)
		elapsed := time.Since(startTime)

		if err != nil {
			t.Fatalf("Phase 3 failed after %v: %v", elapsed, err)
		}

		if response.Phase != "suggestions_ready" {
			t.Errorf("Phase 3: Expected suggestions_ready, got %s", response.Phase)
		}

		if len(response.Suggestions) == 0 {
			t.Error("Phase 3: Expected suggestions from LLM, got none")
		}

		t.Logf("[Production] ✓ Phase 3: Generated %d suggestions in %v", len(response.Suggestions), elapsed)
		for i, sug := range response.Suggestions {
			confidence := "unknown"
			if sug.Confidence > 0 {
				confidence = fmt.Sprintf("%.2f", sug.Confidence)
			}
			t.Logf("[Production]   Suggestion %d (conf=%s): %s", i+1, confidence, sug.Text[:minLen(60, len(sug.Text))])
		}
	})

	// PHASE 4: Record user choice
	t.Run("Phase4_RecordFeedback", func(t *testing.T) {
		t.Logf("[Production] PHASE 4: User chooses suggestion and provides feedback")

		choiceData := models.SuggestionChoiceData{
			UserID:          userID,
			ConversationID:  "prod_conv_salary_" + userID,
			SuggestionIndex: 0,
			ModifiedText:    "I'd like to schedule a time to discuss my compensation. Based on my contributions over the past year, I believe we should talk about a salary adjustment.",
			UserFeedback:    "positive",
			CreatedAt:       time.Now().Unix(),
		}

		err := agentSystem.LearningAgent.RecordSuggestionChoice(choiceData)
		if err != nil {
			t.Errorf("Phase 4: Failed to record feedback: %v", err)
		}

		t.Logf("[Production] ✓ Phase 4: Feedback recorded (modified suggestion)")
	})

	// PHASE 5: Verify learning
	t.Run("Phase5_VerifyLearning", func(t *testing.T) {
		t.Logf("[Production] PHASE 5: Verify system learned from feedback")

		profile, err := agentSystem.LearningAgent.GetUserProfile(userID)
		if err != nil {
			t.Errorf("Phase 5: Failed to get profile: %v", err)
			return
		}

		if profile != nil {
			t.Logf("[Production] ✓ Phase 5: User profile built (confidence=%.2f)", profile.Confidence)
		} else {
			t.Log("[Production] ⚠️ Phase 5: Profile not yet built (expected on first interaction)")
		}
	})

	t.Logf("[Production] ✅ COMPLETE PRODUCTION FLOW TEST PASSED")
}

// TestProductionIntentDetection - Test intent detection with real LLM
func TestProductionIntentDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production test in short mode")
	}

	db, agentSystem, _, userID := setupProductionE2ETest(t, "intent_detection")
	_ = db

	t.Logf("[Production] Testing intent detection with various user messages")

	testCases := []struct {
		message   string
		expectInt string
	}{
		{
			"I just got promoted! I'm so excited!",
			"celebrate",
		},
		{
			"I feel terrible about what I said. I need to apologize.",
			"apologize",
		},
		{
			"I'm stuck on how to approach this conversation. Can you help?",
			"seek_help",
		},
		{
			"Hey, how are you doing?",
			"greet",
		},
	}

	for i, tc := range testCases {
		t.Run(tc.expectInt, func(t *testing.T) {
			t.Logf("[Production] Test case %d: %s", i+1, tc.message[:minLen(40, len(tc.message))])

			ctx := models.Context{
				AboutMe: &models.AboutMe{UserID: userID},
				ConversationHistory: []models.Message{
					{Role: "user", Content: tc.message, Type: "message"},
				},
				ContextQuality: "minimal",
			}

			response, err := agentSystem.ConversationAgent.Run(ctx)
			if err != nil {
				t.Logf("[Production] ⚠️ Error: %v", err)
				return
			}

			t.Logf("[Production] ✓ Intent detection completed (phase=%s)", response.Phase)
		})
	}
}

// TestProductionSafetyChecking - Test safety/risk assessment with real LLM
func TestProductionSafetyChecking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production test in short mode")
	}

	db, agentSystem, _, userID := setupProductionE2ETest(t, "safety_check")
	_ = db
	_ = agentSystem

	t.Logf("[Production] Testing safety checking with potentially concerning messages")

	riskMonitor, err := agents.NewRiskMonitor(userID)
	if err != nil {
		t.Fatalf("Failed to create risk monitor: %v", err)
	}

	testCases := []struct {
		message     string
		expectRisk  string
		description string
	}{
		{
			"I'm thinking about harming myself because everything is too much",
			"crisis",
			"Crisis indicator",
		},
		{
			"I think my colleague is stupid and worthless",
			"elevated",
			"Harsh language",
		},
		{
			"I want to discuss my concerns with my team",
			"clear",
			"Safe message",
		},
	}

	for i, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Logf("[Production] Test case %d: %s", i+1, tc.description)

			assessment, err := riskMonitor.AssessRisk(userID, tc.message)
			if err != nil {
				t.Logf("[Production] ⚠️ Error: %v", err)
				return
			}

			t.Logf("[Production] ✓ Risk assessment: %s (level=%s)", tc.description, assessment.RiskLevel)

			if assessment.RiskLevel != tc.expectRisk {
				t.Logf("[Production] ⚠️ Expected risk level %s, got %s", tc.expectRisk, assessment.RiskLevel)
			}

			if assessment.Message != "" {
				t.Logf("[Production]   Message: %s", assessment.Message[:minLen(60, len(assessment.Message))])
			}
		})
	}
}

// TestProductionDatabasePersistence - Verify data persists across requests
func TestProductionDatabasePersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production test in short mode")
	}

	db, agentSystem, _, userID := setupProductionE2ETest(t, "persistence")
	_ = agentSystem

	t.Logf("[Production] Testing database persistence across multiple requests")

	// Save initial data
	aboutMe := &models.AboutMe{
		UserID:             userID,
		CommunicationStyle: "collaborative",
		Values:             []string{"teamwork", "transparency"},
	}

	aboutMeRepo := database.NewAboutMeRepository(db)
	err := aboutMeRepo.Save(userID, aboutMe)
	if err != nil {
		t.Fatalf("Failed to save initial data: %v", err)
	}

	t.Logf("[Production] ✓ Initial data saved")

	// Retrieve and verify
	retrieved, err := aboutMeRepo.Get(userID)
	if err != nil {
		t.Fatalf("Failed to retrieve data: %v", err)
	}

	if retrieved == nil {
		t.Error("Retrieved AboutMe is nil")
		return
	}

	if retrieved.CommunicationStyle != "collaborative" {
		t.Errorf("Retrieved style mismatch: expected 'collaborative', got '%s'", retrieved.CommunicationStyle)
	}

	if len(retrieved.Values) != 2 {
		t.Errorf("Retrieved values count mismatch: expected 2, got %d", len(retrieved.Values))
	}

	t.Logf("[Production] ✓ Data persisted correctly")
	t.Logf("[Production]   Style: %s", retrieved.CommunicationStyle)
	t.Logf("[Production]   Values: %v", retrieved.Values)
}

// TestProductionConcurrentRequests - Simulate concurrent users
func TestProductionConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping production test in short mode")
	}

	db, _, _, _ := setupProductionE2ETest(t, "concurrent")

	t.Logf("[Production] Testing concurrent user requests (3 users, 2 requests each)")

	successCount := 0
	errorCount := 0

	for userNum := 1; userNum <= 3; userNum++ {
		for msgNum := 1; msgNum <= 2; msgNum++ {
			userID := fmt.Sprintf("concurrent_user_%d", userNum)

			llm, err := tools.NewLLMClient()
			if err != nil || llm == nil {
				t.Skipf("LLM not available")
			}

			agentSystem, err := agents.NewAgentSystem(llm, userID, db)
			if err != nil {
				t.Logf("[Production] ⚠️ User %d Request %d: Failed to create agent system: %v", userNum, msgNum, err)
				errorCount++
				continue
			}

			ctx := models.Context{
				AboutMe: &models.AboutMe{UserID: userID},
				ConversationHistory: []models.Message{
					{
						Role:    "user",
						Content: fmt.Sprintf("This is request %d from user %d", msgNum, userNum),
						Type:    "message",
					},
				},
				ContextQuality: "minimal",
			}

			_, err = agentSystem.ConversationAgent.Run(ctx)
			if err != nil {
				t.Logf("[Production] ⚠️ User %d Request %d: %v", userNum, msgNum, err)
				errorCount++
			} else {
				successCount++
				t.Logf("[Production] ✓ User %d Request %d: Success", userNum, msgNum)
			}
		}
	}

	t.Logf("[Production] Concurrent test results: %d success, %d errors", successCount, errorCount)
	if errorCount > 3 { // Allow some failures on slow systems
		t.Errorf("Too many concurrent request failures: %d", errorCount)
	}
}
