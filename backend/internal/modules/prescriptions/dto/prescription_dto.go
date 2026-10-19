// Package dto defines the prescriptions module's request/response
// payloads.
package dto

import "time"

// ItemRequest is one prescribed medicine line.
type ItemRequest struct {
	MedicineName string `json:"medicineName" binding:"required,min=2,max=200"`
	Dose         string `json:"dose" binding:"required,max=60"`
	Frequency    string `json:"frequency" binding:"required,max=60"`
	Duration     string `json:"duration" binding:"required,max=60"`
	Instructions string `json:"instructions" binding:"max=300"`
}

// CreatePrescriptionRequest issues a new prescription.
type CreatePrescriptionRequest struct {
	PatientID       string        `json:"patientId" binding:"required"`
	DoctorID        string        `json:"doctorId" binding:"required"`
	MedicalRecordID string        `json:"medicalRecordId"`
	Items           []ItemRequest `json:"items" binding:"required,min=1,dive"`
}

// ChangeStatusRequest cancels a prescription (the only transition this
// module makes — see domain.Status doc).
type ChangeStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=CANCELLED"`
}

// SearchPrescriptionsQuery is bound from query parameters for
// GET /prescriptions.
type SearchPrescriptionsQuery struct {
	PatientID string `form:"patientId"`
	DoctorID  string `form:"doctorId"`
	Status    string `form:"status" binding:"omitempty,oneof=ACTIVE CANCELLED"`
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
}

// ItemResponse mirrors domain.Item.
type ItemResponse struct {
	MedicineName string `json:"medicineName"`
	Dose         string `json:"dose"`
	Frequency    string `json:"frequency"`
	Duration     string `json:"duration"`
	Instructions string `json:"instructions,omitempty"`
}

// PrescriptionResponse is the public representation of a prescription.
type PrescriptionResponse struct {
	ID              string         `json:"id"`
	PatientID       string         `json:"patientId"`
	PatientName     string         `json:"patientName"`
	DoctorID        string         `json:"doctorId"`
	DoctorName      string         `json:"doctorName"`
	MedicalRecordID string         `json:"medicalRecordId,omitempty"`
	Items           []ItemResponse `json:"items"`
	Status          string         `json:"status"`
	IssuedAt        time.Time      `json:"issuedAt"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}
