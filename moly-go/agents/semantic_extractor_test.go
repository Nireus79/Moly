package agents

import (
	"context"
	"encoding/json"
	"testing"

	"moly/tools"
)

// MockLLMForExtraction provides predictable responses for testing
type MockLLMForExtraction struct {
	response string
	err      error
}

func (m *MockLLMForExtraction) Call(ctx context.Context, req *tools.LLMRequest) (*tools.LLMResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &tools.LLMResponse{
		Content: m.response,
	}, nil
}

func TestSemanticExtractor_CasualAndFormal(t *testing.T) {
	// Test extraction of communication style
	mockResponse := `[
  {"type":"style", "value":"casual", "subject":"user", "quote":"I'm casual", "confidence":0.95},
  {"type":"trait", "value":"detail-oriented", "subject":"she", "quote":"boss is very detail-oriented", "confidence":0.9}
]`

	mockLLM := &MockLLMForExtraction{response: mockResponse}
	extractor := NewSemanticExtractor(mockLLM)

	facts, err := extractor.Extract("I'm casual but my boss is very detail-oriented.")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(facts) != 2 {
		t.Errorf("Expected 2 facts, got %d", len(facts))
	}

	if facts[0].Value != "casual" {
		t.Errorf("Expected value 'casual', got '%s'", facts[0].Value)
	}
	if facts[0].ProposedSubject != "user" {
		t.Errorf("Expected subject 'user', got '%s'", facts[0].ProposedSubject)
	}
	if facts[0].Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %.2f", facts[0].Confidence)
	}

	if facts[1].Value != "detail-oriented" {
		t.Errorf("Expected value 'detail-oriented', got '%s'", facts[1].Value)
	}
	if facts[1].ProposedSubject != "she" {
		t.Errorf("Expected subject 'she', got '%s'", facts[1].ProposedSubject)
	}
}

func TestSemanticExtractor_MultipleSubjects(t *testing.T) {
	// Test detection of multiple subjects in one message
	mockResponse := `[
  {"type":"style", "value":"casual", "subject":"user", "quote":"I'm casual", "confidence":0.9},
  {"type":"value", "value":"honesty", "subject":"user", "quote":"I value honesty", "confidence":0.9},
  {"type":"preference", "value":"deadline-focused", "subject":"boss", "quote":"boss cares about deadlines", "confidence":0.85}
]`

	mockLLM := &MockLLMForExtraction{response: mockResponse}
	extractor := NewSemanticExtractor(mockLLM)

	facts, err := extractor.Extract("I'm casual and I value honesty. My boss cares about deadlines.")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(facts) != 3 {
		t.Errorf("Expected 3 facts, got %d", len(facts))
	}

	// Count subjects
	subjects := make(map[string]int)
	for _, f := range facts {
		subjects[f.ProposedSubject]++
	}

	if subjects["user"] != 2 {
		t.Errorf("Expected 2 facts about user, got %d", subjects["user"])
	}
	if subjects["boss"] != 1 {
		t.Errorf("Expected 1 fact about boss, got %d", subjects["boss"])
	}
}

func TestSemanticExtractor_AmbiguousPronouns(t *testing.T) {
	// Test detection of ambiguous pronouns (she, he, they)
	mockResponse := `[
  {"type":"trait", "value":"organized", "subject":"she", "quote":"She is organized", "confidence":0.92},
  {"type":"style", "value":"formal", "subject":"he", "quote":"he communicates formally", "confidence":0.88},
  {"type":"value", "value":"teamwork", "subject":"they", "quote":"they value teamwork", "confidence":0.9}
]`

	mockLLM := &MockLLMForExtraction{response: mockResponse}
	extractor := NewSemanticExtractor(mockLLM)

	facts, err := extractor.Extract("She is organized. He communicates formally. They value teamwork.")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Verify pronouns are captured as subjects
	expectedPronouns := []string{"she", "he", "they"}
	for i, expected := range expectedPronouns {
		if i >= len(facts) {
			t.Errorf("Expected fact %d not found", i)
			break
		}
		if facts[i].ProposedSubject != expected {
			t.Errorf("Expected subject '%s', got '%s'", expected, facts[i].ProposedSubject)
		}
	}
}

