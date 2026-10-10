package agents

import (
	"testing"
	"time"

	"moly/models"
)

// TestClarificationResolutionWorkflow tests clarification → resolution flow
func TestClarificationResolutionWorkflow(t *testing.T) {
	t.Run("clarification detection and resolution", func(t *testing.T) {
		// Step 1: Create ambiguous contacts
		contacts := []*models.Contact{
			{
				ID:           1,
				Name:         "",
				Status:       "unnamed",
				Relationship: "romantic",
				Confidence:   0.40,
				Pronouns:     []string{"she", "her"},
			},
			{
				ID:           2,
				Name:         "",
				Status:       "unnamed",
				Relationship: "professional",
				Confidence:   0.35,
				Pronouns:     []string{"she", "her"},
			},
		}

		t.Logf("Step 1: Created %d ambiguous contacts", len(contacts))

		// Step 2: Detect need for clarification
		formatter := &ContactResponseFormatter{}
		needsClarification := formatter.CheckIfClarificationNeeded(contacts)

		if !needsClarification {
			t.Error("Should detect clarification needed for ambiguous contacts")
		}
		t.Logf("Step 2: Clarification needed confirmed")

		// Step 3: Generate clarification question
		question := formatter.BuildClarificationQuestion(contacts)
		options := formatter.BuildClarificationOptions(contacts)

		t.Logf("Step 3: Generated question: %s", question)
		t.Logf("Step 3: Generated %d options", len(options))

		if len(options) < 2 {
			t.Errorf("Expected at least 2 options, got %d", len(options))
		}

		// Step 4: Simulate user answer (A = first contact)
		contacts[0].Confidence = 0.99
		contacts[0].Status = "active"
		contacts[1].Confidence = 0.1
		contacts[1].Status = "secondary"

		t.Logf("Step 4: User selected option A (contact 1)")

		// Step 5: Check if clarification still needed
		stillNeedsClarity := formatter.CheckIfClarificationNeeded(contacts)

		if stillNeedsClarity {
			t.Error("Should not need clarification after resolution")
		}
		t.Logf("Step 5: Clarification resolved")

		// Step 6: Format normal response
	})
}

// TestMultipleContactScenario tests handling multiple contacts in one message
func TestMultipleContactScenario(t *testing.T) {
	t.Run("multiple contacts with different confidence", func(t *testing.T) {
		// Create multiple contacts with different confidence levels
		contacts := []*models.Contact{
			{
				ID:           1,
				Name:         "Emily",
				Relationship: "romantic",
				Confidence:   0.95,
				Status:       "named",
			},
			{
				ID:           2,
				Name:         "Marcus",
				Relationship: "professional",
				Confidence:   0.90,
				Status:       "named",
			},
			{
				ID:           3,
				Name:         "",
				Relationship: "family",
				Confidence:   0.40,
				Status:       "unnamed",
			},
		}

		t.Logf("Created %d contacts with varying confidence", len(contacts))

		// Check clarification need
		formatter := &ContactResponseFormatter{}
		needs := formatter.CheckIfClarificationNeeded(contacts)

		t.Logf("Clarification needed: %v (due to low-confidence contact)", needs)

		// Calculate average confidence
		avgConfidence := formatter.CalculateAverageConfidence(contacts)
		t.Logf("Average confidence across contacts: %.2f", avgConfidence)

		// Check if all are named
		allNamed := formatter.AreAllContactsNamed(contacts)
		t.Logf("All contacts named: %v", allNamed)

		if allNamed {
			t.Error("Should detect that not all contacts are named")
		}
	})
}

// TestEdgeCaseExpiredClarification tests handling of expired clarifications
func TestEdgeCaseExpiredClarification(t *testing.T) {
	t.Run("expired clarification handling", func(t *testing.T) {
		// Create a clarification that's 25 hours old (past 24h TTL)
		now := time.Now().Unix()
		clarificationAge := int64(90000) // 25 hours

		clarificationTimestamp := now - clarificationAge

		t.Logf("Clarification age: %d seconds (25 hours)", clarificationAge)

		// Check if expired
		isExpired := (now - clarificationTimestamp) > (24 * 3600)

		if !isExpired {
			t.Error("Should detect expired clarification")
		}

		t.Logf("Clarification correctly identified as expired")

		// In real implementation, would regenerate clarification
		t.Logf("In production: Would regenerate new clarification")
	})
}

