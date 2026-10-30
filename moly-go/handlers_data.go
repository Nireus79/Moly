package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"moly/agents"
	"moly/database"
	"moly/models"
	"moly/privacy"
	"moly/schema"
	"moly/tools"
)

// ClarificationResponseHandler - Handle user responses to clarification questions
func (srv *APIServer) ClarificationResponseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": authErr.Error()})
		return
	}

	// From here on this request works with the model-calling components of this user, built from this user's own provider.
	srv = srv.forUser(userID)

	var req agents.ClarificationResponseRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.FactID == "" || req.QuestionID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing factId or questionId"})
		return
	}

	log.Printf("[Clarification] Response from user %s to question %s\n", userID, req.QuestionID)

	// Mark question as answered in database to prevent duplicate questions
	conn := srv.database.GetConnection()
	now := time.Now().Unix()
	if _, err := conn.Exec(
		"UPDATE clarification_questions SET status = 'answered', answered_at = ? WHERE id = ? AND user_id = ?",
		now, req.QuestionID, userID,
	); err != nil {
		log.Printf("[Clarification] Warning: Failed to mark question as answered: %v", err)
	}

	// Create TemporaryFactStore for this user
	tempStore := agents.NewTemporaryFactStore(srv.database, userID)

	// Extract linked facts (from temporary store)
	linkedFacts := []agents.ExtractedFact{}
	if req.FactID != "" {
		if tempFact, err := tempStore.Get(req.FactID); err == nil && tempFact != nil {
			// Convert TemporaryFact to ExtractedFact
			linkedFacts = append(linkedFacts, agents.ExtractedFact{
				ID:              tempFact.FactID,
				Value:           tempFact.FactValue,
				FactType:        tempFact.FactType,
				ProposedSubject: tempFact.AttributedTo,
				Confidence:      tempFact.Confidence,
				Evidence:        tempFact.Evidence,
			})
		}
	}

	// Get the conversation ID from the clarification question
	questionRepo := database.NewClarificationQuestionRepository(srv.database)
	question, err := questionRepo.GetQuestion(userID, req.QuestionID)
	conversationID := ""
	if err != nil {
		log.Printf("[Clarification] Warning: Failed to retrieve question: %v", err)
	} else if question != nil {
		conversationID = question.ConversationID
	}

	// Process response with new AnswerProcessor
	processedResult, err := srv.answerProcessor.ProcessResponse(
		req.QuestionID,
		req.UserResponse,
		linkedFacts,
		userID,
		conversationID,
	)
	if err != nil {
		log.Printf("[Clarification] Error processing response: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"error": fmt.Sprintf("Failed to process response: %v", err),
		})
		return
	}

	// Save extracted context to About Me profile
	if processedResult.ContextToSave != nil {
		conn := srv.database.GetConnection()

		// Extract all available context data
		contextStr := ""
		if contextVal, ok := processedResult.ContextToSave["context"].(string); ok {
			contextStr = contextVal
		}

		patternsJSON := "[]"
		if patterns, ok := processedResult.ContextToSave["patterns"].([]interface{}); ok {
			if b, err := json.Marshal(patterns); err == nil {
				patternsJSON = string(b)
			}
		}

		// Update or create About Me profile - save communication_style AND patterns
		// ARCHITECTURAL FIX: Only save communication_style and patterns from clarification
		// Don't auto-populate tone_preference (that's user's configuration, not observed data)
		now := time.Now().Unix()
		_, err := conn.Exec(`
			INSERT INTO about_me (user_id, communication_style, patterns, updated_at, created_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(user_id) DO UPDATE SET
			  communication_style = CASE
				WHEN communication_style IS NULL OR communication_style = '' THEN excluded.communication_style
				ELSE communication_style
			  END,
			  patterns = CASE
				WHEN patterns IS NULL OR patterns = '[]' THEN excluded.patterns
				ELSE patterns
			  END,
			  updated_at = excluded.updated_at
		`, userID, contextStr, patternsJSON, now, now)

		if err != nil {
			log.Printf("[Clarification] Warning: Failed to save About Me: %v", err)
		} else {
			log.Printf("[Clarification] ✓ Saved About Me: style='%s', patterns=%d items", contextStr, len(patternsJSON))
		}
	}

	// FIXED: Don't return early - continue through orchestrator
	// The clarification response is a message and should be evaluated through full pipeline
	log.Printf("[Clarification] ✓ Answer processed, now routing through orchestrator for consistency")

	// Create a Phase5Request from the clarification response to process through orchestrator
	// This ensures the response content gets safety/ethics evaluation
	orchestratorReq := &schema.Phase5Request{
		Message:          req.UserResponse,
		ConversationID:   conversationID,
		BrowserSessionId: "", // Will use conversation's existing session
	}

	log.Printf("[Clarification] ✓ Routing clarification response through orchestrator: %s", conversationID)

	// Build a fake request for the orchestrator
	bodyBytes, _ := json.Marshal(orchestratorReq)
	fakeReq := &http.Request{
		Method:     "POST",
		Body:       io.NopCloser(bytes.NewReader(bodyBytes)),
		Header:     r.Header,
		RemoteAddr: r.RemoteAddr,
	}

	// Add auth token to header if present
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		fakeReq.Header.Set("Authorization", authHeader)
	}

	log.Printf("[Clarification] FIXED: Passing through orchestrator instead of early return")
	// Continue through orchestrator by calling the handler
	srv.MessageProcessorHandler(w, fakeReq)
	// After orchestrator processes, the response is already sent via w
}

