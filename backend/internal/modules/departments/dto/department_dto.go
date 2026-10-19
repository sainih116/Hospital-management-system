// Package dto defines the departments module's request/response payloads.
package dto

import "time"

// CreateDepartmentRequest creates a new department.
type CreateDepartmentRequest struct {
	Name string `json:"name" binding:"required,min=2,max=120"`
	Code string `json:"code" binding:"required,min=2,max=20"`
}

// UpdateDepartmentRequest partially updates a department.
type UpdateDepartmentRequest struct {
	Name   *string `json:"name" binding:"omitempty,min=2,max=120"`
	Status *string `json:"status" binding:"omitempty,oneof=ACTIVE INACTIVE"`
}

// DepartmentResponse is the public representation of a department.
type DepartmentResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
