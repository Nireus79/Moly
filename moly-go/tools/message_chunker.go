package tools

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	// DefaultMaxChunkSize is the target maximum size for each chunk
	DefaultMaxChunkSize = 2000
	// MinChunkSize prevents chunks from being too small
	MinChunkSize = 500
)

// ChunkInfo represents a single message chunk with metadata
type ChunkInfo struct {
	Index      int    // 0-based chunk number
	Content    string // The actual chunk content
	StartByte  int    // Starting byte offset in original message
	EndByte    int    // Ending byte offset in original message
	IsFinal    bool   // Whether this is the last chunk
	ContextKey string // Identifier for grouping related chunks
}

// MessageChunker handles splitting long messages into manageable pieces
type MessageChunker struct {
	maxChunkSize int
	minChunkSize int
}

// NewMessageChunker creates a new message chunker with default settings
func NewMessageChunker() *MessageChunker {
	return &MessageChunker{
		maxChunkSize: DefaultMaxChunkSize,
		minChunkSize: MinChunkSize,
	}
}


// Chunk splits a message into chunks, trying to preserve sentence/thought boundaries
func (mc *MessageChunker) Chunk(message string) []ChunkInfo {
	// If message is small enough, return as single chunk
	if len(message) <= mc.maxChunkSize {
		return []ChunkInfo{
			{
				Index:      0,
				Content:    message,
				StartByte:  0,
				EndByte:    len(message),
				IsFinal:    true,
				ContextKey: "single",
			},
		}
	}

	chunks := mc.smartChunk(message)
	return chunks
}

// smartChunk uses intelligent boundary detection to split the message
func (mc *MessageChunker) smartChunk(message string) []ChunkInfo {
	var chunks []ChunkInfo
	remaining := message
	startByte := 0
	chunkIndex := 0

	for len(remaining) > 0 {
		// Determine if this chunk will be the last one
		willBeLast := len(remaining) <= mc.maxChunkSize

		// Find optimal chunk boundary
		chunkSize := mc.findOptimalBoundary(remaining, willBeLast)

		if chunkSize <= 0 {
			// Fallback: take entire remaining (shouldn't happen)
			chunkSize = len(remaining)
		}

		chunk := remaining[:chunkSize]
		endByte := startByte + chunkSize

		chunks = append(chunks, ChunkInfo{
			Index:      chunkIndex,
			Content:    strings.TrimSpace(chunk),
			StartByte:  startByte,
			EndByte:    endByte,
			IsFinal:    willBeLast,
			ContextKey: fmt.Sprintf("msg_%d", chunkIndex),
		})

		// Move to next chunk
		remaining = remaining[chunkSize:]
		startByte = endByte
		chunkIndex++
	}

	// Mark last chunk
	if len(chunks) > 0 {
		chunks[len(chunks)-1].IsFinal = true
	}

	return chunks
}

// findOptimalBoundary finds the best place to split a message
// Returns the byte position to split at
func (mc *MessageChunker) findOptimalBoundary(text string, isLastChunk bool) int {
	targetSize := mc.maxChunkSize

	// If this is the last chunk or text fits, return it all
	if isLastChunk || len(text) <= targetSize {
		return len(text)
	}

	// Try to split at sentence boundary (period, ?, !)
	sentenceBoundary := mc.findSentenceBoundary(text, targetSize)
	if sentenceBoundary > mc.minChunkSize {
		return sentenceBoundary
	}

	// Try to split at paragraph boundary (newline + newline)
	paragraphBoundary := mc.findParagraphBoundary(text, targetSize)
	if paragraphBoundary > mc.minChunkSize {
		return paragraphBoundary
	}

	// Try to split at line boundary (newline)
	lineBoundary := mc.findLineBoundary(text, targetSize)
	if lineBoundary > mc.minChunkSize {
		return lineBoundary
	}

	// Try to split at clause boundary (comma)
	clauseBoundary := mc.findClauseBoundary(text, targetSize)
	if clauseBoundary > mc.minChunkSize {
		return clauseBoundary
	}

	// Try to split at word boundary (space)
	wordBoundary := mc.findWordBoundary(text, targetSize)
	if wordBoundary > mc.minChunkSize {
		return wordBoundary
	}

	// Last resort: split at target size (might split a word)
	if targetSize < len(text) {
		return targetSize
	}

	return len(text)
}

// findSentenceBoundary looks for sentence-ending punctuation
// Looks backwards from targetSize to find the nearest sentence end
func (mc *MessageChunker) findSentenceBoundary(text string, targetSize int) int {
	// Start from target size and work backwards
	start := targetSize
	if start > len(text) {
		start = len(text)
	}

	for i := start; i >= mc.minChunkSize; i-- {
		if i < len(text) {
			ch := text[i]
			if ch == '.' || ch == '!' || ch == '?' {
				// Found sentence ending, include it and any following whitespace
				for j := i + 1; j < len(text) && unicode.IsSpace(rune(text[j])); j++ {
					i = j
				}
				return i + 1
			}
		}
	}

	return -1 // Not found
}

// findParagraphBoundary looks for double newlines
func (mc *MessageChunker) findParagraphBoundary(text string, targetSize int) int {
	start := targetSize
	if start > len(text) {
		start = len(text)
	}

	for i := start; i >= mc.minChunkSize; i-- {
		if i+1 < len(text) && text[i] == '\n' && text[i+1] == '\n' {
			// Found paragraph boundary
			for j := i + 2; j < len(text) && unicode.IsSpace(rune(text[j])); j++ {
				i = j
			}
			return i + 1
		}
	}

	return -1
}

// findLineBoundary looks for single newlines
func (mc *MessageChunker) findLineBoundary(text string, targetSize int) int {
	start := targetSize
	if start > len(text) {
		start = len(text)
	}

	for i := start; i >= mc.minChunkSize; i-- {
		if text[i] == '\n' {
			// Found line boundary
			for j := i + 1; j < len(text) && unicode.IsSpace(rune(text[j])); j++ {
				i = j
			}
			return i + 1
		}
	}

	return -1
}

// findClauseBoundary looks for commas (clause separators)
func (mc *MessageChunker) findClauseBoundary(text string, targetSize int) int {
	start := targetSize
	if start > len(text) {
		start = len(text)
	}

	for i := start; i >= mc.minChunkSize; i-- {
		if text[i] == ',' {
			// Found clause boundary
			for j := i + 1; j < len(text) && unicode.IsSpace(rune(text[j])); j++ {
				i = j
			}
			return i + 1
		}
	}

	return -1
}

// findWordBoundary looks for spaces (word separators)
func (mc *MessageChunker) findWordBoundary(text string, targetSize int) int {
	start := targetSize
	if start > len(text) {
		start = len(text)
	}

	for i := start; i >= mc.minChunkSize; i-- {
		if text[i] == ' ' {
			return i + 1
		}
	}

	return -1
}

// MergeChunks combines multiple chunks back into a single message
// Useful for reconstructing the original message from chunks
func (mc *MessageChunker) MergeChunks(chunks []ChunkInfo) string {
	if len(chunks) == 0 {
		return ""
	}

	var result strings.Builder
	for i, chunk := range chunks {
		result.WriteString(chunk.Content)
		// Add space between chunks if needed
		if i < len(chunks)-1 {
			lastChar := chunk.Content[len(chunk.Content)-1]
			if !unicode.IsSpace(rune(lastChar)) {
				result.WriteRune(' ')
			}
		}
	}

	return result.String()
}




