package agents

import (
	"log"
	"strings"
)

type IntentClassification string

const (
	ClassificationBenign   IntentClassification = "benign"
	ClassificationHarmful  IntentClassification = "harmful"
	ClassificationUnclear  IntentClassification = "unclear"
)

type DeterministicIntentDetector struct{}

func NewDeterministicIntentDetector() *DeterministicIntentDetector {
	return &DeterministicIntentDetector{}
}

// Classify three-state intent classification: benign, harmful, unclear
// Benign: clearly safe (greetings, learning questions)
// Harmful: only OBVIOUS direct harmful statements (kill, bomb, suicide)
// Unclear: everything else - ask clarifying question
//
// Does NOT try to catch every possible harmful phrasing.
// Relies on Socratic questioning to clarify ambiguous intent.
func (d *DeterministicIntentDetector) Classify(message string) IntentClassification {
	if message == "" {
		return ClassificationBenign
	}

	lower := strings.ToLower(message)

	// OBVIOUS HARMFUL: Direct statements only
	// Keep this list SMALL - only clear cases
	// Do NOT try to catch every variation (that's keyword matching)
	obviouslyHarmful := []string{
		"kill myself",
		"suicide",
		"kill",
		"bomb",
	}

	for _, phrase := range obviouslyHarmful {
		if strings.Contains(lower, phrase) {
			log.Printf("[DeterministicIntentDetector] HARMFUL: %q", message)
			return ClassificationHarmful
		}
	}

	// OBVIOUSLY BENIGN: Greetings, learning, common safe questions
	// Keep this list SMALL and SPECIFIC
	// Note: Includes both English and Greek variants for bot name
	obviouslyBenign := []string{
		"hello",
		"hi",
		"how are you",
		"what is",
		"tell me about",
		"explain",
		"teach",
		"learn",
		"moly",      // English variant
		"μώλυ",      // Greek: Μώλυ (lowercase)
	}

	for _, phrase := range obviouslyBenign {
		if strings.Contains(lower, phrase) {
			log.Printf("[DeterministicIntentDetector] BENIGN: %q", message)
			return ClassificationBenign
		}
	}

	// EVERYTHING ELSE: Ask for clarification (Socratic method)
	// Don't try to guess. Just ask "What do you mean?"
	// Harmful intents become clear through questioning.
	log.Printf("[DeterministicIntentDetector] UNCLEAR: %q (needs clarification)", message)
	return ClassificationUnclear
}

func (d *DeterministicIntentDetector) IsHarmful(classification IntentClassification) bool {
	return classification == ClassificationHarmful
}

func (d *DeterministicIntentDetector) IsUnclear(classification IntentClassification) bool {
	return classification == ClassificationUnclear
}
