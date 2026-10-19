package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// opaqueTokenBytes is the amount of entropy used for refresh/reset tokens
// (32 bytes = 256 bits, encoded as ~43 URL-safe characters).
const opaqueTokenBytes = 32

// GenerateOpaqueToken returns a cryptographically random, URL-safe token
// suitable for refresh tokens and password reset tokens. Only its hash
// (see HashOpaqueToken) should ever be persisted.
func GenerateOpaqueToken() (string, error) {
	buf := make([]byte, opaqueTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("security: generate opaque token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashOpaqueToken returns the SHA-256 hex digest of a raw token, the form
// stored in the database so a leaked database dump cannot be replayed as a
// live session.
func HashOpaqueToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
