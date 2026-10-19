// Package dto defines the pharmacy module's request/response payloads.
package dto

import "time"

// CreateMedicineRequest adds a new medicine to the catalog.
type CreateMedicineRequest struct {
	Name         string `json:"name" binding:"required,min=2,max=150"`
	GenericName  string `json:"genericName" binding:"max=150"`
	Category     string `json:"category" binding:"max=100"`
	Manufacturer string `json:"manufacturer" binding:"max=150"`
	Unit         string `json:"unit" binding:"max=30"`
	ReorderLevel int    `json:"reorderLevel" binding:"min=0"`
}

// UpdateMedicineRequest partially updates a catalog entry.
type UpdateMedicineRequest struct {
	Name         *string `json:"name" binding:"omitempty,min=2,max=150"`
	GenericName  *string `json:"genericName" binding:"omitempty,max=150"`
	Category     *string `json:"category" binding:"omitempty,max=100"`
	Manufacturer *string `json:"manufacturer" binding:"omitempty,max=150"`
	Unit         *string `json:"unit" binding:"omitempty,max=30"`
	ReorderLevel *int    `json:"reorderLevel" binding:"omitempty,min=0"`
	Status       *string `json:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
}

// SearchMedicinesQuery is bound from query parameters for GET /medicines.
type SearchMedicinesQuery struct {
	Query    string `form:"query"`
	LowStock bool   `form:"lowStock"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// MedicineResponse is the public representation of a catalog entry.
type MedicineResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	GenericName  string    `json:"genericName,omitempty"`
	Category     string    `json:"category,omitempty"`
	Manufacturer string    `json:"manufacturer,omitempty"`
	Unit         string    `json:"unit,omitempty"`
	ReorderLevel int       `json:"reorderLevel"`
	CurrentStock int       `json:"currentStock"`
	IsLowStock   bool      `json:"isLowStock"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}