func TestSemanticExtractor_ConfidenceScores(t *testing.T) {
	// Test that confidence scores are properly parsed
	mockResponse := `[
  {"type":"style", "value":"casual", "subject":"user", "quote":"I'm casual", "confidence":0.95},
  {"type":"trait", "value":"maybe organized", "subject":"colleague", "quote":"might be organized", "confidence":0.5}
]`

	mockLLM := &MockLLMForExtraction{response: mockResponse}
	extractor := NewSemanticExtractor(mockLLM)

	facts, err := extractor.Extract("I'm casual. My colleague might be organized.")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if facts[0].Confidence != 0.95 {
		t.Errorf("Expected high confidence 0.95, got %.2f", facts[0].Confidence)
	}
	if facts[1].Confidence != 0.5 {
		t.Errorf("Expected low confidence 0.5, got %.2f", facts[1].Confidence)
	}
}

func TestSemanticExtractor_EmptyMessage(t *testing.T) {
	// Test handling of empty message
	mockLLM := &MockLLMForExtraction{response: "[]"}
	extractor := NewSemanticExtractor(mockLLM)

	facts, err := extractor.Extract("")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(facts) != 0 {
		t.Errorf("Expected 0 facts from empty message, got %d", len(facts))
	}
}

func TestSemanticExtractor_NoFacts(t *testing.T) {
	// Test message with no extractable facts (greeting)
	mockResponse := `[]`

	mockLLM := &MockLLMForExtraction{response: mockResponse}
	extractor := NewSemanticExtractor(mockLLM)

	facts, err := extractor.Extract("Hello there!")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(facts) != 0 {
		t.Errorf("Expected 0 facts from greeting, got %d", len(facts))
	}
}

func TestSemanticExtractor_FactTypes(t *testing.T) {
	// Test all fact types are properly captured
	mockResponse := `[
  {"type":"trait", "value":"organized", "subject":"user", "quote":"I'm organized", "confidence":0.9},
  {"type":"style", "value":"formal", "subject":"user", "quote":"I communicate formally", "confidence":0.9},
  {"type":"value", "value":"honesty", "subject":"user", "quote":"I value honesty", "confidence":0.9},
  {"type":"preference", "value":"async-work", "subject":"user", "quote":"I prefer async work", "confidence":0.9},
  {"type":"goal", "value":"promotion", "subject":"user", "quote":"I want a promotion", "confidence":0.9}
]`

	mockLLM := &MockLLMForExtraction{response: mockResponse}
	extractor := NewSemanticExtractor(mockLLM)

	facts, err := extractor.Extract("I'm organized and communicate formally. I value honesty, prefer async work, and want a promotion.")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	expectedTypes := []string{"trait", "style", "value", "preference", "goal"}
	if len(facts) != len(expectedTypes) {
		t.Errorf("Expected %d facts, got %d", len(expectedTypes), len(facts))
	}

	for i, expected := range expectedTypes {
		if i < len(facts) && facts[i].FactType != expected {
			t.Errorf("Fact %d: expected type '%s', got '%s'", i, expected, facts[i].FactType)
		}
	}
}

func TestSemanticExtractor_Evidence(t *testing.T) {
	// Test that evidence (quotes) are properly captured
	mockResponse := `[
  {"type":"style", "value":"casual", "subject":"user", "quote":"I'm casual", "confidence":0.95}
]`

	mockLLM := &MockLLMForExtraction{response: mockResponse}
	extractor := NewSemanticExtractor(mockLLM)

	facts, err := extractor.Extract("I'm casual and like to keep things relaxed.")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if facts[0].Evidence != "I'm casual" {
		t.Errorf("Expected evidence 'I'm casual', got '%s'", facts[0].Evidence)
	}
}

func TestSemanticExtractor_InvalidJSON(t *testing.T) {
	// Test graceful handling of invalid JSON response
	mockLLM := &MockLLMForExtraction{response: "This is not JSON"}
	extractor := NewSemanticExtractor(mockLLM)

	facts, err := extractor.Extract("Some message")
	if err != nil {
		t.Fatalf("Extract should handle invalid JSON gracefully, but got error: %v", err)
	}

	if len(facts) != 0 {
		t.Errorf("Expected 0 facts on invalid JSON, got %d", len(facts))
	}
}

