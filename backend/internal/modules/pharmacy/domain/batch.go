package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Batch is one received lot of a medicine — the blueprint's DB design
// calls this "medicine_batches"; the phase's own collection name is
// "inventory" (see repository.go), so the Go type is named for what it
// models rather than either literal collection name.
type Batch struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	MedicineID    primitive.ObjectID `bson:"medicineId"`
	MedicineName  string             `bson:"medicineName"`
	BatchNo       string             `bson:"batchNo"`
	ExpiryDate    time.Time          `bson:"expiryDate"`
	Quantity      int                `bson:"quantity"`
	PurchasePrice float64            `bson:"purchasePrice"`
	SellingPrice  float64            `bson:"sellingPrice"`
	ReceivedBy    primitive.ObjectID `bson:"receivedBy,omitempty"`
	CreatedAt     time.Time          `bson:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt"`
}

// IsExpired reports whether the batch's expiry date has passed as of now.
func (b *Batch) IsExpired(now time.Time) bool {
	return !b.ExpiryDate.After(now)
}
