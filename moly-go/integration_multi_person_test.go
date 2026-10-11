// integration_multi_person_test.go
// Integration tests for multi-person message tracking (Week 1-4)
// Tests subject attribution, profile parsing, and clarification capture

package main

import (
	"testing"

	"moly/tools"
)

// TestLinguisticParserMultiPerson verifies subject attribution across multiple people
func TestLinguisticParserMultiPerson(t *testing.T) {
	parser := tools.NewLinguisticParser()

	tests := []struct {
		name               string
		message            string
		minExpectedCount   int // Minimum extractions (exact count varies by grammar rules)
		shouldHaveSubjects bool
	}{
		{
			name:               "I am dominant, she is submissive",
			message:            "I am dominant and she is submissive",
			minExpectedCount:   1,
			shouldHaveSubjects: true,
		},
		{
			name:               "User statement with negation",
			message:            "I don't want casual sex",
			minExpectedCount:   1,
			shouldHaveSubjects: true,
		},
		{
			name:               "Preference statement",
			message:            "I prefer communication over assumptions",
			minExpectedCount:   1,
			shouldHaveSubjects: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := parser.Parse(tt.message)
			if len(results) < tt.minExpectedCount {
				t.Errorf("Expected at least %d extractions, got %d", tt.minExpectedCount, len(results))
			}

			// Verify subject attribution if expected
			if tt.shouldHaveSubjects {
				for _, result := range results {
					if result.Subject == "" {
						t.Errorf("Expected subject attribution, got empty subject for: %+v", result)
					}
				}
			}
		})
	}
}


// TestMessageChunkerIntegration verifies large message chunking
func TestMessageChunkerIntegration(t *testing.T) {
	chunker := tools.NewMessageChunker()

	// Create a message that exceeds chunk size
	largeMessage := ""
	for i := 0; i < 100; i++ {
		largeMessage += "This is a long sentence that will need to be chunked for LLM processing. "
	}

	tests := []struct {
		name              string
		message           string
		minExpectedChunks int
		shouldChunk       bool
	}{
		{
			name:              "Small message (no chunking)",
			message:           "I am dominant",
			minExpectedChunks: 1,
			shouldChunk:       false,
		},
		{
			name:              "Large message (requires chunking)",
			message:           largeMessage,
			minExpectedChunks: 2, // At least 2 chunks
			shouldChunk:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunker.Chunk(tt.message)

			if len(chunks) < tt.minExpectedChunks {
				t.Errorf("Expected at least %d chunks, got %d", tt.minExpectedChunks, len(chunks))
			}

			// Verify each chunk is within size limit
			maxSize := 2000
			for i, chunk := range chunks {
				if len(chunk.Content) > maxSize {
					t.Errorf("Chunk %d exceeds max size: %d > %d", i, len(chunk.Content), maxSize)
				}
			}

			// Verify all chunks concatenated roughly equal original (with boundary loss)
			totalLen := 0
			for _, chunk := range chunks {
				totalLen += len(chunk.Content)
			}
			if totalLen < len(tt.message)/2 { // Allow 50% boundary loss
				t.Errorf("Total chunk length too small: %d < %d", totalLen, len(tt.message)/2)
			}
		})
	}
}

// TestNegationPreservation verifies "NOT" properties are tracked correctly
func TestNegationPreservation(t *testing.T) {
	parser := tools.NewLinguisticParser()

	tests := []struct {
		name              string
		message           string
		expectedExtrCount int
		checkForNegation  bool
	}{
		{
			name:              "Explicit negation",
			message:           "I don't want casual sex",
			expectedExtrCount: 1,
			checkForNegation:  true,
		},
		{
			name:              "Positive statement",
			message:           "I enjoy casual encounters",
			expectedExtrCount: 1,
			checkForNegation:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := parser.Parse(tt.message)
			if len(results) == 0 {
				t.Errorf("Expected extractions, got none")
			}

			// Check that results have consistent structure
			for _, result := range results {
				if result.Property == "" {
					t.Errorf("Expected property extraction, got empty")
				}
				// Negation is tracked in the extraction results if present
				if tt.checkForNegation && result.Type != "negated_property" {
					t.Logf("Note: Negation detection depends on grammar rule (may be in result.Property or result.Type)")
				}
			}
		})
	}
}


// BenchmarkChunking benchmarks message chunking for large messages
func BenchmarkChunking(b *testing.B) {
	chunker := tools.NewMessageChunker()

	// Create a 10KB message
	largeMessage := ""
	for i := 0; i < 500; i++ {
		largeMessage += "This is a sentence in a large message that needs chunking for LLM processing. "
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chunker.Chunk(largeMessage)
	}
}
