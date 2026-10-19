// Package security provides password hashing, opaque token generation and
// JWT issuance/verification shared across the application.
package security

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost balances hashing latency against brute-force resistance. 12 is
// the widely recommended minimum for production password storage.
const bcryptCost = 12

// HashPassword returns a bcrypt hash of the given plaintext password.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("security: hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword reports whether plain matches the given bcrypt hash.
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
