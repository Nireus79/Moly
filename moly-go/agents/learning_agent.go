package agents

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"moly/models"
)

// learningAgent - Builds user behavioral profile (user behavior only, NO contact surveillance)
type learningAgent struct {
	userID string
	db     interface{} // Generic interface to avoid circular imports
}

// NewLearningAgent - Create new learning agent (no database)
func NewLearningAgent(userID string) (models.LearningAgent, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	return &learningAgent{
		userID: userID,
		db:     nil,
	}, nil
}

// NewLearningAgentWithDB - Create new learning agent with database access
func NewLearningAgentWithDB(userID string, db interface{}) (models.LearningAgent, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	return &learningAgent{
		userID: userID,
		db:     db,
	}, nil
}

// GetUserProfile - Retrieve user's behavioral profile
func (la *learningAgent) GetUserProfile(userID string) (*models.UserBehavioralProfile, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	profile := &models.UserBehavioralProfile{
		UserID:                userID,
		CommunicationProfile: make(map[string]interface{}),
		CommunicationGoals:   make(map[string]int),
		SuggestionChoices:    make(map[string]interface{}),
		SuccessMetrics:       make(map[string]interface{}),
		EmergingPersonality:  []string{},
		GrowthTrajectory:     make(map[string]interface{}),
		CreatedAt:            time.Now().Unix(),
		UpdatedAt:            time.Now().Unix(),
		Confidence:           0.5,
	}

	if la.db == nil {
		return profile, nil
	}

	db := la.db.(*sql.DB)
	var choices string
	err := db.QueryRow(`
		SELECT GROUP_CONCAT(context_metadata) FROM interactions
		WHERE topic = 'suggestion_choice' LIMIT 100
	`).Scan(&choices)

	if err == nil && choices != "" {
		json.Unmarshal([]byte(choices), &profile.SuggestionChoices)
		profile.Confidence = 0.7
	}

	return profile, nil
}

// RecordInteraction - Record user's interaction for learning
func (la *learningAgent) RecordInteraction(data models.InteractionData) error {
	if data.UserID == "" {
		return errors.New("userID cannot be empty")
	}

	if la.db == nil {
		return nil
	}

	db := la.db.(*sql.DB)
	_, err := db.Exec(`
		INSERT INTO interactions (conversation_id, topic, user_notes, ai_summary)
		VALUES (?, ?, ?, ?)
	`, data.ConversationID, "user_interaction", "", data.UserMessage)

	return err
}

// RecordSuggestionChoice - Record which suggestions user picked
func (la *learningAgent) RecordSuggestionChoice(data models.SuggestionChoiceData) error {
	if data.UserID == "" {
		return errors.New("userID cannot be empty")
	}

	if la.db == nil {
		return nil
	}

	db := la.db.(*sql.DB)
	choiceJSON, _ := json.Marshal(data)

	_, err := db.Exec(`
		INSERT INTO interactions (conversation_id, topic, ai_summary, context_metadata)
		VALUES (?, ?, ?, ?)
	`, data.ConversationID, "suggestion_choice", data.ModifiedText, string(choiceJSON))

	return err
}

// BuildBehavioralProfile - Build comprehensive user profile
func (la *learningAgent) BuildBehavioralProfile(userID string) (*models.UserBehavioralProfile, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	profile := &models.UserBehavioralProfile{
		UserID:                userID,
		CommunicationProfile: make(map[string]interface{}),
		CommunicationGoals:   make(map[string]int),
		SuggestionChoices:    make(map[string]interface{}),
		SuccessMetrics:       make(map[string]interface{}),
		EmergingPersonality:  []string{},
		GrowthTrajectory:     make(map[string]interface{}),
		CreatedAt:            time.Now().Unix(),
		UpdatedAt:            time.Now().Unix(),
		Confidence:           0.5,
	}

	if la.db == nil {
		return profile, nil
	}

	db := la.db.(*sql.DB)
	var count int
	db.QueryRow("SELECT COUNT(*) FROM interactions WHERE topic='suggestion_choice'").Scan(&count)

	if count > 0 {
		profile.Confidence = 0.7
		profile.CommunicationGoals["analyze"] = count
	}

	return profile, nil
}

// DetectPatterns - Detect patterns in user's communication
func (la *learningAgent) DetectPatterns(userID string) (*models.UserPatterns, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	patterns := &models.UserPatterns{
		UserID:              userID,
		CommunicationStyle:  "developing",
		PreferredTone:       make(map[string]float64),
		SuggestionPickRate:  0.0,
		ModificationRate:    0.0,
		CommunicationGoals:  make(map[string]int),
		EmergingPersonality: []string{},
		ConfidenceLevel:     "low",
	}

	if la.db == nil {
		return patterns, nil
	}

	db := la.db.(*sql.DB)
	var total, choices, mods int
	db.QueryRow("SELECT COUNT(*) FROM interactions WHERE topic IN ('suggestion_choice', 'user_interaction')").Scan(&total)
	db.QueryRow("SELECT COUNT(*) FROM interactions WHERE topic='suggestion_choice'").Scan(&choices)
	db.QueryRow("SELECT COUNT(*) FROM interactions WHERE ai_summary IS NOT NULL").Scan(&mods)

	if total > 0 {
		patterns.SuggestionPickRate = float64(choices) / float64(total)
		patterns.ModificationRate = float64(mods) / float64(total)
		patterns.CommunicationGoals["total_interactions"] = total
		patterns.ConfidenceLevel = "medium"
	}

	return patterns, nil
}
