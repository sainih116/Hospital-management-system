package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BedStatus tracks a bed's current occupancy. AVAILABLE beds can be
// assigned to a new admission; OCCUPIED beds are atomically guarded
// against being assigned twice (see repository.BedRepository.Occupy) —
// the same "no negative stock" guard-filter pattern Pharmacy uses for
// Medicine.CurrentStock, applied here to prevent the exact race the
// blueprint calls out: "a bed cannot be assigned to two active
// admissions at the same time."
type BedStatus string

const (
	BedAvailable   BedStatus = "AVAILABLE"
	BedOccupied    BedStatus = "OCCUPIED"
	BedMaintenance BedStatus = "MAINTENANCE"
)

// Bed is one physical bed within a Ward. WardName is snapshotted at
// creation time (same join-free-reads tradeoff used throughout — Doctors,
// Appointments, Lab Orders).
type Bed struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	WardID    primitive.ObjectID `bson:"wardId"`
	WardName  string             `bson:"wardName"`
	BedNumber string             `bson:"bedNumber"`
	Status    BedStatus          `bson:"status"`
	CreatedAt time.Time          `bson:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt"`
}
