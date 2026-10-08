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

// GetOrCreateSummary gets existing summary, or creates initial one if needed
// Returns the summary (existing or newly created)
func (m *ConversationSummaryManager) GetOrCreateSummary(
	ctx context.Context,
	userID string,
	conversationID string,
	messages []models.Message,
) (*models.ConversationSummary, error) {

	if userID == "" || conversationID == "" {
		return nil, fmt.Errorf("userID and conversationID are required")
	}

	// Try to get existing summary
	summary, err := m.repo.GetSummary(userID, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve summary: %w", err)
	}

	// If summary exists, return it
	if summary != nil {
		return summary, nil
	}

	// Create initial summary if none exists and we have messages
	if len(messages) > 0 {
		log.Printf("[ConversationSummaryManager] No existing summary, creating initial for conversation %s", conversationID)

		summary, err := m.summarizer.SummarizeConversation(ctx, conversationID, userID, messages, nil)
		if err != nil {
			log.Printf("[ConversationSummaryManager] Failed to generate initial summary: %v", err)
			return nil, fmt.Errorf("failed to generate summary: %w", err)
		}

		// Save to database
		if err := m.repo.CreateSummary(summary); err != nil {
			log.Printf("[ConversationSummaryManager] Failed to save summary: %v", err)
			return nil, fmt.Errorf("failed to save summary: %w", err)
		}

		return summary, nil
	}

	return nil, nil // No messages, no summary
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

// UpdateSummaryAfterMessageAdded increments counter and checks if update/creation needed
// Call this after each new message is added to conversation
// OPTIMIZED: Single GetSummary call, passed to UpdateSummaryIfNeeded to avoid double-read
func (m *ConversationSummaryManager) UpdateSummaryAfterMessageAdded(
	ctx context.Context,
	userID string,
	conversationID string,
	allMessages []models.Message,
) error {

	if userID == "" || conversationID == "" {
		return fmt.Errorf("userID and conversationID are required")
	}

	messageCount := len(allMessages)

	// Single GetSummary call - pass to update logic to avoid double-read
	summary, err := m.repo.GetSummary(userID, conversationID)
	if err != nil {
		return fmt.Errorf("failed to retrieve summary: %w", err)
	}

	// If no summary yet, check if we should create one now (early generation at 5 messages)
	if summary == nil {
		// Pass message count as threshold; ShouldUpdateSummary will check if >= 5
		if m.summarizer.ShouldUpdateSummary(nil, messageCount) {
			log.Printf("[ConversationSummaryManager] Creating early summary at %d messages", messageCount)
			updatedSummary, err := m.summarizer.SummarizeConversation(ctx, conversationID, userID, allMessages, nil)
			if err != nil {
				log.Printf("[ConversationSummaryManager] Failed to create early summary: %v", err)
				return nil // Fail gracefully, will retry later
			}
			if err := m.repo.CreateSummary(updatedSummary); err != nil {
				log.Printf("[ConversationSummaryManager] Failed to save early summary: %v", err)
				return nil // Fail gracefully, will retry later
			}
		}
		return nil
	}

	// Increment counter
	if err := m.repo.UpdateMessagesSinceUpdate(userID, conversationID, 1); err != nil {
		log.Printf("[ConversationSummaryManager] Failed to update counter: %v", err)
		return err
	}

	// Check if update is needed (pass existing summary to avoid second GetSummary call)
	// FIX 5: Early summary at message 5 instead of 10 (reduces re-extraction)
	_, err = m.UpdateSummaryIfNeededWithExisting(ctx, userID, conversationID, allMessages, 5, summary)
	return err
}

// UpdateSummaryAfterTopicBoundary updates summary when topic shifts
// Use this to capture topic transitions even before message threshold
func (m *ConversationSummaryManager) UpdateSummaryAfterTopicBoundary(
	ctx context.Context,
	userID string,
	conversationID string,
	allMessages []models.Message,
) (bool, error) {

	if userID == "" || conversationID == "" {
		return false, fmt.Errorf("userID and conversationID are required")
	}

	log.Printf("[ConversationSummaryManager] Topic boundary detected, updating summary")

	// Get current summary
	summary, err := m.repo.GetSummary(userID, conversationID)
	if err != nil {
		return false, err
	}

	// Generate updated summary regardless of threshold
	updatedSummary, err := m.summarizer.SummarizeConversation(ctx, conversationID, userID, allMessages, summary)
	if err != nil {
		return false, fmt.Errorf("failed to regenerate summary: %w", err)
	}

	// Save
	if summary != nil {
		if err := m.repo.UpdateSummary(updatedSummary); err != nil {
			return false, fmt.Errorf("failed to save summary: %w", err)
		}
	} else {
		if err := m.repo.CreateSummary(updatedSummary); err != nil {
			return false, fmt.Errorf("failed to save summary: %w", err)
		}
	}

	log.Printf("[ConversationSummaryManager] ✓ Summary updated after topic boundary")
	return true, nil
}

// UpdateSummaryAfterClarification adds confirmed choice to summary
// Call this after Layer 3 captures a clarification response
func (m *ConversationSummaryManager) UpdateSummaryAfterClarification(
	userID string,
	conversationID string,
	confirmedChoice string,
) error {

	if userID == "" || conversationID == "" || confirmedChoice == "" {
		return fmt.Errorf("userID, conversationID, and confirmedChoice are required")
	}

	log.Printf("[ConversationSummaryManager] Adding confirmed choice to summary: %s", confirmedChoice)

	// Get or create summary
	summary, err := m.repo.GetSummary(userID, conversationID)
	if err != nil {
		return fmt.Errorf("failed to retrieve summary: %w", err)
	}

	if summary == nil {
		// Create empty summary with confirmed choice
		summary = &models.ConversationSummary{
			UserID:              userID,
			ConversationID:      conversationID,
			Arc:                 "Conversation in progress",
			ConfirmedChoices:    []string{confirmedChoice},
			SummaryVersion:      1,
			Confidence:          0.5,
			MessagesSinceUpdate: 0,
		}
		return m.repo.CreateSummary(summary)
	}

	// Add to existing summary
	return m.repo.AddConfirmedChoice(userID, conversationID, confirmedChoice)
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
