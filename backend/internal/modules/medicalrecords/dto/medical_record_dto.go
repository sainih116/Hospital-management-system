// Package dto defines the medical records module's request/response
// payloads.
package dto

import "time"

// CreateMedicalRecordRequest documents a clinical encounter.
type CreateMedicalRecordRequest struct {
	PatientID   string `json:"patientId" binding:"required"`
	DoctorID    string `json:"doctorId" binding:"required"`
	Complaint   string `json:"complaint" binding:"max=1000"`
	HPI         string `json:"hpi" binding:"max=2000"`
	History     string `json:"history" binding:"max=2000"`
	Examination string `json:"examination" binding:"max=2000"`
	Diagnosis   string `json:"diagnosis" binding:"required,min=2,max=1000"`
	Treatment   string `json:"treatment" binding:"max=2000"`
	FollowUp    string `json:"followUp" binding:"max=1000"`
}

// UpdateMedicalRecordRequest amends an existing clinical note. Every field
// is optional so callers only send what changed.
type UpdateMedicalRecordRequest struct {
	Complaint   *string `json:"complaint" binding:"omitempty,max=1000"`
	HPI         *string `json:"hpi" binding:"omitempty,max=2000"`
	History     *string `json:"history" binding:"omitempty,max=2000"`
	Examination *string `json:"examination" binding:"omitempty,max=2000"`
	Diagnosis   *string `json:"diagnosis" binding:"omitempty,min=2,max=1000"`
	Treatment   *string `json:"treatment" binding:"omitempty,max=2000"`
	FollowUp    *string `json:"followUp" binding:"omitempty,max=1000"`
}

// SearchMedicalRecordsQuery is bound from query parameters for
// GET /medical-records — also how "consultation history" (a patient's full
// visit record) is served, via ?patientId=.
type SearchMedicalRecordsQuery struct {
	PatientID string `form:"patientId"`
	DoctorID  string `form:"doctorId"`
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
}

// MedicalRecordResponse is the public representation of a medical record.
type MedicalRecordResponse struct {
	ID          string    `json:"id"`
	PatientID   string    `json:"patientId"`
	PatientName string    `json:"patientName"`
	DoctorID    string    `json:"doctorId"`
	DoctorName  string    `json:"doctorName"`
	Complaint   string    `json:"complaint,omitempty"`
	HPI         string    `json:"hpi,omitempty"`
	History     string    `json:"history,omitempty"`
	Examination string    `json:"examination,omitempty"`
	Diagnosis   string    `json:"diagnosis"`
	Treatment   string    `json:"treatment,omitempty"`
	FollowUp    string    `json:"followUp,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
