package dto

import "time"

// CreateBedRequest adds a new bed to a ward.
type CreateBedRequest struct {
	WardID    string `json:"wardId" binding:"required"`
	BedNumber string `json:"bedNumber" binding:"required,min=1,max=20"`
}

// UpdateBedRequest changes a bed's status directly — e.g. taking it in or
// out of MAINTENANCE. Occupying/freeing a bed as part of an admission
// goes through the admission endpoints instead, never this one, so the
// only statuses settable here are AVAILABLE and MAINTENANCE.
type UpdateBedRequest struct {
	Status string `json:"status" binding:"required,oneof=AVAILABLE MAINTENANCE"`
}

// BedResponse is the public representation of a bed.
type BedResponse struct {
	ID        string    `json:"id"`
	WardID    string    `json:"wardId"`
	WardName  string    `json:"wardName"`
	BedNumber string    `json:"bedNumber"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
