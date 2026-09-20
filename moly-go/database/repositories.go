package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"moly/models"
)

// AboutMeRepository - Manages AboutMe records
type AboutMeRepository struct {
	db *Database
}

// NewAboutMeRepository - Create new repository
func NewAboutMeRepository(db *Database) *AboutMeRepository {
	return &AboutMeRepository{db: db}
}

// Save - Save or update AboutMe
func (r *AboutMeRepository) Save(userID string, aboutMe *models.AboutMe) error {
	log.Printf("[Repository] Saving AboutMe for user %s (style=%s values=%d goals=%d)", userID, aboutMe.CommunicationStyle, len(aboutMe.Values), len(aboutMe.Goals))

	valuesJSON, _ := json.Marshal(aboutMe.Values)
	goalsJSON, _ := json.Marshal(aboutMe.Goals)
	now := time.Now().Unix()

	query := `
		INSERT INTO about_me (user_id, communication_style, core_values, tone_preference, goals, notes, created_at, updated_at, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(user_id) DO UPDATE SET
			communication_style = excluded.communication_style,
			core_values = excluded.core_values,
			tone_preference = excluded.tone_preference,
			goals = excluded.goals,
			notes = excluded.notes,
			updated_at = excluded.updated_at,
			version = version + 1
	`

	_, err := r.db.Exec(query, userID, aboutMe.CommunicationStyle, string(valuesJSON), aboutMe.PreferredTone, string(goalsJSON), aboutMe.Notes, now, now)
	if err != nil {
		log.Printf("[Repository] ERROR saving AboutMe: %v", err)
	} else {
		log.Printf("[Repository] AboutMe saved successfully")
	}
	return err
}

// Get - Get AboutMe for user
func (r *AboutMeRepository) Get(userID string) (*models.AboutMe, error) {
	query := `SELECT communication_style, core_values, tone_preference, goals, notes, created_at, updated_at, version FROM about_me WHERE user_id = ?`

	aboutMe := &models.AboutMe{UserID: userID}
	var valuesJSON sql.NullString
	var goalsJSON sql.NullString
	var notesSQL sql.NullString
	var version sql.NullInt64

	err := r.db.QueryRow(query, userID).Scan(&aboutMe.CommunicationStyle, &valuesJSON, &aboutMe.PreferredTone, &goalsJSON, &notesSQL, &aboutMe.CreatedAt, &aboutMe.UpdatedAt, &version)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is not an error
		}
		return nil, err
	}

	if valuesJSON.Valid {
		if err := json.Unmarshal([]byte(valuesJSON.String), &aboutMe.Values); err != nil {
			log.Printf("[AboutMeRepository] WARNING: Failed to unmarshal About Me values JSON for user %s: %v - values: %s", userID, err, valuesJSON.String)
		}
	}

	if goalsJSON.Valid {
		if err := json.Unmarshal([]byte(goalsJSON.String), &aboutMe.Goals); err != nil {
			log.Printf("[AboutMeRepository] WARNING: Failed to unmarshal About Me goals JSON for user %s: %v - goals: %s", userID, err, goalsJSON.String)
		}
	}

	if notesSQL.Valid {
		aboutMe.Notes = notesSQL.String
	}

	if version.Valid {
		aboutMe.Version = int(version.Int64)
	}

	return aboutMe, nil
}

// InteractionRepository - Manages Interaction records
type InteractionRepository struct {
	db *Database
}

// NewInteractionRepository - Create new repository
func NewInteractionRepository(db *Database) *InteractionRepository {
	return &InteractionRepository{db: db}
}

