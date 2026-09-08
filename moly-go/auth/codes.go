package auth

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"time"
)

// CodeConfig holds configuration for code generation
type CodeConfig struct {
	Length        int           // Number of characters per segment
	Segments      int           // Number of segments (e.g., "moly-XXXXX-YYYYY" = 2 segments)
	ExpirationTTL time.Duration // How long until code expires
}

// DefaultCodeConfig is the standard config for V2.1
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

// GenerateCode generates a new authentication code in format "moly-XXXXX-YYYYY"
func GenerateCode() (string, error) {
	return GenerateCodeWithConfig(DefaultCodeConfig)
}

// GenerateCodeWithConfig generates a code with custom configuration
func GenerateCodeWithConfig(cfg CodeConfig) (string, error) {
	// Generate random hex characters
	segments := make([]string, cfg.Segments)

	for i := 0; i < cfg.Segments; i++ {
		segment, err := generateRandomSegment(cfg.Length)
		if err != nil {
			return "", err
		}
		segments[i] = segment
	}

	// Format: moly-XXXXX-YYYYY
	code := "moly"
	for _, seg := range segments {
		code += "-" + seg
	}

	return code, nil
}

// generateRandomSegment generates a random hex string of length n
func generateRandomSegment(length int) (string, error) {
	b := make([]byte, (length + 1) / 2)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	hex := fmt.Sprintf("%x", b)
	// Return first 'length' characters (in case length is odd)
	if len(hex) < length {
		// This shouldn't happen, but pad if needed
		hex += "0"
	}
	return hex[:length], nil
}

// IsValid checks if a code string has valid format
func IsValid(code string) bool {
	// Format: moly-[a-f0-9]{5}-[a-f0-9]{5}
	pattern := `^moly-[a-f0-9]{5}-[a-f0-9]{5}$`
	matched, _ := regexp.MatchString(pattern, code)
	return matched
}

// IsExpired checks if a code has expired given its creation time
func IsExpired(createdAt time.Time) bool {
	expiresAt := createdAt.Add(DefaultCodeConfig.ExpirationTTL)
	return time.Now().After(expiresAt)
}

// IsExpiredWithTTL checks expiration with custom TTL
func IsExpiredWithTTL(createdAt time.Time, ttl time.Duration) bool {
	expiresAt := createdAt.Add(ttl)
	return time.Now().After(expiresAt)
}
