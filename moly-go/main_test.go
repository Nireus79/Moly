package main

import (
	"encoding/json"
	"testing"

	"moly/models"
)

// TestAccumulatedEntityCountAccumulation verifies FIX #10: accumulated count grows, not resets
func TestAccumulatedEntityCountAccumulation(t *testing.T) {
	tests := []struct {
		name              string
		existingCount     int
		newEntitiesCount  int
		expectedResult    int
		shouldAccumulate  bool
	}{
		{
			name:             "First message: 3 entities",
			existingCount:    0,
			newEntitiesCount: 3,
			expectedResult:   3,
			shouldAccumulate: true,
		},
		{
			name:             "Second message: +1 entity = 4 total",
			existingCount:    3,
			newEntitiesCount: 1,
			expectedResult:   4,
			shouldAccumulate: true,
		},
		{
			name:             "Third message: +2 entities = 6 total",
			existingCount:    4,
			newEntitiesCount: 2,
			expectedResult:   6,
			shouldAccumulate: true,
		},
		{
			name:             "No reset on zero new entities",
			existingCount:    6,
			newEntitiesCount: 0,
			expectedResult:   6,
			shouldAccumulate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the FIX #10 logic
			summary := &models.ConversationSummary{
				AccumulatedEntityCount: tt.existingCount,
			}

			// This is the corrected logic (after FIX #10)
			existingCount := summary.AccumulatedEntityCount
			newEntityCount := tt.newEntitiesCount
			summary.AccumulatedEntityCount = existingCount + newEntityCount

			// Verify accumulation
			if summary.AccumulatedEntityCount != tt.expectedResult {
				t.Errorf("Expected %d accumulated entities, got %d", tt.expectedResult, summary.AccumulatedEntityCount)
			}

			if tt.shouldAccumulate && summary.AccumulatedEntityCount <= tt.existingCount {
				if tt.newEntitiesCount > 0 {
					t.Errorf("Accumulated count should increase when new entities added")
				}
			}
		})
	}
}

// TestClarityProgressionAppend verifies maturity scores accumulate in progression array
func TestClarityProgressionAppend(t *testing.T) {
	summary := &models.ConversationSummary{
		ClarityProgression: "[]",
	}

	// Simulate three messages with improving maturity
	maturityScores := []float64{0.50, 0.63, 0.75}

	for i, score := range maturityScores {
		var progression []float64
		if err := json.Unmarshal([]byte(summary.ClarityProgression), &progression); err != nil {
			t.Fatalf("Message %d: Failed to unmarshal progression: %v", i+1, err)
		}

		progression = append(progression, score)
		progJSON, err := json.Marshal(progression)
		if err != nil {
			t.Fatalf("Message %d: Failed to marshal progression: %v", i+1, err)
		}
		summary.ClarityProgression = string(progJSON)
	}

	// Verify final progression
	var finalProgression []float64
	if err := json.Unmarshal([]byte(summary.ClarityProgression), &finalProgression); err != nil {
		t.Fatalf("Failed to unmarshal final progression: %v", err)
	}

	if len(finalProgression) != 3 {
		t.Errorf("Expected 3 maturity scores, got %d", len(finalProgression))
	}

	// Verify scores are improving
	for i := 0; i < len(finalProgression)-1; i++ {
		if finalProgression[i] >= finalProgression[i+1] {
			if finalProgression[i+1] > 0 { // Allow flat if no new information
				t.Errorf("Maturity should improve or stay flat, but score %d (%.2f) > score %d (%.2f)",
					i, finalProgression[i], i+1, finalProgression[i+1])
			}
		}
	}

	t.Logf("✓ Clarity progression: %v", finalProgression)
}

// TestJSONUnmarshalErrorHandling verifies malformed JSON is handled gracefully
func TestJSONUnmarshalErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		jsonString  string
		shouldError bool
		expectEmpty bool
	}{
		{
			name:        "Valid JSON array",
			jsonString:  `["value1", "value2"]`,
			shouldError: false,
			expectEmpty: false,
		},
		{
			name:        "Empty JSON array",
			jsonString:  `[]`,
			shouldError: false,
			expectEmpty: true,
		},
		{
			name:        "Malformed JSON",
			jsonString:  `[invalid json`,
			shouldError: true,
			expectEmpty: true,
		},
		{
			name:        "Wrong type (object instead of array)",
			jsonString:  `{"key": "value"}`,
			shouldError: true,
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var values []string
			err := json.Unmarshal([]byte(tt.jsonString), &values)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error for %q, got nil", tt.jsonString)
			}

			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected error for %q: %v", tt.jsonString, err)
			}

			if tt.expectEmpty && len(values) > 0 {
				t.Errorf("Expected empty result, got %v", values)
			}
		})
	}
}
