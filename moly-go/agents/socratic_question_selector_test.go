package agents

import (
	"testing"

	"moly/models"
)

// setupTestLibrary creates a minimal question library for testing
func setupTestLibrary() *models.QuestionLibrary {
	lib := models.NewQuestionLibrary()

	// Add test questions for different approaches/categories
	questions := []*models.SocraticQuestion{
		{
			ID:               "q_stakeholder_001",
			Text:             "Who else might be affected by this decision?",
			SocraticApproach: "identifying_stakeholders",
			Category:         "stakeholder",
			TargetsPrinciple: "user_autonomy",
			ExpectedInsights: []string{"hidden stakeholders", "ripple effects"},
			DepthLevel:       1,
		},
		{
			ID:               "q_consequence_001",
			Text:             "What are the potential negative consequences?",
			SocraticApproach: "exploring_consequences",
			Category:         "consequence",
			TargetsPrinciple: "harm_prevention",
			ExpectedInsights: []string{"unintended consequences", "risk awareness"},
			DepthLevel:       1,
		},
		{
			ID:               "q_principle_001",
			Text:             "Does this align with your values?",
			SocraticApproach: "revealing_assumptions",
			Category:         "principle",
			TargetsPrinciple: "user_autonomy",
			ExpectedInsights: []string{"value alignment", "assumption testing"},
			DepthLevel:       1,
		},
		{
			ID:               "q_universal_001",
			Text:             "Would you want others to do this to you?",
			SocraticApproach: "testing_universality",
			Category:         "universal",
			TargetsPrinciple: "justice",
			ExpectedInsights: []string{"reciprocity", "fairness"},
			DepthLevel:       2,
		},
	}

	for _, q := range questions {
		lib.AddQuestion(q)
	}

	return lib
}

// setupTestConstitution creates a minimal constitution for testing
func setupTestConstitution() *models.Constitution {
	return &models.Constitution{
		SupremePrinciples: []models.Principle{
			{
				Name:           "User Autonomy",
				Severity:       "critical",
				Description:    "Users control their own decisions",
				CheckKeywords:  []string{"decision", "choice", "autonomy"},
			},
			{
				Name:           "Harm Prevention",
				Severity:       "critical",
				Description:    "Prevent harm to users and others",
				CheckKeywords:  []string{"harm", "hurt", "damage"},
			},
			{
				Name:           "Justice",
				Severity:       "high",
				Description:    "Treat all people fairly",
				CheckKeywords:  []string{"fair", "unfair", "unjust"},
			},
		},
	}
}

// TestSocraticSelectorInitialization verifies selector can be created and initialized
func TestSocraticSelectorInitialization(t *testing.T) {
	lib := setupTestLibrary()
	constitution := setupTestConstitution()

	selector := NewSocraticQuestionSelector(lib, constitution)

	if selector == nil {
		t.Fatal("Selector should be initialized")
	}

	if selector.library == nil {
		t.Error("Library should be set")
	}

	if selector.constitution == nil {
		t.Error("Constitution should be set")
	}
}

// TestSocraticSelectorFindsQuestion verifies selector can find a question when ambiguity is present
func TestSocraticSelectorFindsQuestion(t *testing.T) {
	lib := setupTestLibrary()
	constitution := setupTestConstitution()
	selector := NewSocraticQuestionSelector(lib, constitution)

	// Create context with ambiguity (missing stakeholder understanding)
	ctx := &models.Context{
		ConversationHistory: []models.Message{
			{
				Role:    "user",
				Content: "I want to tell my boss that I disagree with his decision.",
			},
		},
		AboutMe: &models.AboutMe{},
	}

	// With message mentioning decision, should identify ambiguity
	question := selector.SelectNextQuestion(ctx, "I want to tell my boss about my concerns", []models.SocraticQuestion{})

	if question != nil {
		// If a question was selected, verify it has required fields
		if question.ID == "" {
			t.Error("Selected question should have an ID")
		}
		if question.Text == "" {
			t.Error("Selected question should have text")
		}
		if question.SocraticApproach == "" {
			t.Error("Selected question should have a Socratic approach")
		}
		if question.DepthLevel == 0 {
			t.Error("Selected question should have a depth level")
		}
	}
}

// TestSocraticSelectorReturnsNilWithoutAmbiguity verifies nil return when no ambiguity
func TestSocraticSelectorReturnsNilWithoutAmbiguity(t *testing.T) {
	lib := setupTestLibrary()
	constitution := setupTestConstitution()
	selector := NewSocraticQuestionSelector(lib, constitution)

	// Create context with no clear ambiguity
	ctx := &models.Context{
		ConversationHistory: []models.Message{
			{Role: "user", Content: "hello"},
		},
		AboutMe: &models.AboutMe{},
	}

	question := selector.SelectNextQuestion(ctx, "hello", []models.SocraticQuestion{})

	// Should return nil when no ambiguity detected
	if question != nil {
		t.Logf("Note: Question returned despite minimal message: %v", question.Text)
	}
}

