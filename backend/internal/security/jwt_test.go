package security

import (
	"testing"
	"time"

	"hms-backend/internal/config"
)

func testTokenService() *TokenService {
	return NewTokenService(config.JWTConfig{
		AccessSecret:   "test-access-secret",
		RefreshSecret:  "test-refresh-secret",
		AccessTokenTTL: 15 * time.Minute,
		Issuer:         "hms-backend-test",
	})
}

func TestGenerateAndParseAccessToken(t *testing.T) {
	svc := testTokenService()

	token, expiresAt, err := svc.GenerateAccessToken("user-123", "doctor@example.com", "DOCTOR", []string{"patient.read", "emr.write"})
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}
	if token == "" {
		t.Fatalf("expected a non-empty token")
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("expected expiry to be in the future")
	}

	claims, err := svc.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken returned error: %v", err)
	}

	if claims.UserID != "user-123" {
		t.Errorf("expected UserID user-123, got %s", claims.UserID)
	}
	if claims.Role != "DOCTOR" {
		t.Errorf("expected Role DOCTOR, got %s", claims.Role)
	}
	if len(claims.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(claims.Permissions))
	}
}

func TestParseAccessTokenRejectsTamperedSignature(t *testing.T) {
	svc := testTokenService()

	token, _, err := svc.GenerateAccessToken("user-123", "doctor@example.com", "DOCTOR", nil)
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	tampered := token[:len(token)-1] + "x"

	if _, err := svc.ParseAccessToken(tampered); err == nil {
		t.Fatalf("expected an error parsing a tampered token")
	}
}

func TestParseAccessTokenRejectsWrongIssuer(t *testing.T) {
	issuedBy := NewTokenService(config.JWTConfig{
		AccessSecret:   "shared-secret",
		AccessTokenTTL: time.Minute,
		Issuer:         "issuer-a",
	})
	verifiedBy := NewTokenService(config.JWTConfig{
		AccessSecret:   "shared-secret",
		AccessTokenTTL: time.Minute,
		Issuer:         "issuer-b",
	})

	token, _, err := issuedBy.GenerateAccessToken("user-1", "a@example.com", "ADMIN", nil)
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	if _, err := verifiedBy.ParseAccessToken(token); err == nil {
		t.Fatalf("expected issuer mismatch to be rejected")
	}
}

func TestParseAccessTokenRejectsExpiredToken(t *testing.T) {
	svc := NewTokenService(config.JWTConfig{
		AccessSecret:   "test-access-secret",
		AccessTokenTTL: -time.Minute, // already expired
		Issuer:         "hms-backend-test",
	})

	token, _, err := svc.GenerateAccessToken("user-123", "doctor@example.com", "DOCTOR", nil)
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	if _, err := svc.ParseAccessToken(token); err == nil {
		t.Fatalf("expected an expired token to be rejected")
	}
}
