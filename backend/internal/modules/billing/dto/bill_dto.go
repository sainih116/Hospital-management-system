// Package dto defines the billing module's request/response payloads.
package dto

import "time"

// CreateBillItemRequest is one charge line on a new bill.
type CreateBillItemRequest struct {
	Description string  `json:"description" binding:"required,max=200"`
	Category    string  `json:"category" binding:"required,oneof=CONSULTATION LAB PHARMACY PROCEDURE ROOM_CHARGE OTHER"`
	Quantity    int     `json:"quantity" binding:"required,min=1"`
	UnitPrice   float64 `json:"unitPrice" binding:"min=0"`
}

// CreateBillRequest raises a new itemized bill for a patient.
type CreateBillRequest struct {
	PatientID string                  `json:"patientId" binding:"required"`
	Items     []CreateBillItemRequest `json:"items" binding:"required,min=1,dive"`
	Discount  float64                 `json:"discount" binding:"min=0"`
	Tax       float64                 `json:"tax" binding:"min=0"`
}

// SearchBillsQuery is bound from query parameters for GET /billing/bills.
type SearchBillsQuery struct {
	PatientID       string `form:"patientId"`
	Status          string `form:"status" binding:"omitempty,oneof=UNPAID PARTIALLY_PAID PAID CANCELLED"`
	OutstandingOnly bool   `form:"outstandingOnly"`
	Page            int    `form:"page"`
	PageSize        int    `form:"pageSize"`
}

// BillItemResponse is the public representation of one bill line item.
type BillItemResponse struct {
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	Amount      float64 `json:"amount"`
}

// BillResponse is the public representation of a bill, including its
// payment history.
type BillResponse struct {
	ID          string             `json:"id"`
	BillNo      string             `json:"billNo"`
	PatientID   string             `json:"patientId"`
	PatientName string             `json:"patientName"`
	Items       []BillItemResponse `json:"items"`
	Subtotal    float64            `json:"subtotal"`
	Discount    float64            `json:"discount"`
	Tax         float64            `json:"tax"`
	TotalAmount float64            `json:"totalAmount"`
	AmountPaid  float64            `json:"amountPaid"`
	Balance     float64            `json:"balance"`
	Status      string             `json:"status"`
	Payments    []PaymentResponse  `json:"payments,omitempty"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}
