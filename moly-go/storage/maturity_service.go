package storage

import (
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/models"
)

// MaturityService manages accomplishment-based maturity lifecycle
// Loads, calculates, marks accomplishments, and saves to database
type MaturityService struct {
	db *database.Database
}

// NewMaturityService creates a new maturity service
func NewMaturityService(db *database.Database) *MaturityService {
	return &MaturityService{
		db: db,
	}
}

// LoadOrCreateMaturityContext loads or creates maturity context from database
func (ms *MaturityService) LoadOrCreateMaturityContext(userID, conversationID string) (*models.ConversationMaturity, error) {
	if ms.db == nil {
		log.Printf("[MaturityService] Warning: Database unavailable, creating new context")
		return models.NewConversationMaturity(userID, conversationID), nil
	}

	conn := ms.db.GetConnection()
	if conn == nil {
		log.Printf("[MaturityService] Warning: No database connection, creating new context")
		return models.NewConversationMaturity(userID, conversationID), nil
	}

	// Try to load existing maturity state (FIX #41: maturity_state → maturity column)
	var maturityScore float64
	var currentPhase string
	var accomplishmentsCompleted, accomplishmentsTotal int
	err := conn.QueryRow(`
		SELECT current_phase, maturity, accomplishments_completed, accomplishments_total FROM conversation_maturity
		WHERE user_id = ? AND conversation_id = ?
	`, userID, conversationID).Scan(&currentPhase, &maturityScore, &accomplishmentsCompleted, &accomplishmentsTotal)

	if err != nil {
		log.Printf("[MaturityService] Creating new maturity context for user=%s conv=%s", userID, conversationID)
		return models.NewConversationMaturity(userID, conversationID), nil
	}

	// Reconstruct maturity object from database columns (FIX #41: proper schema mapping)
	cm := models.NewConversationMaturity(userID, conversationID)
	cm.OverallScore = maturityScore

	// CurrentPhase is stored as string in database
	cm.CurrentPhase = currentPhase

	log.Printf("[MaturityService] ✓ Loaded existing maturity context (phase=%s, score=%.2f, accomplishments=%d/%d)",
		currentPhase, maturityScore, accomplishmentsCompleted, accomplishmentsTotal)
	return cm, nil
}


// SaveMaturityContext persists maturity context to database
func (ms *MaturityService) SaveMaturityContext(userID, conversationID string, cm *models.ConversationMaturity) error {
	if ms.db == nil || cm == nil {
		return fmt.Errorf("database or maturity context is nil")
	}

	conn := ms.db.GetConnection()
	if conn == nil {
		return fmt.Errorf("no database connection")
	}

	// FIX #41: Use correct schema columns - CurrentPhase is already string
	_, err := conn.Exec(`
		INSERT INTO conversation_maturity (user_id, conversation_id, current_phase, maturity, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(user_id, conversation_id) DO UPDATE SET
			current_phase = excluded.current_phase,
			maturity = excluded.maturity,
			updated_at = excluded.updated_at
	`, userID, conversationID, cm.CurrentPhase, cm.OverallScore, time.Now().Unix())

	if err != nil {
		return fmt.Errorf("failed to save maturity context: %w", err)
	}

	log.Printf("[MaturityService] ✓ Saved maturity context (overall=%.2f)", cm.OverallScore)
	return nil
}






