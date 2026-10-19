// Package domain holds the patients module's core entity.
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Gender is restricted to a small, explicit set of codes used throughout
// the clinical modules (lab reference ranges, billing, etc. key off it).
type Gender string

const (
	GenderMale   Gender = "M"
	GenderFemale Gender = "F"
	GenderOther  Gender = "O"
)

// AllGenders lists every valid gender code, used for request validation.
var AllGenders = []Gender{GenderMale, GenderFemale, GenderOther}

// PatientType distinguishes outpatients from (currently or previously)
// admitted inpatients; IPD/ward assignment itself is modeled in Phase 7.
type PatientType string

const (
	PatientTypeOPD PatientType = "OPD"
	PatientTypeIPD PatientType = "IPD"
)

// AllPatientTypes lists every valid patient type code.
var AllPatientTypes = []PatientType{PatientTypeOPD, PatientTypeIPD}

// Status tracks whether a patient record is visible in normal operation.
// Patients are never hard-deleted — see Patient.DeletedAt — both to
// preserve the clinical/billing history other modules will reference and
// to satisfy the retention expectations called out in the blueprint's
// compliance notes.
type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

// AllBloodGroups lists the accepted blood group codes.
var AllBloodGroups = []string{"A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"}

// EmergencyContact is a next-of-kin/contact reachable in an emergency.
type EmergencyContact struct {
	Name         string `bson:"name"`
	Relationship string `bson:"relationship,omitempty"`
	Phone        string `bson:"phone"`
}

// Insurance is the patient's payer information, referenced by billing in
// a later phase.
type Insurance struct {
	Provider     string `bson:"provider,omitempty"`
	PolicyNumber string `bson:"policyNumber,omitempty"`
}

// Patient is a hospital patient record: demographics, contact info,
// emergency contact and insurance. Clinical/visit/billing history lives in
// their own collections (medical_records, appointments, bills, ...) added
// in later phases, referencing a patient by ID.
type Patient struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	PatientCode      string             `bson:"patientCode"`
	Name             string             `bson:"name"`
	DOB              time.Time          `bson:"dob"`
	Gender           Gender             `bson:"gender"`
	Phone            string             `bson:"phone"`
	Email            string             `bson:"email,omitempty"`
	Address          string             `bson:"address,omitempty"`
	BloodGroup       string             `bson:"bloodGroup,omitempty"`
	Allergies        []string           `bson:"allergies,omitempty"`
	EmergencyContact *EmergencyContact  `bson:"emergencyContact,omitempty"`
	Insurance        *Insurance         `bson:"insurance,omitempty"`
	Type             PatientType        `bson:"type"`
	Status           Status             `bson:"status"`
	CreatedBy        primitive.ObjectID `bson:"createdBy,omitempty"`
	CreatedAt        time.Time          `bson:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt"`
	DeletedAt        *time.Time         `bson:"deletedAt,omitempty"`
}

// IsValidGender reports whether code is a known gender code.
func IsValidGender(code string) bool {
	for _, g := range AllGenders {
		if string(g) == code {
			return true
		}
	}
	return false
}

// IsValidPatientType reports whether code is a known patient type code.
func IsValidPatientType(code string) bool {
	for _, t := range AllPatientTypes {
		if string(t) == code {
			return true
		}
	}
	return false
}

// IsValidBloodGroup reports whether code is a known blood group code.
func IsValidBloodGroup(code string) bool {
	for _, bg := range AllBloodGroups {
		if bg == code {
			return true
		}
	}
	return false
}
