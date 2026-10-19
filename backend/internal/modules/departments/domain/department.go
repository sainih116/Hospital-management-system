// Package domain holds the departments module's core entity. Departments
// are simple reference data (Cardiology, Pediatrics, ...) that doctors and,
// in later phases, wards/beds link to.
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status tracks whether a department is available for new assignments.
// Departments are never hard-deleted so existing doctor records that
// reference one always resolve.
type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

// Department is a hospital department/ward category.
type Department struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Name      string             `bson:"name"`
	Code      string             `bson:"code"`
	Status    Status             `bson:"status"`
	CreatedAt time.Time          `bson:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt"`
}
