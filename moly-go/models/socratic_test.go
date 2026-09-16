package models

import (
	"testing"
)

func TestQuestionLibraryAddQuestion(t *testing.T) {
	lib := NewQuestionLibrary()

	q := &SocraticQuestion{
		ID:               "test_q_001",
		Text:             "What is your goal?",
		SocraticApproach: "identifying_stakeholders",
		Category:         "stakeholder",
		TargetsPrinciple: "user_autonomy",
		TargetsFramework: "rights_based",
		ExpectedInsights: []string{"reveals goal"},
		DepthLevel:       1,
	}

	err := lib.AddQuestion(q)
	if err != nil {
		t.Fatalf("Failed to add question: %v", err)
	}

	if lib.GetSize() != 1 {
		t.Fatalf("Expected 1 question, got %d", lib.GetSize())
	}
}

func TestQuestionLibraryAddQuestionValidation(t *testing.T) {
	lib := NewQuestionLibrary()

	// Test missing ID
	q := &SocraticQuestion{
		Text:             "What is your goal?",
		SocraticApproach: "identifying_stakeholders",
	}

	err := lib.AddQuestion(q)
	if err == nil {
		t.Fatal("Expected validation error for missing ID")
	}
}

func TestQuestionLibraryFindByID(t *testing.T) {
	lib := NewQuestionLibrary()

	q := &SocraticQuestion{
		ID:               "test_q_001",
		Text:             "What is your goal?",
		SocraticApproach: "identifying_stakeholders",
		Category:         "stakeholder",
		TargetsPrinciple: "user_autonomy",
		TargetsFramework: "rights_based",
	}

	lib.AddQuestion(q)

	found := lib.FindByID("test_q_001")
	if found == nil {
		t.Fatal("Question not found by ID")
	}

	if found.Text != "What is your goal?" {
		t.Fatalf("Expected 'What is your goal?', got '%s'", found.Text)
	}
}

func TestQuestionLibraryFindByApproach(t *testing.T) {
	lib := NewQuestionLibrary()

	q1 := &SocraticQuestion{
		ID:               "q_001",
		Text:             "Q1",
		SocraticApproach: "identifying_stakeholders",
		Category:         "stakeholder",
		TargetsPrinciple: "user_autonomy",
		TargetsFramework: "rights_based",
	}

	q2 := &SocraticQuestion{
		ID:               "q_002",
		Text:             "Q2",
		SocraticApproach: "exploring_consequences",
		Category:         "consequence",
		TargetsPrinciple: "stakeholder_consideration",
		TargetsFramework: "utilitarian",
	}

	lib.AddQuestion(q1)
	lib.AddQuestion(q2)

	found := lib.FindByApproach("identifying_stakeholders")
	if len(found) != 1 {
		t.Fatalf("Expected 1 question, got %d", len(found))
	}

	if found[0].ID != "q_001" {
		t.Fatalf("Expected q_001, got %s", found[0].ID)
	}
}

func TestQuestionLibraryFindByCategory(t *testing.T) {
	lib := NewQuestionLibrary()

	q := &SocraticQuestion{
		ID:               "q_001",
		Text:             "Q1",
		SocraticApproach: "identifying_stakeholders",
		Category:         "stakeholder",
		TargetsPrinciple: "user_autonomy",
		TargetsFramework: "rights_based",
	}

	lib.AddQuestion(q)

	found := lib.FindByCategory("stakeholder")
	if len(found) != 1 {
		t.Fatalf("Expected 1 question, got %d", len(found))
	}
}

func TestQuestionLibraryFindByApproachAndCategory(t *testing.T) {
	lib := NewQuestionLibrary()

	q := &SocraticQuestion{
		ID:               "q_001",
		Text:             "Q1",
		SocraticApproach: "identifying_stakeholders",
		Category:         "stakeholder",
		TargetsPrinciple: "user_autonomy",
		TargetsFramework: "rights_based",
	}

	lib.AddQuestion(q)

	found := lib.FindByApproachAndCategory("identifying_stakeholders", "stakeholder")
	if found == nil {
		t.Fatal("Question not found")
	}

	if found.ID != "q_001" {
		t.Fatalf("Expected q_001, got %s", found.ID)
	}
}

func TestGetApproaches(t *testing.T) {
	lib := NewQuestionLibrary()

	q1 := &SocraticQuestion{
		ID:               "q_001",
		Text:             "Q1",
		SocraticApproach: "identifying_stakeholders",
		Category:         "stakeholder",
		TargetsPrinciple: "user_autonomy",
		TargetsFramework: "rights_based",
	}

	q2 := &SocraticQuestion{
		ID:               "q_002",
		Text:             "Q2",
		SocraticApproach: "exploring_consequences",
		Category:         "consequence",
		TargetsPrinciple: "stakeholder_consideration",
		TargetsFramework: "utilitarian",
	}

	lib.AddQuestion(q1)
	lib.AddQuestion(q2)

	approaches := lib.GetApproaches()
	if len(approaches) != 2 {
		t.Fatalf("Expected 2 approaches, got %d", len(approaches))
	}
}

