package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MovementType is the direction of a stock change.
type MovementType string

const (
	MovementIn  MovementType = "IN"  // a new batch was received
	MovementOut MovementType = "OUT" // stock was dispensed
)

// StockMovement is one line of the stock ledger — what changed, on which
// batch, by how much, and why. This is what "Stock Tracking" (the phase
// checklist item, distinct from just "current quantity") actually means:
// a queryable history, not just a running total.
type StockMovement struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty"`
	MedicineID      primitive.ObjectID  `bson:"medicineId"`
	MedicineName    string              `bson:"medicineName"`
	BatchID         primitive.ObjectID  `bson:"batchId"`
	BatchNo         string              `bson:"batchNo"`
	Type            MovementType        `bson:"type"`
	Quantity        int                 `bson:"quantity"` // always positive; Type gives direction
	Reason          string              `bson:"reason,omitempty"`
	PrescriptionID  *primitive.ObjectID `bson:"prescriptionId,omitempty"`
	PerformedBy     primitive.ObjectID  `bson:"performedBy,omitempty"`
	PerformedByName string              `bson:"performedByName,omitempty"`
	CreatedAt       time.Time           `bson:"createdAt"`
}
