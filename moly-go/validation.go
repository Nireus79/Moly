package main

import (
	"errors"
	"regexp"
	"strings"
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

// NewValidator creates a new validator with default limits
func NewValidator() *Validator {
	return &Validator{
		maxStringLength: 10000,
		maxIntValue:     2147483647, // 32-bit max
		minIntValue:     -2147483648,
	}
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

// ValidateURL validates URL format
func (v *Validator) ValidateURL(url string) error {
	if url == "" {
		return errors.New("URL is required")
	}

	if err := v.ValidateString("url", url, true, 0, 2048); err != nil {
		return err
	}

	// Basic URL validation
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "chrome-extension://") {
		return errors.New("URL must start with http://, https://, or chrome-extension://")
	}

	return nil
}

// ValidateJSON validates JSON object keys and values
func (v *Validator) ValidateJSON(data map[string]interface{}, allowedKeys map[string]bool) error {
	for key := range data {
		if !allowedKeys[key] {
			return errors.New("unexpected field: " + key)
		}
	}
	return nil
}

// SanitizeString removes potentially dangerous characters
func (v *Validator) SanitizeString(value string) string {
	// Remove null bytes
	value = strings.ReplaceAll(value, "\x00", "")

	// Trim whitespace
	value = strings.TrimSpace(value)

	return value
}

// ValidateMessageContent validates chat message content
func (v *Validator) ValidateMessageContent(message string) error {
	if err := v.ValidateString("message", message, true, 1, 5000); err != nil {
		return err
	}

	// Check for SQL injection patterns
	dangerousPatterns := []string{
		"DROP", "DELETE", "INSERT", "UPDATE", "UNION",
		";", "--", "/*", "*/", "xp_", "sp_",
	}

	upperMsg := strings.ToUpper(message)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(upperMsg, pattern) {
			// Note: This is a simple check. Real SQL injection prevention uses parameterized queries.
			Logger.WithFields(map[string]interface{}{
				"pattern": pattern,
				"message": message[:min(len(message), 100)],
			}).Warn("Potential SQL injection pattern detected")
		}
	}

	return nil
}

// ValidateContactName validates contact name
func (v *Validator) ValidateContactName(name string) error {
	if err := v.ValidateString("contact_name", name, true, 1, 255); err != nil {
		return err
	}

	// Only allow alphanumeric, spaces, and common punctuation
	nameRegex := regexp.MustCompile(`^[a-zA-Z0-9\s\-'.]+$`)
	if !nameRegex.MatchString(name) {
		return errors.New("contact name contains invalid characters")
	}

	return nil
}

// ValidateMode validates mode selection
func (v *Validator) ValidateMode(mode string) error {
	validModes := map[string]bool{
		"socratic": true,
		"direct":   true,
	}

	if !validModes[mode] {
		return errors.New("invalid mode: " + mode)
	}

	return nil
}

// ValidateContext validates context selection
func (v *Validator) ValidateContext(context string) error {
	validContexts := map[string]bool{
		"formal":   true,
		"friendly": true,
		"dating":   true,
	}

	if !validContexts[context] {
		return errors.New("invalid context: " + context)
	}

	return nil
}

// ValidateProvider validates LLM provider selection
func (v *Validator) ValidateProvider(provider string) error {
	validProviders := map[string]bool{
		"claude": true,
		"openai": true,
		"ollama": true,
		"local":  true,
	}

	if !validProviders[provider] {
		return errors.New("invalid provider: " + provider)
	}

	return nil
}

// ValidateAPIKey validates API key format
func (v *Validator) ValidateAPIKey(apiKey string) error {
	if apiKey == "" {
		return errors.New("API key is required")
	}

	if err := v.ValidateString("api_key", apiKey, true, 10, 512); err != nil {
		return err
	}

	// API keys should only contain alphanumeric, hyphens, and underscores
	keyRegex := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	if !keyRegex.MatchString(apiKey) {
		return errors.New("API key contains invalid characters")
	}

	return nil
}

// ValidateInteger validates integer value
func (v *Validator) ValidateInteger(field string, value int, minVal, maxVal int) error {
	if value < minVal || value > maxVal {
		return errors.New(field + " must be between " + string(rune(minVal)) + " and " + string(rune(maxVal)))
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
