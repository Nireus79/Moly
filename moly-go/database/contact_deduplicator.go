package database

import (
	"fmt"
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

// CheckForDuplicate analyzes extracted contact and detects if it's a duplicate of existing contact
func (cd *ContactDeduplicator) CheckForDuplicate(
	userID string,
	extractedContact *models.ExtractedContact,
) (*DeduplicationDecision, error) {
	return cd.CheckForDuplicateWithConversation(userID, "", extractedContact)
}

// CheckForDuplicateWithConversation analyzes extracted contact using conversation history for pronoun resolution
func (cd *ContactDeduplicator) CheckForDuplicateWithConversation(
	userID string,
	conversationID string,
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
		return cd.handleGenericNameWithConversation(userID, conversationID, extractedContact, existingContacts)
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

// handleGenericName handles case where extracted name is generic ("Not specified", "He", "She") - no conversation context
func (cd *ContactDeduplicator) handleGenericName(
	userID string,
	extractedContact *models.ExtractedContact,
	existingContacts []*models.Contact,
) (*DeduplicationDecision, error) {
	return cd.handleGenericNameWithConversation(userID, "", extractedContact, existingContacts)
}

// handleGenericNameWithConversation handles generic name resolution using conversation history
// When pronoun like "her" is found, looks at prior messages to find who it refers to
func (cd *ContactDeduplicator) handleGenericNameWithConversation(
	userID string,
	conversationID string,
	extractedContact *models.ExtractedContact,
	existingContacts []*models.Contact,
) (*DeduplicationDecision, error) {

	// Strategy 1: Check conversation history for recently mentioned contacts
	if conversationID != "" {
		recentContactDecision := cd.resolveFromConversationHistory(
			userID, conversationID, extractedContact, existingContacts,
		)
		if recentContactDecision != nil {
			return recentContactDecision, nil
		}
	}

	// Strategy 2: Fallback to relationship-based matching
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
			"When you said '%s', who did you mean? %s",
			extractedContact.Name,
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
		MergeReason:       "Ambiguous generic name, user must clarify",
		Confidence:        0.60,
	}, nil
}

// resolveFromConversationHistory looks at recent messages to find what a pronoun refers to
// Example: "her" in Message 2 should resolve to "Christine" from Message 1
func (cd *ContactDeduplicator) resolveFromConversationHistory(
	userID string,
	conversationID string,
	pronoun *models.ExtractedContact,
	existingContacts []*models.Contact,
) *DeduplicationDecision {

	log.Printf("[ContactDeduplicator] Resolving pronoun '%s' from conversation history", pronoun.Name)

	// Get recent messages to find prior mentions
	chatRepo := NewChatMessageRepository(cd.db.GetConnection())
	recentMessages, err := chatRepo.GetConversationHistory(userID, conversationID, 10)
	if err != nil || len(recentMessages) == 0 {
		log.Printf("[ContactDeduplicator] Could not load conversation history: %v", err)
		return nil
	}

	// Extract mentioned contacts from recent messages
	mentionedContacts := cd.extractMentionedContactsFromMessages(recentMessages, existingContacts)
	log.Printf("[ContactDeduplicator] Found %d recently mentioned contacts in conversation", len(mentionedContacts))

	if len(mentionedContacts) == 0 {
		return nil
	}

	if len(mentionedContacts) == 1 {
		// Only one contact mentioned recently - high confidence this is what pronoun refers to
		targetContact := mentionedContacts[0]
		log.Printf("[ContactDeduplicator] AUTO-MERGE (conversation history): Pronoun '%s' → %s (confidence=0.90)",
			pronoun.Name, targetContact.Name)

		err := cd.mergeContactsAndUpdate(userID, targetContact, pronoun)
		if err != nil {
			log.Printf("[ContactDeduplicator] Error during merge: %v", err)
			return nil
		}

		return &DeduplicationDecision{
			ShouldMerge:       true,
			ShouldSkipSave:    true,
			TargetContact:     targetContact,
			NeedsUserApproval: false,
			MergeReason:       fmt.Sprintf("Pronoun '%s' refers to recently mentioned '%s' (conversation context)", pronoun.Name, targetContact.Name),
			Confidence:        0.90,
		}
	}

	// Multiple recent contacts - ambiguous, ask Moly to clarify
	log.Printf("[ContactDeduplicator] CLARIFICATION NEEDED: Pronoun '%s' could refer to %d people", pronoun.Name, len(mentionedContacts))

	conflict := &ContextConflict{
		UserID:         userID,
		ConflictType:   "contact_ambiguous_pronoun",
		Severity:       "low",
		SavedValue:     fmt.Sprintf("%v", cd.contactNamesToString(mentionedContacts)),
		ExtractedValue: pronoun.Name,
		Description: fmt.Sprintf(
			"When you said '%s', did you mean: %s? I want to make sure I understand correctly.",
			pronoun.Name,
			cd.contactNamesToString(mentionedContacts),
		),
		Status:    "unresolved",
		CreatedAt: time.Now().Unix(),
	}

	err = cd.conflictRepo.Save(conflict)
	if err != nil {
		log.Printf("[ContactDeduplicator] Warning: Failed to save clarification conflict: %v", err)
		return nil
	}

	return &DeduplicationDecision{
		ShouldMerge:       false,
		ShouldSkipSave:    true, // Skip save, wait for clarification
		NeedsUserApproval: true,
		ApprovalConflict:  conflict,
		MergeReason:       "Pronoun is ambiguous - Moly will ask user to clarify",
		Confidence:        0.50,
	}
}

// extractMentionedContactsFromMessages finds contacts mentioned in recent messages
func (cd *ContactDeduplicator) extractMentionedContactsFromMessages(
	messages []*models.ChatMessage,
	existingContacts []*models.Contact,
) []*models.Contact {

	// Build map of existing contacts
	contactMap := make(map[string]*models.Contact)
	for _, c := range existingContacts {
		contactMap[strings.ToLower(c.Name)] = c
	}

	// Track mentioned contacts in order
	mentioned := make([]*models.Contact, 0)
	seenNames := make(map[string]bool)

	// Go through messages in reverse order (most recent first)
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]

		// Check contact mention field
		if msg.ContactMention != nil && msg.ContactMention.PersonName != "" {
			name := strings.ToLower(msg.ContactMention.PersonName)
			if !cd.isGenericName(msg.ContactMention.PersonName) && !seenNames[name] {
				if contact, exists := contactMap[name]; exists {
					mentioned = append(mentioned, contact)
					seenNames[name] = true
				}
			}
		}
	}

	return mentioned
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

func (cd *ContactDeduplicator) contactNamesToString(contacts []*models.Contact) string {
	var names []string
	for _, c := range contacts {
		names = append(names, fmt.Sprintf("'%s' (%s)", c.Name, c.Relationship))
	}
	return strings.Join(names, ", ")
}
