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
	Index      int       // 0-based chunk number
	Content    string    // The actual chunk content
	StartByte  int       // Starting byte offset in original message
	EndByte    int       // Ending byte offset in original message
	IsFinal    bool      // Whether this is the last chunk
	ContextKey string    // Identifier for grouping related chunks
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

// NewMessageChunkerWithSize creates a chunker with custom size limits
func NewMessageChunkerWithSize(maxSize, minSize int) *MessageChunker {
	if maxSize < 1000 {
		maxSize = 1000 // Enforce minimum practical size
	}
	if minSize < 100 {
		minSize = 100
	}
	if minSize > maxSize {
		minSize = maxSize / 2
	}

	return &MessageChunker{
		maxChunkSize: maxSize,
		minChunkSize: minSize,
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

// ChunkStatistics provides metrics about how a message was chunked
type ChunkStatistics struct {
	OriginalSize   int
	ChunkCount     int
	AvgChunkSize   float64
	LargestChunk   int
	SmallestChunk  int
	CompressionRatio float64 // How much overhead from chunking
}

// GetStatistics calculates statistics about the chunks
func (mc *MessageChunker) GetStatistics(message string, chunks []ChunkInfo) ChunkStatistics {
	stats := ChunkStatistics{
		OriginalSize: len(message),
		ChunkCount:   len(chunks),
	}

	if len(chunks) == 0 {
		return stats
	}

	var totalSize int
	stats.LargestChunk = 0
	stats.SmallestChunk = len(message) + 1

	for _, chunk := range chunks {
		chunkSize := len(chunk.Content)
		totalSize += chunkSize

		if chunkSize > stats.LargestChunk {
			stats.LargestChunk = chunkSize
		}
		if chunkSize < stats.SmallestChunk {
			stats.SmallestChunk = chunkSize
		}
	}

	stats.AvgChunkSize = float64(totalSize) / float64(len(chunks))

	// Compression ratio: if message was chunked efficiently, ratio should be close to 1.0
	// Higher means more overhead from chunking
	if len(chunks) > 1 {
		// Account for chunk metadata overhead (index, boundary info, etc)
		metadataPerChunk := 50 // Rough estimate in bytes
		totalOverhead := metadataPerChunk * len(chunks)
		stats.CompressionRatio = float64(len(message)+totalOverhead) / float64(len(message))
	} else {
		stats.CompressionRatio = 1.0
	}

	return stats
}

// AnalyzeMessageForChunking provides information about whether a message should be chunked
type ChunkAnalysis struct {
	ShouldChunk       bool
	MessageSize       int
	EstimatedChunks   int
	Reason            string
	Sentences         int
	Paragraphs        int
	AverageSentenceLen int
}

// AnalyzeMessage examines a message to determine optimal chunking strategy
func (mc *MessageChunker) AnalyzeMessage(message string) ChunkAnalysis {
	analysis := ChunkAnalysis{
		MessageSize: len(message),
		ShouldChunk: len(message) > mc.maxChunkSize,
	}

	if analysis.ShouldChunk {
		analysis.EstimatedChunks = (len(message) + mc.maxChunkSize - 1) / mc.maxChunkSize
		analysis.Reason = fmt.Sprintf("Message size (%d bytes) exceeds max chunk size (%d bytes)", len(message), mc.maxChunkSize)
	} else {
		analysis.EstimatedChunks = 1
		analysis.Reason = "Message is small enough to process as single chunk"
	}

	// Count sentences
	sentenceCount := strings.Count(message, ".") + strings.Count(message, "!") + strings.Count(message, "?")
	analysis.Sentences = sentenceCount

	// Count paragraphs
	paragraphCount := strings.Count(message, "\n\n") + 1
	analysis.Paragraphs = paragraphCount

	// Calculate average sentence length
	if sentenceCount > 0 {
		analysis.AverageSentenceLen = len(message) / sentenceCount
	}

	return analysis
}
