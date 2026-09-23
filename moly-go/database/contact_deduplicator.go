package database

import (
	"fmt"
	"log"
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
	ShouldMerge       bool                   // Whether to merge with existing contact
	ShouldSkipSave    bool                   // Whether to skip the normal save (merge handles it)
	TargetContact     *models.Contact        // Contact to merge into (if ShouldMerge=true)
	NeedsUserApproval bool                   // Whether user needs to approve merge
	ApprovalConflict  *ContextConflict       // Conflict record if approval needed
	MergeReason       string                 // Why merge was chosen
	Confidence        float64                // How confident is this decision (0-1)
}

// CheckForDuplicate analyzes extracted contact and detects if it's a duplicate of existing contact
func (cd *ContactDeduplicator) CheckForDuplicate(
	userID string,
	extractedContact *models.ExtractedContact,
) (*DeduplicationDecision, error) {

	if extractedContact == nil || extractedContact.Name == "" {
		return &DeduplicationDecision{
			ShouldMerge:    false,
			ShouldSkipSave: false,
		}, nil
	}

	log.Printf("[ContactDeduplicator] Checking for duplicate: %s (%s)", extractedContact.Name, extractedContact.Relationship)

	// Load all active contacts for user
	existingContacts, err := cd.contactRepo.GetByUserID(userID)
	if err != nil {
		log.Printf("[ContactDeduplicator] Warning: Failed to load existing contacts: %v", err)
		// On error, don't merge - let normal save proceed
		return &DeduplicationDecision{
			ShouldMerge:    false,
			ShouldSkipSave: false,
		}, nil
	}

	if len(existingContacts) == 0 {
		log.Printf("[ContactDeduplicator] No existing contacts, no duplicates to check")
		return &DeduplicationDecision{
			ShouldMerge:    false,
			ShouldSkipSave: false,
		}, nil
	}

	// Check if this is a generic name (needs mapping to existing contact)
	if cd.isGenericName(extractedContact.Name) {
		log.Printf("[ContactDeduplicator] Extracted name is generic: %s", extractedContact.Name)
		return cd.handleGenericName(userID, extractedContact, existingContacts)
	}

	// Check if this is a specific name that matches existing generic contact
	if cd.hasGenericContacts(existingContacts) {
		log.Printf("[ContactDeduplicator] User has generic contacts, checking if '%s' identifies one", extractedContact.Name)
		return cd.handleSpecificNameIdentifyingGeneric(userID, extractedContact, existingContacts)
	}

	log.Printf("[ContactDeduplicator] No duplicates detected for %s", extractedContact.Name)
	return &DeduplicationDecision{
		ShouldMerge:    false,
		ShouldSkipSave: false,
	}, nil
}

