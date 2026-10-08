package agents

import (
	"log"
	"moly/models"
)

// WhatWhoLinker connects extracted intentions (WHAT) to contacts (WHO)
// Part of "Who is Who" system that provides unified, centralized context
type WhatWhoLinker struct {
}

// NewWhatWhoLinker creates a new WHAT-WHO linker
func NewWhatWhoLinker() *WhatWhoLinker {
	return &WhatWhoLinker{}
}

// LinkWhatToWho updates contact with WHAT context
// This enriches the "Who" record with what the user wants/needs involving that contact
func (l *WhatWhoLinker) LinkWhatToWho(contact *models.Contact, extracted *models.ExtractedContext) {
	if contact == nil || extracted == nil {
		return
	}

	log.Printf("[WhatWhoLinker] Linking WHAT context to contact: %s", contact.Name)

	// Level 1: Topic Association - Is this intention ABOUT this contact?
	if extracted.Intention != "" {
		// Add intention to contact's involved intentions
		if !l.contains(contact.InvolvedInIntentions, extracted.Intention) {
			contact.InvolvedInIntentions = append(contact.InvolvedInIntentions, extracted.Intention)
			log.Printf("[WhatWhoLinker] Added intention to %s: %s", contact.Name, extracted.Intention)
		}
	}

	// Level 2: Goals involving this contact
	for _, goal := range extracted.Goals {
		if !l.contains(contact.InvolvedInIntentions, goal) {
			contact.InvolvedInIntentions = append(contact.InvolvedInIntentions, goal)
		}
	}

	// Infer contact role based on relationship
	if contact.ContactRole == "" && contact.Relationship != "" {
		contact.ContactRole = l.inferContactRole(contact.Relationship, extracted.IntentionPrinciples)
		log.Printf("[WhatWhoLinker] Inferred role for %s (%s): %s",
			contact.Name, contact.Relationship, contact.ContactRole)
	}

	// Update dependencies based on relationship dynamics
	contact.Dependencies = l.inferenceDependencies(contact.Relationship, extracted.IntentionPrinciples)
}

// inferContactRole determines what role this contact plays
func (l *WhatWhoLinker) inferContactRole(relationship string, principles []string) string {
	roleMap := map[string]string{
		"romantic":     "romantic partner",
		"professional": "colleague/mentor",
		"family":       "family member",
		"friend":       "trusted friend",
		"other":        "acquaintance",
	}

	if role, exists := roleMap[relationship]; exists {
		// Could be enhanced to "trusted advisor" if expertise-related principles present
		return role
	}

	return ""
}

// inferenceDependencies determines what must be true for this contact to help
func (l *WhatWhoLinker) inferenceDependencies(relationship string, principles []string) []string {
	deps := []string{}

	// Universal dependencies
	deps = append(deps, "needs to understand context")

	// Relationship-specific dependencies
	switch relationship {
	case "professional":
		deps = append(deps, "has relevant expertise")
		deps = append(deps, "available during work hours")
	case "romantic":
		deps = append(deps, "relationship is healthy")
		deps = append(deps, "shared values alignment")
	case "family":
		deps = append(deps, "family relationship is strong")
		deps = append(deps, "knows family context")
	}

	// Principle-based dependencies
	if l.contains(principles, "autonomy") {
		deps = append(deps, "respects your independence")
	}
	if l.contains(principles, "transparency") {
		deps = append(deps, "can be honest")
	}

	return deps
}

// contains checks if a string is in a slice
func (l *WhatWhoLinker) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