// AboutMeHandler - Get or save user's About Me profile
func (srv *APIServer) AboutMeHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[AboutMe] Unauthorized access attempt: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()

	if r.Method == http.MethodGet {
		log.Printf("[AboutMe] GET request from user %s\n", userID)

		// Fetch About Me profile
		var communicationStyle, coreValues, tonePreference, preferences, goals, patterns string
		err := conn.QueryRow(
			`SELECT COALESCE(communication_style,''), COALESCE(core_values,''), COALESCE(tone_preference,''), COALESCE(preferences,''), COALESCE(goals,''), COALESCE(patterns,'') FROM about_me WHERE user_id = ?`,
			userID,
		).Scan(&communicationStyle, &coreValues, &tonePreference, &preferences, &goals, &patterns)

		if err != nil && err != sql.ErrNoRows {
			log.Printf("[AboutMe] Error retrieving profile: %v\n", err)
		}

		profile := &schema.AboutMeProfile{
			CommunicationStyle: communicationStyle,
			CoreValues:         parseArrayFromStorage(coreValues),
			TonePreference:     tonePreference,
			Preferences:        schema.FlexText(preferences),
			Goals:              parseArrayFromStorage(goals),
			Patterns:           parseArrayFromStorage(patterns),
		}

		log.Printf("[AboutMe] Profile retrieved: style=%s, values=%d, tone=%s\n",
			communicationStyle, len(profile.CoreValues), tonePreference)
		schema.RespondSuccess(w, http.StatusOK, "profile", profile)

	} else if r.Method == http.MethodPost {
		log.Printf("[AboutMe] POST request from user %s\n", userID)

		// Save About Me profile
		req := &schema.AboutMeProfile{}

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			log.Printf("[AboutMe] Invalid request body: %v\n", err)
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		log.Printf("[AboutMe] Parsed profile: style=%s, values=%d, tone=%s, prefs=%s\n",
			req.CommunicationStyle, len(req.CoreValues), req.TonePreference, req.Preferences)

		// Validate using schema validator
		if err := schema.ValidateStruct(req); err != nil {
			log.Printf("[AboutMe] Validation error: %v\n", err)
			schema.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		now := time.Now().Unix()
		_, err := conn.Exec(
			`INSERT OR REPLACE INTO about_me (user_id, communication_style, core_values, tone_preference, preferences, goals, patterns, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			userID,
			req.CommunicationStyle,
			encodeArrayForStorage(req.CoreValues),
			req.TonePreference,
			string(req.Preferences),
			encodeArrayForStorage(req.Goals),
			encodeArrayForStorage(req.Patterns),
			now,
			now,
		)

		if err != nil {
			log.Printf("[AboutMe] Error saving profile: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to save profile")
			return
		}

		log.Printf("[AboutMe] Profile saved successfully for user %s\n", userID)

		// Record audit event for profile update
		auditRepo := srv.database.GetAuditLogRepository()
		if auditRepo != nil {
			auditErr := auditRepo.RecordAction(userID, "profile_updated", map[string]interface{}{
				"communicationStyle": req.CommunicationStyle,
				"coreValues":         req.CoreValues,
				"tonePreference":     req.TonePreference,
				"preferences":        string(req.Preferences),
				"goals":              req.Goals,
			})
			if auditErr != nil {
				log.Printf("[AboutMe] Warning: Failed to record audit event: %v", auditErr)
			}
		}

		schema.RespondSuccess(w, http.StatusOK, "profile", req)
	}
}

// ContextHandler handles GET /api/context - Returns user's context quality for a conversation
func (srv *APIServer) ContextHandler(w http.ResponseWriter, r *http.Request) {
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[Context] Unauthorized: %v", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conversationID := r.URL.Query().Get("conversationId")
	if conversationID == "" {
		schema.RespondError(w, http.StatusBadRequest, "conversationId parameter required")
		return
	}

	conn := srv.database.GetConnection()

	// Count messages in conversation (indicator of context completeness)
	var messageCount int
	err := conn.QueryRow(
		`SELECT COUNT(*) FROM interactions WHERE user_id = ? AND conversation_id = ?`,
		userID, conversationID,
	).Scan(&messageCount)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("[Context] Error counting messages: %v", err)
	}

	// Check AboutMe completeness
	var aboutMeFields int
	err = conn.QueryRow(
		`SELECT COUNT(CASE WHEN communication_style IS NOT NULL AND communication_style != '' THEN 1 END) +
		        COUNT(CASE WHEN core_values IS NOT NULL AND core_values != '' THEN 1 END) +
		        COUNT(CASE WHEN tone_preference IS NOT NULL AND tone_preference != '' THEN 1 END) +
		        COUNT(CASE WHEN goals IS NOT NULL AND goals != '' THEN 1 END)
		 FROM about_me WHERE user_id = ?`,
		userID,
	).Scan(&aboutMeFields)

	// Count contacts
	var contactCount int
	err = conn.QueryRow(
		`SELECT COUNT(*) FROM contacts WHERE user_id = ? AND status = 'active' AND name_status = 'named'`,
		userID,
	).Scan(&contactCount)

	// Calculate context quality
	completenessScore := 0.0
	gaps := []string{}

	if aboutMeFields < 2 {
		gaps = append(gaps, "Missing AboutMe information")
		completenessScore += 0.3
	} else {
		completenessScore += 0.5
	}

	if contactCount == 0 {
		gaps = append(gaps, "No contacts defined")
		completenessScore += 0.2
	} else if contactCount >= 3 {
		completenessScore += 0.3
	} else {
		completenessScore += 0.2
	}

	if messageCount < 5 {
		gaps = append(gaps, "Limited conversation history")
		completenessScore += 0.2
	} else {
		completenessScore += 0.2
	}

	completenessLevel := "minimal"
	if completenessScore >= 0.7 {
		completenessLevel = "complete"
	} else if completenessScore >= 0.4 {
		completenessLevel = "partial"
	}

	response := map[string]interface{}{
		"conversationId": conversationID,
		"contextQuality": map[string]interface{}{
			"overallScore":      completenessScore,
			"completenessLevel": completenessLevel,
		},
		"missingContextGaps": gaps,
	}

	log.Printf("[Context] Quality for %s: score=%.2f level=%s gaps=%d",
		conversationID, completenessScore, completenessLevel, len(gaps))
	schema.RespondSuccess(w, http.StatusOK, "context", response)
}

// ConversationsHandler - Get or create conversations
func (srv *APIServer) ConversationsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()

	if r.Method == http.MethodGet {
		// List user's conversations
		rows, err := conn.Query(
			"SELECT id, name, type, description, purpose, members, settings, notes, created_at, updated_at FROM conversations WHERE user_id = ? ORDER BY updated_at DESC",
			userID,
		)
		if err != nil {
			schema.RespondError(w, http.StatusInternalServerError, "Failed to fetch conversations")
			return
		}
		defer rows.Close()

		conversations := []map[string]interface{}{}
		for rows.Next() {
			var id, name, convType, description string
			var purpose, notes, membersJSON, settingsJSON sql.NullString
			var createdAt, updatedAt int64
			if err := rows.Scan(&id, &name, &convType, &description, &purpose, &membersJSON, &settingsJSON, &notes, &createdAt, &updatedAt); err != nil {
				log.Printf("[ConversationsHandler] ERROR: Failed to scan conversation row: %v", err)
				continue
			}

			conv := map[string]interface{}{
				"id":          id,
				"name":        name,
				"type":        convType,
				"description": description,
				"purpose":     purpose.String, // Empty string if NULL
				"createdAt":   createdAt,
				"updatedAt":   updatedAt,
				"notes":       notes.String, // Empty string if NULL
			}

			// Parse JSON fields
			if membersJSON.Valid && membersJSON.String != "" {
				var members []schema.ConversationMember
				if err := json.Unmarshal([]byte(membersJSON.String), &members); err == nil {
					conv["members"] = members
				}
			}
			if settingsJSON.Valid && settingsJSON.String != "" {
				var settings schema.ConversationSettings
				if err := json.Unmarshal([]byte(settingsJSON.String), &settings); err == nil {
					conv["settings"] = settings
				}
			}

			// Get recent messages preview (last 3 messages)
			msgRows, err := conn.Query(
				"SELECT role, content, created_at FROM chat_messages WHERE conversation_id = ? ORDER BY created_at DESC LIMIT 3",
				id,
			)
			if err == nil {
				defer msgRows.Close()
				messages := []map[string]interface{}{}
				for msgRows.Next() {
					var role, content string
					var msgTime int64
					if err := msgRows.Scan(&role, &content, &msgTime); err == nil {
						content = database.StoredReplyText(content)
						// Truncate content for preview
						preview := content
						if len(preview) > 100 {
							preview = preview[:100] + "..."
						}
						messages = append(messages, map[string]interface{}{
							"role":      role,
							"content":   preview,
							"timestamp": msgTime,
						})
					}
				}
				// Reverse to get chronological order
				for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
					messages[i], messages[j] = messages[j], messages[i]
				}
				conv["recentMessages"] = messages
			}

			conversations = append(conversations, conv)
		}

		if err := rows.Err(); err != nil {
			log.Printf("[ConversationsHandler] Error iterating conversations: %v", err)
			schema.RespondError(w, http.StatusInternalServerError, "Error loading conversations")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"conversations": conversations,
		})

	} else if r.Method == http.MethodPost {
		// Create new conversation
		req := &schema.Conversation{}

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		// Validate using schema validator
		if err := schema.ValidateStruct(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		now := time.Now()
		conversationID := fmt.Sprintf("conv_%d_%d", now.Unix(), now.Nanosecond())
		nowUnix := now.Unix()

		// Encode JSON fields
		membersJSON := ""
		if len(req.Members) > 0 {
			if b, err := json.Marshal(req.Members); err == nil {
				membersJSON = string(b)
			}
		}

		settingsJSON := ""
		if req.Settings.Mode != "" || req.Settings.Context != "" {
			if b, err := json.Marshal(req.Settings); err == nil {
				settingsJSON = string(b)
			}
		}

		_, err := conn.Exec(
			"INSERT INTO conversations (id, user_id, name, type, description, purpose, members, settings, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			conversationID,
			userID,
			req.Name,
			req.Type,
			req.Description,
			req.Purpose,
			membersJSON,
			settingsJSON,
			req.Notes,
			nowUnix,
			nowUnix,
		)

		if err != nil {
			log.Printf("[Conversations] Error creating conversation: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to create conversation")
			return
		}

		response := &schema.Conversation{
			ID:          conversationID,
			Name:        req.Name,
			Type:        req.Type,
			Purpose:     req.Purpose,
			Description: req.Description,
			Members:     req.Members,
			Settings:    req.Settings,
			Notes:       req.Notes,
			CreatedAt:   nowUnix,
			UpdatedAt:   nowUnix,
		}

		schema.RespondSuccess(w, http.StatusOK, "conversation", response)
	} else if r.Method == http.MethodDelete {
		// Delete a conversation by ID from path (using Go 1.22+ path parameters)
		conversationID := r.PathValue("conversationID")

		if conversationID == "" {
			schema.RespondError(w, http.StatusBadRequest, "Conversation ID required")
			return
		}

		// Verify the conversation belongs to this user
		var checkUserID string
		err := conn.QueryRow(
			"SELECT user_id FROM conversations WHERE id = ?",
			conversationID,
		).Scan(&checkUserID)

		if err == sql.ErrNoRows {
			schema.RespondError(w, http.StatusNotFound, "Conversation not found")
			return
		}
		if err != nil {
			schema.RespondError(w, http.StatusInternalServerError, "Failed to verify conversation")
			return
		}

		if checkUserID != userID {
			schema.RespondError(w, http.StatusForbidden, "Not authorized to delete this conversation")
			return
		}

		// Delete the conversation with its messages and related rows, in one transaction
		err = database.DeleteConversationData(conn, userID, conversationID)
		if err != nil {
			log.Printf("[Conversations] Error deleting conversation: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to delete conversation")
			return
		}

		log.Printf("[Conversations] Deleted conversation %s for user %s\n", conversationID, userID)
		schema.RespondSuccess(w, http.StatusOK, "result", map[string]interface{}{
			"id":     conversationID,
			"status": "deleted",
		})
	} else {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// ContactsHandler - Get or create user contacts
func (srv *APIServer) ContactsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()

	if r.Method == http.MethodGet {
		// List user's contacts
		rows, err := conn.Query(
			"SELECT id, name, relationship, notes, created_at FROM contacts WHERE user_id = ? AND name_status = 'named' ORDER BY updated_at DESC",
			userID,
		)
		if err != nil {
			schema.RespondError(w, http.StatusInternalServerError, "Failed to fetch contacts")
			return
		}
		defer rows.Close()

		contacts := []map[string]interface{}{}
		for rows.Next() {
			var id, name, relationship string
			var notes sql.NullString // Handle NULL values properly
			var createdAt int64
			if err := rows.Scan(&id, &name, &relationship, &notes, &createdAt); err != nil {
				log.Printf("[ContactsHandler] ERROR: Failed to scan contact row: %v", err)
				continue
			}
			contacts = append(contacts, map[string]interface{}{
				"id":           id,
				"name":         name,
				"relationship": relationship,
				"notes":        notes.String, // Empty string if NULL
				"createdAt":    createdAt,
			})
		}

		if err := rows.Err(); err != nil {
			log.Printf("[ContactsHandler] Error iterating contacts: %v", err)
			schema.RespondError(w, http.StatusInternalServerError, "Error loading contacts")
			return
		}

		respondJSON(w, http.StatusOK, map[string]interface{}{
			"contacts": contacts,
		})

	} else if r.Method == http.MethodPost {
		// Create new contact
		req := &schema.Contact{}

		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		// Validate using schema validator
		if err := schema.ValidateStruct(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		now := time.Now()
		contactID := fmt.Sprintf("contact_%d_%d", now.Unix(), now.Nanosecond())
		nowUnix := now.Unix()

		_, err := conn.Exec(
			"INSERT INTO contacts (user_id, name, relationship, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			userID,
			req.Name,
			req.Relationship,
			req.Notes,
			nowUnix,
			nowUnix,
		)

		if err != nil {
			log.Printf("[Contacts] Error creating contact: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to create contact")
			return
		}

		response := &schema.Contact{
			ID:           contactID,
			Name:         req.Name,
			Relationship: req.Relationship,
			Notes:        req.Notes,
			CreatedAt:    nowUnix,
		}

		schema.RespondSuccess(w, http.StatusOK, "contact", response)
	}
}

// ContactDetailHandler - Get, update, or delete individual contacts
func (srv *APIServer) ContactDetailHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()
	contactID := r.PathValue("contactID")

	if contactID == "" {
		schema.RespondError(w, http.StatusBadRequest, "contactID is required")
		return
	}

	// Verify contact belongs to user
	var contactUserID string
	err := conn.QueryRow("SELECT user_id FROM contacts WHERE id = ?", contactID).Scan(&contactUserID)
	if err != nil {
		schema.RespondError(w, http.StatusNotFound, "Contact not found")
		return
	}
	if contactUserID != userID {
		schema.RespondError(w, http.StatusForbidden, "Access denied to this contact")
		return
	}

	if r.Method == http.MethodGet {
		// Get single contact
		var id int64
		var name, relationship string
		var characteristics, notes sql.NullString
		var createdAt, updatedAt int64
		err := conn.QueryRow(
			"SELECT id, name, relationship, characteristics, notes, created_at, updated_at FROM contacts WHERE id = ? AND user_id = ?",
			contactID, userID,
		).Scan(&id, &name, &relationship, &characteristics, &notes, &createdAt, &updatedAt)

		if err != nil {
			schema.RespondError(w, http.StatusNotFound, "Contact not found")
			return
		}

		var traits []string
		if characteristics.Valid && characteristics.String != "" {
			json.Unmarshal([]byte(characteristics.String), &traits)
		}

		contact := map[string]interface{}{
			"id":              id,
			"name":            name,
			"relationship":    relationship,
			"characteristics": traits,
			"notes":           notes.String,
			"createdAt":       createdAt,
			"updatedAt":       updatedAt,
		}

		schema.RespondSuccess(w, http.StatusOK, "contact", contact)

	} else if r.Method == http.MethodPut {
		// Update contact
		req := &schema.Contact{}
		if err := json.NewDecoder(r.Body).Decode(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		if err := schema.ValidateStruct(req); err != nil {
			schema.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		now := time.Now().Unix()

		_, err := conn.Exec(
			"UPDATE contacts SET name = ?, relationship = ?, notes = ?, updated_at = ? WHERE id = ? AND user_id = ?",
			req.Name, req.Relationship, req.Notes, now, contactID, userID,
		)

		if err != nil {
			schema.RespondError(w, http.StatusInternalServerError, "Failed to update contact")
			return
		}

		response := map[string]interface{}{
			"id":           contactID,
			"name":         req.Name,
			"relationship": req.Relationship,
			"notes":        req.Notes,
			"updatedAt":    now,
		}

		schema.RespondSuccess(w, http.StatusOK, "contact", response)

	} else if r.Method == http.MethodDelete {
		// Delete contact
		_, err := conn.Exec("DELETE FROM contacts WHERE id = ? AND user_id = ?", contactID, userID)
		if err != nil {
			schema.RespondError(w, http.StatusInternalServerError, "Failed to delete contact")
			return
		}

		schema.RespondSuccess(w, http.StatusOK, "message", "Contact deleted successfully")
	} else {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// MessagesHandler - Get messages for a conversation
func (srv *APIServer) MessagesHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	conn := srv.database.GetConnection()

	if r.Method != http.MethodGet {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get conversationId from query parameter
	conversationID := r.URL.Query().Get("conversationId")
	if conversationID == "" {
		schema.RespondError(w, http.StatusBadRequest, "conversationId parameter required")
		return
	}

	// Verify conversation exists and belongs to user
	var convUserID string
	err := conn.QueryRow(
		"SELECT user_id FROM conversations WHERE id = ?",
		conversationID,
	).Scan(&convUserID)

	if err != nil {
		schema.RespondError(w, http.StatusNotFound, "Conversation not found")
		return
	}

	if convUserID != userID {
		schema.RespondError(w, http.StatusForbidden, "Access denied to this conversation")
		return
	}

	// Fetch messages from chat_messages table
	rows, err := conn.Query(
		"SELECT id, user_id, conversation_id, role, content, context_extracted, contact_mention, created_at FROM chat_messages WHERE conversation_id = ? ORDER BY created_at ASC",
		conversationID,
	)
	if err != nil {
		log.Printf("[Messages] Error fetching messages: %v\n", err)
		schema.RespondError(w, http.StatusInternalServerError, "Failed to fetch messages")
		return
	}
	defer rows.Close()

	messages := []map[string]interface{}{}
	for rows.Next() {
		var id, role, content string
		var contextExtracted, contactMention sql.NullString
		var uID, convID string
		var createdAt int64

		if err := rows.Scan(&id, &uID, &convID, &role, &content, &contextExtracted, &contactMention, &createdAt); err != nil {
			log.Printf("[MessagesHandler] WARNING: Skipping corrupted message row - scan error: %v", err)
			continue
		}

		msgMap := map[string]interface{}{
			"id":             id,
			"conversationId": convID,
			"role":           role,
			"content":        database.StoredReplyText(content),
			"createdAt":      createdAt,
		}

		if contextExtracted.Valid {
			var extracted map[string]interface{}
			if err := json.Unmarshal([]byte(contextExtracted.String), &extracted); err == nil {
				msgMap["contextExtracted"] = extracted
			} else {
				msgMap["contextExtracted"] = contextExtracted.String
			}
		}
		if contactMention.Valid {
			var contact map[string]interface{}
			if err := json.Unmarshal([]byte(contactMention.String), &contact); err == nil {
				msgMap["contactMention"] = contact
			} else {
				msgMap["contactMention"] = contactMention.String
			}
		}

		messages = append(messages, msgMap)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[MessagesHandler] Error iterating messages: %v", err)
		schema.RespondError(w, http.StatusInternalServerError, "Error loading messages")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"messages": messages,
	})
}

// ReflectionsHandler - Get, approve, or reject reflections
func (srv *APIServer) ReflectionsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[Reflections] Unauthorized access attempt: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	if r.Method == http.MethodGet {
		log.Printf("[Reflections] GET request from user %s\n", userID)

		// Get pending reflections for user
		repo := srv.database.GetReflectionRepository()
		if repo == nil {
			log.Printf("[Reflections] ERROR: ReflectionRepository is nil\n")
			schema.RespondError(w, http.StatusInternalServerError, "Reflection service unavailable")
			return
		}
		reflections, err := repo.GetPendingApprovals(userID, "pending_approval")
		if err != nil {
			log.Printf("[Reflections] Error retrieving reflections: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to retrieve reflections")
			return
		}

		// Always return empty array, never nil (frontend expects array)
		if reflections == nil {
			reflections = []models.Reflection{}
		}

		log.Printf("[Reflections] Found %d pending reflections for user %s\n", len(reflections), userID)
		schema.RespondSuccess(w, http.StatusOK, "reflections", reflections)

	} else if r.Method == http.MethodPost {
		log.Printf("[Reflections] POST request from user %s\n", userID)

		var req struct {
			ID     int    `json:"id"`
			Action string `json:"action"` // "approve" or "reject"
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("[Reflections] Invalid request body: %v\n", err)
			schema.RespondError(w, http.StatusBadRequest, "Invalid request")
			return
		}

		if req.ID == 0 || (req.Action != "approve" && req.Action != "reject") {
			log.Printf("[Reflections] Invalid ID or action\n")
			schema.RespondError(w, http.StatusBadRequest, "id and action (approve/reject) required")
			return
		}

		repo := srv.database.GetReflectionRepository()
		if repo == nil {
			log.Printf("[Reflections] ERROR: ReflectionRepository is nil\n")
			schema.RespondError(w, http.StatusInternalServerError, "Reflection service unavailable")
			return
		}
		var err error
		if req.Action == "approve" {
			err = repo.Approve(userID, req.ID)
			log.Printf("[Reflections] Approved reflection %d\n", req.ID)
		} else {
			err = repo.Reject(userID, req.ID)
			log.Printf("[Reflections] Rejected reflection %d\n", req.ID)
		}

		if err != nil {
			log.Printf("[Reflections] Error updating reflection: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to update reflection")
			return
		}

		schema.RespondSuccess(w, http.StatusOK, "result", map[string]string{"status": "ok"})
	} else {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// MetricsHandler - Get learning analytics and question effectiveness metrics
// ConflictsHandler handles both GET (list conflicts) - uses new pending_input system
func (srv *APIServer) ConflictsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[Conflicts] Unauthorized access attempt: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	if r.Method == http.MethodGet {
		log.Printf("[Conflicts] GET request from user %s\n", userID)

		// Get unresolved conflicts from new pending_input system
		pendingRepo := srv.database.GetPendingInputRepository()
		if pendingRepo == nil {
			log.Printf("[Conflicts] ERROR: PendingInputRepository is nil\n")
			schema.RespondError(w, http.StatusInternalServerError, "Conflict service unavailable")
			return
		}

		pending, err := pendingRepo.GetByType(userID, "conflict")
		if err != nil {
			log.Printf("[Conflicts] Error retrieving conflicts: %v\n", err)
			schema.RespondError(w, http.StatusInternalServerError, "Failed to retrieve conflicts")
			return
		}

		// Convert pending_input conflicts to displayable format
		type ConflictResponse struct {
			ID             int64  `json:"id"`
			Type           string `json:"type"`
			Subtype        string `json:"subtype"`
			Question       string `json:"question"`
			StoredValue    string `json:"storedValue"`
			ExtractedValue string `json:"extractedValue"`
			CreatedAt      int64  `json:"createdAt"`
		}

		var conflicts []ConflictResponse
		for _, p := range pending {
			// Parse context JSON to get stored/extracted values
			var ctxData map[string]interface{}
			json.Unmarshal(p.Context, &ctxData)

			conflicts = append(conflicts, ConflictResponse{
				ID:             p.ID,
				Type:           p.Type,
				Subtype:        p.Subtype,
				Question:       p.Question,
				StoredValue:    fmt.Sprintf("%v", ctxData["old_value"]),
				ExtractedValue: fmt.Sprintf("%v", ctxData["new_value"]),
				CreatedAt:      p.CreatedAt,
			})
		}

		// Always return empty array, never nil
		if conflicts == nil {
			conflicts = []ConflictResponse{}
		}

		log.Printf("[Conflicts] Found %d unresolved conflicts for user %s\n", len(conflicts), userID)
		schema.RespondSuccess(w, http.StatusOK, "conflicts", conflicts)

	} else {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// ConflictResolveHandler handles POST /api/conflicts/resolve
func (srv *APIServer) ConflictResolveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[ConflictResolve] Unauthorized access attempt: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	log.Printf("[ConflictResolve] POST request from user %s\n", userID)

	// Parse request
	type ResolveRequest struct {
		ConflictId int64  `json:"conflictId"`
		Resolution string `json:"resolution"` // keep_saved | use_extracted | merge
	}

	req := &ResolveRequest{}
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		log.Printf("[ConflictResolve] Invalid request body: %v\n", err)
		schema.RespondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate resolution choice
	if req.Resolution != "keep_saved" && req.Resolution != "use_extracted" && req.Resolution != "merge" {
		log.Printf("[ConflictResolve] Invalid resolution: %s\n", req.Resolution)
		schema.RespondError(w, http.StatusBadRequest, "Invalid resolution choice")
		return
	}

	log.Printf("[ConflictResolve] Resolving conflict %d with resolution: %s\n", req.ConflictId, req.Resolution)

	// Load the conflict
	conflictRepo := srv.database.GetContextConflictRepository()
	if conflictRepo == nil {
		log.Printf("[ConflictResolve] ERROR: ContextConflictRepository is nil\n")
		schema.RespondError(w, http.StatusInternalServerError, "Conflict service unavailable")
		return
	}
	// The conflict is named by its id, so it is looked up by id (and by the user: another user's conflict is never found).
	targetConflict, err := conflictRepo.GetUnresolvedByID(userID, req.ConflictId)
	if err != nil {
		log.Printf("[ConflictResolve] Error loading conflicts: %v\n", err)
		schema.RespondError(w, http.StatusInternalServerError, "Failed to load conflicts")
		return
	}

	if targetConflict == nil {
		log.Printf("[ConflictResolve] Conflict %d not found or already resolved\n", req.ConflictId)
		schema.RespondError(w, http.StatusNotFound, "Conflict not found or already resolved")
		return
	}

	// Verify conflict belongs to this user
	if targetConflict.UserID != userID {
		log.Printf("[ConflictResolve] Conflict %d does not belong to user %s\n", req.ConflictId, userID)
		schema.RespondError(w, http.StatusForbidden, "Access denied to this conflict")
		return
	}

	// Apply the resolution
	handler := tools.NewConflictResolutionHandler(srv.database)
	result := handler.ApplyResolution(targetConflict, req.Resolution)

	if !result.Success {
		log.Printf("[ConflictResolve] Failed to apply resolution: %s\n", result.Message)
		schema.RespondError(w, http.StatusInternalServerError, result.Message)
		return
	}

	log.Printf("[ConflictResolve] ✓ Conflict resolved: %s\n", result.Message)

	// Record audit event for conflict resolution
	auditRepo := srv.database.GetAuditLogRepository()
	if auditRepo != nil {
		auditErr := auditRepo.RecordAction(userID, "conflict_resolved", map[string]interface{}{
			"conflictId":     req.ConflictId,
			"conflictType":   targetConflict.ConflictType,
			"resolution":     req.Resolution,
			"savedValue":     targetConflict.SavedValue,
			"extractedValue": targetConflict.ExtractedValue,
		})
		if auditErr != nil {
			log.Printf("[ConflictResolve] Warning: Failed to record audit event: %v", auditErr)
		}
	}

	// Phase 3: Trigger behavioral profile rebuild (learning from this resolution)
	log.Printf("[ConflictResolve] Triggering behavioral profile rebuild for user %s", userID)
	learningAgent := srv.GetLearningAgent(userID) // Use cached learning agent
	if learningAgent != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[ConflictResolve] PANIC in BuildBehavioralProfile: %v", r)
				}
			}()
			_, analyzeErr := learningAgent.BuildBehavioralProfile(userID)
			if analyzeErr != nil {
				log.Printf("[ConflictResolve] Warning: Could not rebuild behavioral profile: %v", analyzeErr)
			} else {
				log.Printf("[ConflictResolve] ✓ Behavioral profile rebuilt after conflict resolution")
			}
		}()
	}

	schema.RespondSuccess(w, http.StatusOK, "resolution", result)
}

func (srv *APIServer) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[Metrics] Unauthorized access attempt: %v\n", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	if r.Method != http.MethodGet {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	log.Printf("[Metrics] GET request from user %s\n", userID)

	metricsRepo := srv.database.GetMetricsRepository()
	if metricsRepo == nil {
		// Graceful degradation if metrics not available
		schema.RespondSuccess(w, http.StatusOK, "metrics", map[string]interface{}{
			"question_effectiveness": nil,
			"principle_violations":   nil,
			"approach_comparison":    nil,
		})
		return
	}

	// Get all metrics
	questionStats, err1 := metricsRepo.GetQuestionEffectivenessStats(userID)
	violationStats, err2 := metricsRepo.GetPrincipleViolationStats(userID)
	principleBreakdown, err3 := metricsRepo.GetPrincipleBreakdown(userID)
	approachComparison, err4 := metricsRepo.GetApproachComparison(userID)

	if err1 != nil {
		log.Printf("[Metrics] Error getting question stats: %v\n", err1)
		questionStats = map[string]interface{}{}
	}
	if err2 != nil {
		log.Printf("[Metrics] Error getting violation stats: %v\n", err2)
		violationStats = map[string]interface{}{}
	}
	if err3 != nil {
		log.Printf("[Metrics] Error getting principle breakdown: %v\n", err3)
		principleBreakdown = []map[string]interface{}{}
	}
	if err4 != nil {
		log.Printf("[Metrics] Error getting approach comparison: %v\n", err4)
		approachComparison = []map[string]interface{}{}
	}

	metrics := map[string]interface{}{
		"question_effectiveness": questionStats,
		"principle_violations":   violationStats,
		"principle_breakdown":    principleBreakdown,
		"approach_comparison":    approachComparison,
		"timestamp":              time.Now().Unix(),
	}

	log.Printf("[Metrics] Returning metrics for user %s\n", userID)
	schema.RespondSuccess(w, http.StatusOK, "metrics", metrics)
}

// DeleteProfileHandler handles user profile deletion with password confirmation
func (srv *APIServer) DeleteProfileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodDelete {
		schema.RespondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract and validate Bearer token
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		log.Printf("[DeleteProfile] Authentication failed: %v", authErr)
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	// Parse request body
	var req struct {
		Password string `json:"password"`
		Confirm  string `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[DeleteProfile] Failed to parse request body: %v", err)
		schema.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	log.Printf("[DeleteProfile] User %s requested profile deletion\n", userID)

	// An account is confirmed with its password; the local profile has none and is confirmed by typing DELETE.
	conn := srv.database.GetConnection()
	var passwordHash sql.NullString
	err := conn.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash)
	if err != nil {
		log.Printf("[DeleteProfile] User not found: %v", err)
		schema.RespondError(w, http.StatusNotFound, "User not found")
		return
	}
	if !passwordHash.Valid {
		if req.Confirm != "DELETE" {
			schema.RespondError(w, http.StatusBadRequest, "Type DELETE to confirm")
			return
		}
	} else {
		if req.Password == "" {
			schema.RespondError(w, http.StatusBadRequest, "Password required")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash.String), []byte(req.Password)); err != nil {
			log.Printf("[DeleteProfile] Invalid password for user %s\n", userID)
			schema.RespondError(w, http.StatusUnauthorized, "Invalid password")
			return
		}
	}

	log.Printf("[DeleteProfile] Deleting all data for user %s\n", userID)
	if err := database.DeleteUserData(conn, userID); err != nil {
		log.Printf("[DeleteProfile] ERROR: %v\n", err)
		schema.RespondError(w, http.StatusInternalServerError, "Profile could not be deleted")
		return
	}
	log.Printf("[DeleteProfile] ✓ All data deleted for user %s\n", userID)
	srv.forgetUser(userID)

	// Return success response
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Profile and all associated data have been permanently deleted",
	})
}

