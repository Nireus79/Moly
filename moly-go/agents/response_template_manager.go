package agents

import (
	"database/sql"
	"log"
	"time"

	"moly/database"
)

// ResponseTemplateManager handles dynamic response templates from database
type ResponseTemplateManager struct {
	db *database.Database
}

// NewResponseTemplateManager creates a new template manager
func NewResponseTemplateManager(db *database.Database) *ResponseTemplateManager {
	return &ResponseTemplateManager{db: db}
}

// ResponseTemplate represents a template from the database
type ResponseTemplate struct {
	ID       int
	Context  string
	Category string
	Template string
	Priority int
}

// GetTemplate retrieves a template based on context and category
func (rtm *ResponseTemplateManager) GetTemplate(context, category string) (string, error) {
	if rtm.db == nil {
		log.Printf("[ResponseTemplateManager] WARNING: No database available, using default response")
		return "", nil
	}

	conn := rtm.db.GetConnection()
	if conn == nil {
		return "", nil
	}

	query := `
		SELECT template_text FROM response_templates
		WHERE context = ? AND category = ? AND enabled = 1
		ORDER BY priority DESC
		LIMIT 1
	`

	var template string
	err := conn.QueryRow(query, context, category).Scan(&template)
	if err == sql.ErrNoRows {
		log.Printf("[ResponseTemplateManager] No template found for context=%s category=%s", context, category)
		return "", nil
	} else if err != nil {
		log.Printf("[ResponseTemplateManager] Error querying template: %v", err)
		return "", err
	}

	log.Printf("[ResponseTemplateManager] Loaded template: context=%s category=%s", context, category)
	return template, nil
}

// InitializeDefaultTemplates adds default templates to the database
func (rtm *ResponseTemplateManager) InitializeDefaultTemplates() error {
	if rtm.db == nil {
		return nil
	}

	conn := rtm.db.GetConnection()
	if conn == nil {
		return nil
	}

	now := time.Now().Unix()

	defaultTemplates := []struct {
		context  string
		category string
		template string
		priority int
	}{
		// New user greetings
		{
			context:  "new_user_greeting",
			category: "greeting",
			template: "Hey there! 👋 I'm Μώλυ (Moly), your thinking partner. What's on your mind today? Whether it's about a relationship, work, or just life in general, I'm here to help you think it through.",
			priority: 100,
		},
		{
			context:  "new_user_greeting",
			category: "greeting",
			template: "Welcome! I'm Μώλυ. I'm here to be your thinking partner—someone to help you explore what's really going on. What brought you here today?",
			priority: 90,
		},
		// First message - no topic yet
		{
			context:  "no_topic",
			category: "clarification",
			template: "I'd love to help. What's on your mind? Is it about a relationship, work, or something else?",
			priority: 100,
		},
		{
			context:  "no_topic",
			category: "clarification",
			template: "Tell me more! What's the main thing you want to explore or figure out?",
			priority: 90,
		},
		// Encouragement
		{
			context:  "incomplete_context",
			category: "encouragement",
			template: "Help me understand better—what's the core issue here?",
			priority: 100,
		},
		{
			context:  "incomplete_context",
			category: "encouragement",
			template: "I'm here to listen. What do you need help with?",
			priority: 90,
		},
	}

	for _, t := range defaultTemplates {
		// Check if template already exists
		var exists int
		conn.QueryRow(
			"SELECT COUNT(*) FROM response_templates WHERE context = ? AND category = ? AND template_text = ?",
			t.context, t.category, t.template,
		).Scan(&exists)

		if exists > 0 {
			continue
		}

		_, err := conn.Exec(`
			INSERT INTO response_templates
			(context, category, template_text, priority, enabled, version, created_at, updated_at)
			VALUES (?, ?, ?, ?, 1, 1, ?, ?)
		`, t.context, t.category, t.template, t.priority, now, now)

		if err != nil {
			log.Printf("[ResponseTemplateManager] Error inserting template: %v", err)
			continue
		}

		log.Printf("[ResponseTemplateManager] ✓ Added template: context=%s category=%s", t.context, t.category)
	}

	return nil
}
