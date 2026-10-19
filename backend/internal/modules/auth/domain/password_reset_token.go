package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PasswordResetToken is a single-use, hashed, time-limited token issued by
// the "forgot password" flow.
type PasswordResetToken struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	UserID    primitive.ObjectID `bson:"userId"`
	TokenHash string             `bson:"tokenHash"`
	ExpiresAt time.Time          `bson:"expiresAt"`
	UsedAt    *time.Time         `bson:"usedAt,omitempty"`
	CreatedAt time.Time          `bson:"createdAt"`
}

// IsValid reports whether the token can still be redeemed.
func (t *PasswordResetToken) IsValid(now time.Time) bool {
	return t.UsedAt == nil && now.Before(t.ExpiresAt)
}
