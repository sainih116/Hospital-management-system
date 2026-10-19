// Package dto defines the lab tests module's request/response payloads.
package dto

import "time"

// CreateLabTestRequest adds a new test to the catalog.
type CreateLabTestRequest struct {
	Code        string  `json:"code" binding:"required,min=2,max=20"`
	Name        string  `json:"name" binding:"required,min=2,max=150"`
	Category    string  `json:"category" binding:"max=100"`
	SampleType  string  `json:"sampleType" binding:"max=60"`
	Unit        string  `json:"unit" binding:"max=30"`
	NormalRange string  `json:"normalRange" binding:"max=100"`
	Cost        float64 `json:"cost" binding:"required,min=0"`
}

// UpdateLabTestRequest partially updates a catalog entry.
type UpdateLabTestRequest struct {
	Name        *string  `json:"name" binding:"omitempty,min=2,max=150"`
	Category    *string  `json:"category" binding:"omitempty,max=100"`
	SampleType  *string  `json:"sampleType" binding:"omitempty,max=60"`
	Unit        *string  `json:"unit" binding:"omitempty,max=30"`
	NormalRange *string  `json:"normalRange" binding:"omitempty,max=100"`
	Cost        *float64 `json:"cost" binding:"omitempty,min=0"`
	Status      *string  `json:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
}

// LabTestResponse is the public representation of a catalog entry.
type LabTestResponse struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Category    string    `json:"category,omitempty"`
	SampleType  string    `json:"sampleType,omitempty"`
	Unit        string    `json:"unit,omitempty"`
	NormalRange string    `json:"normalRange,omitempty"`
	Cost        float64   `json:"cost"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
