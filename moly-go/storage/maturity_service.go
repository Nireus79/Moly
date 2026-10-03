package storage

import (
	"encoding/json"
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

	// Try to load existing maturity state
	var maturityJSON string
	err := conn.QueryRow(`
		SELECT maturity_state FROM conversation_maturity
		WHERE user_id = ? AND conversation_id = ?
	`, userID, conversationID).Scan(&maturityJSON)

	if err != nil {
		log.Printf("[MaturityService] Creating new maturity context for user=%s conv=%s", userID, conversationID)
		return models.NewConversationMaturity(userID, conversationID), nil
	}

	// Parse existing maturity state
	if maturityJSON == "" {
		return models.NewConversationMaturity(userID, conversationID), nil
	}

	cm := &models.ConversationMaturity{}
	if err := json.Unmarshal([]byte(maturityJSON), cm); err != nil {
		log.Printf("[MaturityService] Warning: Failed to parse maturity state: %v, creating fresh", err)
		return models.NewConversationMaturity(userID, conversationID), nil
	}

	log.Printf("[MaturityService] ✓ Loaded existing maturity context (phases=%d, overall=%.2f)", len(cm.Phases), cm.OverallScore)
	return cm, nil
}

// MarkAccomplishment marks an accomplishment as complete and updates maturity
func (ms *MaturityService) MarkAccomplishment(userID, conversationID, phaseName, accomplishmentName string) error {
	if userID == "" || conversationID == "" {
		return fmt.Errorf("userID and conversationID required")
	}

	// Load current maturity context
	cm, err := ms.LoadOrCreateMaturityContext(userID, conversationID)
	if err != nil {
		return fmt.Errorf("failed to load maturity context: %w", err)
	}

	// Mark the accomplishment
	if err := cm.MarkAccomplished(phaseName, accomplishmentName); err != nil {
		return fmt.Errorf("failed to mark accomplishment: %w", err)
	}

	// Recalculate overall maturity
	cm.OverallScore = cm.CalculateOverallMaturity()
	cm.CurrentPhase = cm.EstimateCurrentPhase()
	cm.LastUpdated = time.Now().Unix()

	log.Printf("[MaturityService] ✓ Marked accomplished: %s.%s (overall=%.2f, phase=%s)",
		phaseName, accomplishmentName, cm.OverallScore, cm.CurrentPhase)

	// Save to database
	if err := ms.SaveMaturityContext(userID, conversationID, cm); err != nil {
		log.Printf("[MaturityService] Warning: Failed to save maturity context: %v", err)
		// Don't fail - we still marked it in memory
	}

	return nil
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

	// Serialize maturity context
	maturityJSON, err := json.Marshal(cm)
	if err != nil {
		return fmt.Errorf("failed to marshal maturity context: %w", err)
	}

	// Insert or update
	_, err = conn.Exec(`
		INSERT INTO conversation_maturity (user_id, conversation_id, maturity_state, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id, conversation_id) DO UPDATE SET
			maturity_state = excluded.maturity_state,
			updated_at = excluded.updated_at
	`, userID, conversationID, string(maturityJSON), time.Now().Unix())

	if err != nil {
		return fmt.Errorf("failed to save maturity context: %w", err)
	}

	log.Printf("[MaturityService] ✓ Saved maturity context (overall=%.2f)", cm.OverallScore)
	return nil
}

// GetMaturityContext returns current maturity without modification
func (ms *MaturityService) GetMaturityContext(userID, conversationID string) (*models.ConversationMaturity, error) {
	return ms.LoadOrCreateMaturityContext(userID, conversationID)
}

// GetCurrentPhase returns the current phase
func (ms *MaturityService) GetCurrentPhase(userID, conversationID string) (string, error) {
	cm, err := ms.GetMaturityContext(userID, conversationID)
	if err != nil {
		return "", err
	}

	return cm.CurrentPhase, nil
}

// GetOverallMaturity returns the overall maturity score
func (ms *MaturityService) GetOverallMaturity(userID, conversationID string) (float64, error) {
	cm, err := ms.GetMaturityContext(userID, conversationID)
	if err != nil {
		return 0.0, err
	}

	return cm.OverallScore, nil
}

// CanAdvancePhase checks if user has completed all required accomplishments for a phase
func (ms *MaturityService) CanAdvancePhase(userID, conversationID, phaseName string) (bool, error) {
	cm, err := ms.GetMaturityContext(userID, conversationID)
	if err != nil {
		return false, err
	}

	phase, exists := cm.Phases[phaseName]
	if !exists {
		return false, fmt.Errorf("phase %s not found", phaseName)
	}

	return phase.CanAdvance(), nil
}

// GetPhaseMaturity returns maturity percentage for a specific phase
func (ms *MaturityService) GetPhaseMaturity(userID, conversationID, phaseName string) (float64, error) {
	cm, err := ms.GetMaturityContext(userID, conversationID)
	if err != nil {
		return 0.0, err
	}

	phase, exists := cm.Phases[phaseName]
	if !exists {
		return 0.0, fmt.Errorf("phase %s not found", phaseName)
	}

	return phase.CalculateMaturity(), nil
}

// LogMaturityState logs current maturity state for debugging
func (ms *MaturityService) LogMaturityState(userID, conversationID string) {
	cm, err := ms.GetMaturityContext(userID, conversationID)
	if err != nil {
		log.Printf("[MaturityService] Error getting maturity: %v", err)
		return
	}

	log.Printf("[MaturityService] === MATURITY STATE ===")
	log.Printf("[MaturityService] Overall: %.2f | Phase: %s | User: %s | Conv: %s",
		cm.OverallScore, cm.CurrentPhase, userID, conversationID)

	for phaseName, phase := range cm.Phases {
		if phase == nil {
			continue
		}
		completed := phase.GetCompletedCount()
		total := phase.GetTotalCount()
		maturity := phase.CalculateMaturity()
		log.Printf("[MaturityService]   %s: %d/%d (%.0f%%)", phaseName, completed, total, maturity*100)
	}

	log.Printf("[MaturityService] =======================")
}
