package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RefreshToken is a persisted, hashed refresh token used to implement
// rotation and revocation. The raw token is never stored — only its SHA-256
// hash — so a database read alone cannot be used to impersonate a session.
type RefreshToken struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         primitive.ObjectID `bson:"userId"`
	TokenHash      string             `bson:"tokenHash"`
	ExpiresAt      time.Time          `bson:"expiresAt"`
	CreatedAt      time.Time          `bson:"createdAt"`
	RevokedAt      *time.Time         `bson:"revokedAt,omitempty"`
	ReplacedByHash string             `bson:"replacedByHash,omitempty"`
	UserAgent      string             `bson:"userAgent,omitempty"`
	IP             string             `bson:"ip,omitempty"`
}

// IsActive reports whether the token can still be used to obtain a new
// session (not revoked, not expired).
func (t *RefreshToken) IsActive(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}
