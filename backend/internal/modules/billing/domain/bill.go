// Package domain holds the billing module's core entities: a Bill (an
// itemized charge for a patient) and its Payments (money received against
// it). They are modeled together, not as separate modules, because they
// are one tightly-coupled workflow — recording a payment directly drives
// its bill's status, the same relationship Lab Orders has with Lab
// Results.
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status tracks a bill's payment lifecycle.
type Status string

const (
	StatusUnpaid        Status = "UNPAID"
	StatusPartiallyPaid Status = "PARTIALLY_PAID"
	StatusPaid          Status = "PAID"
	StatusCancelled     Status = "CANCELLED"
)

// Item is one charge line on a bill. Category is a free-text label (not a
// typed enum, validated at the DTO layer instead) describing what the
// charge is for — it drives reporting/breakdown display only, never
// branching business logic, the same distinction the Pharmacy module
// draws between typed workflow statuses and plain-string reasons.
type Item struct {
	Description string  `bson:"description"`
	Category    string  `bson:"category"`
	Quantity    int     `bson:"quantity"`
	UnitPrice   float64 `bson:"unitPrice"`
	Amount      float64 `bson:"amount"`
}

// Bill is an itemized charge for a patient. Patient fields are
// snapshotted at creation time (same join-free-reads tradeoff used
// throughout — Doctors, Appointments, Lab Orders). AmountPaid is a
// materialized running total, kept in sync by the same atomic
// guard-filter pattern Pharmacy uses for Medicine.CurrentStock, so
// outstanding-balance search stays a simple status filter instead of a
// payments aggregation per bill per list request.
type Bill struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	BillNo      string             `bson:"billNo"`
	PatientID   primitive.ObjectID `bson:"patientId"`
	PatientName string             `bson:"patientName"`
	Items       []Item             `bson:"items"`
	Subtotal    float64            `bson:"subtotal"`
	Discount    float64            `bson:"discount"`
	Tax         float64            `bson:"tax"`
	TotalAmount float64            `bson:"totalAmount"`
	AmountPaid  float64            `bson:"amountPaid"`
	Status      Status             `bson:"status"`
	CreatedBy   primitive.ObjectID `bson:"createdBy,omitempty"`
	CreatedAt   time.Time          `bson:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt"`
}

// Balance is the amount still owed. Derived rather than stored, so it can
// never drift from TotalAmount/AmountPaid.
func (b *Bill) Balance() float64 {
	return b.TotalAmount - b.AmountPaid
}

// StatusFor computes the status a bill should have given its current
// AmountPaid, used after every payment is recorded.
func StatusFor(totalAmount, amountPaid float64) Status {
	switch {
	case amountPaid <= 0:
		return StatusUnpaid
	case amountPaid >= totalAmount:
		return StatusPaid
	default:
		return StatusPartiallyPaid
	}
}
