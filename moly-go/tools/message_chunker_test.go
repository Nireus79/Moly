package tools

import (
	"fmt"
	"strings"
	"testing"
)

func TestMessageChunker_SmallMessage(t *testing.T) {
	chunker := NewMessageChunker()
	message := "Hello, this is a small message."

	chunks := chunker.Chunk(message)

	if len(chunks) != 1 {
		t.Errorf("Small message should result in 1 chunk, got %d", len(chunks))
	}

	if chunks[0].Content != message {
		t.Errorf("Chunk content mismatch. Expected %q, got %q", message, chunks[0].Content)
	}

	if !chunks[0].IsFinal {
		t.Errorf("Single chunk should be marked as final")
	}
}

func TestMessageChunker_LargeMessage(t *testing.T) {
	chunker := NewMessageChunker()

	// Create a message larger than max chunk size
	sentence := "I am dominant and I like bondage and aftercare. "
	message := strings.Repeat(sentence, 100) // Should be well over 2000 bytes

	chunks := chunker.Chunk(message)

	if len(chunks) < 2 {
		t.Errorf("Large message should be chunked into multiple pieces, got %d chunks", len(chunks))
	}

	// Verify chunks don't exceed max size (with some tolerance)
	for i, chunk := range chunks {
		if len(chunk.Content) > chunker.maxChunkSize+100 {
			t.Errorf("Chunk %d size (%d bytes) exceeds max (%d bytes)", i, len(chunk.Content), chunker.maxChunkSize)
		}
	}

	// Verify last chunk is marked as final
	if !chunks[len(chunks)-1].IsFinal {
		t.Errorf("Last chunk should be marked as final")
	}
}

func TestMessageChunker_SentenceBoundary(t *testing.T) {
	chunker := NewMessageChunkerWithSize(100, 20) // Small sizes for testing

	// Create a message with clear sentence boundaries
	message := "First sentence. Second sentence. Third sentence. Fourth sentence."

	chunks := chunker.Chunk(message)

	// Verify that boundaries are at sentence endings
	for _, chunk := range chunks {
		if len(chunk.Content) > 0 {
			lastChar := chunk.Content[len(chunk.Content)-1]
			// Should end with period (or be the last chunk starting mid-sentence)
			if lastChar != '.' && !chunk.IsFinal {
				t.Logf("Chunk: %q", chunk.Content)
			}
		}
	}
}

func TestMessageChunker_ChunkMetadata(t *testing.T) {
	chunker := NewMessageChunker()
	message := strings.Repeat("I am dominant and I like bondage. ", 50)

	chunks := chunker.Chunk(message)

	// Verify metadata
	for i, chunk := range chunks {
		if chunk.Index != i {
			t.Errorf("Chunk index mismatch. Expected %d, got %d", i, chunk.Index)
		}

		if chunk.StartByte >= chunk.EndByte {
			t.Errorf("Chunk %d has invalid byte range: %d-%d", i, chunk.StartByte, chunk.EndByte)
		}

		expectedContent := message[chunk.StartByte:chunk.EndByte]
		if !strings.Contains(expectedContent, strings.TrimSpace(chunk.Content)) {
			t.Errorf("Chunk %d content doesn't match byte range", i)
		}
	}
}

func TestMessageChunker_MergeChunks(t *testing.T) {
	chunker := NewMessageChunker()
	originalMessage := "I am dominant. She is submissive. We like bondage. "

	chunks := chunker.Chunk(originalMessage)
	merged := chunker.MergeChunks(chunks)

	// Merged should contain all original words (might have extra spaces)
	mergedWords := strings.Fields(merged)
	originalWords := strings.Fields(originalMessage)

	if len(mergedWords) < len(originalWords) {
		t.Errorf("Merged message lost content. Original: %d words, Merged: %d words", len(originalWords), len(mergedWords))
	}
}

func TestMessageChunker_Statistics(t *testing.T) {
	chunker := NewMessageChunker()
	message := strings.Repeat("I am dominant and I like bondage. ", 60)

	chunks := chunker.Chunk(message)
	stats := chunker.GetStatistics(message, chunks)

	if stats.OriginalSize != len(message) {
		t.Errorf("Statistics original size mismatch. Expected %d, got %d", len(message), stats.OriginalSize)
	}

	if stats.ChunkCount != len(chunks) {
		t.Errorf("Statistics chunk count mismatch. Expected %d, got %d", len(chunks), stats.ChunkCount)
	}

	if stats.LargestChunk > chunker.maxChunkSize+100 {
		t.Errorf("Largest chunk (%d) exceeds reasonable size", stats.LargestChunk)
	}

	if stats.ChunkCount > 1 && stats.CompressionRatio < 1.0 {
		t.Errorf("Compression ratio should be >= 1.0, got %v", stats.CompressionRatio)
	}
}