// Save - Save interaction
func (r *InteractionRepository) Save(userID string, conversationID string, content string, interactionType string, metadata map[string]interface{}) error {
	log.Printf("[Repository] Saving interaction for user %s (conv=%s type=%s content_len=%d)", userID, conversationID, interactionType, len(content))

	metadataJSON, _ := json.Marshal(metadata)
	now := time.Now().Unix()

	query := `
		INSERT INTO interactions (user_id, conversation_id, content, type, timestamp, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, userID, conversationID, content, interactionType, now, string(metadataJSON))
	if err != nil {
		log.Printf("[Repository] ERROR saving interaction: %v", err)
	} else {
		log.Printf("[Repository] Interaction saved successfully")
	}
	return err
}

// GetConversation - Get all interactions in a conversation
func (r *InteractionRepository) GetConversation(conversationID string, limit int) ([]models.Message, error) {
	if limit == 0 {
		limit = 50
	}

	query := `
		SELECT content, type, timestamp
		FROM interactions
		WHERE conversation_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		msg := models.Message{}
		var msgType sql.NullString

		if err := rows.Scan(&msg.Content, &msgType, &msg.Timestamp); err != nil {
			return nil, err
		}

		if msgType.Valid {
			msg.Type = msgType.String
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// BehaviorPatternRepository - Manages behavior pattern records
type BehaviorPatternRepository struct {
	db *Database
}

// NewBehaviorPatternRepository - Create new repository
func NewBehaviorPatternRepository(db *Database) *BehaviorPatternRepository {
	return &BehaviorPatternRepository{db: db}
}

// Save - Save behavior pattern
func (r *BehaviorPatternRepository) Save(userID string, profile *models.UserBehavioralProfile) error {
	tonesJSON, _ := json.Marshal(profile.CommunicationGoals)

	query := `
		INSERT INTO behavior_patterns (user_id, total_interactions, modification_rate, preferred_tones, communication_style, growth_trend, confidence, last_analyzed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			total_interactions = excluded.total_interactions,
			modification_rate = excluded.modification_rate,
			preferred_tones = excluded.preferred_tones,
			communication_style = excluded.communication_style,
			growth_trend = excluded.growth_trend,
			confidence = excluded.confidence,
			last_analyzed = excluded.last_analyzed
	`

	_, err := r.db.Exec(query, userID, 0, 0.0, string(tonesJSON), "", "stable", profile.Confidence, time.Now().Unix())
	return err
}

// Get - Get behavior pattern for user
func (r *BehaviorPatternRepository) Get(userID string) (*models.UserBehavioralProfile, error) {
	query := `SELECT total_interactions, modification_rate, preferred_tones, communication_style, growth_trend, confidence FROM behavior_patterns WHERE user_id = ?`

	profile := &models.UserBehavioralProfile{
		CommunicationProfile: make(map[string]interface{}),
		CommunicationGoals:   make(map[string]int),
		SuggestionChoices:    make(map[string]interface{}),
		SuccessMetrics:       make(map[string]interface{}),
		GrowthTrajectory:     make(map[string]interface{}),
	}

	var (
		totalInteractions sql.NullInt64
		modificationRate  sql.NullFloat64
		tonesJSON         sql.NullString
		commStyle         sql.NullString
		growthTrend       sql.NullString
	)

	err := r.db.QueryRow(query, userID).Scan(&totalInteractions, &modificationRate, &tonesJSON, &commStyle, &growthTrend, &profile.Confidence)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Parse JSON fields
	if tonesJSON.Valid {
		if err := json.Unmarshal([]byte(tonesJSON.String), &profile.CommunicationGoals); err != nil {
			log.Printf("[BehavioralProfileRepository] WARNING: Failed to unmarshal communication goals JSON for user %s: %v", userID, err)
		}
	}
	if commStyle.Valid {
		profile.CommunicationProfile["style"] = commStyle.String
	}
	if growthTrend.Valid {
		profile.GrowthTrajectory["trend"] = growthTrend.String
	}

	return profile, nil
}

// ReflectionRepository - Manages Reflection records
type ReflectionRepository struct {
	db *Database
}

// NewReflectionRepository - Create new repository
func NewReflectionRepository(db *Database) *ReflectionRepository {
	return &ReflectionRepository{db: db}
}

// Save - Save reflection
func (r *ReflectionRepository) Save(userID string, reflection *models.Reflection) error {
	charJSON, _ := json.Marshal(reflection.Characteristics)
	interestsJSON, _ := json.Marshal(reflection.Interests)
	intentionsJSON, _ := json.Marshal(reflection.Intentions)
	quotesJSON, _ := json.Marshal(reflection.UserQuotes)
	editsJSON, _ := json.Marshal(reflection.UserEdits)
	now := time.Now().Unix()

	query := `
		INSERT INTO reflections (user_id, conversation_id, contact_id, characteristics, interests, intentions, communication_preferences, user_quotes, user_edits, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, userID, reflection.ConversationID, reflection.ContactID, string(charJSON), string(interestsJSON), string(intentionsJSON), reflection.CommunicationPreferences, string(quotesJSON), string(editsJSON), reflection.Status, now)
	return err
}

// GetPendingApprovals - Get reflections awaiting approval (or any status if specified)
func (r *ReflectionRepository) GetPendingApprovals(userID string, statuses ...string) ([]models.Reflection, error) {
	// Default to pending_approval if no statuses provided
	if len(statuses) == 0 {
		statuses = []string{"pending_approval"}
	}

	// Build query with status filter
	statusPlaceholders := make([]string, len(statuses))
	queryArgs := []interface{}{userID}
	for i, status := range statuses {
		statusPlaceholders[i] = "?"
		queryArgs = append(queryArgs, status)
	}
	statusFilter := strings.Join(statusPlaceholders, ",")

	query := fmt.Sprintf(`SELECT id, conversation_id, contact_id, characteristics, interests, intentions, communication_preferences, user_quotes, user_edits, status, created_at FROM reflections WHERE user_id = ? AND status IN (%s) ORDER BY created_at DESC`, statusFilter)

	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reflections []models.Reflection
	for rows.Next() {
		refl := models.Reflection{}
		var id int64
		var charJSON, interestsJSON, interJSON, commPrefs, quotesJSON, editsJSON sql.NullString
		var conversationID, contactID sql.NullString

		if err := rows.Scan(&id, &conversationID, &contactID, &charJSON, &interestsJSON, &interJSON, &commPrefs, &quotesJSON, &editsJSON, &refl.Status, &refl.CreatedAt); err != nil {
			return nil, err
		}

		refl.ID = fmt.Sprintf("%d", id)
		if conversationID.Valid {
			refl.ConversationID = conversationID.String
		}
		if contactID.Valid {
			refl.ContactID = contactID.String
		}
		if commPrefs.Valid {
			refl.CommunicationPreferences = commPrefs.String
		}

		if charJSON.Valid {
			if err := json.Unmarshal([]byte(charJSON.String), &refl.Characteristics); err != nil {
				log.Printf("[ReflectionRepository] WARNING: Failed to unmarshal characteristics JSON: %v", err)
			}
		}
		if interestsJSON.Valid {
			if err := json.Unmarshal([]byte(interestsJSON.String), &refl.Interests); err != nil {
				log.Printf("[ReflectionRepository] WARNING: Failed to unmarshal interests JSON: %v", err)
			}
		}
		if interJSON.Valid {
			if err := json.Unmarshal([]byte(interJSON.String), &refl.Intentions); err != nil {
				log.Printf("[ReflectionRepository] WARNING: Failed to unmarshal intentions JSON: %v", err)
			}
		}
		if quotesJSON.Valid {
			if err := json.Unmarshal([]byte(quotesJSON.String), &refl.UserQuotes); err != nil {
				log.Printf("[ReflectionRepository] WARNING: Failed to unmarshal user quotes JSON: %v", err)
			}
		}
		if editsJSON.Valid {
			if err := json.Unmarshal([]byte(editsJSON.String), &refl.UserEdits); err != nil {
				log.Printf("[ReflectionRepository] WARNING: Failed to unmarshal user edits JSON: %v", err)
			}
		}

		reflections = append(reflections, refl)
	}

	return reflections, rows.Err()
}

// Approve - Approve a reflection
func (r *ReflectionRepository) Approve(reflectionID int) error {
	query := `UPDATE reflections SET status = 'approved', approved_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, time.Now().Unix(), reflectionID)
	return err
}

// Reject - Reject a reflection
func (r *ReflectionRepository) Reject(reflectionID int) error {
	query := `UPDATE reflections SET status = 'rejected' WHERE id = ?`
	_, err := r.db.Exec(query, reflectionID)
	return err
}

// SuggestionChoiceRepository - Manages suggestion choice tracking
type SuggestionChoiceRepository struct {
	db *Database
}

// NewSuggestionChoiceRepository - Create new repository
func NewSuggestionChoiceRepository(db *Database) *SuggestionChoiceRepository {
	return &SuggestionChoiceRepository{db: db}
}

// Record - Record a suggestion choice
func (r *SuggestionChoiceRepository) Record(userID string, suggestionID string, suggestedText string, userModification string) error {
	query := `
		INSERT INTO suggestion_choices (user_id, suggestion_id, suggested_text, user_modification, chosen_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, userID, suggestionID, suggestedText, userModification, time.Now().Unix())
	return err
}

// GetUserChoices - Get all choices for a user
func (r *SuggestionChoiceRepository) GetUserChoices(userID string, limit int) ([]map[string]interface{}, error) {
	if limit == 0 {
		limit = 100
	}

	query := `SELECT suggestion_id, suggested_text, user_modification, chosen_at FROM suggestion_choices WHERE user_id = ? ORDER BY chosen_at DESC LIMIT ?`

	rows, err := r.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var choices []map[string]interface{}
	for rows.Next() {
		var suggestionID, suggestedText, userMod sql.NullString
		var chosenAt int64

		if err := rows.Scan(&suggestionID, &suggestedText, &userMod, &chosenAt); err != nil {
			return nil, err
		}

		choice := map[string]interface{}{
			"suggestion_id":     suggestionID.String,
			"suggested_text":    suggestedText.String,
			"user_modification": userMod.String,
			"chosen_at":         chosenAt,
		}
		choices = append(choices, choice)
	}

	return choices, rows.Err()
}

// SafetyIncidentRepository - Manages safety incident tracking
type SafetyIncidentRepository struct {
	db *Database
}

// NewSafetyIncidentRepository - Create new repository
func NewSafetyIncidentRepository(db *Database) *SafetyIncidentRepository {
	return &SafetyIncidentRepository{db: db}
}

// Record - Record a safety incident
func (r *SafetyIncidentRepository) Record(userID string, severity string, content string, detectedBy string) error {
	query := `
		INSERT INTO safety_incidents (user_id, severity, detected_at, content, detected_by)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query, userID, severity, time.Now().Unix(), content, detectedBy)
	return err
}

