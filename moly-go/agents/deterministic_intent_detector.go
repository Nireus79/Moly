package agents

import (
	"log"
	"strings"

	"moly/models"
)

type DeterministicIntentDetector struct {
	constitution *models.Constitution
}

type IntentClassification string

const (
	ClassificationBenign   IntentClassification = "benign"
	ClassificationHarmful  IntentClassification = "harmful"
	ClassificationUnclear  IntentClassification = "unclear"
)

func NewDeterministicIntentDetector() *DeterministicIntentDetector {
	return &DeterministicIntentDetector{}
}

// SetConstitution injects the constitution for principle-based evaluation
func (d *DeterministicIntentDetector) SetConstitution(constitution *models.Constitution) {
	d.constitution = constitution
}

// Classify evaluates if an intent violates constitutional principles
// Returns: harmful, benign, or unclear (needs clarification)
func (d *DeterministicIntentDetector) Classify(message string) IntentClassification {
	if message == "" {
		return ClassificationBenign
	}

	lower := strings.ToLower(message)

	// CLEAR HARMFUL: Direct statements about harmful actions
	harmfulPatterns := []string{
		"kill myself", "suicide", "self-harm",
		"kill", "hurt", "harm", "attack", "abuse",
		"bomb", "explosive", "weapon",
		"fraud", "scam", "steal", "rob", "hack",
		"heroin", "cocaine", "meth", "fentanyl",
	}

	for _, pattern := range harmfulPatterns {
		if strings.Contains(lower, pattern) {
			log.Printf("[DeterministicIntentDetector] Classified HARMFUL: contains '%s'", pattern)
			return ClassificationHarmful
		}
	}

	// CLEAR BENIGN: Ordinary conversation (greetings, questions, learning)
	benignPatterns := []string{
		"hello", "hi", "how are", "good morning", "good afternoon", "good evening",
		"what is", "tell me about", "explain", "help me understand", "i want to learn",
		"how can i improve", "advice on communication",
	}

	for _, pattern := range benignPatterns {
		if strings.Contains(lower, pattern) {
			log.Printf("[DeterministicIntentDetector] Classified BENIGN: contains '%s'", pattern)
			return ClassificationBenign
		}
	}

	// UNCLEAR: Vague or ambiguous - needs clarification
	log.Printf("[DeterministicIntentDetector] Classified UNCLEAR: '%s' is vague", message)
	return ClassificationUnclear
}

// IsHarmful returns true if classification is harmful
func (d *DeterministicIntentDetector) IsHarmful(classification IntentClassification) bool {
	return classification == ClassificationHarmful
}

// IsUnclear returns true if classification is unclear
func (d *DeterministicIntentDetector) IsUnclear(classification IntentClassification) bool {
	return classification == ClassificationUnclear
}
