// Package dto defines the lab orders module's request/response payloads.
package dto

import "time"

// CreateLabOrderRequest places a new lab order.
type CreateLabOrderRequest struct {
	PatientID string `json:"patientId" binding:"required"`
	DoctorID  string `json:"doctorId" binding:"required"`
	TestID    string `json:"testId" binding:"required"`
}

// EnterResultRequest records a lab order's measured result.
type EnterResultRequest struct {
	ResultValue string `json:"resultValue" binding:"required,max=500"`
	Flag        string `json:"flag" binding:"required,oneof=NORMAL LOW HIGH CRITICAL"`
	Notes       string `json:"notes" binding:"max=1000"`
}

// SearchLabOrdersQuery is bound from query parameters for GET /lab-orders.
type SearchLabOrdersQuery struct {
	PatientID string `form:"patientId"`
	DoctorID  string `form:"doctorId"`
	Status    string `form:"status" binding:"omitempty,oneof=ORDERED AWAITING_VERIFICATION COMPLETED CANCELLED"`
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
}

// LabResultResponse is the public representation of a lab result.
type LabResultResponse struct {
	ID             string     `json:"id"`
	ResultValue    string     `json:"resultValue"`
	Flag           string     `json:"flag"`
	Notes          string     `json:"notes,omitempty"`
	EnteredByName  string     `json:"enteredByName,omitempty"`
	EnteredAt      time.Time  `json:"enteredAt"`
	VerifiedByName string     `json:"verifiedByName,omitempty"`
	VerifiedAt     *time.Time `json:"verifiedAt,omitempty"`
}

// LabOrderResponse is the public representation of a lab order.
type LabOrderResponse struct {
	ID          string             `json:"id"`
	PatientID   string             `json:"patientId"`
	PatientName string             `json:"patientName"`
	DoctorID    string             `json:"doctorId"`
	DoctorName  string             `json:"doctorName"`
	TestID      string             `json:"testId"`
	TestCode    string             `json:"testCode"`
	TestName    string             `json:"testName"`
	Unit        string             `json:"unit,omitempty"`
	NormalRange string             `json:"normalRange,omitempty"`
	SampleCode  string             `json:"sampleCode"`
	Status      string             `json:"status"`
	Result      *LabResultResponse `json:"result,omitempty"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}

// LabReportResponse is the printable report view for a completed order —
// GET /lab-orders/{id}/report. The frontend renders this as a printable
// page (browser print-to-PDF) rather than a server-rendered binary, so no
// PDF-generation library is required.
type LabReportResponse struct {
	Order LabOrderResponse `json:"order"`
}
