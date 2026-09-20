package agents

import (
	"errors"
	"fmt"
	"log"
	"time"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// learningAgent - Builds user behavioral profile (user behavior only, NO contact surveillance)
type learningAgent struct {
	userID             string
	db                 *database.Database
	choiceRepo         *database.SuggestionChoiceRepository
	behaviorAnalyzer   *tools.BehaviorAnalyzer
	conflictRepo       *database.ContextConflictRepository
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
		userID:           userID,
		db:               db,
		choiceRepo:       database.NewSuggestionChoiceRepository(db),
		behaviorAnalyzer: tools.NewBehaviorAnalyzer(),
		conflictRepo:     database.NewContextConflictRepository(db),
	}, nil
}

// GetUserProfile - Retrieve user's behavioral profile
// Note: Behavioral profiles are deprecated in V2. Use Reflections instead.
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

	return profile, nil
}

// RecordInteraction - Record user's interaction for learning
func (la *learningAgent) RecordInteraction(data models.InteractionData) error {
	if data.UserID == "" {
		return errors.New("userID cannot be empty")
	}

	if la.db == nil {
		log.Printf("[LearningAgent] No database available, skipping interaction recording")
		return nil
	}

	log.Printf("[LearningAgent] Recording interaction for user %s: conv=%s messages=%d", data.UserID, data.ConversationID, data.SuggestionsGenerated)

	conn := la.db.GetConnection()
	if conn == nil {
		return errors.New("database connection unavailable")
	}

	// Save interaction to user_interactions table
	query := `
		INSERT INTO user_interactions (user_id, conversation_id, user_message, suggestions_generated, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	now := time.Now().Unix()
	_, err := conn.Exec(
		query,
		data.UserID,
		data.ConversationID,
		data.UserMessage,
		data.SuggestionsGenerated,
		now,
	)

	if err != nil {
		log.Printf("[LearningAgent] ERROR recording interaction: %v", err)
		return err
	}

	log.Printf("[LearningAgent] ✓ Interaction recorded")
	return nil
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

// BuildBehavioralProfile - Build comprehensive user profile using BehaviorAnalyzer
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

	if la.db == nil || la.conflictRepo == nil || la.behaviorAnalyzer == nil {
		log.Printf("[LearningAgent] Insufficient resources for analysis (DB=%v, conflict=%v, analyzer=%v)",
			la.db != nil, la.conflictRepo != nil, la.behaviorAnalyzer != nil)
		return profile, nil
	}

	log.Printf("[LearningAgent] Building behavioral profile for user %s", userID)

	// Load all resolved conflicts for this user
	resolvedConflicts, err := la.conflictRepo.GetResolved(userID)
	if err != nil {
		log.Printf("[LearningAgent] Warning: Could not load resolved conflicts: %v", err)
		return profile, nil
	}

	if len(resolvedConflicts) == 0 {
		log.Printf("[LearningAgent] No resolved conflicts to analyze yet")
		return profile, nil
	}

	log.Printf("[LearningAgent] Analyzing %d resolved conflicts", len(resolvedConflicts))

	// Convert conflicts to interface{} for analyzer
	var interactions []interface{}
	for _, conflict := range resolvedConflicts {
		interactions = append(interactions, map[string]interface{}{
			"id":                   conflict.ID,
			"conflict_type":        conflict.ConflictType,
			"saved_value":          conflict.SavedValue,
			"extracted_value":      conflict.ExtractedValue,
			"resolution":           conflict.Resolution,
			"timestamp":            conflict.ResolvedAt,
			"context":              la.extractContext(conflict),
			"communication_style":  la.extractCommunicationStyle(conflict),
		})
	}

	// Analyze choice patterns from conflicts
	choiceAnalysis := la.behaviorAnalyzer.AnalyzeChoicePatterns(interactions)
	if choiceAnalysis != nil {
		profile.CommunicationGoals = choiceAnalysis.CommunicationGoals
		profile.SuccessMetrics = choiceAnalysis.SuccessMetrics
		profile.Confidence = choiceAnalysis.Confidence
		log.Printf("[LearningAgent] Analyzed choice patterns: confidence=%.2f", profile.Confidence)
	}

	// Analyze interaction frequency
	frequencyStats := la.behaviorAnalyzer.AnalyzeInteractionFrequency(interactions)
	if frequencyStats != nil {
		profile.CommunicationProfile["frequency"] = frequencyStats
		log.Printf("[LearningAgent] Analyzed frequency: %v", frequencyStats["activity_level"])
	}

	// Analyze growth/evolution
	growthAnalysis := la.behaviorAnalyzer.AnalyzeGrowth(interactions)
	if growthAnalysis != nil {
		profile.GrowthTrajectory = growthAnalysis
		log.Printf("[LearningAgent] Analyzed growth: trend=%s", growthAnalysis["trend"])
	}

	// Build context-specific profiles
	contextProfiles := la.behaviorAnalyzer.BuildContextProfiles(interactions)
	if contextProfiles != nil {
		profile.CommunicationProfile["contexts"] = contextProfiles
		log.Printf("[LearningAgent] Built %d context profiles", len(contextProfiles))
	}

	// Infer overall communication style
	style := la.behaviorAnalyzer.AnalyzeCommunicationStyle(interactions, nil)
	if style != "" && style != "developing" {
		profile.CommunicationProfile["dominant_style"] = style
		log.Printf("[LearningAgent] Identified dominant style: %s", style)
	}

	return profile, nil
}

// Helper: Extract context from conflict
func (la *learningAgent) extractContext(conflict *database.ContextConflict) string {
	if conflict.ResolutionDetails != nil {
		if ctx, ok := conflict.ResolutionDetails["newContext"].(string); ok && ctx != "" {
			return ctx
		}
		if ctx, ok := conflict.ResolutionDetails["context"].(string); ok && ctx != "" {
			return ctx
		}
	}
	return "general"
}

// Helper: Extract communication style from conflict
func (la *learningAgent) extractCommunicationStyle(conflict *database.ContextConflict) string {
	if conflict.ConflictType == "aboutme_communication_style" {
		// For style conflicts, use the extracted (newer) value
		var styleValue string
		if conflict.Resolution == "use_extracted" {
			if sv, ok := conflict.ExtractedValue.(string); ok {
				styleValue = sv
			}
		} else if conflict.Resolution == "keep_saved" {
			if sv, ok := conflict.SavedValue.(string); ok {
				styleValue = sv
			}
		} else {
			// For merge, use extracted
			if sv, ok := conflict.ExtractedValue.(string); ok {
				styleValue = sv
			}
		}
		return styleValue
	}
	return ""
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

	// Analyze suggestion choice patterns
	conn := la.db.GetConnection()
	rows, err := conn.Query(`
		SELECT suggestion_id, suggested_text, user_modification, chosen_at
		FROM suggestion_choices
		WHERE user_id = ?
		ORDER BY chosen_at DESC
		LIMIT 100
	`, userID)
	if err == nil {
		defer rows.Close()

		var choicesList []map[string]interface{}
		for rows.Next() {
			var suggestionID, suggestedText, userModification string
			var chosenAt int64
			if err := rows.Scan(&suggestionID, &suggestedText, &userModification, &chosenAt); err == nil {
				choice := map[string]interface{}{
					"suggestionId":    suggestionID,
					"suggestedText":   suggestedText,
					"userModification": userModification,
					"chosenAt":        chosenAt,
				}
				choicesList = append(choicesList, choice)
			}
		}

		// Analyze patterns if we have choices
		if len(choicesList) > 0 {
			analyzer := &tools.BehaviorAnalyzer{}
			suggestionPatterns := analyzer.AnalyzeSuggestionChoicePatterns(choicesList)
			if rate, ok := suggestionPatterns["acceptance_rate"].(string); ok {
				// Parse percentage string to float
				rateFloat := 0.0
				fmt.Sscanf(rate, "%f%%", &rateFloat)
				patterns.SuggestionPickRate = rateFloat / 100.0
			}
			if confidence, ok := suggestionPatterns["confidence"].(float64); ok && confidence > 0.5 {
				patterns.ConfidenceLevel = "high"
			}
		}
	}

	return patterns, nil
}
