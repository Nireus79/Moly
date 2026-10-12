// integration_multi_person_test.go
// Integration tests for multi-person message tracking (Week 1-4)
// Tests subject attribution, profile parsing, and clarification capture

package main

import (
	"testing"

	"moly/tools"
)

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
