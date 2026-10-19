// Package dto defines the IPD module's request/response payloads.
package dto

import "time"

// CreateWardRequest adds a new ward.
type CreateWardRequest struct {
	Name string `json:"name" binding:"required,min=2,max=120"`
	Code string `json:"code" binding:"required,min=2,max=20"`
}

// UpdateWardRequest partially updates a ward.
type UpdateWardRequest struct {
	Name   *string `json:"name" binding:"omitempty,min=2,max=120"`
	Code   *string `json:"code" binding:"omitempty,min=2,max=20"`
	Status *string `json:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
}

// WardResponse is the public representation of a ward.
type WardResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
