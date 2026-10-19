package dto

import "time"

// DispenseItemRequest is one medicine+quantity line to dispense. The
// pharmacist picks the actual catalog medicine independently of the
// prescription's free-text item names — see domain package doc on why.
type DispenseItemRequest struct {
	MedicineID string `json:"medicineId" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,min=1"`
}

// DispenseRequest fulfills a prescription in a single transaction: every
// requested item is dispensed and the prescription is marked DISPENSED, or
// nothing happens (see service for the compensating-rollback behavior this
// deployment uses in place of a database transaction).
type DispenseRequest struct {
	PrescriptionID string                `json:"prescriptionId" binding:"required"`
	Items          []DispenseItemRequest `json:"items" binding:"required,min=1,dive"`
}

// DispensedItemResponse reports how one requested item was actually
// fulfilled — potentially split across multiple batches (FEFO).
type DispensedItemResponse struct {
	MedicineID   string            `json:"medicineId"`
	MedicineName string            `json:"medicineName"`
	Quantity     int               `json:"quantity"`
	Batches      []BatchAllocation `json:"batches"`
}

// BatchAllocation is the portion of a dispensed quantity taken from one
// specific batch.
type BatchAllocation struct {
	BatchID  string `json:"batchId"`
	BatchNo  string `json:"batchNo"`
	Quantity int    `json:"quantity"`
}

// DispenseResponse is the result of a successful dispense operation.
type DispenseResponse struct {
	PrescriptionID string                  `json:"prescriptionId"`
	DispensedAt    time.Time               `json:"dispensedAt"`
	Items          []DispensedItemResponse `json:"items"`
}

// SearchStockMovementsQuery is bound from query parameters for
// GET /stock-movements.
type SearchStockMovementsQuery struct {
	MedicineID string `form:"medicineId"`
	Type       string `form:"type" binding:"omitempty,oneof=IN OUT"`
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
}

// StockMovementResponse is the public representation of one ledger entry.
type StockMovementResponse struct {
	ID              string    `json:"id"`
	MedicineID      string    `json:"medicineId"`
	MedicineName    string    `json:"medicineName"`
	BatchID         string    `json:"batchId"`
	BatchNo         string    `json:"batchNo"`
	Type            string    `json:"type"`
	Quantity        int       `json:"quantity"`
	Reason          string    `json:"reason,omitempty"`
	PrescriptionID  string    `json:"prescriptionId,omitempty"`
	PerformedByName string    `json:"performedByName,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
}
