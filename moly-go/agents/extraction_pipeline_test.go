package agents

import (
	"testing"

	"moly/database"
	"moly/models"
	"moly/tools"
)

// TestExtractionStoreLifecycle verifies ExtractionStore with 5-min TTL (Phase 7)
func TestExtractionStoreLifecycle(t *testing.T) {
	store := tools.NewExtractionStore()
	defer store.Stop()

	// Create a test artifact
	artifact := &models.ExtractionArtifact{
		ID:                "artifact_1",
		MessageID:         "msg_123",
		UserID:            "user_1",
		ConversationID:    "conv_1",
		Source:            "llm",
		LLMSuccess:        true,
		SubjectAttributed: true,
		NegationPreserved: true,
		AverageConfidence: 0.87,
		Entities: []models.ExtractedEntity{
			{
				Type:       "contact",
				Value:      "Christine",
				Subject:    "her",
				Confidence: 0.95,
			},
			{
				Type:       "preference",
				Value:      "likes threesomes",
				Subject:    "her",
				Confidence: 0.85,
			},
		},
	}

	// Save artifact
	store.Save(artifact)

	// Retrieve and verify
	retrieved := store.Get("msg_123")
	if retrieved == nil {
		t.Fatal("Artifact not found in store after Save")
	}

	if retrieved.ID != artifact.ID {
		t.Errorf("Retrieved artifact ID mismatch: %s vs %s", retrieved.ID, artifact.ID)
	}

	if len(retrieved.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(retrieved.Entities))
	}

	// Verify size tracking
	if store.Size() != 1 {
		t.Errorf("Store size should be 1, got %d", store.Size())
	}

	// Delete artifact
	store.Delete("msg_123")
	if store.Get("msg_123") != nil {
		t.Error("Artifact still in store after Delete")
	}

	t.Log("✓ ExtractionStore lifecycle test PASSED")
}

// TestExtractionArtifactQuality verifies extraction quality metrics (Phase 7)
func TestExtractionArtifactQuality(t *testing.T) {
	artifact := &models.ExtractionArtifact{
		ID:                "artifact_2",
		MessageID:         "msg_456",
		SubjectAttributed: true,
		NegationPreserved: true,
		LLMSuccess:        true,
		Entities: []models.ExtractedEntity{
			{Type: "contact", Value: "her", Subject: "she", Confidence: 0.95, IsAmbiguous: false},
			{Type: "contact", Value: "the girl", Subject: "she", Confidence: 0.88, IsAmbiguous: false},
			{Type: "preference", Value: "dominant", Subject: "user", Confidence: 0.92, IsAmbiguous: false},
			{Type: "characteristic", Value: "adventurous", Subject: "she", Confidence: 0.82, IsAmbiguous: false},
		},
	}

	// Calculate average confidence
	totalConfidence := 0.0
	for _, e := range artifact.Entities {
		totalConfidence += e.Confidence
	}
	artifact.AverageConfidence = totalConfidence / float64(len(artifact.Entities))

	if artifact.AverageConfidence < 0.85 || artifact.AverageConfidence > 0.92 {
		t.Errorf("Average confidence %.2f is out of expected range", artifact.AverageConfidence)
	}

	// Verify high-confidence contacts method
	highConf := artifact.HighConfidenceContacts()
	if len(highConf) < 2 {
		t.Errorf("Expected at least 2 high-confidence contacts, got %d", len(highConf))
	}

	// Verify ambiguous entities method
	ambiguous := artifact.AmbiguousEntities()
	if len(ambiguous) != 0 {
		t.Errorf("Expected 0 ambiguous entities, got %d", len(ambiguous))
	}

	// Test subject grouping
	subjectCounts := make(map[string]int)
	for _, entity := range artifact.Entities {
		if entity.Type == "contact" {
			subjectCounts[entity.Subject]++
		}
	}

	if subjectCounts["she"] != 2 {
		t.Errorf("Expected 2 'she' contacts, got %d", subjectCounts["she"])
	}

	// Note: "dominant" with subject "user" is a preference, not a contact
	// So we expect only 'she' contacts
	totalContacts := 0
	for _, count := range subjectCounts {
		totalContacts += count
	}
	if totalContacts != 2 {
		t.Errorf("Expected 2 total contacts, got %d", totalContacts)
	}

	t.Log("✓ ExtractionArtifact quality test PASSED")
	t.Logf("  - Average confidence: %.2f", artifact.AverageConfidence)
	t.Logf("  - High-confidence contacts: %d", len(highConf))
	t.Logf("  - Subject grouping: %v", subjectCounts)
}

// TestSubjectBasedDeduplication verifies subject grouping logic (Phase 7)
func TestSubjectBasedDeduplication(t *testing.T) {
	// Create a contact deduplicator
	dedup := database.NewContactDeduplicator(nil) // No DB needed for subject grouping

	// Create an artifact with multi-person entities
	artifact := &models.ExtractionArtifact{
		Entities: []models.ExtractedEntity{
			{Type: "contact", Value: "Christine", Subject: "her", Confidence: 0.95},
			{Type: "contact", Value: "the girl", Subject: "her", Confidence: 0.88},
			{Type: "contact", Value: "Sophie", Subject: "she", Confidence: 0.92},
			{Type: "contact", Value: "Kyle", Subject: "user", Confidence: 0.90},
			{Type: "preference", Value: "likes threesomes", Subject: "her", Confidence: 0.85},
		},
	}

	// Perform subject-based deduplication
	entitiesBySubject, contactCount, err := dedup.DeduplicateBySubject("test_user", artifact)

	if err != nil {
		t.Errorf("DeduplicateBySubject returned error: %v", err)
	}

	if contactCount != 4 {
		t.Errorf("Expected 4 contact entities, got %d", contactCount)
	}

	// Verify grouping
	if len(entitiesBySubject["her"]) != 2 {
		t.Errorf("Expected 2 'her' entities, got %d", len(entitiesBySubject["her"]))
	}

	if len(entitiesBySubject["she"]) != 1 {
		t.Errorf("Expected 1 'she' entity, got %d", len(entitiesBySubject["she"]))
	}

	if len(entitiesBySubject["user"]) != 1 {
		t.Errorf("Expected 1 'user' entity, got %d", len(entitiesBySubject["user"]))
	}

	// Merge contacts by subject
	merged := dedup.MergeContactsBySubject(entitiesBySubject)

	if len(merged) != 3 {
		t.Errorf("Expected 3 merged contacts (one per subject group), got %d", len(merged))
	}

	// Verify first merged contact (for "her")
	herContact := merged[0]
	if herContact.Name != "Christine" && herContact.Name != "the girl" {
		t.Errorf("Merged 'her' contact has unexpected name: %s", herContact.Name)
	}

	if herContact.ExtractionCount != 2 {
		t.Errorf("Merged 'her' contact should have extraction_count=2, got %d", herContact.ExtractionCount)
	}

	t.Log("✓ Subject-based deduplication test PASSED")
	t.Logf("  - Grouped %d contacts into %d subjects", contactCount, len(entitiesBySubject))
	t.Logf("  - Merged into %d contacts", len(merged))
}