func TestSemanticExtractor_IDs(t *testing.T) {
	// Test that extracted facts have unique IDs
	mockResponse := `[
  {"type":"style", "value":"casual", "subject":"user", "quote":"I'm casual", "confidence":0.95},
  {"type":"style", "value":"formal", "subject":"user", "quote":"I'm formal", "confidence":0.95}
]`

	mockLLM := &MockLLMForExtraction{response: mockResponse}
	extractor := NewSemanticExtractor(mockLLM)

	facts, err := extractor.Extract("I'm casual and formal.")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if facts[0].ID == facts[1].ID {
		t.Errorf("Expected unique IDs, but both facts have ID: %s", facts[0].ID)
	}

	if facts[0].ID == "" || facts[1].ID == "" {
		t.Errorf("Expected non-empty IDs")
	}
}

func TestSemanticExtractor_Timestamps(t *testing.T) {
	// Test that extracted facts have timestamps
	mockResponse := `[
  {"type":"style", "value":"casual", "subject":"user", "quote":"I'm casual", "confidence":0.95}
]`

	mockLLM := &MockLLMForExtraction{response: mockResponse}
	extractor := NewSemanticExtractor(mockLLM)

	facts, err := extractor.Extract("I'm casual")

	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(facts) == 0 {
		t.Errorf("Expected at least one fact")
		return
	}

	if facts[0].CreatedAt == 0 {
		t.Errorf("Expected non-zero timestamp")
	}
}

func TestSemanticExtractor_ComplexMessage(t *testing.T) {
	// Test a complex real-world message
	mockResponse := `[
  {"type":"style", "value":"casual", "subject":"user", "quote":"I'm pretty casual", "confidence":0.92},
  {"type":"value", "value":"honesty", "subject":"user", "quote":"I really value honesty", "confidence":0.94},
  {"type":"trait", "value":"detail-oriented", "subject":"boss", "quote":"my boss is detail-oriented", "confidence":0.88},
  {"type":"preference", "value":"deadline-focused", "subject":"boss", "quote":"He's obsessed with deadlines", "confidence":0.86}
]`

	mockLLM := &MockLLMForExtraction{response: mockResponse}
	extractor := NewSemanticExtractor(mockLLM)

	message := "I'm pretty casual and I really value honesty. My boss is detail-oriented and He's obsessed with deadlines."
	facts, err := extractor.Extract(message)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(facts) != 4 {
		t.Errorf("Expected 4 facts, got %d", len(facts))
	}

	// Verify structure of each fact
	for _, fact := range facts {
		if fact.Value == "" {
			t.Error("Fact value should not be empty")
		}
		if fact.FactType == "" {
			t.Error("Fact type should not be empty")
		}
		if fact.ProposedSubject == "" {
			t.Error("Proposed subject should not be empty")
		}
		if fact.Evidence == "" {
			t.Error("Evidence should not be empty")
		}
		if fact.Confidence < 0 || fact.Confidence > 1 {
			t.Errorf("Confidence should be 0-1, got %.2f", fact.Confidence)
		}
		if fact.ID == "" {
			t.Error("ID should not be empty")
		}
		if fact.CreatedAt == 0 {
			t.Error("CreatedAt should not be zero")
		}
	}
}

func TestSemanticExtractor_JSONMarshaling(t *testing.T) {
	// Test that ExtractedFact can be marshaled to JSON
	mockResponse := `[
  {"type":"style", "value":"casual", "subject":"user", "quote":"I'm casual", "confidence":0.95}
]`

	mockLLM := &MockLLMForExtraction{response: mockResponse}
	extractor := NewSemanticExtractor(mockLLM)

	facts, err := extractor.Extract("I'm casual")
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Try to marshal to JSON
	jsonBytes, err := json.Marshal(facts[0])
	if err != nil {
		t.Errorf("Failed to marshal fact to JSON: %v", err)
	}

	// Verify we can unmarshal it back
	var fact ExtractedFact
	err = json.Unmarshal(jsonBytes, &fact)
	if err != nil {
		t.Errorf("Failed to unmarshal fact from JSON: %v", err)
	}

	if fact.Value != "casual" {
		t.Errorf("Value changed after marshal/unmarshal: %s", fact.Value)
	}
}
