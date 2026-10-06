package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// SentenceAnalysisRepository handles database operations for sentence analyses
type SentenceAnalysisRepository struct {
	db *sql.DB
}

// NewSentenceAnalysisRepository creates a new repository
func NewSentenceAnalysisRepository(db *sql.DB) *SentenceAnalysisRepository {
	return &SentenceAnalysisRepository{db: db}
}

// SaveSentenceAnalysis saves a sentence analysis to the database
func (r *SentenceAnalysisRepository) SaveSentenceAnalysis(
	userID string,
	messageID string,
	conversationID string,
	sentenceText string,
	sentenceNumber int,
	subject string,
	subjectType string,
	verb string,
	verbType string,
	verbNegated bool,
	object string,
	objectType string,
	negated bool,
	confidence float64,
	parsingMethod string,
) (int64, error) {

	// FIX #32: Validate sentence analysis before save
	if userID == "" || len(userID) > 255 {
		return 0, fmt.Errorf("userId required and must be <= 255 chars")
	}
	if messageID == "" || len(messageID) > 255 {
		return 0, fmt.Errorf("messageId required and must be <= 255 chars")
	}
	if sentenceText == "" || len(sentenceText) > 5000 {
		return 0, fmt.Errorf("sentenceText required and must be <= 5000 chars")
	}
	if confidence < 0 || confidence > 1 {
		return 0, fmt.Errorf("confidence must be in range [0,1], got %.2f", confidence)
	}
	if sentenceNumber < 0 {
		return 0, fmt.Errorf("sentenceNumber must be >= 0")
	}

	query := `
		INSERT INTO sentence_analyses (
			user_id, message_id, conversation_id,
			sentence_text, sentence_number, sentence_start_pos, sentence_end_pos,
			subject, subject_type, verb, verb_type, verb_negated,
			object, object_type, negation,
			confidence, parsing_method, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		userID, messageID, conversationID,
		sentenceText, sentenceNumber, 0, len(sentenceText),
		subject, subjectType, verb, verbType, verbNegated,
		object, objectType, negated,
		confidence, parsingMethod, time.Now().Unix(),
	)

	if err != nil {
		log.Printf("[SentenceAnalysisRepo] ERROR saving sentence: %v", err)
		return 0, fmt.Errorf("failed to save sentence analysis: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("[SentenceAnalysisRepo] ERROR getting last insert id: %v", err)
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	log.Printf("[SentenceAnalysisRepo] ✅ Saved sentence analysis (id=%d, user=%s, sentence=%d)", id, userID, sentenceNumber)
	return id, nil
}

// SentenceAnalysisData represents a stored sentence analysis (avoids import cycle)
type SentenceAnalysisData struct {
	ID                int64
	SentenceNumber    int
	SentenceText      string
	Subject           string
	SubjectType       string
	SubjectResolvedTo string
	Verb              string
	VerbType          string
	VerbNegated       bool
	Object            string
	ObjectType        string
	Negation          bool
	Confidence        float64
	ParsingMethod     string
	CreatedAt         int64
}

// GetSentenceAnalysesByMessage retrieves all sentence analyses for a message
func (r *SentenceAnalysisRepository) GetSentenceAnalysesByMessage(
	userID string,
	messageID string,
) ([]*SentenceAnalysisData, error) {

	query := `
		SELECT
			id, sentence_number, sentence_text, subject, subject_type,
			subject_resolved_to, verb, verb_type, verb_negated, object, object_type,
			negation, confidence, parsing_method, created_at
		FROM sentence_analyses
		WHERE user_id = ? AND message_id = ?
		ORDER BY sentence_number ASC
	`

	rows, err := r.db.Query(query, userID, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sentence analyses: %w", err)
	}
	defer rows.Close()

	var analyses []*SentenceAnalysisData
	for rows.Next() {
		analysis := &SentenceAnalysisData{}
		var resolvedTo sql.NullString

		err := rows.Scan(
			&analysis.ID, &analysis.SentenceNumber, &analysis.SentenceText, &analysis.Subject, &analysis.SubjectType,
			&resolvedTo, &analysis.Verb, &analysis.VerbType, &analysis.VerbNegated,
			&analysis.Object, &analysis.ObjectType, &analysis.Negation,
			&analysis.Confidence, &analysis.ParsingMethod, &analysis.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan sentence analysis: %w", err)
		}

		if resolvedTo.Valid {
			analysis.SubjectResolvedTo = resolvedTo.String
		}

		analyses = append(analyses, analysis)
	}

	return analyses, rows.Err()
}

// UpdateSubjectResolution updates the resolved subject for a sentence
func (r *SentenceAnalysisRepository) UpdateSubjectResolution(
	messageID string,
	sentenceNumber int,
	resolvedTo string,
) error {

	query := `
		UPDATE sentence_analyses
		SET subject_resolved_to = ?
		WHERE message_id = ? AND sentence_number = ?
	`

	_, err := r.db.Exec(query, resolvedTo, messageID, sentenceNumber)
	if err != nil {
		return fmt.Errorf("failed to update subject resolution: %w", err)
	}

	return nil
}

// PronounResolutionData represents stored pronoun resolution
type PronounResolutionData struct {
	ID               int64
	Pronoun          string
	PronounType      string
	AntecedentType   string
	AntecedentValue  string
	AntecedentID     *int64
	Confidence       float64
	EvidenceText     string
	ResolutionMethod string
	ScopeStartSeq    int
	ScopeEndSeq      *int
	IsActive         bool
	CreatedAt        int64
}

// PronounResolutionRepository handles database operations for pronoun resolutions
type PronounResolutionRepository struct {
	db *sql.DB
}

// NewPronounResolutionRepository creates a new repository
func NewPronounResolutionRepository(db *sql.DB) *PronounResolutionRepository {
	return &PronounResolutionRepository{db: db}
}

// SavePronounResolution saves a pronoun resolution to the database
func (r *PronounResolutionRepository) SavePronounResolution(
	userID string,
	conversationID string,
	pronoun string,
	pronounType string,
	antecedentType string,
	antecedentValue string,
	antecedentID *int64,
	messageID string,
	sentencePosition int,
	confidence float64,
	evidenceText string,
	resolutionMethod string,
	scopeStartSeq int,
	scopeEndSeq *int,
	scopeStartMessageID string,
	scopeEndMessageID *string,
) (int64, error) {
	// FIX #32: Validate pronoun resolution before save
	if userID == "" || len(userID) > 255 || conversationID == "" || len(conversationID) > 255 {
		return 0, fmt.Errorf("userId and conversationId required and must be <= 255 chars")
	}
	if pronoun == "" || pronounType == "" || antecedentType == "" {
		return 0, fmt.Errorf("pronoun, pronounType, and antecedentType required")
	}
	if confidence < 0 || confidence > 1 {
		return 0, fmt.Errorf("confidence must be in range [0,1], got %.2f", confidence)
	}

	query := `
		INSERT INTO pronoun_resolutions (
			user_id, conversation_id, pronoun, pronoun_type,
			antecedent_type, antecedent_value, antecedent_id,
			message_id, sentence_position, confidence, evidence_text,
			resolution_method, scope_start_seq, scope_end_seq,
			scope_start_message_id, scope_end_message_id, is_active,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now().Unix()
	result, err := r.db.Exec(
		query,
		userID, conversationID, pronoun, pronounType,
		antecedentType, antecedentValue, antecedentID,
		messageID, sentencePosition, confidence, evidenceText,
		resolutionMethod, scopeStartSeq, scopeEndSeq,
		scopeStartMessageID, scopeEndMessageID, 1,
		now, now,
	)

	if err != nil {
		log.Printf("[PronounResolutionRepo] ERROR saving pronoun resolution: %v", err)
		return 0, fmt.Errorf("failed to save pronoun resolution: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("[PronounResolutionRepo] ERROR getting last insert id: %v", err)
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	log.Printf("[PronounResolutionRepo] ✅ Saved pronoun resolution (id=%d, pronoun=%s, antecedent=%s, confidence=%.2f)", id, pronoun, antecedentValue, confidence)
	return id, nil
}

// GetActivePronounResolution retrieves the active resolution for a pronoun
func (r *PronounResolutionRepository) GetActivePronounResolution(
	userID string,
	pronoun string,
) (*PronounResolutionData, error) {

	query := `
		SELECT
			id, pronoun, pronoun_type, antecedent_type, antecedent_value, antecedent_id,
			confidence, evidence_text, resolution_method,
			scope_start_seq, scope_end_seq, is_active, created_at
		FROM pronoun_resolutions
		WHERE user_id = ? AND pronoun = ? AND is_active = 1
		ORDER BY created_at DESC
		LIMIT 1
	`

	resolution := &PronounResolutionData{}

	var scopeEnd sql.NullInt64
	var antecedentID sql.NullInt64

	err := r.db.QueryRow(query, userID, pronoun).Scan(
		&resolution.ID, &resolution.Pronoun, &resolution.PronounType,
		&resolution.AntecedentType, &resolution.AntecedentValue, &antecedentID,
		&resolution.Confidence,
		&resolution.EvidenceText, &resolution.ResolutionMethod,
		&resolution.ScopeStartSeq, &scopeEnd, &resolution.IsActive, &resolution.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query pronoun resolution: %w", err)
	}

	if antecedentID.Valid {
		resolution.AntecedentID = &antecedentID.Int64
	}

	if scopeEnd.Valid {
		se := int(scopeEnd.Int64)
		resolution.ScopeEndSeq = &se
	}

	return resolution, nil
}

// GroupReferenceData represents stored group reference
type GroupReferenceData struct {
	ID               int64
	ReferencePronoun string
	ReferenceType    string
	Members          []string
	MemberIDs        map[string]int64
	IsUserInGroup    bool
	GroupContext     string
	Confidence       float64
	EvidenceText     string
	IsActive         bool
	CreatedAt        int64
}

// GroupReferenceRepository handles database operations for group references
type GroupReferenceRepository struct {
	db *sql.DB
}

// NewGroupReferenceRepository creates a new repository
func NewGroupReferenceRepository(db *sql.DB) *GroupReferenceRepository {
	return &GroupReferenceRepository{db: db}
}

// SaveGroupReference saves a group reference to the database
func (r *GroupReferenceRepository) SaveGroupReference(
	userID string,
	conversationID string,
	referencePronoun string,
	referenceType string,
	members []string,
	isUserInGroup bool,
	groupContext string,
	messageID string,
	confidence float64,
	evidenceText string,
	scopeEndMessageID *string,
	scopeEndSeq *int,
) (int64, error) {
	// FIX #32: Validate group reference before save
	if userID == "" || len(userID) > 255 || conversationID == "" || len(conversationID) > 255 {
		return 0, fmt.Errorf("userId and conversationId required and must be <= 255 chars")
	}
	if referencePronoun == "" || referenceType == "" {
		return 0, fmt.Errorf("referencePronoun and referenceType required")
	}
	if confidence < 0 || confidence > 1 {
		return 0, fmt.Errorf("confidence must be in range [0,1], got %.2f", confidence)
	}

	// Convert members to JSON (names)
	memberNamesJSON, err := json.Marshal(members)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal member names: %w", err)
	}

	// Create empty member IDs for now (would be filled in when contact IDs are available)
	memberIDsJSON, _ := json.Marshal([]int64{})

	query := `
		INSERT INTO group_references (
			user_id, conversation_id, reference_pronoun, reference_type,
			member_ids, member_names, is_user_in_group, group_context,
			message_id, scope_end_message_id, scope_end_seq,
			established_at, confidence, evidence_text,
			is_active, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now().Unix()
	result, err := r.db.Exec(
		query,
		userID, conversationID, referencePronoun, referenceType,
		string(memberIDsJSON), string(memberNamesJSON), isUserInGroup, groupContext,
		messageID, scopeEndMessageID, scopeEndSeq,
		now, confidence, evidenceText,
		1, now,
	)

	if err != nil {
		log.Printf("[GroupReferenceRepo] ERROR saving group reference: %v", err)
		return 0, fmt.Errorf("failed to save group reference: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Printf("[GroupReferenceRepo] ERROR getting last insert id: %v", err)
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	log.Printf("[GroupReferenceRepo] ✅ Saved group reference (id=%d, pronoun=%s, members=%d, confidence=%.2f)", id, referencePronoun, len(members), confidence)
	return id, nil
}

// GetActiveGroupReference retrieves the active group reference for a pronoun
func (r *GroupReferenceRepository) GetActiveGroupReference(
	userID string,
	pronoun string,
) (*GroupReferenceData, error) {

	query := `
		SELECT
			id, reference_pronoun, reference_type, member_ids,
			is_user_in_group, group_context, confidence, evidence_text,
			is_active, created_at
		FROM group_references
		WHERE user_id = ? AND reference_pronoun = ? AND is_active = 1
		ORDER BY created_at DESC
		LIMIT 1
	`

	ref := &GroupReferenceData{
		Members:   []string{},
		MemberIDs: make(map[string]int64),
	}

	var memberJSON string

	err := r.db.QueryRow(query, userID, pronoun).Scan(
		&ref.ID, &ref.ReferencePronoun, &ref.ReferenceType, &memberJSON,
		&ref.IsUserInGroup, &ref.GroupContext, &ref.Confidence, &ref.EvidenceText,
		&ref.IsActive, &ref.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query group reference: %w", err)
	}

	// Unmarshal members
	if err := json.Unmarshal([]byte(memberJSON), &ref.Members); err != nil {
		// Continue - members may be empty
	}

	return ref, nil
}

// ExtractionSentenceLinkingRepository handles database operations for extraction sentence linking
type ExtractionSentenceLinkingRepository struct {
	db *sql.DB
}

// NewExtractionSentenceLinkingRepository creates a new repository
func NewExtractionSentenceLinkingRepository(db *sql.DB) *ExtractionSentenceLinkingRepository {
	return &ExtractionSentenceLinkingRepository{db: db}
}

// SaveExtractionLink saves a link between an extracted fact and its source sentence
func (r *ExtractionSentenceLinkingRepository) SaveExtractionLink(
	contextAttributeID int64,
	sentenceAnalysisID int64,
	subjectResolvedTo string,
	subjectType string,
	confidence float64,
) error {
	// FIX #32: Validate extraction link before save
	if contextAttributeID <= 0 || sentenceAnalysisID <= 0 {
		return fmt.Errorf("contextAttributeID and sentenceAnalysisID must be > 0")
	}
	if subjectType == "" {
		return fmt.Errorf("subjectType required")
	}
	if confidence < 0 || confidence > 1 {
		return fmt.Errorf("confidence must be in range [0,1], got %.2f", confidence)
	}

	query := `
		INSERT INTO extraction_sentence_linking (
			context_attribute_id, sentence_analysis_id,
			subject_resolved_to, subject_type, attribution_confidence,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(context_attribute_id) DO UPDATE SET
			sentence_analysis_id = excluded.sentence_analysis_id,
			subject_resolved_to = excluded.subject_resolved_to,
			subject_type = excluded.subject_type,
			attribution_confidence = excluded.attribution_confidence
	`

	_, err := r.db.Exec(
		query,
		contextAttributeID, sentenceAnalysisID,
		subjectResolvedTo, subjectType, confidence,
		time.Now().Unix(),
	)

	if err != nil {
		log.Printf("[ExtractionLinkingRepo] ERROR saving extraction link: %v", err)
		return fmt.Errorf("failed to save extraction link: %w", err)
	}

	log.Printf("[ExtractionLinkingRepo] ✅ Saved extraction link (contextAttr=%d, sentence=%d, subject=%s)", contextAttributeID, sentenceAnalysisID, subjectResolvedTo)
	return nil
}

// GetExtractionLink retrieves the link for an extracted fact
func (r *ExtractionSentenceLinkingRepository) GetExtractionLink(
	contextAttributeID int64,
) (sentenceAnalysisID int64, resolvedTo string, confidence float64, err error) {

	query := `
		SELECT sentence_analysis_id, subject_resolved_to, attribution_confidence
		FROM extraction_sentence_linking
		WHERE context_attribute_id = ?
	`

	err = r.db.QueryRow(query, contextAttributeID).Scan(
		&sentenceAnalysisID, &resolvedTo, &confidence,
	)

	if err == sql.ErrNoRows {
		return 0, "", 0, nil
	}
	if err != nil {
		return 0, "", 0, fmt.Errorf("failed to query extraction link: %w", err)
	}

	return sentenceAnalysisID, resolvedTo, confidence, nil
}