// handleGenericName handles case where extracted name is generic ("Not specified", "He", "She")
func (cd *ContactDeduplicator) handleGenericName(
	userID string,
	extractedContact *models.ExtractedContact,
	existingContacts []*models.Contact,
) (*DeduplicationDecision, error) {

	// Find contacts with matching relationship
	candidates := cd.filterByRelationship(existingContacts, extractedContact.Relationship)

	log.Printf("[ContactDeduplicator] Found %d candidates with relationship '%s'", len(candidates), extractedContact.Relationship)

	if len(candidates) == 0 {
		// No matches - allow normal save
		log.Printf("[ContactDeduplicator] No matching contacts by relationship, will create new")
		return &DeduplicationDecision{
			ShouldMerge:    false,
			ShouldSkipSave: false,
		}, nil
	}

	if len(candidates) == 1 {
		// One match - high confidence merge
		targetContact := candidates[0]
		log.Printf("[ContactDeduplicator] AUTO-MERGE: Generic '%s' → %s (relationship match, confidence=0.95)",
			extractedContact.Name, targetContact.Name)

		// Merge the extracted contact into target
		err := cd.mergeContactsAndUpdate(userID, targetContact, extractedContact)
		if err != nil {
			log.Printf("[ContactDeduplicator] Error during merge: %v", err)
			// On merge error, proceed with normal save (don't lose data)
			return &DeduplicationDecision{
				ShouldMerge:    false,
				ShouldSkipSave: false,
			}, nil
		}

		return &DeduplicationDecision{
			ShouldMerge:       true,
			ShouldSkipSave:    true,
			TargetContact:     targetContact,
			NeedsUserApproval: false,
			MergeReason:       "Generic name mapped to single existing contact with matching relationship",
			Confidence:        0.95,
		}, nil
	}

	// Multiple candidates - ambiguous, queue for approval
	log.Printf("[ContactDeduplicator] AMBIGUOUS: Generic '%s' matches %d existing contacts, queueing for user approval",
		extractedContact.Name, len(candidates))

	conflict := &ContextConflict{
		UserID:         userID,
		ConflictType:   "contact_ambiguous_generic_name",
		Severity:       "low",
		SavedValue:     fmt.Sprintf("%v", cd.contactNamesToString(candidates)),
		ExtractedValue: extractedContact.Name,
		Description: fmt.Sprintf(
			"Extracted generic '%s' (%s) could refer to: %s. Which one?",
			extractedContact.Name,
			extractedContact.Relationship,
			cd.contactNamesToString(candidates),
		),
		Status:    "unresolved",
		CreatedAt: time.Now().Unix(),
	}

	err := cd.conflictRepo.Save(conflict)
	if err != nil {
		log.Printf("[ContactDeduplicator] Warning: Failed to save conflict: %v", err)
		// Proceed with normal save if conflict save fails
		return &DeduplicationDecision{
			ShouldMerge:    false,
			ShouldSkipSave: false,
		}, nil
	}

	return &DeduplicationDecision{
		ShouldMerge:       false,
		ShouldSkipSave:    true, // Skip normal save, wait for user approval
		NeedsUserApproval: true,
		ApprovalConflict:  conflict,
		MergeReason:       "Ambiguous generic name, user must choose",
		Confidence:        0.60,
	}, nil
}

