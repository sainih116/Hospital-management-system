package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Flag is the clinical interpretation of a result relative to the test's
// normal range.
type Flag string

const (
	FlagNormal   Flag = "NORMAL"
	FlagLow      Flag = "LOW"
	FlagHigh     Flag = "HIGH"
	FlagCritical Flag = "CRITICAL"
)

// LabResult is the measured outcome of a LabOrder. Unverified until a
// second person (Admin or Lab Technician — never the ordering Doctor, per
// the blueprint's REST table) confirms it via the verify endpoint, at
// which point its parent LabOrder moves to COMPLETED.
type LabResult struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty"`
	LabOrderID     primitive.ObjectID  `bson:"labOrderId"`
	ResultValue    string              `bson:"resultValue"`
	Flag           Flag                `bson:"flag"`
	Notes          string              `bson:"notes,omitempty"`
	EnteredBy      primitive.ObjectID  `bson:"enteredBy,omitempty"`
	EnteredByName  string              `bson:"enteredByName,omitempty"`
	EnteredAt      time.Time           `bson:"enteredAt"`
	VerifiedBy     *primitive.ObjectID `bson:"verifiedBy,omitempty"`
	VerifiedByName string              `bson:"verifiedByName,omitempty"`
	VerifiedAt     *time.Time          `bson:"verifiedAt,omitempty"`
	CreatedAt      time.Time           `bson:"createdAt"`
	UpdatedAt      time.Time           `bson:"updatedAt"`
}

// IsVerified reports whether the result has passed verification.
func (r *LabResult) IsVerified() bool {
	return r.VerifiedAt != nil
}