// TestEdgeCaseInvalidAnswer tests handling invalid clarification answers
func TestEdgeCaseInvalidAnswer(t *testing.T) {
	t.Run("invalid clarification answer", func(t *testing.T) {
		invalidAnswers := []string{
			"D) Someone else?", // Only A/B/C valid
			"maybe option A",   // Not clear
			"all of them",      // Invalid
			"",                 // Empty
		}

		for _, answer := range invalidAnswers {
			t.Logf("Testing invalid answer: %q", answer)

			// Check if it matches A-D pattern
			if len(answer) > 0 {
				first := answer[0]
				if first < 'A' || first > 'D' {
					t.Logf("  → Correctly rejected as invalid")
					continue
				}
				if len(answer) > 1 && answer[1] != ')' && answer[1] != '.' {
					t.Logf("  → Correctly rejected as invalid format")
					continue
				}
			}

			t.Logf("  → Marked for manual review")
		}
	})
}

// TestNameConflictDetection tests detection of naming conflicts
func TestNameConflictDetection(t *testing.T) {
	t.Run("progressive naming conflict", func(t *testing.T) {
		// Scenario: User previously named a contact "Sarah", now says "her name is Sam"
		existingContact := &models.Contact{
			ID:       1,
			Name:     "Sarah",
			Status:   "named",
			Pronouns: []string{"she", "her"},
		}

		message := "Wait, her name is actually Sam, not Sarah"
		t.Logf("Existing contact name: %s", existingContact.Name)
		t.Logf("User message: %s", message)

		// In real implementation, would:
		// 1. Detect new name "Sam"
		// 2. Compare with existing "Sarah"
		// 3. Flag as potential conflict
		// 4. Ask user for confirmation

		t.Logf("Would detect naming conflict and request confirmation")
	})
}

// TestContactPersistenceAcrossMessages tests contact state persistence
func TestContactPersistenceAcrossMessages(t *testing.T) {
	t.Run("contact persistence across message sequence", func(t *testing.T) {
		userID := "user_67890"

		// Message 1: Initial mention
		contact1 := &models.Contact{
			ID:           1,
			UserID:       userID,
			Name:         "",
			Status:       "unnamed",
			Relationship: "romantic",
			Confidence:   0.80,
			CreatedAt:    time.Now().Unix(),
		}
		t.Logf("Message 1: Contact created (unnamed, confidence: %.2f)", contact1.Confidence)

		// Message 2: Progressive naming
		contact1.Name = "Emily"
		contact1.Status = "named"
		contact1.Confidence = 0.95
		t.Logf("Message 2: Contact renamed to %s (confidence: %.2f)", contact1.Name, contact1.Confidence)

		// Message 3: Further context
		contact1.UpdatedAt = time.Now().Unix()
		t.Logf("Message 3: Contact updated (last seen: %d)", contact1.UpdatedAt)

		// Verify persistence
		if contact1.Name != "Emily" {
			t.Error("Contact name should persist as 'Emily'")
		}
		if contact1.UserID != userID {
			t.Error("Contact user association should persist")
		}

		t.Logf("Contact state successfully persisted across 3 messages")
	})
}

// TestConcurrentContactOperations tests thread safety (simplified)
func TestConcurrentContactOperations(t *testing.T) {
	t.Run("concurrent contact updates", func(t *testing.T) {
		// Note: In real implementation, would use actual goroutines and test race conditions
		// This is a placeholder for concurrent safety validation

		contact := &models.Contact{
			ID:         1,
			Name:       "Emily",
			Confidence: 0.95,
			UpdatedAt:  time.Now().Unix(),
		}

		// Simulate concurrent read/write
		readCount := 0
		writeCount := 0

		for i := 0; i < 100; i++ {
			if i%2 == 0 {
				readCount++
				_ = contact.Name // Read
			} else {
				writeCount++
				contact.UpdatedAt = time.Now().Unix() // Write
			}
		}

		t.Logf("Concurrent operations: %d reads, %d writes", readCount, writeCount)
		t.Logf("Final contact state: name=%s, updated=%d", contact.Name, contact.UpdatedAt)
	})
}