// GetRecentIncidents - Get recent incidents for a user
func (r *SafetyIncidentRepository) GetRecentIncidents(userID string, limit int) ([]map[string]interface{}, error) {
	if limit == 0 {
		limit = 20
	}

	query := `SELECT severity, content, detected_at, detected_by FROM safety_incidents WHERE user_id = ? ORDER BY detected_at DESC LIMIT ?`

	rows, err := r.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []map[string]interface{}
	for rows.Next() {
		var severity, content, detectedBy sql.NullString
		var detectedAt int64

		if err := rows.Scan(&severity, &content, &detectedAt, &detectedBy); err != nil {
			return nil, err
		}

		incident := map[string]interface{}{
			"severity":    severity.String,
			"content":     content.String,
			"detected_at": detectedAt,
			"detected_by": detectedBy.String,
		}
		incidents = append(incidents, incident)
	}

	return incidents, rows.Err()
}

// QuestionEffectivenessRepository - Tracks Socratic question effectiveness
type QuestionEffectivenessRepository struct {
	db *Database
}

// NewQuestionEffectivenessRepository - Create new repository
func NewQuestionEffectivenessRepository(db *Database) *QuestionEffectivenessRepository {
	return &QuestionEffectivenessRepository{db: db}
}

// Save - Record question effectiveness after user answers
func (r *QuestionEffectivenessRepository) Save(
	userID string,
	questionID string,
	socraticApproach string,
	questionText string,
	userResponse string,
	reducedAmbiguity bool,
	insightGained string,
	depthLevelAdvanced bool,
	principleClarified string,
) error {
	log.Printf("[Repository] Saving question effectiveness: user=%s question=%s approach=%s", userID, questionID, socraticApproach)

	id := fmt.Sprintf("qe_%d", time.Now().UnixNano())
	now := time.Now().Unix()

	query := `
		INSERT INTO question_effectiveness (
			id, user_id, question_id, socratic_approach, question_text, user_response,
			reduced_ambiguity, insight_gained, depth_level_advanced, principle_clarified, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(
		query,
		id, userID, questionID, socraticApproach, questionText, userResponse,
		reducedAmbiguity, insightGained, depthLevelAdvanced, principleClarified, now,
	)

	if err != nil {
		log.Printf("[Repository] ERROR saving question effectiveness: %v", err)
	} else {
		log.Printf("[Repository] Question effectiveness saved: id=%s", id)
	}

	return err
}

// GetEffectiveQuestions - Get most effective questions for a user (for learning)
func (r *QuestionEffectivenessRepository) GetEffectiveQuestions(userID string, limit int) ([]map[string]interface{}, error) {
	if limit == 0 {
		limit = 20
	}

	// Questions that reduced ambiguity are more effective
	query := `
		SELECT question_id, socratic_approach, reduced_ambiguity, insight_gained, COUNT(*) as usage_count
		FROM question_effectiveness
		WHERE user_id = ? AND reduced_ambiguity = true
		GROUP BY question_id, socratic_approach
		ORDER BY usage_count DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var questions []map[string]interface{}
	for rows.Next() {
		var questionID, approach string
		var reduced bool
		var insight sql.NullString
		var usageCount int

		if err := rows.Scan(&questionID, &approach, &reduced, &insight, &usageCount); err != nil {
			return nil, err
		}

		q := map[string]interface{}{
			"question_id":       questionID,
			"approach":          approach,
			"reduced_ambiguity": reduced,
			"usage_count":       usageCount,
		}

		if insight.Valid {
			q["insight_gained"] = insight.String
		}

		questions = append(questions, q)
	}

	return questions, rows.Err()
}

