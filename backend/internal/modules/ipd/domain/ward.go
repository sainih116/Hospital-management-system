// Package domain holds the IPD (in-patient department) module's core
// entities: Ward and Bed (the physical inventory) and Admission (the
// workflow that occupies a bed). Modeled as one module, not split the
// way Departments/Doctors are, because all three are tightly coupled —
// admitting a patient directly changes a bed's status, the same
// relationship Lab Orders has with Lab Results.
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status tracks whether a ward is available for new bed assignments.
// Wards are never hard-deleted so existing bed/admission records that
// reference one always resolve — the same pattern Departments uses.
type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

// Ward is a hospital ward category (General Ward, ICU, ...) — simple
// reference data that Beds link to, the same role Departments plays for
// Doctors.
type Ward struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Name      string             `bson:"name"`
	Code      string             `bson:"code"`
	Status    Status             `bson:"status"`
	CreatedAt time.Time          `bson:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt"`
}
