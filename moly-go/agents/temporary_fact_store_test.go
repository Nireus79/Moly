package agents

import (
	"testing"
)

func TestTemporaryFactStore_Store(t *testing.T) {
	store := NewTemporaryFactStore()

	fact := &TemporaryFact{
		FactID:            "fact_1",
		FactType:          "trait",
		FactValue:         "organized",
		AttributedTo:      "user",
		LinkedQuestionIDs: []string{"q1", "q2"},
	}

	err := store.Store(fact)
	if err != nil {
		t.Errorf("Store() error = %v", err)
	}

	// Verify it's stored
	retrieved, _ := store.Get("fact_1")
	if retrieved == nil {
		t.Error("Store() did not persist fact")
	}
	if retrieved.FactValue != "organized" {
		t.Errorf("Store() got value %s, want organized", retrieved.FactValue)
	}
}

func TestTemporaryFactStore_RecordAnswer(t *testing.T) {
	store := NewTemporaryFactStore()
	fact := &TemporaryFact{
		FactID:            "fact_1",
		FactType:          "trait",
		FactValue:         "organized",
		AttributedTo:      "user",
		LinkedQuestionIDs: []string{"q1", "q2"},
	}
	store.Store(fact)

	// Record answer to first question
	err := store.RecordAnswer("fact_1", "q1", "At work")
	if err != nil {
		t.Errorf("RecordAnswer() error = %v", err)
	}

	// Verify answer recorded
	retrieved, _ := store.Get("fact_1")
	if retrieved.ClarificationAnswers["q1"] != "At work" {
		t.Errorf("RecordAnswer() got %s, want 'At work'", retrieved.ClarificationAnswers["q1"])
	}

	// Verify not complete yet (only 1/2 answered)
	if store.IsComplete("fact_1") {
		t.Error("IsComplete() returned true with only 1/2 questions answered")
	}

	// Record answer to second question
	store.RecordAnswer("fact_1", "q2", "Very consistent")
	if !store.IsComplete("fact_1") {
		t.Error("IsComplete() returned false with all questions answered")
	}
}

func TestTemporaryFactStore_RemainingQuestions(t *testing.T) {
	store := NewTemporaryFactStore()
	fact := &TemporaryFact{
		FactID:            "fact_1",
		FactType:          "trait",
		FactValue:         "organized",
		LinkedQuestionIDs: []string{"q1", "q2", "q3"},
	}
	store.Store(fact)

	// Initially all 3 pending
	remaining := store.RemainingQuestions("fact_1")
	if len(remaining) != 3 {
		t.Errorf("RemainingQuestions() got %d, want 3", len(remaining))
	}

	// Answer one
	store.RecordAnswer("fact_1", "q1", "answer1")
	remaining = store.RemainingQuestions("fact_1")
	if len(remaining) != 2 {
		t.Errorf("RemainingQuestions() got %d after 1 answer, want 2", len(remaining))
	}

	// Answer another
	store.RecordAnswer("fact_1", "q2", "answer2")
	remaining = store.RemainingQuestions("fact_1")
	if len(remaining) != 1 {
		t.Errorf("RemainingQuestions() got %d after 2 answers, want 1", len(remaining))
	}
}

func TestTemporaryFactStore_GetBySubject(t *testing.T) {
	store := NewTemporaryFactStore()

	// Store facts for different subjects
	fact1 := &TemporaryFact{FactID: "fact_1", AttributedTo: "user", FactValue: "organized"}
	fact2 := &TemporaryFact{FactID: "fact_2", AttributedTo: "user", FactValue: "casual"}
	fact3 := &TemporaryFact{FactID: "fact_3", AttributedTo: "contact_bob", FactValue: "smart"}

	store.Store(fact1)
	store.Store(fact2)
	store.Store(fact3)

	// Get facts about user
	userFacts := store.GetBySubject("user")
	if len(userFacts) != 2 {
		t.Errorf("GetBySubject(user) got %d facts, want 2", len(userFacts))
	}

	// Get facts about contact
	contactFacts := store.GetBySubject("contact_bob")
	if len(contactFacts) != 1 {
		t.Errorf("GetBySubject(contact_bob) got %d facts, want 1", len(contactFacts))
	}
}

func TestTemporaryFactStore_Remove(t *testing.T) {
	store := NewTemporaryFactStore()
	fact := &TemporaryFact{
		FactID:       "fact_1",
		FactValue:    "organized",
		AttributedTo: "user",
	}
	store.Store(fact)

	// Verify it exists
	_, err := store.Get("fact_1")
	if err != nil {
		t.Error("Get() should not error before removal")
	}

	// Remove it
	err = store.Remove("fact_1")
	if err != nil {
		t.Errorf("Remove() error = %v", err)
	}

	// Verify it's gone
	_, err = store.Get("fact_1")
	if err == nil {
		t.Error("Get() should error after removal")
	}
}

func TestClarificationEngine_GenerateUserContextClarification(t *testing.T) {
	engine := NewClarificationEngine()
	fact := ExtractedFact{
		ID:       "fact_1",
		FactType: "trait",
		Value:    "organized",
		Evidence: "I'm very organized",
	}

	questions := engine.GenerateUserContextClarification(fact)

	if len(questions) != 3 {
		t.Errorf("GenerateUserContextClarification() got %d questions, want 3", len(questions))
	}

	// Check first question is about context
	if questions[0].Type != "user_context" {
		t.Errorf("First question type = %s, want user_context", questions[0].Type)
	}

	// Check questions mention the trait
	if len(questions[0].Options) == 0 {
		t.Error("Questions should have options")
	}
}

func TestClarificationEngine_GenerateContactContextClarification(t *testing.T) {
	engine := NewClarificationEngine()
	fact := ExtractedFact{
		ID:       "fact_1",
		FactType: "trait",
		Value:    "smart",
		Evidence: "Sarah is very smart",
	}

	questions := engine.GenerateContactContextClarification("Sarah", fact)

	if len(questions) != 3 {
		t.Errorf("GenerateContactContextClarification() got %d questions, want 3", len(questions))
	}

	// Check questions are about contact context
	for _, q := range questions {
		if q.Type != "contact_context" {
			t.Errorf("Question type = %s, want contact_context", q.Type)
		}
	}

	// Verify questions are linked to the fact
	for _, q := range questions {
		if len(q.LinkedFacts) == 0 {
			t.Error("Questions should have linked facts")
		}
	}
}
