package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/database"
)

// ProfileService provides read API for user profile data
type ProfileService struct {
	db *database.Database
}

// NewProfileService creates a new profile service
func NewProfileService(db *database.Database) *ProfileService {
	if db == nil {
		log.Fatal("[ProfileService] Database cannot be nil")
	}
	return &ProfileService{db: db}
}

// UserProfile contains complete profile snapshot
type UserProfile struct {
	UserID           string                    `json:"userId"`
	AboutMe          *AboutMeProfile           `json:"aboutMe"`
	Contacts         []ContactProfile          `json:"contacts"`
	Patterns         []PatternProfile          `json:"patterns"`
	Goals            []GoalProfile             `json:"goals"`
	Learnings        []LearningProfile         `json:"learnings"`
	Reflections      []ReflectionEntry         `json:"reflections"`
	LastUpdated      int64                     `json:"lastUpdated"`
	ConfidenceScores map[string]ConfidenceStats `json:"confidenceScores"`
}

// AboutMeProfile represents communication profile
type AboutMeProfile struct {
	ID                   int       `json:"id"`
	CommunicationStyle   string    `json:"communicationStyle"`
	TonePreference       string    `json:"tonePreference"`
	CoreValues           []string  `json:"coreValues"`
	Preferences          map[string]interface{} `json:"preferences"`
	Confidence           float64   `json:"confidence"`
	ExtractedFromCount   int       `json:"extractedFromCount"`
	LastUpdated          int64     `json:"lastUpdated"`
}

// ContactProfile represents a contact with patterns
type ContactProfile struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	RelationshipType  string    `json:"relationshipType"`
	Context           string    `json:"context"`
	FirstMentioned    int64     `json:"firstMentioned"`
	TimesMentioned    int       `json:"timesMentioned"`
	LastMentioned     int64     `json:"lastMentioned"`
	CommPatterns      []CommPattern `json:"communicationPatterns"`
}

// CommPattern represents communication pattern with a contact
type CommPattern struct {
	Frequency        string   `json:"frequency"`
	ToneObserved     string   `json:"toneObserved"`
	Patterns         []string `json:"patterns"`
	MainTopics       []string `json:"mainTopics"`
	RecentOutcome    string   `json:"recentOutcome"`
	UserNotes        string   `json:"userNotes"`
}

// PatternProfile represents observed communication pattern
type PatternProfile struct {
	ID            int       `json:"id"`
	Pattern       string    `json:"pattern"`
	Category      string    `json:"category"`
	ObservationCount int    `json:"observationCount"`
	IsActive      bool      `json:"isActive"`
	IsGrowthArea  bool      `json:"isGrowthArea"`
	Confidence    float64   `json:"confidence"`
	FirstObserved int64     `json:"firstObserved"`
	LastObserved  int64     `json:"lastObserved"`
}

// GoalProfile represents communication goal
type GoalProfile struct {
	ID           int       `json:"id"`
	Goal         string    `json:"goal"`
	Category     string    `json:"category"`
	Status       string    `json:"status"` // active, achieved, paused, abandoned
	StartedAt    int64     `json:"startedAt"`
	TargetDate   int64     `json:"targetDate"`
	ProgressNotes string   `json:"progressNotes"`
	Confidence   float64   `json:"confidence"`
	Created      int64     `json:"created"`
}

// LearningProfile represents learned insight
type LearningProfile struct {
	ID          int       `json:"id"`
	LearningType string   `json:"learningType"` // about_me, pattern, contact, goal
	LearningKey string    `json:"learningKey"`
	LearningValue string  `json:"learningValue"`
	Source       string    `json:"source"` // extraction, user_input
	Confidence   float64   `json:"confidence"`
	IsConfirmed  bool      `json:"isConfirmed"`
	IsRejected   bool      `json:"isRejected"`
	CreatedAt    int64     `json:"createdAt"`
}

