package database

import (
	"database/sql"
	"fmt"
)

// SentenceAnalysisRepository handles database operations for sentence analyses
type SentenceAnalysisRepository struct {
	db *sql.DB
}

// NewSentenceAnalysisRepository creates a new repository
func NewSentenceAnalysisRepository(db *sql.DB) *SentenceAnalysisRepository {
	return &SentenceAnalysisRepository{db: db}
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
