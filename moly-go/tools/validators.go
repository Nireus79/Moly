package tools

import (
	"regexp"
)

// Validators - Input validation utilities
type Validators struct{}






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









