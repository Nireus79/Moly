// Package secrets protects credentials that Moly stores.
//
// The database is encrypted, but its key lives in the operating system's keychain, where a program running as the
// same user can sometimes reach it. A sealed secret adds a second lock that only the user's passphrase opens: a stolen
// database file, a backup, or malware that reads the keychain gets ciphertext, not the key.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"

	"golang.org/x/crypto/scrypt"
)

const (
	sealedPrefix = "sealed:v1:"
	plainPrefix  = "plain:"
	scryptN      = 1 << 15 // about 100 ms per guess; N=32768, r=8, p=1
	scryptR      = 8
	scryptP      = 1
	saltLen      = 16
	minPassLen   = 8
)

var (
	ErrWrongPassphrase = errors.New("wrong passphrase")
	ErrShortPass       = errors.New("the passphrase must be at least 8 characters")
)

func deriveKey(pass string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(pass), salt, scryptN, scryptR, scryptP, 32)
}

// Seal encrypts secret with a key derived from the passphrase (scrypt, AES-256-GCM) and returns a storable string.
func Seal(passphrase, secret string) (string, error) {
	if len(passphrase) < minPassLen {
		return "", ErrShortPass
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := deriveKey(passphrase, salt)
	if err != nil {
		return "", err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := append(append(salt, nonce...), gcm.Seal(nil, nonce, []byte(secret), []byte(sealedPrefix))...)
	return sealedPrefix + base64.StdEncoding.EncodeToString(out), nil
}

// Open reverses Seal. A wrong passphrase and a damaged value both give ErrWrongPassphrase.
func Open(passphrase, stored string) (string, error) {
	if !IsSealed(stored) {
		return "", errors.New("not a sealed value")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, sealedPrefix))
	if err != nil || len(raw) < saltLen+12+16 {
		return "", ErrWrongPassphrase
	}
	key, err := deriveKey(passphrase, raw[:saltLen])
	if err != nil {
		return "", err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	nonce := raw[saltLen : saltLen+gcm.NonceSize()]
	plain, err := gcm.Open(nil, nonce, raw[saltLen+gcm.NonceSize():], []byte(sealedPrefix))
	if err != nil {
		return "", ErrWrongPassphrase
	}
	return string(plain), nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func IsSealed(stored string) bool { return strings.HasPrefix(stored, sealedPrefix) }

// Plain marks a value that is stored without a passphrase (still inside the encrypted database).
func Plain(secret string) string { return plainPrefix + secret }

// FromPlain returns the secret of a Plain value.
func FromPlain(stored string) (string, bool) {
	if strings.HasPrefix(stored, plainPrefix) {
		return strings.TrimPrefix(stored, plainPrefix), true
	}
	return "", false
}