// GetApproachEffectiveness - Get effectiveness statistics per approach
func (r *QuestionEffectivenessRepository) GetApproachEffectiveness(userID string) (map[string]map[string]interface{}, error) {
	query := `
		SELECT socratic_approach, COUNT(*) as total_asked, SUM(CASE WHEN reduced_ambiguity THEN 1 ELSE 0 END) as successful
		FROM question_effectiveness
		WHERE user_id = ?
		GROUP BY socratic_approach
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make(map[string]map[string]interface{})
	for rows.Next() {
		var approach string
		var totalAsked, successful sql.NullInt64

		if err := rows.Scan(&approach, &totalAsked, &successful); err != nil {
			return nil, err
		}

		total := int(totalAsked.Int64)
		succ := int(successful.Int64)
		successRate := 0.0
		if total > 0 {
			successRate = float64(succ) / float64(total)
		}

		results[approach] = map[string]interface{}{
			"total_asked":  total,
			"successful":   succ,
			"success_rate": successRate,
		}
	}

	return results, rows.Err()
}

// MetricsRepository provides query methods for learning analytics
type MetricsRepository struct {
	db *Database
}

// NewMetricsRepository creates a new metrics repository
func NewMetricsRepository(db *Database) *MetricsRepository {
	return &MetricsRepository{db: db}
}

// GetQuestionEffectivenessStats returns overall statistics on question effectiveness
func (m *MetricsRepository) GetQuestionEffectivenessStats(userID string) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_asked,
			SUM(CASE WHEN reduced_ambiguity THEN 1 ELSE 0 END) as total_reduced,
			SUM(CASE WHEN insight_gained THEN 1 ELSE 0 END) as total_insights,
			SUM(CASE WHEN depth_level_advanced THEN 1 ELSE 0 END) as total_depth_advanced,
			SUM(CASE WHEN principle_clarified THEN 1 ELSE 0 END) as total_principles_clarified
		FROM question_effectiveness
		WHERE user_id = ?
	`

	var totalAsked, totalReduced, totalInsights, totalDepthAdvanced, totalPrinciplesClarified sql.NullInt64
	err := m.db.GetConnection().QueryRow(query, userID).Scan(
		&totalAsked, &totalReduced, &totalInsights, &totalDepthAdvanced, &totalPrinciplesClarified,
	)

	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_asked":                   toInt(totalAsked),
		"total_reduced_ambiguity":       toInt(totalReduced),
		"total_insights_gained":         toInt(totalInsights),
		"total_depth_advanced":          toInt(totalDepthAdvanced),
		"total_principles_clarified":    toInt(totalPrinciplesClarified),
		"ambiguity_reduction_rate":      calculateRate(toInt(totalReduced), toInt(totalAsked)),
		"insight_generation_rate":       calculateRate(toInt(totalInsights), toInt(totalAsked)),
	}

	return stats, nil
}

