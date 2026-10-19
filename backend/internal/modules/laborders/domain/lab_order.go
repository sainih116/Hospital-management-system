// Package domain holds the lab orders module's core entities: a LabOrder
// (a doctor's request for a test) and its LabResult (the measured value,
// once entered and verified). They are modeled together, not as separate
// modules, because they are one tightly-coupled workflow — verifying a
// result directly drives its order's status.
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status is intentionally just four states — the phase this module was
// built for calls for "Create Lab Order, Result Entry, Report Generation"
// only; granular sample-collection tracking (collected/received/
// processing, mentioned in the blueprint's broader workflow narrative but
// not in this phase's checklist) is not modeled here.
type Status string

const (
	StatusOrdered              Status = "ORDERED"
	StatusAwaitingVerification Status = "AWAITING_VERIFICATION"
	StatusCompleted            Status = "COMPLETED"
	StatusCancelled            Status = "CANCELLED"
)

// LabOrder is a doctor's request for a patient to have a catalog test
// performed. Patient/doctor/test fields are snapshotted at creation time
// (same join-free-reads tradeoff used throughout — Doctors, Appointments,
// Medical Records).
type LabOrder struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	PatientID   primitive.ObjectID `bson:"patientId"`
	PatientName string             `bson:"patientName"`
	DoctorID    primitive.ObjectID `bson:"doctorId"`
	DoctorName  string             `bson:"doctorName"`
	TestID      primitive.ObjectID `bson:"testId"`
	TestCode    string             `bson:"testCode"`
	TestName    string             `bson:"testName"`
	Unit        string             `bson:"unit,omitempty"`
	NormalRange string             `bson:"normalRange,omitempty"`
	SampleCode  string             `bson:"sampleCode"`
	Status      Status             `bson:"status"`
	CreatedBy   primitive.ObjectID `bson:"createdBy,omitempty"`
	CreatedAt   time.Time          `bson:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt"`
}