// ReflectionEntry represents journal entry
type ReflectionEntry struct {
	ID            int       `json:"id"`
	Content       string    `json:"content"`
	Tags          []string  `json:"tags"`
	EntryType     string    `json:"entryType"` // reflection, learning, breakthrough, struggle
	AboutContactID *int     `json:"aboutContactId"`
	Created       int64     `json:"created"`
}

// ConfidenceStats tracks confidence distribution
type ConfidenceStats struct {
	Average   float64 `json:"average"`
	Count     int     `json:"count"`
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
}

// GetUserProfile returns complete profile snapshot
func (ps *ProfileService) GetUserProfile(userID string) (*UserProfile, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	profile := &UserProfile{
		UserID:           userID,
		LastUpdated:      time.Now().Unix(),
		ConfidenceScores: make(map[string]ConfidenceStats),
	}

	// Get all components in parallel
	aboutMe, err := ps.GetAboutMe(userID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[ProfileService] Error getting AboutMe: %v", err)
		return nil, err
	}
	profile.AboutMe = aboutMe

	contacts, err := ps.GetContacts(userID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[ProfileService] Error getting contacts: %v", err)
		return nil, err
	}
	profile.Contacts = contacts

	patterns, err := ps.GetPatterns(userID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[ProfileService] Error getting patterns: %v", err)
		return nil, err
	}
	profile.Patterns = patterns

	goals, err := ps.GetGoals(userID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[ProfileService] Error getting goals: %v", err)
		return nil, err
	}
	profile.Goals = goals

	learnings, err := ps.GetLearnings(userID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[ProfileService] Error getting learnings: %v", err)
		return nil, err
	}
	profile.Learnings = learnings

	reflections, err := ps.GetReflections(userID)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[ProfileService] Error getting reflections: %v", err)
		return nil, err
	}
	profile.Reflections = reflections

	// Calculate confidence stats
	ps.calculateConfidenceStats(profile)

	return profile, nil
}

// GetAboutMe returns communication profile
func (ps *ProfileService) GetAboutMe(userID string) (*AboutMeProfile, error) {
	query := `
		SELECT id, communication_style, tone_preference, core_values, preferences,
		       confidence, extracted_from_count, updated_at
		FROM about_me_profile
		WHERE user_id = ?
		LIMIT 1
	`

	var profile AboutMeProfile
	var coreValuesJSON, preferencesJSON sql.NullString

	err := ps.db.QueryRow(query, userID).Scan(
		&profile.ID,
		&profile.CommunicationStyle,
		&profile.TonePreference,
		&coreValuesJSON,
		&preferencesJSON,
		&profile.Confidence,
		&profile.ExtractedFromCount,
		&profile.LastUpdated,
	)

	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	if coreValuesJSON.Valid {
		json.Unmarshal([]byte(coreValuesJSON.String), &profile.CoreValues)
	}
	if preferencesJSON.Valid {
		json.Unmarshal([]byte(preferencesJSON.String), &profile.Preferences)
	}

	return &profile, nil
}

// GetContacts returns contacts with their communication patterns
func (ps *ProfileService) GetContacts(userID string) ([]ContactProfile, error) {
	query := `
		SELECT id, name, relationship_type, context, first_mentioned,
		       times_mentioned, last_mentioned
		FROM contacts
		WHERE user_id = ?
		ORDER BY times_mentioned DESC
	`

	rows, err := ps.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []ContactProfile
	for rows.Next() {
		var contact ContactProfile
		err := rows.Scan(
			&contact.ID,
			&contact.Name,
			&contact.RelationshipType,
			&contact.Context,
			&contact.FirstMentioned,
			&contact.TimesMentioned,
			&contact.LastMentioned,
		)
		if err != nil {
			log.Printf("[ProfileService] Error scanning contact: %v", err)
			continue
		}

		// Get communication patterns for this contact
		patterns, err := ps.getContactPatterns(contact.ID)
		if err == nil {
			contact.CommPatterns = patterns
		}

		contacts = append(contacts, contact)
	}

	return contacts, nil
}