// GetPrincipleViolationStats returns statistics on principle violations
func (m *MetricsRepository) GetPrincipleViolationStats(userID string) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_violations,
			SUM(CASE WHEN severity = 'critical' THEN 1 ELSE 0 END) as critical_count,
			SUM(CASE WHEN severity = 'high' THEN 1 ELSE 0 END) as high_count,
			SUM(CASE WHEN severity = 'medium' THEN 1 ELSE 0 END) as medium_count,
			SUM(CASE WHEN resolved = true THEN 1 ELSE 0 END) as resolved_count
		FROM principle_violations
		WHERE user_id = ?
	`

	var totalViolations, criticalCount, highCount, mediumCount, resolvedCount sql.NullInt64
	err := m.db.GetConnection().QueryRow(query, userID).Scan(
		&totalViolations, &criticalCount, &highCount, &mediumCount, &resolvedCount,
	)

	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_violations":    toInt(totalViolations),
		"critical":            toInt(criticalCount),
		"high":                toInt(highCount),
		"medium":              toInt(mediumCount),
		"resolved":            toInt(resolvedCount),
		"resolution_rate":     calculateRate(toInt(resolvedCount), toInt(totalViolations)),
	}

	return stats, nil
}

// GetPrincipleBreakdown returns violations grouped by principle
func (m *MetricsRepository) GetPrincipleBreakdown(userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			principle_name,
			severity,
			COUNT(*) as count,
			SUM(CASE WHEN resolved THEN 1 ELSE 0 END) as resolved
		FROM principle_violations
		WHERE user_id = ?
		GROUP BY principle_name, severity
		ORDER BY principle_name,
			CASE severity WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END
	`

	rows, err := m.db.GetConnection().Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var breakdown []map[string]interface{}
	for rows.Next() {
		var principle, severity string
		var count, resolved sql.NullInt64

		if err := rows.Scan(&principle, &severity, &count, &resolved); err != nil {
			return nil, err
		}

		breakdown = append(breakdown, map[string]interface{}{
			"principle": principle,
			"severity":  severity,
			"count":     toInt(count),
			"resolved":  toInt(resolved),
		})
	}

	return breakdown, rows.Err()
}

