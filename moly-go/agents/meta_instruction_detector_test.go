package agents

import (
	"context"
	"testing"
)

// MockLLMProvider for testing without actual LLM calls
type MockLLMProvider struct {
	shouldReturn bool
}

func (m *MockLLMProvider) Call(ctx context.Context, req *interface{}) (*interface{}, error) {
	return nil, nil
}

func TestDetectViaLinguisticParser_FocusDirective(t *testing.T) {
	detector := NewMetaInstructionDetector(nil)

	// Test: "Lace is my focus" - should be detected via Tier 1 (LinguisticParser)
	result := detector.Detect(context.Background(), "Lace is my focus")

	if result == nil {
		t.Errorf("Expected result for 'Lace is my focus', got nil")
	} else if result.Type != "focus" {
		t.Errorf("Expected type 'focus', got '%s'", result.Type)
	} else if result.Source != "linguistic_parser" {
		t.Errorf("Expected source 'linguistic_parser', got '%s'", result.Source)
	} else if result.TargetTopic != "lace" {
		t.Errorf("Expected target 'lace', got '%s'", result.TargetTopic)
	} else if result.Confidence < 0.90 {
		t.Errorf("Expected high confidence (>0.90), got %.2f", result.Confidence)
	}
}

func TestDetectViaLinguisticParser_Constraint(t *testing.T) {
	detector := NewMetaInstructionDetector(nil)

	// Test: "remember to be patient" - should be detected via Tier 1 (LinguisticParser)
	result := detector.Detect(context.Background(), "Remember to be patient with me")

	if result == nil {
		t.Errorf("Expected result for 'Remember to be patient', got nil")
	} else if result.Type != "constraint" {
		t.Errorf("Expected type 'constraint', got '%s'", result.Type)
	} else if result.Source != "linguistic_parser" {
		t.Errorf("Expected source 'linguistic_parser', got '%s'", result.Source)
	}
}

func TestDetectViaLinguisticParser_Priority(t *testing.T) {
	detector := NewMetaInstructionDetector(nil)

	// Test: "my priority is intimacy" - should be detected via Tier 1 (LinguisticParser)
	result := detector.Detect(context.Background(), "My priority is intimacy in our relationship")

	if result == nil {
		t.Errorf("Expected result for 'My priority is intimacy', got nil")
	} else if result.Type != "focus" {
		t.Errorf("Expected type 'focus' (mapped from priority), got '%s'", result.Type)
	} else if result.Source != "linguistic_parser" {
		t.Errorf("Expected source 'linguistic_parser', got '%s'", result.Source)
	}
}

func TestDetectViaKeywords_IdentitySelfReference(t *testing.T) {
	detector := NewMetaInstructionDetector(nil)

	// Test: "You are Moly" - should be detected via Tier 2 (Keywords)
	result := detector.Detect(context.Background(), "You are Moly")

	if result == nil {
		t.Errorf("Expected result for 'You are Moly', got nil")
	} else if result.Type != "identity" {
		t.Errorf("Expected type 'identity', got '%s'", result.Type)
	} else if result.Source != "keywords" {
		t.Errorf("Expected source 'keywords', got '%s'", result.Source)
	} else if result.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %.2f", result.Confidence)
	}
}

func TestDetectNegatedDirective(t *testing.T) {
	detector := NewMetaInstructionDetector(nil)

	// Test: "I don't focus on drama" - should parse correctly with IsNegated=true
	result := detector.Detect(context.Background(), "I don't focus on drama")

	if result == nil {
		t.Errorf("Expected result for negated directive, got nil")
	} else if result.Type != "focus" {
		t.Errorf("Expected type 'focus', got '%s'", result.Type)
	} else if !result.IsNegated {
		t.Errorf("Expected IsNegated=true, got %v", result.IsNegated)
	} else if result.Source != "linguistic_parser" {
		t.Errorf("Expected source 'linguistic_parser', got '%s'", result.Source)
	}
}

func TestDetectCompoundInstructions(t *testing.T) {
	detector := NewMetaInstructionDetector(nil)

	// Test: "Lace is my focus and remember to be patient"
	// Should extract the first meta-instruction (focus)
	result := detector.Detect(context.Background(), "Lace is my focus and remember to be patient")

	if result == nil {
		t.Errorf("Expected result for compound instruction, got nil")
	} else if result.Type != "focus" {
		t.Errorf("Expected first directive to be 'focus', got '%s'", result.Type)
	} else if result.Source != "linguistic_parser" {
		t.Errorf("Expected source 'linguistic_parser', got '%s'", result.Source)
	}
}

func TestDetectSubjectAttribution(t *testing.T) {
	detector := NewMetaInstructionDetector(nil)

	// Test: Subject should be extracted from parser
	result := detector.Detect(context.Background(), "Lace is my focus")

	if result == nil {
		t.Errorf("Expected result, got nil")
	} else if len(result.Subjects) == 0 {
		t.Errorf("Expected subjects to be populated, got empty slice")
	} else if result.Subjects[0] != "user" {
		t.Errorf("Expected subject 'user', got '%s'", result.Subjects[0])
	}
}

func TestBackwardCompatibility_MetaInstructionType(t *testing.T) {
	detector := NewMetaInstructionDetector(nil)

	// Test: MetaInstruction type should still have all original fields
	result := detector.Detect(context.Background(), "Lace is my focus")

	if result == nil {
		t.Errorf("Expected result, got nil")
		return
	}

	// Check all fields exist and are accessible
	_ = result.Type
	_ = result.Confidence
	_ = result.TargetTopic
	_ = result.RawInstruction
	_ = result.Reasoning

	// New fields
	_ = result.Source
	_ = result.IsNegated
	_ = result.Subjects
}

func TestPerformance_FastPath(t *testing.T) {
	detector := NewMetaInstructionDetector(nil)

	// Test: LinguisticParser detection should be very fast (<100ms)
	// We won't enforce timing in tests, but verify it completes quickly
	result := detector.Detect(context.Background(), "Communication is my focus")

	if result == nil {
		t.Errorf("Expected result for performance test, got nil")
	}
}

func TestNegationPreservation_MultiplePatterns(t *testing.T) {
	detector := NewMetaInstructionDetector(nil)

	// Test 1: "not interested in casual"
	result1 := detector.Detect(context.Background(), "I'm not interested in casual encounters")
	if result1 == nil {
		t.Errorf("Expected result for 'not interested in casual', got nil")
	} else if !result1.IsNegated {
		t.Errorf("Expected IsNegated=true for 'not interested', got %v", result1.IsNegated)
	}

	// Test 2: "focus on communication" (positive, no negation)
	result2 := detector.Detect(context.Background(), "Let's focus on communication")
	if result2 == nil {
		t.Errorf("Expected result for 'focus on communication', got nil")
	} else if result2.IsNegated {
		t.Errorf("Expected IsNegated=false for positive focus, got %v", result2.IsNegated)
	}
}

func TestTierPrecedence(t *testing.T) {
	detector := NewMetaInstructionDetector(nil)

	// Test: Tier 1 (LinguisticParser) should take precedence over Tier 2 (Keywords)
	// Even though the message contains "focus", the grammar pattern should match first
	result := detector.Detect(context.Background(), "Let's focus on communication")

	if result == nil {
		t.Errorf("Expected result, got nil")
	} else if result.Source != "linguistic_parser" {
		t.Errorf("Expected Tier 1 (linguistic_parser) to detect first, got source '%s'", result.Source)
	}
}