// handleSpecificNameIdentifyingGeneric handles case where specific name (Kyle) identifies generic contact (Not specified)
func (cd *ContactDeduplicator) handleSpecificNameIdentifyingGeneric(
	userID string,
	extractedContact *models.ExtractedContact,
	existingContacts []*models.Contact,
) (*DeduplicationDecision, error) {

	// Find generic contacts with matching relationship
	genericCandidates := cd.getGenericContactsWithRelationship(existingContacts, extractedContact.Relationship)

	if len(genericCandidates) == 0 {
		// No generic contacts with this relationship - normal save
		log.Printf("[ContactDeduplicator] No generic contacts match relationship '%s'", extractedContact.Relationship)
		return &DeduplicationDecision{
			ShouldMerge:    false,
			ShouldSkipSave: false,
		}, nil
	}

	// Only one generic contact - this is likely the identification
	if len(genericCandidates) == 1 {
		genericContact := genericCandidates[0]
		similarity := cd.calculateSimilarity(genericContact, extractedContact)

		log.Printf("[ContactDeduplicator] Specific '%s' may identify generic '%s' (similarity=%.2f)",
			extractedContact.Name, genericContact.Name, similarity)

		if similarity >= 0.85 {
			// High confidence - merge
			log.Printf("[ContactDeduplicator] HIGH CONFIDENCE: Merging '%s' into '%s'",
				genericContact.Name, extractedContact.Name)

			err := cd.mergeGenericIntoSpecific(userID, genericContact, extractedContact)
			if err != nil {
				log.Printf("[ContactDeduplicator] Error during merge: %v", err)
				return &DeduplicationDecision{
					ShouldMerge:    false,
					ShouldSkipSave: false,
				}, nil
			}

			return &DeduplicationDecision{
				ShouldMerge:       true,
				ShouldSkipSave:    true,
				TargetContact:     genericContact,
				NeedsUserApproval: false,
				MergeReason:       "Specific name identifies previously generic contact",
				Confidence:        similarity,
			}, nil
		}

		if similarity >= 0.60 {
			// Uncertain - queue for user approval
			log.Printf("[ContactDeduplicator] UNCERTAIN: Merge '%s' → '%s'? (similarity=%.2f)",
				genericContact.Name, extractedContact.Name, similarity)

			conflict := &ContextConflict{
				UserID:         userID,
				ConflictType:   "contact_identification_uncertain",
				Severity:       "low",
				SavedValue:     genericContact.Name,
				ExtractedValue: extractedContact.Name,
				Description: fmt.Sprintf(
					"Is '%s' the actual name for '%s'? (Similarity: %.0f%%)",
					extractedContact.Name, genericContact.Name, similarity*100,
				),
				Status:    "unresolved",
				CreatedAt: time.Now().Unix(),
				ResolutionDetails: map[string]interface{}{
					"generic_contact_id": genericContact.ID,
					"specific_name":      extractedContact.Name,
					"similarity_score":   similarity,
				},
			}

			err := cd.conflictRepo.Save(conflict)
			if err != nil {
				log.Printf("[ContactDeduplicator] Warning: Failed to save conflict: %v", err)
				return &DeduplicationDecision{
					ShouldMerge:    false,
					ShouldSkipSave: false,
				}, nil
			}

			return &DeduplicationDecision{
				ShouldMerge:       false,
				ShouldSkipSave:    true,
				NeedsUserApproval: true,
				ApprovalConflict:  conflict,
				MergeReason:       "Uncertain if specific name identifies generic contact",
				Confidence:        similarity,
			}, nil
		}
	}

	// Multiple generic candidates - too ambiguous
	log.Printf("[ContactDeduplicator] '%s' could identify %d generic contacts, skipping auto-merge",
		extractedContact.Name, len(genericCandidates))

	return &DeduplicationDecision{
		ShouldMerge:    false,
		ShouldSkipSave: false,
	}, nil
}

// mergeContactsAndUpdate merges extracted contact traits into target contact
func (cd *ContactDeduplicator) mergeContactsAndUpdate(
	userID string,
	targetContact *models.Contact,
	extractedContact *models.ExtractedContact,
) error {

	log.Printf("[ContactDeduplicator] Merging extracted traits into contact %d (%s)", targetContact.ID, targetContact.Name)

	// Combine traits (avoid duplicates)
	mergedTraits := make(map[string]bool)
	for _, trait := range targetContact.Characteristics {
		mergedTraits[trait] = true
	}
	for _, trait := range extractedContact.Traits {
		mergedTraits[trait] = true
	}

	// Convert back to slice
	var finalTraits []string
	for trait := range mergedTraits {
		finalTraits = append(finalTraits, trait)
	}

	targetContact.Characteristics = finalTraits
	targetContact.UpdatedAt = time.Now().Unix()

	// Update target contact in database
	err := cd.contactRepo.Update(targetContact)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	log.Printf("[ContactDeduplicator] ✓ Merged contact %d now has %d traits", targetContact.ID, len(finalTraits))
	return nil
}

// mergeGenericIntoSpecific archives generic contact and updates specific with merged traits
func (cd *ContactDeduplicator) mergeGenericIntoSpecific(
	userID string,
	genericContact *models.Contact,
	extractedContact *models.ExtractedContact,
) error {

	log.Printf("[ContactDeduplicator] Merging generic contact %d into new specific contact '%s'",
		genericContact.ID, extractedContact.Name)

	// Update generic contact name to specific name and merge traits
	genericContact.Name = extractedContact.Name
	genericContact.UpdatedAt = time.Now().Unix()

	// Combine traits
	mergedTraits := make(map[string]bool)
	for _, trait := range genericContact.Characteristics {
		mergedTraits[trait] = true
	}
	for _, trait := range extractedContact.Traits {
		mergedTraits[trait] = true
	}

	var finalTraits []string
	for trait := range mergedTraits {
		finalTraits = append(finalTraits, trait)
	}

	genericContact.Characteristics = finalTraits

	// Update in database
	err := cd.contactRepo.Update(genericContact)
	if err != nil {
		return fmt.Errorf("failed to update contact: %w", err)
	}

	log.Printf("[ContactDeduplicator] ✓ Renamed '%s' → '%s' with %d combined traits",
		"Not specified", extractedContact.Name, len(finalTraits))
	return nil
}

