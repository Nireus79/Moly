package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/agents"
	"moly/database"
)

// EphemeralConversationManager manages the lifecycle of conversations
// Saves → Extracts → Deletes with 24h TTL
type EphemeralConversationManager struct {
	db       *database.Database
	analyzer *agents.ConversationAnalyzer
	updater  *ProfileUpdater
}

// ConversationMetadata tracks a conversation in the system
type ConversationMetadata struct {
	ID                string `json:"id"`
	UserID            string `json:"userId"`
	ConversationID    string `json:"conversationId"`
	StartedAt         int64  `json:"startedAt"`
	EndedAt           int64  `json:"endedAt"`
	MessageCount      int    `json:"messageCount"`
	CreatedAt         int64  `json:"createdAt"`
	ExpiresAt         int64  `json:"expiresAt"`
	ExtractionStatus  string `json:"extractionStatus"` // pending, processing, completed, failed
	ExtractionAttempt int    `json:"extractionAttempt"`
}

// ExtractionQueueItem represents a conversation pending extraction
type ExtractionQueueItem struct {
	ID                int     `json:"id"`
	ConversationID    string  `json:"conversationId"`
	UserID            string  `json:"userId"`
	ExtractionType    string  `json:"extractionType"`   // "full_extraction"
	Status            string  `json:"status"`           // pending, processing, completed, failed
	ExtractionResult  string  `json:"extractionResult"` // JSON
	Confidence        float64 `json:"confidence"`
	QueuedAt          int64   `json:"queuedAt"`
	AttemptedAt       int64   `json:"attemptedAt"`
	CompletedAt       int64   `json:"completedAt"`
	ErrorMessage      string  `json:"errorMessage"`
	ExtractionAttempt int     `json:"extractionAttempt"`
	MaxRetries        int     `json:"maxRetries"`
}

// NewEphemeralConversationManager creates a new manager
func NewEphemeralConversationManager(
	db *database.Database,
	analyzer *agents.ConversationAnalyzer,
	updater *ProfileUpdater,
) *EphemeralConversationManager {
	if db == nil || analyzer == nil || updater == nil {
		log.Fatal("[EphemeralManager] Dependencies cannot be nil")
	}

	return &EphemeralConversationManager{
		db:       db,
		analyzer: analyzer,
		updater:  updater,
	}
}

