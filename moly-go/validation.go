package main

import (
	"errors"
	"regexp"
	"unicode/utf8"
)

// ValidationError represents a validation error with field and reason
type ValidationError struct {
	Field  string
	Reason string
}

// Validator contains validation rules and methods
type Validator struct {
	maxStringLength int
	maxIntValue     int
	minIntValue     int
}


// ValidateString checks if a string meets requirements
func (v *Validator) ValidateString(field, value string, required bool, minLen, maxLen int) error {
	if value == "" && required {
		return errors.New(field + " is required")
	}

	if value == "" {
		return nil
	}

	// Check for invalid UTF-8
	if !utf8.ValidString(value) {
		return errors.New(field + " contains invalid UTF-8")
	}

	runeCount := utf8.RuneCountInString(value)

	if minLen > 0 && runeCount < minLen {
		return errors.New(field + " must be at least " + string(rune(minLen)) + " characters")
	}

	if maxLen > 0 && runeCount > maxLen {
		return errors.New(field + " must not exceed " + string(rune(maxLen)) + " characters")
	}

	// Check total length limit
	if len(value) > v.maxStringLength {
		return errors.New(field + " exceeds maximum length")
	}

	return nil
}

// ValidateEmail validates email format
func (v *Validator) ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}

	if err := v.ValidateString("email", email, true, 0, 254); err != nil {
		return err
	}

	// Basic email regex validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return errors.New("email format is invalid")
	}

	return nil
}











// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