// Helper functions

func (cd *ContactDeduplicator) isGenericName(name string) bool {
	genericNames := map[string]bool{
		"not specified": true,
		"contact":       true,
		"he":            true,
		"she":           true,
		"they":          true,
		"him":           true,
		"her":           true,
		"it":            true,
		"unknown":       true,
		"person":        true,
	}
	return genericNames[strings.ToLower(strings.TrimSpace(name))]
}

func (cd *ContactDeduplicator) hasGenericContacts(contacts []*models.Contact) bool {
	for _, c := range contacts {
		if cd.isGenericName(c.Name) {
			return true
		}
	}
	return false
}

func (cd *ContactDeduplicator) filterByRelationship(contacts []*models.Contact, relationship string) []*models.Contact {
	var filtered []*models.Contact
	for _, c := range contacts {
		if c.Relationship == relationship && c.Status == "active" {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (cd *ContactDeduplicator) getGenericContactsWithRelationship(
	contacts []*models.Contact,
	relationship string,
) []*models.Contact {
	var filtered []*models.Contact
	for _, c := range contacts {
		if cd.isGenericName(c.Name) && c.Relationship == relationship && c.Status == "active" {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func (cd *ContactDeduplicator) calculateSimilarity(
	existingContact *models.Contact,
	extractedContact *models.ExtractedContact,
) float64 {
	score := 0.0

	// Factor 1: Relationship match (weight: 0.4)
	if existingContact.Relationship == extractedContact.Relationship {
		score += 0.4
		log.Printf("[ContactDeduplicator] Similarity: relationship match (+0.4)")
	}

	// Factor 2: Trait overlap (weight: 0.3)
	overlap := cd.calculateTraitOverlap(existingContact.Characteristics, extractedContact.Traits)
	score += overlap * 0.3
	log.Printf("[ContactDeduplicator] Similarity: trait overlap %.2f (+%.2f)", overlap, overlap*0.3)

	// Factor 3: Relationship type consistency (weight: 0.2)
	// Same relationship type suggests same person
	if existingContact.Relationship == extractedContact.Relationship {
		score += 0.2
		log.Printf("[ContactDeduplicator] Similarity: context consistency (+0.2)")
	}

	log.Printf("[ContactDeduplicator] Total similarity score: %.2f", score)
	return score
}

func (cd *ContactDeduplicator) calculateTraitOverlap(traits1 []string, traits2 []string) float64 {
	if len(traits1) == 0 && len(traits2) == 0 {
		return 1.0 // Both empty is a match
	}
	if len(traits1) == 0 || len(traits2) == 0 {
		return 0.5 // One empty, one has traits - weak signal
	}

	// Count overlapping traits
	trait1Map := make(map[string]bool)
	for _, t := range traits1 {
		trait1Map[strings.ToLower(t)] = true
	}

	overlap := 0
	for _, t := range traits2 {
		if trait1Map[strings.ToLower(t)] {
			overlap++
		}
	}

	// Overlap ratio: overlapping / total unique
	totalUnique := len(trait1Map) + len(traits2) - overlap
	if totalUnique == 0 {
		return 0.0
	}

	return float64(overlap) / float64(totalUnique)
}

func (cd *ContactDeduplicator) contactNamesToString(contacts []*models.Contact) string {
	var names []string
	for _, c := range contacts {
		names = append(names, fmt.Sprintf("'%s' (%s)", c.Name, c.Relationship))
	}
	return strings.Join(names, ", ")
}
