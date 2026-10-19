package dto

import "time"

// AdmitPatientRequest admits a patient into a specific bed.
type AdmitPatientRequest struct {
	PatientID string `json:"patientId" binding:"required"`
	DoctorID  string `json:"doctorId" binding:"required"`
	BedID     string `json:"bedId" binding:"required"`
	Reason    string `json:"reason" binding:"max=500"`
}

// DischargeRequest discharges an admitted patient and releases their bed.
type DischargeRequest struct {
	Note string `json:"note" binding:"max=1000"`
}

// SearchAdmissionsQuery is bound from query parameters for GET /ipd/admissions.
type SearchAdmissionsQuery struct {
	PatientID string `form:"patientId"`
	DoctorID  string `form:"doctorId"`
	Status    string `form:"status" binding:"omitempty,oneof=ADMITTED DISCHARGED CANCELLED"`
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
}

// AdmissionResponse is the public representation of an admission.
type AdmissionResponse struct {
	ID            string     `json:"id"`
	PatientID     string     `json:"patientId"`
	PatientName   string     `json:"patientName"`
	DoctorID      string     `json:"doctorId"`
	DoctorName    string     `json:"doctorName"`
	BedID         string     `json:"bedId"`
	BedNumber     string     `json:"bedNumber"`
	WardName      string     `json:"wardName"`
	Reason        string     `json:"reason,omitempty"`
	Status        string     `json:"status"`
	AdmittedAt    time.Time  `json:"admittedAt"`
	DischargedAt  *time.Time `json:"dischargedAt,omitempty"`
	DischargeNote string     `json:"dischargeNote,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}