// GetApproachComparison returns effectiveness comparison across all approaches
func (m *MetricsRepository) GetApproachComparison(userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			socratic_approach,
			COUNT(*) as total_used,
			SUM(CASE WHEN reduced_ambiguity THEN 1 ELSE 0 END) as successful,
			AVG(CASE WHEN depth_level_advanced THEN 1 ELSE 0 END) as avg_depth_advancement,
			AVG(CASE WHEN insight_gained THEN 1 ELSE 0 END) as avg_insight_rate
		FROM question_effectiveness
		WHERE user_id = ?
		GROUP BY socratic_approach
		ORDER BY successful DESC
	`

	rows, err := m.db.GetConnection().Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comparison []map[string]interface{}
	for rows.Next() {
		var approach string
		var totalUsed, successful sql.NullInt64
		var avgDepthAdvancement, avgInsightRate sql.NullFloat64

		if err := rows.Scan(&approach, &totalUsed, &successful, &avgDepthAdvancement, &avgInsightRate); err != nil {
			return nil, err
		}

		comparison = append(comparison, map[string]interface{}{
			"approach":           approach,
			"total_used":         toInt(totalUsed),
			"successful":         toInt(successful),
			"success_rate":       calculateRate(toInt(successful), toInt(totalUsed)),
			"avg_depth_advancement": toFloat(avgDepthAdvancement),
			"avg_insight_rate":   toFloat(avgInsightRate),
		})
	}

	return comparison, rows.Err()
}

// RecordViolation records a principle violation
func (m *MetricsRepository) RecordViolation(
	userID string,
	principleName string,
	severity string, // "critical", "high", "medium", "low"
	details string,
) error {
	query := `
		INSERT INTO principle_violations (user_id, principle_name, severity, details, detected_at, resolved)
		VALUES (?, ?, ?, ?, ?, false)
	`

	now := time.Now().Unix()
	_, err := m.db.Exec(query, userID, principleName, severity, details, now)
	if err != nil {
		log.Printf("[MetricsRepository] ERROR recording principle violation: %v", err)
	} else {
		log.Printf("[MetricsRepository] ✓ Principle violation recorded: principle=%s severity=%s", principleName, severity)
	}

	return err
}

// Helper functions
func toInt(n sql.NullInt64) int {
	if n.Valid {
		return int(n.Int64)
	}
	return 0
}

func toFloat(n sql.NullFloat64) float64 {
	if n.Valid {
		return n.Float64
	}
	return 0.0
}

func calculateRate(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0.0
	}
	return float64(numerator) / float64(denominator)
}

// QuestionHistoryRepository manages Socratic question history tracking
type QuestionHistoryRepository struct {
	db *Database
}

// NewQuestionHistoryRepository creates a new question history repository
func NewQuestionHistoryRepository(db *Database) *QuestionHistoryRepository {
	return &QuestionHistoryRepository{db: db}
}

// RecordQuestion saves a Socratic question that was asked to the user
func (qhr *QuestionHistoryRepository) RecordQuestion(
	userID string,
	conversationID string,
	question *models.SocraticQuestion,
	emotionState string,
	riskLevel string,
) error {
	log.Printf("[QuestionHistory] Recording question %s for user %s in conversation %s",
		question.ID, userID, conversationID)

	query := `
		INSERT INTO question_history
		(user_id, conversation_id, question_id, question_text, asked_at,
		 emotion_state, risk_level, depth_level, socratic_approach, category)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now().Unix()
	_, err := qhr.db.Exec(query,
		userID,
		conversationID,
		question.ID,
		question.Text,
		now,
		emotionState,
		riskLevel,
		question.DepthLevel,
		question.SocraticApproach,
		question.Category,
	)

	if err != nil {
		log.Printf("[QuestionHistory] ERROR recording question: %v", err)
		return err
	}

	log.Printf("[QuestionHistory] Question recorded successfully")
	return nil
}

