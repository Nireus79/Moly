package agents

import (
	"errors"
	"log"
	"time"

	"moly/database"
	"moly/models"
)

// learningAgent - Builds user behavioral profile (user behavior only, NO contact surveillance)
type learningAgent struct {
	userID      string
	db          *database.Database
	choiceRepo  *database.SuggestionChoiceRepository
	patternRepo *database.BehaviorPatternRepository
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
func NewLearningAgentWithDB(userID string, db *database.Database) (models.LearningAgent, error) {
	log.Printf("[LearningAgent] Initializing for user %s (DB available: %v)", userID, db != nil)

	if userID == "" {
		log.Printf("[LearningAgent] ERROR: userID cannot be empty")
		return nil, errors.New("userID cannot be empty")
	}

	if db == nil {
		log.Printf("[LearningAgent] No database available, running in memory-only mode")
		return &learningAgent{userID: userID, db: nil}, nil
	}

	log.Printf("[LearningAgent] Initialized with database access")
	return &learningAgent{
		userID:      userID,
		db:          db,
		choiceRepo:  database.NewSuggestionChoiceRepository(db),
		patternRepo: database.NewBehaviorPatternRepository(db),
	}, nil
}

// GetUserProfile - Retrieve user's behavioral profile
func (la *learningAgent) GetUserProfile(userID string) (*models.UserBehavioralProfile, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	profile := &models.UserBehavioralProfile{
		UserID:               userID,
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

	if la.db == nil || la.patternRepo == nil {
		return profile, nil
	}

	retrievedProfile, err := la.patternRepo.Get(userID)
	if err == nil && retrievedProfile != nil {
		return retrievedProfile, nil
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

	return nil // Interactions are recorded via InteractionRepository in context_manager
}

// RecordSuggestionChoice - Record which suggestions user picked
func (la *learningAgent) RecordSuggestionChoice(data models.SuggestionChoiceData) error {
	log.Printf("[LearningAgent] Recording suggestion choice: user=%s conv=%s suggestion=%d modified=%v",
		data.UserID, data.ConversationID, data.SuggestionIndex, data.ModifiedText != "")

	if data.UserID == "" {
		log.Printf("[LearningAgent] ERROR: userID cannot be empty")
		return errors.New("userID cannot be empty")
	}

	if la.db == nil || la.choiceRepo == nil {
		log.Printf("[LearningAgent] No database available, skipping suggestion recording")
		return nil
	}

	suggestionID := string(rune(data.SuggestionIndex))
	err := la.choiceRepo.Record(data.UserID, suggestionID, "", data.ModifiedText)
	if err != nil {
		log.Printf("[LearningAgent] ERROR recording choice: %v", err)
	} else {
		log.Printf("[LearningAgent] Suggestion choice recorded successfully")
	}
	return err
}

// BuildBehavioralProfile - Build comprehensive user profile
func (la *learningAgent) BuildBehavioralProfile(userID string) (*models.UserBehavioralProfile, error) {
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	profile := &models.UserBehavioralProfile{
		UserID:               userID,
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

	if la.db == nil || la.patternRepo == nil {
		return profile, nil
	}

	// Load from pattern repository
	if retrieved, err := la.patternRepo.Get(userID); err == nil && retrieved != nil {
		return retrieved, nil
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

	if la.db == nil || la.patternRepo == nil {
		return patterns, nil
	}

	// Load pattern data from repository
	profile, err := la.patternRepo.Get(userID)
	if err == nil && profile != nil {
		patterns.ModificationRate = profile.Confidence
		patterns.ConfidenceLevel = "medium"
	}

	return patterns, nil
}
