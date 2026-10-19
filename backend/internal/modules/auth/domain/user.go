// Package domain holds the auth module's core entities and RBAC registry —
// the vocabulary every other layer (repository, service, handler) builds on.
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserStatus represents the lifecycle state of a user account.
type UserStatus string

const (
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusInactive UserStatus = "INACTIVE"
	UserStatusLocked   UserStatus = "LOCKED"
)

// User is a staff account (Admin, Doctor, Nurse, Receptionist, Pharmacist,
// Lab Technician). Patients are not users of the system and are modeled
// separately in Phase 2.
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Name         string             `bson:"name"`
	Email        string             `bson:"email"`
	PasswordHash string             `bson:"passwordHash"`
	RoleCode     string             `bson:"roleCode"`
	Status       UserStatus         `bson:"status"`
	LastLoginAt  *time.Time         `bson:"lastLoginAt,omitempty"`
	CreatedAt    time.Time          `bson:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt"`
}

// IsActive reports whether the account is allowed to authenticate.
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}
