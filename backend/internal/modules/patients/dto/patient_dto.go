// Package dto defines the patients module's request/response payloads.
package dto

import "time"

// EmergencyContactRequest is the nested emergency-contact payload. When
// provided, Name and Phone are required (Relationship is optional).
type EmergencyContactRequest struct {
	Name         string `json:"name" binding:"required,min=2,max=120"`
	Relationship string `json:"relationship" binding:"max=60"`
	Phone        string `json:"phone" binding:"required"`
}

// InsuranceRequest is the nested insurance payload; both fields are
// optional but capped in length.
type InsuranceRequest struct {
	Provider     string `json:"provider" binding:"max=120"`
	PolicyNumber string `json:"policyNumber" binding:"max=60"`
}

// CreatePatientRequest registers a new patient.
type CreatePatientRequest struct {
	Name             string                   `json:"name" binding:"required,min=2,max=150"`
	DOB              string                   `json:"dob" binding:"required"` // "YYYY-MM-DD"
	Gender           string                   `json:"gender" binding:"required,oneof=M F O"`
	Phone            string                   `json:"phone" binding:"required"`
	Email            string                   `json:"email" binding:"omitempty,email,max=254"`
	Address          string                   `json:"address" binding:"max=300"`
	BloodGroup       string                   `json:"bloodGroup" binding:"omitempty,oneof=A+ A- B+ B- AB+ AB- O+ O-"`
	Allergies        []string                 `json:"allergies"`
	EmergencyContact *EmergencyContactRequest `json:"emergencyContact"`
	Insurance        *InsuranceRequest        `json:"insurance"`
	Type             string                   `json:"type" binding:"required,oneof=OPD IPD"`
}

// UpdatePatientRequest partially updates a patient. Every field is a
// pointer so the service layer can distinguish "omitted" from "cleared".
type UpdatePatientRequest struct {
	Name             *string                  `json:"name" binding:"omitempty,min=2,max=150"`
	DOB              *string                  `json:"dob" binding:"omitempty"`
	Gender           *string                  `json:"gender" binding:"omitempty,oneof=M F O"`
	Phone            *string                  `json:"phone" binding:"omitempty"`
	Email            *string                  `json:"email" binding:"omitempty,email,max=254"`
	Address          *string                  `json:"address" binding:"omitempty,max=300"`
	BloodGroup       *string                  `json:"bloodGroup" binding:"omitempty,oneof=A+ A- B+ B- AB+ AB- O+ O-"`
	Allergies        *[]string                `json:"allergies"`
	EmergencyContact *EmergencyContactRequest `json:"emergencyContact"`
	Insurance        *InsuranceRequest        `json:"insurance"`
	Type             *string                  `json:"type" binding:"omitempty,oneof=OPD IPD"`
}

// SearchPatientsQuery is bound from query parameters for GET /patients.
type SearchPatientsQuery struct {
	Query      string `form:"query"`
	Gender     string `form:"gender" binding:"omitempty,oneof=M F O"`
	BloodGroup string `form:"bloodGroup" binding:"omitempty,oneof=A+ A- B+ B- AB+ AB- O+ O-"`
	Type       string `form:"type" binding:"omitempty,oneof=OPD IPD"`
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
}

// PatientResponse is the public-safe representation of a patient.
type PatientResponse struct {
	ID               string            `json:"id"`
	PatientCode      string            `json:"patientCode"`
	Name             string            `json:"name"`
	DOB              string            `json:"dob"`
	Gender           string            `json:"gender"`
	Phone            string            `json:"phone"`
	Email            string            `json:"email,omitempty"`
	Address          string            `json:"address,omitempty"`
	BloodGroup       string            `json:"bloodGroup,omitempty"`
	Allergies        []string          `json:"allergies,omitempty"`
	EmergencyContact *EmergencyContact `json:"emergencyContact,omitempty"`
	Insurance        *Insurance        `json:"insurance,omitempty"`
	Type             string            `json:"type"`
	Status           string            `json:"status"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

// EmergencyContact mirrors domain.EmergencyContact for the response body.
type EmergencyContact struct {
	Name         string `json:"name"`
	Relationship string `json:"relationship,omitempty"`
	Phone        string `json:"phone"`
}

// Insurance mirrors domain.Insurance for the response body.
type Insurance struct {
	Provider     string `json:"provider,omitempty"`
	PolicyNumber string `json:"policyNumber,omitempty"`
}
