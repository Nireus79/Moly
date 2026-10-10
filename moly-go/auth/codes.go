package auth

import (
	"regexp"
	"time"
)

// CodeConfig holds configuration for code generation
type CodeConfig struct {
	Length        int           // Number of characters per segment
	Segments      int           // Number of segments (e.g., "moly-XXXXX-YYYYY" = 2 segments)
	ExpirationTTL time.Duration // How long until code expires
}

// DefaultCodeConfig is the standard config 
var DefaultCodeConfig = CodeConfig{
	Length:        5,
	Segments:      2,
	ExpirationTTL: 1 * time.Hour,
}

// Code represents a generated authentication code
type Code struct {
	Value     string
	CreatedAt time.Time
	ExpiresAt time.Time
}




// IsValid checks if a code string has valid format
func IsValid(code string) bool {
	// Format: moly-[a-f0-9]{5}-[a-f0-9]{5}
	pattern := `^moly-[a-f0-9]{5}-[a-f0-9]{5}$`
	matched, _ := regexp.MatchString(pattern, code)
	return matched
}


