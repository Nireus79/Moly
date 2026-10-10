package tools

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"moly/database"
	"moly/models"
)

// ConversationSummaryManager orchestrates summary creation and updates
// Coordinates between repository (persistence) and summarizer (LLM)
type ConversationSummaryManager struct {
	repo       *database.ConversationSummaryRepository
	summarizer *ConversationSummarizer
	db         *sql.DB
}

// NewConversationSummaryManager creates a new manager
func NewConversationSummaryManager(
	db *sql.DB,
	repo *database.ConversationSummaryRepository,
	summarizer *ConversationSummarizer,
) *ConversationSummaryManager {
	return &ConversationSummaryManager{
		repo:       repo,
		summarizer: summarizer,
		db:         db,
	}
}


// UpdateSummaryIfNeeded checks if update is needed and updates if so
// Returns true if summary was updated, false otherwise
// This is called after new messages are added to conversation
func (m *ConversationSummaryManager) UpdateSummaryIfNeeded(
	ctx context.Context,
	userID string,
	conversationID string,
	allMessages []models.Message,
	updateThreshold int,
) (bool, error) {
	// Default implementation: fetch summary first
	return m.UpdateSummaryIfNeededWithExisting(ctx, userID, conversationID, allMessages, updateThreshold, nil)
}

// UpdateSummaryIfNeededWithExisting is the optimized version that accepts existing summary
// to avoid redundant database calls. Use this when you already have the summary.
// Returns true if summary was updated, false otherwise
func (m *ConversationSummaryManager) UpdateSummaryIfNeededWithExisting(
	ctx context.Context,
	userID string,
	conversationID string,
	allMessages []models.Message,
	updateThreshold int,
	existingSummary *models.ConversationSummary,
) (bool, error) {

	if userID == "" || conversationID == "" {
		return false, fmt.Errorf("userID and conversationID are required")
	}

	// Use provided summary or fetch from database
	summary := existingSummary
	if summary == nil {
		var err error
		summary, err = m.repo.GetSummary(userID, conversationID)
		if err != nil {
			return false, fmt.Errorf("failed to retrieve summary: %w", err)
		}
	}

	// Check if update is needed
	if !m.summarizer.ShouldUpdateSummary(summary, updateThreshold) {
		return false, nil
	}

	log.Printf("[ConversationSummaryManager] Updating summary for conversation %s (trigger: threshold or stale)", conversationID)

	// Generate updated summary
	updatedSummary, err := m.summarizer.SummarizeConversation(ctx, conversationID, userID, allMessages, summary)
	if err != nil {
		log.Printf("[ConversationSummaryManager] Failed to regenerate summary: %v", err)
		// RECOVERY: Reset counter on LLM failure to retry sooner than 1 hour
		// This allows the next message to trigger retry instead of waiting for staleness timeout
		if resetErr := m.repo.ResetMessagesSinceUpdate(userID, conversationID); resetErr != nil {
			log.Printf("[ConversationSummaryManager] Warning: Failed to reset counter after LLM failure: %v", resetErr)
		}
		return false, fmt.Errorf("failed to regenerate summary: %w", err)
	}

	// Save updated summary (create if new, update if existing)
	if updatedSummary.ID == 0 {
		if err := m.repo.CreateSummary(updatedSummary); err != nil {
			log.Printf("[ConversationSummaryManager] Failed to save new summary: %v", err)
			return false, fmt.Errorf("failed to save summary: %w", err)
		}
	} else {
		if err := m.repo.UpdateSummary(updatedSummary); err != nil {
			log.Printf("[ConversationSummaryManager] Failed to save updated summary: %v", err)
			return false, fmt.Errorf("failed to save summary: %w", err)
		}
	}

	log.Printf("[ConversationSummaryManager] ✓ Summary updated (version %d)", updatedSummary.SummaryVersion)
	return true, nil
}




// GetSummaryNeedingUpdate retrieves summaries that should be updated
// Use this in a background task to batch-update stale summaries
func (m *ConversationSummaryManager) GetSummariesNeedingUpdate(
	userID string,
	threshold int,
) ([]*models.ConversationSummary, error) {

	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	if threshold <= 0 {
		threshold = 10
	}

	return m.repo.GetSummariesNeedingUpdate(userID, threshold)
}
