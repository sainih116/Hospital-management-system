package dto

import "time"

// ReceiveBatchRequest records a newly received batch of stock.
type ReceiveBatchRequest struct {
	BatchNo       string    `json:"batchNo" binding:"required,max=60"`
	ExpiryDate    time.Time `json:"expiryDate" binding:"required"`
	Quantity      int       `json:"quantity" binding:"required,min=1"`
	PurchasePrice float64   `json:"purchasePrice" binding:"min=0"`
	SellingPrice  float64   `json:"sellingPrice" binding:"min=0"`
}

// BatchResponse is the public representation of a received batch.
type BatchResponse struct {
	ID            string    `json:"id"`
	MedicineID    string    `json:"medicineId"`
	MedicineName  string    `json:"medicineName"`
	BatchNo       string    `json:"batchNo"`
	ExpiryDate    time.Time `json:"expiryDate"`
	Quantity      int       `json:"quantity"`
	IsExpired     bool      `json:"isExpired"`
	PurchasePrice float64   `json:"purchasePrice"`
	SellingPrice  float64   `json:"sellingPrice"`
	CreatedAt     time.Time `json:"createdAt"`
}
