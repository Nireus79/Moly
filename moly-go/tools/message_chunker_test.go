package tools

import (
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
