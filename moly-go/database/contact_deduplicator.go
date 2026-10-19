package database

import (
	"log"
	"sort"
	"strings"
	"time"

	"moly/models"
)

// ContactDeduplicator detects and merges duplicate contacts during extraction
// Handles: generic→specific naming (Not specified→Kyle), pronoun resolution (He→Kyle)
type ContactDeduplicator struct {
	db           *Database
	contactRepo  *ContactRepository
	conflictRepo *ContextConflictRepository
}

// NewContactDeduplicator creates a new contact deduplicator
func NewContactDeduplicator(db *Database) *ContactDeduplicator {
	return &ContactDeduplicator{
		db:           db,
		contactRepo:  NewContactRepository(db),
		conflictRepo: db.GetContextConflictRepository(),
	}
}

// DeduplicationDecision represents the decision about what to do with a contact
type DeduplicationDecision struct {
	ShouldMerge       bool             // Whether to merge with existing contact
	ShouldSkipSave    bool             // Whether to skip the normal save (merge handles it)
	TargetContact     *models.Contact  // Contact to merge into (if ShouldMerge=true)
	NeedsUserApproval bool             // Whether user needs to approve merge
	ApprovalConflict  *ContextConflict // Conflict record if approval needed
	MergeReason       string           // Why merge was chosen
	Confidence        float64          // How confident is this decision (0-1)
}

// Helper functions

// DeduplicateBySubject performs subject-based contact deduplication using ExtractionArtifact
// Phase 4 integration: Groups extracted contacts by their subject (who said/has what)
// This solves multi-person tracking where "Christine" and "the girl" both have subject="her"
func (cd *ContactDeduplicator) DeduplicateBySubject(
	userID string,
	artifact *models.ExtractionArtifact,
) (map[string][]*models.ExtractedEntity, int, error) {

	if artifact == nil || len(artifact.Entities) == 0 {
		return make(map[string][]*models.ExtractedEntity), 0, nil
	}

	log.Printf("[ContactDeduplicator] Subject-based deduplication (Phase 4): artifact has %d entities, subject_attributed=%v",
		len(artifact.Entities), artifact.SubjectAttributed)

	// Group entities by subject
	entitiesBySubject := make(map[string][]*models.ExtractedEntity)
	contactCount := 0

	for i, entity := range artifact.Entities {
		// Only track contact entities
		if entity.Type != "contact" {
			continue
		}

		// Use subject as grouping key (e.g., "her", "his", "user", "you", "them")
		subject := entity.Subject
		if subject == "" || subject == "unknown" {
			subject = "unattributed"
		}

		entitiesBySubject[subject] = append(entitiesBySubject[subject], &artifact.Entities[i])
		contactCount++

		log.Printf("[ContactDeduplicator]   - Contact: %s (subject=%s, confidence=%.2f)",
			entity.Value, entity.Subject, entity.Confidence)
	}

	log.Printf("[ContactDeduplicator] Grouped %d contacts into %d subjects",
		contactCount, len(entitiesBySubject))

	// Log groupings for debugging multi-person cases
	for subject, entities := range entitiesBySubject {
		if len(entities) > 1 {
			names := []string{}
			for _, e := range entities {
				names = append(names, e.Value)
			}
			log.Printf("[ContactDeduplicator] Subject '%s' has %d descriptors: %v",
				subject, len(entities), names)
		}
	}

	return entitiesBySubject, contactCount, nil
}

// MergeContactsBySubject combines contact entities that refer to the same subject
// Returns merged contacts ready for database insertion
func (cd *ContactDeduplicator) MergeContactsBySubject(
	entitiesBySubject map[string][]*models.ExtractedEntity,
) []*models.Contact {

	var mergedContacts []*models.Contact

	// Iterate subjects in sorted order so the output is the same on every run.
	subjects := make([]string, 0, len(entitiesBySubject))
	for subject := range entitiesBySubject {
		subjects = append(subjects, subject)
	}
	sort.Strings(subjects)

	for _, subject := range subjects {
		entities := entitiesBySubject[subject]
		if len(entities) == 0 {
			continue
		}

		// Use highest-confidence entity as base
		var baseEntity *models.ExtractedEntity
		for _, e := range entities {
			if baseEntity == nil || e.Confidence > baseEntity.Confidence {
				baseEntity = e
			}
		}

		if baseEntity == nil {
			continue
		}

		// Collect all characteristics from all entities with this subject
		characteristicsMap := make(map[string]bool)
		for _, e := range entities {
			if e.Value != "" {
				characteristicsMap[strings.ToLower(e.Value)] = true
			}
		}

		var characteristics []string
		for c := range characteristicsMap {
			characteristics = append(characteristics, c)
		}

		// Create merged contact
		contact := &models.Contact{
			Name:             baseEntity.Value,
			Relationship:     "", // Will be determined by other layers
			Characteristics:  characteristics,
			Confidence:       baseEntity.Confidence,
			CreatedVia:       "extraction_artifact",
			Status:           "active",
			FirstMentionedAt: time.Now().Unix(),
			LastMentionedAt:  time.Now().Unix(),
			ExtractionCount:  len(entities),
		}

		mergedContacts = append(mergedContacts, contact)

		log.Printf("[ContactDeduplicator] Merged subject '%s': %d variants → '%s' (%d characteristics)",
			subject, len(entities), contact.Name, len(characteristics))
	}

	return mergedContacts
}
