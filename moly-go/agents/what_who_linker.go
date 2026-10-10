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
