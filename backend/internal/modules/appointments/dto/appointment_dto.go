// Package dto defines the appointments module's request/response payloads.
package dto

import "time"

// BookAppointmentRequest books a new appointment.
type BookAppointmentRequest struct {
	PatientID string    `json:"patientId" binding:"required"`
	DoctorID  string    `json:"doctorId" binding:"required"`
	StartAt   time.Time `json:"startAt" binding:"required"`
	EndAt     time.Time `json:"endAt" binding:"required"`
	Reason    string    `json:"reason" binding:"max=500"`
}

// ChangeStatusRequest transitions an appointment to a new status.
// Reason is used for CANCELLED/NO_SHOW to record why.
type ChangeStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=CONFIRMED IN_PROGRESS COMPLETED CANCELLED NO_SHOW"`
	Reason string `json:"reason" binding:"max=500"`
}

// RescheduleRequest moves an appointment to a new time slot. The original
// appointment is marked RESCHEDULED and a new one is created.
type RescheduleRequest struct {
	StartAt time.Time `json:"startAt" binding:"required"`
	EndAt   time.Time `json:"endAt" binding:"required"`
}

// SearchAppointmentsQuery is bound from query parameters for
// GET /appointments — also how "doctor schedule view" (a doctor's booked
// appointments in a date range) is served.
type SearchAppointmentsQuery struct {
	PatientID string     `form:"patientId"`
	DoctorID  string     `form:"doctorId"`
	Status    string     `form:"status" binding:"omitempty,oneof=SCHEDULED CONFIRMED IN_PROGRESS COMPLETED CANCELLED NO_SHOW RESCHEDULED"`
	From      *time.Time `form:"from" time_format:"2006-01-02T15:04:05Z07:00"`
	To        *time.Time `form:"to" time_format:"2006-01-02T15:04:05Z07:00"`
	Page      int        `form:"page"`
	PageSize  int        `form:"pageSize"`
}

// AppointmentResponse is the public representation of an appointment.
type AppointmentResponse struct {
	ID                string    `json:"id"`
	PatientID         string    `json:"patientId"`
	PatientName       string    `json:"patientName"`
	DoctorID          string    `json:"doctorId"`
	DoctorName        string    `json:"doctorName"`
	StartAt           time.Time `json:"startAt"`
	EndAt             time.Time `json:"endAt"`
	Status            string    `json:"status"`
	Reason            string    `json:"reason,omitempty"`
	Notes             string    `json:"notes,omitempty"`
	CancelReason      string    `json:"cancelReason,omitempty"`
	RescheduledFromID string    `json:"rescheduledFromId,omitempty"`
	RescheduledToID   string    `json:"rescheduledToId,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}