// SaveConversation stores a conversation for 24h and queues for extraction
func (ecm *EphemeralConversationManager) SaveConversation(
	userID string,
	conversationID string,
	messages []agents.Message,
	startedAt, endedAt int64,
) error {
	if userID == "" || conversationID == "" || len(messages) == 0 {
		return fmt.Errorf("invalid conversation: userID, conversationID, and messages required")
	}

	now := time.Now().Unix()
	expiresAt := now + (24 * 3600) // 24 hours

	// Convert messages to JSON
	messagesJSON, err := json.Marshal(messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	// Insert into conversation_ephemeral
	query := `
		INSERT INTO conversation_ephemeral
		(id, user_id, conversation_id, started_at, ended_at, messages,
		 extraction_status, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	conversationKey := fmt.Sprintf("conv_%s_%d", conversationID, now)
	_, err = ecm.db.Exec(query,
		conversationKey, userID, conversationID, startedAt, endedAt,
		string(messagesJSON),
		"pending",
		now, expiresAt)

	if err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
	}

	// Queue for extraction
	queueQuery := `
		INSERT INTO extraction_queue
		(conversation_id, user_id, extraction_type, status, queued_at, max_retries)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err = ecm.db.Exec(queueQuery,
		conversationID, userID, "full_extraction", "pending", now, 3)

	if err != nil {
		return fmt.Errorf("failed to queue extraction: %w", err)
	}

	log.Printf("[EphemeralManager] Saved conversation %s for user %s (%d messages, expires at %s)",
		conversationID, userID, len(messages), time.Unix(expiresAt, 0).Format(time.RFC3339))

	return nil
}

// ProcessQueue runs extraction on pending conversations (background job)
func (ecm *EphemeralConversationManager) ProcessQueue() (ProcessQueueStats, error) {
	stats := ProcessQueueStats{}

	// Get all pending items
	query := `
		SELECT id, conversation_id, user_id, extraction_type, status, extraction_attempt
		FROM extraction_queue
		WHERE status = 'pending' AND extraction_attempt < max_retries
		ORDER BY queued_at ASC
		LIMIT 10
	`

	rows, err := ecm.db.Query(query)
	if err != nil {
		return stats, fmt.Errorf("failed to fetch queue: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item ExtractionQueueItem
		err := rows.Scan(&item.ID, &item.ConversationID, &item.UserID,
			&item.ExtractionType, &item.Status, &item.ExtractionAttempt)
		if err != nil {
			log.Printf("[EphemeralManager] Error scanning queue item: %v", err)
			stats.Failed++
			continue
		}

		// Process this item
		success := ecm.processQueueItem(&item)
		if success {
			stats.Processed++
		} else {
			stats.Failed++
		}
	}

	log.Printf("[EphemeralManager] Queue processing complete: %d processed, %d failed",
		stats.Processed, stats.Failed)

	return stats, nil
}

// processQueueItem extracts and applies a single conversation
func (ecm *EphemeralConversationManager) processQueueItem(item *ExtractionQueueItem) bool {
	now := time.Now().Unix()

	// Update status to processing
	ecm.db.Exec(`UPDATE extraction_queue SET status = ?, attempted_at = ? WHERE id = ?`,
		"processing", now, item.ID)

	// Fetch conversation
	conv, err := ecm.getConversation(item.ConversationID)
	if err != nil {
		log.Printf("[EphemeralManager] Error fetching conversation: %v", err)
		ecm.markQueueItemFailed(item.ID, fmt.Sprintf("fetch error: %v", err))
		return false
	}

	// Parse messages
	var messages []agents.Message
	err = json.Unmarshal([]byte(conv.Messages), &messages)
	if err != nil {
		log.Printf("[EphemeralManager] Error parsing messages: %v", err)
		ecm.markQueueItemFailed(item.ID, fmt.Sprintf("parse error: %v", err))
		return false
	}

	// Run extraction
	result, err := ecm.analyzer.AnalyzeConversation(
		nil, // context.Background() would need to be passed
		item.UserID,
		item.ConversationID,
		messages)

	if err != nil {
		log.Printf("[EphemeralManager] Extraction failed: %v", err)
		ecm.markQueueItemFailed(item.ID, fmt.Sprintf("extraction error: %v", err))
		return false
	}

	// Apply to profile
	stats, err := ecm.updater.UpdateProfile(item.UserID, result)
	if err != nil {
		log.Printf("[EphemeralManager] Profile update failed: %v", err)
		ecm.markQueueItemFailed(item.ID, fmt.Sprintf("update error: %v", err))
		return false
	}

	// Serialize result
	resultJSON, err := result.ToJSON()
	if err != nil {
		log.Printf("[EphemeralManager] Failed to serialize result: %v", err)
		ecm.markQueueItemFailed(item.ID, fmt.Sprintf("serialization error: %v", err))
		return false
	}

	// Mark completed
	completedQuery := `
		UPDATE extraction_queue
		SET status = ?, extraction_result = ?, confidence = ?, completed_at = ?
		WHERE id = ?
	`

	_, err = ecm.db.Exec(completedQuery,
		"completed", resultJSON, result.ConfidenceScore, now, item.ID)

	if err != nil {
		log.Printf("[EphemeralManager] Failed to mark queue item completed: %v", err)
		return false
	}

	// Update conversation status
	ecm.db.Exec(`UPDATE conversation_ephemeral SET extraction_status = ? WHERE conversation_id = ?`,
		"completed", item.ConversationID)

	log.Printf("[EphemeralManager] Extracted conversation %s: %d AboutMe, %d patterns, %d contacts, %d goals",
		item.ConversationID,
		stats.AboutMeUpdated, stats.PatternsAdded, stats.ContactsAdded, stats.GoalsUpdated)

	return true
}

// getConversation retrieves a conversation from ephemeral storage
func (ecm *EphemeralConversationManager) getConversation(conversationID string) (*struct {
	Messages string
}, error) {
	query := `
		SELECT messages FROM conversation_ephemeral
		WHERE conversation_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`

	var messages string
	err := ecm.db.QueryRow(query, conversationID).Scan(&messages)
	if err != nil {
		return nil, err
	}

	return &struct {
		Messages string
	}{Messages: messages}, nil
}

// markQueueItemFailed marks a queue item as failed
func (ecm *EphemeralConversationManager) markQueueItemFailed(queueID int, errorMsg string) {
	now := time.Now().Unix()
	query := `
		UPDATE extraction_queue
		SET status = ?, error_message = ?, extraction_attempt = extraction_attempt + 1, completed_at = ?
		WHERE id = ?
	`

	ecm.db.Exec(query, "failed", errorMsg, now, queueID)
}

// CleanupExpired deletes conversations older than 24h (background job)
func (ecm *EphemeralConversationManager) CleanupExpired() (CleanupStats, error) {
	stats := CleanupStats{}

	now := time.Now().Unix()

	// Count before deletion
	countQuery := `
		SELECT COUNT(*) FROM conversation_ephemeral
		WHERE expires_at < ?
	`

	err := ecm.db.QueryRow(countQuery, now).Scan(&stats.DeletedCount)
	if err != nil {
		return stats, fmt.Errorf("failed to count expired: %w", err)
	}

	// Delete expired conversations
	deleteQuery := `
		DELETE FROM conversation_ephemeral
		WHERE expires_at < ?
	`

	result, err := ecm.db.Exec(deleteQuery, now)
	if err != nil {
		return stats, fmt.Errorf("failed to delete expired: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return stats, err
	}

	stats.DeletedCount = int(deleted)

	log.Printf("[EphemeralManager] Cleanup complete: deleted %d expired conversations", stats.DeletedCount)

	return stats, nil
}

// ProcessQueueStats tracks queue processing results
type ProcessQueueStats struct {
	Processed int
	Failed    int
	Total     int
}

// CleanupStats tracks cleanup results
type CleanupStats struct {
	DeletedCount int
	ErrorCount   int
}

// GetQueueStatus returns current queue status
func (ecm *EphemeralConversationManager) GetQueueStatus() (map[string]int, error) {
	status := make(map[string]int)

	query := `
		SELECT status, COUNT(*) as count
		FROM extraction_queue
		GROUP BY status
	`

	rows, err := ecm.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue status: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var statusStr string
		var count int
		err := rows.Scan(&statusStr, &count)
		if err != nil {
			log.Printf("[EphemeralManager] Error scanning status: %v", err)
			continue
		}
		status[statusStr] = count
	}

	return status, nil
}

// GetEphemeralConversationCount returns number of active ephemeral conversations
func (ecm *EphemeralConversationManager) GetEphemeralConversationCount() (int, error) {
	now := time.Now().Unix()
	query := `
		SELECT COUNT(*) FROM conversation_ephemeral
		WHERE expires_at > ?
	`

	var count int
	err := ecm.db.QueryRow(query, now).Scan(&count)
	return count, err
}

// RetryFailedExtractions retries failed extractions
func (ecm *EphemeralConversationManager) RetryFailedExtractions() (int, error) {
	query := `
		UPDATE extraction_queue
		SET status = 'pending', error_message = NULL
		WHERE status = 'failed' AND extraction_attempt < max_retries
	`

	result, err := ecm.db.Exec(query)
	if err != nil {
		return 0, err
	}

	rows, err := result.RowsAffected()
	return int(rows), err
}
