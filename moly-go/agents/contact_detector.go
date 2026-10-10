package agents

import (
	"moly/database"
)

// ContactDetector finds contacts mentioned in a message
type ContactDetector struct {
	db *database.Database
}

// isCommonWord checks if a capitalized word is a common word (not a name)
func (cd *ContactDetector) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"I":         true,
		"A":         true,
		"The":       true,
		"And":       true,
		"But":       true,
		"Or":        true,
		"In":        true,
		"On":        true,
		"At":        true,
		"To":        true,
		"From":      true,
		"With":      true,
		"My":        true,
		"Your":      true,
		"His":       true,
		"Her":       true,
		"Their":     true,
		"That":      true,
		"This":      true,
		"What":      true,
		"When":      true,
		"Where":     true,
		"Why":       true,
		"How":       true,
		"Who":       true,
		"Monday":    true,
		"Tuesday":   true,
		"Wednesday": true,
		"Thursday":  true,
		"Friday":    true,
		"Saturday":  true,
		"Sunday":    true,
		"January":   true,
		"February":  true,
		"March":     true,
		"April":     true,
		"May":       true,
		"June":      true,
		"July":      true,
		"August":    true,
		"September": true,
		"October":   true,
		"November":  true,
		"December":  true,
	}

	return commonWords[word]
}
