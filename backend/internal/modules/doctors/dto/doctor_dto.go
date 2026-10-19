// Package dto defines the doctors module's request/response payloads.
package dto

import "time"

// AvailabilitySlotRequest is one recurring weekly availability window.
type AvailabilitySlotRequest struct {
	DayOfWeek int    `json:"dayOfWeek" binding:"required,min=0,max=6"`
	StartTime string `json:"startTime" binding:"required"`
	EndTime   string `json:"endTime" binding:"required"`
}

// CreateDoctorRequest links an existing user (role DOCTOR) to a clinical
// profile.
type CreateDoctorRequest struct {
	UserID         string                    `json:"userId" binding:"required"`
	DepartmentID   string                    `json:"departmentId" binding:"required"`
	LicenseNo      string                    `json:"licenseNo" binding:"required,min=2,max=60"`
	Specialization string                    `json:"specialization" binding:"required,min=2,max=120"`
	Fee            float64                   `json:"fee" binding:"required,min=0"`
	Bio            string                    `json:"bio" binding:"max=1000"`
	Availability   []AvailabilitySlotRequest `json:"availability"`
}

// UpdateDoctorRequest partially updates a doctor profile.
type UpdateDoctorRequest struct {
	DepartmentID   *string                    `json:"departmentId"`
	LicenseNo      *string                    `json:"licenseNo" binding:"omitempty,min=2,max=60"`
	Specialization *string                    `json:"specialization" binding:"omitempty,min=2,max=120"`
	Fee            *float64                   `json:"fee" binding:"omitempty,min=0"`
	Bio            *string                    `json:"bio" binding:"omitempty,max=1000"`
	Availability   *[]AvailabilitySlotRequest `json:"availability"`
}

// SearchDoctorsQuery is bound from query parameters for GET /doctors.
type SearchDoctorsQuery struct {
	Query          string `form:"query"`
	DepartmentID   string `form:"departmentId"`
	Specialization string `form:"specialization"`
	Page           int    `form:"page"`
	PageSize       int    `form:"pageSize"`
}

// AvailabilitySlotResponse mirrors domain.AvailabilitySlot.
type AvailabilitySlotResponse struct {
	DayOfWeek int    `json:"dayOfWeek"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

// DoctorResponse is the public representation of a doctor profile.
type DoctorResponse struct {
	ID             string                     `json:"id"`
	UserID         string                     `json:"userId"`
	Name           string                     `json:"name"`
	Email          string                     `json:"email"`
	DepartmentID   string                     `json:"departmentId"`
	DepartmentName string                     `json:"departmentName"`
	LicenseNo      string                     `json:"licenseNo"`
	Specialization string                     `json:"specialization"`
	Fee            float64                    `json:"fee"`
	Bio            string                     `json:"bio,omitempty"`
	Availability   []AvailabilitySlotResponse `json:"availability,omitempty"`
	Status         string                     `json:"status"`
	CreatedAt      time.Time                  `json:"createdAt"`
	UpdatedAt      time.Time                  `json:"updatedAt"`
}

// ScheduleResponse is the lightweight, purpose-built response for
// GET /doctors/{id}/schedule.
type ScheduleResponse struct {
	DoctorID     string                     `json:"doctorId"`
	Availability []AvailabilitySlotResponse `json:"availability"`
}