// GetAskedQuestionsInConversation retrieves all Socratic questions asked in a conversation
func (qhr *QuestionHistoryRepository) GetAskedQuestionsInConversation(
	userID string,
	conversationID string,
) ([]models.SocraticQuestion, error) {
	log.Printf("[QuestionHistory] Retrieving asked questions for user %s in conversation %s",
		userID, conversationID)

	query := `
		SELECT question_id, question_text, depth_level, socratic_approach, category
		FROM question_history
		WHERE user_id = ? AND conversation_id = ?
		ORDER BY asked_at ASC
	`

	rows, err := qhr.db.Query(query, userID, conversationID)
	if err != nil {
		log.Printf("[QuestionHistory] ERROR querying questions: %v", err)
		return nil, err
	}
	defer rows.Close()

	var questions []models.SocraticQuestion
	for rows.Next() {
		var q models.SocraticQuestion
		if err := rows.Scan(&q.ID, &q.Text, &q.DepthLevel, &q.SocraticApproach, &q.Category); err != nil {
			log.Printf("[QuestionHistory] ERROR scanning row: %v", err)
			return nil, err
		}
		questions = append(questions, q)
	}

	log.Printf("[QuestionHistory] Retrieved %d asked questions", len(questions))
	return questions, nil
}

// RecordAnswer saves the user's response to a Socratic question
func (qhr *QuestionHistoryRepository) RecordAnswer(
	userID string,
	conversationID string,
	questionID string,
	userResponse string,
) error {
	log.Printf("[QuestionHistory] Recording answer to question %s", questionID)

	query := `
		UPDATE question_history
		SET user_response = ?, response_length = ?
		WHERE user_id = ? AND conversation_id = ? AND question_id = ?
		AND user_response IS NULL
	`

	result, err := qhr.db.Exec(query, userResponse, len(userResponse), userID, conversationID, questionID)
	if err != nil {
		log.Printf("[QuestionHistory] ERROR recording answer: %v", err)
		return err
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		log.Printf("[QuestionHistory] Warning: No pending question found for ID %s", questionID)
	} else {
		log.Printf("[QuestionHistory] Answer recorded successfully")
	}

	return nil
}

