// Package domain holds the medical records module's core entity — the
// SOAP-style clinical note the blueprint calls for: complaint, history of
// present illness, examination, diagnosis, treatment and follow-up.
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MedicalRecord is one clinical encounter's documentation. PatientName and
// DoctorName are snapshotted at creation time (same join-free-reads
// tradeoff used by Doctors and Appointments).
//
// Medical records are never hard-deleted or replaced in place: they are
// clinical/legal documentation, so updates amend the existing document
// (audit-logged) rather than the module offering any delete operation.
type MedicalRecord struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	PatientID   primitive.ObjectID `bson:"patientId"`
	PatientName string             `bson:"patientName"`
	DoctorID    primitive.ObjectID `bson:"doctorId"`
	DoctorName  string             `bson:"doctorName"`
	Complaint   string             `bson:"complaint,omitempty"`
	HPI         string             `bson:"hpi,omitempty"`
	History     string             `bson:"history,omitempty"`
	Examination string             `bson:"examination,omitempty"`
	Diagnosis   string             `bson:"diagnosis"`
	Treatment   string             `bson:"treatment,omitempty"`
	FollowUp    string             `bson:"followUp,omitempty"`
	CreatedBy   primitive.ObjectID `bson:"createdBy,omitempty"`
	CreatedAt   time.Time          `bson:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt"`
}
