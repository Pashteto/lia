// Package password hashes and verifies user passwords with bcrypt.
package password

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// ErrMismatch is returned when a plaintext password does not match the hash.
var ErrMismatch = errors.New("password mismatch")

// Cost is the bcrypt cost factor (>=10 per the security baseline).
const Cost = 12

// Hash returns a bcrypt hash of the plaintext password.
func Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), Cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Compare returns nil if plain matches hash, ErrMismatch otherwise.
func Compare(hash, plain string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		return ErrMismatch
	}
	return nil
}