// GetPreviousQuestions retrieves questions asked to a user in past conversations
func (qhr *QuestionHistoryRepository) GetPreviousQuestions(
	userID string,
	limit int,
) ([]map[string]interface{}, error) {
	log.Printf("[QuestionHistory] Retrieving previous questions for user %s (limit: %d)", userID, limit)

	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, question, user_response, response_length, emotion_state, risk_level, asked_at
		FROM question_history
		WHERE user_id = ? AND user_response IS NOT NULL
		ORDER BY asked_at DESC
		LIMIT ?
	`

	rows, err := qhr.db.Query(query, userID, limit)
	if err != nil {
		log.Printf("[QuestionHistory] ERROR querying previous questions: %v", err)
		return nil, err
	}
	defer rows.Close()

	var questions []map[string]interface{}
	for rows.Next() {
		var id, question, response, emotionState, riskLevel string
		var responseLength int
		var askedAt int64

		if err := rows.Scan(&id, &question, &response, &responseLength, &emotionState, &riskLevel, &askedAt); err != nil {
			log.Printf("[QuestionHistory] ERROR scanning row: %v", err)
			continue
		}

		q := map[string]interface{}{
			"id":               id,
			"question":         question,
			"userResponse":     response,
			"responseLength":   responseLength,
			"emotionState":     emotionState,
			"riskLevel":        riskLevel,
			"askedAt":          askedAt,
		}
		questions = append(questions, q)
	}

	log.Printf("[QuestionHistory] Retrieved %d previous questions", len(questions))
	return questions, nil
}

// AuditLogRepository manages audit log records
type AuditLogRepository struct {
	db *Database
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *Database) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// RecordAction records an audit log entry
func (a *AuditLogRepository) RecordAction(
	userID string,
	action string,
	details map[string]interface{},
) error {
	log.Printf("[AuditLog] Recording action: user=%s action=%s", userID, action)

	id := fmt.Sprintf("audit_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
	now := time.Now().Unix()

	// Marshal details to JSON
	detailsJSON, _ := json.Marshal(details)

	query := `
		INSERT INTO audit_log (id, user_id, action, details, timestamp)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := a.db.Exec(query, id, userID, action, string(detailsJSON), now)
	if err != nil {
		log.Printf("[AuditLog] ERROR recording action: %v", err)
	} else {
		log.Printf("[AuditLog] ✓ Action recorded: %s", action)
	}

	return err
}