// ExportProfileHandler - GET /api/user/export. Everything stored about the signed-in user, as one JSON file (the password hash
// and the session tokens are left out). This is the way to keep or move one's data.
func (srv *APIServer) ExportProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}
	data, err := database.ExportUserData(srv.database.GetConnection(), userID)
	if err != nil {
		log.Printf("[Export] failed for user %s: %v", userID, err)
		schema.RespondError(w, http.StatusInternalServerError, "The export could not be made")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="moly-export-`+time.Now().Format("2006-01-02")+`.json"`)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(data)
}

// ImportProfileHandler - POST /api/user/import. Restores a file made by the export into the (empty) profile of the signed-in
// user. Nothing is changed if the file does not fit or the profile already has data.
func (srv *APIServer) ImportProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 200<<20)
	var data map[string]any
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		schema.RespondError(w, http.StatusBadRequest, "The file could not be read as a Moly export")
		return
	}
	res, err := database.ImportUserData(srv.database.GetConnection(), userID, data)
	var bad database.ErrBadExport
	switch {
	case errors.Is(err, database.ErrProfileNotEmpty):
		schema.RespondError(w, http.StatusConflict, err.Error())
		return
	case errors.As(err, &bad):
		schema.RespondError(w, http.StatusBadRequest, err.Error())
		return
	case err != nil:
		log.Printf("[Import] failed for user %s: %v", userID, err)
		schema.RespondError(w, http.StatusInternalServerError, "The import could not be completed; nothing was changed")
		return
	}
	srv.forgetUser(userID)
	respondJSON(w, http.StatusOK, map[string]any{"success": true, "rows": res.Rows, "tables": res.Tables})
}

// ConversationMemoryHandler returns what Moly remembers of one of the user's conversations: the summary, the locked goal, the
// people, and the questions still open. It is the user's own data (it lets the user see and check the memory) and it lets a
// live test see what the model wrote without logging any message text.
func (srv *APIServer) ConversationMemoryHandler(w http.ResponseWriter, r *http.Request) {
	userID, authErr := extractAndValidateToken(r, srv.database)
	if authErr != nil {
		schema.RespondError(w, http.StatusUnauthorized, authErr.Error())
		return
	}
	conversationID := r.PathValue("conversationID")
	conn := srv.database.GetConnection()
	var owned int
	if err := conn.QueryRow(`SELECT COUNT(*) FROM conversations WHERE id = ? AND user_id = ?`, conversationID, userID).Scan(&owned); err != nil || owned == 0 {
		schema.RespondError(w, http.StatusNotFound, "Conversation not found")
		return
	}
	out := map[string]interface{}{"conversationId": conversationID, "summary": "", "summaryCoversMessages": 0}
	if summary, err := database.NewConversationSummaryRepository(srv.database.GetConnection()).GetSummary(userID, conversationID); err == nil && summary != nil {
		out["summary"] = summary.Arc
		out["summaryCoversMessages"] = summary.MessageCount
	}
	if prev := srv.loadPreviousExtraction(userID, conversationID); prev != nil {
		out["goal"] = prev.PrimaryGoal
		if out["goal"] == "" {
			out["goal"] = prev.Goal
		}
	}
	people := []string{}
	// the people this conversation has brought up (see database.EstablishedPeople), not everyone the user has
	memoryPeople := database.NewContactRepository(srv.database).InConversation(conversationID)
	var texts []string
	if rows, err := conn.Query(`SELECT content FROM chat_messages WHERE user_id = ? AND conversation_id = ?`, userID, conversationID); err == nil {
		for rows.Next() {
			var content string
			if rows.Scan(&content) == nil {
				texts = append(texts, content)
			}
		}
		rows.Close()
	}
	brought := map[string]bool{}
	for _, c := range database.EstablishedPeople(memoryPeople, userID, texts) {
		brought[c.Name] = true
	}
	if contacts, err := memoryPeople.GetByUserID(userID); err == nil {
		for _, c := range contacts {
			if c.Status == "active" && brought[c.Name] {
				people = append(people, database.PersonLine(c))
			}
		}
	}
	out["people"] = people
	// memory mode: the structured memory of the conversation, beside the old people list (docs/MEMORY_REDESIGN.md)
	if snap, snapErr := database.NewMemoryStore(conn).Snapshot(userID, conversationID); snapErr == nil && (len(snap.People) > 0 || snap.Goal != "" || len(snap.Questions) > 0 || len(snap.Facts) > 0) {
		out["structured"] = snap
	}
	open := []string{}
	if questions, err := database.NewClarificationHistoryRepository(srv.database).OpenQuestions(userID, conversationID); err == nil {
		for _, q := range questions {
			open = append(open, q.Text)
		}
	}
	out["openQuestions"] = open
	respondJSON(w, http.StatusOK, out)
}

// forgetUser drops what the running server still holds in memory about a deleted user: the saved conversation states, the skip offers,
// the learning agent, the model-calling components built for the user, their unlocked API key, and (as the log scrubber cannot tell
// whose strings are whose) the log scrubber's list, which every turn refills.
func (srv *APIServer) forgetUser(userID string) {
	prefix := userID + "|"
	for _, m := range []*sync.Map{srv.previousExtractionCache, srv.skipOffers} {
		m.Range(func(k, _ interface{}) bool {
			if key, ok := k.(string); ok && strings.HasPrefix(key, prefix) {
				m.Delete(k)
			}
			return true
		})
	}
	srv.learningAgentCache.Delete(userID)
	srv.userParts.Delete(userID)
	if f, ok := srv.source.(interface{ Forget(string) }); ok {
		f.Forget(userID)
	}
	privacy.Forget()
}
