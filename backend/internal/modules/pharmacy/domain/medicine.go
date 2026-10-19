// Package domain holds the pharmacy module's core entities: the medicine
// catalog, per-batch inventory, and the stock-movement ledger that backs
// dispensing history.
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status tracks whether a medicine is currently orderable/dispensable.
// Never hard-deleted so historical batches/movements/dispenses always
// resolve to a valid medicine.
type Status string

const (
	StatusActive   Status = "ACTIVE"
	StatusInactive Status = "INACTIVE"
)

// Medicine is one catalog entry. CurrentStock is a materialized running
// total (kept in sync by the repository whenever a batch is received or
// dispensed from — see repository.AdjustStock) rather than computed by
// summing batches on every read, so "list medicines below their reorder
// level" is a cheap indexed query instead of an aggregation.
type Medicine struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	Name         string             `bson:"name"`
	GenericName  string             `bson:"genericName,omitempty"`
	Category     string             `bson:"category,omitempty"`
	Manufacturer string             `bson:"manufacturer,omitempty"`
	Unit         string             `bson:"unit,omitempty"`
	ReorderLevel int                `bson:"reorderLevel"`
	CurrentStock int                `bson:"currentStock"`
	Status       Status             `bson:"status"`
	CreatedAt    time.Time          `bson:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt"`
}

// IsLowStock reports whether current stock has fallen to or below the
// reorder threshold.
func (m *Medicine) IsLowStock() bool {
	return m.CurrentStock <= m.ReorderLevel
}