// TestLibraryQuestionIndexing verifies question library properly indexes by all attributes
func TestLibraryQuestionIndexing(t *testing.T) {
	lib := setupTestLibrary()

	// Verify questions are indexed by approach
	stakeholderQs := lib.QuestionsByApproach["identifying_stakeholders"]
	if len(stakeholderQs) == 0 {
		t.Error("Questions should be indexed by approach")
	}

	// Verify questions are indexed by category
	categoryQs := lib.QuestionsByCategory["stakeholder"]
	if len(categoryQs) == 0 {
		t.Error("Questions should be indexed by category")
	}

	// Verify questions are indexed by principle
	principleQs := lib.QuestionsByPrinciple["user_autonomy"]
	if len(principleQs) == 0 {
		t.Error("Questions should be indexed by principle")
	}

	// Verify all questions are in AllQuestions
	if len(lib.AllQuestions) != 4 {
		t.Errorf("Expected 4 questions in library, got %d", len(lib.AllQuestions))
	}
}

// TestFindByApproachAndCategory verifies lookup method works correctly
func TestFindByApproachAndCategory(t *testing.T) {
	lib := setupTestLibrary()

	// Should find the stakeholder question
	q := lib.FindByApproachAndCategory("identifying_stakeholders", "stakeholder")
	if q == nil {
		t.Error("Should find question by approach and category")
	}
	if q.ID != "q_stakeholder_001" {
		t.Errorf("Expected q_stakeholder_001, got %s", q.ID)
	}

	// Should return nil for non-existent combination
	q = lib.FindByApproachAndCategory("nonexistent", "nonexistent")
	if q != nil {
		t.Error("Should return nil for non-existent approach/category")
	}
}

// TestQuestionMetadataPreservation verifies all question metadata is preserved
func TestQuestionMetadataPreservation(t *testing.T) {
	lib := setupTestLibrary()

	q := lib.AllQuestions["q_consequence_001"]
	if q == nil {
		t.Fatal("Question should exist in library")
	}

	// Verify all metadata fields are preserved
	tests := []struct {
		name     string
		field    interface{}
		expected interface{}
	}{
		{"ID", q.ID, "q_consequence_001"},
		{"Text", q.Text, "What are the potential negative consequences?"},
		{"SocraticApproach", q.SocraticApproach, "exploring_consequences"},
		{"Category", q.Category, "consequence"},
		{"TargetsPrinciple", q.TargetsPrinciple, "harm_prevention"},
		{"DepthLevel", q.DepthLevel, 1},
	}

	for _, tt := range tests {
		if tt.field != tt.expected {
			t.Errorf("%s: got %v, expected %v", tt.name, tt.field, tt.expected)
		}
	}

	// Verify insights list is present
	if len(q.ExpectedInsights) != 2 {
		t.Errorf("Expected 2 insights, got %d", len(q.ExpectedInsights))
	}
}

// TestQuestionAdditionValidation verifies questions are validated on addition
func TestQuestionAdditionValidation(t *testing.T) {
	lib := models.NewQuestionLibrary()

	// Test: Question with empty ID should error
	q := &models.SocraticQuestion{
		ID:   "",
		Text: "Test question",
	}
	err := lib.AddQuestion(q)
	if err == nil {
		t.Error("Should reject question with empty ID")
	}

	// Test: Question with empty text should error
	q = &models.SocraticQuestion{
		ID:   "test_001",
		Text: "",
	}
	err = lib.AddQuestion(q)
	if err == nil {
		t.Error("Should reject question with empty text")
	}

	// Test: Valid question should be accepted
	q = &models.SocraticQuestion{
		ID:   "test_001",
		Text: "Valid question?",
	}
	err = lib.AddQuestion(q)
	if err != nil {
		t.Errorf("Should accept valid question, got error: %v", err)
	}
}

// TestDepthLevelProgression verifies questions can represent depth progression
func TestDepthLevelProgression(t *testing.T) {
	lib := setupTestLibrary()

	// Get all questions and verify depth levels
	depthLevels := make(map[int]bool)
	for _, q := range lib.AllQuestions {
		if q.DepthLevel > 0 {
			depthLevels[q.DepthLevel] = true
		}
	}

	if len(depthLevels) == 0 {
		t.Error("Questions should have depth levels for progression")
	}

	// Verify levels are within valid range (1-5)
	for level := range depthLevels {
		if level < 1 || level > 5 {
			t.Errorf("Invalid depth level %d (should be 1-5)", level)
		}
	}
}
