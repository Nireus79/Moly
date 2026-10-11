package tools

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"moly/database"
	"moly/models"
)

// ConversationSummaryManager keeps the summary of each conversation up to date: after every turn it is rewritten by the
// summarizer and stored. One conversation is updated by one writer at a time.
type ConversationSummaryManager struct {
	repo       *database.ConversationSummaryRepository
	summarizer *ConversationSummarizer
	db         *sql.DB
	locks      sync.Map // conversation id -> *sync.Mutex
}

// NewConversationSummaryManager creates a new manager
func NewConversationSummaryManager(
	db *sql.DB,
	repo *database.ConversationSummaryRepository,
	summarizer *ConversationSummarizer,
) *ConversationSummaryManager {
	return &ConversationSummaryManager{repo: repo, summarizer: summarizer, db: db}
}

// UpdateWithTurn rewrites the summary with the newest turn (the user's message and Moly's reply). messageCount is how many
// messages the conversation holds once this turn is stored, so the next prompt can tell whether the summary is behind.
// If the rewrite fails, the previous summary stays.
func (m *ConversationSummaryManager) UpdateWithTurn(
	ctx context.Context,
	userID, conversationID string,
	messageCount int,
	anchors SummaryAnchors,
	userMessage, molyReply string,
) error {
	if userID == "" || conversationID == "" {
		return fmt.Errorf("userID and conversationID are required")
	}
	lock, _ := m.locks.LoadOrStore(conversationID, &sync.Mutex{})
	mu := lock.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	existing, err := m.repo.GetSummary(userID, conversationID)
	if err != nil {
		return fmt.Errorf("failed to read the summary: %w", err)
	}
	previous := ""
	if existing != nil {
		previous = existing.Arc
	}
	text, err := m.summarizer.SummarizeTurn(ctx, previous, anchors, userMessage, molyReply)
	if err != nil {
		return err
	}
	if existing == nil {
		now := time.Now().Unix()
		return m.repo.CreateSummary(&models.ConversationSummary{
			UserID: userID, ConversationID: conversationID, Arc: text, MessageCount: messageCount,
			SummaryVersion: 1, Confidence: 0.7, LastUpdated: now, CreatedAt: now, UpdatedAt: now,
		})
	}
	err = m.repo.UpdateSummaryWithRetry(userID, conversationID, func(s *models.ConversationSummary) {
		s.Arc = text
		s.MessageCount = messageCount
		s.MessagesSinceUpdate = 0
		s.LastUpdated = time.Now().Unix()
	})
	if err != nil {
		log.Printf("[ConversationSummaryManager] Failed to store the summary: %v", err)
	}
	return err
}
