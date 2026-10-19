package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AdmissionStatus tracks an admission's lifecycle. Intentionally just
// three states — the blueprint's IPD checklist ("Bed board, admission,
// care notes, discharge, bed release") doesn't call for the broader
// nursing/vitals/medication-administration workflow also mentioned in
// its workflow narrative, so that's deliberately not modeled here, the
// same scope discipline applied to EMR attachments and Lab's
// sample-collection tracking in earlier phases.
type AdmissionStatus string

const (
	AdmissionActive     AdmissionStatus = "ADMITTED"
	AdmissionDischarged AdmissionStatus = "DISCHARGED"
	AdmissionCancelled  AdmissionStatus = "CANCELLED"
)

// Admission is one patient's stay in a bed. Patient/doctor/bed fields are
// snapshotted at creation time (same join-free-reads tradeoff used
// throughout this codebase).
type Admission struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	PatientID     primitive.ObjectID `bson:"patientId"`
	PatientName   string             `bson:"patientName"`
	DoctorID      primitive.ObjectID `bson:"doctorId"`
	DoctorName    string             `bson:"doctorName"`
	BedID         primitive.ObjectID `bson:"bedId"`
	BedNumber     string             `bson:"bedNumber"`
	WardName      string             `bson:"wardName"`
	Reason        string             `bson:"reason,omitempty"`
	Status        AdmissionStatus    `bson:"status"`
	AdmittedAt    time.Time          `bson:"admittedAt"`
	DischargedAt  *time.Time         `bson:"dischargedAt,omitempty"`
	DischargeNote string             `bson:"dischargeNote,omitempty"`
	CreatedBy     primitive.ObjectID `bson:"createdBy,omitempty"`
	CreatedAt     time.Time          `bson:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt"`
}
