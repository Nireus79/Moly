package auth

import (
	"crypto/sha256"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a Moly user account
type User struct {
	ID           string `db:"id"`
	Email        string `db:"email"`
	PasswordHash string `db:"password_hash"`
	CreatedAt    int64  `db:"created_at"`
	UpdatedAt    int64  `db:"updated_at"`
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword checks if a password matches the hash
func VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateUserID generates a unique user ID
func GenerateUserID(email string) string {
	hash := sha256.Sum256([]byte(email + fmt.Sprintf("%d", time.Now().UnixNano())))
	return fmt.Sprintf("user_%x", hash)[:20]
}

// ValidateEmail checks if email is valid format
func ValidateEmail(email string) bool {
	// Basic email validation
	return len(email) > 0 && len(email) < 254
}

// ValidatePassword checks password strength
func ValidatePassword(password string) bool {
	// Minimum 8 characters
	return len(password) >= 8
}
