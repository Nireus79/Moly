package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"moly/models"
)

// UserContextSnapshot - Complete user context loaded at a point in time
type UserContextSnapshot struct {
	UserID           string
	ConversationID   string
	AboutMe          *models.AboutMe
	Contacts         []models.Contact
	RecentMessages   []models.ChatMessage
	PendingInputs    []PendingInput
	RecentInsights   []models.Reflection
	LoadedAt         time.Time
}

// LoadUserContext - Load complete user context in minimal DB queries
// Returns all data needed for message processing in one efficient load
func (db *Database) LoadUserContext(userID, conversationID string, recentMessageLimit int, recentInsightLimit int) (*UserContextSnapshot, error) {
	log.Printf("[ContextLoader] Loading context for user=%s conversation=%s", userID, conversationID)

	ctx := &UserContextSnapshot{
		UserID:         userID,
		ConversationID: conversationID,
		LoadedAt:       time.Now(),
	}

	// Query 1: AboutMe
	if aboutMe, err := db.GetAboutMeRepository().Get(userID); err != nil {
		log.Printf("[ContextLoader] Warning: Failed to load AboutMe: %v", err)
	} else {
		ctx.AboutMe = aboutMe
	}

	// Query 2: Recent messages for context
	if messages, err := db.getRecentMessages(conversationID, recentMessageLimit); err != nil {
		log.Printf("[ContextLoader] Warning: Failed to load recent messages: %v", err)
	} else {
		ctx.RecentMessages = messages
	}

	// Query 3: Pending inputs (all unresolved)
	if pending, err := db.GetPendingInputRepository().GetUnresolved(userID); err != nil {
		log.Printf("[ContextLoader] Warning: Failed to load pending inputs: %v", err)
	} else {
		ctx.PendingInputs = pending
	}

	// Query 4: Recent insights/reflections
	if insights, err := db.getRecentInsights(conversationID, recentInsightLimit); err != nil {
		log.Printf("[ContextLoader] Warning: Failed to load recent insights: %v", err)
	} else {
		ctx.RecentInsights = insights
	}

	// Query 5: Contacts (mentioned in recent messages or about_me)
	if contacts, err := db.getContacts(userID); err != nil {
		log.Printf("[ContextLoader] Warning: Failed to load contacts: %v", err)
	} else {
		ctx.Contacts = contacts
	}

	log.Printf("[ContextLoader] Loaded context: %d messages, %d pending, %d insights, %d contacts",
		len(ctx.RecentMessages), len(ctx.PendingInputs), len(ctx.RecentInsights), len(ctx.Contacts))

	return ctx, nil
}

// getRecentMessages - Get recent messages from conversation
func (db *Database) getRecentMessages(conversationID string, limit int) ([]models.ChatMessage, error) {
	query := `
		SELECT id, user_id, conversation_id, role, content, context_extracted, contact_mention, metadata, created_at
		FROM chat_messages
		WHERE conversation_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := db.Query(query, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var msg models.ChatMessage
		var contextExtracted, contactMention, metadata sql.NullString

		err := rows.Scan(&msg.ID, &msg.UserID, &msg.ConversationID, &msg.Role, &msg.Content, &contextExtracted, &contactMention, &metadata, &msg.CreatedAt)
		if err != nil {
			log.Printf("[ContextLoader] Error scanning message: %v", err)
			continue
		}

		// Parse JSON fields if present
		if contextExtracted.Valid {
			var extracted map[string]interface{}
			if err := json.Unmarshal([]byte(contextExtracted.String), &extracted); err != nil {
				log.Printf("[ContextLoader] ERROR: Failed to parse contextExtracted for message %s: %v", msg.ID, err)
			} else {
				msg.ContextExtracted = extracted
			}
		}

		if contactMention.Valid {
			var contact models.ContactMentionDetected
			if err := json.Unmarshal([]byte(contactMention.String), &contact); err != nil {
				log.Printf("[ContextLoader] ERROR: Failed to parse contactMention for message %s: %v", msg.ID, err)
			} else {
				msg.ContactMention = &contact
			}
		}

		messages = append(messages, msg)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	return messages, nil
}

// getRecentInsights - Get recent reflections/insights
func (db *Database) getRecentInsights(conversationID string, limit int) ([]models.Reflection, error) {
	query := `
		SELECT id, conversation_id, contact_id, characteristics, interests, intentions, status, created_at, approved_at
		FROM reflections
		WHERE conversation_id = ? AND status IN ('pending_approval', 'approved')
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := db.Query(query, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query insights: %w", err)
	}
	defer rows.Close()

	var insights []models.Reflection
	for rows.Next() {
		var insight models.Reflection
		var characteristics, interests, intentions sql.NullString
		var contactID sql.NullString
		var approvedAt sql.NullInt64

		err := rows.Scan(&insight.ID, &insight.ConversationID, &contactID, &characteristics, &interests, &intentions, &insight.Status, &insight.CreatedAt, &approvedAt)
		if err != nil {
			log.Printf("[ContextLoader] Error scanning insight: %v", err)
			continue
		}

		if contactID.Valid {
			insight.ContactID = contactID.String
		}

		// Parse JSON arrays
		if characteristics.Valid {
			if err := json.Unmarshal([]byte(characteristics.String), &insight.Characteristics); err != nil {
				log.Printf("[ContextLoader] Warning: Failed to parse characteristics: %v", err)
			}
		}

		if interests.Valid {
			if err := json.Unmarshal([]byte(interests.String), &insight.Interests); err != nil {
				log.Printf("[ContextLoader] Warning: Failed to parse interests: %v", err)
			}
		}

		if intentions.Valid {
			if err := json.Unmarshal([]byte(intentions.String), &insight.Intentions); err != nil {
				log.Printf("[ContextLoader] Warning: Failed to parse intentions: %v", err)
			}
		}

		if approvedAt.Valid {
			insight.ApprovedAt = approvedAt.Int64
		}

		insights = append(insights, insight)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating insights: %w", err)
	}

	return insights, nil
}

// getContacts - Get contacts for user
func (db *Database) getContacts(userID string) ([]models.Contact, error) {
	query := `
		SELECT id, user_id, name, relationship, characteristics, created_at
		FROM user_contacts
		WHERE user_id = ? AND status = 'active'
		ORDER BY updated_at DESC
	`

	rows, err := db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query contacts: %w", err)
	}
	defer rows.Close()

	var contacts []models.Contact
	for rows.Next() {
		var contact models.Contact
		var characteristics sql.NullString

		err := rows.Scan(&contact.ID, &contact.UserID, &contact.Name, &contact.Relationship, &characteristics, &contact.CreatedAt)
		if err != nil {
			log.Printf("[ContextLoader] Error scanning contact: %v", err)
			continue
		}

		// Parse characteristics JSON
		if characteristics.Valid {
			if err := json.Unmarshal([]byte(characteristics.String), &contact.Characteristics); err != nil {
				log.Printf("[ContextLoader] Warning: Failed to parse characteristics: %v", err)
			}
		}

		contacts = append(contacts, contact)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating contacts: %w", err)
	}

	return contacts, nil
}
