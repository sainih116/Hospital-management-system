// Package domain holds the prescriptions module's core entity.
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status tracks a prescription's lifecycle. DISPENSED is set directly by
// the Pharmacy module (via the repository, not this module's own
// ChangeStatus — see prescriptions/service.ChangeStatus, which still only
// ever allows ACTIVE -> CANCELLED) once dispensing against the medicines
// catalog succeeds.
type Status string

const (
	StatusActive    Status = "ACTIVE"
	StatusCancelled Status = "CANCELLED"
	StatusDispensed Status = "DISPENSED"
)

// Item is one medicine line on a prescription. MedicineName stays free
// text even now that Phase 7 (Pharmacy) has a real medicines catalog — a
// doctor's free-text order ("Paracetamol 500mg") and a pharmacist's
// stocked product name won't always match exactly, so Pharmacy's dispense
// flow lets the pharmacist search and pick the actual catalog medicine to
// dispense against (and the quantity) independently, rather than this
// module attempting unreliable text-matching against the catalog.
type Item struct {
	MedicineName string `bson:"medicineName"`
	Dose         string `bson:"dose"`
	Frequency    string `bson:"frequency"`
	Duration     string `bson:"duration"`
	Instructions string `bson:"instructions,omitempty"`
}

// Prescription is a doctor's set of medicine orders for a patient,
// optionally linked back to the clinical note it was written from.
type Prescription struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty"`
	PatientID       primitive.ObjectID  `bson:"patientId"`
	PatientName     string              `bson:"patientName"`
	DoctorID        primitive.ObjectID  `bson:"doctorId"`
	DoctorName      string              `bson:"doctorName"`
	MedicalRecordID *primitive.ObjectID `bson:"medicalRecordId,omitempty"`
	Items           []Item              `bson:"items"`
	Status          Status              `bson:"status"`
	IssuedAt        time.Time           `bson:"issuedAt"`
	CreatedBy       primitive.ObjectID  `bson:"createdBy,omitempty"`
	CreatedAt       time.Time           `bson:"createdAt"`
	UpdatedAt       time.Time           `bson:"updatedAt"`
}