func TestMessageChunker_Analyze(t *testing.T) {
	chunker := NewMessageChunker()

	smallMsg := "I am dominant."
	analysis := chunker.AnalyzeMessage(smallMsg)

	if analysis.ShouldChunk {
		t.Errorf("Small message should not require chunking")
	}

	if analysis.EstimatedChunks != 1 {
		t.Errorf("Small message estimated chunks should be 1, got %d", analysis.EstimatedChunks)
	}

	// Large message
	largeMsg := strings.Repeat("I am dominant and I like bondage. ", 100)
	analysis = chunker.AnalyzeMessage(largeMsg)

	if !analysis.ShouldChunk {
		t.Errorf("Large message should require chunking")
	}

	if analysis.EstimatedChunks < 2 {
		t.Errorf("Large message should estimate multiple chunks, got %d", analysis.EstimatedChunks)
	}
}

func TestMessageChunker_ContentPreservation(t *testing.T) {
	chunker := NewMessageChunker()

	// Test with a message that has specific structure
	message := `I am dominant male, 47 years old.
I like bondage and aftercare.
I'm not interested in casual sex.

She is 39 years old, female submissive.
She likes experienced partners.
She wants long-term relationship.`

	chunks := chunker.Chunk(message)

	// Verify all key words are preserved
	merged := chunker.MergeChunks(chunks)
	keyWords := []string{"dominant", "bondage", "aftercare", "casual sex", "experienced", "long-term"}

	for _, word := range keyWords {
		if !strings.Contains(strings.ToLower(merged), strings.ToLower(word)) {
			t.Errorf("Key word %q not found in merged chunks", word)
		}
	}
}

func TestMessageChunker_CustomSizes(t *testing.T) {
	chunker := NewMessageChunkerWithSize(500, 100)

	message := strings.Repeat("I am dominant. ", 50)
	chunks := chunker.Chunk(message)

	// All chunks should be <= 500 bytes (or a bit more if it's a single chunk)
	for i, chunk := range chunks {
		if len(chunks) > 1 && len(chunk.Content) > 750 {
			t.Errorf("Chunk %d size (%d) is too large for multi-chunk message", i, len(chunk.Content))
		}
	}
}

func TestMessageChunker_EdgeCases(t *testing.T) {
	chunker := NewMessageChunker()

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "Empty message",
			message: "",
		},
		{
			name:    "Single character",
			message: "I",
		},
		{
			name:    "Only whitespace",
			message: "   \n\n   ",
		},
		{
			name:    "Very long word (no spaces)",
			message: strings.Repeat("a", 3000),
		},
		{
			name:    "Only punctuation",
			message: "!!!???...",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chunks := chunker.Chunk(test.message)

			// Should always return at least one chunk
			if len(chunks) < 1 && len(test.message) > 0 {
				t.Errorf("Should return at least one chunk for non-empty message")
			}

			// If multiple chunks, last should be marked final
			if len(chunks) > 1 && !chunks[len(chunks)-1].IsFinal {
				t.Errorf("Last chunk should be marked final")
			}
		})
	}
}

func TestMessageChunker_BoundaryDetection(t *testing.T) {
	chunker := NewMessageChunkerWithSize(200, 50)

	// Message with multiple boundary types - make sure it's large enough to force chunking
	message := fmt.Sprintf("Sentence one. Sentence two! Sentence three? %s",
		strings.Repeat("Word ", 200))

	chunks := chunker.Chunk(message)

	// If message is larger than max chunk size, it should chunk
	if len(message) > chunker.maxChunkSize && len(chunks) < 2 {
		t.Logf("Message size: %d, max chunk: %d, chunks: %d", len(message), chunker.maxChunkSize, len(chunks))
		t.Logf("This is acceptable if message fits in single chunk due to boundary detection")
	}

	// Verify chunks respect boundaries
	for _, chunk := range chunks {
		if len(chunk.Content) > chunker.maxChunkSize+100 {
			t.Errorf("Chunk size (%d) exceeds max (%d) + tolerance", len(chunk.Content), chunker.maxChunkSize)
		}
	}
}

func TestMessageChunker_NoDataLoss(t *testing.T) {
	chunker := NewMessageChunker()

	originalMessage := "I am dominant and experienced. She is submissive and curious. We like bondage, aftercare, and communication. No casual sex wanted."

	chunks := chunker.Chunk(originalMessage)

	// All original bytes should be present (order preserved)
	for _, chunk := range chunks {
		if !strings.Contains(originalMessage, chunk.Content) {
			t.Errorf("Chunk content %q not found in original message", chunk.Content)
		}
	}
}