// getContactPatterns retrieves communication patterns for a specific contact
func (ps *ProfileService) getContactPatterns(contactID int) ([]CommPattern, error) {
	query := `
		SELECT frequency, tone_observed, patterns, main_topics, recent_outcome, user_notes
		FROM contact_communication_patterns
		WHERE contact_id = ?
	`

	rows, err := ps.db.Query(query, contactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []CommPattern
	for rows.Next() {
		var pattern CommPattern
		var patternsJSON, topicsJSON sql.NullString

		err := rows.Scan(
			&pattern.Frequency,
			&pattern.ToneObserved,
			&patternsJSON,
			&topicsJSON,
			&pattern.RecentOutcome,
			&pattern.UserNotes,
		)
		if err != nil {
			continue
		}

		if patternsJSON.Valid {
			json.Unmarshal([]byte(patternsJSON.String), &pattern.Patterns)
		}
		if topicsJSON.Valid {
			json.Unmarshal([]byte(topicsJSON.String), &pattern.MainTopics)
		}

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// GetPatterns returns observed communication patterns
func (ps *ProfileService) GetPatterns(userID string) ([]PatternProfile, error) {
	query := `
		SELECT id, pattern, category, observation_count, is_active, is_growth_area,
		       confidence, first_observed, last_observed
		FROM communication_patterns
		WHERE user_id = ?
		ORDER BY observation_count DESC
	`

	rows, err := ps.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []PatternProfile
	for rows.Next() {
		var pattern PatternProfile
		err := rows.Scan(
			&pattern.ID,
			&pattern.Pattern,
			&pattern.Category,
			&pattern.ObservationCount,
			&pattern.IsActive,
			&pattern.IsGrowthArea,
			&pattern.Confidence,
			&pattern.FirstObserved,
			&pattern.LastObserved,
		)
		if err != nil {
			log.Printf("[ProfileService] Error scanning pattern: %v", err)
			continue
		}

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

// GetGoals returns communication goals
func (ps *ProfileService) GetGoals(userID string) ([]GoalProfile, error) {
	query := `
		SELECT id, goal, category, status, started_at, target_date, progress_notes,
		       confidence, created_at
		FROM communication_goals
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := ps.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var goals []GoalProfile
	for rows.Next() {
		var goal GoalProfile
		err := rows.Scan(
			&goal.ID,
			&goal.Goal,
			&goal.Category,
			&goal.Status,
			&goal.StartedAt,
			&goal.TargetDate,
			&goal.ProgressNotes,
			&goal.Confidence,
			&goal.Created,
		)
		if err != nil {
			log.Printf("[ProfileService] Error scanning goal: %v", err)
			continue
		}

		goals = append(goals, goal)
	}

	return goals, nil
}

// GetLearnings returns learned insights with confirmation status
func (ps *ProfileService) GetLearnings(userID string) ([]LearningProfile, error) {
	query := `
		SELECT id, learning_type, learning_key, learning_value, source, confidence,
		       is_confirmed, is_rejected, created_at
		FROM implicit_learning
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := ps.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var learnings []LearningProfile
	for rows.Next() {
		var learning LearningProfile
		err := rows.Scan(
			&learning.ID,
			&learning.LearningType,
			&learning.LearningKey,
			&learning.LearningValue,
			&learning.Source,
			&learning.Confidence,
			&learning.IsConfirmed,
			&learning.IsRejected,
			&learning.CreatedAt,
		)
		if err != nil {
			log.Printf("[ProfileService] Error scanning learning: %v", err)
			continue
		}

		learnings = append(learnings, learning)
	}

	return learnings, nil
}

// GetReflections returns user's journal entries
func (ps *ProfileService) GetReflections(userID string) ([]ReflectionEntry, error) {
	query := `
		SELECT id, content, tags, entry_type, about_contact_id, created_at
		FROM reflection_journal
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := ps.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reflections []ReflectionEntry
	for rows.Next() {
		var entry ReflectionEntry
		var tagsJSON sql.NullString

		err := rows.Scan(
			&entry.ID,
			&entry.Content,
			&tagsJSON,
			&entry.EntryType,
			&entry.AboutContactID,
			&entry.Created,
		)
		if err != nil {
			log.Printf("[ProfileService] Error scanning reflection: %v", err)
			continue
		}

		if tagsJSON.Valid {
			json.Unmarshal([]byte(tagsJSON.String), &entry.Tags)
		}

		reflections = append(reflections, entry)
	}

	return reflections, nil
}

// ConfirmLearning marks a learning as confirmed
func (ps *ProfileService) ConfirmLearning(userID string, learningID int) error {
	query := `
		UPDATE implicit_learning
		SET is_confirmed = 1, is_rejected = 0
		WHERE id = ? AND user_id = ?
	`

	result, err := ps.db.Exec(query, learningID, userID)
	if err != nil {
		return fmt.Errorf("failed to confirm learning: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("learning not found")
	}

	log.Printf("[ProfileService] Confirmed learning %d for user %s", learningID, userID)
	return nil
}

// RejectLearning marks a learning as rejected
func (ps *ProfileService) RejectLearning(userID string, learningID int) error {
	query := `
		UPDATE implicit_learning
		SET is_rejected = 1, is_confirmed = 0
		WHERE id = ? AND user_id = ?
	`

	result, err := ps.db.Exec(query, learningID, userID)
	if err != nil {
		return fmt.Errorf("failed to reject learning: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("learning not found")
	}

	log.Printf("[ProfileService] Rejected learning %d for user %s", learningID, userID)
	return nil
}

// AddReflection adds a new reflection journal entry
func (ps *ProfileService) AddReflection(userID, content, entryType string, tags []string, aboutContactID *int) (int, error) {
	if userID == "" || content == "" {
		return 0, fmt.Errorf("userID and content required")
	}

	tagsJSON, _ := json.Marshal(tags)

	query := `
		INSERT INTO reflection_journal
		(user_id, content, tags, entry_type, about_contact_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := ps.db.Exec(query, userID, content, string(tagsJSON), entryType, aboutContactID, time.Now().Unix())
	if err != nil {
		return 0, fmt.Errorf("failed to add reflection: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	log.Printf("[ProfileService] Added reflection %d for user %s", id, userID)
	return int(id), nil
}

// UpdateGoalProgress updates progress on a communication goal
func (ps *ProfileService) UpdateGoalProgress(userID string, goalID int, progressNotes string, status string) error {
	query := `
		UPDATE communication_goals
		SET progress_notes = ?, status = ?
		WHERE id = ? AND user_id = ?
	`

	result, err := ps.db.Exec(query, progressNotes, status, goalID, userID)
	if err != nil {
		return fmt.Errorf("failed to update goal: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("goal not found")
	}

	log.Printf("[ProfileService] Updated goal %d for user %s", goalID, userID)
	return nil
}

// calculateConfidenceStats computes confidence distribution across profile
func (ps *ProfileService) calculateConfidenceStats(profile *UserProfile) {
	scores := make(map[string][]float64)

	// Collect AboutMe confidence
	if profile.AboutMe != nil {
		scores["aboutMe"] = append(scores["aboutMe"], profile.AboutMe.Confidence)
	}

	// Collect Pattern confidences
	for _, p := range profile.Patterns {
		scores["patterns"] = append(scores["patterns"], p.Confidence)
	}

	// Collect Goal confidences
	for _, g := range profile.Goals {
		scores["goals"] = append(scores["goals"], g.Confidence)
	}

	// Collect Learning confidences
	for _, l := range profile.Learnings {
		scores["learnings"] = append(scores["learnings"], l.Confidence)
	}

	// Calculate stats for each category
	for category, values := range scores {
		if len(values) == 0 {
			continue
		}

		sum := 0.0
		min := 1.0
		max := 0.0

		for _, v := range values {
			sum += v
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}

		profile.ConfidenceScores[category] = ConfidenceStats{
			Average: sum / float64(len(values)),
			Count:   len(values),
			Min:     min,
			Max:     max,
		}
	}
}
