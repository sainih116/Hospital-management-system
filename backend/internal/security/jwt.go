package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"hms-backend/internal/config"
)

// ErrInvalidToken is returned for any token that fails signature, format or
// expiry validation. Callers should not attempt to distinguish sub-cases —
// treat every failure as "not authenticated".
var ErrInvalidToken = errors.New("security: invalid or expired token")

// Claims are the JWT access-token claims. Role and Permissions are baked in
// at issuance time so authorization checks never need a database round
// trip; the short access-token TTL bounds how stale they can get.
type Claims struct {
	UserID      string   `json:"uid"`
	Email       string   `json:"email"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// TokenService issues and verifies JWT access tokens.
type TokenService struct {
	accessSecret []byte
	accessTTL    time.Duration
	issuer       string
}

// NewTokenService builds a TokenService from JWT configuration.
func NewTokenService(cfg config.JWTConfig) *TokenService {
	return &TokenService{
		accessSecret: []byte(cfg.AccessSecret),
		accessTTL:    cfg.AccessTokenTTL,
		issuer:       cfg.Issuer,
	}
}

// GenerateAccessToken issues a signed JWT for the given identity, returning
// the token and its absolute expiry time.
func (s *TokenService) GenerateAccessToken(userID, email, role string, permissions []string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(s.accessTTL)

	claims := Claims{
		UserID:      userID,
		Email:       email,
		Role:        role,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.accessSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("security: sign access token: %w", err)
	}

	return signed, expiresAt, nil
}

// ParseAccessToken validates a signed access token and returns its claims.
func (s *TokenService) ParseAccessToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.accessSecret, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
