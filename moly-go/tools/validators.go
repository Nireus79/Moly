package tools

import (
	"regexp"
	"strings"
)

// Validators - Input validation utilities
type Validators struct{}

// NewValidators - Create new validators
func NewValidators() *Validators {
	return &Validators{}
}

// ValidateMessage - Validate user message
func (v *Validators) ValidateMessage(message string) (bool, string) {
	if strings.TrimSpace(message) == "" {
		return false, "Message cannot be empty"
	}

	if len(message) > 5000 {
		return false, "Message too long (max 5000 characters)"
	}

	if len(message) < 2 {
		return false, "Message too short (min 2 characters)"
	}

	return true, ""
}

// ValidateUserID - Validate user ID
func (v *Validators) ValidateUserID(userID string) (bool, string) {
	if userID == "" {
		return false, "User ID cannot be empty"
	}

	// Valid user ID format: alphanumeric, dashes, underscores
	re := regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)
	if !re.MatchString(userID) {
		return false, "Invalid user ID format"
	}

	return true, ""
}

// ValidateConversationID - Validate conversation ID
func (v *Validators) ValidateConversationID(convID string) (bool, string) {
	if convID == "" {
		return false, "Conversation ID cannot be empty"
	}

	re := regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)
	if !re.MatchString(convID) {
		return false, "Invalid conversation ID format"
	}

	return true, ""
}

// ValidateContactName - Validate contact name
func (v *Validators) ValidateContactName(name string) (bool, string) {
	if strings.TrimSpace(name) == "" {
		return false, "Contact name cannot be empty"
	}

	if len(name) > 100 {
		return false, "Contact name too long (max 100 characters)"
	}

	if len(name) < 2 {
		return false, "Contact name too short (min 2 characters)"
	}

	return true, ""
}

// ValidateEmail - Validate email address
func (v *Validators) ValidateEmail(email string) (bool, string) {
	if email == "" {
		return true, "" // Optional field
	}

	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !re.MatchString(email) {
		return false, "Invalid email format"
	}

	return true, ""
}

// ValidateURL - Validate URL
func (v *Validators) ValidateURL(urlStr string) (bool, string) {
	if urlStr == "" {
		return true, "" // Optional field
	}

	re := regexp.MustCompile(`^https?://`)
	if !re.MatchString(urlStr) {
		return false, "URL must start with http:// or https://"
	}

	if len(urlStr) > 2048 {
		return false, "URL too long"
	}

	return true, ""
}

// ValidatePhoneNumber - Validate phone number
func (v *Validators) ValidatePhoneNumber(phone string) (bool, string) {
	if phone == "" {
		return true, "" // Optional field
	}

	// Accept various formats: +1-555-555-5555, (555) 555-5555, 555-555-5555, etc.
	re := regexp.MustCompile(`^[\d\-\+\(\)\s]{10,20}$`)
	if !re.MatchString(phone) {
		return false, "Invalid phone number format"
	}

	return true, ""
}

// IsSafeText - Check if text is safe (no suspicious patterns)
func (v *Validators) IsSafeText(text string) bool {
	dangerousPatterns := []string{
		"<script",
		"javascript:",
		"onclick=",
		"onerror=",
		"eval(",
		"exec(",
	}

	lowerText := strings.ToLower(text)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerText, pattern) {
			return false
		}
	}

	return true
}

// NormalizeText - Normalize text for storage
func (v *Validators) NormalizeText(text string) string {
	// Trim whitespace
	text = strings.TrimSpace(text)

	// Remove multiple spaces
	text = strings.Join(strings.Fields(text), " ")

	// Remove control characters
	text = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, text)

	return text
}

// SanitizeForDisplay - Sanitize text for safe display
func (v *Validators) SanitizeForDisplay(text string) string {
	// HTML escape
	replacements := map[string]string{
		"&":  "&amp;",
		"<":  "&lt;",
		">":  "&gt;",
		"\"": "&quot;",
		"'":  "&#39;",
	}

	result := text
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	return result
}

// ContainsProfanity - Check if text contains profanity
func (v *Validators) ContainsProfanity(text string) bool {
	// Simplified - real implementation would use a proper filter
	profanityList := []string{
		// Add common profanity patterns
		// This is a placeholder
	}

	lowerText := strings.ToLower(text)
	for _, word := range profanityList {
		if strings.Contains(lowerText, word) {
			return true
		}
	}

	return false
}

// ExtractMentions - Extract @mentions from text
func (v *Validators) ExtractMentions(text string) []string {
	re := regexp.MustCompile(`@([a-zA-Z0-9_-]+)`)
	matches := re.FindAllStringSubmatch(text, -1)

	mentions := []string{}
	for _, match := range matches {
		if len(match) > 1 {
			mentions = append(mentions, match[1])
		}
	}

	return mentions
}

// ExtractHashtags - Extract #hashtags from text
func (v *Validators) ExtractHashtags(text string) []string {
	re := regexp.MustCompile(`#([a-zA-Z0-9_]+)`)
	matches := re.FindAllStringSubmatch(text, -1)

	hashtags := []string{}
	for _, match := range matches {
		if len(match) > 1 {
			hashtags = append(hashtags, match[1])
		}
	}

	return hashtags
}

// ExtractURLs - Extract URLs from text
func (v *Validators) ExtractURLs(text string) []string {
	re := regexp.MustCompile(`https?://[^\s]+`)
	urls := re.FindAllString(text, -1)
	return urls
}
